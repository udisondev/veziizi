package events

import (
	"codeberg.org/udison/veziizi/backend/internal/domain/notification/values"
	"codeberg.org/udison/veziizi/backend/internal/infrastructure/persistence/eventstore"
	"github.com/google/uuid"
)

const AggregateType = "notification"

// Event type constants
const (
	TypeInAppCreated        = "notification.inapp_created"
	TypeInAppRead           = "notification.inapp_read"
	TypeInAppBatchRead      = "notification.inapp_batch_read"
	TypePreferencesUpdated  = "notification.preferences_updated"
	TypeTelegramConnected   = "notification.telegram_connected"
	TypeTelegramDisconnected = "notification.telegram_disconnected"
)

func init() {
	eventstore.RegisterEventType[InAppCreated](TypeInAppCreated)
	eventstore.RegisterEventType[InAppRead](TypeInAppRead)
	eventstore.RegisterEventType[InAppBatchRead](TypeInAppBatchRead)
	eventstore.RegisterEventType[PreferencesUpdated](TypePreferencesUpdated)
	eventstore.RegisterEventType[TelegramConnected](TypeTelegramConnected)
	eventstore.RegisterEventType[TelegramDisconnected](TypeTelegramDisconnected)
}

// InAppCreated is emitted when a new in-app notification is created
type InAppCreated struct {
	eventstore.BaseEvent
	NotificationID   uuid.UUID                `json:"notification_id"`
	MemberID         uuid.UUID                `json:"member_id"`
	OrganizationID   uuid.UUID                `json:"organization_id"`
	NotificationType values.NotificationType  `json:"notification_type"`
	Title            string                   `json:"title"`
	Body             string                   `json:"body,omitempty"`
	Link             string                   `json:"link,omitempty"`
	EntityType       values.EntityType        `json:"entity_type,omitempty"`
	EntityID         uuid.UUID                `json:"entity_id,omitempty"`
}

func (e InAppCreated) EventType() string { return TypeInAppCreated }

// InAppRead is emitted when a single notification is marked as read
type InAppRead struct {
	eventstore.BaseEvent
	NotificationID uuid.UUID `json:"notification_id"`
	MemberID       uuid.UUID `json:"member_id"`
}

func (e InAppRead) EventType() string { return TypeInAppRead }

// InAppBatchRead is emitted when multiple notifications are marked as read
type InAppBatchRead struct {
	eventstore.BaseEvent
	MemberID        uuid.UUID   `json:"member_id"`
	NotificationIDs []uuid.UUID `json:"notification_ids,omitempty"` // if empty, means all
}

func (e InAppBatchRead) EventType() string { return TypeInAppBatchRead }

// PreferencesUpdated is emitted when member updates notification preferences
type PreferencesUpdated struct {
	eventstore.BaseEvent
	MemberID          uuid.UUID                `json:"member_id"`
	EnabledCategories values.EnabledCategories `json:"enabled_categories"`
}

func (e PreferencesUpdated) EventType() string { return TypePreferencesUpdated }

// TelegramConnected is emitted when member connects Telegram
type TelegramConnected struct {
	eventstore.BaseEvent
	MemberID         uuid.UUID `json:"member_id"`
	TelegramChatID   int64     `json:"telegram_chat_id"`
	TelegramUsername string    `json:"telegram_username,omitempty"`
}

func (e TelegramConnected) EventType() string { return TypeTelegramConnected }

// TelegramDisconnected is emitted when member disconnects Telegram
type TelegramDisconnected struct {
	eventstore.BaseEvent
	MemberID uuid.UUID `json:"member_id"`
}

func (e TelegramDisconnected) EventType() string { return TypeTelegramDisconnected }
