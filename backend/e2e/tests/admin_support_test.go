package tests

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
	"github.com/udisondev/veziizi/backend/e2e/fixtures"
	"github.com/udisondev/veziizi/backend/internal/domain/support/entities"
)

// AdminSupportSuite combines all admin support tests with shared context.
type AdminSupportSuite struct {
	suite.Suite
	baseURL string
	ctx     *fixtures.TestContext

	// Shared ticket for tests
	sharedTicketID uuid.UUID
}

func TestAdminSupportSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(AdminSupportSuite))
}

func (s *AdminSupportSuite) SetupSuite() {
	testSuite := getSuite(s.T())
	s.baseURL = testSuite.BaseURL
	s.ctx = fixtures.NewTestContext(s.T(), s.baseURL)

	// Create a shared ticket for tests
	ticketResp, err := s.ctx.Customer.Client.CreateTicket("Shared ticket for admin tests", "Test message")
	s.Require().NoError(err)
	s.Require().Equal(http.StatusCreated, ticketResp.StatusCode)
	s.sharedTicketID = ticketResp.Body.ID
}

// ==================== GET /api/v1/admin/support/tickets ====================

func (s *AdminSupportSuite) TestASUP001_ListTickets() {
	resp, err := s.ctx.AdminClient.AdminListTickets("", 20, 0)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusOK, resp.StatusCode, string(resp.RawBody))
	s.Assert().GreaterOrEqual(resp.Body.Total, 0)
}

func (s *AdminSupportSuite) TestASUP002_FilterByStatus() {
	resp, err := s.ctx.AdminClient.AdminListTickets("open", 20, 0)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusOK, resp.StatusCode, string(resp.RawBody))

	for _, ticket := range resp.Body.Tickets {
		s.Assert().Equal("open", ticket.Status)
	}
}

func (s *AdminSupportSuite) TestASUP003_Pagination() {
	resp, err := s.ctx.AdminClient.AdminListTickets("", 5, 0)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusOK, resp.StatusCode, string(resp.RawBody))
}

func (s *AdminSupportSuite) TestASUP004_WithoutAdminSession() {
	resp, err := s.ctx.AnonClient.AdminListTickets("", 20, 0)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusUnauthorized, resp.StatusCode)
}

// ==================== GET /api/v1/admin/support/tickets/{id} ====================

func (s *AdminSupportSuite) TestASUP005_GetTicket() {
	resp, err := s.ctx.AdminClient.AdminGetTicket(s.sharedTicketID)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusOK, resp.StatusCode, string(resp.RawBody))
	s.Assert().Equal(s.sharedTicketID, resp.Body.ID)
	s.Assert().NotEmpty(resp.Body.Subject)
	s.Assert().NotEmpty(resp.Body.Messages)
}

func (s *AdminSupportSuite) TestASUP006_NonexistentTicket() {
	resp, err := s.ctx.AdminClient.AdminGetTicket(uuid.New())
	s.Require().NoError(err)
	s.Require().Equal(http.StatusNotFound, resp.StatusCode)
}

// ==================== POST /api/v1/admin/support/tickets/{id}/messages ====================

func (s *AdminSupportSuite) TestASUP007_AddMessage() {
	// Create a ticket for this test
	ticketResp, err := s.ctx.Customer.Client.CreateTicket("Admin message test", "Initial message")
	s.Require().NoError(err)
	s.Require().Equal(http.StatusCreated, ticketResp.StatusCode)

	// Admin adds a message
	resp, err := s.ctx.AdminClient.AdminAddTicketMessage(ticketResp.Body.ID, "Admin response")
	s.Require().NoError(err)
	s.Require().Equal(http.StatusNoContent, resp.StatusCode, string(resp.RawBody))

	// Verify message was added
	detailResp, err := s.ctx.AdminClient.AdminGetTicket(ticketResp.Body.ID)
	s.Require().NoError(err)
	s.Assert().GreaterOrEqual(len(detailResp.Body.Messages), 2)

	// Find admin message
	foundAdminMsg := false
	for _, msg := range detailResp.Body.Messages {
		if msg.SenderType == string(entities.SenderTypeAdmin) && msg.Content == "Admin response" {
			foundAdminMsg = true
			break
		}
	}
	s.Assert().True(foundAdminMsg, "should find admin message")
}

func (s *AdminSupportSuite) TestASUP008_EmptyMessage() {
	ticketResp, err := s.ctx.Customer.Client.CreateTicket("Empty msg test", "Initial")
	s.Require().NoError(err)
	s.Require().Equal(http.StatusCreated, ticketResp.StatusCode)

	resp, err := s.ctx.AdminClient.AdminAddTicketMessage(ticketResp.Body.ID, "")
	s.Require().NoError(err)
	s.Require().Equal(http.StatusBadRequest, resp.StatusCode)
}

// ==================== POST /api/v1/admin/support/tickets/{id}/close ====================

func (s *AdminSupportSuite) TestASUP009_CloseTicket() {
	// Create a ticket
	ticketResp, err := s.ctx.Customer.Client.CreateTicket("Admin close test", "Initial message")
	s.Require().NoError(err)
	s.Require().Equal(http.StatusCreated, ticketResp.StatusCode)

	// Admin closes the ticket
	resp, err := s.ctx.AdminClient.AdminCloseTicket(ticketResp.Body.ID)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusNoContent, resp.StatusCode, string(resp.RawBody))

	// Verify status changed
	detailResp, err := s.ctx.AdminClient.AdminGetTicket(ticketResp.Body.ID)
	s.Require().NoError(err)
	s.Assert().Equal("closed", detailResp.Body.Status)
}

func (s *AdminSupportSuite) TestASUP010_AlreadyClosed() {
	// Create and close a ticket
	ticketResp, err := s.ctx.Customer.Client.CreateTicket("Already closed test", "Initial message")
	s.Require().NoError(err)
	s.Require().Equal(http.StatusCreated, ticketResp.StatusCode)

	_, err = s.ctx.AdminClient.AdminCloseTicket(ticketResp.Body.ID)
	s.Require().NoError(err)

	// Try to close again
	resp, err := s.ctx.AdminClient.AdminCloseTicket(ticketResp.Body.ID)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusConflict, resp.StatusCode)
}

// ==================== User session cannot access admin support endpoints ====================

func (s *AdminSupportSuite) TestListTicketsUserSession() {
	resp, err := s.ctx.Customer.Client.AdminListTickets("", 20, 0)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusUnauthorized, resp.StatusCode)
}

func (s *AdminSupportSuite) TestGetTicketUserSession() {
	resp, err := s.ctx.Customer.Client.AdminGetTicket(uuid.New())
	s.Require().NoError(err)
	s.Require().Equal(http.StatusUnauthorized, resp.StatusCode)
}

func (s *AdminSupportSuite) TestAddMessageUserSession() {
	resp, err := s.ctx.Customer.Client.AdminAddTicketMessage(uuid.New(), "test")
	s.Require().NoError(err)
	s.Require().Equal(http.StatusUnauthorized, resp.StatusCode)
}

func (s *AdminSupportSuite) TestCloseTicketUserSession() {
	resp, err := s.ctx.Customer.Client.AdminCloseTicket(uuid.New())
	s.Require().NoError(err)
	s.Require().Equal(http.StatusUnauthorized, resp.StatusCode)
}
