package tests

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
	"github.com/udisondev/veziizi/backend/e2e/client"
	"github.com/udisondev/veziizi/backend/e2e/fixtures"
	"github.com/udisondev/veziizi/backend/e2e/helpers"
	"github.com/udisondev/veziizi/backend/internal/domain/review/values"
	"github.com/udisondev/veziizi/backend/internal/infrastructure/projections"
	"github.com/udisondev/veziizi/backend/internal/pkg/factory"
)

// RatingFullCycleSuite покрывает пробел, описанный в organization_ratings_test.go:
// "review worker'ы не запущены в e2e, тестируем через прямой вызов projection".
//
// Эти тесты гоняют полный pipeline через HTTP + event handlers + ручную активацию:
//
//	LeaveReview → review-receiver → review-analyzer → reviews-projection (approved)
//	→ ForceActivateReview → ReviewActivated → reviews-projection (active)
//	→ organization_ratings.weighted_average обновлён
//	→ GET /organizations/{id}/rating возвращает обновлённые цифры.
//
// Активатор (scheduled worker) в e2e suite не запущен и activation_date по умолчанию
// 7-14 дней в будущем — поэтому используется fixtures.ForceActivateReview, который
// переписывает activation_date в прошлое и вызывает ReviewService.Activate.
type RatingFullCycleSuite struct {
	suite.Suite
	baseURL string
	f       *factory.Factory
	ctx     *fixtures.TestContext
}

func TestRatingFullCycleSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(RatingFullCycleSuite))
}

func (s *RatingFullCycleSuite) SetupSuite() {
	testSuite := getSuite(s.T())
	s.baseURL = testSuite.BaseURL
	s.f = testSuite.Factory
	s.ctx = fixtures.NewTestContext(s.T(), s.baseURL)
}

// waitForReviewStatus ждёт, пока отзыв достигнет указанного статуса.
// При таймауте дополнительно логирует текущее состояние review + fraud signals для отладки.
func (s *RatingFullCycleSuite) waitForReviewStatus(reviewID uuid.UUID, expected string) *projections.ReviewLookupRow {
	s.T().Helper()

	ctx := context.Background()
	var last *projections.ReviewLookupRow
	// 60s для стабильности при параллельном запуске всех suites — event pipeline
	// может отставать под нагрузкой (watermill SQL publisher/subscriber на одной БД).
	deadline := time.Now().Add(60 * time.Second)
	for time.Now().Before(deadline) {
		row, err := s.f.ReviewsProjection().GetReviewByID(ctx, reviewID)
		if err == nil && row != nil {
			last = row
			if row.Status == expected {
				return row
			}
		}
		time.Sleep(200 * time.Millisecond)
	}

	// Таймаут — дампим состояние для отладки
	if last != nil {
		signals, _ := s.f.ReviewsProjection().GetFraudSignalsByReviewID(ctx, reviewID)
		signalDesc := make([]string, 0, len(signals))
		for _, sig := range signals {
			signalDesc = append(signalDesc, sig.Type+"("+sig.Description+")")
		}
		s.T().Fatalf("review %s: expected status=%s, got status=%s, fraud_score=%.2f, requires_moderation=%t, signals=%v",
			reviewID, expected, last.Status, last.FraudScore, last.RequiresModeration, signalDesc)
	} else {
		s.T().Fatalf("review %s: not found in projection after 60s (expected status=%s)", reviewID, expected)
	}
	return nil
}

// leaveReview обёртка над HTTP LeaveReview c ожиданием 201 и возвратом review_id.
func (s *RatingFullCycleSuite) leaveReview(c *client.Client, frID uuid.UUID, rating int, comment string) uuid.UUID {
	s.T().Helper()
	resp, err := c.LeaveFreightRequestReview(frID, rating, &comment)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusCreated, resp.StatusCode, "LeaveReview: %s", string(resp.RawBody))
	s.Require().NotEqual(uuid.Nil, resp.Body.ReviewID)
	return resp.Body.ReviewID
}

// prepareOrgsForCleanReview задаёт уникальные IP/fingerprint обеим организациям
// (чтобы не триггерить SameIP между e2e-орг) и опционально бекдейтит created_at
// reviewer'а для максимального orgAgeWeight.
func (s *RatingFullCycleSuite) prepareOrgsForCleanReview(customerID, carrierID uuid.UUID, agedReviewer bool) {
	s.T().Helper()
	fixtures.SetUniqueOrgMetadata(s.T(), s.f, customerID)
	fixtures.SetUniqueOrgMetadata(s.T(), s.f, carrierID)
	if agedReviewer {
		fixtures.SetOrgCreatedAt(s.T(), s.f, customerID, time.Now().AddDate(0, -13, 0))
		fixtures.SetOrgCreatedAt(s.T(), s.f, carrierID, time.Now().AddDate(0, -13, 0))
	}
}

// TestRFC001_CustomerReviewReachesCarrierRating:
// customer оставляет отзыв → ждём approved → ForceActivate → GET rating у carrier
// показывает 1 отзыв и корректный average_rating.
func (s *RatingFullCycleSuite) TestRFC001_CustomerReviewReachesCarrierRating() {
	s.prepareOrgsForCleanReview(s.ctx.Customer.OrganizationID, s.ctx.Carrier.OrganizationID, true)

	completed := s.ctx.CreateFullyCompletedFreightRequest()

	// Baseline: у carrier до отзыва рейтинг нулевой
	before, err := s.ctx.AnonClient.GetOrganizationRating(s.ctx.Carrier.OrganizationID)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusOK, before.StatusCode)
	baselineTotal := before.Body.TotalReviews
	baselineSum := before.Body.AverageRating * float64(baselineTotal)

	reviewID := s.leaveReview(s.ctx.Customer.Client, completed.FreightRequest.ID, 5, "excellent carrier")

	// Event pipeline дожжигает review до approved (чистый отзыв — auto-approve)
	s.waitForReviewStatus(reviewID, values.StatusApproved.String())

	// Pending_reviews счётчик вырос и затем снизился (approved) — между Received и Approved
	// окно крошечное, поэтому проверяем конечное состояние после approve.

	// Форсированная активация (обход 7-дневной задержки)
	fixtures.ForceActivateReview(s.T(), s.f, reviewID)
	s.waitForReviewStatus(reviewID, values.StatusActive.String())

	// Rating у carrier должен отразить новый отзыв
	helpers.WaitWithConfig(s.T(),
		helpers.WaitConfig{Timeout: 10 * time.Second, Interval: 200 * time.Millisecond},
		func() bool {
			resp, err := s.ctx.AnonClient.GetOrganizationRating(s.ctx.Carrier.OrganizationID)
			return err == nil && resp.StatusCode == http.StatusOK &&
				resp.Body.TotalReviews == baselineTotal+1
		},
		"carrier rating to include new review",
	)

	after, err := s.ctx.AnonClient.GetOrganizationRating(s.ctx.Carrier.OrganizationID)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusOK, after.StatusCode)
	s.Assert().Equal(baselineTotal+1, after.Body.TotalReviews, "должен добавиться 1 отзыв")

	expectedAvg := (baselineSum + 5.0) / float64(baselineTotal+1)
	s.Assert().InDelta(expectedAvg, after.Body.AverageRating, 0.01,
		"average_rating должен учесть новую 5★")
	s.Assert().Greater(after.Body.WeightedAverage, 0.0,
		"weighted_average должен стать > 0 после активации")
}

// TestRFC002_MutualReviewsBothAppear:
// оба участника сделки оставляют отзывы друг другу. Порог mutual_reviews = 5/месяц,
// так что 2 взаимных отзыва не должны попасть на модерацию → обе стороны получат рейтинг.
// Это ключевой happy path для "новых пользователей, впервые работающих друг с другом".
func (s *RatingFullCycleSuite) TestRFC002_MutualReviewsBothAppear() {
	customer := s.ctx.QuickCustomer()
	carrier := s.ctx.QuickCarrier()
	s.prepareOrgsForCleanReview(customer.OrganizationID, carrier.OrganizationID, true)

	// Создаём FR между этими двумя свежими организациями (обход shared ctx.Customer/Carrier)
	fr := fixtures.NewFreightRequest(s.T(), customer.Client).Create()
	offer := fixtures.NewOffer(s.T(), carrier.Client, fr.ID).Create()

	selectResp, err := customer.Client.SelectOffer(fr.ID, offer.OfferID)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusNoContent, selectResp.StatusCode)

	confirmResp, err := carrier.Client.ConfirmOffer(fr.ID, offer.OfferID)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusNoContent, confirmResp.StatusCode)

	// Оба completes
	for _, c := range []*client.Client{customer.Client, carrier.Client} {
		resp, err := c.CompleteFreightRequest(fr.ID)
		s.Require().NoError(err)
		s.Require().Equal(http.StatusNoContent, resp.StatusCode)
	}

	// Оба оставляют отзывы
	customerReviewID := s.leaveReview(customer.Client, fr.ID, 5, "great carrier, fast delivery")
	carrierReviewID := s.leaveReview(carrier.Client, fr.ID, 4, "good customer, paid on time")

	// Оба должны дойти до approved (fraud_score < 0.3 — mutual=2, порог > 5)
	s.waitForReviewStatus(customerReviewID, values.StatusApproved.String())
	s.waitForReviewStatus(carrierReviewID, values.StatusApproved.String())

	// Активируем оба
	fixtures.ForceActivateReview(s.T(), s.f, customerReviewID)
	fixtures.ForceActivateReview(s.T(), s.f, carrierReviewID)
	s.waitForReviewStatus(customerReviewID, values.StatusActive.String())
	s.waitForReviewStatus(carrierReviewID, values.StatusActive.String())

	// У carrier рейтинг = 5 (от customer), у customer рейтинг = 4 (от carrier)
	carrierRating, err := s.ctx.AnonClient.GetOrganizationRating(carrier.OrganizationID)
	s.Require().NoError(err)
	s.Assert().Equal(1, carrierRating.Body.TotalReviews, "carrier должен получить 1 отзыв")
	s.Assert().InDelta(5.0, carrierRating.Body.AverageRating, 0.01)

	customerRating, err := s.ctx.AnonClient.GetOrganizationRating(customer.OrganizationID)
	s.Require().NoError(err)
	s.Assert().Equal(1, customerRating.Body.TotalReviews, "customer должен получить 1 отзыв")
	s.Assert().InDelta(4.0, customerRating.Body.AverageRating, 0.01)
}

// TestRFC003_NewOrgCanReceiveRating проверяет сценарий из жалобы пользователя:
// обе организации созданы только что (created_at = now), orgAgeWeight reviewer'а = 0.3.
// Рейтинг должен появиться с уменьшенным weight, но TotalReviews=1.
func (s *RatingFullCycleSuite) TestRFC003_NewOrgCanReceiveRating() {
	customer := s.ctx.QuickCustomer()
	carrier := s.ctx.QuickCarrier()
	// НЕ backdate created_at — организации действительно новые
	s.prepareOrgsForCleanReview(customer.OrganizationID, carrier.OrganizationID, false)

	fr := fixtures.NewFreightRequest(s.T(), customer.Client).Create()
	offer := fixtures.NewOffer(s.T(), carrier.Client, fr.ID).Create()

	selectResp, err := customer.Client.SelectOffer(fr.ID, offer.OfferID)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusNoContent, selectResp.StatusCode)

	confirmResp, err := carrier.Client.ConfirmOffer(fr.ID, offer.OfferID)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusNoContent, confirmResp.StatusCode)

	for _, c := range []*client.Client{customer.Client, carrier.Client} {
		resp, err := c.CompleteFreightRequest(fr.ID)
		s.Require().NoError(err)
		s.Require().Equal(http.StatusNoContent, resp.StatusCode)
	}

	reviewID := s.leaveReview(customer.Client, fr.ID, 5, "new org test")
	row := s.waitForReviewStatus(reviewID, values.StatusApproved.String())

	// Fraud_score должен остаться < 0.3 (без SameIP, без mutual, без burst)
	s.Assert().Less(row.FraudScore, 0.3,
		"для новой организации с одним отзывом fraud_score должен быть < 0.3, получили %.2f", row.FraudScore)

	// Активация
	fixtures.ForceActivateReview(s.T(), s.f, reviewID)
	s.waitForReviewStatus(reviewID, values.StatusActive.String())

	rating, err := s.ctx.AnonClient.GetOrganizationRating(carrier.OrganizationID)
	s.Require().NoError(err)
	s.Assert().Equal(1, rating.Body.TotalReviews, "TotalReviews=1 даже для новой org")
	s.Assert().InDelta(5.0, rating.Body.AverageRating, 0.01,
		"average_rating не зависит от weight — должен быть 5.0")
	s.Assert().Greater(rating.Body.WeightedAverage, 0.0,
		"weighted_average должен быть > 0 даже с пониженным weight")
}

// TestRFC004_RejectedReviewDoesNotAffectRating: отзыв с SameIP/fp попадает на модерацию,
// админ отклоняет → рейтинг carrier не меняется. Pending_reviews должен сброситься.
func (s *RatingFullCycleSuite) TestRFC004_RejectedReviewDoesNotAffectRating() {
	customer := s.ctx.QuickCustomer()
	carrier := s.ctx.QuickCarrier()

	// ОДИНАКОВЫЕ IP/FP — триггерит SameIP (0.5) + SameFingerprint (0.5) → pending_moderation
	sharedIP := "192.168.99.77"
	sharedFP := "fp-rfc004-shared"
	fixtures.SetMemberMetadata(s.T(), s.f, customer.OrganizationID, sharedIP, sharedFP)
	fixtures.SetMemberMetadata(s.T(), s.f, carrier.OrganizationID, sharedIP, sharedFP)

	fr := fixtures.NewFreightRequest(s.T(), customer.Client).Create()
	offer := fixtures.NewOffer(s.T(), carrier.Client, fr.ID).Create()

	selectResp, err := customer.Client.SelectOffer(fr.ID, offer.OfferID)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusNoContent, selectResp.StatusCode)

	confirmResp, err := carrier.Client.ConfirmOffer(fr.ID, offer.OfferID)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusNoContent, confirmResp.StatusCode)

	for _, c := range []*client.Client{customer.Client, carrier.Client} {
		resp, err := c.CompleteFreightRequest(fr.ID)
		s.Require().NoError(err)
		s.Require().Equal(http.StatusNoContent, resp.StatusCode)
	}

	baselineRating, err := s.ctx.AnonClient.GetOrganizationRating(carrier.OrganizationID)
	s.Require().NoError(err)
	baselineTotal := baselineRating.Body.TotalReviews

	reviewID := s.leaveReview(customer.Client, fr.ID, 5, "suspicious same-ip review")

	// Ожидаем pending_moderation (не auto-approved)
	s.waitForReviewStatus(reviewID, values.StatusPendingModeration.String())

	// Счётчик pending_reviews у carrier вырос
	helpers.Wait(s.T(), func() bool {
		r, err := s.ctx.AnonClient.GetOrganizationRating(carrier.OrganizationID)
		return err == nil && r.Body.PendingReviews >= 1
	}, "pending_reviews to include new review")

	// Админ отклоняет
	rejectResp, err := s.ctx.AdminClient.AdminRejectReview(reviewID, "sock puppet detected")
	s.Require().NoError(err)
	s.Require().Equal(http.StatusNoContent, rejectResp.StatusCode)

	s.waitForReviewStatus(reviewID, values.StatusRejected.String())

	// Pending_reviews снизился; total_reviews не изменился
	helpers.Wait(s.T(), func() bool {
		r, err := s.ctx.AnonClient.GetOrganizationRating(carrier.OrganizationID)
		return err == nil &&
			r.Body.PendingReviews == 0 &&
			r.Body.TotalReviews == baselineTotal
	}, "pending_reviews decremented and total_reviews unchanged")

	final, err := s.ctx.AnonClient.GetOrganizationRating(carrier.OrganizationID)
	s.Require().NoError(err)
	s.Assert().Equal(baselineTotal, final.Body.TotalReviews,
		"rejected отзыв не должен попасть в total_reviews")
	s.Assert().Equal(0, final.Body.PendingReviews)
	s.Assert().InDelta(0.0, final.Body.WeightedAverage, 0.01,
		"rejected отзыв не должен влиять на weighted_average")
}

// TestRFC005_ApprovedButNotActivatedWeightedAverageZero: auto-approved отзыв до активации
// ВИДЕН в total_reviews (API считает approved+active), но average_rating и weighted_average
// остаются 0 — эти поля обновляются только при ReviewActivated через AddWeightedRating.
func (s *RatingFullCycleSuite) TestRFC005_ApprovedButNotActivatedWeightedAverageZero() {
	customer := s.ctx.QuickCustomer()
	carrier := s.ctx.QuickCarrier()
	s.prepareOrgsForCleanReview(customer.OrganizationID, carrier.OrganizationID, true)

	fr := fixtures.NewFreightRequest(s.T(), customer.Client).Create()
	offer := fixtures.NewOffer(s.T(), carrier.Client, fr.ID).Create()

	selectResp, err := customer.Client.SelectOffer(fr.ID, offer.OfferID)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusNoContent, selectResp.StatusCode)

	confirmResp, err := carrier.Client.ConfirmOffer(fr.ID, offer.OfferID)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusNoContent, confirmResp.StatusCode)

	for _, c := range []*client.Client{customer.Client, carrier.Client} {
		resp, err := c.CompleteFreightRequest(fr.ID)
		s.Require().NoError(err)
		s.Require().Equal(http.StatusNoContent, resp.StatusCode)
	}

	reviewID := s.leaveReview(customer.Client, fr.ID, 5, "clean review, await activation")
	s.waitForReviewStatus(reviewID, values.StatusApproved.String())

	// API видит approved отзыв: он в total_reviews (COUNT status IN active/approved),
	// но average_rating и weighted_average = 0 — поля organization_ratings не трогаются
	// до ReviewActivated.
	rating, err := s.ctx.AnonClient.GetOrganizationRating(carrier.OrganizationID)
	s.Require().NoError(err)
	s.Assert().Equal(1, rating.Body.TotalReviews,
		"approved отзыв виден в total_reviews сразу после одобрения")
	s.Assert().InDelta(0.0, rating.Body.AverageRating, 0.01,
		"average_rating=0 пока отзыв не активирован — AddWeightedRating ещё не вызван")
	s.Assert().InDelta(0.0, rating.Body.WeightedAverage, 0.01,
		"weighted_average=0 пока отзыв не активирован")
	s.Assert().Equal(0, rating.Body.PendingReviews,
		"pending_reviews=0 — отзыв уже approved")

	// Контрольная проверка: после форсированной активации average_rating становится 5.0
	fixtures.ForceActivateReview(s.T(), s.f, reviewID)
	s.waitForReviewStatus(reviewID, values.StatusActive.String())

	after, err := s.ctx.AnonClient.GetOrganizationRating(carrier.OrganizationID)
	s.Require().NoError(err)
	s.Assert().Equal(1, after.Body.TotalReviews, "total_reviews не должен удвоиться")
	s.Assert().InDelta(5.0, after.Body.AverageRating, 0.01,
		"после activation average_rating = 5.0")
	s.Assert().Greater(after.Body.WeightedAverage, 0.0,
		"после activation weighted_average > 0")
}
