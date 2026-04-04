package tests

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

type HealthSuite struct {
	suite.Suite
	baseURL string
}

func TestHealthSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(HealthSuite))
}

func (s *HealthSuite) SetupSuite() {
	testSuite := getSuite(s.T())
	s.baseURL = testSuite.BaseURL
}

// healthResponse — generic response для health endpoints.
type healthResponse struct {
	Status string         `json:"status"`
	Checks map[string]any `json:"checks,omitempty"`
	Uptime string         `json:"uptime,omitempty"`
}

func (s *HealthSuite) getHealth(path string) (*http.Response, healthResponse) {
	resp, err := http.Get(s.baseURL + path)
	s.Require().NoError(err)
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	s.Require().NoError(err)

	var result healthResponse
	s.Require().NoError(json.Unmarshal(body, &result))

	return resp, result
}

// --- /livez ---

func (s *HealthSuite) TestLivez_ReturnsPass() {
	resp, body := s.getHealth("/livez")

	s.Assert().Equal(http.StatusOK, resp.StatusCode)
	s.Assert().Equal("pass", body.Status)
	s.Assert().Equal("application/health+json", resp.Header.Get("Content-Type"))
	s.Assert().Contains(resp.Header.Get("Cache-Control"), "no-cache")
}

func (s *HealthSuite) TestLivez_NoDependencyChecks() {
	_, body := s.getHealth("/livez")

	// livez не должен содержать checks — только status
	s.Assert().Nil(body.Checks, "livez should not include dependency checks")
}

// --- /readyz ---

func (s *HealthSuite) TestReadyz_ReturnsPassWhenDBAvailable() {
	resp, body := s.getHealth("/readyz")

	s.Assert().Equal(http.StatusOK, resp.StatusCode)
	s.Assert().Equal("pass", body.Status)
	s.Assert().Equal("application/health+json", resp.Header.Get("Content-Type"))
}

func (s *HealthSuite) TestReadyz_IncludesPostgresCheck() {
	_, body := s.getHealth("/readyz")

	s.Require().NotNil(body.Checks)
	pg, ok := body.Checks["postgres"]
	s.Require().True(ok, "readyz should include postgres check")

	pgMap, ok := pg.(map[string]any)
	s.Require().True(ok)
	s.Assert().Equal("pass", pgMap["status"])
}

func (s *HealthSuite) TestReadyz_NoErrorFieldOnSuccess() {
	_, body := s.getHealth("/readyz")

	pgMap := body.Checks["postgres"].(map[string]any)
	_, hasError := pgMap["error"]
	s.Assert().False(hasError, "should not contain error field when healthy")
}

// --- /healthz ---

func (s *HealthSuite) TestHealthz_ReturnsFullReport() {
	resp, body := s.getHealth("/healthz")

	s.Assert().Equal(http.StatusOK, resp.StatusCode)
	s.Assert().Contains([]string{"pass", "warn"}, body.Status)
	s.Assert().NotEmpty(body.Uptime, "healthz should include uptime")
}

func (s *HealthSuite) TestHealthz_IncludesPostgresCheck() {
	_, body := s.getHealth("/healthz")

	s.Require().NotNil(body.Checks)
	pg, ok := body.Checks["postgres"]
	s.Require().True(ok)

	pgMap, ok := pg.(map[string]any)
	s.Require().True(ok)
	s.Assert().Equal("pass", pgMap["status"])
}

func (s *HealthSuite) TestHealthz_IncludesPoolStats() {
	_, body := s.getHealth("/healthz")

	pool, ok := body.Checks["pool"]
	s.Require().True(ok, "healthz should include pool stats")

	poolMap, ok := pool.(map[string]any)
	s.Require().True(ok)
	s.Assert().Equal("pass", poolMap["status"])
	s.Assert().Contains(poolMap, "total_conns")
	s.Assert().Contains(poolMap, "idle_conns")
	s.Assert().Contains(poolMap, "max_conns")
}

func (s *HealthSuite) TestHealthz_IncludesWorkersSection() {
	_, body := s.getHealth("/healthz")

	workers, ok := body.Checks["workers"]
	s.Require().True(ok, "healthz should include workers section")

	workersMap, ok := workers.(map[string]any)
	s.Require().True(ok)
	s.Assert().Contains(workersMap, "status")
	s.Assert().Contains(workersMap, "workers")
}

func (s *HealthSuite) TestHealthz_IncludesDescription() {
	resp, err := http.Get(s.baseURL + "/healthz")
	s.Require().NoError(err)
	defer resp.Body.Close()

	rawBody, err := io.ReadAll(resp.Body)
	s.Require().NoError(err)

	var full map[string]any
	s.Require().NoError(json.Unmarshal(rawBody, &full))

	s.Assert().Equal("Veziizi API Server", full["description"])
}

// --- Идемпотентность ---

func (s *HealthSuite) TestHealthEndpoints_Idempotent() {
	for _, path := range []string{"/livez", "/readyz", "/healthz"} {
		resp1, body1 := s.getHealth(path)
		time.Sleep(10 * time.Millisecond)
		resp2, body2 := s.getHealth(path)

		s.Assert().Equal(resp1.StatusCode, resp2.StatusCode, "status code should be stable for %s", path)
		s.Assert().Equal(body1.Status, body2.Status, "status should be stable for %s", path)
	}
}

// --- No Auth Required ---

func (s *HealthSuite) TestHealthEndpoints_NoAuthRequired() {
	// Без cookies, без сессии — должны отвечать
	plainClient := &http.Client{Timeout: 5 * time.Second}

	for _, path := range []string{"/livez", "/readyz", "/healthz"} {
		resp, err := plainClient.Get(s.baseURL + path)
		s.Require().NoError(err, "should not error for %s", path)
		resp.Body.Close()

		s.Assert().NotEqual(http.StatusUnauthorized, resp.StatusCode, "%s should not require auth", path)
		s.Assert().NotEqual(http.StatusForbidden, resp.StatusCode, "%s should not require auth", path)
	}
}
