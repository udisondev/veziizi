package tests

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"

	reviewApp "github.com/udisondev/veziizi/backend/internal/application/review"
	"github.com/udisondev/veziizi/backend/internal/domain/review"
	"github.com/udisondev/veziizi/backend/internal/domain/review/values"
	"github.com/udisondev/veziizi/backend/internal/pkg/factory"

	"github.com/udisondev/veziizi/backend/e2e/fixtures"
	"github.com/udisondev/veziizi/backend/e2e/helpers"
)

type FraudSignalsSuite struct {
	suite.Suite
	analyzer *reviewApp.Analyzer
	f        *factory.Factory
}

func TestFraudSignalsSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(FraudSignalsSuite))
}

func (s *FraudSignalsSuite) SetupSuite() {
	testSuite := getSuite(s.T())
	s.analyzer = testSuite.Factory.ReviewAnalyzer()
	s.f = testSuite.Factory
}

// insertMember вставляет запись в members_lookup для организации.
// Необходимо для GetOrgCreatedAt, который делает MIN(created_at) FROM members_lookup.
func (s *FraudSignalsSuite) insertMember(orgID uuid.UUID, createdAt time.Time) {
	s.T().Helper()
	_, err := s.f.DB().Exec(context.Background(),
		`INSERT INTO members_lookup (id, organization_id, email, password_hash, name, role, status, created_at)
		 VALUES ($1, $2, $3, 'hash', 'Test Member', 'owner', 'active', $4)`,
		uuid.New(), orgID, helpers.RandomEmail(), createdAt)
	s.Require().NoError(err)
}

// insertOldMembers вставляет мемберов для обеих организаций с created_at 2+ года назад,
// чтобы orgAge weight = 1.0 и NewOrgBurst не срабатывал.
func (s *FraudSignalsSuite) insertOldMembers(reviewerOrgID, reviewedOrgID uuid.UUID) {
	s.T().Helper()
	twoYearsAgo := time.Now().AddDate(-2, 0, 0)
	s.insertMember(reviewerOrgID, twoYearsAgo)
	s.insertMember(reviewedOrgID, twoYearsAgo)
}

// hasSignal проверяет наличие сигнала указанного типа в результате анализа.
func (s *FraudSignalsSuite) hasSignal(result *reviewApp.AnalysisResult, signalType string) bool {
	for _, sig := range result.FraudSignals {
		if sig.Type == signalType {
			return true
		}
	}
	return false
}

// newDefaultReview создаёт review с нормальными параметрами (без fraud-индикаторов).
func (s *FraudSignalsSuite) newDefaultReview(reviewerOrgID, reviewedOrgID uuid.UUID) *review.Review {
	s.T().Helper()
	return review.New(
		uuid.New(),
		uuid.New(),
		reviewerOrgID,
		reviewedOrgID,
		4,
		"normal review comment",
		10_000_000, // 100K RUB
		"RUB",
		time.Now().Add(-24*time.Hour),
		time.Now().Add(-12*time.Hour),
	)
}

// --- FRS001: Fast Completion ---

func (s *FraudSignalsSuite) TestFRS001_FastCompletion() {
	reviewerOrgID := uuid.New()
	reviewedOrgID := uuid.New()
	s.insertOldMembers(reviewerOrgID, reviewedOrgID)

	r := review.New(
		uuid.New(), uuid.New(),
		reviewerOrgID, reviewedOrgID,
		4, "fast order",
		10_000_000, "RUB",
		time.Now().Add(-2*time.Hour),  // создан 2 часа назад
		time.Now().Add(-1*time.Hour),  // завершён 1 час назад (duration = 1h < 2h)
	)

	result, err := s.analyzer.Analyze(context.Background(), r)
	s.Require().NoError(err)
	s.Assert().True(s.hasSignal(result, values.SignalFastCompletion.String()),
		"ожидался сигнал fast_completion при завершении за 1 час")
}

func (s *FraudSignalsSuite) TestFRS002_FastCompletion_NegativeCase() {
	reviewerOrgID := uuid.New()
	reviewedOrgID := uuid.New()
	s.insertOldMembers(reviewerOrgID, reviewedOrgID)

	r := review.New(
		uuid.New(), uuid.New(),
		reviewerOrgID, reviewedOrgID,
		4, "normal speed order",
		10_000_000, "RUB",
		time.Now().Add(-6*time.Hour),
		time.Now().Add(-3*time.Hour), // duration = 3h > 2h
	)

	result, err := s.analyzer.Analyze(context.Background(), r)
	s.Require().NoError(err)
	s.Assert().False(s.hasSignal(result, values.SignalFastCompletion.String()),
		"не должно быть сигнала fast_completion при завершении за 3 часа")
}

// --- FRS003: Mutual Reviews ---

func (s *FraudSignalsSuite) TestFRS003_MutualReviews() {
	reviewerOrgID := uuid.New()
	reviewedOrgID := uuid.New()
	s.insertOldMembers(reviewerOrgID, reviewedOrgID)

	now := time.Now()

	// 3 отзыва A→B
	fixtures.InsertMultipleReviews(s.T(), s.f, 3, fixtures.ReviewOpts{
		ReviewerOrgID: reviewerOrgID,
		ReviewedOrgID: reviewedOrgID,
		Rating:        4,
		Status:        "active",
		CreatedAt:     now.AddDate(0, 0, -5),
	})

	// 3 отзыва B→A
	fixtures.InsertMultipleReviews(s.T(), s.f, 3, fixtures.ReviewOpts{
		ReviewerOrgID: reviewedOrgID,
		ReviewedOrgID: reviewerOrgID,
		Rating:        4,
		Status:        "active",
		CreatedAt:     now.AddDate(0, 0, -3),
	})

	r := s.newDefaultReview(reviewerOrgID, reviewedOrgID)

	result, err := s.analyzer.Analyze(context.Background(), r)
	s.Require().NoError(err)
	s.Assert().True(s.hasSignal(result, values.SignalMutualReviews.String()),
		"ожидался сигнал mutual_reviews при 6 взаимных отзывах за месяц (порог >5)")
}

// --- FRS004: Perfect Ratings ---

func (s *FraudSignalsSuite) TestFRS004_PerfectRatings() {
	reviewerOrgID := uuid.New()
	reviewedOrgID := uuid.New()
	s.insertOldMembers(reviewerOrgID, reviewedOrgID)

	// 3 предыдущих отзыва с рейтингом 5 (status != rejected/deactivated)
	fixtures.InsertMultipleReviews(s.T(), s.f, 3, fixtures.ReviewOpts{
		ReviewerOrgID: reviewerOrgID,
		ReviewedOrgID: reviewedOrgID,
		Rating:        5,
		Status:        "pending_analysis",
	})

	// Анализируем 4-й отзыв с рейтингом 5 → итого count=4 > 3, avg=5.0
	r := review.New(
		uuid.New(), uuid.New(),
		reviewerOrgID, reviewedOrgID,
		5, "perfect again",
		10_000_000, "RUB",
		time.Now().Add(-24*time.Hour),
		time.Now().Add(-12*time.Hour),
	)

	result, err := s.analyzer.Analyze(context.Background(), r)
	s.Require().NoError(err)
	s.Assert().True(s.hasSignal(result, values.SignalPerfectRatings.String()),
		"ожидался сигнал perfect_ratings при 4 отзывах по 5 звёзд")
}

// --- FRS005, FRS006: New Org Burst ---

func (s *FraudSignalsSuite) TestFRS005_NewOrgBurst() {
	reviewerOrgID := uuid.New()
	reviewedOrgID := uuid.New()

	twoYearsAgo := time.Now().AddDate(-2, 0, 0)
	s.insertMember(reviewerOrgID, twoYearsAgo)

	// reviewed org создана 3 дня назад (< 7 дней)
	threeDaysAgo := time.Now().AddDate(0, 0, -3)
	s.insertMember(reviewedOrgID, threeDaysAgo)

	// 11 отзывов в адрес reviewed org (порог >10)
	fixtures.InsertMultipleReviews(s.T(), s.f, 11, fixtures.ReviewOpts{
		ReviewerOrgID: uuid.New(), // от случайных org
		ReviewedOrgID: reviewedOrgID,
		Rating:        4,
		Status:        "active",
		CreatedAt:     time.Now().AddDate(0, 0, -1),
	})

	r := s.newDefaultReview(reviewerOrgID, reviewedOrgID)

	result, err := s.analyzer.Analyze(context.Background(), r)
	s.Require().NoError(err)
	s.Assert().True(s.hasSignal(result, values.SignalNewOrgBurst.String()),
		"ожидался сигнал new_org_burst для новой org с 11 отзывами за первую неделю")
}

func (s *FraudSignalsSuite) TestFRS006_NewOrgBurst_OldOrg() {
	reviewerOrgID := uuid.New()
	reviewedOrgID := uuid.New()

	twoYearsAgo := time.Now().AddDate(-2, 0, 0)
	s.insertMember(reviewerOrgID, twoYearsAgo)

	// reviewed org создана 30 дней назад (> 7 дней)
	thirtyDaysAgo := time.Now().AddDate(0, 0, -30)
	s.insertMember(reviewedOrgID, thirtyDaysAgo)

	// 11 отзывов в адрес reviewed org
	fixtures.InsertMultipleReviews(s.T(), s.f, 11, fixtures.ReviewOpts{
		ReviewerOrgID: uuid.New(),
		ReviewedOrgID: reviewedOrgID,
		Rating:        4,
		Status:        "active",
		CreatedAt:     time.Now().AddDate(0, 0, -1),
	})

	r := s.newDefaultReview(reviewerOrgID, reviewedOrgID)

	result, err := s.analyzer.Analyze(context.Background(), r)
	s.Require().NoError(err)
	s.Assert().False(s.hasSignal(result, values.SignalNewOrgBurst.String()),
		"не должно быть сигнала new_org_burst для org старше 7 дней")
}

// --- FRS007: Same IP ---

func (s *FraudSignalsSuite) TestFRS007_SameIP() {
	reviewerOrgID := uuid.New()
	reviewedOrgID := uuid.New()
	s.insertOldMembers(reviewerOrgID, reviewedOrgID)

	sharedIP := "192.168.1.100"
	fixtures.SetMemberMetadata(s.T(), s.f, reviewerOrgID, sharedIP, "fp-reviewer-007")
	fixtures.SetMemberMetadata(s.T(), s.f, reviewedOrgID, sharedIP, "fp-reviewed-007")

	r := s.newDefaultReview(reviewerOrgID, reviewedOrgID)

	result, err := s.analyzer.Analyze(context.Background(), r)
	s.Require().NoError(err)
	s.Assert().True(s.hasSignal(result, values.SignalSameIP.String()),
		"ожидался сигнал same_ip при одинаковом IP у обеих организаций")
}

// --- FRS008: Same Fingerprint ---

func (s *FraudSignalsSuite) TestFRS008_SameFingerprint() {
	reviewerOrgID := uuid.New()
	reviewedOrgID := uuid.New()
	s.insertOldMembers(reviewerOrgID, reviewedOrgID)

	sharedFP := "shared-fingerprint-abc123"
	fixtures.SetMemberMetadata(s.T(), s.f, reviewerOrgID, "10.0.0.1", sharedFP)
	fixtures.SetMemberMetadata(s.T(), s.f, reviewedOrgID, "10.0.0.2", sharedFP)

	r := s.newDefaultReview(reviewerOrgID, reviewedOrgID)

	result, err := s.analyzer.Analyze(context.Background(), r)
	s.Require().NoError(err)
	s.Assert().True(s.hasSignal(result, values.SignalSameFingerprint.String()),
		"ожидался сигнал same_fingerprint при одинаковом fingerprint")
}

// --- FRS009, FRS010: Timing Pattern ---

func (s *FraudSignalsSuite) TestFRS009_TimingPattern() {
	reviewerOrgID := uuid.New()
	reviewedOrgID := uuid.New()
	s.insertOldMembers(reviewerOrgID, reviewedOrgID)

	// 10 отзывов, все созданы между 14:00-15:59 UTC (90%+ в 2-часовом окне)
	for i := range 10 {
		createdAt := time.Date(2026, 3, 20-i, 14, 30, 0, 0, time.UTC)
		fixtures.InsertReviewWithTiming(s.T(), s.f, reviewerOrgID, createdAt)
	}

	r := s.newDefaultReview(reviewerOrgID, reviewedOrgID)

	result, err := s.analyzer.Analyze(context.Background(), r)
	s.Require().NoError(err)
	s.Assert().True(s.hasSignal(result, values.SignalTimingPattern.String()),
		"ожидался сигнал review_timing_pattern при 100%% отзывов в 2-часовом окне")
}

func (s *FraudSignalsSuite) TestFRS010_TimingPattern_NegativeCase() {
	reviewerOrgID := uuid.New()
	reviewedOrgID := uuid.New()
	s.insertOldMembers(reviewerOrgID, reviewedOrgID)

	// 10 отзывов, разбросанных по разным часам
	hours := []int{2, 5, 8, 11, 14, 17, 20, 23, 3, 7}
	for i, h := range hours {
		createdAt := time.Date(2026, 3, 20-i, h, 0, 0, 0, time.UTC)
		fixtures.InsertReviewWithTiming(s.T(), s.f, reviewerOrgID, createdAt)
	}

	r := s.newDefaultReview(reviewerOrgID, reviewedOrgID)

	result, err := s.analyzer.Analyze(context.Background(), r)
	s.Require().NoError(err)
	s.Assert().False(s.hasSignal(result, values.SignalTimingPattern.String()),
		"не должно быть сигнала review_timing_pattern при равномерном распределении по часам")
}

// --- FRS011: Dormant Reviewer ---

func (s *FraudSignalsSuite) TestFRS011_DormantReviewer() {
	ctx := context.Background()
	reviewerOrgID := uuid.New()
	reviewedOrgID := uuid.New()
	s.insertOldMembers(reviewerOrgID, reviewedOrgID)

	// 1. Старый отзыв (120 дней назад) — last activity, исключая последние 7 дней
	row := fixtures.DefaultReviewLookupRow(reviewerOrgID, uuid.New())
	row.CreatedAt = time.Now().AddDate(0, 0, -120)
	fixtures.InsertReviewLookup(s.T(), s.f, row)

	// 2. Старый заказ (120 дней назад) — ещё один индикатор активности
	_, err := s.f.DB().Exec(ctx,
		`INSERT INTO freight_requests_lookup
			(id, customer_org_id, status, created_at, expires_at, request_number)
		 VALUES ($1, $2, 'completed', $3, $3, 0)`,
		uuid.New(), reviewerOrgID, time.Now().AddDate(0, 0, -120))
	s.Require().NoError(err)

	// 3. 6 недавних отзывов (за последние 7 дней) — burst после спячки (порог >5)
	fixtures.InsertMultipleReviews(s.T(), s.f, 6, fixtures.ReviewOpts{
		ReviewerOrgID: reviewerOrgID,
		ReviewedOrgID: uuid.New(),
		Rating:        5,
		Status:        "active",
		CreatedAt:     time.Now().AddDate(0, 0, -2),
	})

	r := s.newDefaultReview(reviewerOrgID, reviewedOrgID)

	result, err := s.analyzer.Analyze(ctx, r)
	s.Require().NoError(err)
	s.Assert().True(s.hasSignal(result, values.SignalDormantReviewer.String()),
		"ожидался сигнал dormant_reviewer при 120 днях неактивности и 6 отзывах за неделю")
}

// --- FRS012: Burst After Low Rating ---

func (s *FraudSignalsSuite) TestFRS012_BurstAfterLow() {
	reviewerOrgID := uuid.New()
	reviewedOrgID := uuid.New()
	s.insertOldMembers(reviewerOrgID, reviewedOrgID)

	// 1 активный отзыв с низким рейтингом (<=2), 5 дней назад
	lowRow := fixtures.DefaultReviewLookupRow(uuid.New(), reviewedOrgID)
	lowRow.Rating = 1
	lowRow.Status = "active"
	lowRow.CreatedAt = time.Now().AddDate(0, 0, -5)
	fixtures.InsertReviewLookup(s.T(), s.f, lowRow)

	// 5 пятизвёздочных отзывов, 3 дня назад (в пределах 7 дней после low)
	fixtures.InsertMultipleReviews(s.T(), s.f, 5, fixtures.ReviewOpts{
		ReviewerOrgID: uuid.New(),
		ReviewedOrgID: reviewedOrgID,
		Rating:        5,
		Status:        "pending_analysis",
		CreatedAt:     time.Now().AddDate(0, 0, -3),
	})

	r := s.newDefaultReview(reviewerOrgID, reviewedOrgID)

	result, err := s.analyzer.Analyze(context.Background(), r)
	s.Require().NoError(err)
	s.Assert().True(s.hasSignal(result, values.SignalBurstAfterLow.String()),
		"ожидался сигнал burst_after_low_rating при 5 пятизвёздочных отзывах после низкой оценки")
}

// --- FRS013: Rating Manipulation ---

func (s *FraudSignalsSuite) TestFRS013_RatingManipulation() {
	reviewerOrgID := uuid.New()
	reviewedOrgID := uuid.New()
	s.insertOldMembers(reviewerOrgID, reviewedOrgID)

	friendOrg1 := uuid.New()
	friendOrg2 := uuid.New()
	otherOrg1 := uuid.New()
	otherOrg2 := uuid.New()

	// "Друзья": reviewer имеет 3+ взаимных отзыва с каждым, средний рейтинг = 5.0
	for _, friendOrgID := range []uuid.UUID{friendOrg1, friendOrg2} {
		// reviewer → friend: 3 отзыва по 5 звёзд
		fixtures.InsertMultipleReviews(s.T(), s.f, 3, fixtures.ReviewOpts{
			ReviewerOrgID: reviewerOrgID,
			ReviewedOrgID: friendOrgID,
			Rating:        5,
			Status:        "active",
		})
		// friend → reviewer: 3 отзыва (для выполнения mutual count >= 3)
		fixtures.InsertMultipleReviews(s.T(), s.f, 3, fixtures.ReviewOpts{
			ReviewerOrgID: friendOrgID,
			ReviewedOrgID: reviewerOrgID,
			Rating:        4,
			Status:        "active",
		})
	}

	// "Другие": reviewer имеет <3 взаимных отзыва, средний рейтинг = 2.0
	for _, otherOrgID := range []uuid.UUID{otherOrg1, otherOrg2} {
		// reviewer → other: 2 отзыва по 2 звезды (mutual count < 3)
		fixtures.InsertMultipleReviews(s.T(), s.f, 2, fixtures.ReviewOpts{
			ReviewerOrgID: reviewerOrgID,
			ReviewedOrgID: otherOrgID,
			Rating:        2,
			Status:        "active",
		})
	}

	r := s.newDefaultReview(reviewerOrgID, reviewedOrgID)

	result, err := s.analyzer.Analyze(context.Background(), r)
	s.Require().NoError(err)
	s.Assert().True(s.hasSignal(result, values.SignalRatingManipulation.String()),
		"ожидался сигнал rating_manipulation: avg друзья=5.0 (>=4.5), avg другие=2.0 (<=2.5)")
}

// --- FRS014, FRS015: Text Similarity ---

func (s *FraudSignalsSuite) TestFRS014_TextSimilarity_Latin() {
	reviewerOrgID := uuid.New()
	reviewedOrgID := uuid.New()
	s.insertOldMembers(reviewerOrgID, reviewedOrgID)

	// 3 предыдущих отзыва с почти идентичным текстом (>80% similarity)
	comments := []string{
		"Great service, fast delivery and good price",
		"Great service fast delivery and good price",
		"Great service, fast delivery and good price!",
	}
	for _, comment := range comments {
		row := fixtures.DefaultReviewLookupRow(reviewerOrgID, uuid.New())
		row.Comment = comment
		fixtures.InsertReviewLookup(s.T(), s.f, row)
	}

	// Анализируем отзыв с аналогичным текстом
	r := review.New(
		uuid.New(), uuid.New(),
		reviewerOrgID, reviewedOrgID,
		4, "Great service, fast delivery and good price",
		10_000_000, "RUB",
		time.Now().Add(-24*time.Hour),
		time.Now().Add(-12*time.Hour),
	)

	result, err := s.analyzer.Analyze(context.Background(), r)
	s.Require().NoError(err)
	s.Assert().True(s.hasSignal(result, values.SignalTextSimilarity.String()),
		"ожидался сигнал review_text_similarity при идентичных латинских текстах")
}

func (s *FraudSignalsSuite) TestFRS015_TextSimilarity_Cyrillic() {
	reviewerOrgID := uuid.New()
	reviewedOrgID := uuid.New()
	s.insertOldMembers(reviewerOrgID, reviewedOrgID)

	// 3 предыдущих отзыва с почти идентичным кириллическим текстом
	comments := []string{
		"Отличный сервис, быстрая доставка",
		"Отличный сервис быстрая доставка",
		"Отличный сервис, быстрая доставка!",
	}
	for _, comment := range comments {
		row := fixtures.DefaultReviewLookupRow(reviewerOrgID, uuid.New())
		row.Comment = comment
		fixtures.InsertReviewLookup(s.T(), s.f, row)
	}

	// Анализируем отзыв с аналогичным кириллическим текстом
	r := review.New(
		uuid.New(), uuid.New(),
		reviewerOrgID, reviewedOrgID,
		4, "Отличный сервис, быстрая доставка",
		10_000_000, "RUB",
		time.Now().Add(-24*time.Hour),
		time.Now().Add(-12*time.Hour),
	)

	result, err := s.analyzer.Analyze(context.Background(), r)
	s.Require().NoError(err)
	s.Assert().True(s.hasSignal(result, values.SignalTextSimilarity.String()),
		"ожидался сигнал review_text_similarity при идентичных кириллических текстах (rune-based Levenshtein)")
}

// --- FRS016: Clean Review (negative case for all signals) ---

func (s *FraudSignalsSuite) TestFRS016_CleanReview() {
	reviewerOrgID := uuid.New()
	reviewedOrgID := uuid.New()
	s.insertOldMembers(reviewerOrgID, reviewedOrgID)

	// Нормальный отзыв: completion time = 12h, 100K RUB, org старше 12 месяцев,
	// первый отзыв в адрес reviewed org, никаких fraud-индикаторов.
	r := review.New(
		uuid.New(), uuid.New(),
		reviewerOrgID, reviewedOrgID,
		4, "unique review text that is not similar to anything",
		10_000_000, "RUB",
		time.Now().Add(-24*time.Hour),
		time.Now().Add(-12*time.Hour),
	)

	result, err := s.analyzer.Analyze(context.Background(), r)
	s.Require().NoError(err)
	s.Assert().Empty(result.FraudSignals,
		"не должно быть fraud сигналов для чистого отзыва")
	s.Assert().InDelta(0.0, result.FraudScore, 0.001,
		"fraud score должен быть 0 для чистого отзыва")
}

