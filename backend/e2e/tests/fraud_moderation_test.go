package tests

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
	reviewApp "github.com/udisondev/veziizi/backend/internal/application/review"
	"github.com/udisondev/veziizi/backend/internal/domain/review"
	"github.com/udisondev/veziizi/backend/internal/domain/review/events"
	"github.com/udisondev/veziizi/backend/internal/domain/review/values"
	"github.com/udisondev/veziizi/backend/internal/pkg/factory"
	"github.com/udisondev/veziizi/backend/e2e/fixtures"
	"github.com/udisondev/veziizi/backend/e2e/helpers"
)

// FraudModerationSuite тестирует расчёт fraud score, пороги модерации и задержки активации.
type FraudModerationSuite struct {
	suite.Suite
	analyzer *reviewApp.Analyzer
	f        *factory.Factory
}

func TestFraudModerationSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(FraudModerationSuite))
}

func (s *FraudModerationSuite) SetupSuite() {
	testSuite := getSuite(s.T())
	s.f = testSuite.Factory
	s.analyzer = s.f.ReviewAnalyzer()
}

// insertMember создаёт запись в members_lookup для указанной организации.
func (s *FraudModerationSuite) insertMember(orgID uuid.UUID, createdAt time.Time) {
	s.T().Helper()

	ctx := context.Background()
	_, err := s.f.DB().Exec(ctx,
		`INSERT INTO members_lookup (id, organization_id, email, password_hash, name, role, status, created_at)
		 VALUES ($1, $2, $3, 'hash', 'Test', 'owner', 'active', $4)`,
		uuid.New(), orgID, helpers.RandomEmail(), createdAt,
	)
	s.Require().NoError(err, "insert member lookup")
}

// newReview создаёт review-агрегат с нормальными таймингами (orderCreatedAt: -24ч, completedAt: -12ч),
// чтобы не срабатывал сигнал FastCompletion (порог < 2ч).
func (s *FraudModerationSuite) newReview(reviewerOrgID, reviewedOrgID uuid.UUID) *review.Review {
	return review.New(
		uuid.New(),
		uuid.New(),
		reviewerOrgID,
		reviewedOrgID,
		4,
		"normal delivery, no issues",
		10_000_000, // 100K RUB
		"RUB",
		time.Now().Add(-24*time.Hour),
		time.Now().Add(-12*time.Hour),
	)
}

// newFastReview создаёт review с быстрым выполнением заказа (1ч < 2ч порога).
func (s *FraudModerationSuite) newFastReview(reviewerOrgID, reviewedOrgID uuid.UUID) *review.Review {
	return review.New(
		uuid.New(),
		uuid.New(),
		reviewerOrgID,
		reviewedOrgID,
		4,
		"fast delivery",
		10_000_000,
		"RUB",
		time.Now().Add(-2*time.Hour),
		time.Now().Add(-1*time.Hour), // 1 час — FastCompletion сработает
	)
}

// TestFRM001_ScoreSumAndCap проверяет, что fraud score = сумма ScoreImpact сигналов
// и что при превышении 1.0 значение ограничивается (capped).
// SameIP (0.5) + SameFingerprint (0.5) = 1.0 — ровно на границе.
func (s *FraudModerationSuite) TestFRM001_ScoreSumAndCap() {
	ctx := context.Background()

	reviewerOrgID := uuid.New()
	reviewedOrgID := uuid.New()

	// Орг возрастом 13 месяцев -> orgAgeWeight = 1.0
	s.insertMember(reviewerOrgID, time.Now().AddDate(0, -13, 0))
	s.insertMember(reviewedOrgID, time.Now().AddDate(0, -13, 0))

	// Одинаковый IP и fingerprint для обеих организаций
	sharedIP := "192.168.1.100"
	sharedFP := "fp-same-device-abc123"
	fixtures.SetMemberMetadata(s.T(), s.f, reviewerOrgID, sharedIP, sharedFP)
	fixtures.SetMemberMetadata(s.T(), s.f, reviewedOrgID, sharedIP, sharedFP)

	r := s.newReview(reviewerOrgID, reviewedOrgID)

	result, err := s.analyzer.Analyze(ctx, r)
	s.Require().NoError(err)

	// SameIP (0.5) + SameFingerprint (0.5) = 1.0, cap = 1.0
	s.Assert().InDelta(1.0, result.FraudScore, 0.01,
		"fraud score должен быть SameIP(0.5) + SameFingerprint(0.5) = 1.0")

	// Проверяем наличие обоих сигналов
	signalTypes := make(map[string]bool)
	for _, sig := range result.FraudSignals {
		signalTypes[sig.Type] = true
	}
	s.Assert().True(signalTypes[values.SignalSameIP.String()], "должен быть сигнал SameIP")
	s.Assert().True(signalTypes[values.SignalSameFingerprint.String()], "должен быть сигнал SameFingerprint")
}

// TestFRM002_ModerationThreshold проверяет, что fraud_score >= 0.3 требует модерации.
// FastCompletion (0.2) + BurstAfterLow (0.25) = 0.45 >= 0.3 → RequiresModeration = true.
func (s *FraudModerationSuite) TestFRM002_ModerationThreshold() {
	ctx := context.Background()

	reviewerOrgID := uuid.New()
	reviewedOrgID := uuid.New()

	// orgAge 13 мес -> weight 1.0
	s.insertMember(reviewerOrgID, time.Now().AddDate(0, -13, 0))
	s.insertMember(reviewedOrgID, time.Now().AddDate(0, -13, 0))

	// Вставляем низкую оценку, чтобы сработал BurstAfterLow
	testSuite := getSuite(s.T())
	fixtures.InsertMultipleReviews(s.T(), testSuite.Factory, 1, fixtures.ReviewOpts{
		ReviewerOrgID: uuid.New(), // другой рецензент
		ReviewedOrgID: reviewedOrgID,
		Rating:        1, // низкая оценка
		Comment:       "terrible service",
		Status:        "active",
		CreatedAt:     time.Now().Add(-48 * time.Hour),
	})
	// 5 пятизвёздочных отзывов после низкой оценки (порог = 5)
	fixtures.InsertMultipleReviews(s.T(), testSuite.Factory, 5, fixtures.ReviewOpts{
		ReviewerOrgID: uuid.New(),
		ReviewedOrgID: reviewedOrgID,
		Rating:        5,
		Comment:       "great",
		Status:        "active",
		CreatedAt:     time.Now().Add(-24 * time.Hour),
	})

	// FastCompletion: заказ выполнен за 1 час (< 2ч порога)
	r := s.newFastReview(reviewerOrgID, reviewedOrgID)

	result, err := s.analyzer.Analyze(ctx, r)
	s.Require().NoError(err)

	// FastCompletion (0.2) + BurstAfterLow (0.25) = 0.45
	s.Assert().GreaterOrEqual(result.FraudScore, values.FraudModerationScoreThreshold,
		"fraud score %.2f должен быть >= порога модерации %.2f",
		result.FraudScore, values.FraudModerationScoreThreshold)
	s.Assert().True(result.RequiresModeration, "должна требоваться модерация")
}

// TestFRM003_BelowModeration проверяет, что fraud_score < 0.3 не требует модерации.
// Только FastCompletion (0.2) < 0.3 → RequiresModeration = false.
func (s *FraudModerationSuite) TestFRM003_BelowModeration() {
	ctx := context.Background()

	reviewerOrgID := uuid.New()
	reviewedOrgID := uuid.New()

	s.insertMember(reviewerOrgID, time.Now().AddDate(0, -13, 0))
	s.insertMember(reviewedOrgID, time.Now().AddDate(0, -13, 0))

	// FastCompletion: заказ за 1 час
	r := s.newFastReview(reviewerOrgID, reviewedOrgID)

	result, err := s.analyzer.Analyze(ctx, r)
	s.Require().NoError(err)

	// FastCompletion = 0.2, ниже порога 0.3
	s.Assert().Less(result.FraudScore, values.FraudModerationScoreThreshold,
		"fraud score %.2f должен быть < порога модерации %.2f",
		result.FraudScore, values.FraudModerationScoreThreshold)
	s.Assert().False(result.RequiresModeration, "модерация не должна требоваться")
}

// TestFRM004_SuspiciousDelay проверяет, что подозрительный отзыв (score > 0.1)
// получает задержку активации ~14 дней.
func (s *FraudModerationSuite) TestFRM004_SuspiciousDelay() {
	ctx := context.Background()

	reviewerOrgID := uuid.New()
	reviewedOrgID := uuid.New()

	s.insertMember(reviewerOrgID, time.Now().AddDate(0, -13, 0))
	s.insertMember(reviewedOrgID, time.Now().AddDate(0, -13, 0))

	// PerfectRatings (0.15) > 0.1: нужны >3 пятёрок от одного рецензента
	testSuite := getSuite(s.T())
	fixtures.InsertMultipleReviews(s.T(), testSuite.Factory, 3, fixtures.ReviewOpts{
		ReviewerOrgID: reviewerOrgID,
		ReviewedOrgID: reviewedOrgID,
		Rating:        5,
		Comment:       "excellent",
		Status:        "active",
	})

	// Текущий отзыв тоже 5 звёзд — итого 4 пятёрки → PerfectRatings сработает
	r := review.New(
		uuid.New(), uuid.New(),
		reviewerOrgID, reviewedOrgID,
		5, "perfect again",
		10_000_000, "RUB",
		time.Now().Add(-24*time.Hour),
		time.Now().Add(-12*time.Hour),
	)

	result, err := s.analyzer.Analyze(ctx, r)
	s.Require().NoError(err)

	s.Assert().Greater(result.FraudScore, 0.1,
		"fraud score должен быть > 0.1 для suspicious delay")

	// Ожидаемая задержка: ~14 дней от сейчас
	expectedDate := time.Now().AddDate(0, 0, values.FraudSuspiciousDelayDays)
	s.Assert().WithinDuration(expectedDate, result.ActivationDate, 24*time.Hour,
		"activation date должна быть ~%d дней от сейчас", values.FraudSuspiciousDelayDays)
}

// TestFRM005_NormalDelay проверяет, что чистый отзыв (score = 0)
// получает задержку активации ~7 дней.
func (s *FraudModerationSuite) TestFRM005_NormalDelay() {
	ctx := context.Background()

	reviewerOrgID := uuid.New()
	reviewedOrgID := uuid.New()

	// orgAge 13 мес -> weight 1.0, нет предыдущих отзывов, нет фрод-сигналов
	s.insertMember(reviewerOrgID, time.Now().AddDate(0, -13, 0))
	s.insertMember(reviewedOrgID, time.Now().AddDate(0, -13, 0))

	r := s.newReview(reviewerOrgID, reviewedOrgID)

	result, err := s.analyzer.Analyze(ctx, r)
	s.Require().NoError(err)

	s.Assert().InDelta(0.0, result.FraudScore, 0.001, "fraud score должен быть 0 для чистого отзыва")
	s.Assert().False(result.RequiresModeration, "модерация не требуется")

	expectedDate := time.Now().AddDate(0, 0, values.FraudActivationDelayDays)
	s.Assert().WithinDuration(expectedDate, result.ActivationDate, 24*time.Hour,
		"activation date должна быть ~%d дней от сейчас", values.FraudActivationDelayDays)
}

// TestFRM006_AutoApproval проверяет, что RecordAnalysis с requiresModeration=false
// автоматически переводит отзыв в статус approved с FinalWeight = RawWeight.
func (s *FraudModerationSuite) TestFRM006_AutoApproval() {
	reviewerOrgID := uuid.New()
	reviewedOrgID := uuid.New()

	r := review.New(
		uuid.New(), uuid.New(),
		reviewerOrgID, reviewedOrgID,
		4, "good service",
		10_000_000, "RUB",
		time.Now().Add(-24*time.Hour),
		time.Now().Add(-12*time.Hour),
	)

	rawWeight := 0.85
	activationDate := time.Now().AddDate(0, 0, 7)

	err := r.RecordAnalysis(rawWeight, nil, 0.0, false, activationDate)
	s.Require().NoError(err)

	s.Assert().Equal(values.StatusApproved, r.Status(),
		"статус должен быть approved после auto-approval")
	s.Assert().InDelta(rawWeight, r.FinalWeight(), 0.001,
		"FinalWeight должен совпадать с RawWeight при auto-approval")
	s.Assert().NotNil(r.ActivationDate(), "activation date должна быть установлена")

	// Проверяем, что сгенерированы оба события: ReviewAnalyzed + ReviewApproved
	changes := r.Changes()
	s.Assert().GreaterOrEqual(len(changes), 3, "должны быть ReviewReceived + ReviewAnalyzed + ReviewApproved")

	var hasAnalyzed, hasApproved bool
	for _, evt := range changes {
		switch evt.(type) {
		case events.ReviewAnalyzed:
			hasAnalyzed = true
		case events.ReviewApproved:
			hasApproved = true
		}
	}
	s.Assert().True(hasAnalyzed, "должно быть событие ReviewAnalyzed")
	s.Assert().True(hasApproved, "должно быть событие ReviewApproved")
}
