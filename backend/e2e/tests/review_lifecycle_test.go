package tests

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
	"github.com/udisondev/veziizi/backend/e2e/helpers"
	reviewApp "github.com/udisondev/veziizi/backend/internal/application/review"
	"github.com/udisondev/veziizi/backend/internal/domain/review"
	"github.com/udisondev/veziizi/backend/internal/domain/review/events"
	"github.com/udisondev/veziizi/backend/internal/domain/review/values"
	"github.com/udisondev/veziizi/backend/internal/infrastructure/projections"
	"github.com/udisondev/veziizi/backend/internal/pkg/factory"
)

// ReviewLifecycleSuite тестирует полный жизненный цикл отзыва:
// создание → анализ → (модерация) → approve/reject → активация → влияние на рейтинг.
// Тесты опираются на event handler'ы, зарегистрированные в shared suite
// (ReviewReceiverHandler, ReviewAnalyzerHandler, ReviewsProjectionHandler).
type ReviewLifecycleSuite struct {
	suite.Suite
	f *factory.Factory
}

func TestReviewLifecycleSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(ReviewLifecycleSuite))
}

func (s *ReviewLifecycleSuite) SetupSuite() {
	testSuite := getSuite(s.T())
	s.f = testSuite.Factory
}

// insertMember создаёт запись в members_lookup для организации.
func (s *ReviewLifecycleSuite) insertMember(orgID uuid.UUID, createdAt time.Time) {
	s.T().Helper()

	ctx := context.Background()
	_, err := s.f.DB().Exec(ctx,
		`INSERT INTO members_lookup (id, organization_id, email, password_hash, name, role, status, created_at)
		 VALUES ($1, $2, $3, 'hash', 'Test', 'owner', 'active', $4)`,
		uuid.New(), orgID, helpers.RandomEmail(), createdAt,
	)
	s.Require().NoError(err, "insert member lookup")
}

// waitForReviewStatus ожидает, пока отзыв достигнет указанного статуса в reviews_lookup.
func (s *ReviewLifecycleSuite) waitForReviewStatus(reviewID uuid.UUID, expectedStatus string) *projections.ReviewLookupRow {
	s.T().Helper()

	return helpers.WaitFor(s.T(), func() (*projections.ReviewLookupRow, bool) {
		row, err := s.f.ReviewsProjection().GetReviewByID(context.Background(), reviewID)
		if err != nil {
			return nil, false
		}
		return row, row.Status == expectedStatus
	}, "review "+reviewID.String()+" to reach status "+expectedStatus)
}

// TestRLC001_FullPipelineCleanReview проверяет полный пайплайн для чистого отзыва:
// CreateFromFreightReview → review_analyzer (auto-approve) → reviews_projection → status=approved.
func (s *ReviewLifecycleSuite) TestRLC001_FullPipelineCleanReview() {
	ctx := context.Background()

	reviewerOrgID := uuid.New()
	reviewedOrgID := uuid.New()

	// Организации старше 12 мес — все weight компоненты = 1.0
	s.insertMember(reviewerOrgID, time.Now().AddDate(0, -13, 0))
	s.insertMember(reviewedOrgID, time.Now().AddDate(0, -13, 0))

	reviewID := uuid.New()
	reviewService := s.f.ReviewService()

	err := reviewService.CreateFromFreightReview(ctx, reviewApp.CreateFromFreightReviewInput{
		ReviewID:         reviewID,
		FreightRequestID: uuid.New(),
		ReviewerOrgID:    reviewerOrgID,
		ReviewedOrgID:    reviewedOrgID,
		Rating:           4,
		Comment:          "Good service, delivered on time",
		FreightAmount:    10_000_000, // 100K RUB
		FreightCurrency:  "RUB",
		FreightCreatedAt: time.Now().Add(-24 * time.Hour),
		CompletedAt:      time.Now().Add(-12 * time.Hour),
	})
	s.Require().NoError(err, "CreateFromFreightReview не должен возвращать ошибку")

	// Ожидаем, что review пройдёт через analyzer и станет approved (auto-approval)
	row := s.waitForReviewStatus(reviewID, values.StatusApproved.String())

	s.Assert().InDelta(0.0, row.FraudScore, 0.01, "fraud score чистого отзыва должен быть 0")
	s.Assert().False(row.RequiresModeration, "модерация не должна требоваться")
	s.Assert().NotNil(row.ActivationDate, "activation date должна быть установлена")

	expectedActivation := time.Now().AddDate(0, 0, values.FraudActivationDelayDays)
	s.Assert().WithinDuration(expectedActivation, *row.ActivationDate, 24*time.Hour,
		"activation date должна быть ~%d дней от сейчас", values.FraudActivationDelayDays)
}

// TestRLC002_ModerationPath проверяет, что отзыв с SameIP (fraud_score >= 0.3)
// попадает на модерацию (status = pending_moderation).
func (s *ReviewLifecycleSuite) TestRLC002_ModerationPath() {
	ctx := context.Background()

	reviewerOrgID := uuid.New()
	reviewedOrgID := uuid.New()

	s.insertMember(reviewerOrgID, time.Now().AddDate(0, -13, 0))
	s.insertMember(reviewedOrgID, time.Now().AddDate(0, -13, 0))

	// Общий IP → SameIP (0.5) и общий fingerprint → SameFingerprint (0.5) — итого >= 0.3
	sharedIP := "10.0.0.42"
	sharedFP := "fp-shared-device-xyz"

	_, err := s.f.DB().Exec(ctx,
		`UPDATE members_lookup SET registration_ip = $1, last_login_ip = $1,
		 registration_fingerprint = $2, last_login_fingerprint = $2
		 WHERE organization_id = $3`,
		sharedIP, sharedFP, reviewerOrgID,
	)
	s.Require().NoError(err)

	_, err = s.f.DB().Exec(ctx,
		`UPDATE members_lookup SET registration_ip = $1, last_login_ip = $1,
		 registration_fingerprint = $2, last_login_fingerprint = $2
		 WHERE organization_id = $3`,
		sharedIP, sharedFP, reviewedOrgID,
	)
	s.Require().NoError(err)

	reviewID := uuid.New()
	reviewService := s.f.ReviewService()

	err = reviewService.CreateFromFreightReview(ctx, reviewApp.CreateFromFreightReviewInput{
		ReviewID:         reviewID,
		FreightRequestID: uuid.New(),
		ReviewerOrgID:    reviewerOrgID,
		ReviewedOrgID:    reviewedOrgID,
		Rating:           5,
		Comment:          "Suspicious review from same device",
		FreightAmount:    10_000_000,
		FreightCurrency:  "RUB",
		FreightCreatedAt: time.Now().Add(-24 * time.Hour),
		CompletedAt:      time.Now().Add(-12 * time.Hour),
	})
	s.Require().NoError(err)

	// Ожидаем pending_moderation
	row := s.waitForReviewStatus(reviewID, values.StatusPendingModeration.String())

	s.Assert().GreaterOrEqual(row.FraudScore, values.FraudModerationScoreThreshold,
		"fraud score должен быть >= порога модерации")
	s.Assert().True(row.RequiresModeration, "должна требоваться модерация")
}

// TestRLC003_ModeratorApproveAndActivation тестирует одобрение модератором
// и последующую активацию отзыва на уровне агрегата.
// Полный E2E путь: aggregate.RecordAnalysis(requiresModeration=true) → Approve → Activate.
func (s *ReviewLifecycleSuite) TestRLC003_ModeratorApproveAndActivation() {
	reviewerOrgID := uuid.New()
	reviewedOrgID := uuid.New()
	moderatorID := uuid.New()

	// Создаём агрегат напрямую
	r := review.New(
		uuid.New(), uuid.New(),
		reviewerOrgID, reviewedOrgID,
		3, "mediocre service",
		10_000_000, "RUB",
		time.Now().Add(-24*time.Hour),
		time.Now().Add(-12*time.Hour),
	)

	// Анализ с высоким fraud score — требует модерации
	signals := []events.FraudSignal{
		{
			Type:        values.SignalSameIP.String(),
			Severity:    values.SeverityHigh.String(),
			Description: "orgs share IP",
			ScoreImpact: 0.5,
		},
	}
	err := r.RecordAnalysis(0.85, signals, 0.5, true, time.Now().Add(-1*time.Hour))
	s.Require().NoError(err)
	s.Assert().Equal(values.StatusPendingModeration, r.Status())

	// Модератор одобряет с скорректированным весом
	adjustedWeight := 0.6
	err = r.Approve(moderatorID, adjustedWeight, "approved after manual check")
	s.Require().NoError(err)
	s.Assert().Equal(values.StatusApproved, r.Status())
	s.Assert().InDelta(adjustedWeight, r.FinalWeight(), 0.001)
	s.Assert().Equal(&moderatorID, r.ModeratedBy())

	// Активация: activation_date уже в прошлом (установлена при RecordAnalysis)
	err = r.Activate()
	s.Require().NoError(err)
	s.Assert().Equal(values.StatusActive, r.Status())
	s.Assert().NotNil(r.ActivatedAt())
	s.Assert().InDelta(adjustedWeight, r.FinalWeight(), 0.001,
		"FinalWeight должен сохраниться после активации")
}

// TestRLC004_DeactivationRemovesRating проверяет, что после деактивации отзыва
// рейтинг организации пересчитывается (weighted_average уменьшается).
func (s *ReviewLifecycleSuite) TestRLC004_DeactivationRemovesRating() {
	ratings := s.f.OrganizationRatingsProjection()
	ctx := context.Background()
	orgID := uuid.New()

	// Добавляем взвешенный рейтинг
	err := ratings.AddWeightedRating(ctx, orgID, 5, 0.9)
	s.Require().NoError(err)

	rating, err := ratings.GetRating(ctx, orgID)
	s.Require().NoError(err)
	s.Assert().Greater(rating.WeightedAverage, 0.0, "рейтинг должен быть > 0 после добавления")

	// Удаляем тот же рейтинг (имитация деактивации)
	err = ratings.RemoveWeightedRating(ctx, orgID, 5, 0.9)
	s.Require().NoError(err)

	rating, err = ratings.GetRating(ctx, orgID)
	s.Require().NoError(err)
	s.Assert().InDelta(0.0, rating.WeightedAverage, 0.01, "weighted_average должен стать 0 после удаления")
	s.Assert().InDelta(0.0, rating.AverageRating, 0.01, "average_rating должен стать 0 после удаления")
}

// TestRLC005_ModeratorReject тестирует отклонение отзыва модератором.
func (s *ReviewLifecycleSuite) TestRLC005_ModeratorReject() {
	reviewerOrgID := uuid.New()
	reviewedOrgID := uuid.New()
	moderatorID := uuid.New()

	r := review.New(
		uuid.New(), uuid.New(),
		reviewerOrgID, reviewedOrgID,
		5, "totally fake review",
		10_000_000, "RUB",
		time.Now().Add(-24*time.Hour),
		time.Now().Add(-12*time.Hour),
	)

	// Анализ → pending_moderation
	err := r.RecordAnalysis(0.5, nil, 0.6, true, time.Now().AddDate(0, 0, 14))
	s.Require().NoError(err)
	s.Assert().Equal(values.StatusPendingModeration, r.Status())

	// Модератор отклоняет
	err = r.Reject(moderatorID, "obvious spam/fake review")
	s.Require().NoError(err)
	s.Assert().Equal(values.StatusRejected, r.Status())
	s.Assert().True(r.Status().IsTerminal(), "rejected — терминальный статус")

	// Проверяем, что повторный reject невозможен
	err = r.Reject(moderatorID, "reject again")
	s.Assert().Error(err, "повторный reject должен вернуть ошибку")

	// Проверяем, что активация невозможна
	err = r.Activate()
	s.Assert().Error(err, "активация rejected отзыва должна вернуть ошибку")
}

// TestRLC006_ActivationBeforeDateFails проверяет, что Activate возвращает ошибку,
// если activation_date ещё не наступила.
func (s *ReviewLifecycleSuite) TestRLC006_ActivationBeforeDateFails() {
	r := review.New(
		uuid.New(), uuid.New(),
		uuid.New(), uuid.New(),
		4, "good",
		10_000_000, "RUB",
		time.Now().Add(-24*time.Hour),
		time.Now().Add(-12*time.Hour),
	)

	// activation_date в будущем (через 7 дней)
	futureDate := time.Now().AddDate(0, 0, 7)
	err := r.RecordAnalysis(0.9, nil, 0.0, false, futureDate)
	s.Require().NoError(err)
	s.Assert().Equal(values.StatusApproved, r.Status())

	// Попытка активации до наступления даты
	err = r.Activate()
	s.Assert().ErrorIs(err, review.ErrActivationDateNotPassed,
		"активация до activation_date должна вернуть ErrActivationDateNotPassed")
}

// TestRLC007_StatusTransitionGuards проверяет, что недопустимые переходы статусов блокируются.
func (s *ReviewLifecycleSuite) TestRLC007_StatusTransitionGuards() {
	// Нельзя approve pending_analysis (не прошедший анализ)
	r1 := review.New(
		uuid.New(), uuid.New(),
		uuid.New(), uuid.New(),
		4, "test", 10_000_000, "RUB",
		time.Now().Add(-24*time.Hour), time.Now().Add(-12*time.Hour),
	)
	err := r1.Approve(uuid.New(), 0.8, "premature approve")
	s.Assert().ErrorIs(err, review.ErrReviewNotPendingMod,
		"approve без анализа должен вернуть ошибку")

	// Нельзя RecordAnalysis дважды
	r2 := review.New(
		uuid.New(), uuid.New(),
		uuid.New(), uuid.New(),
		4, "test", 10_000_000, "RUB",
		time.Now().Add(-24*time.Hour), time.Now().Add(-12*time.Hour),
	)
	err = r2.RecordAnalysis(0.9, nil, 0.0, false, time.Now().Add(-1*time.Hour))
	s.Require().NoError(err)

	err = r2.RecordAnalysis(0.5, nil, 0.1, false, time.Now().Add(-1*time.Hour))
	s.Assert().ErrorIs(err, review.ErrReviewAlreadyAnalyzed,
		"повторный RecordAnalysis должен вернуть ErrReviewAlreadyAnalyzed")

	// Нельзя деактивировать rejected отзыв
	r3 := review.New(
		uuid.New(), uuid.New(),
		uuid.New(), uuid.New(),
		4, "test", 10_000_000, "RUB",
		time.Now().Add(-24*time.Hour), time.Now().Add(-12*time.Hour),
	)
	_ = r3.RecordAnalysis(0.5, nil, 0.5, true, time.Now().Add(7*24*time.Hour))
	_ = r3.Reject(uuid.New(), "spam")

	err = r3.Deactivate("fraudster")
	s.Assert().ErrorIs(err, review.ErrReviewTerminalStatus,
		"деактивация rejected отзыва должна вернуть ErrReviewTerminalStatus")
}
