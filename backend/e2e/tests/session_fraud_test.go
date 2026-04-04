package tests

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
	"github.com/udisondev/veziizi/backend/internal/application/session"
	"github.com/udisondev/veziizi/backend/internal/infrastructure/projections"
)

// SessionFraudSuite тестирует детекцию фрода сессий: impossible travel,
// unusual login time, country change, API rate limiting, scraping.
type SessionFraudSuite struct {
	suite.Suite
	analyzer     *session.SessionAnalyzer
	sessionFraud *projections.SessionFraudProjection
}

func TestSessionFraudSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(SessionFraudSuite))
}

func (s *SessionFraudSuite) SetupSuite() {
	testSuite := getSuite(s.T())
	s.sessionFraud = testSuite.Factory.SessionFraudProjection()
	s.analyzer = testSuite.Factory.SessionAnalyzer()
}

// setupBehavior создаёт запись member_session_behavior с историей логинов.
func (s *SessionFraudSuite) setupBehavior(memberID uuid.UUID, lastLoginAt time.Time, lat, lon float64, country string, totalLogins int, typicalHours map[int]int) {
	s.T().Helper()
	ctx := context.Background()

	ip := "1.2.3.4"
	behavior := &projections.MemberSessionBehavior{
		MemberID:         memberID,
		TypicalHours:     typicalHours,
		TypicalCountries: []string{country},
		TypicalIPs:       []string{ip},
		LastLoginAt:      &lastLoginAt,
		LastLoginIP:      &ip,
		LastLoginCountry: &country,
		LastLoginLat:     &lat,
		LastLoginLon:     &lon,
		TotalLogins:      totalLogins,
	}

	err := s.sessionFraud.UpsertMemberSessionBehavior(ctx, behavior)
	s.Require().NoError(err)
}

// recordLogins записывает N login events в заданный час для гистограммы.
func (s *SessionFraudSuite) recordLogins(memberID, orgID uuid.UUID, count int, hour int) {
	s.T().Helper()
	ctx := context.Background()

	for i := range count {
		loginTime := time.Date(2025, 1, 1+i%28, hour, 30, 0, 0, time.UTC)
		event := &projections.SessionEvent{
			MemberID:       memberID,
			OrganizationID: orgID,
			EventType:      "login",
			CreatedAt:      loginTime,
		}
		err := s.sessionFraud.RecordSessionEvent(ctx, event)
		s.Require().NoError(err)
	}
}

// --- TestSFR001: Impossible travel (Москва → Алматы за 1 час) ---

func (s *SessionFraudSuite) TestSFR001_GeoJump_ImpossibleTravel() {
	memberID := uuid.New()
	orgID := uuid.New()

	// Москва: 55.7558, 37.6173
	moscowLat, moscowLon := 55.7558, 37.6173
	// Алматы: 43.2220, 76.8512
	almatyLat, almatyLon := 43.2220, 76.8512

	// Расстояние ≈ 3100 км. За 1 час → ~3100 km/h > 900 km/h threshold.
	lastLogin := time.Now().Add(-1 * time.Hour)
	s.setupBehavior(memberID, lastLogin, moscowLat, moscowLon, "RU", 10, nil)

	result, err := s.analyzer.AnalyzeLogin(context.Background(), session.LoginAnalysisInput{
		MemberID:       memberID,
		OrganizationID: orgID,
		IPAddress:      "5.6.7.8",
		GeoCountry:     "KZ",
		GeoCity:        "Almaty",
		GeoLat:         almatyLat,
		GeoLon:         almatyLon,
		LoginTime:      time.Now(),
	})

	s.Require().NoError(err)
	s.Assert().True(result.IsSuspicious, "должен быть подозрительным")
	s.Assert().Contains(result.Signals, projections.SignalLoginGeoJump)
}

// --- TestSFR002: Possible travel (6 часов — самолёт) ---

func (s *SessionFraudSuite) TestSFR002_GeoJump_PossibleTravel() {
	memberID := uuid.New()
	orgID := uuid.New()

	moscowLat, moscowLon := 55.7558, 37.6173
	almatyLat, almatyLon := 43.2220, 76.8512

	// 6 часов → ~517 km/h < 900 km/h threshold
	lastLogin := time.Now().Add(-6 * time.Hour)
	s.setupBehavior(memberID, lastLogin, moscowLat, moscowLon, "RU", 10, nil)

	result, err := s.analyzer.AnalyzeLogin(context.Background(), session.LoginAnalysisInput{
		MemberID:       memberID,
		OrganizationID: orgID,
		IPAddress:      "5.6.7.8",
		GeoCountry:     "KZ",
		GeoCity:        "Almaty",
		GeoLat:         almatyLat,
		GeoLon:         almatyLon,
		LoginTime:      time.Now(),
	})

	s.Require().NoError(err)
	s.Assert().NotContains(result.Signals, projections.SignalLoginGeoJump,
		"не должен быть geo jump при скорости < 900 km/h")
}

// --- TestSFR003: Short distance — не проверяется ---

func (s *SessionFraudSuite) TestSFR003_GeoJump_ShortDistance() {
	memberID := uuid.New()
	orgID := uuid.New()

	// Два района Москвы, ~30 км
	lat1, lon1 := 55.7558, 37.6173
	lat2, lon2 := 55.9000, 37.5000

	lastLogin := time.Now().Add(-5 * time.Minute)
	s.setupBehavior(memberID, lastLogin, lat1, lon1, "RU", 10, nil)

	result, err := s.analyzer.AnalyzeLogin(context.Background(), session.LoginAnalysisInput{
		MemberID:       memberID,
		OrganizationID: orgID,
		IPAddress:      "5.6.7.8",
		GeoCountry:     "RU",
		GeoCity:        "Moscow",
		GeoLat:         lat2,
		GeoLon:         lon2,
		LoginTime:      time.Now(),
	})

	s.Require().NoError(err)
	s.Assert().NotContains(result.Signals, projections.SignalLoginGeoJump,
		"не должен проверять при расстоянии < 100 км")
}

// --- TestSFR004: Unusual login time ---

func (s *SessionFraudSuite) TestSFR004_UnusualLoginTime() {
	memberID := uuid.New()
	orgID := uuid.New()
	ctx := context.Background()

	// Записываем 10 логинов в 10:00-11:00 для создания гистограммы
	s.recordLogins(memberID, orgID, 10, 10)

	// Настраиваем behavior с 10+ логинами
	typicalHours := map[int]int{10: 8, 11: 2}
	lastLogin := time.Now().Add(-8 * time.Hour)
	s.setupBehavior(memberID, lastLogin, 55.75, 37.62, "RU", 10, typicalHours)

	// Логин в 3:00 — разница 7 часов с типичными 10:00
	loginTime := time.Date(2026, 3, 31, 3, 0, 0, 0, time.UTC)

	result, err := s.analyzer.AnalyzeLogin(ctx, session.LoginAnalysisInput{
		MemberID:       memberID,
		OrganizationID: orgID,
		IPAddress:      "1.2.3.4",
		GeoCountry:     "RU",
		GeoCity:        "Moscow",
		GeoLat:         55.75,
		GeoLon:         37.62,
		LoginTime:      loginTime,
	})

	s.Require().NoError(err)
	s.Assert().True(result.IsSuspicious)
	s.Assert().Contains(result.Signals, projections.SignalSessionAnomaly)
}

// --- TestSFR005: Country change within 24h ---

func (s *SessionFraudSuite) TestSFR005_CountryChangeWithin24h() {
	memberID := uuid.New()
	orgID := uuid.New()
	ctx := context.Background()

	// Предыдущий логин из России 12 часов назад
	// Ставим близкие координаты чтобы geo jump не сработал
	lastLogin := time.Now().Add(-12 * time.Hour)
	typicalHours := map[int]int{14: 10}
	s.setupBehavior(memberID, lastLogin, 55.75, 37.62, "RU", 10, typicalHours)
	s.recordLogins(memberID, orgID, 10, 14)

	// Новый логин из Казахстана через 12 часов, но с близких координат
	// (чтобы не триггерить geo_jump — только country change)
	result, err := s.analyzer.AnalyzeLogin(ctx, session.LoginAnalysisInput{
		MemberID:       memberID,
		OrganizationID: orgID,
		IPAddress:      "5.6.7.8",
		GeoCountry:     "KZ",
		GeoCity:        "Aktau",
		GeoLat:         55.80, // близко к предыдущему — не geo jump
		GeoLon:         37.70,
		LoginTime:      time.Now(),
	})

	s.Require().NoError(err)
	s.Assert().True(result.IsSuspicious)
	s.Assert().Contains(result.Signals, projections.SignalSessionAnomaly,
		"country change RU→KZ в течение 24ч должен быть session_anomaly")
}

// --- TestSFR006: API rate limiting ---

func (s *SessionFraudSuite) TestSFR006_APIRateLimiting() {
	memberID := uuid.New()
	orgID := uuid.New()
	ctx := context.Background()

	// Сбрасываем лимит для конкретного ключа
	key := fmt.Sprintf("member:%s", memberID.String())
	err := s.sessionFraud.ResetRateLimit(ctx, key)
	s.Require().NoError(err)

	// Сохраняем оригинальный лимит и ставим низкий для теста
	origLimit := projections.SessionFraudThresholds.MaxRequestsPerMinute
	projections.SessionFraudThresholds.MaxRequestsPerMinute = 5
	defer func() {
		projections.SessionFraudThresholds.MaxRequestsPerMinute = origLimit
	}()

	// Делаем 6 запросов (> 5 лимита)
	var lastResult *session.LoginAnalysisResult
	for range 6 {
		lastResult, err = s.analyzer.CheckAPIAbuse(ctx, memberID, orgID, "GET /api/v1/freight")
		s.Require().NoError(err)
	}

	s.Assert().True(lastResult.IsSuspicious)
	s.Assert().True(lastResult.BlockLogin, "должен заблокировать при превышении лимита")
	s.Assert().Contains(lastResult.Signals, projections.SignalAPIAbuse)
}

// --- TestSFR007: Scraping detection ---

func (s *SessionFraudSuite) TestSFR007_ScrapingDetection() {
	memberID := uuid.New()
	orgID := uuid.New()
	ctx := context.Background()

	// Сбрасываем rate limit чтобы не сработал лимит запросов
	key := fmt.Sprintf("member:%s", memberID.String())
	err := s.sessionFraud.ResetRateLimit(ctx, key)
	s.Require().NoError(err)

	// Сохраняем и ставим высокий лимит чтобы rate limit не блокировал
	origLimit := projections.SessionFraudThresholds.MaxRequestsPerMinute
	projections.SessionFraudThresholds.MaxRequestsPerMinute = 100000
	origScraping := projections.SessionFraudThresholds.ScrapingThreshold
	projections.SessionFraudThresholds.ScrapingThreshold = 5 // порог скрейпинга = 5 для теста
	defer func() {
		projections.SessionFraudThresholds.MaxRequestsPerMinute = origLimit
		projections.SessionFraudThresholds.ScrapingThreshold = origScraping
	}()

	// Записываем 6 GET запросов без POST (> 5 scraping threshold)
	for range 6 {
		event := &projections.SessionEvent{
			MemberID:       memberID,
			OrganizationID: orgID,
			EventType:      "api_call",
			Endpoint:       strPtr("GET /api/v1/freight"),
		}
		err := s.sessionFraud.RecordSessionEvent(ctx, event)
		s.Require().NoError(err)
	}

	// Проверяем через CheckAPIAbuse
	result, err := s.analyzer.CheckAPIAbuse(ctx, memberID, orgID, "GET /api/v1/freight")
	s.Require().NoError(err)

	s.Assert().True(result.IsSuspicious, "должен быть подозрительным при скрейпинге")
	s.Assert().Contains(result.Signals, projections.SignalAPIAbuse)
}

func strPtr(s string) *string { return &s }
