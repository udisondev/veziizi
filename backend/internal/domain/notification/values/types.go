package values

// NotificationType определяет тип уведомления
type NotificationType string

const (
	// Freight Requests (заявки)
	TypeNewFreightRequest NotificationType = "new_freight_request" // Опубликована новая заявка

	// Offers (предложения)
	TypeNewOffer       NotificationType = "new_offer"       // Получено новое предложение на заявку
	TypeOfferSelected  NotificationType = "offer_selected"  // Ваше предложение выбрано
	TypeOfferRejected  NotificationType = "offer_rejected"  // Ваше предложение отклонено
	TypeOfferConfirmed NotificationType = "offer_confirmed" // Предложение подтверждено
	TypeOfferDeclined  NotificationType = "offer_declined"  // Выбранное предложение отклонено перевозчиком
	TypeOfferWithdrawn NotificationType = "offer_withdrawn" // Предложение отозвано

	// Freight completion (завершение перевозки)
	TypeFreightCompleted          NotificationType = "freight_completed"           // Перевозка завершена
	TypeFreightCancelledConfirmed NotificationType = "freight_cancelled_confirmed" // Перевозка отменена после подтверждения

	// Orders (заказы) - deprecated, kept for backwards compatibility
	TypeOrderCreated   NotificationType = "order_created"   // Заказ создан
	TypeOrderMessage   NotificationType = "order_message"   // Новое сообщение в заказе
	TypeOrderDocument  NotificationType = "order_document"  // Новый документ в заказе
	TypeOrderCompleted NotificationType = "order_completed" // Заказ завершён
	TypeOrderCancelled NotificationType = "order_cancelled" // Заказ отменён

	// Reviews (отзывы)
	TypeReviewReceived NotificationType = "review_received" // Получен отзыв

	// Organization (организация)
	TypeMemberInvited    NotificationType = "member_invited"     // Приглашение в организацию
	TypeMemberJoined     NotificationType = "member_joined"      // Сотрудник присоединился
	TypeOrgStatusChanged NotificationType = "org_status_changed" // Статус организации изменился
)

// Category возвращает категорию для типа уведомления
func (t NotificationType) Category() NotificationCategory {
	switch t {
	case TypeNewFreightRequest, TypeFreightCompleted, TypeFreightCancelledConfirmed:
		return CategoryFreightRequests
	case TypeNewOffer, TypeOfferSelected, TypeOfferRejected, TypeOfferConfirmed, TypeOfferDeclined, TypeOfferWithdrawn:
		return CategoryOffers
	case TypeOrderCreated, TypeOrderMessage, TypeOrderDocument, TypeOrderCompleted, TypeOrderCancelled:
		return CategoryOrders
	case TypeReviewReceived:
		return CategoryReviews
	case TypeMemberInvited, TypeMemberJoined, TypeOrgStatusChanged:
		return CategoryOrganization
	default:
		return CategoryOrganization
	}
}

// NotificationCategory определяет категорию уведомлений для группировки и настроек
type NotificationCategory string

const (
	CategoryFreightRequests NotificationCategory = "freight_requests"
	CategoryOffers          NotificationCategory = "offers"
	CategoryOrders          NotificationCategory = "orders"
	CategoryReviews         NotificationCategory = "reviews"
	CategoryOrganization    NotificationCategory = "organization"
)

// AllCategories возвращает список всех категорий
func AllCategories() []NotificationCategory {
	return []NotificationCategory{
		CategoryFreightRequests,
		CategoryOffers,
		CategoryOrders,
		CategoryReviews,
		CategoryOrganization,
	}
}

// NotificationChannel определяет канал доставки
type NotificationChannel string

const (
	ChannelInApp    NotificationChannel = "in_app"
	ChannelTelegram NotificationChannel = "telegram"
	ChannelEmail    NotificationChannel = "email"
)

// EntityType определяет тип сущности, к которой относится уведомление
type EntityType string

const (
	EntityFreightRequest EntityType = "freight_request"
	EntityOrder          EntityType = "order"
	EntityOrganization   EntityType = "organization"
	EntityMember         EntityType = "member"
)

// DeliveryStatus определяет статус доставки уведомления
type DeliveryStatus string

const (
	DeliveryStatusSent      DeliveryStatus = "sent"
	DeliveryStatusFailed    DeliveryStatus = "failed"
	DeliveryStatusSkipped   DeliveryStatus = "skipped"
	DeliveryStatusDelivered DeliveryStatus = "delivered" // Email: подтверждена доставка
	DeliveryStatusBounced   DeliveryStatus = "bounced"   // Email: отклонено сервером получателя
	DeliveryStatusOpened    DeliveryStatus = "opened"    // Email: маркетинговые - письмо открыто
	DeliveryStatusClicked   DeliveryStatus = "clicked"   // Email: маркетинговые - клик по ссылке
)

// CategorySettings настройки для одной категории
type CategorySettings struct {
	InApp    bool `json:"in_app"`
	Telegram bool `json:"telegram"`
	Email    bool `json:"email"`
}

// EnabledCategories настройки всех категорий
type EnabledCategories map[NotificationCategory]CategorySettings

// DefaultEnabledCategories возвращает настройки по умолчанию
func DefaultEnabledCategories() EnabledCategories {
	return EnabledCategories{
		CategoryFreightRequests: {InApp: true, Telegram: false, Email: false},
		CategoryOffers:          {InApp: true, Telegram: false, Email: false},
		CategoryOrders:          {InApp: true, Telegram: false, Email: false},
		CategoryReviews:         {InApp: true, Telegram: false, Email: false},
		CategoryOrganization:    {InApp: true, Telegram: false, Email: false},
	}
}

// IsEnabled проверяет включена ли категория для канала
func (c EnabledCategories) IsEnabled(category NotificationCategory, channel NotificationChannel) bool {
	settings, ok := c[category]
	if !ok {
		return false
	}
	switch channel {
	case ChannelInApp:
		return settings.InApp
	case ChannelTelegram:
		return settings.Telegram
	case ChannelEmail:
		return settings.Email
	default:
		return false
	}
}

// EnableTelegramForAll включает telegram для всех категорий
func (c EnabledCategories) EnableTelegramForAll() {
	for _, cat := range AllCategories() {
		settings := c[cat]
		settings.Telegram = true
		c[cat] = settings
	}
}

// EnableEmailForAll включает email для всех категорий
func (c EnabledCategories) EnableEmailForAll() {
	for _, cat := range AllCategories() {
		settings := c[cat]
		settings.Email = true
		c[cat] = settings
	}
}

// EmailType определяет тип email для разделения транзакционных и маркетинговых
type EmailType string

const (
	// EmailTypeTransactional — транзакционные письма (сброс пароля, подтверждение email, верификация)
	// Высокий приоритет, без tracking открытий/кликов
	EmailTypeTransactional EmailType = "transactional"

	// EmailTypeMarketing — маркетинговые письма (рассылки, промо-акции)
	// Обычный приоритет, с tracking открытий/кликов, требует opt-in
	EmailTypeMarketing EmailType = "marketing"
)

// IsTransactional проверяет является ли тип транзакционным
func (t EmailType) IsTransactional() bool {
	return t == EmailTypeTransactional
}

// AllowsTracking проверяет разрешён ли tracking для типа email
func (t EmailType) AllowsTracking() bool {
	return t == EmailTypeMarketing
}
