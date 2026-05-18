package handlers

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/ThreeDotsLabs/watermill/components/cqrs"
	"github.com/google/uuid"

	"github.com/udisondev/veziizi/backend/internal/infrastructure/messaging"
	"github.com/udisondev/veziizi/backend/internal/infrastructure/notifications"
	"github.com/udisondev/veziizi/backend/internal/infrastructure/projections"
	"github.com/udisondev/veziizi/backend/internal/pkg/config"
)

// TelegramSenderHandler отправляет уведомления в Telegram. Регистрируется как
// cqrs.NewEventHandler[messaging.TelegramNotification] в воркере telegram-sender.
//
// Идемпотентность: при at-least-once доставке тот же message UUID может
// прийти повторно (рестарт воркера между внешним вызовом и Ack). Перед
// SendMessage берём резерв в notification_dedup; при дубле — return nil.
type TelegramSenderHandler struct {
	client      *notifications.TelegramClient
	appConfig   *config.Config
	deliveryLog *projections.NotificationDeliveryLogProjection
	dedup       *projections.NotificationDedupProjection
}

func NewTelegramSenderHandler(
	client *notifications.TelegramClient,
	appConfig *config.Config,
	deliveryLog *projections.NotificationDeliveryLogProjection,
	dedup *projections.NotificationDedupProjection,
) *TelegramSenderHandler {
	return &TelegramSenderHandler{
		client:      client,
		appConfig:   appConfig,
		deliveryLog: deliveryLog,
		dedup:       dedup,
	}
}

// OnTelegramNotification — CQRS-handler. Возвращает error для retry/DLQ при
// сбое отправки. Доставка логируется в notification_delivery_log в обоих исходах.
func (h *TelegramSenderHandler) OnTelegramNotification(ctx context.Context, n *messaging.TelegramNotification) error {
	// Резервируем message UUID за каналом ДО внешнего вызова.
	// Если уже был — повторная доставка, не шлём в Telegram ещё раз.
	msg := cqrs.OriginalMessageFromCtx(ctx)
	msgUUID, err := uuid.Parse(msg.UUID)
	if err != nil {
		return fmt.Errorf("parse message uuid: %w", err)
	}
	first, err := h.dedup.MarkSent(ctx, msgUUID, "telegram")
	if err != nil {
		return fmt.Errorf("dedup mark sent: %w", err)
	}
	if !first {
		slog.Debug("telegram notification already sent, skipping",
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

	text := notifications.FormatNotification(n.Title, n.Body)
	if err := h.client.SendMessageWithButton(n.ChatID, text, "Открыть в приложении", link); err != nil {
		slog.Error("failed to send telegram message",
			slog.Int64("chat_id", n.ChatID),
			slog.String("member_id", n.MemberID.String()),
			slog.String("error", err.Error()))

		if h.deliveryLog != nil {
			if logErr := h.deliveryLog.LogDelivery(ctx, projections.DeliveryLogInput{
				MemberID:         n.MemberID,
				NotificationType: "telegram",
				Channel:          "telegram",
				Status:           "failed",
				ErrorMessage:     err.Error(),
			}); logErr != nil {
				slog.Error("failed to log delivery failure",
					slog.String("member_id", n.MemberID.String()),
					slog.String("error", logErr.Error()))
			}
		}
		return fmt.Errorf("send message: %w", err)
	}

	slog.Info("telegram message sent",
		slog.Int64("chat_id", n.ChatID),
		slog.String("member_id", n.MemberID.String()))

	if h.deliveryLog != nil {
		if err := h.deliveryLog.LogDelivery(ctx, projections.DeliveryLogInput{
			MemberID:         n.MemberID,
			NotificationType: "telegram",
			Channel:          "telegram",
			Status:           "sent",
		}); err != nil {
			slog.Error("failed to log successful delivery",
				slog.String("member_id", n.MemberID.String()),
				slog.String("error", err.Error()))
		}
	}
	return nil
}
