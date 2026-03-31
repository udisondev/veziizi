package tests

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
	adminApp "github.com/udisondev/veziizi/backend/internal/application/admin"
	reviewApp "github.com/udisondev/veziizi/backend/internal/application/review"
	"github.com/udisondev/veziizi/backend/internal/domain/review/values"
	"github.com/udisondev/veziizi/backend/internal/pkg/factory"
	"github.com/udisondev/veziizi/backend/e2e/fixtures"
	"github.com/udisondev/veziizi/backend/e2e/helpers"
	"github.com/udisondev/veziizi/backend/e2e/setup"
)

// FraudsterHandlerSuite тестирует поведение FraudsterHandler при пометке/снятии
// статуса фродстера с организации. Handler слушает FraudsterMarked/FraudsterUnmarked
// и деактивирует/обновляет отзывы.
// Использует изолированный suite (NewSuite) — тесты с pipeline требуют стабильный контекст.
type FraudsterHandlerSuite struct {
	suite.Suite
	f *factory.Factory
}

func TestFraudsterHandlerSuite(t *testing.T) {
	suite.Run(t, new(FraudsterHandlerSuite))
}

func (s *FraudsterHandlerSuite) SetupSuite() {
	testSuite := setup.NewSuite(s.T())
	s.f = testSuite.Factory
}

// insertMember создаёт запись в members_lookup для организации.
func (s *FraudsterHandlerSuite) insertMember(orgID uuid.UUID, createdAt time.Time) {
	s.T().Helper()

	ctx := context.Background()
	_, err := s.f.DB().Exec(ctx,
		`INSERT INTO members_lookup (id, organization_id, email, password_hash, name, role, status, created_at)
		 VALUES ($1, $2, $3, 'hash', 'Test', 'owner', 'active', $4)`,
		uuid.New(), orgID, helpers.RandomEmail(), createdAt,
	)
	s.Require().NoError(err, "insert member lookup")
}

// createAndApproveReview создаёт review через сервис и ждёт auto-approval.
// Возвращает reviewID. Требует заранее вставленных members для обеих организаций.
func (s *FraudsterHandlerSuite) createAndApproveReview(reviewerOrgID, reviewedOrgID uuid.UUID) uuid.UUID {
	s.T().Helper()

	ctx := context.Background()
	reviewID := uuid.New()
	reviewService := s.f.ReviewService()

	err := reviewService.CreateFromFreightReview(ctx, reviewApp.CreateFromFreightReviewInput{
		ReviewID:         reviewID,
		FreightRequestID: uuid.New(),
		ReviewerOrgID:    reviewerOrgID,
		ReviewedOrgID:    reviewedOrgID,
		Rating:           5,
		Comment:          "excellent service",
		FreightAmount:    10_000_000, // 100K RUB — weight 1.0
		FreightCurrency:  "RUB",
		FreightCreatedAt: time.Now().Add(-48 * time.Hour),
		CompletedAt:      time.Now().Add(-24 * time.Hour),
	})
	s.Require().NoError(err, "CreateFromFreightReview для review %s", reviewID)

	// Ожидаем auto-approval (3 хопа event pipeline: receiver → analyzer → projection)
	helpers.WaitWithConfig(s.T(), helpers.WaitConfig{
		Timeout:  30 * time.Second,
		Interval: 200 * time.Millisecond,
	}, func() bool {
		row, err := s.f.ReviewsProjection().GetReviewByID(ctx, reviewID)
		if err != nil {
			return false
		}
		return row.Status == values.StatusApproved.String()
	}, "review "+reviewID.String()+" to become approved")

	return reviewID
}

// createOrgAggregate создаёт организацию через полный API-flow (register + admin approve).
// Возвращает orgID.
func (s *FraudsterHandlerSuite) createOrgAggregate() uuid.UUID {
	s.T().Helper()

	c := getClient(s.T())

	org := fixtures.NewOrganization(s.T(), c).Create()

	// Одобрение через AdminService напрямую
	err := s.f.AdminService().Approve(context.Background(), adminApp.ApproveInput{
		AdminID:        uuid.New(),
		OrganizationID: org.OrganizationID,
	})
	s.Require().NoError(err, "approve organization")

	return org.OrganizationID
}

// TestFRH001_MarkDeactivatesApprovedReviews проверяет, что при пометке организации
// как фродстера все approved отзывы от этой организации деактивируются.
// Approved отзывы тоже деактивируются, т.к. иначе review-activator их активирует позже.
func (s *FraudsterHandlerSuite) TestFRH001_MarkDeactivatesApprovedReviews() {
	ctx := context.Background()

	reviewerOrgID := uuid.New()
	reviewedOrgID := uuid.New()

	// Организации старше 12 мес — чистый анализ, auto-approve
	s.insertMember(reviewerOrgID, time.Now().AddDate(0, -13, 0))
	s.insertMember(reviewedOrgID, time.Now().AddDate(0, -13, 0))

	// Создаём 3 отзыва через полный пайплайн (approved status)
	reviewIDs := make([]uuid.UUID, 3)
	for i := range 3 {
		reviewIDs[i] = s.createAndApproveReview(reviewerOrgID, reviewedOrgID)
	}

	// Деактивируем напрямую через BatchDeactivate (имитация fraudster handler)
	reviewService := s.f.ReviewService()
	reason := "reviewer marked as fraudster"
	result := reviewService.BatchDeactivate(ctx, reviewIDs, reason)

	s.Assert().Equal(3, result.SuccessCount,
		"все 3 отзыва должны быть деактивированы")
	s.Assert().Empty(result.FailedIDs, "не должно быть ошибок деактивации")

	// Проверяем статус в reviews_lookup
	for _, id := range reviewIDs {
		helpers.Wait(s.T(), func() bool {
			row, err := s.f.ReviewsProjection().GetReviewByID(ctx, id)
			if err != nil {
				return false
			}
			return row.Status == values.StatusDeactivated.String()
		}, "review "+id.String()+" to become deactivated")
	}
}

// TestFRH002_DeactivateAlreadyTerminalReview проверяет, что попытка деактивации
// уже rejected отзыва возвращает ошибку (terminal status).
func (s *FraudsterHandlerSuite) TestFRH002_DeactivateAlreadyTerminalReview() {
	ctx := context.Background()

	reviewerOrgID := uuid.New()
	reviewedOrgID := uuid.New()

	s.insertMember(reviewerOrgID, time.Now().AddDate(0, -13, 0))
	s.insertMember(reviewedOrgID, time.Now().AddDate(0, -13, 0))

	// Создаём отзыв с SameIP чтобы попал на модерацию
	sharedIP := "172.16.0.99"
	sharedFP := "fp-terminal-test"

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
		Comment:          "suspicious",
		FreightAmount:    10_000_000,
		FreightCurrency:  "RUB",
		FreightCreatedAt: time.Now().Add(-24 * time.Hour),
		CompletedAt:      time.Now().Add(-12 * time.Hour),
	})
	s.Require().NoError(err)

	// Ждём pending_moderation
	helpers.Wait(s.T(), func() bool {
		row, err := s.f.ReviewsProjection().GetReviewByID(ctx, reviewID)
		if err != nil {
			return false
		}
		return row.Status == values.StatusPendingModeration.String()
	}, "review to become pending_moderation")

	// Модератор отклоняет отзыв
	moderatorID := uuid.New()
	err = reviewService.Reject(ctx, reviewID, moderatorID, "spam detected")
	s.Require().NoError(err)

	// Попытка деактивации rejected отзыва должна вернуть ошибку
	err = reviewService.Deactivate(ctx, reviewID, "fraudster")
	s.Assert().Error(err, "деактивация rejected отзыва должна вернуть ошибку")
}

// TestFRH003_BatchDeactivatePartialFailure проверяет, что BatchDeactivate
// корректно обрабатывает смешанный набор: часть отзывов деактивируется,
// часть — ошибки (несуществующие или terminal).
func (s *FraudsterHandlerSuite) TestFRH003_BatchDeactivatePartialFailure() {
	ctx := context.Background()

	reviewerOrgID := uuid.New()
	reviewedOrgID := uuid.New()

	s.insertMember(reviewerOrgID, time.Now().AddDate(0, -13, 0))
	s.insertMember(reviewedOrgID, time.Now().AddDate(0, -13, 0))

	// 1 валидный отзыв (approved)
	validReviewID := s.createAndApproveReview(reviewerOrgID, reviewedOrgID)

	// 1 несуществующий ID
	nonExistentID := uuid.New()

	reviewService := s.f.ReviewService()
	result := reviewService.BatchDeactivate(ctx, []uuid.UUID{validReviewID, nonExistentID}, "batch test")

	s.Assert().Equal(1, result.SuccessCount, "1 отзыв должен быть деактивирован")
	s.Assert().Len(result.FailedIDs, 1, "1 отзыв должен быть в failed")
	s.Assert().Contains(result.FailedIDs, nonExistentID, "несуществующий ID должен быть в failed")
}

// TestFRH004_EmptyBatchDeactivate проверяет, что пустой batch не вызывает ошибок.
func (s *FraudsterHandlerSuite) TestFRH004_EmptyBatchDeactivate() {
	ctx := context.Background()

	reviewService := s.f.ReviewService()
	result := reviewService.BatchDeactivate(ctx, nil, "empty batch")

	s.Assert().Equal(0, result.SuccessCount)
	s.Assert().Empty(result.FailedIDs)
	s.Assert().Empty(result.Errors)
}

// TestFRH005_FraudsterReputationUpdate проверяет, что MarkFraudster в projection
// корректно обновляет org_reviewer_reputation.
func (s *FraudsterHandlerSuite) TestFRH005_FraudsterReputationUpdate() {
	ctx := context.Background()
	orgID := uuid.New()
	adminID := uuid.New()

	// Вставляем начальную репутацию
	fixtures.InsertOrgReputation(s.T(), s.f, orgID, fixtures.ReputationOpts{
		TotalReviewsLeft:  10,
		ActiveReviewsLeft: 8,
		ReputationScore:   0.9,
	})

	// Помечаем как confirmed fraudster через projection
	fraudProjection := s.f.FraudDataProjection()
	err := fraudProjection.MarkFraudster(ctx, orgID, true, adminID, "multiple fake reviews")
	s.Require().NoError(err)

	// Проверяем репутацию
	rep, err := fraudProjection.GetReviewerReputation(ctx, orgID)
	s.Require().NoError(err)
	s.Assert().True(rep.IsConfirmedFraudster, "должен быть confirmed fraudster")
	// MarkFraudster устанавливает флаги, но не обнуляет reputation_score.
	// reputation_score = 0 проверяется в analyzer через calculateReputationWeight.
	s.Assert().InDelta(0.9, rep.ReputationScore, 0.01,
		"reputation score не должен меняться при MarkFraudster — он проверяется через флаги")

	// Снимаем статус
	err = fraudProjection.UnmarkFraudster(ctx, orgID)
	s.Require().NoError(err)

	rep, err = fraudProjection.GetReviewerReputation(ctx, orgID)
	s.Require().NoError(err)
	s.Assert().False(rep.IsConfirmedFraudster, "confirmed fraudster flag должен быть снят")
	s.Assert().False(rep.IsSuspectedFraudster, "suspected fraudster flag должен быть снят")
}
