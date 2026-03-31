package fixtures

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/udisondev/veziizi/backend/internal/infrastructure/projections"
	"github.com/udisondev/veziizi/backend/internal/pkg/factory"
)

// DefaultReviewLookupRow returns a ReviewLookupRow with sensible defaults.
// Caller can override specific fields after creation.
func DefaultReviewLookupRow(reviewerOrgID, reviewedOrgID uuid.UUID) *projections.ReviewLookupRow {
	now := time.Now()
	return &projections.ReviewLookupRow{
		ID:               uuid.New(),
		OrderID:          uuid.New(),
		ReviewerOrgID:    reviewerOrgID,
		ReviewedOrgID:    reviewedOrgID,
		Rating:           4,
		Comment:          "good service",
		OrderAmount:      100_000_00,
		OrderCurrency:    "RUB",
		OrderCreatedAt:   now.Add(-48 * time.Hour),
		OrderCompletedAt: now.Add(-24 * time.Hour),
		RawWeight:        1.0,
		FinalWeight:      1.0,
		FraudScore:       0,
		Status:           "active",
		CreatedAt:        now,
	}
}

// InsertReviewLookup inserts a review row directly into reviews_lookup via SQL.
func InsertReviewLookup(t *testing.T, f *factory.Factory, row *projections.ReviewLookupRow) {
	t.Helper()

	ctx := context.Background()
	query := `
		INSERT INTO reviews_lookup (
			id, order_id, reviewer_org_id, reviewed_org_id, rating, comment,
			order_amount, order_currency, order_created_at, order_completed_at,
			raw_weight, final_weight, fraud_score, requires_moderation, status,
			activation_date, created_at, analyzed_at, moderated_at, moderated_by, activated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21)
	`

	if _, err := f.DB().Exec(ctx, query,
		row.ID, row.OrderID, row.ReviewerOrgID, row.ReviewedOrgID, row.Rating, row.Comment,
		row.OrderAmount, row.OrderCurrency, row.OrderCreatedAt, row.OrderCompletedAt,
		row.RawWeight, row.FinalWeight, row.FraudScore, row.RequiresModeration, row.Status,
		row.ActivationDate, row.CreatedAt, row.AnalyzedAt, row.ModeratedAt, row.ModeratedBy, row.ActivatedAt,
	); err != nil {
		t.Fatalf("insert review lookup: %v", err)
	}
}

// InsertInteractionStats inserts a row into org_interaction_stats.
// Ensures org_a < org_b constraint by swapping IDs and review counts when needed.
func InsertInteractionStats(t *testing.T, f *factory.Factory, orgA, orgB uuid.UUID, reviewsAToB, reviewsBToA, sumRatingAToB, sumRatingBToA int) {
	t.Helper()

	// Гарантируем org_a < org_b, при необходимости свопаем направления отзывов
	if orgA.String() > orgB.String() {
		orgA, orgB = orgB, orgA
		reviewsAToB, reviewsBToA = reviewsBToA, reviewsAToB
		sumRatingAToB, sumRatingBToA = sumRatingBToA, sumRatingAToB
	}

	ctx := context.Background()
	query := `
		INSERT INTO org_interaction_stats (
			org_a, org_b, reviews_a_to_b, reviews_b_to_a,
			sum_rating_a_to_b, sum_rating_b_to_a, last_interaction_at
		) VALUES ($1, $2, $3, $4, $5, $6, NOW())
		ON CONFLICT (org_a, org_b) DO UPDATE SET
			reviews_a_to_b = EXCLUDED.reviews_a_to_b,
			reviews_b_to_a = EXCLUDED.reviews_b_to_a,
			sum_rating_a_to_b = EXCLUDED.sum_rating_a_to_b,
			sum_rating_b_to_a = EXCLUDED.sum_rating_b_to_a,
			last_interaction_at = NOW()
	`

	if _, err := f.DB().Exec(ctx, query, orgA, orgB, reviewsAToB, reviewsBToA, sumRatingAToB, sumRatingBToA); err != nil {
		t.Fatalf("insert interaction stats: %v", err)
	}
}

// ReputationOpts configures org_reviewer_reputation fields.
type ReputationOpts struct {
	TotalReviewsLeft     int
	ActiveReviewsLeft    int
	RejectedReviews      int
	DeactivatedReviews   int
	ReputationScore      float64
	IsSuspectedFraudster bool
	IsConfirmedFraudster bool
}

// InsertOrgReputation inserts or updates a row in org_reviewer_reputation.
func InsertOrgReputation(t *testing.T, f *factory.Factory, orgID uuid.UUID, opts ReputationOpts) {
	t.Helper()

	ctx := context.Background()
	query := `
		INSERT INTO org_reviewer_reputation (
			org_id, total_reviews_left, active_reviews_left,
			rejected_reviews, deactivated_reviews, reputation_score,
			is_suspected_fraudster, is_confirmed_fraudster, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW())
		ON CONFLICT (org_id) DO UPDATE SET
			total_reviews_left = EXCLUDED.total_reviews_left,
			active_reviews_left = EXCLUDED.active_reviews_left,
			rejected_reviews = EXCLUDED.rejected_reviews,
			deactivated_reviews = EXCLUDED.deactivated_reviews,
			reputation_score = EXCLUDED.reputation_score,
			is_suspected_fraudster = EXCLUDED.is_suspected_fraudster,
			is_confirmed_fraudster = EXCLUDED.is_confirmed_fraudster,
			updated_at = NOW()
	`

	if _, err := f.DB().Exec(ctx, query,
		orgID, opts.TotalReviewsLeft, opts.ActiveReviewsLeft,
		opts.RejectedReviews, opts.DeactivatedReviews, opts.ReputationScore,
		opts.IsSuspectedFraudster, opts.IsConfirmedFraudster,
	); err != nil {
		t.Fatalf("insert org reputation: %v", err)
	}
}

// SetOrgCreatedAt backdates created_at in members_lookup for weight calculation tests.
func SetOrgCreatedAt(t *testing.T, f *factory.Factory, orgID uuid.UUID, createdAt time.Time) {
	t.Helper()

	ctx := context.Background()
	query := `UPDATE members_lookup SET created_at = $1 WHERE organization_id = $2`

	if _, err := f.DB().Exec(ctx, query, createdAt, orgID); err != nil {
		t.Fatalf("set org created_at: %v", err)
	}
}

// SetMemberMetadata updates registration/login IP and fingerprint in members_lookup.
// Used for SameIP/SameFingerprint fraud detection tests.
func SetMemberMetadata(t *testing.T, f *factory.Factory, orgID uuid.UUID, ip, fingerprint string) {
	t.Helper()

	ctx := context.Background()
	query := `
		UPDATE members_lookup SET
			registration_ip = $1,
			last_login_ip = $1,
			registration_fingerprint = $2,
			last_login_fingerprint = $2
		WHERE organization_id = $3
	`

	if _, err := f.DB().Exec(ctx, query, ip, fingerprint, orgID); err != nil {
		t.Fatalf("set member metadata: %v", err)
	}
}

// ReviewOpts configures shared fields for InsertMultipleReviews.
type ReviewOpts struct {
	ReviewerOrgID uuid.UUID
	ReviewedOrgID uuid.UUID
	Rating        int
	Comment       string
	Status        string    // по умолчанию "active"
	CreatedAt     time.Time // по умолчанию time.Now()
	OrderAmount   int64     // копейки, по умолчанию 10_000_00
	OrderCurrency string    // по умолчанию "RUB"
}

// InsertMultipleReviews inserts count review rows into reviews_lookup.
// Each row gets unique id and order_id. Shared fields taken from opts.
func InsertMultipleReviews(t *testing.T, f *factory.Factory, count int, opts ReviewOpts) {
	t.Helper()

	if opts.Status == "" {
		opts.Status = "active"
	}
	if opts.CreatedAt.IsZero() {
		opts.CreatedAt = time.Now()
	}
	if opts.OrderAmount == 0 {
		opts.OrderAmount = 10_000_00
	}
	if opts.OrderCurrency == "" {
		opts.OrderCurrency = "RUB"
	}

	for range count {
		row := &projections.ReviewLookupRow{
			ID:               uuid.New(),
			OrderID:          uuid.New(),
			ReviewerOrgID:    opts.ReviewerOrgID,
			ReviewedOrgID:    opts.ReviewedOrgID,
			Rating:           opts.Rating,
			Comment:          opts.Comment,
			OrderAmount:      opts.OrderAmount,
			OrderCurrency:    opts.OrderCurrency,
			OrderCreatedAt:   opts.CreatedAt.Add(-48 * time.Hour),
			OrderCompletedAt: opts.CreatedAt.Add(-24 * time.Hour),
			RawWeight:        1.0,
			FinalWeight:      1.0,
			Status:           opts.Status,
			CreatedAt:        opts.CreatedAt,
		}
		InsertReviewLookup(t, f, row)
	}
}

// InsertReviewWithTiming inserts a minimal review row for timing pattern tests.
// Only reviewer_org_id and created_at are specified; other fields use defaults.
func InsertReviewWithTiming(t *testing.T, f *factory.Factory, reviewerOrgID uuid.UUID, createdAt time.Time) {
	t.Helper()

	row := DefaultReviewLookupRow(reviewerOrgID, uuid.New())
	row.CreatedAt = createdAt
	InsertReviewLookup(t, f, row)
}
