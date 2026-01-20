package notifications

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/resend/resend-go/v3"

	"codeberg.org/udison/veziizi/backend/internal/domain/notification/values"
)

// EmailMessage сообщение для отправки по email
type EmailMessage struct {
	// To адрес получателя
	To string
	// Subject тема письма
	Subject string
	// BodyHTML HTML-содержимое письма
	BodyHTML string
	// BodyText текстовое содержимое письма (fallback)
	BodyText string
	// TrackingID идентификатор для tracking (только для маркетинговых)
	TrackingID *uuid.UUID
	// EmailType тип письма (transactional/marketing)
	EmailType values.EmailType
	// ReplyTo адрес для ответа (опционально)
	ReplyTo string
	// Tags теги для аналитики
	Tags map[string]string
}

// SendResult результат отправки email
type SendResult struct {
	// MessageID идентификатор сообщения от провайдера
	MessageID string
}

// EmailProvider интерфейс для отправки email
type EmailProvider interface {
	// Send отправляет email
	Send(ctx context.Context, msg EmailMessage) (*SendResult, error)
}

// ResendProvider реализация EmailProvider через Resend API
type ResendProvider struct {
	client      *resend.Client
	fromAddress string
	fromName    string
}

// NewResendProvider создает новый Resend провайдер
func NewResendProvider(apiKey, fromAddress, fromName string) *ResendProvider {
	return &ResendProvider{
		client:      resend.NewClient(apiKey),
		fromAddress: fromAddress,
		fromName:    fromName,
	}
}

// Send отправляет email через Resend API
func (p *ResendProvider) Send(ctx context.Context, msg EmailMessage) (*SendResult, error) {
	from := p.fromAddress
	if p.fromName != "" {
		from = fmt.Sprintf("%s <%s>", p.fromName, p.fromAddress)
	}

	params := &resend.SendEmailRequest{
		To:      []string{msg.To},
		From:    from,
		Subject: msg.Subject,
		Html:    msg.BodyHTML,
		Text:    msg.BodyText,
	}

	if msg.ReplyTo != "" {
		params.ReplyTo = msg.ReplyTo
	}

	// Добавляем теги для tracking
	if len(msg.Tags) > 0 {
		tags := make([]resend.Tag, 0, len(msg.Tags))
		for name, value := range msg.Tags {
			tags = append(tags, resend.Tag{
				Name:  name,
				Value: value,
			})
		}
		params.Tags = tags
	}

	// Добавляем тег типа email
	params.Tags = append(params.Tags, resend.Tag{
		Name:  "email_type",
		Value: string(msg.EmailType),
	})

	// Добавляем tracking ID если есть
	if msg.TrackingID != nil {
		params.Tags = append(params.Tags, resend.Tag{
			Name:  "tracking_id",
			Value: msg.TrackingID.String(),
		})
	}

	sent, err := p.client.Emails.Send(params)
	if err != nil {
		return nil, fmt.Errorf("resend send email: %w", err)
	}

	slog.Info("email sent via resend",
		slog.String("message_id", sent.Id),
		slog.String("to", msg.To),
		slog.String("subject", msg.Subject),
		slog.String("email_type", string(msg.EmailType)),
	)

	return &SendResult{
		MessageID: sent.Id,
	}, nil
}

// NoopEmailProvider заглушка для development/testing
type NoopEmailProvider struct{}

// NewNoopEmailProvider создает noop провайдер
func NewNoopEmailProvider() *NoopEmailProvider {
	return &NoopEmailProvider{}
}

// Send логирует email без реальной отправки
func (p *NoopEmailProvider) Send(ctx context.Context, msg EmailMessage) (*SendResult, error) {
	fakeID := uuid.New().String()

	slog.Info("email send (noop)",
		slog.String("message_id", fakeID),
		slog.String("to", msg.To),
		slog.String("subject", msg.Subject),
		slog.String("email_type", string(msg.EmailType)),
		slog.Bool("has_html", msg.BodyHTML != ""),
		slog.Bool("has_text", msg.BodyText != ""),
	)

	return &SendResult{
		MessageID: fakeID,
	}, nil
}

// NewEmailProvider создает EmailProvider на основе конфигурации
func NewEmailProvider(provider, apiKey, fromAddress, fromName string, enabled bool) EmailProvider {
	if !enabled {
		slog.Info("email provider disabled, using noop")
		return NewNoopEmailProvider()
	}

	switch provider {
	case "resend":
		if apiKey == "" {
			slog.Warn("resend API key not set, using noop provider")
			return NewNoopEmailProvider()
		}
		slog.Info("using resend email provider",
			slog.String("from", fromAddress),
		)
		return NewResendProvider(apiKey, fromAddress, fromName)
	default:
		slog.Warn("unknown email provider, using noop",
			slog.String("provider", provider),
		)
		return NewNoopEmailProvider()
	}
}
