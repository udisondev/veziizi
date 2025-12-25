package projections

import (
	"context"
	"errors"
	"fmt"

	"codeberg.org/udison/veziizi/backend/internal/pkg/dbtx"
	"github.com/Masterminds/squirrel"
	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/lib/pq"
)

// FreightRequestSubscriptionsProjection работает с подписками на заявки
type FreightRequestSubscriptionsProjection struct {
	db   dbtx.TxManager
	psql squirrel.StatementBuilderType
}

// NewFreightRequestSubscriptionsProjection создает новый projection
func NewFreightRequestSubscriptionsProjection(db dbtx.TxManager) *FreightRequestSubscriptionsProjection {
	return &FreightRequestSubscriptionsProjection{
		db:   db,
		psql: squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar),
	}
}

// SubscriptionLookup представляет подписку пользователя
type SubscriptionLookup struct {
	MemberID              uuid.UUID `db:"member_id"`
	OriginCountryIDs      []int64   `db:"origin_country_ids"`
	DestinationCountryIDs []int64   `db:"destination_country_ids"`
	CargoTypes            []string  `db:"cargo_types"`
	MinWeight             *float64  `db:"min_weight"`
	MaxWeight             *float64  `db:"max_weight"`
	BodyTypes             []string  `db:"body_types"`
	Unsubscribed          bool      `db:"unsubscribed"`
}

// SubscriptionFilter фильтры для поиска подписчиков
type SubscriptionFilter struct {
	OriginCountryID      *int
	DestinationCountryID *int
	CargoType            string
	CargoWeight          float64
	BodyTypes            []string
}

// SubscriberInfo информация о подписчике
type SubscriberInfo struct {
	MemberID       uuid.UUID
	OrganizationID uuid.UUID
}

// GetByMemberID возвращает подписку пользователя
func (p *FreightRequestSubscriptionsProjection) GetByMemberID(ctx context.Context, memberID uuid.UUID) (*SubscriptionLookup, error) {
	query, args, err := p.psql.
		Select(
			"member_id",
			"origin_country_ids",
			"destination_country_ids",
			"cargo_types",
			"min_weight",
			"max_weight",
			"body_types",
			"unsubscribed",
		).
		From("freight_request_subscriptions").
		Where(squirrel.Eq{"member_id": memberID}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build select query: %w", err)
	}

	var sub SubscriptionLookup
	if err := pgxscan.Get(ctx, p.db, &sub, query, args...); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get subscription: %w", err)
	}

	return &sub, nil
}

// Upsert создает или обновляет подписку
func (p *FreightRequestSubscriptionsProjection) Upsert(ctx context.Context, sub *SubscriptionLookup) error {
	query, args, err := p.psql.
		Insert("freight_request_subscriptions").
		Columns(
			"member_id",
			"origin_country_ids",
			"destination_country_ids",
			"cargo_types",
			"min_weight",
			"max_weight",
			"body_types",
			"unsubscribed",
			"updated_at",
		).
		Values(
			sub.MemberID,
			pq.Array(sub.OriginCountryIDs),
			pq.Array(sub.DestinationCountryIDs),
			pq.Array(sub.CargoTypes),
			sub.MinWeight,
			sub.MaxWeight,
			pq.Array(sub.BodyTypes),
			sub.Unsubscribed,
			squirrel.Expr("NOW()"),
		).
		Suffix(`ON CONFLICT (member_id) DO UPDATE SET
			origin_country_ids = EXCLUDED.origin_country_ids,
			destination_country_ids = EXCLUDED.destination_country_ids,
			cargo_types = EXCLUDED.cargo_types,
			min_weight = EXCLUDED.min_weight,
			max_weight = EXCLUDED.max_weight,
			body_types = EXCLUDED.body_types,
			unsubscribed = EXCLUDED.unsubscribed,
			updated_at = NOW()`).
		ToSql()
	if err != nil {
		return fmt.Errorf("build upsert query: %w", err)
	}

	if _, err := p.db.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("upsert subscription: %w", err)
	}

	return nil
}

// GetSubscribedMembers возвращает подписанных пользователей для заданных фильтров
// Логика opt-out: если у пользователя нет записи - он подписан на всё
func (p *FreightRequestSubscriptionsProjection) GetSubscribedMembers(
	ctx context.Context,
	filter SubscriptionFilter,
	excludeMemberID uuid.UUID,
) ([]SubscriberInfo, error) {
	// Запрос сложный: нужно найти тех, кто подписан
	// Пользователь подписан если:
	// 1. Нет записи в таблице (подписан на всё по умолчанию)
	// 2. Или есть запись с unsubscribed=false И фильтры совпадают

	// Используем raw SQL для сложного запроса
	query := `
		SELECT m.id as member_id, m.organization_id
		FROM members_lookup m
		LEFT JOIN freight_request_subscriptions s ON s.member_id = m.id
		WHERE m.id != $1
		  AND m.status = 'active'
		  AND (
		      -- Нет записи = подписан на всё
		      s.member_id IS NULL
		      OR (
		          -- Есть запись и не отписан
		          s.unsubscribed = false
		          -- Проверяем фильтры (NULL = все)
		          AND (s.origin_country_ids IS NULL OR $2::int IS NULL OR $2 = ANY(s.origin_country_ids))
		          AND (s.destination_country_ids IS NULL OR $3::int IS NULL OR $3 = ANY(s.destination_country_ids))
		          AND (s.cargo_types IS NULL OR array_length(s.cargo_types, 1) IS NULL OR $4 = ANY(s.cargo_types))
		          AND (s.min_weight IS NULL OR $5 >= s.min_weight)
		          AND (s.max_weight IS NULL OR $5 <= s.max_weight)
		          AND (s.body_types IS NULL OR array_length(s.body_types, 1) IS NULL OR s.body_types && $6::text[])
		      )
		  )`

	rows, err := p.db.Query(ctx, query,
		excludeMemberID,
		filter.OriginCountryID,
		filter.DestinationCountryID,
		filter.CargoType,
		filter.CargoWeight,
		pq.Array(filter.BodyTypes),
	)
	if err != nil {
		return nil, fmt.Errorf("query subscribed members: %w", err)
	}
	defer rows.Close()

	var subscribers []SubscriberInfo
	for rows.Next() {
		var sub SubscriberInfo
		if err := rows.Scan(&sub.MemberID, &sub.OrganizationID); err != nil {
			return nil, fmt.Errorf("scan subscriber: %w", err)
		}
		subscribers = append(subscribers, sub)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate subscribers: %w", err)
	}

	return subscribers, nil
}
