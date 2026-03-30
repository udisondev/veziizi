package tests

import (
	"net/http"
	"strings"
	"testing"

	"github.com/udisondev/veziizi/backend/e2e/client"
	"github.com/udisondev/veziizi/backend/e2e/fixtures"
	"github.com/udisondev/veziizi/backend/e2e/helpers"
	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
)

// OrganizationsSuite combines all organization tests with shared context.
type OrganizationsSuite struct {
	suite.Suite
	baseURL string
	c       *client.Client // Anonymous client for registration tests

	// Shared organization for tests that need an existing org
	org      *fixtures.CreatedOrganization
	otherOrg *fixtures.CreatedOrganization
}

func TestOrganizationsSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(OrganizationsSuite))
}

func (s *OrganizationsSuite) SetupSuite() {
	testSuite := getSuite(s.T())
	s.baseURL = testSuite.BaseURL
	s.c = client.New(s.baseURL)

	// Create shared organizations once
	s.org = fixtures.NewOrganization(s.T(), s.c).Create()
	s.otherOrg = fixtures.NewOrganization(s.T(), s.c).Create()
}

// ==================== POST /api/v1/organizations ====================

func (s *OrganizationsSuite) TestORG001_RegisterRUOrganization() {
	builder := fixtures.NewOrganization(s.T(), s.c).WithCountry("RU")
	resp, err := builder.CreateWithStatus()
	s.Require().NoError(err)
	s.Require().Equal(http.StatusCreated, resp.StatusCode, string(resp.RawBody))
	s.Assert().NotEmpty(resp.Body.OrganizationID.String(), "organization_id")
	s.Assert().NotEmpty(resp.Body.MemberID.String(), "member_id")
}

func (s *OrganizationsSuite) TestORG002_RegisterKZOrganization() {
	builder := fixtures.NewOrganization(s.T(), s.c).WithCountry("KZ")
	resp, err := builder.CreateWithStatus()
	s.Require().NoError(err)
	s.Require().Equal(http.StatusCreated, resp.StatusCode, string(resp.RawBody))
}

func (s *OrganizationsSuite) TestORG003_RegisterBYOrganization() {
	builder := fixtures.NewOrganization(s.T(), s.c).WithCountry("BY")
	resp, err := builder.CreateWithStatus()
	s.Require().NoError(err)
	s.Require().Equal(http.StatusCreated, resp.StatusCode, string(resp.RawBody))
}

func (s *OrganizationsSuite) TestORG009_InvalidCountryCode() {
	builder := fixtures.NewOrganization(s.T(), s.c).WithCountry("XX")
	resp, err := builder.CreateWithStatus()
	s.Require().NoError(err)
	s.Require().Equal(http.StatusBadRequest, resp.StatusCode)
	s.Assert().Contains(strings.ToLower(string(resp.RawBody)), "invalid country")
}

func (s *OrganizationsSuite) TestORG010_EmptyCountry() {
	builder := fixtures.NewOrganization(s.T(), s.c).WithCountry("")
	resp, err := builder.CreateWithStatus()
	s.Require().NoError(err)
	s.Require().Equal(http.StatusBadRequest, resp.StatusCode)
}

func (s *OrganizationsSuite) TestORG019_SQLInjectionInName() {
	builder := fixtures.NewOrganization(s.T(), s.c).WithName("'; DROP TABLE--")
	resp, err := builder.CreateWithStatus()
	s.Require().NoError(err)
	s.Require().Equal(http.StatusCreated, resp.StatusCode, string(resp.RawBody))
}

func (s *OrganizationsSuite) TestORG022_UnicodeInData() {
	builder := fixtures.NewOrganization(s.T(), s.c).
		WithName("中文公司 🚛").
		WithAddress("北京市朝阳区")
	resp, err := builder.CreateWithStatus()
	s.Require().NoError(err)
	s.Require().Equal(http.StatusCreated, resp.StatusCode, string(resp.RawBody))
}

// ==================== GET /api/v1/organizations/{id} ====================

func (s *OrganizationsSuite) TestORG026_GetExistingOrganization() {
	resp, err := s.c.GetOrganization(s.org.OrganizationID)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusOK, resp.StatusCode, string(resp.RawBody))
	s.Assert().Equal(s.org.OrganizationID, resp.Body.ID, "id")
	s.Assert().Equal("pending", resp.Body.Status, "status")
}

func (s *OrganizationsSuite) TestORG028_GetWithoutAuth() {
	resp, err := s.c.GetOrganization(s.org.OrganizationID)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusOK, resp.StatusCode, string(resp.RawBody))
}

func (s *OrganizationsSuite) TestORG030_InvalidUUID() {
	status, _, err := s.c.Raw(http.MethodGet, "/api/v1/organizations/not-a-uuid", nil, nil)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusBadRequest, status)
}

func (s *OrganizationsSuite) TestORG031_NonexistentOrganization() {
	resp, err := s.c.GetOrganization(uuid.New())
	s.Require().NoError(err)
	s.Require().Equal(http.StatusNotFound, resp.StatusCode)
}

// ==================== GET /api/v1/organizations/{id}/full ====================

func (s *OrganizationsSuite) TestORG034_GetOwnOrganizationWithMembers() {
	resp, err := s.org.Client.GetOrganizationFull(s.org.OrganizationID)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusOK, resp.StatusCode, string(resp.RawBody))
	s.Assert().True(len(resp.Body.Members) > 0, "should have members")
}

func (s *OrganizationsSuite) TestORG036_FullWithoutAuth() {
	resp, err := s.c.GetOrganizationFull(s.org.OrganizationID)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusUnauthorized, resp.StatusCode)
}

func (s *OrganizationsSuite) TestORG037_FullDifferentOrganization() {
	resp, err := s.otherOrg.Client.GetOrganizationFull(s.org.OrganizationID)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusForbidden, resp.StatusCode)
}

// ==================== GET /api/v1/organizations/{id}/rating ====================

func (s *OrganizationsSuite) TestORG040_RatingWithoutReviews() {
	resp, err := s.c.GetOrganizationRating(s.org.OrganizationID)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusOK, resp.StatusCode, string(resp.RawBody))
	s.Assert().Equal(0, resp.Body.TotalReviews, "total_reviews")
	s.Assert().Equal(0.0, resp.Body.AverageRating, "average_rating")
}

func (s *OrganizationsSuite) TestORG042_RatingPublicAccess() {
	resp, err := s.c.GetOrganizationRating(s.org.OrganizationID)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusOK, resp.StatusCode)
}

// ==================== POST /api/v1/organizations/{id}/invitations ====================

func (s *OrganizationsSuite) TestORG052_CreateAdministratorInvitation() {
	resp, err := s.org.Client.CreateInvitation(s.org.OrganizationID, client.CreateInvitationRequest{
		Email: helpers.RandomEmail(),
		Role:  "administrator",
	})
	s.Require().NoError(err)
	s.Require().Equal(http.StatusCreated, resp.StatusCode, string(resp.RawBody))
}

func (s *OrganizationsSuite) TestORG053_CreateEmployeeInvitation() {
	resp, err := s.org.Client.CreateInvitation(s.org.OrganizationID, client.CreateInvitationRequest{
		Email: helpers.RandomEmail(),
		Role:  "employee",
	})
	s.Require().NoError(err)
	s.Require().Equal(http.StatusCreated, resp.StatusCode, string(resp.RawBody))
}

func (s *OrganizationsSuite) TestORG056_InvalidRole() {
	resp, err := s.org.Client.CreateInvitation(s.org.OrganizationID, client.CreateInvitationRequest{
		Email: helpers.RandomEmail(),
		Role:  "superadmin",
	})
	s.Require().NoError(err)
	s.Require().Equal(http.StatusBadRequest, resp.StatusCode)
	s.Assert().Contains(strings.ToLower(string(resp.RawBody)), "invalid role")
}

func (s *OrganizationsSuite) TestORG060_InvitationWithoutAuth() {
	resp, err := s.c.CreateInvitation(s.org.OrganizationID, client.CreateInvitationRequest{
		Email: helpers.RandomEmail(),
		Role:  "employee",
	})
	s.Require().NoError(err)
	s.Require().Equal(http.StatusUnauthorized, resp.StatusCode)
}

func (s *OrganizationsSuite) TestORG061_InvitationDifferentOrganization() {
	resp, err := s.otherOrg.Client.CreateInvitation(s.org.OrganizationID, client.CreateInvitationRequest{
		Email: helpers.RandomEmail(),
		Role:  "employee",
	})
	s.Require().NoError(err)
	s.Require().Equal(http.StatusNotFound, resp.StatusCode)
	s.Assert().Contains(strings.ToLower(string(resp.RawBody)), "member not found")
}

func (s *OrganizationsSuite) TestORG064_DuplicateInvitation() {
	email := helpers.RandomEmail()

	// First invitation
	resp1, err := s.org.Client.CreateInvitation(s.org.OrganizationID, client.CreateInvitationRequest{
		Email: email,
		Role:  "employee",
	})
	s.Require().NoError(err)
	s.Require().Equal(http.StatusCreated, resp1.StatusCode, string(resp1.RawBody))

	// Second invitation with same email
	resp2, err := s.org.Client.CreateInvitation(s.org.OrganizationID, client.CreateInvitationRequest{
		Email: email,
		Role:  "employee",
	})
	s.Require().NoError(err)
	s.Require().Equal(http.StatusConflict, resp2.StatusCode)
	s.Assert().Contains(strings.ToLower(string(resp2.RawBody)), "already invited")
}

// ==================== POST /api/v1/invitations/{token}/accept ====================

func (s *OrganizationsSuite) TestORG082_SuccessfulAccept() {
	email := helpers.RandomEmail()
	invResp, err := s.org.Client.CreateInvitation(s.org.OrganizationID, client.CreateInvitationRequest{
		Email: email,
		Role:  "employee",
	})
	s.Require().NoError(err)

	// Wait for invitation to be available
	token := invResp.Body.Token
	helpers.WaitFor(s.T(), func() (bool, bool) {
		getResp, err := s.c.GetInvitationByToken(token)
		return err == nil && getResp.StatusCode == 200, err == nil && getResp.StatusCode == 200
	}, "invitation should be available")

	resp, err := s.c.AcceptInvitation(token, client.AcceptInvitationRequest{
		Password: "password123",
		Name:     helpers.StringPtr("New Member"),
		Phone:    helpers.StringPtr("+79001234567"),
	})
	s.Require().NoError(err)
	s.Require().Equal(http.StatusOK, resp.StatusCode, string(resp.RawBody))
}

func (s *OrganizationsSuite) TestORG085_EmptyPassword() {
	email := helpers.RandomEmail()
	invResp, err := s.org.Client.CreateInvitation(s.org.OrganizationID, client.CreateInvitationRequest{
		Email: email,
		Role:  "employee",
	})
	s.Require().NoError(err)

	// Wait for invitation
	token := invResp.Body.Token
	helpers.WaitFor(s.T(), func() (bool, bool) {
		getResp, err := s.c.GetInvitationByToken(token)
		return err == nil && getResp.StatusCode == 200, err == nil && getResp.StatusCode == 200
	}, "invitation should be available")

	resp, err := s.c.AcceptInvitation(token, client.AcceptInvitationRequest{
		Password: "",
	})
	s.Require().NoError(err)
	s.Require().Equal(http.StatusBadRequest, resp.StatusCode)
}

func (s *OrganizationsSuite) TestORG088_NonexistentToken() {
	resp, err := s.c.AcceptInvitation("nonexistent-token", client.AcceptInvitationRequest{
		Password: "password123",
		Name:     helpers.StringPtr("Test"),
		Phone:    helpers.StringPtr("+79001234567"),
	})
	s.Require().NoError(err)
	s.Require().Equal(http.StatusNotFound, resp.StatusCode)
}

// ==================== Block/Unblock Member ====================

func (s *OrganizationsSuite) TestORG101_OwnerBlocksMember() {
	// Create and accept invitation to have a second member
	email := helpers.RandomEmail()
	invResp, err := s.org.Client.CreateInvitation(s.org.OrganizationID, client.CreateInvitationRequest{
		Email: email,
		Role:  "employee",
	})
	s.Require().NoError(err)

	token := invResp.Body.Token
	helpers.WaitFor(s.T(), func() (bool, bool) {
		getResp, err := s.c.GetInvitationByToken(token)
		return err == nil && getResp.StatusCode == 200, err == nil && getResp.StatusCode == 200
	}, "invitation should be available")

	name := "Member to Block"
	phone := "+79001234567"
	acceptResp, err := s.c.AcceptInvitation(token, client.AcceptInvitationRequest{
		Password: "password123",
		Name:     &name,
		Phone:    &phone,
	})
	s.Require().NoError(err)
	memberID := acceptResp.Body.MemberID

	// Wait for member to be available
	helpers.WaitFor(s.T(), func() (bool, bool) {
		meResp, err := s.c.Login(email, "password123")
		return err == nil && meResp.StatusCode == 200, err == nil && meResp.StatusCode == 200
	}, "member should be available")

	// Block
	resp, err := s.org.Client.BlockMember(s.org.OrganizationID, memberID, "test block reason")
	s.Require().NoError(err)
	s.Require().Equal(http.StatusNoContent, resp.StatusCode, string(resp.RawBody))

	// Unblock
	resp2, err := s.org.Client.UnblockMember(s.org.OrganizationID, memberID)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusNoContent, resp2.StatusCode, string(resp2.RawBody))
}

func (s *OrganizationsSuite) TestORG104_CannotBlockSelf() {
	resp, err := s.org.Client.BlockMember(s.org.OrganizationID, s.org.MemberID, "test block reason")
	s.Require().NoError(err)
	s.Require().Equal(http.StatusBadRequest, resp.StatusCode)
	s.Assert().Contains(strings.ToLower(string(resp.RawBody)), "cannot block yourself")
}
