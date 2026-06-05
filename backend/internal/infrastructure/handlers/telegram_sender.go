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
// прийти повторно (рестарт воркера между внешним вызовом и Ack). До
// SendMessage проверяем notification_dedup.IsSent; после успешной отправки
// фиксируем MarkSent. Транзиентная ошибка → возвращаем err, Retry middleware
// перезапустит handler без потери уведомления.
// TelegramSender — отправка сообщения в Telegram. *notifications.TelegramClient
// реализует интерфейс; e2e подменяет записывающим fake'ом (suite не должен
// ходить в api.telegram.org).
type TelegramSender interface {
	SendMessageWithButton(chatID int64, text, buttonText, buttonURL string) error
}

type TelegramSenderHandler struct {
	client      TelegramSender
	appConfig   *config.Config
	deliveryLog *projections.NotificationDeliveryLogProjection
	dedup       *projections.NotificationDedupProjection
}

func NewTelegramSenderHandler(
	client TelegramSender,
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
	// До внешнего вызова: если message UUID уже отмечен как отправленный,
	// это повторная доставка watermill после успешного предыдущего Send.
	msg := cqrs.OriginalMessageFromCtx(ctx)
	msgUUID, err := uuid.Parse(msg.UUID)
	if err != nil {
		return fmt.Errorf("parse message uuid: %w", err)
	}
	sent, err := h.dedup.IsSent(ctx, msgUUID, "telegram")
	if err != nil {
		return fmt.Errorf("dedup is sent: %w", err)
	}
	if sent {
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

	// Фиксируем sent ПОСЛЕ успешного API-вызова. Ошибка БД здесь не должна
	// триггерить retry — Telegram сообщение уже доставлено, повторная
	// отправка хуже потери строки в notification_dedup.
	if err := h.dedup.MarkSent(ctx, msgUUID, "telegram"); err != nil {
		slog.Error("failed to mark notification as sent",
			slog.String("message_uuid", msg.UUID),
			slog.String("member_id", n.MemberID.String()),
			slog.String("error", err.Error()))
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
