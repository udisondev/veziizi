package projections

import (
	"context"
	"fmt"
	"time"

	"codeberg.org/udison/veziizi/backend/internal/domain/notification/values"
	"codeberg.org/udison/veziizi/backend/internal/pkg/dbtx"
	"github.com/Masterminds/squirrel"
	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/google/uuid"
)

type InAppNotificationsProjection struct {
	db   dbtx.TxManager
	psql squirrel.StatementBuilderType
}

func NewInAppNotificationsProjection(db dbtx.TxManager) *InAppNotificationsProjection {
	return &InAppNotificationsProjection{
		db:   db,
		psql: squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar),
	}
}

// InAppNotificationLookup представляет in-app уведомление
type InAppNotificationLookup struct {
	ID               uuid.UUID  `db:"id" json:"id"`
	MemberID         uuid.UUID  `db:"member_id" json:"member_id"`
	OrganizationID   uuid.UUID  `db:"organization_id" json:"organization_id"`
	NotificationType string     `db:"notification_type" json:"notification_type"`
	Title            string     `db:"title" json:"title"`
	Body             *string    `db:"body" json:"body,omitempty"`
	Link             *string    `db:"link" json:"link,omitempty"`
	EntityType       *string    `db:"entity_type" json:"entity_type,omitempty"`
	EntityID         *uuid.UUID `db:"entity_id" json:"entity_id,omitempty"`
	IsRead           bool       `db:"is_read" json:"is_read"`
	ReadAt           *time.Time `db:"read_at" json:"read_at,omitempty"`
	CreatedAt        time.Time  `db:"created_at" json:"created_at"`
}

// InAppNotificationListFilter фильтры для списка уведомлений
type InAppNotificationListFilter struct {
	Category *values.NotificationCategory
	IsRead   *bool
	Limit    int
	Offset   int
}

// List возвращает список уведомлений member с фильтрацией
func (p *InAppNotificationsProjection) List(ctx context.Context, memberID uuid.UUID, filter InAppNotificationListFilter) ([]InAppNotificationLookup, error) {
	builder := p.psql.
		Select("id", "member_id", "organization_id", "notification_type", "title", "body", "link", "entity_type", "entity_id", "is_read", "read_at", "created_at").
		From("inapp_notifications").
		Where(squirrel.Eq{"member_id": memberID}).
		OrderBy("created_at DESC")

	if filter.Category != nil {
		// Фильтруем по типам, относящимся к категории
		types := getNotificationTypesForCategory(*filter.Category)
		builder = builder.Where(squirrel.Eq{"notification_type": types})
	}

	if filter.IsRead != nil {
		builder = builder.Where(squirrel.Eq{"is_read": *filter.IsRead})
	}

	if filter.Limit > 0 {
		builder = builder.Limit(uint64(filter.Limit))
	} else {
		builder = builder.Limit(50) // default limit
	}

	if filter.Offset > 0 {
		builder = builder.Offset(uint64(filter.Offset))
	}

	query, args, err := builder.ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build select query: %w", err)
	}

	var notifications []InAppNotificationLookup
	if err := pgxscan.Select(ctx, p.db, &notifications, query, args...); err != nil {
		return nil, fmt.Errorf("failed to list notifications: %w", err)
	}

	return notifications, nil
}

// GetUnreadCount возвращает количество непрочитанных уведомлений
func (p *InAppNotificationsProjection) GetUnreadCount(ctx context.Context, memberID uuid.UUID) (int, error) {
	query, args, err := p.psql.
		Select("COUNT(*)").
		From("inapp_notifications").
		Where(squirrel.Eq{"member_id": memberID, "is_read": false}).
		ToSql()
	if err != nil {
		return 0, fmt.Errorf("failed to build count query: %w", err)
	}

	var count int
	if err := pgxscan.Get(ctx, p.db, &count, query, args...); err != nil {
		return 0, fmt.Errorf("failed to get unread count: %w", err)
	}

	return count, nil
}

// CreateNotificationInput данные для создания уведомления
type CreateNotificationInput struct {
	ID               uuid.UUID
	MemberID         uuid.UUID
	OrganizationID   uuid.UUID
	NotificationType values.NotificationType
	Title            string
	Body             string
	Link             string
	EntityType       values.EntityType
	EntityID         uuid.UUID
}

// Insert создает новое уведомление
func (p *InAppNotificationsProjection) Insert(ctx context.Context, input CreateNotificationInput) error {
	builder := p.psql.
		Insert("inapp_notifications").
		Columns("id", "member_id", "organization_id", "notification_type", "title", "body", "link", "entity_type", "entity_id", "created_at")

	values := []any{
		input.ID,
		input.MemberID,
		input.OrganizationID,
		string(input.NotificationType),
		input.Title,
		nilIfEmpty(input.Body),
		nilIfEmpty(input.Link),
		nilIfEmpty(string(input.EntityType)),
		nilIfEmptyUUID(input.EntityID),
		time.Now(),
	}

	builder = builder.Values(values...)

	query, args, err := builder.ToSql()
	if err != nil {
		return fmt.Errorf("failed to build insert query: %w", err)
	}

	if _, err := p.db.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("failed to insert notification: %w", err)
	}

	return nil
}

// MarkAsRead помечает уведомления как прочитанные
func (p *InAppNotificationsProjection) MarkAsRead(ctx context.Context, memberID uuid.UUID, ids []uuid.UUID) error {
	if len(ids) == 0 {
		return nil
	}

	query, args, err := p.psql.
		Update("inapp_notifications").
		Set("is_read", true).
		Set("read_at", time.Now()).
		Where(squirrel.Eq{"member_id": memberID, "id": ids, "is_read": false}).
		ToSql()
	if err != nil {
		return fmt.Errorf("failed to build update query: %w", err)
	}

	if _, err := p.db.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("failed to mark as read: %w", err)
	}

	return nil
}

// MarkAllAsRead помечает все уведомления member как прочитанные
func (p *InAppNotificationsProjection) MarkAllAsRead(ctx context.Context, memberID uuid.UUID) error {
	query, args, err := p.psql.
		Update("inapp_notifications").
		Set("is_read", true).
		Set("read_at", time.Now()).
		Where(squirrel.Eq{"member_id": memberID, "is_read": false}).
		ToSql()
	if err != nil {
		return fmt.Errorf("failed to build update query: %w", err)
	}

	if _, err := p.db.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("failed to mark all as read: %w", err)
	}

	return nil
}

// GetByID возвращает уведомление по ID
func (p *InAppNotificationsProjection) GetByID(ctx context.Context, id uuid.UUID) (*InAppNotificationLookup, error) {
	query, args, err := p.psql.
		Select("id", "member_id", "organization_id", "notification_type", "title", "body", "link", "entity_type", "entity_id", "is_read", "read_at", "created_at").
		From("inapp_notifications").
		Where(squirrel.Eq{"id": id}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build select query: %w", err)
	}

	var notification InAppNotificationLookup
	if err := pgxscan.Get(ctx, p.db, &notification, query, args...); err != nil {
		return nil, fmt.Errorf("failed to get notification: %w", err)
	}

	return &notification, nil
}

// DeleteOld удаляет старые уведомления (для cleanup job)
func (p *InAppNotificationsProjection) DeleteOld(ctx context.Context, olderThan time.Time) (int64, error) {
	query, args, err := p.psql.
		Delete("inapp_notifications").
		Where(squirrel.Lt{"created_at": olderThan}).
		ToSql()
	if err != nil {
		return 0, fmt.Errorf("failed to build delete query: %w", err)
	}

	result, err := p.db.Exec(ctx, query, args...)
	if err != nil {
		return 0, fmt.Errorf("failed to delete old notifications: %w", err)
	}

	return result.RowsAffected(), nil
}

// getNotificationTypesForCategory возвращает типы уведомлений для категории
func getNotificationTypesForCategory(category values.NotificationCategory) []string {
	switch category {
	case values.CategoryOffers:
		return []string{
			string(values.TypeNewOffer),
			string(values.TypeOfferSelected),
			string(values.TypeOfferRejected),
			string(values.TypeOfferConfirmed),
			string(values.TypeOfferDeclined),
			string(values.TypeOfferWithdrawn),
		}
	case values.CategoryOrders:
		return []string{
			string(values.TypeOrderCreated),
			string(values.TypeOrderMessage),
			string(values.TypeOrderDocument),
			string(values.TypeOrderCompleted),
			string(values.TypeOrderCancelled),
		}
	case values.CategoryReviews:
		return []string{
			string(values.TypeReviewReceived),
		}
	case values.CategoryOrganization:
		return []string{
			string(values.TypeMemberInvited),
			string(values.TypeMemberJoined),
			string(values.TypeOrgStatusChanged),
		}
	default:
		return nil
	}
}

// nilIfEmpty возвращает nil если строка пустая
func nilIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// nilIfEmptyUUID возвращает nil если UUID пустой
func nilIfEmptyUUID(id uuid.UUID) *uuid.UUID {
	if id == uuid.Nil {
		return nil
	}
	return &id
}
