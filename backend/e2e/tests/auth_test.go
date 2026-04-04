package tests

import (
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"
	"github.com/udisondev/veziizi/backend/e2e/client"
	"github.com/udisondev/veziizi/backend/e2e/fixtures"
)

// AuthSuite combines all authentication tests with shared context.
type AuthSuite struct {
	suite.Suite
	baseURL string
	c       *client.Client // Anonymous client

	// Shared organization for auth tests
	org *fixtures.CreatedOrganization
}

func TestAuthSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(AuthSuite))
}

func (s *AuthSuite) SetupSuite() {
	testSuite := getSuite(s.T())
	s.baseURL = testSuite.BaseURL
	s.c = client.New(s.baseURL)

	// Create shared organization once
	s.org = fixtures.NewOrganization(s.T(), s.c).Create()
}

// ==================== POST /api/v1/auth/login ====================

func (s *AuthSuite) TestAUTH001_SuccessfulLogin() {
	testClient := s.c.Clone()
	resp, err := testClient.Login(s.org.OwnerEmail, s.org.OwnerPassword)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusOK, resp.StatusCode, string(resp.RawBody))
	s.Assert().NotEmpty(resp.Body.Email, "email in response")
	s.Assert().NotEmpty(resp.Body.MemberID.String(), "member_id should be set")
}

func (s *AuthSuite) TestAUTH005_MissingEmail() {
	testClient := s.c.Clone()
	resp, err := testClient.Login("", "password123")
	s.Require().NoError(err)
	s.Require().Equal(http.StatusBadRequest, resp.StatusCode)
}

func (s *AuthSuite) TestAUTH006_MissingPassword() {
	testClient := s.c.Clone()
	resp, err := testClient.Login(s.org.OwnerEmail, "")
	s.Require().NoError(err)
	s.Require().Equal(http.StatusBadRequest, resp.StatusCode)
}

func (s *AuthSuite) TestAUTH007_NonexistentEmail() {
	testClient := s.c.Clone()
	resp, err := testClient.Login("nonexistent@test.local", "password123")
	s.Require().NoError(err)
	s.Require().Equal(http.StatusUnauthorized, resp.StatusCode)
	s.Assert().Contains(strings.ToLower(string(resp.RawBody)), "invalid credentials")
}

func (s *AuthSuite) TestAUTH008_WrongPassword() {
	testClient := s.c.Clone()
	resp, err := testClient.Login(s.org.OwnerEmail, "wrongpassword")
	s.Require().NoError(err)
	s.Require().Equal(http.StatusUnauthorized, resp.StatusCode)
	s.Assert().Contains(strings.ToLower(string(resp.RawBody)), "invalid credentials")
}

func (s *AuthSuite) TestAUTH012_SQLInjectionInEmail() {
	testClient := s.c.Clone()
	resp, err := testClient.Login("'; DROP TABLE members--", "password123")
	s.Require().NoError(err)
	s.Require().Equal(http.StatusUnauthorized, resp.StatusCode)
}

func (s *AuthSuite) TestAUTH014_UnicodeInEmail() {
	testClient := s.c.Clone()
	resp, err := testClient.Login("тест@test.com", "password123")
	s.Require().NoError(err)
	s.Require().Equal(http.StatusUnauthorized, resp.StatusCode)
}

func (s *AuthSuite) TestAUTH016_EmptyPassword() {
	testClient := s.c.Clone()
	resp, err := testClient.Login(s.org.OwnerEmail, "")
	s.Require().NoError(err)
	s.Require().Equal(http.StatusBadRequest, resp.StatusCode)
}

// ==================== POST /api/v1/auth/logout ====================

func (s *AuthSuite) TestAUTH025_SuccessfulLogout() {
	testClient := s.c.Clone()
	// Login first
	_, err := testClient.Login(s.org.OwnerEmail, s.org.OwnerPassword)
	s.Require().NoError(err)

	resp, err := testClient.Logout()
	s.Require().NoError(err)
	s.Require().Equal(http.StatusNoContent, resp.StatusCode, string(resp.RawBody))
}

func (s *AuthSuite) TestAUTH027_LogoutWithoutSession() {
	testClient := s.c.Clone()
	resp, err := testClient.Logout()
	s.Require().NoError(err)
	s.Require().Equal(http.StatusUnauthorized, resp.StatusCode)
}

// ==================== GET /api/v1/auth/me ====================

func (s *AuthSuite) TestAUTH030_GetProfileWhenLoggedIn() {
	testClient := s.c.Clone()
	_, err := testClient.Login(s.org.OwnerEmail, s.org.OwnerPassword)
	s.Require().NoError(err)

	resp, err := testClient.Me()
	s.Require().NoError(err)
	s.Require().Equal(http.StatusOK, resp.StatusCode, string(resp.RawBody))
	s.Assert().Equal(s.org.OwnerEmail, resp.Body.Email, "email")
	s.Assert().Equal("owner", resp.Body.Role, "role")
	s.Assert().NotEmpty(resp.Body.OrganizationID.String(), "organization_id")
}

func (s *AuthSuite) TestAUTH036_GetProfileWithoutAuth() {
	testClient := s.c.Clone()
	resp, err := testClient.Me()
	s.Require().NoError(err)
	s.Require().Equal(http.StatusUnauthorized, resp.StatusCode)
}

// ==================== AUTH-029: Actions after logout ====================

func (s *AuthSuite) TestAUTH029_ActionsAfterLogout() {
	testClient := s.c.Clone()

	// Login
	_, err := testClient.Login(s.org.OwnerEmail, s.org.OwnerPassword)
	s.Require().NoError(err)

	// Verify logged in
	meResp, err := testClient.Me()
	s.Require().NoError(err)
	s.Require().Equal(http.StatusOK, meResp.StatusCode, string(meResp.RawBody))

	// Logout
	logoutResp, err := testClient.Logout()
	s.Require().NoError(err)
	s.Require().Equal(http.StatusNoContent, logoutResp.StatusCode, string(logoutResp.RawBody))

	// Try to access protected endpoint
	meResp2, err := testClient.Me()
	s.Require().NoError(err)
	s.Require().Equal(http.StatusUnauthorized, meResp2.StatusCode, string(meResp2.RawBody))
}
