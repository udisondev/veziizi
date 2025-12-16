package projections

import (
	"context"
	"fmt"
	"time"

	"codeberg.org/udison/veziizi/backend/internal/pkg/dbtx"
	"github.com/Masterminds/squirrel"
	"github.com/google/uuid"
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
	OrgID         uuid.UUID `json:"org_id"`
	TotalReviews  int       `json:"total_reviews"`
	AverageRating float64   `json:"average_rating"`
}

// ReviewListItem represents a single review for listing
type ReviewListItem struct {
	ID              uuid.UUID `json:"id"`
	OrderID         uuid.UUID `json:"order_id"`
	ReviewerOrgID   uuid.UUID `json:"reviewer_org_id"`
	ReviewerOrgName string    `json:"reviewer_org_name"`
	Rating          int       `json:"rating"`
	Comment         string    `json:"comment"`
	CreatedAt       time.Time `json:"created_at"`
}

// GetRating returns aggregated rating for an organization
func (p *OrganizationRatingsProjection) GetRating(ctx context.Context, orgID uuid.UUID) (*OrganizationRating, error) {
	query, args, err := p.psql.
		Select("org_id", "total_reviews", "average_rating").
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
	); err != nil {
		// If no rating exists, return zero rating
		return &OrganizationRating{
			OrgID:         orgID,
			TotalReviews:  0,
			AverageRating: 0,
		}, nil
	}

	return &rating, nil
}

// ListReviews returns reviews for an organization with pagination
func (p *OrganizationRatingsProjection) ListReviews(ctx context.Context, orgID uuid.UUID, limit, offset int) ([]ReviewListItem, int, error) {
	// Count total
	countQuery, countArgs, err := p.psql.
		Select("COUNT(*)").
		From("organization_reviews_lookup").
		Where(squirrel.Eq{"reviewed_org_id": orgID}).
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
		Select("id", "order_id", "reviewer_org_id", "reviewer_org_name", "rating", "comment", "created_at").
		From("organization_reviews_lookup").
		Where(squirrel.Eq{"reviewed_org_id": orgID}).
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
			&item.ReviewerOrgName,
			&item.Rating,
			&item.Comment,
			&item.CreatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan row: %w", err)
		}
		result = append(result, item)
	}

	return result, total, nil
}

// AddReview inserts a new review into the lookup table
func (p *OrganizationRatingsProjection) AddReview(ctx context.Context, review ReviewListItem, reviewedOrgID uuid.UUID) error {
	query, args, err := p.psql.
		Insert("organization_reviews_lookup").
		Columns("id", "order_id", "reviewer_org_id", "reviewer_org_name", "reviewed_org_id", "rating", "comment", "created_at").
		Values(review.ID, review.OrderID, review.ReviewerOrgID, review.ReviewerOrgName, reviewedOrgID, review.Rating, review.Comment, review.CreatedAt).
		ToSql()
	if err != nil {
		return fmt.Errorf("build insert query: %w", err)
	}

	if _, err := p.db.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("insert review: %w", err)
	}

	return nil
}

// UpdateRating updates the aggregated rating for an organization (UPSERT)
func (p *OrganizationRatingsProjection) UpdateRating(ctx context.Context, orgID uuid.UUID, rating int) error {
	query := `
		INSERT INTO organization_ratings (org_id, total_reviews, sum_rating, average_rating)
		VALUES ($1, 1, $2, $2)
		ON CONFLICT (org_id) DO UPDATE SET
			total_reviews = organization_ratings.total_reviews + 1,
			sum_rating = organization_ratings.sum_rating + $2,
			average_rating = (organization_ratings.sum_rating + $2)::numeric / (organization_ratings.total_reviews + 1)
	`

	if _, err := p.db.Exec(ctx, query, orgID, rating); err != nil {
		return fmt.Errorf("update rating: %w", err)
	}

	return nil
}
