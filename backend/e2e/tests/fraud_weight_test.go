package tests

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
	"github.com/udisondev/veziizi/backend/e2e/fixtures"
	"github.com/udisondev/veziizi/backend/e2e/helpers"
	reviewApp "github.com/udisondev/veziizi/backend/internal/application/review"
	"github.com/udisondev/veziizi/backend/internal/domain/review"
)

// FraudWeightSuite тестирует расчёт весов (weight) в системе фрод-детекции.
// Формула: RawWeight = OrderAmountWeight * OrgAgeWeight * DiversityWeight * ReputationWeight
type FraudWeightSuite struct {
	suite.Suite
	analyzer *reviewApp.Analyzer
}

func TestFraudWeightSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(FraudWeightSuite))
}

func (s *FraudWeightSuite) SetupSuite() {
	testSuite := getSuite(s.T())
	s.analyzer = testSuite.Factory.ReviewAnalyzer()
}

// insertMember создаёт запись в members_lookup для указанной организации.
// Без неё GetOrgCreatedAt вернёт ошибку, т.к. запрос делает SELECT MIN(created_at) FROM members_lookup.
func (s *FraudWeightSuite) insertMember(orgID uuid.UUID, createdAt time.Time) {
	s.T().Helper()

	ctx := context.Background()
	testSuite := getSuite(s.T())

	_, err := testSuite.Factory.DB().Exec(ctx,
		`INSERT INTO members_lookup (id, organization_id, email, password_hash, name, role, status, created_at)
		 VALUES ($1, $2, $3, 'hash', 'Test', 'owner', 'active', $4)`,
		uuid.New(), orgID, helpers.RandomEmail(), createdAt,
	)
	s.Require().NoError(err, "insert member lookup")
}

// newReview создаёт review-агрегат с нормальными таймингами (orderCreatedAt: -24ч, completedAt: -12ч),
// чтобы не срабатывал сигнал FastCompletion (порог < 2ч).
func (s *FraudWeightSuite) newReview(reviewerOrgID, reviewedOrgID uuid.UUID, orderAmount int64) *review.Review {
	return review.New(
		uuid.New(),
		uuid.New(),
		reviewerOrgID,
		reviewedOrgID,
		4,
		"normal delivery, no issues",
		orderAmount,
		"RUB",
		time.Now().Add(-24*time.Hour),
		time.Now().Add(-12*time.Hour),
	)
}

// TestFRW001_OrderAmountWeightBrackets проверяет зависимость веса от суммы заказа.
// Остальные компоненты (orgAge, diversity, reputation) нейтрализованы до 1.0,
// поэтому RawWeight = orderAmountWeight.
func (s *FraudWeightSuite) TestFRW001_OrderAmountWeightBrackets() {
	tests := []struct {
		name           string
		orderAmount    int64   // копейки
		expectedWeight float64 // ожидаемый orderAmountWeight
	}{
		{"100K RUB", 10_000_000, 1.0},
		{"50K RUB", 5_000_000, 0.9},
		{"10K RUB", 1_000_000, 0.7},
		{"1K RUB", 100_000, 0.5},
		{"50 RUB", 5_000, 0.3},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			ctx := context.Background()

			reviewerOrgID := uuid.New()
			reviewedOrgID := uuid.New()

			// Орг возрастом 13 месяцев -> orgAgeWeight = 1.0
			s.insertMember(reviewerOrgID, time.Now().AddDate(0, -13, 0))
			// reviewed org тоже нужен для GetOrgCreatedAt (checkNewOrgBurst)
			s.insertMember(reviewedOrgID, time.Now().AddDate(0, -13, 0))

			r := s.newReview(reviewerOrgID, reviewedOrgID, tt.orderAmount)

			result, err := s.analyzer.Analyze(ctx, r)
			s.Require().NoError(err, "Analyze не должен возвращать ошибку")

			s.Assert().InDelta(tt.expectedWeight, result.RawWeight, 0.01,
				"RawWeight для суммы %d копеек должен быть %.1f", tt.orderAmount, tt.expectedWeight)
		})
	}
}

// TestFRW002_OrgAgeWeightBrackets проверяет зависимость веса от возраста организации-рецензента.
// Сумма заказа 100K (weight=1.0), без предыдущих отзывов (diversity=1.0), без репутации (reputation=1.0).
// RawWeight = 1.0 * orgAgeWeight * 1.0 * 1.0 = orgAgeWeight.
func (s *FraudWeightSuite) TestFRW002_OrgAgeWeightBrackets() {
	tests := []struct {
		name           string
		ageMonths      int
		expectedWeight float64
	}{
		{"13 months", 13, 1.0},
		{"8 months", 8, 0.8},
		{"4 months", 4, 0.6},
		{"1 month", 1, 0.3},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			ctx := context.Background()

			reviewerOrgID := uuid.New()
			reviewedOrgID := uuid.New()

			s.insertMember(reviewerOrgID, time.Now().AddDate(0, -tt.ageMonths, 0))
			s.insertMember(reviewedOrgID, time.Now().AddDate(0, -13, 0))

			// 100K RUB -> orderAmountWeight = 1.0
			r := s.newReview(reviewerOrgID, reviewedOrgID, 10_000_000)

			result, err := s.analyzer.Analyze(ctx, r)
			s.Require().NoError(err, "Analyze не должен возвращать ошибку")

			s.Assert().InDelta(tt.expectedWeight, result.RawWeight, 0.01,
				"RawWeight для орг возраста %d мес должен быть %.1f", tt.ageMonths, tt.expectedWeight)
		})
	}
}

// TestFRW003_DiversityWeight проверяет зависимость веса от количества предыдущих отзывов
// от рецензента к рецензируемой организации.
// 0 предыдущих -> 1.0, 1 -> 0.5, 2+ -> 0.1
func (s *FraudWeightSuite) TestFRW003_DiversityWeight() {
	tests := []struct {
		name            string
		previousReviews int
		expectedWeight  float64
	}{
		{"no previous", 0, 1.0},
		{"1 previous", 1, 0.5},
		{"2 previous", 2, 0.1},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			ctx := context.Background()
			testSuite := getSuite(s.T())

			reviewerOrgID := uuid.New()
			reviewedOrgID := uuid.New()

			// orgAge 13 мес -> 1.0, orderAmount 100K -> 1.0, reputation -> 1.0
			s.insertMember(reviewerOrgID, time.Now().AddDate(0, -13, 0))
			s.insertMember(reviewedOrgID, time.Now().AddDate(0, -13, 0))

			// Вставляем предыдущие отзывы в reviews_lookup (статус pending_analysis,
			// не rejected/deactivated — считаются в GetPreviousReviewsFromReviewer)
			if tt.previousReviews > 0 {
				fixtures.InsertMultipleReviews(s.T(), testSuite.Factory, tt.previousReviews, fixtures.ReviewOpts{
					ReviewerOrgID: reviewerOrgID,
					ReviewedOrgID: reviewedOrgID,
					Rating:        4,
					Comment:       "previous review",
					Status:        "pending_analysis",
				})
			}

			r := s.newReview(reviewerOrgID, reviewedOrgID, 10_000_000)

			result, err := s.analyzer.Analyze(ctx, r)
			s.Require().NoError(err, "Analyze не должен возвращать ошибку")

			s.Assert().InDelta(tt.expectedWeight, result.RawWeight, 0.01,
				"RawWeight при %d предыдущих отзывах должен быть %.1f", tt.previousReviews, tt.expectedWeight)
		})
	}
}

// TestFRW004_ReputationWeight проверяет зависимость веса от репутации рецензента.
// confirmed fraudster -> 0.0, suspected -> 0.3, normal (score=1.0) -> 1.0.
func (s *FraudWeightSuite) TestFRW004_ReputationWeight() {
	tests := []struct {
		name           string
		opts           fixtures.ReputationOpts
		expectedWeight float64
	}{
		{
			"confirmed fraudster",
			fixtures.ReputationOpts{
				ReputationScore:      0.0,
				IsConfirmedFraudster: true,
			},
			0.0,
		},
		{
			"suspected fraudster",
			fixtures.ReputationOpts{
				ReputationScore:      0.5,
				IsSuspectedFraudster: true,
			},
			0.3,
		},
		{
			"normal reputation",
			fixtures.ReputationOpts{
				ReputationScore: 1.0,
			},
			1.0,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			ctx := context.Background()
			testSuite := getSuite(s.T())

			reviewerOrgID := uuid.New()
			reviewedOrgID := uuid.New()

			// orgAge 13 мес -> 1.0, orderAmount 100K -> 1.0, diversity -> 1.0
			s.insertMember(reviewerOrgID, time.Now().AddDate(0, -13, 0))
			s.insertMember(reviewedOrgID, time.Now().AddDate(0, -13, 0))

			fixtures.InsertOrgReputation(s.T(), testSuite.Factory, reviewerOrgID, tt.opts)

			r := s.newReview(reviewerOrgID, reviewedOrgID, 10_000_000)

			result, err := s.analyzer.Analyze(ctx, r)
			s.Require().NoError(err, "Analyze не должен возвращать ошибку")

			s.Assert().InDelta(tt.expectedWeight, result.RawWeight, 0.01,
				"RawWeight при репутации %q должен быть %.1f", tt.name, tt.expectedWeight)
		})
	}
}

// TestFRW005_CombinedWeightMultiplication проверяет мультипликативный расчёт:
// 50K (0.9) * 8 мес (0.8) * 2-й отзыв (0.5) * suspected fraudster (0.3) = 0.108
func (s *FraudWeightSuite) TestFRW005_CombinedWeightMultiplication() {
	ctx := context.Background()
	testSuite := getSuite(s.T())

	reviewerOrgID := uuid.New()
	reviewedOrgID := uuid.New()

	// orgAge 8 мес -> 0.8
	s.insertMember(reviewerOrgID, time.Now().AddDate(0, -8, 0))
	s.insertMember(reviewedOrgID, time.Now().AddDate(0, -13, 0))

	// 1 предыдущий отзыв -> diversity = 0.5
	fixtures.InsertMultipleReviews(s.T(), testSuite.Factory, 1, fixtures.ReviewOpts{
		ReviewerOrgID: reviewerOrgID,
		ReviewedOrgID: reviewedOrgID,
		Rating:        4,
		Comment:       "previous combined review",
		Status:        "active",
	})

	// suspected fraudster -> reputation = 0.3
	fixtures.InsertOrgReputation(s.T(), testSuite.Factory, reviewerOrgID, fixtures.ReputationOpts{
		ReputationScore:      0.5,
		IsSuspectedFraudster: true,
	})

	// 50K RUB -> orderAmountWeight = 0.9
	r := s.newReview(reviewerOrgID, reviewedOrgID, 5_000_000)

	result, err := s.analyzer.Analyze(ctx, r)
	s.Require().NoError(err, "Analyze не должен возвращать ошибку")

	// 0.9 * 0.8 * 0.5 * 0.3 = 0.108
	expected := 0.9 * 0.8 * 0.5 * 0.3
	s.Assert().InDelta(expected, result.RawWeight, 0.01,
		"RawWeight должен быть 0.9*0.8*0.5*0.3 = %.3f, получили %.3f", expected, result.RawWeight)
}
