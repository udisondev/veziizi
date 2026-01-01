package tests

import (
	"net/http"
	"strings"
	"testing"

	"codeberg.org/udison/veziizi/backend/e2e/client"
	"codeberg.org/udison/veziizi/backend/e2e/fixtures"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGetOrganizationHistory tests GET /api/v1/organizations/{id}/history
func TestGetOrganizationHistory(t *testing.T) {
	t.Parallel()
	suite := getSuite(t)
	ctx := fixtures.NewTestContext(t, suite.BaseURL)

	otherOrg := ctx.QuickCustomer()

	tests := []struct {
		id         string
		name       string
		client     *client.Client
		orgID      uuid.UUID
		limit      int
		offset     int
		wantStatus int
		wantErr    string
	}{
		// Happy path
		{
			id:         "HIST-001",
			name:       "own organization history",
			client:     ctx.Customer.Client,
			orgID:      ctx.Customer.OrganizationID,
			limit:      20,
			offset:     0,
			wantStatus: http.StatusOK,
		},
		{
			id:         "HIST-002",
			name:       "pagination",
			client:     ctx.Customer.Client,
			orgID:      ctx.Customer.OrganizationID,
			limit:      5,
			offset:     0,
			wantStatus: http.StatusOK,
		},

		// Auth errors (handler checks role first, returns 403 even for anon)
		{
			id:         "HIST-003",
			name:       "without auth",
			client:     ctx.AnonClient,
			orgID:      ctx.Customer.OrganizationID,
			limit:      20,
			offset:     0,
			wantStatus: http.StatusForbidden,
		},

		// Access errors
		{
			id:         "HIST-004",
			name:       "other organization",
			client:     otherOrg.Client,
			orgID:      ctx.Customer.OrganizationID,
			limit:      20,
			offset:     0,
			wantStatus: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.id+"_"+tt.name, func(t *testing.T) {
			resp, err := tt.client.GetOrganizationHistory(tt.orgID, tt.limit, tt.offset)
			require.NoError(t, err)
			require.Equal(t, tt.wantStatus, resp.StatusCode, string(resp.RawBody))

			if tt.wantErr != "" {
				assert.Contains(t, strings.ToLower(string(resp.RawBody)), strings.ToLower(tt.wantErr))
			}
		})
	}
}

// TestGetFreightRequestHistory tests GET /api/v1/freight-requests/{id}/history
func TestGetFreightRequestHistory(t *testing.T) {
	t.Parallel()
	suite := getSuite(t)
	ctx := fixtures.NewTestContext(t, suite.BaseURL)

	// Create a freight request
	fr := fixtures.NewFreightRequest(t, ctx.Customer.Client).Create()

	otherOrg := ctx.QuickCustomer()

	tests := []struct {
		id         string
		name       string
		client     *client.Client
		frID       uuid.UUID
		limit      int
		offset     int
		wantStatus int
		wantErr    string
	}{
		// Happy path
		{
			id:         "HIST-005",
			name:       "freight request history",
			client:     ctx.Customer.Client,
			frID:       fr.ID,
			limit:      20,
			offset:     0,
			wantStatus: http.StatusOK,
		},

		// Auth errors
		{
			id:         "HIST-006",
			name:       "without auth",
			client:     ctx.AnonClient,
			frID:       fr.ID,
			limit:      20,
			offset:     0,
			wantStatus: http.StatusUnauthorized,
		},

		// Not found
		{
			id:         "HIST-007",
			name:       "nonexistent freight request",
			client:     ctx.Customer.Client,
			frID:       uuid.New(),
			limit:      20,
			offset:     0,
			wantStatus: http.StatusNotFound,
		},

		// Access errors (carrier cannot see customer's FR history)
		{
			id:         "HIST-007b",
			name:       "other organization",
			client:     otherOrg.Client,
			frID:       fr.ID,
			limit:      20,
			offset:     0,
			wantStatus: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.id+"_"+tt.name, func(t *testing.T) {
			resp, err := tt.client.GetFreightRequestHistory(tt.frID, tt.limit, tt.offset)
			require.NoError(t, err)
			require.Equal(t, tt.wantStatus, resp.StatusCode, string(resp.RawBody))

			if tt.wantErr != "" {
				assert.Contains(t, strings.ToLower(string(resp.RawBody)), strings.ToLower(tt.wantErr))
			}
		})
	}
}

// TestGetOrderHistory tests GET /api/v1/orders/{id}/history
func TestGetOrderHistory(t *testing.T) {
	t.Parallel()
	suite := getSuite(t)
	ctx := fixtures.NewTestContext(t, suite.BaseURL)

	// Create a confirmed order
	_, _, orderID := ctx.CreateConfirmedOrder()

	otherOrg := ctx.QuickCustomer()

	tests := []struct {
		id         string
		name       string
		client     *client.Client
		orderID    uuid.UUID
		limit      int
		offset     int
		wantStatus int
		wantErr    string
	}{
		// Happy path
		{
			id:         "HIST-008",
			name:       "order history as participant",
			client:     ctx.Customer.Client,
			orderID:    orderID,
			limit:      20,
			offset:     0,
			wantStatus: http.StatusOK,
		},

		// Auth errors
		{
			id:         "HIST-009",
			name:       "without auth",
			client:     ctx.AnonClient,
			orderID:    orderID,
			limit:      20,
			offset:     0,
			wantStatus: http.StatusUnauthorized,
		},

		// Access errors
		{
			id:         "HIST-010",
			name:       "non-participant",
			client:     otherOrg.Client,
			orderID:    orderID,
			limit:      20,
			offset:     0,
			wantStatus: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.id+"_"+tt.name, func(t *testing.T) {
			resp, err := tt.client.GetOrderHistory(tt.orderID, tt.limit, tt.offset)
			require.NoError(t, err)
			require.Equal(t, tt.wantStatus, resp.StatusCode, string(resp.RawBody))

			if tt.wantErr != "" {
				assert.Contains(t, strings.ToLower(string(resp.RawBody)), strings.ToLower(tt.wantErr))
			}
		})
	}
}

// TestMemberRoleAccessToHistory tests that only owner/admin can access history
func TestMemberRoleAccessToHistory(t *testing.T) {
	t.Parallel()
	suite := getSuite(t)
	ctx := fixtures.NewTestContext(t, suite.BaseURL)

	// Add a regular employee to customer org
	memberClient := ctx.AddMemberToOrg(ctx.Customer, "employee")

	// Member should not be able to access history
	resp, err := memberClient.GetOrganizationHistory(ctx.Customer.OrganizationID, 20, 0)
	require.NoError(t, err)
	require.Equal(t, http.StatusForbidden, resp.StatusCode)
}
