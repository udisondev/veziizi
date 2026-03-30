package handlers

import (
	"encoding/json"
	"fmt"
	"html"
	"log/slog"
	"strings"

	"github.com/udisondev/veziizi/backend/internal/domain/notification/values"
	"github.com/udisondev/veziizi/backend/internal/infrastructure/notifications"
	"github.com/udisondev/veziizi/backend/internal/infrastructure/projections"
	"github.com/udisondev/veziizi/backend/internal/pkg/config"
	"github.com/ThreeDotsLabs/watermill/message"
)

// EmailSenderHandler отправляет уведомления по Email
type EmailSenderHandler struct {
	provider    notifications.EmailProvider
	appConfig   *config.Config
	deliveryLog *projections.NotificationDeliveryLogProjection
}

// NewEmailSenderHandler создает новый handler
func NewEmailSenderHandler(
	provider notifications.EmailProvider,
	appConfig *config.Config,
	deliveryLog *projections.NotificationDeliveryLogProjection,
) *EmailSenderHandler {
	return &EmailSenderHandler{
		provider:    provider,
		appConfig:   appConfig,
		deliveryLog: deliveryLog,
	}
}

// Handle обрабатывает сообщение из очереди
func (h *EmailSenderHandler) Handle(msg *message.Message) error {
	var notification EmailNotification
	if err := json.Unmarshal(msg.Payload, &notification); err != nil {
		slog.Error("failed to unmarshal email notification",
			slog.String("error", err.Error()))
		return fmt.Errorf("unmarshal notification: %w", err)
	}

	// Формируем ссылку с доменом приложения
	link := ""
	if notification.Link != "" {
		baseURL := h.appConfig.App.BaseURL
		if baseURL == "" {
			baseURL = "https://veziizi.ru" // fallback
		}
		link = baseURL + notification.Link
	}

	// Формируем HTML содержимое письма
	bodyHTML := h.formatEmailHTML(notification.Title, notification.Body, link)
	bodyText := h.formatEmailText(notification.Title, notification.Body, link)

	// Создаем сообщение
	emailMsg := notifications.EmailMessage{
		To:        notification.Email,
		Subject:   notification.Title,
		BodyHTML:  bodyHTML,
		BodyText:  bodyText,
		EmailType: values.EmailTypeTransactional, // уведомления = транзакционные
	}

	// Отправляем email
	result, err := h.provider.Send(msg.Context(), emailMsg)
	if err != nil {
		slog.Error("failed to send email",
			slog.String("email", notification.Email),
			slog.String("member_id", notification.MemberID.String()),
			slog.String("error", err.Error()))

		// Логируем ошибку доставки
		if h.deliveryLog != nil {
			if logErr := h.deliveryLog.LogDelivery(msg.Context(), projections.DeliveryLogInput{
				MemberID:         notification.MemberID,
				NotificationType: notification.NotificationType,
				Channel:          "email",
				Status:           "failed",
				ErrorMessage:     err.Error(),
			}); logErr != nil {
				slog.Error("failed to log delivery failure",
					slog.String("member_id", notification.MemberID.String()),
					slog.String("error", logErr.Error()))
			}
		}

		// Возвращаем ошибку для retry
		return fmt.Errorf("send email: %w", err)
	}

	slog.Info("email sent",
		slog.String("message_id", result.MessageID),
		slog.String("email", notification.Email),
		slog.String("member_id", notification.MemberID.String()))

	// Логируем успешную доставку
	if h.deliveryLog != nil {
		if err := h.deliveryLog.LogDelivery(msg.Context(), projections.DeliveryLogInput{
			MemberID:         notification.MemberID,
			NotificationType: notification.NotificationType,
			Channel:          "email",
			Status:           "sent",
		}); err != nil {
			slog.Error("failed to log successful delivery",
				slog.String("member_id", notification.MemberID.String()),
				slog.String("error", err.Error()))
		}
	}

	return nil
}

// formatEmailHTML форматирует HTML письмо
func (h *EmailSenderHandler) formatEmailHTML(title, body, link string) string {
	// Экранируем пользовательский контент для защиты от HTML injection
	safeTitle := html.EscapeString(title)
	safeBody := html.EscapeString(body)

	linkSection := ""
	// Валидируем link — только http/https протоколы
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

	// BaseURL — контролируется конфигом, но экранируем на всякий случай
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

// formatEmailText форматирует текстовую версию письма
func (h *EmailSenderHandler) formatEmailText(title, body, link string) string {
	text := fmt.Sprintf("%s\n\n%s", title, body)
	if link != "" {
		text += fmt.Sprintf("\n\nОткрыть в приложении: %s", link)
	}
	text += fmt.Sprintf("\n\n---\nНастроить уведомления: %s/settings/notifications", h.appConfig.App.BaseURL)
	return text
}
