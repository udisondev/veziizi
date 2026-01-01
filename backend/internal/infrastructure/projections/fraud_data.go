package projections

import (
	"context"
	"errors"
	"fmt"
	"time"

	"codeberg.org/udison/veziizi/backend/internal/pkg/dbtx"
	"github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// FraudDataProjection provides data access for fraud detection analysis
type FraudDataProjection struct {
	db   dbtx.TxManager
	psql squirrel.StatementBuilderType
}

func NewFraudDataProjection(db dbtx.TxManager) *FraudDataProjection {
	return &FraudDataProjection{
		db:   db,
		psql: squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar),
	}
}

// OrgInteractionStats holds interaction statistics between two organizations
type OrgInteractionStats struct {
	OrgA            uuid.UUID
	OrgB            uuid.UUID
	TotalOrders     int
	CompletedOrders int
	ReviewsAToB     int
	ReviewsBToA     int
	SumRatingAToB   int
	SumRatingBToA   int
}

// GetInteractionStats returns interaction statistics between two organizations
func (p *FraudDataProjection) GetInteractionStats(ctx context.Context, orgA, orgB uuid.UUID) (*OrgInteractionStats, error) {
	// Ensure consistent ordering (org_a < org_b in table)
	if orgA.String() > orgB.String() {
		orgA, orgB = orgB, orgA
	}

	query, args, err := p.psql.
		Select(
			"org_a", "org_b", "total_orders", "completed_orders",
			"reviews_a_to_b", "reviews_b_to_a", "sum_rating_a_to_b", "sum_rating_b_to_a",
		).
		From("org_interaction_stats").
		Where(squirrel.And{
			squirrel.Eq{"org_a": orgA},
			squirrel.Eq{"org_b": orgB},
		}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build query: %w", err)
	}

	var stats OrgInteractionStats
	if err := p.db.QueryRow(ctx, query, args...).Scan(
		&stats.OrgA, &stats.OrgB, &stats.TotalOrders, &stats.CompletedOrders,
		&stats.ReviewsAToB, &stats.ReviewsBToA, &stats.SumRatingAToB, &stats.SumRatingBToA,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// No interactions yet - return empty stats
			return &OrgInteractionStats{
				OrgA: orgA,
				OrgB: orgB,
			}, nil
		}
		return nil, fmt.Errorf("scan interaction stats: %w", err)
	}

	return &stats, nil
}

// UpsertInteractionStats creates or updates interaction statistics
func (p *FraudDataProjection) UpsertInteractionStats(ctx context.Context, orgA, orgB uuid.UUID, update func(stats *OrgInteractionStats)) error {
	// Ensure consistent ordering
	if orgA.String() > orgB.String() {
		orgA, orgB = orgB, orgA
	}

	stats, err := p.GetInteractionStats(ctx, orgA, orgB)
	if err != nil {
		return fmt.Errorf("get stats: %w", err)
	}

	update(stats)

	query := `
		INSERT INTO org_interaction_stats (
			org_a, org_b, total_orders, completed_orders,
			reviews_a_to_b, reviews_b_to_a, sum_rating_a_to_b, sum_rating_b_to_a,
			last_interaction_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW())
		ON CONFLICT (org_a, org_b) DO UPDATE SET
			total_orders = $3,
			completed_orders = $4,
			reviews_a_to_b = $5,
			reviews_b_to_a = $6,
			sum_rating_a_to_b = $7,
			sum_rating_b_to_a = $8,
			last_interaction_at = NOW()
	`

	if _, err := p.db.Exec(ctx, query,
		stats.OrgA, stats.OrgB, stats.TotalOrders, stats.CompletedOrders,
		stats.ReviewsAToB, stats.ReviewsBToA, stats.SumRatingAToB, stats.SumRatingBToA,
	); err != nil {
		return fmt.Errorf("upsert interaction stats: %w", err)
	}

	return nil
}

// ReviewerReputation holds reputation data for a reviewer organization
type ReviewerReputation struct {
	OrgID                uuid.UUID
	TotalReviewsLeft     int
	ActiveReviewsLeft    int
	RejectedReviews      int
	DeactivatedReviews   int
	ReputationScore      float64
	IsSuspectedFraudster bool
	IsConfirmedFraudster bool
}

// GetReviewerReputation returns reputation data for a reviewer organization
func (p *FraudDataProjection) GetReviewerReputation(ctx context.Context, orgID uuid.UUID) (*ReviewerReputation, error) {
	query, args, err := p.psql.
		Select(
			"org_id", "total_reviews_left", "active_reviews_left",
			"rejected_reviews", "deactivated_reviews", "reputation_score",
			"is_suspected_fraudster", "is_confirmed_fraudster",
		).
		From("org_reviewer_reputation").
		Where(squirrel.Eq{"org_id": orgID}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build query: %w", err)
	}

	var rep ReviewerReputation
	if err := p.db.QueryRow(ctx, query, args...).Scan(
		&rep.OrgID, &rep.TotalReviewsLeft, &rep.ActiveReviewsLeft,
		&rep.RejectedReviews, &rep.DeactivatedReviews, &rep.ReputationScore,
		&rep.IsSuspectedFraudster, &rep.IsConfirmedFraudster,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// New organization, no reputation yet - return default
			return &ReviewerReputation{
				OrgID:           orgID,
				ReputationScore: 1.0,
			}, nil
		}
		return nil, fmt.Errorf("scan reviewer reputation: %w", err)
	}

	return &rep, nil
}

// UpsertReviewerReputation creates or updates reviewer reputation
func (p *FraudDataProjection) UpsertReviewerReputation(ctx context.Context, orgID uuid.UUID, update func(rep *ReviewerReputation)) error {
	rep, err := p.GetReviewerReputation(ctx, orgID)
	if err != nil {
		return fmt.Errorf("get reputation: %w", err)
	}

	update(rep)

	query := `
		INSERT INTO org_reviewer_reputation (
			org_id, total_reviews_left, active_reviews_left,
			rejected_reviews, deactivated_reviews, reputation_score,
			is_suspected_fraudster, is_confirmed_fraudster, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW())
		ON CONFLICT (org_id) DO UPDATE SET
			total_reviews_left = $2,
			active_reviews_left = $3,
			rejected_reviews = $4,
			deactivated_reviews = $5,
			reputation_score = $6,
			is_suspected_fraudster = $7,
			is_confirmed_fraudster = $8,
			updated_at = NOW()
	`

	if _, err := p.db.Exec(ctx, query,
		rep.OrgID, rep.TotalReviewsLeft, rep.ActiveReviewsLeft,
		rep.RejectedReviews, rep.DeactivatedReviews, rep.ReputationScore,
		rep.IsSuspectedFraudster, rep.IsConfirmedFraudster,
	); err != nil {
		return fmt.Errorf("upsert reviewer reputation: %w", err)
	}

	return nil
}

// CountReviewsInPeriod counts reviews left by reviewer to reviewed in a time period
func (p *FraudDataProjection) CountReviewsInPeriod(ctx context.Context, reviewerOrgID, reviewedOrgID uuid.UUID, since time.Time) (int, error) {
	query, args, err := p.psql.
		Select("COUNT(*)").
		From("reviews_lookup").
		Where(squirrel.And{
			squirrel.Eq{"reviewer_org_id": reviewerOrgID},
			squirrel.Eq{"reviewed_org_id": reviewedOrgID},
			squirrel.GtOrEq{"created_at": since},
		}).
		ToSql()
	if err != nil {
		return 0, fmt.Errorf("build query: %w", err)
	}

	var count int
	if err := p.db.QueryRow(ctx, query, args...).Scan(&count); err != nil {
		return 0, fmt.Errorf("count reviews: %w", err)
	}

	return count, nil
}

// CountReviewsReceivedInPeriod counts reviews received by an organization in a time period
func (p *FraudDataProjection) CountReviewsReceivedInPeriod(ctx context.Context, orgID uuid.UUID, since time.Time) (int, error) {
	query, args, err := p.psql.
		Select("COUNT(*)").
		From("reviews_lookup").
		Where(squirrel.And{
			squirrel.Eq{"reviewed_org_id": orgID},
			squirrel.GtOrEq{"created_at": since},
		}).
		ToSql()
	if err != nil {
		return 0, fmt.Errorf("build query: %w", err)
	}

	var count int
	if err := p.db.QueryRow(ctx, query, args...).Scan(&count); err != nil {
		return 0, fmt.Errorf("count reviews: %w", err)
	}

	return count, nil
}

// CountMutualReviewsInPeriod counts mutual reviews between two orgs in a period
func (p *FraudDataProjection) CountMutualReviewsInPeriod(ctx context.Context, orgA, orgB uuid.UUID, since time.Time) (aToB int, bToA int, err error) {
	aToB, err = p.CountReviewsInPeriod(ctx, orgA, orgB, since)
	if err != nil {
		return 0, 0, err
	}

	bToA, err = p.CountReviewsInPeriod(ctx, orgB, orgA, since)
	if err != nil {
		return 0, 0, err
	}

	return aToB, bToA, nil
}

// GetPreviousReviewsFromReviewer returns count and sum of ratings from reviewer to reviewed
func (p *FraudDataProjection) GetPreviousReviewsFromReviewer(ctx context.Context, reviewerOrgID, reviewedOrgID uuid.UUID) (count int, sumRating int, err error) {
	query, args, err := p.psql.
		Select("COUNT(*)", "COALESCE(SUM(rating), 0)").
		From("reviews_lookup").
		Where(squirrel.And{
			squirrel.Eq{"reviewer_org_id": reviewerOrgID},
			squirrel.Eq{"reviewed_org_id": reviewedOrgID},
			squirrel.Eq{"status": "active"},
		}).
		ToSql()
	if err != nil {
		return 0, 0, fmt.Errorf("build query: %w", err)
	}

	if err := p.db.QueryRow(ctx, query, args...).Scan(&count, &sumRating); err != nil {
		return 0, 0, fmt.Errorf("query reviews: %w", err)
	}

	return count, sumRating, nil
}

// GetOrgCreatedAt returns the creation time of an organization
func (p *FraudDataProjection) GetOrgCreatedAt(ctx context.Context, orgID uuid.UUID) (time.Time, error) {
	// Query from members_lookup which has created_at from org creation
	query, args, err := p.psql.
		Select("MIN(created_at)").
		From("members_lookup").
		Where(squirrel.Eq{"org_id": orgID}).
		ToSql()
	if err != nil {
		return time.Time{}, fmt.Errorf("build query: %w", err)
	}

	var createdAt time.Time
	if err := p.db.QueryRow(ctx, query, args...).Scan(&createdAt); err != nil {
		return time.Time{}, fmt.Errorf("query org created_at: %w", err)
	}

	return createdAt, nil
}

// ReviewLookupRow represents a review in the lookup table
type ReviewLookupRow struct {
	ID                 uuid.UUID
	OrderID            uuid.UUID
	ReviewerOrgID      uuid.UUID
	ReviewedOrgID      uuid.UUID
	Rating             int
	Comment            string
	OrderAmount        int64
	OrderCurrency      string
	OrderCreatedAt     time.Time
	OrderCompletedAt   time.Time
	RawWeight          float64
	FinalWeight        float64
	FraudScore         float64
	RequiresModeration bool
	Status             string
	ActivationDate     *time.Time
	CreatedAt          time.Time
	AnalyzedAt         *time.Time
	ModeratedAt        *time.Time
	ModeratedBy        *uuid.UUID
	ActivatedAt        *time.Time
}

// UpsertReviewLookup inserts or updates a review in the lookup table
func (p *FraudDataProjection) UpsertReviewLookup(ctx context.Context, row *ReviewLookupRow) error {
	query := `
		INSERT INTO reviews_lookup (
			id, order_id, reviewer_org_id, reviewed_org_id, rating, comment,
			order_amount, order_currency, order_created_at, order_completed_at,
			raw_weight, final_weight, fraud_score, requires_moderation, status,
			activation_date, created_at, analyzed_at, moderated_at, moderated_by, activated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21)
		ON CONFLICT (id) DO UPDATE SET
			raw_weight = EXCLUDED.raw_weight,
			final_weight = EXCLUDED.final_weight,
			fraud_score = EXCLUDED.fraud_score,
			requires_moderation = EXCLUDED.requires_moderation,
			status = EXCLUDED.status,
			activation_date = EXCLUDED.activation_date,
			analyzed_at = EXCLUDED.analyzed_at,
			moderated_at = EXCLUDED.moderated_at,
			moderated_by = EXCLUDED.moderated_by,
			activated_at = EXCLUDED.activated_at
	`

	if _, err := p.db.Exec(ctx, query,
		row.ID, row.OrderID, row.ReviewerOrgID, row.ReviewedOrgID, row.Rating, row.Comment,
		row.OrderAmount, row.OrderCurrency, row.OrderCreatedAt, row.OrderCompletedAt,
		row.RawWeight, row.FinalWeight, row.FraudScore, row.RequiresModeration, row.Status,
		row.ActivationDate, row.CreatedAt, row.AnalyzedAt, row.ModeratedAt, row.ModeratedBy, row.ActivatedAt,
	); err != nil {
		return fmt.Errorf("upsert review lookup: %w", err)
	}

	return nil
}

// InsertFraudSignal inserts a fraud signal for a review
func (p *FraudDataProjection) InsertFraudSignal(ctx context.Context, reviewID uuid.UUID, signalType, severity, description string, scoreImpact float64, evidence string) error {
	query := `
		INSERT INTO review_fraud_signals (review_id, signal_type, severity, description, score_impact, evidence)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	var evidenceJSON any = nil
	if evidence != "" {
		evidenceJSON = evidence
	}

	if _, err := p.db.Exec(ctx, query, reviewID, signalType, severity, description, scoreImpact, evidenceJSON); err != nil {
		return fmt.Errorf("insert fraud signal: %w", err)
	}

	return nil
}

// MarkFraudster marks organization as fraudster in org_reviewer_reputation
func (p *FraudDataProjection) MarkFraudster(ctx context.Context, orgID uuid.UUID, isConfirmed bool, markedBy uuid.UUID, reason string) error {
	query := `
		INSERT INTO org_reviewer_reputation (
			org_id, is_confirmed_fraudster, is_suspected_fraudster,
			fraudster_marked_at, fraudster_marked_by, fraudster_reason, updated_at
		) VALUES ($1, $2, $3, NOW(), $4, $5, NOW())
		ON CONFLICT (org_id) DO UPDATE SET
			is_confirmed_fraudster = $2,
			is_suspected_fraudster = $3,
			fraudster_marked_at = NOW(),
			fraudster_marked_by = $4,
			fraudster_reason = $5,
			updated_at = NOW()
	`

	if _, err := p.db.Exec(ctx, query, orgID, isConfirmed, !isConfirmed, markedBy, reason); err != nil {
		return fmt.Errorf("mark fraudster: %w", err)
	}
	return nil
}

// UnmarkFraudster clears fraudster status in org_reviewer_reputation
func (p *FraudDataProjection) UnmarkFraudster(ctx context.Context, orgID uuid.UUID) error {
	query := `
		UPDATE org_reviewer_reputation SET
			is_confirmed_fraudster = FALSE,
			is_suspected_fraudster = FALSE,
			fraudster_marked_at = NULL,
			fraudster_marked_by = NULL,
			fraudster_reason = NULL,
			updated_at = NOW()
		WHERE org_id = $1
	`

	if _, err := p.db.Exec(ctx, query, orgID); err != nil {
		return fmt.Errorf("unmark fraudster: %w", err)
	}
	return nil
}

// FraudsterInfo contains fraudster details for listing
type FraudsterInfo struct {
	OrgID              uuid.UUID
	OrgName            string
	IsConfirmed        bool
	MarkedAt           time.Time
	MarkedBy           uuid.UUID
	Reason             string
	TotalReviewsLeft   int
	DeactivatedReviews int
	ReputationScore    float64
}

// ListFraudsters returns all fraudsters with pagination
func (p *FraudDataProjection) ListFraudsters(ctx context.Context, limit, offset int) ([]FraudsterInfo, int, error) {
	// Count total
	countQuery := `
		SELECT COUNT(*) FROM org_reviewer_reputation
		WHERE is_confirmed_fraudster = TRUE OR is_suspected_fraudster = TRUE
	`
	var total int
	if err := p.db.QueryRow(ctx, countQuery).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count fraudsters: %w", err)
	}

	if total == 0 {
		return []FraudsterInfo{}, 0, nil
	}

	// Select with org name from organizations_lookup
	query := `
		SELECT
			r.org_id,
			COALESCE(o.name, '') as org_name,
			r.is_confirmed_fraudster,
			COALESCE(r.fraudster_marked_at, NOW()) as marked_at,
			COALESCE(r.fraudster_marked_by, '00000000-0000-0000-0000-000000000000'::uuid) as marked_by,
			COALESCE(r.fraudster_reason, '') as reason,
			r.total_reviews_left,
			r.deactivated_reviews,
			r.reputation_score
		FROM org_reviewer_reputation r
		LEFT JOIN organizations_lookup o ON o.id = r.org_id
		WHERE r.is_confirmed_fraudster = TRUE OR r.is_suspected_fraudster = TRUE
		ORDER BY r.fraudster_marked_at DESC NULLS LAST
		LIMIT $1 OFFSET $2
	`

	rows, err := p.db.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("query fraudsters: %w", err)
	}
	defer rows.Close()

	result := make([]FraudsterInfo, 0)
	for rows.Next() {
		var f FraudsterInfo
		if err := rows.Scan(
			&f.OrgID, &f.OrgName, &f.IsConfirmed,
			&f.MarkedAt, &f.MarkedBy, &f.Reason,
			&f.TotalReviewsLeft, &f.DeactivatedReviews, &f.ReputationScore,
		); err != nil {
			return nil, 0, fmt.Errorf("scan fraudster: %w", err)
		}
		result = append(result, f)
	}

	return result, total, nil
}

// ReviewTextInfo contains review text with metadata
type ReviewTextInfo struct {
	ID        uuid.UUID
	Comment   string
	CreatedAt time.Time
}

// GetRecentReviewTexts returns recent review texts from a reviewer organization
func (p *FraudDataProjection) GetRecentReviewTexts(ctx context.Context, reviewerOrgID uuid.UUID, limit int) ([]ReviewTextInfo, error) {
	query, args, err := p.psql.
		Select("id", "COALESCE(comment, '')", "created_at").
		From("reviews_lookup").
		Where(squirrel.And{
			squirrel.Eq{"reviewer_org_id": reviewerOrgID},
			squirrel.NotEq{"comment": ""},
		}).
		OrderBy("created_at DESC").
		Limit(uint64(limit)).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build query: %w", err)
	}

	rows, err := p.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query review texts: %w", err)
	}
	defer rows.Close()

	var result []ReviewTextInfo
	for rows.Next() {
		var r ReviewTextInfo
		if err := rows.Scan(&r.ID, &r.Comment, &r.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan review text: %w", err)
		}
		result = append(result, r)
	}

	return result, nil
}

// GetReviewTimings returns creation times of recent reviews from a reviewer
func (p *FraudDataProjection) GetReviewTimings(ctx context.Context, reviewerOrgID uuid.UUID, limit int) ([]time.Time, error) {
	query, args, err := p.psql.
		Select("created_at").
		From("reviews_lookup").
		Where(squirrel.Eq{"reviewer_org_id": reviewerOrgID}).
		OrderBy("created_at DESC").
		Limit(uint64(limit)).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build query: %w", err)
	}

	rows, err := p.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query review timings: %w", err)
	}
	defer rows.Close()

	var result []time.Time
	for rows.Next() {
		var t time.Time
		if err := rows.Scan(&t); err != nil {
			return nil, fmt.Errorf("scan timing: %w", err)
		}
		result = append(result, t)
	}

	return result, nil
}

// RatingPatternData contains rating statistics grouped by relationship type
type RatingPatternData struct {
	FriendsCount     int
	FriendsRatingSum int
	OthersCount      int
	OthersRatingSum  int
}

// GetRatingPatternsByRelationship returns avg rating to "friends" vs "others"
// Friends are orgs with whom reviewer has >= minMutualReviews mutual reviews
func (p *FraudDataProjection) GetRatingPatternsByRelationship(ctx context.Context, reviewerOrgID uuid.UUID, minMutualReviews int) (*RatingPatternData, error) {
	// Query to get reviews with mutual review count for each reviewed org
	query := `
		WITH reviewer_reviews AS (
			SELECT reviewed_org_id, rating
			FROM reviews_lookup
			WHERE reviewer_org_id = $1 AND status = 'active'
		),
		mutual_counts AS (
			SELECT
				reviewed_org_id,
				(SELECT COUNT(*) FROM reviews_lookup
				 WHERE reviewer_org_id = rr.reviewed_org_id
				   AND reviewed_org_id = $1
				   AND status = 'active') as mutual_count
			FROM reviewer_reviews rr
			GROUP BY reviewed_org_id
		)
		SELECT
			COALESCE(SUM(CASE WHEN mc.mutual_count >= $2 THEN 1 ELSE 0 END), 0) as friends_count,
			COALESCE(SUM(CASE WHEN mc.mutual_count >= $2 THEN rr.rating ELSE 0 END), 0) as friends_rating_sum,
			COALESCE(SUM(CASE WHEN mc.mutual_count < $2 THEN 1 ELSE 0 END), 0) as others_count,
			COALESCE(SUM(CASE WHEN mc.mutual_count < $2 THEN rr.rating ELSE 0 END), 0) as others_rating_sum
		FROM reviewer_reviews rr
		JOIN mutual_counts mc ON rr.reviewed_org_id = mc.reviewed_org_id
	`

	var data RatingPatternData
	if err := p.db.QueryRow(ctx, query, reviewerOrgID, minMutualReviews).Scan(
		&data.FriendsCount, &data.FriendsRatingSum,
		&data.OthersCount, &data.OthersRatingSum,
	); err != nil {
		return nil, fmt.Errorf("query rating patterns: %w", err)
	}

	return &data, nil
}

// BurstAfterLowData contains data for burst-after-low detection
type BurstAfterLowData struct {
	LastLowRatingAt  *time.Time
	FiveStarCountAfter int
}

// GetBurstAfterLowRating checks if org received burst of 5★ after a low rating
func (p *FraudDataProjection) GetBurstAfterLowRating(ctx context.Context, reviewedOrgID uuid.UUID, lowThreshold int, burstDays int) (*BurstAfterLowData, error) {
	// Find last low rating
	query := `
		WITH last_low AS (
			SELECT created_at
			FROM reviews_lookup
			WHERE reviewed_org_id = $1 AND rating <= $2 AND status = 'active'
			ORDER BY created_at DESC
			LIMIT 1
		)
		SELECT
			(SELECT created_at FROM last_low) as last_low_at,
			COALESCE(
				(SELECT COUNT(*) FROM reviews_lookup
				 WHERE reviewed_org_id = $1
				   AND rating = 5
				   AND status IN ('active', 'approved', 'pending_analysis', 'pending_moderation')
				   AND created_at > (SELECT created_at FROM last_low)
				   AND created_at <= (SELECT created_at FROM last_low) + ($3 || ' days')::interval
				), 0
			) as five_star_count
	`

	var data BurstAfterLowData
	if err := p.db.QueryRow(ctx, query, reviewedOrgID, lowThreshold, burstDays).Scan(
		&data.LastLowRatingAt, &data.FiveStarCountAfter,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return &BurstAfterLowData{}, nil
		}
		return nil, fmt.Errorf("query burst after low: %w", err)
	}

	return &data, nil
}

// OrgActivityData contains org's last activity data
type OrgActivityData struct {
	LastOrderAt      *time.Time
	LastReviewLeftAt *time.Time
	RecentReviewsCount int
}

// GetOrgLastActivity returns last activity time and recent review count
func (p *FraudDataProjection) GetOrgLastActivity(ctx context.Context, orgID uuid.UUID, recentDays int) (*OrgActivityData, error) {
	query := `
		SELECT
			(SELECT MAX(created_at) FROM orders_lookup
			 WHERE customer_org_id = $1 OR carrier_org_id = $1) as last_order,
			(SELECT MAX(created_at) FROM reviews_lookup
			 WHERE reviewer_org_id = $1) as last_review,
			(SELECT COUNT(*) FROM reviews_lookup
			 WHERE reviewer_org_id = $1
			   AND created_at > NOW() - ($2 || ' days')::interval) as recent_reviews
	`

	var data OrgActivityData
	if err := p.db.QueryRow(ctx, query, orgID, recentDays).Scan(
		&data.LastOrderAt, &data.LastReviewLeftAt, &data.RecentReviewsCount,
	); err != nil {
		return nil, fmt.Errorf("query org activity: %w", err)
	}

	return &data, nil
}
