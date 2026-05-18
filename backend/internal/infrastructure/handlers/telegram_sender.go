package handlers

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/udisondev/veziizi/backend/internal/infrastructure/messaging"
	"github.com/udisondev/veziizi/backend/internal/infrastructure/notifications"
	"github.com/udisondev/veziizi/backend/internal/infrastructure/projections"
	"github.com/udisondev/veziizi/backend/internal/pkg/config"
)

// TelegramSenderHandler отправляет уведомления в Telegram. Регистрируется как
// cqrs.NewEventHandler[messaging.TelegramNotification] в воркере telegram-sender.
type TelegramSenderHandler struct {
	client      *notifications.TelegramClient
	appConfig   *config.Config
	deliveryLog *projections.NotificationDeliveryLogProjection
}

func NewTelegramSenderHandler(
	client *notifications.TelegramClient,
	appConfig *config.Config,
	deliveryLog *projections.NotificationDeliveryLogProjection,
) *TelegramSenderHandler {
	return &TelegramSenderHandler{
		client:      client,
		appConfig:   appConfig,
		deliveryLog: deliveryLog,
	}
}

// OnTelegramNotification — CQRS-handler. Возвращает error для retry/DLQ при
// сбое отправки. Доставка логируется в notification_delivery_log в обоих исходах.
func (h *TelegramSenderHandler) OnTelegramNotification(ctx context.Context, n *messaging.TelegramNotification) error {
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
