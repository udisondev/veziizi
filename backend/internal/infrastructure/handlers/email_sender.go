package handlers

import (
	"context"
	"fmt"
	"html"
	"log/slog"
	"strings"

	"github.com/ThreeDotsLabs/watermill/components/cqrs"
	"github.com/google/uuid"

	"github.com/udisondev/veziizi/backend/internal/domain/notification/values"
	"github.com/udisondev/veziizi/backend/internal/infrastructure/messaging"
	"github.com/udisondev/veziizi/backend/internal/infrastructure/notifications"
	"github.com/udisondev/veziizi/backend/internal/infrastructure/projections"
	"github.com/udisondev/veziizi/backend/internal/pkg/config"
)

// EmailSenderHandler отправляет уведомления по Email. Регистрируется как
// cqrs.NewEventHandler[messaging.EmailNotification] в воркере email-sender.
//
// Идемпотентность: см. TelegramSenderHandler. До provider.Send проверяем
// notification_dedup.IsSent, после успеха — MarkSent. Транзиентная ошибка
// от provider возвращается наверх для Retry middleware, без записи в dedup.
type EmailSenderHandler struct {
	provider    notifications.EmailProvider
	appConfig   *config.Config
	deliveryLog *projections.NotificationDeliveryLogProjection
	dedup       *projections.NotificationDedupProjection
}

func NewEmailSenderHandler(
	provider notifications.EmailProvider,
	appConfig *config.Config,
	deliveryLog *projections.NotificationDeliveryLogProjection,
	dedup *projections.NotificationDedupProjection,
) *EmailSenderHandler {
	return &EmailSenderHandler{
		provider:    provider,
		appConfig:   appConfig,
		deliveryLog: deliveryLog,
		dedup:       dedup,
	}
}

func (h *EmailSenderHandler) OnEmailNotification(ctx context.Context, n *messaging.EmailNotification) error {
	msg := cqrs.OriginalMessageFromCtx(ctx)
	msgUUID, err := uuid.Parse(msg.UUID)
	if err != nil {
		return fmt.Errorf("parse message uuid: %w", err)
	}
	sent, err := h.dedup.IsSent(ctx, msgUUID, "email")
	if err != nil {
		return fmt.Errorf("dedup is sent: %w", err)
	}
	if sent {
		slog.Debug("email notification already sent, skipping",
			slog.String("message_uuid", msg.UUID),
			slog.String("member_id", n.MemberID.String()))
		return nil
	}

	link := ""
	if n.Link != "" {
		baseURL := h.appConfig.App.BaseURL
		if baseURL == "" {
			baseURL = "https://veziizi.ru"
		}
		link = baseURL + n.Link
	}

	bodyHTML := h.formatEmailHTML(n.Title, n.Body, link)
	bodyText := h.formatEmailText(n.Title, n.Body, link)

	emailMsg := notifications.EmailMessage{
		To:        n.Email,
		Subject:   n.Title,
		BodyHTML:  bodyHTML,
		BodyText:  bodyText,
		EmailType: values.EmailTypeTransactional,
	}

	result, err := h.provider.Send(ctx, emailMsg)
	if err != nil {
		slog.Error("failed to send email",
			slog.String("email", n.Email),
			slog.String("member_id", n.MemberID.String()),
			slog.String("error", err.Error()))

		if h.deliveryLog != nil {
			if logErr := h.deliveryLog.LogDelivery(ctx, projections.DeliveryLogInput{
				MemberID:         n.MemberID,
				NotificationType: n.NotificationType,
				Channel:          "email",
				Status:           "failed",
				ErrorMessage:     err.Error(),
			}); logErr != nil {
				slog.Error("failed to log delivery failure",
					slog.String("member_id", n.MemberID.String()),
					slog.String("error", logErr.Error()))
			}
		}
		return fmt.Errorf("send email: %w", err)
	}

	// Фиксируем sent ПОСЛЕ успешного provider.Send. Ошибка БД не возвращается
	// наверх — письмо уже ушло, повторная отправка хуже потери строки в
	// notification_dedup.
	if err := h.dedup.MarkSent(ctx, msgUUID, "email"); err != nil {
		slog.Error("failed to mark notification as sent",
			slog.String("message_uuid", msg.UUID),
			slog.String("member_id", n.MemberID.String()),
			slog.String("error", err.Error()))
	}

	slog.Info("email sent",
		slog.String("message_id", result.MessageID),
		slog.String("email", n.Email),
		slog.String("member_id", n.MemberID.String()))

	if h.deliveryLog != nil {
		if err := h.deliveryLog.LogDelivery(ctx, projections.DeliveryLogInput{
			MemberID:         n.MemberID,
			NotificationType: n.NotificationType,
			Channel:          "email",
			Status:           "sent",
		}); err != nil {
			slog.Error("failed to log successful delivery",
				slog.String("member_id", n.MemberID.String()),
				slog.String("error", err.Error()))
		}
	}
	return nil
}

func (h *EmailSenderHandler) formatEmailHTML(title, body, link string) string {
	safeTitle := html.EscapeString(title)
	safeBody := html.EscapeString(body)

	linkSection := ""
	if link != "" && (strings.HasPrefix(link, "http://") || strings.HasPrefix(link, "https://")) {
		safeLink := html.EscapeString(link)
		linkSection = fmt.Sprintf(`
			<p style="margin-top: 20px;">
				<a href="%s" style="background-color: #4F46E5; color: white; padding: 12px 24px; text-decoration: none; border-radius: 6px; display: inline-block;">
					Открыть в приложении
				</a>
			</p>
		`, safeLink)
	}

	safeBaseURL := html.EscapeString(h.appConfig.App.BaseURL)

	return fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
	<meta charset="UTF-8">
	<meta name="viewport" content="width=device-width, initial-scale=1.0">
</head>
<body style="font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; line-height: 1.6; color: #333; max-width: 600px; margin: 0 auto; padding: 20px;">
	<div style="background-color: #f8f9fa; border-radius: 8px; padding: 30px;">
		<h1 style="color: #1a1a1a; margin-top: 0; font-size: 24px;">%s</h1>
		<p style="color: #4a4a4a; font-size: 16px;">%s</p>
		%s
	</div>
	<p style="color: #999; font-size: 12px; margin-top: 20px; text-align: center;">
		Это письмо отправлено автоматически. Пожалуйста, не отвечайте на него.
		<br>
		<a href="%s/settings/notifications" style="color: #666;">Настроить уведомления</a>
	</p>
</body>
</html>
	`, safeTitle, safeBody, linkSection, safeBaseURL)
}

func (h *EmailSenderHandler) formatEmailText(title, body, link string) string {
	text := fmt.Sprintf("%s\n\n%s", title, body)
	if link != "" {
		text += fmt.Sprintf("\n\nОткрыть в приложении: %s", link)
	}
	text += fmt.Sprintf("\n\n---\nНастроить уведомления: %s/settings/notifications", h.appConfig.App.BaseURL)
	return text
}
