package projections

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"codeberg.org/udison/veziizi/backend/internal/pkg/dbtx"
	"github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// OrderFraudProjection handles order fraud detection data
type OrderFraudProjection struct {
	db   dbtx.TxManager
	psql squirrel.StatementBuilderType
}

func NewOrderFraudProjection(db dbtx.TxManager) *OrderFraudProjection {
	return &OrderFraudProjection{
		db:   db,
		psql: squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar),
	}
}

// OrderFraudSignal represents a fraud signal for an order
type OrderFraudSignal struct {
	ID          uuid.UUID
	OrderID     uuid.UUID
	OrgID       uuid.UUID
	SignalType  string
	Severity    string
	Description string
	ScoreImpact float64
	Evidence    string
	CreatedAt   time.Time
}

// InsertOrderFraudSignal inserts a fraud signal for an order
func (p *OrderFraudProjection) InsertOrderFraudSignal(ctx context.Context, signal *OrderFraudSignal) error {
	query := `
		INSERT INTO order_fraud_signals (order_id, org_id, signal_type, severity, description, score_impact, evidence)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	var evidenceJSON any = nil
	if signal.Evidence != "" {
		evidenceJSON = signal.Evidence
	}

	if _, err := p.db.Exec(ctx, query,
		signal.OrderID, signal.OrgID, signal.SignalType, signal.Severity,
		signal.Description, signal.ScoreImpact, evidenceJSON,
	); err != nil {
		return fmt.Errorf("insert order fraud signal: %w", err)
	}

	return nil
}

// OrgOrderBehavior represents order behavior statistics for an organization
type OrgOrderBehavior struct {
	OrgID                  uuid.UUID
	TotalOrdersAsCustomer  int
	CompletedAsCustomer    int
	CancelledAsCustomer    int
	TotalOrdersAsCarrier   int
	CompletedAsCarrier     int
	CancelledAsCarrier     int
	AvgCompletionHours     *float64
	MinCompletionHours     *float64
	IsSuspicious           bool
	SuspiciousReason       *string
}

// GetOrgOrderBehavior returns order behavior for an organization
func (p *OrderFraudProjection) GetOrgOrderBehavior(ctx context.Context, orgID uuid.UUID) (*OrgOrderBehavior, error) {
	query, args, err := p.psql.
		Select(
			"org_id", "total_orders_as_customer", "completed_as_customer", "cancelled_as_customer",
			"total_orders_as_carrier", "completed_as_carrier", "cancelled_as_carrier",
			"avg_completion_hours", "min_completion_hours", "is_suspicious", "suspicious_reason",
		).
		From("org_order_behavior").
		Where(squirrel.Eq{"org_id": orgID}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build query: %w", err)
	}

	var b OrgOrderBehavior
	if err := p.db.QueryRow(ctx, query, args...).Scan(
		&b.OrgID, &b.TotalOrdersAsCustomer, &b.CompletedAsCustomer, &b.CancelledAsCustomer,
		&b.TotalOrdersAsCarrier, &b.CompletedAsCarrier, &b.CancelledAsCarrier,
		&b.AvgCompletionHours, &b.MinCompletionHours, &b.IsSuspicious, &b.SuspiciousReason,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return &OrgOrderBehavior{OrgID: orgID}, nil
		}
		return nil, fmt.Errorf("query org order behavior: %w", err)
	}

	return &b, nil
}

// UpsertOrgOrderBehavior creates or updates organization order behavior
func (p *OrderFraudProjection) UpsertOrgOrderBehavior(ctx context.Context, b *OrgOrderBehavior) error {
	query := `
		INSERT INTO org_order_behavior (
			org_id, total_orders_as_customer, completed_as_customer, cancelled_as_customer,
			total_orders_as_carrier, completed_as_carrier, cancelled_as_carrier,
			avg_completion_hours, min_completion_hours, is_suspicious, suspicious_reason, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, NOW())
		ON CONFLICT (org_id) DO UPDATE SET
			total_orders_as_customer = $2,
			completed_as_customer = $3,
			cancelled_as_customer = $4,
			total_orders_as_carrier = $5,
			completed_as_carrier = $6,
			cancelled_as_carrier = $7,
			avg_completion_hours = $8,
			min_completion_hours = $9,
			is_suspicious = $10,
			suspicious_reason = $11,
			updated_at = NOW()
	`

	if _, err := p.db.Exec(ctx, query,
		b.OrgID, b.TotalOrdersAsCustomer, b.CompletedAsCustomer, b.CancelledAsCustomer,
		b.TotalOrdersAsCarrier, b.CompletedAsCarrier, b.CancelledAsCarrier,
		b.AvgCompletionHours, b.MinCompletionHours, b.IsSuspicious, b.SuspiciousReason,
	); err != nil {
		return fmt.Errorf("upsert org order behavior: %w", err)
	}

	return nil
}

// IncrementOrderCreated increments order count for customer and carrier
func (p *OrderFraudProjection) IncrementOrderCreated(ctx context.Context, customerOrgID, carrierOrgID uuid.UUID) error {
	query := `
		INSERT INTO org_order_behavior (org_id, total_orders_as_customer, updated_at)
		VALUES ($1, 1, NOW())
		ON CONFLICT (org_id) DO UPDATE SET
			total_orders_as_customer = org_order_behavior.total_orders_as_customer + 1,
			updated_at = NOW()
	`
	if _, err := p.db.Exec(ctx, query, customerOrgID); err != nil {
		return fmt.Errorf("increment customer orders: %w", err)
	}

	query = `
		INSERT INTO org_order_behavior (org_id, total_orders_as_carrier, updated_at)
		VALUES ($1, 1, NOW())
		ON CONFLICT (org_id) DO UPDATE SET
			total_orders_as_carrier = org_order_behavior.total_orders_as_carrier + 1,
			updated_at = NOW()
	`
	if _, err := p.db.Exec(ctx, query, carrierOrgID); err != nil {
		return fmt.Errorf("increment carrier orders: %w", err)
	}

	return nil
}

// IncrementOrderCancelled increments cancelled order count
func (p *OrderFraudProjection) IncrementOrderCancelled(ctx context.Context, orgID uuid.UUID, asCustomer bool) error {
	var query string
	if asCustomer {
		query = `
			UPDATE org_order_behavior SET
				cancelled_as_customer = cancelled_as_customer + 1,
				updated_at = NOW()
			WHERE org_id = $1
		`
	} else {
		query = `
			UPDATE org_order_behavior SET
				cancelled_as_carrier = cancelled_as_carrier + 1,
				updated_at = NOW()
			WHERE org_id = $1
		`
	}

	if _, err := p.db.Exec(ctx, query, orgID); err != nil {
		return fmt.Errorf("increment cancelled orders: %w", err)
	}

	return nil
}

// IncrementOrderCompleted increments completed order count and updates completion time
func (p *OrderFraudProjection) IncrementOrderCompleted(ctx context.Context, customerOrgID, carrierOrgID uuid.UUID, completionHours float64) error {
	// Update customer
	query := `
		UPDATE org_order_behavior SET
			completed_as_customer = completed_as_customer + 1,
			avg_completion_hours = CASE
				WHEN avg_completion_hours IS NULL THEN $2
				ELSE (avg_completion_hours * completed_as_customer + $2) / (completed_as_customer + 1)
			END,
			min_completion_hours = CASE
				WHEN min_completion_hours IS NULL THEN $2
				WHEN $2 < min_completion_hours THEN $2
				ELSE min_completion_hours
			END,
			updated_at = NOW()
		WHERE org_id = $1
	`
	if _, err := p.db.Exec(ctx, query, customerOrgID, completionHours); err != nil {
		return fmt.Errorf("increment customer completed: %w", err)
	}

	// Update carrier
	if _, err := p.db.Exec(ctx, query, carrierOrgID, completionHours); err != nil {
		return fmt.Errorf("increment carrier completed: %w", err)
	}

	return nil
}

// GetCancelRate returns cancellation rate for an organization
func (p *OrderFraudProjection) GetCancelRate(ctx context.Context, orgID uuid.UUID) (customerRate, carrierRate float64, err error) {
	b, err := p.GetOrgOrderBehavior(ctx, orgID)
	if err != nil {
		return 0, 0, err
	}

	if b.TotalOrdersAsCustomer > 0 {
		customerRate = float64(b.CancelledAsCustomer) / float64(b.TotalOrdersAsCustomer)
	}
	if b.TotalOrdersAsCarrier > 0 {
		carrierRate = float64(b.CancelledAsCarrier) / float64(b.TotalOrdersAsCarrier)
	}

	return customerRate, carrierRate, nil
}

// CircularOrderCheck contains data for circular order detection
type CircularOrderCheck struct {
	OrgA           uuid.UUID
	OrgB           uuid.UUID
	OrdersAToB     int
	OrdersBToA     int
	LastOrderAToB  *time.Time
	LastOrderBToA  *time.Time
}

// GetCircularOrderData returns data for detecting circular orders between two orgs
func (p *OrderFraudProjection) GetCircularOrderData(ctx context.Context, orgA, orgB uuid.UUID, sinceDays int) (*CircularOrderCheck, error) {
	query := `
		SELECT
			COALESCE((SELECT COUNT(*) FROM orders_lookup
			 WHERE customer_org_id = $1 AND carrier_org_id = $2
			   AND status = 'completed'
			   AND created_at > NOW() - ($3 || ' days')::interval), 0) as orders_a_to_b,
			COALESCE((SELECT COUNT(*) FROM orders_lookup
			 WHERE customer_org_id = $2 AND carrier_org_id = $1
			   AND status = 'completed'
			   AND created_at > NOW() - ($3 || ' days')::interval), 0) as orders_b_to_a,
			(SELECT MAX(created_at) FROM orders_lookup
			 WHERE customer_org_id = $1 AND carrier_org_id = $2
			   AND status = 'completed') as last_a_to_b,
			(SELECT MAX(created_at) FROM orders_lookup
			 WHERE customer_org_id = $2 AND carrier_org_id = $1
			   AND status = 'completed') as last_b_to_a
	`

	var check CircularOrderCheck
	check.OrgA = orgA
	check.OrgB = orgB

	if err := p.db.QueryRow(ctx, query, orgA, orgB, sinceDays).Scan(
		&check.OrdersAToB, &check.OrdersBToA,
		&check.LastOrderAToB, &check.LastOrderBToA,
	); err != nil {
		return nil, fmt.Errorf("query circular order data: %w", err)
	}

	return &check, nil
}

// OrderChain represents a detected order chain
type OrderChain struct {
	ID          uuid.UUID
	ChainOrgs   []uuid.UUID
	ChainLength int
	OrderIDs    []uuid.UUID
	TotalAmount int64
	FirstOrderAt time.Time
	LastOrderAt  time.Time
	IsSuspicious bool
}

// InsertOrderChain inserts a detected order chain
func (p *OrderFraudProjection) InsertOrderChain(ctx context.Context, chain *OrderChain) error {
	orgsJSON, err := json.Marshal(chain.ChainOrgs)
	if err != nil {
		return fmt.Errorf("marshal chain orgs: %w", err)
	}

	orderIDsJSON, err := json.Marshal(chain.OrderIDs)
	if err != nil {
		return fmt.Errorf("marshal order ids: %w", err)
	}

	query := `
		INSERT INTO org_order_chains (
			chain_orgs, chain_length, order_ids, total_amount,
			first_order_at, last_order_at, is_suspicious
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	if _, err := p.db.Exec(ctx, query,
		orgsJSON, chain.ChainLength, orderIDsJSON, chain.TotalAmount,
		chain.FirstOrderAt, chain.LastOrderAt, chain.IsSuspicious,
	); err != nil {
		return fmt.Errorf("insert order chain: %w", err)
	}

	return nil
}

// GetOrderCompletionTime returns order creation and completion times
func (p *OrderFraudProjection) GetOrderCompletionTime(ctx context.Context, orderID uuid.UUID) (createdAt, completedAt *time.Time, err error) {
	query := `
		SELECT created_at,
			(SELECT MAX(e.occurred_at)
			 FROM events e
			 WHERE e.aggregate_id = $1
			   AND e.event_type = 'order.completed') as completed_at
		FROM orders_lookup
		WHERE id = $1
	`

	if err := p.db.QueryRow(ctx, query, orderID).Scan(&createdAt, &completedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil, nil
		}
		return nil, nil, fmt.Errorf("query order times: %w", err)
	}

	return createdAt, completedAt, nil
}

// GetSuspiciousOrgs returns list of suspicious organizations
func (p *OrderFraudProjection) GetSuspiciousOrgs(ctx context.Context, limit, offset int) ([]OrgOrderBehavior, int, error) {
	// Count total
	var total int
	if err := p.db.QueryRow(ctx, `SELECT COUNT(*) FROM org_order_behavior WHERE is_suspicious = TRUE`).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count suspicious: %w", err)
	}

	if total == 0 {
		return []OrgOrderBehavior{}, 0, nil
	}

	query := `
		SELECT org_id, total_orders_as_customer, completed_as_customer, cancelled_as_customer,
			   total_orders_as_carrier, completed_as_carrier, cancelled_as_carrier,
			   avg_completion_hours, min_completion_hours, is_suspicious, suspicious_reason
		FROM org_order_behavior
		WHERE is_suspicious = TRUE
		ORDER BY updated_at DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := p.db.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("query suspicious: %w", err)
	}
	defer rows.Close()

	var result []OrgOrderBehavior
	for rows.Next() {
		var b OrgOrderBehavior
		if err := rows.Scan(
			&b.OrgID, &b.TotalOrdersAsCustomer, &b.CompletedAsCustomer, &b.CancelledAsCustomer,
			&b.TotalOrdersAsCarrier, &b.CompletedAsCarrier, &b.CancelledAsCarrier,
			&b.AvgCompletionHours, &b.MinCompletionHours, &b.IsSuspicious, &b.SuspiciousReason,
		); err != nil {
			return nil, 0, fmt.Errorf("scan behavior: %w", err)
		}
		result = append(result, b)
	}

	return result, total, nil
}
