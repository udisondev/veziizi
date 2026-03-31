package tests

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
)

// OrganizationRatingsSuite тестирует корректность SQL-запросов в organization_ratings projection.
// Регрессионный тест для бага: SQL type mismatch в AddWeightedRating ($2 int
// использовался для столбцов разных типов — int sum_rating и numeric average_rating/weighted_average).
// После фикса параметры разделены: $2=rating(int), $3=ratingFloat(float64), $4=weightedRating, $5=weight.
type OrganizationRatingsSuite struct {
	suite.Suite
}

func TestOrganizationRatingsSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(OrganizationRatingsSuite))
}

// TestRAT001_AddWeightedRatingNoSQLError проверяет, что AddWeightedRating
// не вызывает SQL ошибку "inconsistent types deduced for parameter $2".
// Воспроизведение: создаём завершённую заявку, оставляем отзыв.
// Внутри review-analyzer worker вызывается AddWeightedRating.
// Если SQL параметры некорректны, операция завалится и рейтинг не появится.
//
// Поскольку review worker'ы не запущены в e2e, тестируем через прямой вызов projection.
func (s *OrganizationRatingsSuite) TestRAT001_AddWeightedRatingNoSQLError() {
	// Вызываем AddWeightedRating напрямую через factory
	testSuite := getSuite(s.T())
	projection := testSuite.Factory.OrganizationRatingsProjection()

	orgID := uuid.New()

	// Тестируем INSERT (новая запись) — это ветка, где ранее падало
	err := projection.AddWeightedRating(context.Background(), orgID, 5, 0.85)
	s.Require().NoError(err, "AddWeightedRating INSERT должен работать без SQL ошибок")

	// Тестируем UPDATE (запись уже существует) — ON CONFLICT DO UPDATE ветка
	err = projection.AddWeightedRating(context.Background(), orgID, 3, 0.70)
	s.Require().NoError(err, "AddWeightedRating UPDATE должен работать без SQL ошибок")

	// Проверяем, что данные корректно записались
	rating, err := projection.GetRating(context.Background(), orgID)
	s.Require().NoError(err, "GetRating должен вернуть запись")
	s.Assert().Equal(orgID, rating.OrgID)
	// После двух вызовов (rating=5, rating=3): average = (5+3)/2 = 4.0
	s.Assert().InDelta(4.0, rating.AverageRating, 0.01, "average_rating должен быть (5+3)/2=4.0")
}

// TestRAT002_AddWeightedRatingWithVariousValues проверяет граничные значения
// параметров, чтобы убедиться, что типы int и float64 корректно разделены.
func (s *OrganizationRatingsSuite) TestRAT002_AddWeightedRatingWithVariousValues() {
	testSuite := getSuite(s.T())
	projection := testSuite.Factory.OrganizationRatingsProjection()

	tests := []struct {
		name   string
		rating int
		weight float64
	}{
		{"min rating", 1, 0.1},
		{"max rating", 5, 1.0},
		{"mid rating with fractional weight", 3, 0.5555},
		{"rating with zero weight", 4, 0.0},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			orgID := uuid.New()
			err := projection.AddWeightedRating(context.Background(), orgID, tt.rating, tt.weight)
			s.Require().NoError(err, "AddWeightedRating(%d, %f) не должен возвращать ошибку", tt.rating, tt.weight)
		})
	}
}

// TestRAT003_RemoveWeightedRatingNoSQLError проверяет симметричную операцию RemoveWeightedRating.
func (s *OrganizationRatingsSuite) TestRAT003_RemoveWeightedRatingNoSQLError() {
	testSuite := getSuite(s.T())
	projection := testSuite.Factory.OrganizationRatingsProjection()

	orgID := uuid.New()

	// Сначала добавляем рейтинг
	err := projection.AddWeightedRating(context.Background(), orgID, 4, 0.9)
	s.Require().NoError(err)

	// Потом удаляем
	err = projection.RemoveWeightedRating(context.Background(), orgID, 4, 0.9)
	s.Require().NoError(err, "RemoveWeightedRating не должен возвращать SQL ошибку")

	// Проверяем, что рейтинг обнулился
	rating, err := projection.GetRating(context.Background(), orgID)
	s.Require().NoError(err)
	s.Assert().InDelta(0.0, rating.AverageRating, 0.01)
	s.Assert().InDelta(0.0, rating.WeightedAverage, 0.01)
}

// TestRAT004_PendingReviewsCounters проверяет increment/decrement pending reviews.
func (s *OrganizationRatingsSuite) TestRAT004_PendingReviewsCounters() {
	testSuite := getSuite(s.T())
	projection := testSuite.Factory.OrganizationRatingsProjection()

	orgID := uuid.New()

	// Increment создаёт запись через INSERT ON CONFLICT
	err := projection.IncrementPendingReviews(context.Background(), orgID)
	s.Require().NoError(err)

	err = projection.IncrementPendingReviews(context.Background(), orgID)
	s.Require().NoError(err)

	rating, err := projection.GetRating(context.Background(), orgID)
	s.Require().NoError(err)
	s.Assert().Equal(2, rating.PendingReviews)

	// Decrement
	err = projection.DecrementPendingReviews(context.Background(), orgID)
	s.Require().NoError(err)

	rating, err = projection.GetRating(context.Background(), orgID)
	s.Require().NoError(err)
	s.Assert().Equal(1, rating.PendingReviews)
}

// TestRAT005_MultipleAddsThenGetRating проверяет корректность weighted average
// после нескольких AddWeightedRating с разными весами. Это ключевой сценарий,
// где баг с type mismatch проявлялся наиболее ярко.
func (s *OrganizationRatingsSuite) TestRAT005_MultipleAddsThenGetRating() {
	testSuite := getSuite(s.T())
	projection := testSuite.Factory.OrganizationRatingsProjection()

	orgID := uuid.New()

	// Добавляем 3 оценки с разными весами
	s.Require().NoError(projection.AddWeightedRating(context.Background(), orgID, 5, 1.0))   // weighted: 5.0
	s.Require().NoError(projection.AddWeightedRating(context.Background(), orgID, 3, 0.5))   // weighted: 1.5
	s.Require().NoError(projection.AddWeightedRating(context.Background(), orgID, 4, 0.75))  // weighted: 3.0

	rating, err := projection.GetRating(context.Background(), orgID)
	s.Require().NoError(err)

	// average_rating = (5+3+4)/3 = 4.0
	s.Assert().InDelta(4.0, rating.AverageRating, 0.01)

	// weighted_average = (5.0+1.5+3.0)/(1.0+0.5+0.75) = 9.5/2.25 ≈ 4.22
	s.Assert().InDelta(4.22, rating.WeightedAverage, 0.02)
}

// TestRAT006_GetRatingNonExistentOrg проверяет, что GetRating для несуществующей
// организации возвращает нулевые значения, а не ошибку.
func (s *OrganizationRatingsSuite) TestRAT006_GetRatingNonExistentOrg() {
	testSuite := getSuite(s.T())
	projection := testSuite.Factory.OrganizationRatingsProjection()

	orgID := uuid.New()

	rating, err := projection.GetRating(context.Background(), orgID)
	s.Require().NoError(err)
	s.Assert().Equal(orgID, rating.OrgID)
	s.Assert().InDelta(0.0, rating.AverageRating, 0.001)
	s.Assert().InDelta(0.0, rating.WeightedAverage, 0.001)
	s.Assert().Equal(0, rating.PendingReviews)
}
