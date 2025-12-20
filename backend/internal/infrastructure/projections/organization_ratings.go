package projections

import (
	"context"
	"errors"
	"fmt"
	"time"

	"codeberg.org/udison/veziizi/backend/internal/domain/review/values"
	"codeberg.org/udison/veziizi/backend/internal/pkg/dbtx"
	"github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type OrganizationRatingsProjection struct {
	db   dbtx.TxManager
	psql squirrel.StatementBuilderType
}

func NewOrganizationRatingsProjection(db dbtx.TxManager) *OrganizationRatingsProjection {
	return &OrganizationRatingsProjection{
		db:   db,
		psql: squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar),
	}
}

// OrganizationRating represents aggregated rating for an organization
type OrganizationRating struct {
	OrgID           uuid.UUID `json:"org_id"`
	TotalReviews    int       `json:"total_reviews"`
	AverageRating   float64   `json:"average_rating"`
	WeightedAverage float64   `json:"weighted_average"`
	PendingReviews  int       `json:"pending_reviews"`
}

// ReviewListItem represents a single review for listing
type ReviewListItem struct {
	ID              uuid.UUID `json:"id"`
	OrderID         uuid.UUID `json:"order_id"`
	ReviewerOrgID   uuid.UUID `json:"reviewer_org_id"`
	ReviewedOrgID   uuid.UUID `json:"reviewed_org_id"`
	Rating          int       `json:"rating"`
	Comment         string    `json:"comment"`
	Status          string    `json:"status"`
	RawWeight       float64   `json:"raw_weight"`
	FinalWeight     float64   `json:"final_weight"`
	FraudScore      float64   `json:"fraud_score"`
	ActivationDate  *time.Time `json:"activation_date,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
}

// GetRating returns aggregated rating for an organization
func (p *OrganizationRatingsProjection) GetRating(ctx context.Context, orgID uuid.UUID) (*OrganizationRating, error) {
	query, args, err := p.psql.
		Select("org_id", "total_reviews", "average_rating", "weighted_average", "pending_reviews").
		From("organization_ratings").
		Where(squirrel.Eq{"org_id": orgID}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build select query: %w", err)
	}

	var rating OrganizationRating
	if err := p.db.QueryRow(ctx, query, args...).Scan(
		&rating.OrgID,
		&rating.TotalReviews,
		&rating.AverageRating,
		&rating.WeightedAverage,
		&rating.PendingReviews,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// If no rating exists, return zero rating
			return &OrganizationRating{
				OrgID:           orgID,
				TotalReviews:    0,
				AverageRating:   0,
				WeightedAverage: 0,
				PendingReviews:  0,
			}, nil
		}
		return nil, fmt.Errorf("scan rating: %w", err)
	}

	return &rating, nil
}

// ListReviews returns active reviews for an organization with pagination
func (p *OrganizationRatingsProjection) ListReviews(ctx context.Context, orgID uuid.UUID, limit, offset int) ([]ReviewListItem, int, error) {
	// Count total active reviews
	countQuery, countArgs, err := p.psql.
		Select("COUNT(*)").
		From("reviews_lookup").
		Where(squirrel.And{
			squirrel.Eq{"reviewed_org_id": orgID},
			squirrel.Eq{"status": values.StatusActive.String()},
		}).
		ToSql()
	if err != nil {
		return nil, 0, fmt.Errorf("build count query: %w", err)
	}

	var total int
	if err := p.db.QueryRow(ctx, countQuery, countArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count reviews: %w", err)
	}

	if total == 0 {
		return []ReviewListItem{}, 0, nil
	}

	// Get reviews
	query, args, err := p.psql.
		Select("id", "order_id", "reviewer_org_id", "reviewed_org_id", "rating", "comment", "status", "raw_weight", "final_weight", "fraud_score", "activation_date", "created_at").
		From("reviews_lookup").
		Where(squirrel.And{
			squirrel.Eq{"reviewed_org_id": orgID},
			squirrel.Eq{"status": values.StatusActive.String()},
		}).
		OrderBy("created_at DESC").
		Limit(uint64(limit)).
		Offset(uint64(offset)).
		ToSql()
	if err != nil {
		return nil, 0, fmt.Errorf("build select query: %w", err)
	}

	rows, err := p.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("query reviews: %w", err)
	}
	defer rows.Close()

	result := make([]ReviewListItem, 0)
	for rows.Next() {
		var item ReviewListItem
		if err := rows.Scan(
			&item.ID,
			&item.OrderID,
			&item.ReviewerOrgID,
			&item.ReviewedOrgID,
			&item.Rating,
			&item.Comment,
			&item.Status,
			&item.RawWeight,
			&item.FinalWeight,
			&item.FraudScore,
			&item.ActivationDate,
			&item.CreatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan row: %w", err)
		}
		result = append(result, item)
	}

	return result, total, nil
}

// AddWeightedRating updates the aggregated rating when a review is activated
func (p *OrganizationRatingsProjection) AddWeightedRating(ctx context.Context, orgID uuid.UUID, rating int, weight float64) error {
	weightedRating := float64(rating) * weight

	query := `
		INSERT INTO organization_ratings (org_id, total_reviews, sum_rating, average_rating, weighted_sum, weight_total, weighted_average, updated_at)
		VALUES ($1, 1, $2, $2, $3, $4, $2, NOW())
		ON CONFLICT (org_id) DO UPDATE SET
			total_reviews = organization_ratings.total_reviews + 1,
			sum_rating = organization_ratings.sum_rating + $2,
			average_rating = (organization_ratings.sum_rating + $2)::numeric / (organization_ratings.total_reviews + 1),
			weighted_sum = organization_ratings.weighted_sum + $3,
			weight_total = organization_ratings.weight_total + $4,
			weighted_average = CASE
				WHEN organization_ratings.weight_total + $4 > 0
				THEN (organization_ratings.weighted_sum + $3) / (organization_ratings.weight_total + $4)
				ELSE 0
			END,
			updated_at = NOW()
	`

	if _, err := p.db.Exec(ctx, query, orgID, rating, weightedRating, weight); err != nil {
		return fmt.Errorf("add weighted rating: %w", err)
	}

	return nil
}

// RemoveWeightedRating removes a review's contribution from the rating (when deactivated)
func (p *OrganizationRatingsProjection) RemoveWeightedRating(ctx context.Context, orgID uuid.UUID, rating int, weight float64) error {
	weightedRating := float64(rating) * weight

	query := `
		UPDATE organization_ratings SET
			total_reviews = GREATEST(0, total_reviews - 1),
			sum_rating = GREATEST(0, sum_rating - $2),
			average_rating = CASE
				WHEN total_reviews - 1 > 0
				THEN GREATEST(0, sum_rating - $2)::numeric / (total_reviews - 1)
				ELSE 0
			END,
			weighted_sum = GREATEST(0, weighted_sum - $3),
			weight_total = GREATEST(0, weight_total - $4),
			weighted_average = CASE
				WHEN weight_total - $4 > 0
				THEN GREATEST(0, weighted_sum - $3) / (weight_total - $4)
				ELSE 0
			END,
			updated_at = NOW()
		WHERE org_id = $1
	`

	if _, err := p.db.Exec(ctx, query, orgID, rating, weightedRating, weight); err != nil {
		return fmt.Errorf("remove weighted rating: %w", err)
	}

	return nil
}

// IncrementPendingReviews increments the pending reviews counter
func (p *OrganizationRatingsProjection) IncrementPendingReviews(ctx context.Context, orgID uuid.UUID) error {
	query := `
		INSERT INTO organization_ratings (org_id, pending_reviews, updated_at)
		VALUES ($1, 1, NOW())
		ON CONFLICT (org_id) DO UPDATE SET
			pending_reviews = organization_ratings.pending_reviews + 1,
			updated_at = NOW()
	`

	if _, err := p.db.Exec(ctx, query, orgID); err != nil {
		return fmt.Errorf("increment pending reviews: %w", err)
	}

	return nil
}

// DecrementPendingReviews decrements the pending reviews counter
func (p *OrganizationRatingsProjection) DecrementPendingReviews(ctx context.Context, orgID uuid.UUID) error {
	query := `
		UPDATE organization_ratings SET
			pending_reviews = GREATEST(0, pending_reviews - 1),
			updated_at = NOW()
		WHERE org_id = $1
	`

	if _, err := p.db.Exec(ctx, query, orgID); err != nil {
		return fmt.Errorf("decrement pending reviews: %w", err)
	}

	return nil
}

// IncrementRejectedReviews increments the rejected reviews counter
func (p *OrganizationRatingsProjection) IncrementRejectedReviews(ctx context.Context, orgID uuid.UUID) error {
	query := `
		INSERT INTO organization_ratings (org_id, rejected_reviews, updated_at)
		VALUES ($1, 1, NOW())
		ON CONFLICT (org_id) DO UPDATE SET
			rejected_reviews = organization_ratings.rejected_reviews + 1,
			updated_at = NOW()
	`

	if _, err := p.db.Exec(ctx, query, orgID); err != nil {
		return fmt.Errorf("increment rejected reviews: %w", err)
	}

	return nil
}
