package handlers

import (
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/udisondev/veziizi/backend/internal/infrastructure/notifications"
	"github.com/udisondev/veziizi/backend/internal/infrastructure/projections"
	"github.com/udisondev/veziizi/backend/internal/pkg/config"
)

// TelegramSenderHandler отправляет уведомления в Telegram
type TelegramSenderHandler struct {
	client      *notifications.TelegramClient
	appConfig   *config.Config
	deliveryLog *projections.NotificationDeliveryLogProjection
}

// NewTelegramSenderHandler создает новый handler
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

// Handle обрабатывает сообщение из очереди
func (h *TelegramSenderHandler) Handle(msg *message.Message) error {
	var notification TelegramNotification
	if err := json.Unmarshal(msg.Payload, &notification); err != nil {
		slog.Error("failed to unmarshal telegram notification",
			slog.String("error", err.Error()))
		return fmt.Errorf("unmarshal notification: %w", err)
	}

	// Формируем ссылку с доменом приложения
	link := ""
	if notification.Link != "" {
		// Используем APP_BASE_URL из конфига если есть
		baseURL := h.appConfig.App.BaseURL
		if baseURL == "" {
			baseURL = "https://veziizi.ru" // fallback
		}
		link = baseURL + notification.Link
	}

	// Форматируем сообщение (без ссылки — она в кнопке)
	text := notifications.FormatNotification(notification.Title, notification.Body)

	// Отправляем с inline кнопкой
	if err := h.client.SendMessageWithButton(notification.ChatID, text, "Открыть в приложении", link); err != nil {
		slog.Error("failed to send telegram message",
			slog.Int64("chat_id", notification.ChatID),
			slog.String("member_id", notification.MemberID.String()),
			slog.String("error", err.Error()))

		// Логируем ошибку доставки
		if h.deliveryLog != nil {
			if logErr := h.deliveryLog.LogDelivery(msg.Context(), projections.DeliveryLogInput{
				MemberID:         notification.MemberID,
				NotificationType: "telegram",
				Channel:          "telegram",
				Status:           "failed",
				ErrorMessage:     err.Error(),
			}); logErr != nil {
				slog.Error("failed to log delivery failure",
					slog.String("member_id", notification.MemberID.String()),
					slog.String("error", logErr.Error()))
			}
		}

		// Возвращаем ошибку для retry
		return fmt.Errorf("send message: %w", err)
	}

	slog.Info("telegram message sent",
		slog.Int64("chat_id", notification.ChatID),
		slog.String("member_id", notification.MemberID.String()))

	// Логируем успешную доставку
	if h.deliveryLog != nil {
		if err := h.deliveryLog.LogDelivery(msg.Context(), projections.DeliveryLogInput{
			MemberID:         notification.MemberID,
			NotificationType: "telegram",
			Channel:          "telegram",
			Status:           "sent",
		}); err != nil {
			slog.Error("failed to log successful delivery",
				slog.String("member_id", notification.MemberID.String()),
				slog.String("error", err.Error()))
		}
	}

	return nil
}
