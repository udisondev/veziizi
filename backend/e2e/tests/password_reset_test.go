package tests

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"codeberg.org/udison/veziizi/backend/e2e/client"
	"codeberg.org/udison/veziizi/backend/e2e/fixtures"
	"codeberg.org/udison/veziizi/backend/e2e/setup"
	"codeberg.org/udison/veziizi/backend/internal/infrastructure/projections"
	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
)

// PasswordResetSuite combines all password reset tests with shared context.
type PasswordResetSuite struct {
	suite.Suite
	baseURL string
	c       *client.Client // Anonymous client
	suite   *setup.Suite

	// Shared organization for password reset tests
	org *fixtures.CreatedOrganization
}

func TestPasswordResetSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(PasswordResetSuite))
}

func (s *PasswordResetSuite) SetupSuite() {
	testSuite := getSuite(s.T())
	s.baseURL = testSuite.BaseURL
	s.suite = testSuite
	s.c = client.New(s.baseURL)

	// Create shared organization once
	s.org = fixtures.NewOrganization(s.T(), s.c).Create()
}

// ==================== POST /api/v1/auth/forgot-password ====================

func (s *PasswordResetSuite) TestPWD001_EmptyEmail() {
	testClient := s.c.Clone()
	resp, err := testClient.ForgotPassword("")
	s.Require().NoError(err)
	s.Require().Equal(http.StatusBadRequest, resp.StatusCode)
	s.Assert().Contains(strings.ToLower(string(resp.RawBody)), "email is required")
}

func (s *PasswordResetSuite) TestPWD002_InvalidEmailFormat() {
	testClient := s.c.Clone()
	resp, err := testClient.ForgotPassword("invalid-email")
	s.Require().NoError(err)
	s.Require().Equal(http.StatusBadRequest, resp.StatusCode)
	s.Assert().Contains(strings.ToLower(string(resp.RawBody)), "invalid email format")
}

func (s *PasswordResetSuite) TestPWD003_InvalidEmailFormatNoAt() {
	testClient := s.c.Clone()
	resp, err := testClient.ForgotPassword("notanemail")
	s.Require().NoError(err)
	s.Require().Equal(http.StatusBadRequest, resp.StatusCode)
	s.Assert().Contains(strings.ToLower(string(resp.RawBody)), "invalid email format")
}

func (s *PasswordResetSuite) TestPWD004_NonExistentEmail() {
	testClient := s.c.Clone()
	// Non-existent email should return 204 to prevent email enumeration
	resp, err := testClient.ForgotPassword("nonexistent-user-" + uuid.New().String()[:8] + "@test.local")
	s.Require().NoError(err)
	s.Require().Equal(http.StatusNoContent, resp.StatusCode, "should return 204 to prevent email enumeration")
}

func (s *PasswordResetSuite) TestPWD005_ExistingEmailCreatesToken() {
	// Create a new organization for this test
	org := fixtures.NewOrganization(s.T(), s.c).Create()

	testClient := s.c.Clone()
	resp, err := testClient.ForgotPassword(org.OwnerEmail)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusNoContent, resp.StatusCode)

	// Verify token was created in DB
	token := s.getLastTokenFromDB(org.MemberID)
	s.Assert().NotEmpty(token, "token should be created in database")
}

func (s *PasswordResetSuite) TestPWD006_RateLimit() {
	// Temporarily set lower rate limits for this test
	projections.SetPasswordResetRateLimits(3, 5)
	defer projections.SetPasswordResetRateLimits(10000, 10000) // Restore after test

	// Create a new organization for this test
	org := fixtures.NewOrganization(s.T(), s.c).Create()

	testClient := s.c.Clone()

	// Make 4 requests - first 3 should succeed, 4th should be rate limited
	// But API always returns 204 to prevent enumeration
	for i := range 4 {
		resp, err := testClient.ForgotPassword(org.OwnerEmail)
		s.Require().NoError(err, "request %d should not error", i+1)
		s.Require().Equal(http.StatusNoContent, resp.StatusCode, "request %d should return 204", i+1)
	}

	// Verify only 3 tokens were actually created (rate limit = 3 per hour per member)
	count := s.countTokensForMember(org.MemberID)
	s.Assert().Equal(3, count, "only 3 tokens should be created due to rate limiting")
}

// ==================== GET /api/v1/auth/reset-password/{token} ====================

func (s *PasswordResetSuite) TestPWD010_InvalidToken() {
	testClient := s.c.Clone()
	resp, err := testClient.ValidateResetToken("invalid-token-" + uuid.New().String())
	s.Require().NoError(err)
	s.Require().Equal(http.StatusNotFound, resp.StatusCode)
	s.Assert().Contains(strings.ToLower(string(resp.RawBody)), "invalid token")
}

func (s *PasswordResetSuite) TestPWD011_ValidToken() {
	// Create a new organization for this test
	org := fixtures.NewOrganization(s.T(), s.c).Create()

	// Request password reset
	testClient := s.c.Clone()
	resp, err := testClient.ForgotPassword(org.OwnerEmail)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusNoContent, resp.StatusCode)

	// Get token from DB
	token := s.getLastTokenFromDB(org.MemberID)
	s.Require().NotEmpty(token, "token should exist")

	// Validate token
	validateResp, err := testClient.ValidateResetToken(token)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusNoContent, validateResp.StatusCode)
}

func (s *PasswordResetSuite) TestPWD012_UsedToken() {
	// Create a new organization for this test
	org := fixtures.NewOrganization(s.T(), s.c).Create()

	// Create and use a token
	testClient := s.c.Clone()
	_, err := testClient.ForgotPassword(org.OwnerEmail)
	s.Require().NoError(err)

	token := s.getLastTokenFromDB(org.MemberID)
	s.Require().NotEmpty(token)

	// Use the token by resetting password
	_, err = testClient.ResetPassword(token, "newpassword123")
	s.Require().NoError(err)

	// Try to validate the used token
	validateResp, err := testClient.ValidateResetToken(token)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusGone, validateResp.StatusCode)
	s.Assert().Contains(strings.ToLower(string(validateResp.RawBody)), "token has already been used")
}

func (s *PasswordResetSuite) TestPWD013_ExpiredToken() {
	// Create a new organization for this test
	org := fixtures.NewOrganization(s.T(), s.c).Create()

	// Create an expired token directly in DB
	token := s.createExpiredToken(org.MemberID)

	// Try to validate the expired token
	testClient := s.c.Clone()
	validateResp, err := testClient.ValidateResetToken(token)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusGone, validateResp.StatusCode)
	s.Assert().Contains(strings.ToLower(string(validateResp.RawBody)), "token has expired")
}

// ==================== POST /api/v1/auth/reset-password ====================

func (s *PasswordResetSuite) TestPWD020_EmptyToken() {
	testClient := s.c.Clone()
	resp, err := testClient.ResetPassword("", "newpassword123")
	s.Require().NoError(err)
	s.Require().Equal(http.StatusBadRequest, resp.StatusCode)
	s.Assert().Contains(strings.ToLower(string(resp.RawBody)), "token is required")
}

func (s *PasswordResetSuite) TestPWD021_EmptyPassword() {
	testClient := s.c.Clone()
	resp, err := testClient.ResetPassword("some-token", "")
	s.Require().NoError(err)
	s.Require().Equal(http.StatusBadRequest, resp.StatusCode)
	s.Assert().Contains(strings.ToLower(string(resp.RawBody)), "new_password is required")
}

func (s *PasswordResetSuite) TestPWD022_InvalidToken() {
	testClient := s.c.Clone()
	resp, err := testClient.ResetPassword("invalid-token-"+uuid.New().String(), "newpassword123")
	s.Require().NoError(err)
	s.Require().Equal(http.StatusBadRequest, resp.StatusCode)
	s.Assert().Contains(strings.ToLower(string(resp.RawBody)), "invalid or expired token")
}

func (s *PasswordResetSuite) TestPWD023_ShortPassword() {
	// Create a new organization for this test
	org := fixtures.NewOrganization(s.T(), s.c).Create()

	// Create token
	testClient := s.c.Clone()
	_, err := testClient.ForgotPassword(org.OwnerEmail)
	s.Require().NoError(err)

	token := s.getLastTokenFromDB(org.MemberID)
	s.Require().NotEmpty(token)

	// Try to reset with short password
	resp, err := testClient.ResetPassword(token, "short")
	s.Require().NoError(err)
	s.Require().Equal(http.StatusBadRequest, resp.StatusCode)
	s.Assert().Contains(strings.ToLower(string(resp.RawBody)), "password must be at least 8 characters")
}

func (s *PasswordResetSuite) TestPWD024_SuccessfulReset() {
	// Create a new organization for this test
	org := fixtures.NewOrganization(s.T(), s.c).Create()

	// Create token
	testClient := s.c.Clone()
	_, err := testClient.ForgotPassword(org.OwnerEmail)
	s.Require().NoError(err)

	token := s.getLastTokenFromDB(org.MemberID)
	s.Require().NotEmpty(token)

	// Reset password
	newPassword := "newpassword123"
	resp, err := testClient.ResetPassword(token, newPassword)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusNoContent, resp.StatusCode)

	// Verify old password no longer works
	loginResp, err := testClient.Login(org.OwnerEmail, org.OwnerPassword)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusUnauthorized, loginResp.StatusCode, "old password should not work")

	// Verify new password works
	loginResp, err = testClient.Login(org.OwnerEmail, newPassword)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusOK, loginResp.StatusCode, "new password should work")
}

func (s *PasswordResetSuite) TestPWD025_ReuseToken() {
	// Create a new organization for this test
	org := fixtures.NewOrganization(s.T(), s.c).Create()

	// Create token
	testClient := s.c.Clone()
	_, err := testClient.ForgotPassword(org.OwnerEmail)
	s.Require().NoError(err)

	token := s.getLastTokenFromDB(org.MemberID)
	s.Require().NotEmpty(token)

	// Reset password first time
	resp, err := testClient.ResetPassword(token, "newpassword123")
	s.Require().NoError(err)
	s.Require().Equal(http.StatusNoContent, resp.StatusCode)

	// Try to reuse the same token
	resp, err = testClient.ResetPassword(token, "anotherpassword123")
	s.Require().NoError(err)
	s.Require().Equal(http.StatusBadRequest, resp.StatusCode)
	s.Assert().Contains(strings.ToLower(string(resp.RawBody)), "token has already been used")
}

func (s *PasswordResetSuite) TestPWD026_LoginWithNewPassword() {
	// Create a new organization for this test
	org := fixtures.NewOrganization(s.T(), s.c).Create()

	// Create token
	testClient := s.c.Clone()
	_, err := testClient.ForgotPassword(org.OwnerEmail)
	s.Require().NoError(err)

	token := s.getLastTokenFromDB(org.MemberID)
	s.Require().NotEmpty(token)

	// Reset password
	newPassword := "verysecurepassword123"
	resp, err := testClient.ResetPassword(token, newPassword)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusNoContent, resp.StatusCode)

	// Login with new password and verify user data
	loginResp, err := testClient.Login(org.OwnerEmail, newPassword)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusOK, loginResp.StatusCode)
	s.Assert().Equal(org.MemberID, loginResp.Body.MemberID, "member ID should match")
	s.Assert().Equal(org.OwnerEmail, loginResp.Body.Email, "email should match")
}

// ==================== Helper Methods ====================

// getLastTokenFromDB retrieves the most recent token for a member from the database.
func (s *PasswordResetSuite) getLastTokenFromDB(memberID uuid.UUID) string {
	s.T().Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Wait a bit for async processing
	time.Sleep(50 * time.Millisecond)

	var token string
	err := s.suite.Factory.DB().QueryRow(ctx, `
		SELECT token FROM password_reset_tokens
		WHERE member_id = $1
		ORDER BY created_at DESC
		LIMIT 1
	`, memberID).Scan(&token)
	if err != nil {
		return ""
	}
	return token
}

// countTokensForMember counts all tokens for a member in the database.
func (s *PasswordResetSuite) countTokensForMember(memberID uuid.UUID) int {
	s.T().Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Wait a bit for async processing
	time.Sleep(50 * time.Millisecond)

	var count int
	err := s.suite.Factory.DB().QueryRow(ctx, `
		SELECT COUNT(*) FROM password_reset_tokens
		WHERE member_id = $1
	`, memberID).Scan(&count)
	if err != nil {
		s.T().Fatalf("failed to count tokens: %v", err)
	}
	return count
}

// createExpiredToken creates an expired token directly in the database for testing.
func (s *PasswordResetSuite) createExpiredToken(memberID uuid.UUID) string {
	s.T().Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Generate a random token
	token := uuid.New().String() + uuid.New().String()
	token = strings.ReplaceAll(token, "-", "")[:64] // 64 hex chars

	// Insert expired token (expired 1 hour ago)
	_, err := s.suite.Factory.DB().Exec(ctx, `
		INSERT INTO password_reset_tokens (member_id, token, expires_at)
		VALUES ($1, $2, NOW() - INTERVAL '1 hour')
	`, memberID, token)
	if err != nil {
		s.T().Fatalf("failed to create expired token: %v", err)
	}

	return token
}
