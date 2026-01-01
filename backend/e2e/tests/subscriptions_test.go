package tests

import (
	"net/http"
	"testing"

	"codeberg.org/udison/veziizi/backend/e2e/client"
	"codeberg.org/udison/veziizi/backend/e2e/fixtures"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGetSubscriptions tests GET /api/v1/subscriptions
func TestGetSubscriptions(t *testing.T) {
	t.Parallel()
	suite := getSuite(t)
	ctx := fixtures.NewTestContext(t, suite.BaseURL)

	// Create a subscription first
	subResp, err := ctx.Customer.Client.CreateSubscription(client.CreateSubscriptionRequest{
		Name:     "Test Subscription",
		IsActive: true,
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, subResp.StatusCode)

	tests := []struct {
		id         string
		name       string
		client     *client.Client
		wantStatus int
	}{
		// Happy path
		{
			id:         "SUB-001",
			name:       "list subscriptions",
			client:     ctx.Customer.Client,
			wantStatus: http.StatusOK,
		},

		// Auth errors
		{
			id:         "SUB-003",
			name:       "without auth",
			client:     ctx.AnonClient,
			wantStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.id+"_"+tt.name, func(t *testing.T) {
			resp, err := tt.client.GetSubscriptions()
			require.NoError(t, err)
			require.Equal(t, tt.wantStatus, resp.StatusCode, string(resp.RawBody))
		})
	}

	t.Run("SUB-002_empty_list", func(t *testing.T) {
		// Create a new user who has no subscriptions
		newOrg := ctx.QuickCustomer()

		resp, err := newOrg.Client.GetSubscriptions()
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Empty(t, resp.Body)
	})
}

// TestCreateSubscription tests POST /api/v1/subscriptions
func TestCreateSubscription(t *testing.T) {
	t.Parallel()
	suite := getSuite(t)
	ctx := fixtures.NewTestContext(t, suite.BaseURL)

	t.Run("SUB-004_create_subscription", func(t *testing.T) {
		resp, err := ctx.Customer.Client.CreateSubscription(client.CreateSubscriptionRequest{
			Name:     "My Subscription",
			IsActive: true,
		})
		require.NoError(t, err)
		require.Equal(t, http.StatusCreated, resp.StatusCode, string(resp.RawBody))

		assert.NotEqual(t, uuid.Nil, resp.Body.ID)
		assert.Equal(t, "My Subscription", resp.Body.Name)
		assert.True(t, resp.Body.IsActive)
	})

	t.Run("SUB-005_with_filters", func(t *testing.T) {
		minWeight := 100.0
		maxWeight := 500.0
		resp, err := ctx.Customer.Client.CreateSubscription(client.CreateSubscriptionRequest{
			Name:         "Filtered Subscription",
			MinWeight:    &minWeight,
			MaxWeight:    &maxWeight,
			VehicleTypes: []string{"tent"},
			IsActive:     true,
		})
		require.NoError(t, err)
		require.Equal(t, http.StatusCreated, resp.StatusCode, string(resp.RawBody))

		assert.NotNil(t, resp.Body.MinWeight)
		assert.NotNil(t, resp.Body.MaxWeight)
		assert.Contains(t, resp.Body.VehicleTypes, "tent")
	})

	t.Run("SUB-006_empty_name", func(t *testing.T) {
		resp, err := ctx.Customer.Client.CreateSubscription(client.CreateSubscriptionRequest{
			Name:     "",
			IsActive: true,
		})
		require.NoError(t, err)
		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}

// TestGetSubscription tests GET /api/v1/subscriptions/{id}
func TestGetSubscription(t *testing.T) {
	t.Parallel()
	suite := getSuite(t)
	ctx := fixtures.NewTestContext(t, suite.BaseURL)

	// Create a subscription
	subResp, err := ctx.Customer.Client.CreateSubscription(client.CreateSubscriptionRequest{
		Name:     "Test Get Subscription",
		IsActive: true,
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, subResp.StatusCode)

	otherOrg := ctx.QuickCustomer()

	tests := []struct {
		id         string
		name       string
		client     *client.Client
		subID      uuid.UUID
		wantStatus int
	}{
		// Happy path
		{
			id:         "SUB-007",
			name:       "get own subscription",
			client:     ctx.Customer.Client,
			subID:      subResp.Body.ID,
			wantStatus: http.StatusOK,
		},

		// Access errors
		{
			id:         "SUB-008",
			name:       "other orgs subscription",
			client:     otherOrg.Client,
			subID:      subResp.Body.ID,
			wantStatus: http.StatusForbidden,
		},

		// Not found
		{
			id:         "SUB-009",
			name:       "nonexistent subscription",
			client:     ctx.Customer.Client,
			subID:      uuid.New(),
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.id+"_"+tt.name, func(t *testing.T) {
			resp, err := tt.client.GetSubscription(tt.subID)
			require.NoError(t, err)
			require.Equal(t, tt.wantStatus, resp.StatusCode, string(resp.RawBody))

			if resp.StatusCode == http.StatusOK {
				assert.Equal(t, tt.subID, resp.Body.ID)
			}
		})
	}
}

// TestUpdateSubscription tests PUT /api/v1/subscriptions/{id}
func TestUpdateSubscription(t *testing.T) {
	t.Parallel()
	suite := getSuite(t)
	ctx := fixtures.NewTestContext(t, suite.BaseURL)

	t.Run("SUB-010_update_filters", func(t *testing.T) {
		// Create a subscription
		subResp, err := ctx.Customer.Client.CreateSubscription(client.CreateSubscriptionRequest{
			Name:     "Original Name",
			IsActive: true,
		})
		require.NoError(t, err)
		require.Equal(t, http.StatusCreated, subResp.StatusCode)

		// Update it
		minWeight := 200.0
		updateResp, err := ctx.Customer.Client.UpdateSubscription(subResp.Body.ID, client.CreateSubscriptionRequest{
			Name:      "Updated Name",
			MinWeight: &minWeight,
			IsActive:  true,
		})
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, updateResp.StatusCode, string(updateResp.RawBody))

		assert.Equal(t, "Updated Name", updateResp.Body.Name)
		assert.NotNil(t, updateResp.Body.MinWeight)
	})
}

// TestDeleteSubscription tests DELETE /api/v1/subscriptions/{id}
func TestDeleteSubscription(t *testing.T) {
	t.Parallel()
	suite := getSuite(t)
	ctx := fixtures.NewTestContext(t, suite.BaseURL)

	t.Run("SUB-011_delete_own", func(t *testing.T) {
		// Create a subscription
		subResp, err := ctx.Customer.Client.CreateSubscription(client.CreateSubscriptionRequest{
			Name:     "To Delete",
			IsActive: true,
		})
		require.NoError(t, err)
		require.Equal(t, http.StatusCreated, subResp.StatusCode)

		// Delete it
		deleteResp, err := ctx.Customer.Client.DeleteSubscription(subResp.Body.ID)
		require.NoError(t, err)
		require.Equal(t, http.StatusNoContent, deleteResp.StatusCode, string(deleteResp.RawBody))

		// Verify it's gone
		getResp, err := ctx.Customer.Client.GetSubscription(subResp.Body.ID)
		require.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, getResp.StatusCode)
	})
}

// TestSetSubscriptionActive tests PATCH /api/v1/subscriptions/{id}/active
func TestSetSubscriptionActive(t *testing.T) {
	t.Parallel()
	suite := getSuite(t)
	ctx := fixtures.NewTestContext(t, suite.BaseURL)

	// Create an active subscription
	subResp, err := ctx.Customer.Client.CreateSubscription(client.CreateSubscriptionRequest{
		Name:     "Active Subscription",
		IsActive: true,
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, subResp.StatusCode)
	require.True(t, subResp.Body.IsActive)

	t.Run("SUB-013_deactivate", func(t *testing.T) {
		resp, err := ctx.Customer.Client.SetSubscriptionActive(subResp.Body.ID, false)
		require.NoError(t, err)
		require.Equal(t, http.StatusNoContent, resp.StatusCode, string(resp.RawBody))

		// Verify it's deactivated
		getResp, err := ctx.Customer.Client.GetSubscription(subResp.Body.ID)
		require.NoError(t, err)
		assert.False(t, getResp.Body.IsActive)
	})

	t.Run("SUB-012_activate", func(t *testing.T) {
		resp, err := ctx.Customer.Client.SetSubscriptionActive(subResp.Body.ID, true)
		require.NoError(t, err)
		require.Equal(t, http.StatusNoContent, resp.StatusCode, string(resp.RawBody))

		// Verify it's activated
		getResp, err := ctx.Customer.Client.GetSubscription(subResp.Body.ID)
		require.NoError(t, err)
		assert.True(t, getResp.Body.IsActive)
	})
}

// TestSubscriptionWithoutAuth tests that subscription endpoints require auth
func TestSubscriptionWithoutAuth(t *testing.T) {
	t.Parallel()
	suite := getSuite(t)
	ctx := fixtures.NewTestContext(t, suite.BaseURL)

	t.Run("create_without_auth", func(t *testing.T) {
		resp, err := ctx.AnonClient.CreateSubscription(client.CreateSubscriptionRequest{
			Name:     "Test",
			IsActive: true,
		})
		require.NoError(t, err)
		require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("get_without_auth", func(t *testing.T) {
		resp, err := ctx.AnonClient.GetSubscription(uuid.New())
		require.NoError(t, err)
		require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})
}
