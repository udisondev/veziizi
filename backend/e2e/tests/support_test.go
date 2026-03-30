package tests

import (
	"net/http"
	"testing"

	"github.com/udisondev/veziizi/backend/e2e/client"
	"github.com/udisondev/veziizi/backend/e2e/fixtures"
	"github.com/udisondev/veziizi/backend/e2e/helpers"
	"github.com/udisondev/veziizi/backend/internal/domain/support/values"
	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
)

// SupportSuite combines all support tests with shared context.
type SupportSuite struct {
	suite.Suite
	baseURL string
	ctx     *fixtures.TestContext

	// Other organization for access tests
	otherOrg *fixtures.CreatedOrganization

	// Shared ticket for tests
	sharedTicketID uuid.UUID
}

func TestSupportSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(SupportSuite))
}

func (s *SupportSuite) SetupSuite() {
	testSuite := getSuite(s.T())
	s.baseURL = testSuite.BaseURL
	s.ctx = fixtures.NewTestContext(s.T(), s.baseURL)
	s.otherOrg = s.ctx.QuickCustomer()

	// Create a shared ticket
	ticketResp, err := s.ctx.Customer.Client.CreateTicket("Shared test ticket", "Test message")
	s.Require().NoError(err)
	s.Require().Equal(http.StatusCreated, ticketResp.StatusCode)
	s.sharedTicketID = ticketResp.Body.ID
}

// ==================== GET /api/v1/support/faq ====================

func (s *SupportSuite) TestSUP001_ListFAQ() {
	resp, err := s.ctx.Customer.Client.GetFAQ()
	s.Require().NoError(err)
	s.Require().Equal(http.StatusOK, resp.StatusCode, string(resp.RawBody))
}

func (s *SupportSuite) TestSUP002_PublicAccess() {
	resp, err := s.ctx.AnonClient.GetFAQ()
	s.Require().NoError(err)
	s.Require().Equal(http.StatusOK, resp.StatusCode, string(resp.RawBody))
}

// ==================== POST /api/v1/support/tickets ====================

func (s *SupportSuite) TestSUP003_CreateTicket() {
	resp, err := s.ctx.Customer.Client.CreateTicket("Test subject", "Test message content")
	s.Require().NoError(err)
	s.Require().Equal(http.StatusCreated, resp.StatusCode, string(resp.RawBody))
	s.Assert().NotEqual(uuid.Nil, resp.Body.ID)
	s.Assert().Equal("Test subject", resp.Body.Subject)
	s.Assert().Equal("open", resp.Body.Status)
}

func (s *SupportSuite) TestSUP004_WithCategory() {
	resp, err := s.ctx.Customer.Client.CreateTicket("Technical issue", "I have a problem with...")
	s.Require().NoError(err)
	s.Require().Equal(http.StatusCreated, resp.StatusCode, string(resp.RawBody))
}

func (s *SupportSuite) TestSUP005_EmptySubject() {
	resp, err := s.ctx.Customer.Client.CreateTicket("", "Some message")
	s.Require().NoError(err)
	s.Require().Equal(http.StatusBadRequest, resp.StatusCode)
}

func (s *SupportSuite) TestSUP006_EmptyMessage() {
	resp, err := s.ctx.Customer.Client.CreateTicket("Subject", "")
	s.Require().NoError(err)
	s.Require().Equal(http.StatusBadRequest, resp.StatusCode)
}

func (s *SupportSuite) TestSUP007_WithoutAuth() {
	resp, err := s.ctx.AnonClient.CreateTicket("Test", "Test message")
	s.Require().NoError(err)
	s.Require().Equal(http.StatusUnauthorized, resp.StatusCode)
}

// ==================== GET /api/v1/support/tickets ====================

func (s *SupportSuite) TestSUP008_ListTickets() {
	// Wait for projection to sync
	tickets := helpers.WaitFor(s.T(), func() ([]client.TicketResponse, bool) {
		resp, err := s.ctx.Customer.Client.GetMyTickets()
		if err != nil || resp.StatusCode != http.StatusOK {
			return nil, false
		}
		// Check if our ticket is in the list
		for _, ticket := range resp.Body {
			if ticket.ID == s.sharedTicketID {
				return resp.Body, true
			}
		}
		return nil, false
	}, "ticket should appear in list")

	s.Assert().GreaterOrEqual(len(tickets), 1)
}

func (s *SupportSuite) TestSUP009_EmptyList() {
	// Create a new user who has no tickets
	newOrg := s.ctx.QuickCustomer()

	resp, err := newOrg.Client.GetMyTickets()
	s.Require().NoError(err)
	s.Require().Equal(http.StatusOK, resp.StatusCode)
	s.Assert().Empty(resp.Body)
}

func (s *SupportSuite) TestSUP010_FilterByStatus() {
	resp, err := s.ctx.Customer.Client.GetMyTicketsFiltered("open")
	s.Require().NoError(err)
	s.Require().Equal(http.StatusOK, resp.StatusCode)

	for _, ticket := range resp.Body {
		s.Assert().Equal("open", ticket.Status)
	}
}

// ==================== GET /api/v1/support/tickets/{id} ====================

func (s *SupportSuite) TestSUP011_GetOwnTicket() {
	resp, err := s.ctx.Customer.Client.GetTicket(s.sharedTicketID)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusOK, resp.StatusCode, string(resp.RawBody))
	s.Assert().Equal(s.sharedTicketID, resp.Body.ID)
	s.Assert().NotEmpty(resp.Body.Subject)
}

func (s *SupportSuite) TestSUP012_OtherUsersTicket() {
	resp, err := s.otherOrg.Client.GetTicket(s.sharedTicketID)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusForbidden, resp.StatusCode)
}

func (s *SupportSuite) TestSUP013_NonexistentTicket() {
	resp, err := s.ctx.Customer.Client.GetTicket(uuid.New())
	s.Require().NoError(err)
	s.Require().Equal(http.StatusNotFound, resp.StatusCode)
}

// ==================== POST /api/v1/support/tickets/{id}/messages ====================

func (s *SupportSuite) TestSUP014_AddMessage() {
	// Create a ticket
	ticketResp, err := s.ctx.Customer.Client.CreateTicket("Ticket for messages", "Initial message")
	s.Require().NoError(err)
	s.Require().Equal(http.StatusCreated, ticketResp.StatusCode)

	// Add a message
	resp, err := s.ctx.Customer.Client.AddTicketMessage(ticketResp.Body.ID, "Follow-up message")
	s.Require().NoError(err)
	s.Require().Equal(http.StatusNoContent, resp.StatusCode, string(resp.RawBody))

	// Verify message was added
	detailResp, err := s.ctx.Customer.Client.GetTicket(ticketResp.Body.ID)
	s.Require().NoError(err)
	s.Assert().GreaterOrEqual(len(detailResp.Body.Messages), 2) // Initial + follow-up
}

func (s *SupportSuite) TestSUP015_EmptyAddMessage() {
	ticketResp, err := s.ctx.Customer.Client.CreateTicket("Ticket for empty msg test", "Initial message")
	s.Require().NoError(err)
	s.Require().Equal(http.StatusCreated, ticketResp.StatusCode)

	resp, err := s.ctx.Customer.Client.AddTicketMessage(ticketResp.Body.ID, "")
	s.Require().NoError(err)
	s.Require().Equal(http.StatusBadRequest, resp.StatusCode)
}

func (s *SupportSuite) TestSUP016_ClosedTicket() {
	// Create a ticket
	ticketResp, err := s.ctx.Customer.Client.CreateTicket("Ticket to close", "Initial message")
	s.Require().NoError(err)
	s.Require().Equal(http.StatusCreated, ticketResp.StatusCode)

	// Admin closes the ticket (only admins can close tickets)
	closeResp, err := s.ctx.AdminClient.AdminCloseTicket(ticketResp.Body.ID)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusNoContent, closeResp.StatusCode)

	// Try to add message to closed ticket
	resp, err := s.ctx.Customer.Client.AddTicketMessage(ticketResp.Body.ID, "Message after close")
	s.Require().NoError(err)
	s.Require().Equal(http.StatusConflict, resp.StatusCode)
}

// ==================== POST /api/v1/support/tickets/{id}/reopen ====================

func (s *SupportSuite) TestSUP017_ReopenClosed() {
	// Create a ticket
	ticketResp, err := s.ctx.Customer.Client.CreateTicket("Ticket to reopen", "Initial message")
	s.Require().NoError(err)
	s.Require().Equal(http.StatusCreated, ticketResp.StatusCode)

	// Admin closes the ticket (only admins can close tickets)
	closeResp, err := s.ctx.AdminClient.AdminCloseTicket(ticketResp.Body.ID)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusNoContent, closeResp.StatusCode)

	// Reopen ticket
	resp, err := s.ctx.Customer.Client.ReopenTicket(ticketResp.Body.ID)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusNoContent, resp.StatusCode, string(resp.RawBody))

	// Verify status changed to awaiting_reply (user reopened and awaits admin response)
	detailResp, err := s.ctx.Customer.Client.GetTicket(ticketResp.Body.ID)
	s.Require().NoError(err)
	s.Assert().Equal(string(values.TicketStatusAwaitingReply), detailResp.Body.Status)
}

func (s *SupportSuite) TestSUP018_AlreadyOpen() {
	// Create a ticket (already open)
	ticketResp, err := s.ctx.Customer.Client.CreateTicket("Open ticket", "Initial message")
	s.Require().NoError(err)
	s.Require().Equal(http.StatusCreated, ticketResp.StatusCode)

	// Try to reopen an already open ticket
	resp, err := s.ctx.Customer.Client.ReopenTicket(ticketResp.Body.ID)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusConflict, resp.StatusCode)
}

// ==================== Without auth ====================

func (s *SupportSuite) TestGetTicketsWithoutAuth() {
	resp, err := s.ctx.AnonClient.GetMyTickets()
	s.Require().NoError(err)
	s.Require().Equal(http.StatusUnauthorized, resp.StatusCode)
}

func (s *SupportSuite) TestGetTicketWithoutAuth() {
	resp, err := s.ctx.AnonClient.GetTicket(uuid.New())
	s.Require().NoError(err)
	s.Require().Equal(http.StatusUnauthorized, resp.StatusCode)
}

func (s *SupportSuite) TestAddMessageWithoutAuth() {
	resp, err := s.ctx.AnonClient.AddTicketMessage(uuid.New(), "test")
	s.Require().NoError(err)
	s.Require().Equal(http.StatusUnauthorized, resp.StatusCode)
}
