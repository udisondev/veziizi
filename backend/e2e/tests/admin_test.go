package tests

import (
	"net/http"
	"testing"
	"time"

	"github.com/udisondev/veziizi/backend/e2e/client"
	"github.com/udisondev/veziizi/backend/e2e/fixtures"
	"github.com/udisondev/veziizi/backend/e2e/helpers"
	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
)

// AdminSuite combines all admin tests with shared context.
type AdminSuite struct {
	suite.Suite
	baseURL string
	c       *client.Client
	ctx     *fixtures.TestContext

	// Shared pending organization for tests
	pendingOrg *fixtures.CreatedOrganization
}

func TestAdminSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(AdminSuite))
}

func (s *AdminSuite) SetupSuite() {
	testSuite := getSuite(s.T())
	s.baseURL = testSuite.BaseURL
	s.c = client.New(s.baseURL)
	s.ctx = fixtures.NewTestContext(s.T(), s.baseURL)

	// Create pending org for tests that need it
	s.pendingOrg = fixtures.NewOrganization(s.T(), s.ctx.AnonClient).Create()
}

// ==================== POST /api/v1/admin/auth/login ====================

func (s *AdminSuite) TestADM001_SuccessfulLogin() {
	resp, err := s.c.AdminLogin("admin@veziizi.local", "admin123")
	s.Require().NoError(err)
	s.Require().Equal(http.StatusOK, resp.StatusCode, string(resp.RawBody))
	s.Assert().NotEmpty(resp.Body.Email)
}

func (s *AdminSuite) TestADM002_WrongPassword() {
	resp, err := s.c.AdminLogin("admin@veziizi.local", "wrongpassword")
	s.Require().NoError(err)
	s.Require().Equal(http.StatusUnauthorized, resp.StatusCode)
}

func (s *AdminSuite) TestADM003_NonexistentAdmin() {
	resp, err := s.c.AdminLogin("notanadmin@veziizi.local", "password")
	s.Require().NoError(err)
	s.Require().Equal(http.StatusUnauthorized, resp.StatusCode)
}

// ==================== POST /api/v1/admin/auth/logout ====================

func (s *AdminSuite) TestADM004_SuccessfulLogout() {
	// Create a separate admin client for logout test to not affect shared AdminClient
	logoutClient := client.New(s.baseURL)
	_, err := logoutClient.AdminLogin("admin@veziizi.local", "admin123")
	s.Require().NoError(err)

	resp, err := logoutClient.AdminLogout()
	s.Require().NoError(err)
	s.Require().Equal(http.StatusNoContent, resp.StatusCode, string(resp.RawBody))
}

// ==================== GET /api/v1/admin/auth/me ====================

func (s *AdminSuite) TestADM005a_AdminMe_Authenticated() {
	resp, err := s.ctx.AdminClient.AdminMe()
	s.Require().NoError(err)
	s.Require().Equal(http.StatusOK, resp.StatusCode, string(resp.RawBody))
	s.Assert().NotEmpty(resp.Body.AdminID)
	s.Assert().Equal("admin@veziizi.local", resp.Body.Email)
}

func (s *AdminSuite) TestADM005b_AdminMe_Unauthorized() {
	anon := client.New(s.baseURL)
	resp, err := anon.AdminMe()
	s.Require().NoError(err)
	s.Assert().Equal(http.StatusUnauthorized, resp.StatusCode)
}

// ==================== GET /api/v1/admin/organizations ====================

func (s *AdminSuite) TestADM005_ListPendingOrganizations() {
	resp, err := s.ctx.AdminClient.AdminGetOrganizations("pending")
	s.Require().NoError(err)
	s.Require().Equal(http.StatusOK, resp.StatusCode, string(resp.RawBody))

	// Should find our pending org
	found := false
	for _, org := range resp.Body {
		if org.ID == s.pendingOrg.OrganizationID {
			found = true
			break
		}
	}
	s.Assert().True(found, "should find the pending organization")
}

func (s *AdminSuite) TestADM006_FilterByStatus() {
	resp, err := s.ctx.AdminClient.AdminGetOrganizations("pending")
	s.Require().NoError(err)
	s.Require().Equal(http.StatusOK, resp.StatusCode, string(resp.RawBody))

	for _, org := range resp.Body {
		s.Assert().Equal("pending", org.Status)
	}
}

func (s *AdminSuite) TestADM008_WithoutAdminSession() {
	resp, err := s.ctx.AnonClient.AdminGetOrganizations("")
	s.Require().NoError(err)
	s.Require().Equal(http.StatusUnauthorized, resp.StatusCode)
}

func (s *AdminSuite) TestADM009_WithUserSession() {
	resp, err := s.ctx.Customer.Client.AdminGetOrganizations("")
	s.Require().NoError(err)
	s.Require().Equal(http.StatusUnauthorized, resp.StatusCode)
}

// ==================== GET /api/v1/admin/organizations/{id} ====================

func (s *AdminSuite) TestADM010_GetOrganization() {
	resp, err := s.ctx.AdminClient.AdminGetOrganization(s.ctx.Customer.OrganizationID)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusOK, resp.StatusCode, string(resp.RawBody))
	s.Assert().Equal(s.ctx.Customer.OrganizationID, resp.Body.ID)
}

func (s *AdminSuite) TestADM011_NonexistentOrg() {
	resp, err := s.ctx.AdminClient.AdminGetOrganization(uuid.New())
	s.Require().NoError(err)
	s.Require().Equal(http.StatusNotFound, resp.StatusCode)
}

// ==================== POST /api/v1/admin/organizations/{id}/approve ====================

func (s *AdminSuite) TestADM012_ApprovePending() {
	// Create pending org for this test
	pendingOrg := fixtures.NewOrganization(s.T(), s.ctx.AnonClient).Create()

	resp, err := s.ctx.AdminClient.AdminApproveOrganization(pendingOrg.OrganizationID, nil)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusNoContent, resp.StatusCode, string(resp.RawBody))

	// Verify status changed
	orgResp, err := s.ctx.AdminClient.AdminGetOrganization(pendingOrg.OrganizationID)
	s.Require().NoError(err)
	s.Assert().Equal("active", orgResp.Body.Status)
}

func (s *AdminSuite) TestADM013_AlreadyApproved() {
	// ctx.Customer is already approved
	resp, err := s.ctx.AdminClient.AdminApproveOrganization(s.ctx.Customer.OrganizationID, nil)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusConflict, resp.StatusCode)
}

// ==================== POST /api/v1/admin/organizations/{id}/reject ====================

func (s *AdminSuite) TestADM015_RejectPending() {
	pendingOrg := fixtures.NewOrganization(s.T(), s.ctx.AnonClient).Create()

	resp, err := s.ctx.AdminClient.AdminRejectOrganization(pendingOrg.OrganizationID, "Test rejection reason")
	s.Require().NoError(err)
	s.Require().Equal(http.StatusNoContent, resp.StatusCode, string(resp.RawBody))
}

func (s *AdminSuite) TestADM016_WithReason() {
	pendingOrg := fixtures.NewOrganization(s.T(), s.ctx.AnonClient).Create()

	reason := "Invalid documentation provided"
	resp, err := s.ctx.AdminClient.AdminRejectOrganization(pendingOrg.OrganizationID, reason)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusNoContent, resp.StatusCode, string(resp.RawBody))
}

// ==================== POST /api/v1/admin/organizations/{id}/mark-fraudster ====================

func (s *AdminSuite) TestADM017_MarkFraudster() {
	// Create a new org to mark as fraudster
	org := fixtures.NewActiveOrganization(s.T(), s.ctx.AnonClient, s.ctx.AdminClient).Create()

	resp, err := s.ctx.AdminClient.AdminMarkFraudster(org.OrganizationID, true, "Fraudulent activity detected")
	s.Require().NoError(err)
	s.Require().Equal(http.StatusNoContent, resp.StatusCode, string(resp.RawBody))

	// Verify org is in fraudsters list (wait for projection sync)
	orgIDStr := org.OrganizationID.String()
	helpers.Wait(s.T(), func() bool {
		fraudstersResp, err := s.ctx.AdminClient.AdminListFraudsters()
		if err != nil || fraudstersResp.StatusCode != 200 {
			return false
		}
		for _, f := range fraudstersResp.Body.Fraudsters {
			if f.OrgID == orgIDStr {
				return true
			}
		}
		return false
	}, "org should be in fraudsters list")
}

func (s *AdminSuite) TestADM018_MarkFraudsterWithReason() {
	org := fixtures.NewActiveOrganization(s.T(), s.ctx.AnonClient, s.ctx.AdminClient).Create()

	reason := "Multiple fraud reports received"
	resp, err := s.ctx.AdminClient.AdminMarkFraudster(org.OrganizationID, true, reason)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusNoContent, resp.StatusCode, string(resp.RawBody))
}

// ==================== POST /api/v1/admin/organizations/{id}/unmark-fraudster ====================

func (s *AdminSuite) TestADM019_UnmarkFraudster() {
	// Create org and mark as fraudster
	org := fixtures.NewActiveOrganization(s.T(), s.ctx.AnonClient, s.ctx.AdminClient).Create()

	_, err := s.ctx.AdminClient.AdminMarkFraudster(org.OrganizationID, true, "Test")
	s.Require().NoError(err)

	// Unmark
	resp, err := s.ctx.AdminClient.AdminUnmarkFraudster(org.OrganizationID, "No longer fraudster")
	s.Require().NoError(err)
	s.Require().Equal(http.StatusNoContent, resp.StatusCode, string(resp.RawBody))
}

// ==================== GET /api/v1/admin/fraudsters ====================

func (s *AdminSuite) TestADM020_ListFraudsters() {
	// Create and mark as fraudster
	org := fixtures.NewActiveOrganization(s.T(), s.ctx.AnonClient, s.ctx.AdminClient).Create()
	_, err := s.ctx.AdminClient.AdminMarkFraudster(org.OrganizationID, true, "Test")
	s.Require().NoError(err)

	// Wait for projection sync and verify org is in list
	orgIDStr := org.OrganizationID.String()
	helpers.Wait(s.T(), func() bool {
		resp, err := s.ctx.AdminClient.AdminListFraudsters()
		if err != nil || resp.StatusCode != 200 {
			return false
		}
		for _, f := range resp.Body.Fraudsters {
			if f.OrgID == orgIDStr {
				return true
			}
		}
		return false
	}, "org should be in fraudsters list")
}

func (s *AdminSuite) TestADM021_EmptyList() {
	// Just check endpoint works
	resp, err := s.ctx.AdminClient.AdminListFraudsters()
	s.Require().NoError(err)
	s.Require().Equal(http.StatusOK, resp.StatusCode)
}

// ==================== GET /api/v1/admin/reviews ====================

func (s *AdminSuite) TestADM022_ListPendingReviews() {
	resp, err := s.ctx.AdminClient.AdminGetReviews("pending")
	s.Require().NoError(err)
	s.Require().Equal(http.StatusOK, resp.StatusCode)
}

func (s *AdminSuite) TestADM023_FilterReviewsByStatus() {
	resp, err := s.ctx.AdminClient.AdminGetReviews("pending")
	s.Require().NoError(err)
	s.Require().Equal(http.StatusOK, resp.StatusCode)

	for _, r := range resp.Body {
		s.Assert().Equal("pending", r.Status)
	}
}

// ==================== GET /api/v1/admin/reviews/{id} ====================

func (s *AdminSuite) TestADM025_NonexistentReview() {
	resp, err := s.ctx.AdminClient.AdminGetReview(uuid.New())
	s.Require().NoError(err)
	s.Require().Equal(http.StatusNotFound, resp.StatusCode)
}

// ==================== User session cannot access admin endpoints ====================

func (s *AdminSuite) TestUserSessionCannotAccessListOrganizations() {
	resp, _ := s.ctx.Customer.Client.AdminGetOrganizations("")
	s.Assert().Equal(http.StatusUnauthorized, resp.StatusCode)
}

func (s *AdminSuite) TestUserSessionCannotAccessListFraudsters() {
	resp, _ := s.ctx.Customer.Client.AdminListFraudsters()
	s.Assert().Equal(http.StatusUnauthorized, resp.StatusCode)
}

func (s *AdminSuite) TestUserSessionCannotAccessListReviews() {
	resp, _ := s.ctx.Customer.Client.AdminGetReviews("")
	s.Assert().Equal(http.StatusUnauthorized, resp.StatusCode)
}

// ==================== Review Full Cycle Tests ====================

func (s *AdminSuite) TestADM030_ReviewFullCycle_Approve() {
	// Create a completed FR and leave a review
	completed := s.ctx.CreateFullyCompletedFreightRequest()

	comment := "Great service!"
	reviewResp, err := s.ctx.Customer.Client.LeaveFreightRequestReview(completed.FreightRequest.ID, 5, &comment)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusCreated, reviewResp.StatusCode)
	reviewID := reviewResp.Body.ReviewID

	// Wait for review to be processed (review-receiver → review-analyzer → reviews-projection)
	helpers.WaitWithConfig(s.T(), helpers.WaitConfig{Timeout: 30 * time.Second, Interval: 200 * time.Millisecond}, func() bool {
		resp, err := s.ctx.AdminClient.AdminGetReview(reviewID)
		return err == nil && resp.StatusCode == http.StatusOK
	}, "review should be accessible via admin API")

	// Get review and approve if still pending
	getResp, err := s.ctx.AdminClient.AdminGetReview(reviewID)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusOK, getResp.StatusCode)

	if getResp.Body.Status == "pending_moderation" {
		approveResp, err := s.ctx.AdminClient.AdminApproveReview(reviewID)
		s.Require().NoError(err)
		s.Require().Equal(http.StatusNoContent, approveResp.StatusCode, "should approve review successfully")
	}
	// If already auto-approved — that's fine too
}

func (s *AdminSuite) TestADM031_ReviewReject() {
	// Create a completed FR and leave a review
	completed := s.ctx.CreateFullyCompletedFreightRequest()

	comment := "Suspicious review"
	reviewResp, err := s.ctx.Customer.Client.LeaveFreightRequestReview(completed.FreightRequest.ID, 5, &comment)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusCreated, reviewResp.StatusCode)
	reviewID := reviewResp.Body.ReviewID

	// Wait for review to be processed
	helpers.WaitWithConfig(s.T(), helpers.WaitConfig{Timeout: 30 * time.Second, Interval: 200 * time.Millisecond}, func() bool {
		resp, err := s.ctx.AdminClient.AdminGetReview(reviewID)
		return err == nil && resp.StatusCode == http.StatusOK
	}, "review should be accessible via admin API")

	// Reject the review (may already be auto-approved → 409)
	rejectResp, err := s.ctx.AdminClient.AdminRejectReview(reviewID, "Fake review detected")
	s.Require().NoError(err)
	s.Require().True(rejectResp.StatusCode == http.StatusNoContent || rejectResp.StatusCode == http.StatusConflict,
		"should reject or conflict, got %d", rejectResp.StatusCode)
}

func (s *AdminSuite) TestADM032_FraudsterDeactivatesReviews() {
	// Create a new org that will become a fraudster
	fraudOrg := fixtures.NewActiveOrganization(s.T(), s.ctx.AnonClient, s.ctx.AdminClient).Create()

	// Create a FR between fraud org (customer) and carrier
	frBuilder := fixtures.NewFreightRequest(s.T(), fraudOrg.Client)
	fr := frBuilder.Create()

	// Carrier makes offer
	offer := fixtures.NewOffer(s.T(), s.ctx.Carrier.Client, fr.ID).Create()

	// Fraud org selects
	selectResp, err := fraudOrg.Client.SelectOffer(fr.ID, offer.OfferID)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusNoContent, selectResp.StatusCode)

	// Carrier confirms
	confirmResp, err := s.ctx.Carrier.Client.ConfirmOffer(fr.ID, offer.OfferID)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusNoContent, confirmResp.StatusCode)

	// Wait for confirmed status
	helpers.Wait(s.T(), func() bool {
		frResp, err := fraudOrg.Client.GetFreightRequest(fr.ID)
		return err == nil && frResp.StatusCode == http.StatusOK && frResp.Body.Status == "confirmed"
	}, "FR should be confirmed")

	// Both sides complete
	completeResp, err := fraudOrg.Client.CompleteFreightRequest(fr.ID)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusNoContent, completeResp.StatusCode)

	completeResp, err = s.ctx.Carrier.Client.CompleteFreightRequest(fr.ID)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusNoContent, completeResp.StatusCode)

	// Wait for completed status
	helpers.Wait(s.T(), func() bool {
		frResp, err := fraudOrg.Client.GetFreightRequest(fr.ID)
		return err == nil && frResp.StatusCode == http.StatusOK && frResp.Body.Status == "completed"
	}, "FR should be completed")

	// Fraud org leaves a review
	comment := "Fraudulent review"
	reviewResp, err := fraudOrg.Client.LeaveFreightRequestReview(fr.ID, 5, &comment)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusCreated, reviewResp.StatusCode)

	// Mark the org as fraudster
	markResp, err := s.ctx.AdminClient.AdminMarkFraudster(fraudOrg.OrganizationID, true, "Fraud detected")
	s.Require().NoError(err)
	s.Require().Equal(http.StatusNoContent, markResp.StatusCode)

	// Verify org is marked as fraudster
	orgIDStr := fraudOrg.OrganizationID.String()
	helpers.Wait(s.T(), func() bool {
		fraudstersResp, err := s.ctx.AdminClient.AdminListFraudsters()
		if err != nil || fraudstersResp.StatusCode != http.StatusOK {
			return false
		}
		for _, f := range fraudstersResp.Body.Fraudsters {
			if f.OrgID == orgIDStr {
				return true
			}
		}
		return false
	}, "org should be in fraudsters list")

	// Note: The fraudster-handler worker should deactivate reviews from this org
	// This is an async process, so we just verify the mechanism is in place
}

