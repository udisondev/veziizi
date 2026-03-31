package projections

import (
	"context"
	"fmt"
	"time"

	"github.com/udisondev/veziizi/backend/internal/domain/review/values"
	"github.com/udisondev/veziizi/backend/internal/pkg/dbtx"
	"github.com/Masterminds/squirrel"
	"github.com/google/uuid"
)

// ReviewsProjection provides read-side operations for reviews
type ReviewsProjection struct {
	db   dbtx.TxManager
	psql squirrel.StatementBuilderType
}

func NewReviewsProjection(db dbtx.TxManager) *ReviewsProjection {
	return &ReviewsProjection{
		db:   db,
		psql: squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar),
	}
}

// ReviewForModeration represents a review pending moderation
type ReviewForModeration struct {
	ID                uuid.UUID       `json:"id"`
	OrderID           uuid.UUID       `json:"order_id"`
	ReviewerOrgID     uuid.UUID       `json:"reviewer_org_id"`
	ReviewerOrgName   string          `json:"reviewer_org_name,omitempty"`
	ReviewedOrgID     uuid.UUID       `json:"reviewed_org_id"`
	ReviewedOrgName   string          `json:"reviewed_org_name,omitempty"`
	Rating            int             `json:"rating"`
	Comment           string          `json:"comment"`
	OrderAmount       int64           `json:"order_amount"`
	OrderCurrency     string          `json:"order_currency"`
	RawWeight         float64         `json:"raw_weight"`
	FraudScore        float64         `json:"fraud_score"`
	FraudSignals      []FraudSignalInfo `json:"fraud_signals"`
	ActivationDate    *time.Time      `json:"activation_date"`
	CreatedAt         time.Time       `json:"created_at"`
	AnalyzedAt        *time.Time      `json:"analyzed_at"`
}

// FraudSignalInfo represents fraud signal details
type FraudSignalInfo struct {
	Type        string  `json:"type"`
	Severity    string  `json:"severity"`
	Description string  `json:"description"`
	ScoreImpact float64 `json:"score_impact"`
}

// ListPendingModeration returns reviews pending moderation with pagination
// Использует COUNT(*) OVER() для получения total в одном запросе
func (p *ReviewsProjection) ListPendingModeration(ctx context.Context, limit, offset int) ([]ReviewForModeration, int, error) {
	// Один запрос с COUNT(*) OVER() вместо двух отдельных
	query := `
		SELECT
			id, order_id, reviewer_org_id, reviewed_org_id,
			rating, comment, order_amount, order_currency,
			raw_weight, fraud_score, activation_date, created_at, analyzed_at,
			COUNT(*) OVER() as total_count
		FROM reviews_lookup
		WHERE status = $1
		ORDER BY fraud_score DESC, created_at ASC
		LIMIT $2 OFFSET $3
	`

	rows, err := p.db.Query(ctx, query, values.StatusPendingModeration.String(), limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("query pending reviews: %w", err)
	}
	defer rows.Close()

	var total int
	result := make([]ReviewForModeration, 0, limit)
	for rows.Next() {
		var r ReviewForModeration
		if err := rows.Scan(
			&r.ID, &r.OrderID, &r.ReviewerOrgID, &r.ReviewedOrgID,
			&r.Rating, &r.Comment, &r.OrderAmount, &r.OrderCurrency,
			&r.RawWeight, &r.FraudScore, &r.ActivationDate, &r.CreatedAt, &r.AnalyzedAt,
			&total,
		); err != nil {
			return nil, 0, fmt.Errorf("scan row: %w", err)
		}
		result = append(result, r)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("rows iteration: %w", err)
	}

	if len(result) == 0 {
		return []ReviewForModeration{}, 0, nil
	}

	// Load fraud signals for each review
	for i := range result {
		signals, err := p.getFraudSignals(ctx, result[i].ID)
		if err != nil {
			return nil, 0, fmt.Errorf("get fraud signals: %w", err)
		}
		result[i].FraudSignals = signals
	}

	return result, total, nil
}

// GetFraudSignalsByReviewID returns fraud signals for a review (public)
func (p *ReviewsProjection) GetFraudSignalsByReviewID(ctx context.Context, reviewID uuid.UUID) ([]FraudSignalInfo, error) {
	return p.getFraudSignals(ctx, reviewID)
}

// getFraudSignals returns fraud signals for a review
func (p *ReviewsProjection) getFraudSignals(ctx context.Context, reviewID uuid.UUID) ([]FraudSignalInfo, error) {
	query, args, err := p.psql.
		Select("signal_type", "severity", "description", "score_impact").
		From("review_fraud_signals").
		Where(squirrel.Eq{"review_id": reviewID}).
		OrderBy("score_impact DESC").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build signals query: %w", err)
	}

	rows, err := p.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query signals: %w", err)
	}
	defer rows.Close()

	signals := make([]FraudSignalInfo, 0)
	for rows.Next() {
		var s FraudSignalInfo
		if err := rows.Scan(&s.Type, &s.Severity, &s.Description, &s.ScoreImpact); err != nil {
			return nil, fmt.Errorf("scan signal: %w", err)
		}
		signals = append(signals, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration: %w", err)
	}

	return signals, nil
}

// GetReviewByID returns a review by ID
func (p *ReviewsProjection) GetReviewByID(ctx context.Context, id uuid.UUID) (*ReviewLookupRow, error) {
	query, args, err := p.psql.
		Select(
			"id", "order_id", "reviewer_org_id", "reviewed_org_id",
			"rating", "comment", "order_amount", "order_currency",
			"order_created_at", "order_completed_at",
			"raw_weight", "final_weight", "fraud_score", "requires_moderation",
			"status", "activation_date", "created_at", "analyzed_at",
			"moderated_at", "moderated_by", "activated_at",
		).
		From("reviews_lookup").
		Where(squirrel.Eq{"id": id}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build query: %w", err)
	}

	var r ReviewLookupRow
	if err := p.db.QueryRow(ctx, query, args...).Scan(
		&r.ID, &r.OrderID, &r.ReviewerOrgID, &r.ReviewedOrgID,
		&r.Rating, &r.Comment, &r.OrderAmount, &r.OrderCurrency,
		&r.OrderCreatedAt, &r.OrderCompletedAt,
		&r.RawWeight, &r.FinalWeight, &r.FraudScore, &r.RequiresModeration,
		&r.Status, &r.ActivationDate, &r.CreatedAt, &r.AnalyzedAt,
		&r.ModeratedAt, &r.ModeratedBy, &r.ActivatedAt,
	); err != nil {
		return nil, fmt.Errorf("scan review: %w", err)
	}

	return &r, nil
}

// ListReviewsForActivation returns approved reviews ready for activation
func (p *ReviewsProjection) ListReviewsForActivation(ctx context.Context, limit int) ([]uuid.UUID, error) {
	query, args, err := p.psql.
		Select("id").
		From("reviews_lookup").
		Where(squirrel.And{
			squirrel.Eq{"status": values.StatusApproved.String()},
			squirrel.LtOrEq{"activation_date": time.Now()},
		}).
		OrderBy("activation_date ASC").
		Limit(uint64(limit)).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build query: %w", err)
	}

	rows, err := p.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query reviews for activation: %w", err)
	}
	defer rows.Close()

	ids := make([]uuid.UUID, 0)
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan id: %w", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration: %w", err)
	}

	return ids, nil
}

// ReviewsByReviewer returns reviews left by an organization
type ReviewsByReviewerFilter struct {
	ReviewerOrgID uuid.UUID
	Status        *values.ReviewStatus
	Limit         int
	Offset        int
}

func (p *ReviewsProjection) ListByReviewer(ctx context.Context, filter ReviewsByReviewerFilter) ([]ReviewListItem, int, error) {
	// Строим запрос с COUNT(*) OVER() для получения total в одном запросе
	var statusFilter string
	args := []any{filter.ReviewerOrgID}
	argNum := 2

	if filter.Status != nil {
		statusFilter = fmt.Sprintf(" AND status = $%d", argNum)
		args = append(args, filter.Status.String())
		argNum++
	}

	query := fmt.Sprintf(`
		SELECT
			id, order_id, reviewer_org_id, reviewed_org_id,
			rating, comment, status, raw_weight, final_weight,
			fraud_score, activation_date, created_at,
			COUNT(*) OVER() as total_count
		FROM reviews_lookup
		WHERE reviewer_org_id = $1%s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, statusFilter, argNum, argNum+1)

	args = append(args, filter.Limit, filter.Offset)

	rows, err := p.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("query reviews: %w", err)
	}
	defer rows.Close()

	var total int
	result := make([]ReviewListItem, 0, filter.Limit)
	for rows.Next() {
		var item ReviewListItem
		if err := rows.Scan(
			&item.ID, &item.OrderID, &item.ReviewerOrgID, &item.ReviewedOrgID,
			&item.Rating, &item.Comment, &item.Status, &item.RawWeight,
			&item.FinalWeight, &item.FraudScore, &item.ActivationDate, &item.CreatedAt,
			&total,
		); err != nil {
			return nil, 0, fmt.Errorf("scan row: %w", err)
		}
		result = append(result, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("rows iteration: %w", err)
	}

	if len(result) == 0 {
		return []ReviewListItem{}, 0, nil
	}

	return result, total, nil
}

// ReviewLookupRow is re-exported from fraud_data for convenience
// Already defined in fraud_data.go, no need to redefine

// ListActiveReviewsByReviewer returns IDs of active reviews by reviewer organization
func (p *ReviewsProjection) ListActiveReviewsByReviewer(ctx context.Context, reviewerOrgID uuid.UUID) ([]uuid.UUID, error) {
	query, args, err := p.psql.
		Select("id").
		From("reviews_lookup").
		Where(squirrel.And{
			squirrel.Eq{"reviewer_org_id": reviewerOrgID},
			squirrel.Eq{"status": values.StatusActive.String()},
		}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build query: %w", err)
	}

	rows, err := p.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query active reviews: %w", err)
	}
	defer rows.Close()

	ids := make([]uuid.UUID, 0)
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan id: %w", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration: %w", err)
	}

	return ids, nil
}
