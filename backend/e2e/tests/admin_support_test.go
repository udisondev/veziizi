package tests

import (
	"net/http"
	"testing"

	"codeberg.org/udison/veziizi/backend/e2e/client"
	"codeberg.org/udison/veziizi/backend/e2e/fixtures"
	"codeberg.org/udison/veziizi/backend/internal/domain/support/entities"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAdminListTickets tests GET /api/v1/admin/support/tickets
func TestAdminListTickets(t *testing.T) {
	t.Parallel()
	suite := getSuite(t)
	ctx := fixtures.NewTestContext(t, suite.BaseURL)

	// Create a ticket first
	ticketResp, err := ctx.Customer.Client.CreateTicket("Test ticket for admin", "Test message")
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, ticketResp.StatusCode)

	tests := []struct {
		id         string
		name       string
		client     *client.Client
		status     string
		limit      int
		offset     int
		wantStatus int
	}{
		// Happy path
		{
			id:         "ASUP-001",
			name:       "list tickets",
			client:     ctx.AdminClient,
			status:     "",
			limit:      20,
			offset:     0,
			wantStatus: http.StatusOK,
		},
		{
			id:         "ASUP-002",
			name:       "filter by status",
			client:     ctx.AdminClient,
			status:     "open",
			limit:      20,
			offset:     0,
			wantStatus: http.StatusOK,
		},
		{
			id:         "ASUP-003",
			name:       "pagination",
			client:     ctx.AdminClient,
			status:     "",
			limit:      5,
			offset:     0,
			wantStatus: http.StatusOK,
		},

		// Auth errors
		{
			id:         "ASUP-004",
			name:       "without admin session",
			client:     ctx.AnonClient,
			status:     "",
			limit:      20,
			offset:     0,
			wantStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.id+"_"+tt.name, func(t *testing.T) {
			resp, err := tt.client.AdminListTickets(tt.status, tt.limit, tt.offset)
			require.NoError(t, err)
			require.Equal(t, tt.wantStatus, resp.StatusCode, string(resp.RawBody))

			if resp.StatusCode == http.StatusOK {
				assert.GreaterOrEqual(t, resp.Body.Total, 0)
				if tt.status != "" {
					for _, ticket := range resp.Body.Tickets {
						assert.Equal(t, tt.status, ticket.Status)
					}
				}
			}
		})
	}
}

// TestAdminGetTicket tests GET /api/v1/admin/support/tickets/{id}
func TestAdminGetTicket(t *testing.T) {
	t.Parallel()
	suite := getSuite(t)
	ctx := fixtures.NewTestContext(t, suite.BaseURL)

	// Create a ticket
	ticketResp, err := ctx.Customer.Client.CreateTicket("Admin ticket detail test", "Test message content")
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, ticketResp.StatusCode)

	tests := []struct {
		id         string
		name       string
		ticketID   uuid.UUID
		wantStatus int
	}{
		// Happy path
		{
			id:         "ASUP-005",
			name:       "get ticket",
			ticketID:   ticketResp.Body.ID,
			wantStatus: http.StatusOK,
		},

		// Not found
		{
			id:         "ASUP-006",
			name:       "nonexistent ticket",
			ticketID:   uuid.New(),
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.id+"_"+tt.name, func(t *testing.T) {
			resp, err := ctx.AdminClient.AdminGetTicket(tt.ticketID)
			require.NoError(t, err)
			require.Equal(t, tt.wantStatus, resp.StatusCode, string(resp.RawBody))

			if resp.StatusCode == http.StatusOK {
				assert.Equal(t, tt.ticketID, resp.Body.ID)
				assert.NotEmpty(t, resp.Body.Subject)
				assert.NotEmpty(t, resp.Body.Messages)
			}
		})
	}
}

// TestAdminAddTicketMessage tests POST /api/v1/admin/support/tickets/{id}/messages
func TestAdminAddTicketMessage(t *testing.T) {
	t.Parallel()
	suite := getSuite(t)
	ctx := fixtures.NewTestContext(t, suite.BaseURL)

	t.Run("ASUP-007_add_message", func(t *testing.T) {
		// Create a ticket
		ticketResp, err := ctx.Customer.Client.CreateTicket("Admin message test", "Initial message")
		require.NoError(t, err)
		require.Equal(t, http.StatusCreated, ticketResp.StatusCode)

		// Admin adds a message
		resp, err := ctx.AdminClient.AdminAddTicketMessage(ticketResp.Body.ID, "Admin response")
		require.NoError(t, err)
		require.Equal(t, http.StatusNoContent, resp.StatusCode, string(resp.RawBody))

		// Verify message was added
		detailResp, err := ctx.AdminClient.AdminGetTicket(ticketResp.Body.ID)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(detailResp.Body.Messages), 2)

		// Find admin message
		foundAdminMsg := false
		for _, msg := range detailResp.Body.Messages {
			if msg.SenderType == string(entities.SenderTypeAdmin) && msg.Content == "Admin response" {
				foundAdminMsg = true
				break
			}
		}
		assert.True(t, foundAdminMsg, "should find admin message")
	})

	t.Run("ASUP-008_empty_message", func(t *testing.T) {
		ticketResp, err := ctx.Customer.Client.CreateTicket("Empty msg test", "Initial")
		require.NoError(t, err)
		require.Equal(t, http.StatusCreated, ticketResp.StatusCode)

		resp, err := ctx.AdminClient.AdminAddTicketMessage(ticketResp.Body.ID, "")
		require.NoError(t, err)
		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}

// TestAdminCloseTicket tests POST /api/v1/admin/support/tickets/{id}/close
func TestAdminCloseTicket(t *testing.T) {
	t.Parallel()
	suite := getSuite(t)
	ctx := fixtures.NewTestContext(t, suite.BaseURL)

	t.Run("ASUP-009_close_ticket", func(t *testing.T) {
		// Create a ticket
		ticketResp, err := ctx.Customer.Client.CreateTicket("Admin close test", "Initial message")
		require.NoError(t, err)
		require.Equal(t, http.StatusCreated, ticketResp.StatusCode)

		// Admin closes the ticket
		resp, err := ctx.AdminClient.AdminCloseTicket(ticketResp.Body.ID)
		require.NoError(t, err)
		require.Equal(t, http.StatusNoContent, resp.StatusCode, string(resp.RawBody))

		// Verify status changed
		detailResp, err := ctx.AdminClient.AdminGetTicket(ticketResp.Body.ID)
		require.NoError(t, err)
		assert.Equal(t, "closed", detailResp.Body.Status)
	})

	t.Run("ASUP-010_already_closed", func(t *testing.T) {
		// Create and close a ticket
		ticketResp, err := ctx.Customer.Client.CreateTicket("Already closed test", "Initial message")
		require.NoError(t, err)
		require.Equal(t, http.StatusCreated, ticketResp.StatusCode)

		_, err = ctx.AdminClient.AdminCloseTicket(ticketResp.Body.ID)
		require.NoError(t, err)

		// Try to close again
		resp, err := ctx.AdminClient.AdminCloseTicket(ticketResp.Body.ID)
		require.NoError(t, err)
		require.Equal(t, http.StatusConflict, resp.StatusCode)
	})
}

// TestAdminSupportWithUserSession tests that admin support endpoints require admin session
func TestAdminSupportWithUserSession(t *testing.T) {
	t.Parallel()
	suite := getSuite(t)
	ctx := fixtures.NewTestContext(t, suite.BaseURL)

	t.Run("list_tickets_user_session", func(t *testing.T) {
		resp, err := ctx.Customer.Client.AdminListTickets("", 20, 0)
		require.NoError(t, err)
		require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("get_ticket_user_session", func(t *testing.T) {
		resp, err := ctx.Customer.Client.AdminGetTicket(uuid.New())
		require.NoError(t, err)
		require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("add_message_user_session", func(t *testing.T) {
		resp, err := ctx.Customer.Client.AdminAddTicketMessage(uuid.New(), "test")
		require.NoError(t, err)
		require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("close_ticket_user_session", func(t *testing.T) {
		resp, err := ctx.Customer.Client.AdminCloseTicket(uuid.New())
		require.NoError(t, err)
		require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})
}
