package handlers

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/udisondev/veziizi/backend/internal/domain/support/entities"
	"github.com/udisondev/veziizi/backend/internal/domain/support/events"
	"github.com/udisondev/veziizi/backend/internal/infrastructure/messaging"
	"github.com/udisondev/veziizi/backend/internal/infrastructure/persistence/admin"
)

// AdminTelegramGetter интерфейс для получения админов с Telegram
type AdminTelegramGetter interface {
	GetAdminsWithTelegram(ctx context.Context) ([]admin.AdminWithTelegram, error)
}

// SupportAdminNotifierHandler — CQRS-handlers: подписан на support.events,
// уведомляет админов в Telegram о новых тикетах и пользовательских сообщениях.
// Запускается в отдельном воркере support-tickets-notifier с собственной
// consumer group — сбой отправки не должен блокировать обновление проекции
// support-tickets-projection.
type SupportAdminNotifierHandler struct {
	adminRepo AdminTelegramGetter
	notifBus  *messaging.NotificationBus
}

func NewSupportAdminNotifierHandler(
	adminRepo AdminTelegramGetter,
	notifBus *messaging.NotificationBus,
) *SupportAdminNotifierHandler {
	return &SupportAdminNotifierHandler{
		adminRepo: adminRepo,
		notifBus:  notifBus,
	}
}

// OnTicketCreated — рассылка админам о новом тикете.
func (h *SupportAdminNotifierHandler) OnTicketCreated(ctx context.Context, e *events.TicketCreated) error {
	admins, err := h.adminRepo.GetAdminsWithTelegram(ctx)
	if err != nil {
		slog.Error("failed to get admins with telegram", slog.String("error", err.Error()))
		// Возвращаем nil: иначе очередь застрянет на проблеме с БД админов,
		// которая ортогональна созданию тикета. PoisonQueue middleware ловит
		// только persistent fail — здесь это transient (БД может ожить).
		// TODO(observability): добавить counter в metrics.
		return nil
	}
	if len(admins) == 0 {
		slog.Debug("no admins with telegram to notify")
		return nil
	}

	title := fmt.Sprintf("Новый тикет #%d", e.TicketNumber)
	body := fmt.Sprintf("Тема: %s", e.Subject)
	link := fmt.Sprintf("/admin/support/%s", e.AggregateID().String())

	for _, a := range admins {
		h.publish(ctx, a, title, body, link)
	}

	slog.Info("admin notifications sent for new ticket",
		slog.Int64("ticket_number", e.TicketNumber),
		slog.Int("admin_count", len(admins)))
	return nil
}

// OnMessageAdded — рассылка только если пишет пользователь (не админ).
func (h *SupportAdminNotifierHandler) OnMessageAdded(ctx context.Context, e *events.MessageAdded) error {
	if e.SenderType != entities.SenderTypeUser {
		return nil
	}
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
		h.publish(ctx, a, title, body, link)
	}
	return nil
}

func (h *SupportAdminNotifierHandler) publish(ctx context.Context, a admin.AdminWithTelegram, title, body, link string) {
	if err := h.notifBus.Publish(ctx, messaging.TelegramNotification{
		MemberID: a.ID,
		ChatID:   a.TelegramChatID,
		Title:    title,
		Body:     body,
		Link:     link,
	}); err != nil {
		slog.Warn("failed to publish admin notification",
			slog.String("admin_id", a.ID.String()),
			slog.String("error", err.Error()))
	}
}

func truncateContent(content string, maxLen int) string {
	runes := []rune(content)
	if len(runes) <= maxLen {
		return content
	}
	return string(runes[:maxLen]) + "..."
}
