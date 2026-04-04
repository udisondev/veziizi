package tests

import (
	"net/http"
	"testing"

	"github.com/udisondev/veziizi/backend/e2e/client"
	"github.com/udisondev/veziizi/backend/e2e/fixtures"
	"github.com/stretchr/testify/suite"
)

type EmailNotificationsSuite struct {
	suite.Suite
	baseURL string
	ctx     *fixtures.TestContext
}

func TestEmailNotificationsSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(EmailNotificationsSuite))
}

func (s *EmailNotificationsSuite) SetupSuite() {
	testSuite := getSuite(s.T())
	s.baseURL = testSuite.BaseURL
	s.ctx = fixtures.NewTestContext(s.T(), s.baseURL)
}

// ==================== SET EMAIL ====================

func (s *EmailNotificationsSuite) TestEML001_SetEmail_Success() {
	resp, err := s.ctx.Customer.Client.SetEmail("test-eml001@example.com")
	s.Require().NoError(err)
	s.Assert().Equal(http.StatusNoContent, resp.StatusCode, string(resp.RawBody))
}

func (s *EmailNotificationsSuite) TestEML002_SetEmail_EmptyEmail() {
	resp, err := s.ctx.Customer.Client.SetEmail("")
	s.Require().NoError(err)
	s.Assert().Equal(http.StatusBadRequest, resp.StatusCode)
}

func (s *EmailNotificationsSuite) TestEML003_SetEmail_Unauthorized() {
	anon := client.New(s.baseURL)
	resp, err := anon.SetEmail("test@example.com")
	s.Require().NoError(err)
	s.Assert().Equal(http.StatusUnauthorized, resp.StatusCode)
}

// ==================== DISCONNECT EMAIL ====================

func (s *EmailNotificationsSuite) TestEML010_DisconnectEmail_Success() {
	// Set email first
	setResp, err := s.ctx.Carrier.Client.SetEmail("disconnect-test@example.com")
	s.Require().NoError(err)
	s.Require().Equal(http.StatusNoContent, setResp.StatusCode)

	// Disconnect
	resp, err := s.ctx.Carrier.Client.DisconnectEmail()
	s.Require().NoError(err)
	s.Assert().Equal(http.StatusNoContent, resp.StatusCode)
}

func (s *EmailNotificationsSuite) TestEML011_DisconnectEmail_Idempotent() {
	resp1, err := s.ctx.Carrier.Client.DisconnectEmail()
	s.Require().NoError(err)
	s.Assert().Equal(http.StatusNoContent, resp1.StatusCode)

	resp2, err := s.ctx.Carrier.Client.DisconnectEmail()
	s.Require().NoError(err)
	s.Assert().Equal(http.StatusNoContent, resp2.StatusCode)
}

func (s *EmailNotificationsSuite) TestEML012_DisconnectEmail_Unauthorized() {
	anon := client.New(s.baseURL)
	resp, err := anon.DisconnectEmail()
	s.Require().NoError(err)
	s.Assert().Equal(http.StatusUnauthorized, resp.StatusCode)
}

// ==================== MARKETING CONSENT ====================

func (s *EmailNotificationsSuite) TestEML020_SetMarketingConsent_True() {
	resp, err := s.ctx.Customer.Client.SetMarketingConsent(true)
	s.Require().NoError(err)
	s.Assert().Equal(http.StatusNoContent, resp.StatusCode)
}

func (s *EmailNotificationsSuite) TestEML021_SetMarketingConsent_False() {
	resp, err := s.ctx.Customer.Client.SetMarketingConsent(false)
	s.Require().NoError(err)
	s.Assert().Equal(http.StatusNoContent, resp.StatusCode)
}

func (s *EmailNotificationsSuite) TestEML022_SetMarketingConsent_Unauthorized() {
	anon := client.New(s.baseURL)
	resp, err := anon.SetMarketingConsent(true)
	s.Require().NoError(err)
	s.Assert().Equal(http.StatusUnauthorized, resp.StatusCode)
}

// ==================== RESEND VERIFICATION ====================

func (s *EmailNotificationsSuite) TestEML030_ResendVerification_NoEmailSet() {
	// Use carrier which had email disconnected in previous tests
	// First disconnect to ensure clean state
	_, _ = s.ctx.Carrier.Client.DisconnectEmail()

	resp, err := s.ctx.Carrier.Client.ResendEmailVerification()
	s.Require().NoError(err)
	s.Assert().Equal(http.StatusBadRequest, resp.StatusCode)
}

func (s *EmailNotificationsSuite) TestEML031_ResendVerification_AfterSetEmail() {
	// Set email first
	setResp, err := s.ctx.Customer.Client.SetEmail("resend-test@example.com")
	s.Require().NoError(err)
	s.Require().Equal(http.StatusNoContent, setResp.StatusCode)

	// Resend verification
	resp, err := s.ctx.Customer.Client.ResendEmailVerification()
	s.Require().NoError(err)
	s.Assert().Equal(http.StatusNoContent, resp.StatusCode)
}

func (s *EmailNotificationsSuite) TestEML032_ResendVerification_Unauthorized() {
	anon := client.New(s.baseURL)
	resp, err := anon.ResendEmailVerification()
	s.Require().NoError(err)
	s.Assert().Equal(http.StatusUnauthorized, resp.StatusCode)
}

// ==================== VERIFY EMAIL BY TOKEN ====================

func (s *EmailNotificationsSuite) TestEML040_VerifyEmail_InvalidToken() {
	resp, err := s.ctx.Customer.Client.VerifyEmailByToken("invalid-token-12345")
	s.Require().NoError(err)
	s.Assert().Equal(http.StatusBadRequest, resp.StatusCode)
}

func (s *EmailNotificationsSuite) TestEML041_VerifyEmail_EmptyToken() {
	resp, err := s.ctx.Customer.Client.VerifyEmailByToken("")
	s.Require().NoError(err)
	s.Assert().Equal(http.StatusBadRequest, resp.StatusCode)
}

// ==================== FULL FLOW ====================

func (s *EmailNotificationsSuite) TestEML050_FullFlow_SetAndDisconnect() {
	// Use a fresh org to avoid rate limits from previous tests
	org := fixtures.NewActiveOrganization(s.T(), client.New(s.baseURL), s.ctx.AdminClient).Create()

	// Set email
	setResp, err := org.Client.SetEmail("flow-test@example.com")
	s.Require().NoError(err)
	s.Require().Equal(http.StatusNoContent, setResp.StatusCode)

	// Set marketing consent
	consentResp, err := org.Client.SetMarketingConsent(true)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusNoContent, consentResp.StatusCode)

	// Disconnect email
	disconnectResp, err := org.Client.DisconnectEmail()
	s.Require().NoError(err)
	s.Require().Equal(http.StatusNoContent, disconnectResp.StatusCode)

	// Resend verification should fail after disconnect
	resendResp, err := org.Client.ResendEmailVerification()
	s.Require().NoError(err)
	s.Assert().Equal(http.StatusBadRequest, resendResp.StatusCode, "should fail after email disconnected")
}
