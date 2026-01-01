package tests

import (
	"net/http"
	"testing"

	"codeberg.org/udison/veziizi/backend/e2e/client"
	"codeberg.org/udison/veziizi/backend/e2e/fixtures"
	"codeberg.org/udison/veziizi/backend/e2e/helpers"
	"codeberg.org/udison/veziizi/backend/internal/domain/support/values"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGetFAQ tests GET /api/v1/support/faq
func TestGetFAQ(t *testing.T) {
	t.Parallel()
	suite := getSuite(t)
	ctx := fixtures.NewTestContext(t, suite.BaseURL)

	t.Run("SUP-001_list_faq", func(t *testing.T) {
		resp, err := ctx.Customer.Client.GetFAQ()
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode, string(resp.RawBody))
		// FAQ can be empty or have items
	})

	t.Run("SUP-002_public_access", func(t *testing.T) {
		resp, err := ctx.AnonClient.GetFAQ()
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode, string(resp.RawBody))
	})
}

// TestCreateTicket tests POST /api/v1/support/tickets
func TestCreateTicket(t *testing.T) {
	t.Parallel()
	suite := getSuite(t)
	ctx := fixtures.NewTestContext(t, suite.BaseURL)

	tests := []struct {
		id         string
		name       string
		client     *client.Client
		subject    string
		message    string
		wantStatus int
	}{
		// Happy path
		{
			id:         "SUP-003",
			name:       "create ticket",
			client:     ctx.Customer.Client,
			subject:    "Test subject",
			message:    "Test message content",
			wantStatus: http.StatusCreated,
		},
		{
			id:         "SUP-004",
			name:       "with category",
			client:     ctx.Customer.Client,
			subject:    "Technical issue",
			message:    "I have a problem with...",
			wantStatus: http.StatusCreated,
		},

		// Validation errors
		{
			id:         "SUP-005",
			name:       "empty subject",
			client:     ctx.Customer.Client,
			subject:    "",
			message:    "Some message",
			wantStatus: http.StatusBadRequest,
		},
		{
			id:         "SUP-006",
			name:       "empty message",
			client:     ctx.Customer.Client,
			subject:    "Subject",
			message:    "",
			wantStatus: http.StatusBadRequest,
		},

		// Auth errors
		{
			id:         "SUP-007",
			name:       "without auth",
			client:     ctx.AnonClient,
			subject:    "Test",
			message:    "Test message",
			wantStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.id+"_"+tt.name, func(t *testing.T) {
			resp, err := tt.client.CreateTicket(tt.subject, tt.message)
			require.NoError(t, err)
			require.Equal(t, tt.wantStatus, resp.StatusCode, string(resp.RawBody))

			if resp.StatusCode == http.StatusCreated {
				assert.NotEqual(t, uuid.Nil, resp.Body.ID)
				assert.Equal(t, tt.subject, resp.Body.Subject)
				assert.Equal(t, "open", resp.Body.Status)
			}
		})
	}
}

// TestGetMyTickets tests GET /api/v1/support/tickets
func TestGetMyTickets(t *testing.T) {
	t.Parallel()
	suite := getSuite(t)
	ctx := fixtures.NewTestContext(t, suite.BaseURL)

	// Create a ticket first
	ticketResp, err := ctx.Customer.Client.CreateTicket("Test ticket", "Test message")
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, ticketResp.StatusCode)

	t.Run("SUP-008_list_tickets", func(t *testing.T) {
		// Wait for projection to sync
		tickets := helpers.WaitFor(t, func() ([]client.TicketResponse, bool) {
			resp, err := ctx.Customer.Client.GetMyTickets()
			if err != nil || resp.StatusCode != http.StatusOK {
				return nil, false
			}
			// Check if our ticket is in the list
			for _, ticket := range resp.Body {
				if ticket.ID == ticketResp.Body.ID {
					return resp.Body, true
				}
			}
			return nil, false
		}, "ticket should appear in list")

		assert.GreaterOrEqual(t, len(tickets), 1)
	})

	t.Run("SUP-009_empty_list", func(t *testing.T) {
		// Create a new user who has no tickets
		newOrg := ctx.QuickCustomer()

		resp, err := newOrg.Client.GetMyTickets()
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Empty(t, resp.Body)
	})

	t.Run("SUP-010_filter_by_status", func(t *testing.T) {
		resp, err := ctx.Customer.Client.GetMyTicketsFiltered("open")
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)

		for _, ticket := range resp.Body {
			assert.Equal(t, "open", ticket.Status)
		}
	})
}

// TestGetTicket tests GET /api/v1/support/tickets/{id}
func TestGetTicket(t *testing.T) {
	t.Parallel()
	suite := getSuite(t)
	ctx := fixtures.NewTestContext(t, suite.BaseURL)

	// Create a ticket
	ticketResp, err := ctx.Customer.Client.CreateTicket("Test ticket for get", "Test message")
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, ticketResp.StatusCode)

	otherOrg := ctx.QuickCustomer()

	tests := []struct {
		id         string
		name       string
		client     *client.Client
		ticketID   uuid.UUID
		wantStatus int
	}{
		// Happy path
		{
			id:         "SUP-011",
			name:       "get own ticket",
			client:     ctx.Customer.Client,
			ticketID:   ticketResp.Body.ID,
			wantStatus: http.StatusOK,
		},

		// Access errors
		{
			id:         "SUP-012",
			name:       "other users ticket",
			client:     otherOrg.Client,
			ticketID:   ticketResp.Body.ID,
			wantStatus: http.StatusForbidden,
		},

		// Not found
		{
			id:         "SUP-013",
			name:       "nonexistent ticket",
			client:     ctx.Customer.Client,
			ticketID:   uuid.New(),
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.id+"_"+tt.name, func(t *testing.T) {
			resp, err := tt.client.GetTicket(tt.ticketID)
			require.NoError(t, err)
			require.Equal(t, tt.wantStatus, resp.StatusCode, string(resp.RawBody))

			if resp.StatusCode == http.StatusOK {
				assert.Equal(t, tt.ticketID, resp.Body.ID)
				assert.NotEmpty(t, resp.Body.Subject)
			}
		})
	}
}

// TestAddTicketMessage tests POST /api/v1/support/tickets/{id}/messages
func TestAddTicketMessage(t *testing.T) {
	t.Parallel()
	suite := getSuite(t)
	ctx := fixtures.NewTestContext(t, suite.BaseURL)

	t.Run("SUP-014_add_message", func(t *testing.T) {
		// Create a ticket
		ticketResp, err := ctx.Customer.Client.CreateTicket("Ticket for messages", "Initial message")
		require.NoError(t, err)
		require.Equal(t, http.StatusCreated, ticketResp.StatusCode)

		// Add a message
		resp, err := ctx.Customer.Client.AddTicketMessage(ticketResp.Body.ID, "Follow-up message")
		require.NoError(t, err)
		require.Equal(t, http.StatusNoContent, resp.StatusCode, string(resp.RawBody))

		// Verify message was added
		detailResp, err := ctx.Customer.Client.GetTicket(ticketResp.Body.ID)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(detailResp.Body.Messages), 2) // Initial + follow-up
	})

	t.Run("SUP-015_empty_message", func(t *testing.T) {
		ticketResp, err := ctx.Customer.Client.CreateTicket("Ticket for empty msg test", "Initial message")
		require.NoError(t, err)
		require.Equal(t, http.StatusCreated, ticketResp.StatusCode)

		resp, err := ctx.Customer.Client.AddTicketMessage(ticketResp.Body.ID, "")
		require.NoError(t, err)
		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("SUP-016_closed_ticket", func(t *testing.T) {
		// Create a ticket
		ticketResp, err := ctx.Customer.Client.CreateTicket("Ticket to close", "Initial message")
		require.NoError(t, err)
		require.Equal(t, http.StatusCreated, ticketResp.StatusCode)

		// Admin closes the ticket (only admins can close tickets)
		closeResp, err := ctx.AdminClient.AdminCloseTicket(ticketResp.Body.ID)
		require.NoError(t, err)
		require.Equal(t, http.StatusNoContent, closeResp.StatusCode)

		// Try to add message to closed ticket
		resp, err := ctx.Customer.Client.AddTicketMessage(ticketResp.Body.ID, "Message after close")
		require.NoError(t, err)
		require.Equal(t, http.StatusConflict, resp.StatusCode)
	})
}

// TestReopenTicket tests POST /api/v1/support/tickets/{id}/reopen
func TestReopenTicket(t *testing.T) {
	t.Parallel()
	suite := getSuite(t)
	ctx := fixtures.NewTestContext(t, suite.BaseURL)

	t.Run("SUP-017_reopen_closed", func(t *testing.T) {
		// Create a ticket
		ticketResp, err := ctx.Customer.Client.CreateTicket("Ticket to reopen", "Initial message")
		require.NoError(t, err)
		require.Equal(t, http.StatusCreated, ticketResp.StatusCode)

		// Admin closes the ticket (only admins can close tickets)
		closeResp, err := ctx.AdminClient.AdminCloseTicket(ticketResp.Body.ID)
		require.NoError(t, err)
		require.Equal(t, http.StatusNoContent, closeResp.StatusCode)

		// Reopen ticket
		resp, err := ctx.Customer.Client.ReopenTicket(ticketResp.Body.ID)
		require.NoError(t, err)
		require.Equal(t, http.StatusNoContent, resp.StatusCode, string(resp.RawBody))

		// Verify status changed to awaiting_reply (user reopened and awaits admin response)
		detailResp, err := ctx.Customer.Client.GetTicket(ticketResp.Body.ID)
		require.NoError(t, err)
		assert.Equal(t, string(values.TicketStatusAwaitingReply), detailResp.Body.Status)
	})

	t.Run("SUP-018_already_open", func(t *testing.T) {
		// Create a ticket (already open)
		ticketResp, err := ctx.Customer.Client.CreateTicket("Open ticket", "Initial message")
		require.NoError(t, err)
		require.Equal(t, http.StatusCreated, ticketResp.StatusCode)

		// Try to reopen an already open ticket
		resp, err := ctx.Customer.Client.ReopenTicket(ticketResp.Body.ID)
		require.NoError(t, err)
		require.Equal(t, http.StatusConflict, resp.StatusCode)
	})
}

// TestTicketWithoutAuth tests that ticket endpoints require auth
func TestTicketWithoutAuth(t *testing.T) {
	t.Parallel()
	suite := getSuite(t)
	ctx := fixtures.NewTestContext(t, suite.BaseURL)

	t.Run("get_tickets_without_auth", func(t *testing.T) {
		resp, err := ctx.AnonClient.GetMyTickets()
		require.NoError(t, err)
		require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("get_ticket_without_auth", func(t *testing.T) {
		resp, err := ctx.AnonClient.GetTicket(uuid.New())
		require.NoError(t, err)
		require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("add_message_without_auth", func(t *testing.T) {
		resp, err := ctx.AnonClient.AddTicketMessage(uuid.New(), "test")
		require.NoError(t, err)
		require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})
}
