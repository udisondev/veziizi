package tests

import (
	"net/http"
	"testing"

	"codeberg.org/udison/veziizi/backend/e2e/fixtures"
	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
)

// HistorySuite combines all history tests with shared context.
type HistorySuite struct {
	suite.Suite
	baseURL string
	ctx     *fixtures.TestContext

	// Other organization for access tests
	otherOrg *fixtures.CreatedOrganization

	// Shared entities
	sharedFR      *fixtures.CreatedFreightRequest
	sharedOrderID uuid.UUID
}

func TestHistorySuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(HistorySuite))
}

func (s *HistorySuite) SetupSuite() {
	testSuite := getSuite(s.T())
	s.baseURL = testSuite.BaseURL
	s.ctx = fixtures.NewTestContext(s.T(), s.baseURL)
	s.otherOrg = s.ctx.QuickCustomer()

	// Create shared freight request
	s.sharedFR = fixtures.NewFreightRequest(s.T(), s.ctx.Customer.Client).Create()

	// Create shared confirmed order
	_, _, s.sharedOrderID = s.ctx.CreateConfirmedOrder()
}

// ==================== GET /api/v1/organizations/{id}/history ====================

func (s *HistorySuite) TestHIST001_OwnOrganizationHistory() {
	resp, err := s.ctx.Customer.Client.GetOrganizationHistory(s.ctx.Customer.OrganizationID, 20, 0)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusOK, resp.StatusCode, string(resp.RawBody))
}

func (s *HistorySuite) TestHIST002_Pagination() {
	resp, err := s.ctx.Customer.Client.GetOrganizationHistory(s.ctx.Customer.OrganizationID, 5, 0)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusOK, resp.StatusCode, string(resp.RawBody))
}

func (s *HistorySuite) TestHIST003_WithoutAuth() {
	// Handler checks role first, returns 403 even for anon
	resp, err := s.ctx.AnonClient.GetOrganizationHistory(s.ctx.Customer.OrganizationID, 20, 0)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusForbidden, resp.StatusCode)
}

func (s *HistorySuite) TestHIST004_OtherOrganization() {
	resp, err := s.otherOrg.Client.GetOrganizationHistory(s.ctx.Customer.OrganizationID, 20, 0)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusForbidden, resp.StatusCode)
}

// ==================== GET /api/v1/freight-requests/{id}/history ====================

func (s *HistorySuite) TestHIST005_FreightRequestHistory() {
	resp, err := s.ctx.Customer.Client.GetFreightRequestHistory(s.sharedFR.ID, 20, 0)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusOK, resp.StatusCode, string(resp.RawBody))
}

func (s *HistorySuite) TestHIST006_FRHistoryWithoutAuth() {
	resp, err := s.ctx.AnonClient.GetFreightRequestHistory(s.sharedFR.ID, 20, 0)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusUnauthorized, resp.StatusCode)
}

func (s *HistorySuite) TestHIST007_NonexistentFreightRequest() {
	resp, err := s.ctx.Customer.Client.GetFreightRequestHistory(uuid.New(), 20, 0)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusNotFound, resp.StatusCode)
}

func (s *HistorySuite) TestHIST007b_OtherOrgFRHistory() {
	// Carrier cannot see customer's FR history
	resp, err := s.otherOrg.Client.GetFreightRequestHistory(s.sharedFR.ID, 20, 0)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusForbidden, resp.StatusCode)
}

// ==================== GET /api/v1/orders/{id}/history ====================

func (s *HistorySuite) TestHIST008_OrderHistoryAsParticipant() {
	resp, err := s.ctx.Customer.Client.GetOrderHistory(s.sharedOrderID, 20, 0)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusOK, resp.StatusCode, string(resp.RawBody))
}

func (s *HistorySuite) TestHIST009_OrderHistoryWithoutAuth() {
	resp, err := s.ctx.AnonClient.GetOrderHistory(s.sharedOrderID, 20, 0)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusUnauthorized, resp.StatusCode)
}

func (s *HistorySuite) TestHIST010_OrderHistoryNonParticipant() {
	resp, err := s.otherOrg.Client.GetOrderHistory(s.sharedOrderID, 20, 0)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusForbidden, resp.StatusCode)
}

// ==================== Member role access ====================

func (s *HistorySuite) TestMemberRoleAccessToHistory() {
	// Add a regular employee to customer org
	memberClient := s.ctx.AddMemberToOrg(s.ctx.Customer, "employee")

	// Member should not be able to access history
	resp, err := memberClient.GetOrganizationHistory(s.ctx.Customer.OrganizationID, 20, 0)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusForbidden, resp.StatusCode)
}
