package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/udisondev/veziizi/backend/internal/domain/support/entities"
	"github.com/udisondev/veziizi/backend/internal/domain/support/events"
	"github.com/udisondev/veziizi/backend/internal/infrastructure/persistence/admin"
	"github.com/udisondev/veziizi/backend/internal/infrastructure/persistence/eventstore"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"
)

// AdminTelegramGetter интерфейс для получения админов с Telegram
type AdminTelegramGetter interface {
	GetAdminsWithTelegram(ctx context.Context) ([]admin.AdminWithTelegram, error)
}

// SupportAdminNotifierHandler отправляет уведомления админам о тикетах
type SupportAdminNotifierHandler struct {
	adminRepo AdminTelegramGetter
	publisher message.Publisher
}

// NewSupportAdminNotifierHandler создает новый handler
func NewSupportAdminNotifierHandler(
	adminRepo AdminTelegramGetter,
	publisher message.Publisher,
) *SupportAdminNotifierHandler {
	return &SupportAdminNotifierHandler{
		adminRepo: adminRepo,
		publisher: publisher,
	}
}

// Handle обрабатывает событие и отправляет уведомления админам
func (h *SupportAdminNotifierHandler) Handle(msg *message.Message) error {
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

	return h.handleEvent(msg.Context(), evt)
}

func (h *SupportAdminNotifierHandler) handleEvent(ctx context.Context, evt eventstore.Event) error {
	switch e := evt.(type) {
	case events.TicketCreated:
		return h.onTicketCreated(ctx, e)
	case events.MessageAdded:
		// Уведомляем админов только о сообщениях от пользователей
		if e.SenderType == entities.SenderTypeUser {
			return h.onUserMessageAdded(ctx, e)
		}
	}
	return nil
}

func (h *SupportAdminNotifierHandler) onTicketCreated(ctx context.Context, e events.TicketCreated) error {
	admins, err := h.adminRepo.GetAdminsWithTelegram(ctx)
	if err != nil {
		slog.Error("failed to get admins with telegram",
			slog.String("error", err.Error()))
		return nil // Не блокируем очередь
	}

	if len(admins) == 0 {
		slog.Debug("no admins with telegram to notify")
		return nil
	}

	title := fmt.Sprintf("Новый тикет #%d", e.TicketNumber)
	body := fmt.Sprintf("Тема: %s", e.Subject)
	link := fmt.Sprintf("/admin/support/%s", e.AggregateID().String())

	for _, a := range admins {
		if err := h.sendTelegramNotification(ctx, a, title, body, link); err != nil {
			slog.Warn("failed to send admin notification",
				slog.String("admin_id", a.ID.String()),
				slog.String("error", err.Error()))
		}
	}

	slog.Info("admin notifications sent for new ticket",
		slog.Int64("ticket_number", e.TicketNumber),
		slog.Int("admin_count", len(admins)))

	return nil
}

func (h *SupportAdminNotifierHandler) onUserMessageAdded(ctx context.Context, e events.MessageAdded) error {
	admins, err := h.adminRepo.GetAdminsWithTelegram(ctx)
	if err != nil {
		return fmt.Errorf("get admins with telegram: %w", err)
	}

	if len(admins) == 0 {
		return nil
	}

	title := "Новое сообщение в тикете"
	body := truncateContent(e.Content, 100)
	link := fmt.Sprintf("/admin/support/%s", e.AggregateID().String())

	for _, a := range admins {
		if err := h.sendTelegramNotification(ctx, a, title, body, link); err != nil {
			slog.Warn("failed to send admin notification",
				slog.String("admin_id", a.ID.String()),
				slog.String("error", err.Error()))
		}
	}

	return nil
}

func (h *SupportAdminNotifierHandler) sendTelegramNotification(
	ctx context.Context,
	adm admin.AdminWithTelegram,
	title, body, link string,
) error {
	notification := TelegramNotification{
		MemberID: adm.ID, // Используем admin ID для логирования
		ChatID:   adm.TelegramChatID,
		Title:    title,
		Body:     body,
		Link:     link,
	}

	payload, err := json.Marshal(notification)
	if err != nil {
		return fmt.Errorf("marshal notification: %w", err)
	}

	msg := message.NewMessage(uuid.New().String(), payload)
	if err := h.publisher.Publish("notification.telegram", msg); err != nil {
		return fmt.Errorf("publish notification: %w", err)
	}

	return nil
}

// truncateContent обрезает текст до указанной длины
func truncateContent(content string, maxLen int) string {
	runes := []rune(content)
	if len(runes) <= maxLen {
		return content
	}
	return string(runes[:maxLen]) + "..."
}
