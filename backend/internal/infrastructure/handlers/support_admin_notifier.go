package handlers

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/udisondev/veziizi/backend/internal/domain/support/entities"
	"github.com/udisondev/veziizi/backend/internal/domain/support/events"
	"github.com/udisondev/veziizi/backend/internal/infrastructure/messaging"
	"github.com/udisondev/veziizi/backend/internal/infrastructure/persistence/admin"
	"github.com/udisondev/veziizi/backend/internal/infrastructure/projections"
	"github.com/udisondev/veziizi/backend/internal/pkg/dbtx"
)

// AdminTelegramGetter интерфейс для получения админов с Telegram
type AdminTelegramGetter interface {
	GetAdminsWithTelegram(ctx context.Context) ([]admin.AdminWithTelegram, error)
}

const supportAdminNotifierName = "support-admin-notifier"

// SupportAdminNotifierHandler — CQRS-handlers: подписан на support.events,
// уведомляет админов в Telegram о новых тикетах и пользовательских сообщениях.
// Запускается в отдельном воркере support-tickets-notifier с собственной
// consumer group — сбой отправки не должен блокировать обновление проекции
// support-tickets-projection.
//
// Обработка обёрнута в dedupGuard: каждая публикация в NotificationBus
// генерирует новый message UUID, поэтому notification_dedup в telegram-sender
// не распознал бы повторную at-least-once доставку — админы получили бы дубль.
type SupportAdminNotifierHandler struct {
	db        dbtx.TxManager
	dedup     *projections.ProjectionEventDedupProjection
	adminRepo AdminTelegramGetter
	notifBus  *messaging.NotificationBus
}

func NewSupportAdminNotifierHandler(
	db dbtx.TxManager,
	dedup *projections.ProjectionEventDedupProjection,
	adminRepo AdminTelegramGetter,
	notifBus *messaging.NotificationBus,
) *SupportAdminNotifierHandler {
	return &SupportAdminNotifierHandler{
		db:        db,
		dedup:     dedup,
		adminRepo: adminRepo,
		notifBus:  notifBus,
	}
}

// withDedup — тонкая обёртка над dedupGuard с event_id из CQRS-контекста.
func (h *SupportAdminNotifierHandler) withDedup(ctx context.Context, fn func(ctx context.Context) error) error {
	eventID, err := eventIDFromCtx(ctx)
	if err != nil {
		return err
	}
	return dedupGuard(ctx, h.db, h.dedup, supportAdminNotifierName, eventID, fn)
}

// OnTicketCreated — рассылка админам о новом тикете.
func (h *SupportAdminNotifierHandler) OnTicketCreated(ctx context.Context, e *events.TicketCreated) error {
	return h.withDedup(ctx, func(ctx context.Context) error {
		return h.notifyTicketCreated(ctx, e)
	})
}

func (h *SupportAdminNotifierHandler) notifyTicketCreated(ctx context.Context, e *events.TicketCreated) error {
	admins, err := h.adminRepo.GetAdminsWithTelegram(ctx)
	if err != nil {
		// Ошибку обязаны пробросить: мы внутри dedup-tx, return nil закоммитил бы
		// dedup-резерв и redelivery было бы подавлено — уведомление о новом тикете
		// потерялось бы навсегда. Rollback снимает резерв → retry.
		return fmt.Errorf("get admins with telegram: %w", err)
	}
	if len(admins) == 0 {
		slog.Debug("no admins with telegram to notify")
		return nil
	}

	title := fmt.Sprintf("Новый тикет #%d", e.TicketNumber)
	body := fmt.Sprintf("Тема: %s", e.Subject)
	link := fmt.Sprintf("/admin/support/%s", e.AggregateID().String())

	for _, a := range admins {
		if err := h.publish(ctx, a, title, body, link); err != nil {
			return err
		}
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
	return h.withDedup(ctx, func(ctx context.Context) error {
		return h.notifyMessageAdded(ctx, e)
	})
}

func (h *SupportAdminNotifierHandler) notifyMessageAdded(ctx context.Context, e *events.MessageAdded) error {
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
		if err := h.publish(ctx, a, title, body, link); err != nil {
			return err
		}
	}
	return nil
}

// publish — ошибка публикации пробрасывается наружу: мы внутри dedup-tx,
// проглоченная ошибка = закоммиченный dedup-резерв = потерянное уведомление.
// NotificationBus пишет в outbox той же tx, так что rollback откатит и уже
// опубликованные в этом цикле сообщения — дублей при retry не будет.
func (h *SupportAdminNotifierHandler) publish(ctx context.Context, a admin.AdminWithTelegram, title, body, link string) error {
	if err := h.notifBus.Publish(ctx, messaging.TelegramNotification{
		MemberID: a.ID,
		ChatID:   a.TelegramChatID,
		Title:    title,
		Body:     body,
		Link:     link,
	}); err != nil {
		return fmt.Errorf("publish admin notification for %s: %w", a.ID, err)
	}
	return nil
}

func truncateContent(content string, maxLen int) string {
	runes := []rune(content)
	if len(runes) <= maxLen {
		return content
	}
	return string(runes[:maxLen]) + "..."
}
