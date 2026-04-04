package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"
	notifApp "github.com/udisondev/veziizi/backend/internal/application/notification"
	"github.com/udisondev/veziizi/backend/internal/domain/notification/rules"
	"github.com/udisondev/veziizi/backend/internal/domain/notification/values"
	"github.com/udisondev/veziizi/backend/internal/infrastructure/persistence/eventstore"
)

// TelegramNotification представляет сообщение для отправки в Telegram
type TelegramNotification struct {
	MemberID uuid.UUID `json:"member_id"`
	ChatID   int64     `json:"chat_id"`
	Title    string    `json:"title"`
	Body     string    `json:"body"`
	Link     string    `json:"link,omitempty"`
}

// EmailNotification представляет сообщение для отправки по Email
type EmailNotification struct {
	MemberID         uuid.UUID `json:"member_id"`
	Email            string    `json:"email"`
	NotificationType string    `json:"notification_type"`
	Title            string    `json:"title"`
	Body             string    `json:"body"`
	Link             string    `json:"link,omitempty"`
}

// NotificationDispatcherHandler модульный dispatcher с правилами
type NotificationDispatcherHandler struct {
	registry     *rules.Registry
	notifService *notifApp.Service
	publisher    message.Publisher
}

// NewNotificationDispatcherHandler создает новый handler
func NewNotificationDispatcherHandler(
	registry *rules.Registry,
	notifService *notifApp.Service,
	publisher message.Publisher,
) *NotificationDispatcherHandler {
	return &NotificationDispatcherHandler{
		registry:     registry,
		notifService: notifService,
		publisher:    publisher,
	}
}

// Handle обрабатывает сообщение
func (h *NotificationDispatcherHandler) Handle(msg *message.Message) error {
	var envelope eventstore.EventEnvelope
	if err := json.Unmarshal(msg.Payload, &envelope); err != nil {
		slog.Error("failed to unmarshal event envelope", slog.String("error", err.Error()))
		return fmt.Errorf("unmarshal event envelope: %w", err)
	}

	evt, err := envelope.UnmarshalEvent()
	if err != nil {
		slog.Error("failed to unmarshal event", slog.String("error", err.Error()))
		return fmt.Errorf("unmarshal event: %w", err)
	}

	return h.processEvent(msg.Context(), evt)
}

func (h *NotificationDispatcherHandler) processEvent(ctx context.Context, evt eventstore.Event) error {
	// Получаем все уведомления от правил
	requests, err := h.registry.Process(ctx, evt)
	if err != nil {
		slog.Error("failed to process notification rules",
			"event_type", evt.EventType(),
			"error", err,
			"action", "skipped_retry")
		return nil // Не блокируем очередь
	}

	if len(requests) == 0 {
		return nil
	}

	slog.Info("processing notifications",
		slog.String("event_type", evt.EventType()),
		slog.Int("count", len(requests)))

	// Отправляем каждое уведомление
	for _, req := range requests {
		// Проверяем отмену контекста
		select {
		case <-ctx.Done():
			slog.Info("notification dispatch cancelled",
				slog.String("event_type", evt.EventType()))
			return ctx.Err()
		default:
		}

		if err := h.sendNotification(ctx, req); err != nil {
			slog.Warn("failed to send notification",
				"member_id", req.RecipientMemberID.String(),
				"type", string(req.NotificationType),
				"error", err,
				"action", "skipped_retry")
		}
	}

	return nil
}

func (h *NotificationDispatcherHandler) sendNotification(ctx context.Context, req rules.NotificationRequest) error {
	// Проверяем настройки для in_app
	shouldInApp, err := h.notifService.ShouldNotify(ctx, req.RecipientMemberID, req.NotificationType, values.ChannelInApp)
	if err != nil {
		slog.Warn("failed to check in_app settings, using default",
			slog.String("member_id", req.RecipientMemberID.String()),
			slog.String("error", err.Error()))
		shouldInApp = true
	}

	if shouldInApp {
		if err := h.notifService.CreateInApp(ctx, notifApp.CreateNotificationInput{
			MemberID:         req.RecipientMemberID,
			OrganizationID:   req.RecipientOrgID,
			NotificationType: req.NotificationType,
			Title:            req.Title,
			Body:             req.Body,
			Link:             req.Link,
			EntityType:       req.EntityType,
			EntityID:         req.EntityID,
		}); err != nil {
			slog.Error("failed to create in-app notification",
				slog.String("member_id", req.RecipientMemberID.String()),
				slog.String("error", err.Error()))
		} else {
			slog.Debug("in-app notification created",
				slog.String("member_id", req.RecipientMemberID.String()),
				slog.String("type", string(req.NotificationType)))
		}
	}

	// Проверяем настройки для telegram
	shouldTelegram, err := h.notifService.ShouldNotify(ctx, req.RecipientMemberID, req.NotificationType, values.ChannelTelegram)
	if err == nil && shouldTelegram {
		chatID, err := h.notifService.GetTelegramChatID(ctx, req.RecipientMemberID)
		if err == nil && chatID != nil {
			if err := h.publishTelegram(req, *chatID); err != nil {
				slog.Warn("failed to publish telegram notification",
					slog.String("member_id", req.RecipientMemberID.String()),
					slog.String("error", err.Error()))
			}
		}
	}

	// Проверяем настройки для email
	shouldEmail, err := h.notifService.ShouldSendEmail(ctx, req.RecipientMemberID, req.NotificationType)
	if err == nil && shouldEmail {
		email, err := h.notifService.GetMemberEmail(ctx, req.RecipientMemberID)
		if err == nil && email != nil {
			if err := h.publishEmail(req, *email); err != nil {
				slog.Warn("failed to publish email notification",
					slog.String("member_id", req.RecipientMemberID.String()),
					slog.String("error", err.Error()))
			}
		}
	}

	return nil
}

func (h *NotificationDispatcherHandler) publishTelegram(req rules.NotificationRequest, chatID int64) error {
	telegramMsg := TelegramNotification{
		MemberID: req.RecipientMemberID,
		ChatID:   chatID,
		Title:    req.Title,
		Body:     req.Body,
		Link:     req.Link,
	}

	payload, err := json.Marshal(telegramMsg)
	if err != nil {
		return fmt.Errorf("marshal telegram notification: %w", err)
	}

	msg := message.NewMessage(uuid.New().String(), payload)
	if err := h.publisher.Publish("notification.telegram", msg); err != nil {
		slog.Error("failed to publish telegram notification",
			slog.String("member_id", req.RecipientMemberID.String()),
			slog.String("error", err.Error()))
		return err
	}

	slog.Debug("telegram notification published",
		slog.String("member_id", req.RecipientMemberID.String()),
		slog.String("type", string(req.NotificationType)))

	return nil
}

func (h *NotificationDispatcherHandler) publishEmail(req rules.NotificationRequest, email string) error {
	emailMsg := EmailNotification{
		MemberID:         req.RecipientMemberID,
		Email:            email,
		NotificationType: string(req.NotificationType),
		Title:            req.Title,
		Body:             req.Body,
		Link:             req.Link,
	}

	payload, err := json.Marshal(emailMsg)
	if err != nil {
		return fmt.Errorf("marshal email notification: %w", err)
	}

	msg := message.NewMessage(uuid.New().String(), payload)
	if err := h.publisher.Publish("notification.email", msg); err != nil {
		slog.Error("failed to publish email notification",
			slog.String("member_id", req.RecipientMemberID.String()),
			slog.String("error", err.Error()))
		return err
	}

	slog.Debug("email notification published",
		slog.String("member_id", req.RecipientMemberID.String()),
		slog.String("email", email),
		slog.String("type", string(req.NotificationType)))

	return nil
}
