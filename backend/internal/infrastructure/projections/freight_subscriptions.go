package projections

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/lib/pq"
	"github.com/udisondev/veziizi/backend/internal/domain/freightrequest/values"
	"github.com/udisondev/veziizi/backend/internal/pkg/dbtx"
)

var _ = pgx.ErrNoRows // используется для проверки ErrNoRows

// FreightSubscriptionsProjection работает с подписками на заявки (opt-in модель)
type FreightSubscriptionsProjection struct {
	db   dbtx.TxManager
	psql squirrel.StatementBuilderType
}

// NewFreightSubscriptionsProjection создает новый projection
func NewFreightSubscriptionsProjection(db dbtx.TxManager) *FreightSubscriptionsProjection {
	return &FreightSubscriptionsProjection{
		db:   db,
		psql: squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar),
	}
}

// SubscriptionRow представляет запись в БД
type SubscriptionRow struct {
	ID              uuid.UUID `db:"id"`
	MemberID        uuid.UUID `db:"member_id"`
	Name            string    `db:"name"`
	MinWeight       *float64  `db:"min_weight"`
	MaxWeight       *float64  `db:"max_weight"`
	MinPrice        *int64    `db:"min_price"`
	MaxPrice        *int64    `db:"max_price"`
	MinVolume       *float64  `db:"min_volume"`
	MaxVolume       *float64  `db:"max_volume"`
	VehicleTypes    []string  `db:"vehicle_types"`
	VehicleSubTypes []string  `db:"vehicle_subtypes"`
	PaymentMethods  []string  `db:"payment_methods"`
	PaymentTerms    []string  `db:"payment_terms"`
	VatTypes        []string  `db:"vat_types"`
	IsActive        bool      `db:"is_active"`
	CreatedAt       time.Time `db:"created_at"`
	UpdatedAt       time.Time `db:"updated_at"`
}

// RoutePointRow представляет точку маршрута в БД
type RoutePointRow struct {
	ID             uuid.UUID `db:"id"`
	SubscriptionID uuid.UUID `db:"subscription_id"`
	CountryID      int       `db:"country_id"`
	CityID         *int      `db:"city_id"`
	PointOrder     int       `db:"point_order"`
}

// Create создает новую подписку
func (p *FreightSubscriptionsProjection) Create(ctx context.Context, memberID uuid.UUID, criteria values.SubscriptionCriteria) (*values.Subscription, error) {
	// Проверяем лимит подписок
	count, err := p.CountByMemberID(ctx, memberID)
	if err != nil {
		return nil, fmt.Errorf("count subscriptions: %w", err)
	}
	if count >= values.MaxSubscriptionsPerMember {
		return nil, fmt.Errorf("subscription limit reached (max %d)", values.MaxSubscriptionsPerMember)
	}

	// Валидируем критерии
	if err := criteria.Validate(); err != nil {
		return nil, fmt.Errorf("validate criteria: %w", err)
	}

	subscriptionID := uuid.New()
	now := time.Now()

	// Выполняем в транзакции
	err = p.db.InTx(ctx, func(txCtx context.Context) error {
		// Вставляем подписку
		query, args, err := p.psql.
			Insert("freight_subscriptions").
			Columns(
				"id", "member_id", "name",
				"min_weight", "max_weight",
				"min_price", "max_price",
				"min_volume", "max_volume",
				"vehicle_types", "vehicle_subtypes",
				"payment_methods", "payment_terms", "vat_types",
				"is_active", "created_at", "updated_at",
			).
			Values(
				subscriptionID, memberID, criteria.Name,
				criteria.MinWeight, criteria.MaxWeight,
				criteria.MinPrice, criteria.MaxPrice,
				criteria.MinVolume, criteria.MaxVolume,
				pq.Array(vehicleTypesToStrings(criteria.VehicleTypes)),
				pq.Array(vehicleSubTypesToStrings(criteria.VehicleSubTypes)),
				pq.Array(paymentMethodsToStrings(criteria.PaymentMethods)),
				pq.Array(paymentTermsToStrings(criteria.PaymentTerms)),
				pq.Array(vatTypesToStrings(criteria.VatTypes)),
				criteria.IsActive, now, now,
			).
			ToSql()
		if err != nil {
			return fmt.Errorf("build insert query: %w", err)
		}

		if _, err := p.db.Exec(txCtx, query, args...); err != nil {
			return fmt.Errorf("insert subscription: %w", err)
		}

		// Вставляем route points
		if len(criteria.RoutePoints) > 0 {
			if err := p.insertRoutePointsInTx(txCtx, subscriptionID, criteria.RoutePoints); err != nil {
				return fmt.Errorf("insert route points: %w", err)
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return &values.Subscription{
		ID:        subscriptionID,
		MemberID:  memberID,
		Criteria:  criteria,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

// Update обновляет подписку
func (p *FreightSubscriptionsProjection) Update(ctx context.Context, subscriptionID uuid.UUID, memberID uuid.UUID, criteria values.SubscriptionCriteria) error {
	// Валидируем критерии
	if err := criteria.Validate(); err != nil {
		return fmt.Errorf("validate criteria: %w", err)
	}

	// Выполняем в транзакции
	return p.db.InTx(ctx, func(txCtx context.Context) error {
		// Обновляем подписку
		query, args, err := p.psql.
			Update("freight_subscriptions").
			Set("name", criteria.Name).
			Set("min_weight", criteria.MinWeight).
			Set("max_weight", criteria.MaxWeight).
			Set("min_price", criteria.MinPrice).
			Set("max_price", criteria.MaxPrice).
			Set("min_volume", criteria.MinVolume).
			Set("max_volume", criteria.MaxVolume).
			Set("vehicle_types", pq.Array(vehicleTypesToStrings(criteria.VehicleTypes))).
			Set("vehicle_subtypes", pq.Array(vehicleSubTypesToStrings(criteria.VehicleSubTypes))).
			Set("payment_methods", pq.Array(paymentMethodsToStrings(criteria.PaymentMethods))).
			Set("payment_terms", pq.Array(paymentTermsToStrings(criteria.PaymentTerms))).
			Set("vat_types", pq.Array(vatTypesToStrings(criteria.VatTypes))).
			Set("is_active", criteria.IsActive).
			Set("updated_at", squirrel.Expr("NOW()")).
			Where(squirrel.Eq{"id": subscriptionID, "member_id": memberID}).
			ToSql()
		if err != nil {
			return fmt.Errorf("build update query: %w", err)
		}

		result, err := p.db.Exec(txCtx, query, args...)
		if err != nil {
			return fmt.Errorf("update subscription: %w", err)
		}
		if result.RowsAffected() == 0 {
			return fmt.Errorf("subscription not found")
		}

		// Удаляем старые route points
		deleteQuery, deleteArgs, err := p.psql.
			Delete("freight_subscription_route_points").
			Where(squirrel.Eq{"subscription_id": subscriptionID}).
			ToSql()
		if err != nil {
			return fmt.Errorf("build delete route points query: %w", err)
		}

		if _, err := p.db.Exec(txCtx, deleteQuery, deleteArgs...); err != nil {
			return fmt.Errorf("delete route points: %w", err)
		}

		// Вставляем новые route points
		if len(criteria.RoutePoints) > 0 {
			if err := p.insertRoutePointsInTx(txCtx, subscriptionID, criteria.RoutePoints); err != nil {
				return fmt.Errorf("insert route points: %w", err)
			}
		}

		return nil
	})
}

// Delete удаляет подписку
func (p *FreightSubscriptionsProjection) Delete(ctx context.Context, subscriptionID uuid.UUID, memberID uuid.UUID) error {
	query, args, err := p.psql.
		Delete("freight_subscriptions").
		Where(squirrel.Eq{"id": subscriptionID, "member_id": memberID}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build delete query: %w", err)
	}

	result, err := p.db.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("delete subscription: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("subscription not found")
	}

	return nil
}

// GetByID возвращает подписку по ID
func (p *FreightSubscriptionsProjection) GetByID(ctx context.Context, subscriptionID uuid.UUID) (*values.Subscription, error) {
	query, args, err := p.psql.
		Select(
			"id", "member_id", "name",
			"min_weight", "max_weight",
			"min_price", "max_price",
			"min_volume", "max_volume",
			"vehicle_types", "vehicle_subtypes",
			"payment_methods", "payment_terms", "vat_types",
			"is_active", "created_at", "updated_at",
		).
		From("freight_subscriptions").
		Where(squirrel.Eq{"id": subscriptionID}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build select query: %w", err)
	}

	var row SubscriptionRow
	if err := pgxscan.Get(ctx, p.db, &row, query, args...); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get subscription: %w", err)
	}

	// Получаем route points
	routePoints, err := p.getRoutePoints(ctx, subscriptionID)
	if err != nil {
		return nil, fmt.Errorf("get route points: %w", err)
	}

	return p.rowToSubscription(&row, routePoints), nil
}

// GetByMemberID возвращает все подписки пользователя
func (p *FreightSubscriptionsProjection) GetByMemberID(ctx context.Context, memberID uuid.UUID) ([]values.Subscription, error) {
	query, args, err := p.psql.
		Select(
			"id", "member_id", "name",
			"min_weight", "max_weight",
			"min_price", "max_price",
			"min_volume", "max_volume",
			"vehicle_types", "vehicle_subtypes",
			"payment_methods", "payment_terms", "vat_types",
			"is_active", "created_at", "updated_at",
		).
		From("freight_subscriptions").
		Where(squirrel.Eq{"member_id": memberID}).
		OrderBy("created_at DESC").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build select query: %w", err)
	}

	var rows []SubscriptionRow
	if err := pgxscan.Select(ctx, p.db, &rows, query, args...); err != nil {
		return nil, fmt.Errorf("select subscriptions: %w", err)
	}

	// Собираем все subscription IDs
	subIDs := make([]uuid.UUID, len(rows))
	for i, row := range rows {
		subIDs[i] = row.ID
	}

	// Получаем все route points за один запрос
	routePointsMap, err := p.getRoutePointsBatch(ctx, subIDs)
	if err != nil {
		return nil, fmt.Errorf("get route points batch: %w", err)
	}

	subscriptions := make([]values.Subscription, len(rows))
	for i, row := range rows {
		subscriptions[i] = *p.rowToSubscription(&row, routePointsMap[row.ID])
	}

	return subscriptions, nil
}

// CountByMemberID возвращает количество подписок пользователя
func (p *FreightSubscriptionsProjection) CountByMemberID(ctx context.Context, memberID uuid.UUID) (int, error) {
	query, args, err := p.psql.
		Select("COUNT(*)").
		From("freight_subscriptions").
		Where(squirrel.Eq{"member_id": memberID}).
		ToSql()
	if err != nil {
		return 0, fmt.Errorf("build count query: %w", err)
	}

	var count int
	if err := p.db.QueryRow(ctx, query, args...).Scan(&count); err != nil {
		return 0, fmt.Errorf("count subscriptions: %w", err)
	}

	return count, nil
}

// SetActive включает/выключает подписку
func (p *FreightSubscriptionsProjection) SetActive(ctx context.Context, subscriptionID uuid.UUID, memberID uuid.UUID, isActive bool) error {
	query, args, err := p.psql.
		Update("freight_subscriptions").
		Set("is_active", isActive).
		Set("updated_at", squirrel.Expr("NOW()")).
		Where(squirrel.Eq{"id": subscriptionID, "member_id": memberID}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build update query: %w", err)
	}

	result, err := p.db.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update is_active: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("subscription not found")
	}

	return nil
}

// FindMatchingSubscriptions находит подписки, которые соответствуют заявке
func (p *FreightSubscriptionsProjection) FindMatchingSubscriptions(
	ctx context.Context,
	data values.FreightRequestMatchData,
	excludeMemberID uuid.UUID,
) ([]values.MatchedSubscription, error) {
	// Этап 1: SQL фильтрация по числовым диапазонам и ENUM массивам
	// Это быстрый этап с использованием индексов

	var priceAmount *int64
	if data.Payment.Price != nil {
		priceAmount = &data.Payment.Price.Amount
	}

	query := `
		SELECT
			s.id as subscription_id,
			s.name as subscription_name,
			s.member_id,
			m.organization_id,
			s.vehicle_types,
			s.vehicle_subtypes,
			s.payment_methods,
			s.payment_terms,
			s.vat_types,
			s.min_weight,
			s.max_weight,
			s.min_price,
			s.max_price,
			s.min_volume,
			s.max_volume
		FROM freight_subscriptions s
		JOIN members_lookup m ON m.id = s.member_id
		WHERE s.is_active = true
		  AND m.status = 'active'
		  AND s.member_id != $1

		  -- Числовые диапазоны (NULL = без ограничения)
		  AND (s.min_weight IS NULL OR $2 >= s.min_weight)
		  AND (s.max_weight IS NULL OR $2 <= s.max_weight)
		  AND (s.min_price IS NULL OR $3::bigint IS NULL OR $3 >= s.min_price)
		  AND (s.max_price IS NULL OR $3::bigint IS NULL OR $3 <= s.max_price)
		  AND (s.min_volume IS NULL OR $4 >= s.min_volume)
		  AND (s.max_volume IS NULL OR $4 <= s.max_volume)

		  -- ENUM фильтры (NULL/пустой = все подходят)
		  AND (s.vehicle_types IS NULL OR array_length(s.vehicle_types, 1) IS NULL
		       OR $5 = ANY(s.vehicle_types))
		  AND (s.vehicle_subtypes IS NULL OR array_length(s.vehicle_subtypes, 1) IS NULL
		       OR $6 = ANY(s.vehicle_subtypes))
		  AND (s.payment_methods IS NULL OR array_length(s.payment_methods, 1) IS NULL
		       OR $7 = ANY(s.payment_methods))
		  AND (s.payment_terms IS NULL OR array_length(s.payment_terms, 1) IS NULL
		       OR $8 = ANY(s.payment_terms))
		  AND (s.vat_types IS NULL OR array_length(s.vat_types, 1) IS NULL
		       OR $9 = ANY(s.vat_types))
	`

	rows, err := p.db.Query(ctx, query,
		excludeMemberID,
		data.Cargo.Weight,
		priceAmount,
		data.Cargo.Volume,
		string(data.VehicleReqs.VehicleType),
		string(data.VehicleReqs.VehicleSubType),
		string(data.Payment.Method),
		string(data.Payment.Terms),
		string(data.Payment.VatType),
	)
	if err != nil {
		return nil, fmt.Errorf("query matching subscriptions: %w", err)
	}
	defer rows.Close()

	// Этап 2: Собираем кандидатов для проверки route matching
	type candidate struct {
		SubscriptionID   uuid.UUID
		SubscriptionName string
		MemberID         uuid.UUID
		OrganizationID   uuid.UUID
	}
	var candidates []candidate

	for rows.Next() {
		var c candidate
		var vehicleTypes, vehicleSubTypes, paymentMethods, paymentTerms, vatTypes []string
		var minWeight, maxWeight, minVolume, maxVolume *float64
		var minPrice, maxPrice *int64

		if err := rows.Scan(
			&c.SubscriptionID,
			&c.SubscriptionName,
			&c.MemberID,
			&c.OrganizationID,
			&vehicleTypes,
			&vehicleSubTypes,
			&paymentMethods,
			&paymentTerms,
			&vatTypes,
			&minWeight,
			&maxWeight,
			&minPrice,
			&maxPrice,
			&minVolume,
			&maxVolume,
		); err != nil {
			return nil, fmt.Errorf("scan candidate: %w", err)
		}

		candidates = append(candidates, c)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate candidates: %w", err)
	}

	if len(candidates) == 0 {
		return nil, nil
	}

	// Этап 3: Получаем route points для всех кандидатов
	candidateIDs := make([]uuid.UUID, len(candidates))
	for i, c := range candidates {
		candidateIDs[i] = c.SubscriptionID
	}

	routePointsMap, err := p.getRoutePointsBatch(ctx, candidateIDs)
	if err != nil {
		return nil, fmt.Errorf("get route points batch: %w", err)
	}

	// Этап 4: Subsequence matching в Go
	var matched []values.MatchedSubscription
	for _, c := range candidates {
		routePoints := routePointsMap[c.SubscriptionID]

		// Если нет route points - подписка подходит (любой маршрут)
		// Иначе проверяем subsequence
		if len(routePoints) == 0 || values.IsSubsequence(routePoints, data.Route) {
			matched = append(matched, values.MatchedSubscription{
				SubscriptionID:   c.SubscriptionID,
				SubscriptionName: c.SubscriptionName,
				MemberID:         c.MemberID,
				OrganizationID:   c.OrganizationID,
			})
		}
	}

	return matched, nil
}

// insertRoutePointsInTx вставляет точки маршрута в транзакции
func (p *FreightSubscriptionsProjection) insertRoutePointsInTx(ctx context.Context, subscriptionID uuid.UUID, points []values.RoutePointCriteria) error {
	builder := p.psql.
		Insert("freight_subscription_route_points").
		Columns("id", "subscription_id", "country_id", "city_id", "point_order")

	for _, point := range points {
		builder = builder.Values(uuid.New(), subscriptionID, point.CountryID, point.CityID, point.Order)
	}

	query, args, err := builder.ToSql()
	if err != nil {
		return fmt.Errorf("build insert query: %w", err)
	}

	if _, err := p.db.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("insert route points: %w", err)
	}

	return nil
}

// getRoutePoints получает точки маршрута для подписки
func (p *FreightSubscriptionsProjection) getRoutePoints(ctx context.Context, subscriptionID uuid.UUID) ([]values.RoutePointCriteria, error) {
	query, args, err := p.psql.
		Select("country_id", "city_id", "point_order").
		From("freight_subscription_route_points").
		Where(squirrel.Eq{"subscription_id": subscriptionID}).
		OrderBy("point_order ASC").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build select query: %w", err)
	}

	rows, err := p.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query route points: %w", err)
	}
	defer rows.Close()

	var points []values.RoutePointCriteria
	for rows.Next() {
		var point values.RoutePointCriteria
		if err := rows.Scan(&point.CountryID, &point.CityID, &point.Order); err != nil {
			return nil, fmt.Errorf("scan route point: %w", err)
		}
		points = append(points, point)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate route points: %w", err)
	}

	return points, nil
}

// getRoutePointsBatch получает точки маршрута для нескольких подписок
func (p *FreightSubscriptionsProjection) getRoutePointsBatch(ctx context.Context, subscriptionIDs []uuid.UUID) (map[uuid.UUID][]values.RoutePointCriteria, error) {
	if len(subscriptionIDs) == 0 {
		return make(map[uuid.UUID][]values.RoutePointCriteria), nil
	}

	query, args, err := p.psql.
		Select("subscription_id", "country_id", "city_id", "point_order").
		From("freight_subscription_route_points").
		Where(squirrel.Eq{"subscription_id": subscriptionIDs}).
		OrderBy("subscription_id", "point_order ASC").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build select query: %w", err)
	}

	rows, err := p.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query route points: %w", err)
	}
	defer rows.Close()

	result := make(map[uuid.UUID][]values.RoutePointCriteria)
	for rows.Next() {
		var subscriptionID uuid.UUID
		var point values.RoutePointCriteria
		if err := rows.Scan(&subscriptionID, &point.CountryID, &point.CityID, &point.Order); err != nil {
			return nil, fmt.Errorf("scan route point: %w", err)
		}
		result[subscriptionID] = append(result[subscriptionID], point)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate route points: %w", err)
	}

	// Сортируем точки по order
	for subID := range result {
		sort.Slice(result[subID], func(i, j int) bool {
			return result[subID][i].Order < result[subID][j].Order
		})
	}

	return result, nil
}

// rowToSubscription конвертирует строку БД в Subscription
func (p *FreightSubscriptionsProjection) rowToSubscription(row *SubscriptionRow, routePoints []values.RoutePointCriteria) *values.Subscription {
	return &values.Subscription{
		ID:       row.ID,
		MemberID: row.MemberID,
		Criteria: values.SubscriptionCriteria{
			Name:            row.Name,
			MinWeight:       row.MinWeight,
			MaxWeight:       row.MaxWeight,
			MinPrice:        row.MinPrice,
			MaxPrice:        row.MaxPrice,
			MinVolume:       row.MinVolume,
			MaxVolume:       row.MaxVolume,
			VehicleTypes:    stringsToVehicleTypes(row.VehicleTypes),
			VehicleSubTypes: stringsToVehicleSubTypes(row.VehicleSubTypes),
			PaymentMethods:  stringsToPaymentMethods(row.PaymentMethods),
			PaymentTerms:    stringsToPaymentTerms(row.PaymentTerms),
			VatTypes:        stringsToVatTypes(row.VatTypes),
			RoutePoints:     routePoints,
			IsActive:        row.IsActive,
		},
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
	}
}

// Вспомогательные функции конвертации

func vehicleTypesToStrings(types []values.VehicleType) []string {
	result := make([]string, len(types))
	for i, t := range types {
		result[i] = string(t)
	}
	return result
}

func stringsToVehicleTypes(strings []string) []values.VehicleType {
	result := make([]values.VehicleType, len(strings))
	for i, s := range strings {
		result[i] = values.VehicleType(s)
	}
	return result
}

func vehicleSubTypesToStrings(types []values.VehicleSubType) []string {
	result := make([]string, len(types))
	for i, t := range types {
		result[i] = string(t)
	}
	return result
}

func stringsToVehicleSubTypes(strings []string) []values.VehicleSubType {
	result := make([]values.VehicleSubType, len(strings))
	for i, s := range strings {
		result[i] = values.VehicleSubType(s)
	}
	return result
}

func paymentMethodsToStrings(types []values.PaymentMethod) []string {
	result := make([]string, len(types))
	for i, t := range types {
		result[i] = string(t)
	}
	return result
}

func stringsToPaymentMethods(strings []string) []values.PaymentMethod {
	result := make([]values.PaymentMethod, len(strings))
	for i, s := range strings {
		result[i] = values.PaymentMethod(s)
	}
	return result
}

func paymentTermsToStrings(types []values.PaymentTerms) []string {
	result := make([]string, len(types))
	for i, t := range types {
		result[i] = string(t)
	}
	return result
}

func stringsToPaymentTerms(strings []string) []values.PaymentTerms {
	result := make([]values.PaymentTerms, len(strings))
	for i, s := range strings {
		result[i] = values.PaymentTerms(s)
	}
	return result
}

func vatTypesToStrings(types []values.VatType) []string {
	result := make([]string, len(types))
	for i, t := range types {
		result[i] = string(t)
	}
	return result
}

func stringsToVatTypes(strings []string) []values.VatType {
	result := make([]values.VatType, len(strings))
	for i, s := range strings {
		result[i] = values.VatType(s)
	}
	return result
}
