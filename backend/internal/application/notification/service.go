package notification

import (
	"context"
	"fmt"

	"codeberg.org/udison/veziizi/backend/internal/domain/notification/values"
	"codeberg.org/udison/veziizi/backend/internal/infrastructure/projections"
	"github.com/google/uuid"
)

// Service предоставляет операции над уведомлениями
type Service struct {
	preferences  *projections.NotificationPreferencesProjection
	inapp        *projections.InAppNotificationsProjection
	telegramLink *projections.TelegramLinkProjection
}

// NewService создает новый сервис уведомлений
func NewService(
	preferences *projections.NotificationPreferencesProjection,
	inapp *projections.InAppNotificationsProjection,
	telegramLink *projections.TelegramLinkProjection,
) *Service {
	return &Service{
		preferences:  preferences,
		inapp:        inapp,
		telegramLink: telegramLink,
	}
}

// ===============================
// Preferences (настройки)
// ===============================

// GetPreferencesResponse DTO для ответа API
type GetPreferencesResponse struct {
	MemberID          uuid.UUID                `json:"member_id"`
	EnabledCategories values.EnabledCategories `json:"enabled_categories"`
	Telegram          TelegramStatusResponse   `json:"telegram"`
	Email             EmailStatusResponse      `json:"email"`
}

// TelegramStatusResponse статус Telegram
type TelegramStatusResponse struct {
	Connected   bool    `json:"connected"`
	Username    *string `json:"username,omitempty"`
	ConnectedAt *string `json:"connected_at,omitempty"`
}

// GetPreferences возвращает настройки уведомлений member
func (s *Service) GetPreferences(ctx context.Context, memberID uuid.UUID) (*GetPreferencesResponse, error) {
	pref, err := s.preferences.GetOrCreateByMemberID(ctx, memberID)
	if err != nil {
		return nil, fmt.Errorf("get preferences: %w", err)
	}

	categories, err := pref.ParseEnabledCategories()
	if err != nil {
		// Используем дефолт если не удалось распарсить
		categories = values.DefaultEnabledCategories()
	}

	response := &GetPreferencesResponse{
		MemberID:          memberID,
		EnabledCategories: categories,
		Telegram: TelegramStatusResponse{
			Connected: pref.TelegramChatID != nil,
		},
		Email: EmailStatusResponse{
			Connected:        pref.Email != nil,
			Email:            pref.Email,
			Verified:         pref.EmailVerified,
			MarketingConsent: pref.EmailMarketingConsent,
		},
	}

	if pref.TelegramUsername != nil {
		response.Telegram.Username = pref.TelegramUsername
	}
	if pref.TelegramConnectedAt != nil {
		formatted := pref.TelegramConnectedAt.Format("2006-01-02T15:04:05Z")
		response.Telegram.ConnectedAt = &formatted
	}
	if pref.EmailVerifiedAt != nil {
		formatted := pref.EmailVerifiedAt.Format("2006-01-02T15:04:05Z")
		response.Email.VerifiedAt = &formatted
	}

	return response, nil
}

// UpdatePreferencesInput входные данные для обновления настроек
type UpdatePreferencesInput struct {
	Categories values.EnabledCategories `json:"categories"`
}

// UpdatePreferences обновляет настройки категорий
func (s *Service) UpdatePreferences(ctx context.Context, memberID uuid.UUID, input UpdatePreferencesInput) error {
	if err := s.preferences.UpdateCategories(ctx, memberID, input.Categories); err != nil {
		return fmt.Errorf("update categories: %w", err)
	}
	return nil
}

// ===============================
// Telegram
// ===============================

// ConnectTelegram подключает Telegram
func (s *Service) ConnectTelegram(ctx context.Context, memberID uuid.UUID, chatID int64, username string) error {
	if err := s.preferences.ConnectTelegram(ctx, memberID, chatID, username); err != nil {
		return fmt.Errorf("connect telegram: %w", err)
	}
	return nil
}

// DisconnectTelegram отключает Telegram
func (s *Service) DisconnectTelegram(ctx context.Context, memberID uuid.UUID) error {
	if err := s.preferences.DisconnectTelegram(ctx, memberID); err != nil {
		return fmt.Errorf("disconnect telegram: %w", err)
	}
	return nil
}

// ===============================
// Telegram Link Codes (привязка через бота)
// ===============================

// GenerateLinkCodeResponse ответ с кодом привязки
type GenerateLinkCodeResponse struct {
	Code      string `json:"code"`
	ExpiresIn int    `json:"expires_in"` // секунды
	BotURL    string `json:"bot_url"`
}

// GenerateLinkCode генерирует код для привязки Telegram через бота
func (s *Service) GenerateLinkCode(ctx context.Context, memberID uuid.UUID, botUsername string) (*GenerateLinkCodeResponse, error) {
	code, err := s.telegramLink.GenerateCode(ctx, memberID)
	if err != nil {
		return nil, fmt.Errorf("generate link code: %w", err)
	}

	return &GenerateLinkCodeResponse{
		Code:      code,
		ExpiresIn: int(projections.LinkCodeTTL.Seconds()),
		BotURL:    fmt.Sprintf("https://t.me/%s?start=%s", botUsername, code),
	}, nil
}

// ConfirmLinkCode подтверждает код привязки (вызывается ботом)
// Идемпотентный: если chatID уже подключён, возвращает успех
func (s *Service) ConfirmLinkCode(ctx context.Context, code string, chatID int64, username string) error {
	// Получаем код
	linkCode, err := s.telegramLink.GetByCode(ctx, code)
	if err != nil {
		// Код не найден или истёк — проверяем, может chatID уже подключён (идемпотентность)
		connected, checkErr := s.preferences.IsChatIDConnected(ctx, chatID)
		if checkErr != nil {
			return fmt.Errorf("invalid or expired code")
		}
		if connected {
			// Уже подключён — успех (идемпотентность)
			return nil
		}
		return fmt.Errorf("invalid or expired code")
	}

	// Сначала удаляем код (чтобы повторный запрос не прошёл до IsChatIDConnected)
	// Ошибка удаления не критична — код всё равно истечёт по TTL
	_ = s.telegramLink.DeleteByCode(ctx, code)

	// Подключаем Telegram
	if err := s.preferences.ConnectTelegram(ctx, linkCode.MemberID, chatID, username); err != nil {
		return fmt.Errorf("connect telegram: %w", err)
	}

	return nil
}

// ===============================
// In-App Notifications
// ===============================

// ListNotificationsInput входные параметры для списка уведомлений
type ListNotificationsInput struct {
	Category *values.NotificationCategory
	IsRead   *bool
	Limit    int
	Offset   int
}

// ListNotifications возвращает список уведомлений
func (s *Service) ListNotifications(ctx context.Context, memberID uuid.UUID, input ListNotificationsInput) ([]projections.InAppNotificationLookup, error) {
	filter := projections.InAppNotificationListFilter{
		Category: input.Category,
		IsRead:   input.IsRead,
		Limit:    input.Limit,
		Offset:   input.Offset,
	}

	notifications, err := s.inapp.List(ctx, memberID, filter)
	if err != nil {
		return nil, fmt.Errorf("list notifications: %w", err)
	}

	return notifications, nil
}

// GetUnreadCount возвращает количество непрочитанных
func (s *Service) GetUnreadCount(ctx context.Context, memberID uuid.UUID) (int, error) {
	count, err := s.inapp.GetUnreadCount(ctx, memberID)
	if err != nil {
		return 0, fmt.Errorf("get unread count: %w", err)
	}
	return count, nil
}

// MarkAsReadInput входные данные для пометки прочитанными
type MarkAsReadInput struct {
	NotificationIDs []uuid.UUID `json:"notification_ids"`
}

// MarkAsRead помечает уведомления как прочитанные
func (s *Service) MarkAsRead(ctx context.Context, memberID uuid.UUID, input MarkAsReadInput) error {
	if err := s.inapp.MarkAsRead(ctx, memberID, input.NotificationIDs); err != nil {
		return fmt.Errorf("mark as read: %w", err)
	}
	return nil
}

// MarkAllAsRead помечает все уведомления как прочитанные
func (s *Service) MarkAllAsRead(ctx context.Context, memberID uuid.UUID) error {
	if err := s.inapp.MarkAllAsRead(ctx, memberID); err != nil {
		return fmt.Errorf("mark all as read: %w", err)
	}
	return nil
}

// ===============================
// Create Notification (для dispatcher)
// ===============================

// CreateNotificationInput входные данные для создания уведомления
type CreateNotificationInput struct {
	MemberID         uuid.UUID
	OrganizationID   uuid.UUID
	NotificationType values.NotificationType
	Title            string
	Body             string
	Link             string
	EntityType       values.EntityType
	EntityID         uuid.UUID
}

// CreateInApp создает in-app уведомление
func (s *Service) CreateInApp(ctx context.Context, input CreateNotificationInput) error {
	projInput := projections.CreateNotificationInput{
		ID:               uuid.New(),
		MemberID:         input.MemberID,
		OrganizationID:   input.OrganizationID,
		NotificationType: input.NotificationType,
		Title:            input.Title,
		Body:             input.Body,
		Link:             input.Link,
		EntityType:       input.EntityType,
		EntityID:         input.EntityID,
	}

	if err := s.inapp.Insert(ctx, projInput); err != nil {
		return fmt.Errorf("create notification: %w", err)
	}

	return nil
}

// ===============================
// Check Preferences (для dispatcher)
// ===============================

// ShouldNotify проверяет, нужно ли отправлять уведомление member
func (s *Service) ShouldNotify(ctx context.Context, memberID uuid.UUID, notifType values.NotificationType, channel values.NotificationChannel) (bool, error) {
	pref, err := s.preferences.GetByMemberID(ctx, memberID)
	if err != nil {
		// Нет настроек - используем дефолт (in_app = true, telegram = false)
		defaults := values.DefaultEnabledCategories()
		return defaults.IsEnabled(notifType.Category(), channel), nil
	}

	categories, err := pref.ParseEnabledCategories()
	if err != nil {
		// Ошибка парсинга - используем дефолт
		defaults := values.DefaultEnabledCategories()
		return defaults.IsEnabled(notifType.Category(), channel), nil
	}

	// Для Telegram дополнительно проверяем что подключен
	if channel == values.ChannelTelegram && pref.TelegramChatID == nil {
		return false, nil
	}

	return categories.IsEnabled(notifType.Category(), channel), nil
}

// GetTelegramChatID возвращает chat ID для отправки в Telegram
func (s *Service) GetTelegramChatID(ctx context.Context, memberID uuid.UUID) (*int64, error) {
	return s.preferences.GetTelegramChatID(ctx, memberID)
}

// ===============================
// Email
// ===============================

// EmailStatusResponse статус Email
type EmailStatusResponse struct {
	Connected        bool    `json:"connected"`
	Email            *string `json:"email,omitempty"`
	Verified         bool    `json:"verified"`
	VerifiedAt       *string `json:"verified_at,omitempty"`
	MarketingConsent bool    `json:"marketing_consent"`
}

// GetEmailStatus возвращает статус email для member
func (s *Service) GetEmailStatus(ctx context.Context, memberID uuid.UUID) (*EmailStatusResponse, error) {
	pref, err := s.preferences.GetOrCreateByMemberID(ctx, memberID)
	if err != nil {
		return nil, fmt.Errorf("get preferences: %w", err)
	}

	response := &EmailStatusResponse{
		Connected:        pref.Email != nil,
		Email:            pref.Email,
		Verified:         pref.EmailVerified,
		MarketingConsent: pref.EmailMarketingConsent,
	}

	if pref.EmailVerifiedAt != nil {
		formatted := pref.EmailVerifiedAt.Format("2006-01-02T15:04:05Z")
		response.VerifiedAt = &formatted
	}

	return response, nil
}

// SetEmailInput входные данные для установки email
type SetEmailInput struct {
	Email string `json:"email" validate:"required,email"`
}

// SetEmail устанавливает email для уведомлений (требует верификации)
func (s *Service) SetEmail(ctx context.Context, memberID uuid.UUID, input SetEmailInput) error {
	if err := s.preferences.SetEmail(ctx, memberID, input.Email); err != nil {
		return fmt.Errorf("set email: %w", err)
	}
	return nil
}

// VerifyEmail подтверждает email
func (s *Service) VerifyEmail(ctx context.Context, memberID uuid.UUID) error {
	if err := s.preferences.VerifyEmail(ctx, memberID); err != nil {
		return fmt.Errorf("verify email: %w", err)
	}
	return nil
}

// DisconnectEmail отключает email для уведомлений
func (s *Service) DisconnectEmail(ctx context.Context, memberID uuid.UUID) error {
	if err := s.preferences.DisconnectEmail(ctx, memberID); err != nil {
		return fmt.Errorf("disconnect email: %w", err)
	}
	return nil
}

// SetMarketingConsentInput входные данные для согласия на маркетинг
type SetMarketingConsentInput struct {
	Consent bool `json:"consent"`
}

// SetMarketingConsent устанавливает согласие на маркетинговые рассылки
func (s *Service) SetMarketingConsent(ctx context.Context, memberID uuid.UUID, input SetMarketingConsentInput) error {
	if err := s.preferences.SetMarketingConsent(ctx, memberID, input.Consent); err != nil {
		return fmt.Errorf("set marketing consent: %w", err)
	}
	return nil
}

// ResendEmailVerification повторно отправляет письмо верификации
// TODO: реализовать отправку письма с кодом верификации
func (s *Service) ResendEmailVerification(ctx context.Context, memberID uuid.UUID) error {
	pref, err := s.preferences.GetByMemberID(ctx, memberID)
	if err != nil {
		return fmt.Errorf("get preferences: %w", err)
	}

	if pref.Email == nil {
		return fmt.Errorf("email not set")
	}

	if pref.EmailVerified {
		return fmt.Errorf("email already verified")
	}

	// TODO: отправить письмо с кодом верификации
	// Пока просто возвращаем успех — реальная отправка будет реализована
	// когда будет готова система email-верификации

	return nil
}

// GetMemberEmail возвращает email для member (если установлен и верифицирован)
func (s *Service) GetMemberEmail(ctx context.Context, memberID uuid.UUID) (*string, error) {
	pref, err := s.preferences.GetByMemberID(ctx, memberID)
	if err != nil {
		return nil, nil // нет настроек — нет email
	}

	if pref.Email == nil || !pref.EmailVerified {
		return nil, nil // email не установлен или не верифицирован
	}

	return pref.Email, nil
}

// ShouldSendEmail проверяет, нужно ли отправлять email уведомление
// Проверяет: email установлен, верифицирован, категория включена
func (s *Service) ShouldSendEmail(ctx context.Context, memberID uuid.UUID, notifType values.NotificationType) (bool, error) {
	pref, err := s.preferences.GetByMemberID(ctx, memberID)
	if err != nil {
		return false, nil // нет настроек — не отправляем
	}

	// Email должен быть установлен и верифицирован
	if pref.Email == nil || !pref.EmailVerified {
		return false, nil
	}

	// Проверяем настройки категории
	categories, err := pref.ParseEnabledCategories()
	if err != nil {
		return false, nil
	}

	return categories.IsEnabled(notifType.Category(), values.ChannelEmail), nil
}
