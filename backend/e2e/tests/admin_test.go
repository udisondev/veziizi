package tests

import (
	"net/http"
	"strings"
	"testing"

	"codeberg.org/udison/veziizi/backend/e2e/client"
	"codeberg.org/udison/veziizi/backend/e2e/fixtures"
	"codeberg.org/udison/veziizi/backend/e2e/helpers"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAdminLogin tests POST /api/v1/admin/auth/login
func TestAdminLogin(t *testing.T) {
	t.Parallel()
	suite := getSuite(t)
	c := client.New(suite.BaseURL)

	tests := []struct {
		id         string
		name       string
		email      string
		password   string
		wantStatus int
		wantErr    string
	}{
		// Happy path
		{
			id:         "ADM-001",
			name:       "successful login",
			email:      "admin@veziizi.local",
			password:   "admin123",
			wantStatus: http.StatusOK,
		},

		// Auth errors
		{
			id:         "ADM-002",
			name:       "wrong password",
			email:      "admin@veziizi.local",
			password:   "wrongpassword",
			wantStatus: http.StatusUnauthorized,
		},
		{
			id:         "ADM-003",
			name:       "nonexistent admin",
			email:      "notanadmin@veziizi.local",
			password:   "password",
			wantStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.id+"_"+tt.name, func(t *testing.T) {
			resp, err := c.AdminLogin(tt.email, tt.password)
			require.NoError(t, err)
			require.Equal(t, tt.wantStatus, resp.StatusCode, string(resp.RawBody))

			if tt.wantErr != "" {
				assert.Contains(t, strings.ToLower(string(resp.RawBody)), strings.ToLower(tt.wantErr))
			}

			if resp.StatusCode == http.StatusOK {
				assert.NotEmpty(t, resp.Body.Email)
			}
		})
	}
}

// TestAdminLogout tests POST /api/v1/admin/auth/logout
func TestAdminLogout(t *testing.T) {
	t.Parallel()
	suite := getSuite(t)
	ctx := fixtures.NewTestContext(t, suite.BaseURL)

	t.Run("ADM-004_successful_logout", func(t *testing.T) {
		resp, err := ctx.AdminClient.AdminLogout()
		require.NoError(t, err)
		require.Equal(t, http.StatusNoContent, resp.StatusCode, string(resp.RawBody))
	})
}

// TestAdminListOrganizations tests GET /api/v1/admin/organizations
func TestAdminListOrganizations(t *testing.T) {
	t.Parallel()
	suite := getSuite(t)
	ctx := fixtures.NewTestContext(t, suite.BaseURL)

	// Create pending org
	pendingOrg := fixtures.NewOrganization(t, ctx.AnonClient).Create()

	tests := []struct {
		id         string
		name       string
		client     *client.Client
		status     string
		wantStatus int
		check      func(*testing.T, *client.Response[[]client.OrganizationResponse])
	}{
		// Happy path
		{
			id:         "ADM-005",
			name:       "list pending organizations",
			client:     ctx.AdminClient,
			status:     "pending",
			wantStatus: http.StatusOK,
			check: func(t *testing.T, resp *client.Response[[]client.OrganizationResponse]) {
				// Should find our pending org
				found := false
				for _, org := range resp.Body {
					if org.ID == pendingOrg.OrganizationID {
						found = true
						break
					}
				}
				assert.True(t, found, "should find the pending organization")
			},
		},
		{
			id:         "ADM-006",
			name:       "filter by status",
			client:     ctx.AdminClient,
			status:     "pending",
			wantStatus: http.StatusOK,
			check: func(t *testing.T, resp *client.Response[[]client.OrganizationResponse]) {
				for _, org := range resp.Body {
					assert.Equal(t, "pending", org.Status)
				}
			},
		},

		// Auth errors
		{
			id:         "ADM-008",
			name:       "without admin session",
			client:     ctx.AnonClient,
			status:     "",
			wantStatus: http.StatusUnauthorized,
		},
		{
			id:         "ADM-009",
			name:       "with user session",
			client:     ctx.Customer.Client,
			status:     "",
			wantStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.id+"_"+tt.name, func(t *testing.T) {
			resp, err := tt.client.AdminGetOrganizations(tt.status)
			require.NoError(t, err)
			require.Equal(t, tt.wantStatus, resp.StatusCode, string(resp.RawBody))

			if tt.check != nil && resp.StatusCode == http.StatusOK {
				tt.check(t, resp)
			}
		})
	}
}

// TestAdminGetOrganization tests GET /api/v1/admin/organizations/{id}
func TestAdminGetOrganization(t *testing.T) {
	t.Parallel()
	suite := getSuite(t)
	ctx := fixtures.NewTestContext(t, suite.BaseURL)

	tests := []struct {
		id         string
		name       string
		orgID      uuid.UUID
		wantStatus int
	}{
		// Happy path
		{
			id:         "ADM-010",
			name:       "get organization",
			orgID:      ctx.Customer.OrganizationID,
			wantStatus: http.StatusOK,
		},

		// Not found
		{
			id:         "ADM-011",
			name:       "nonexistent org",
			orgID:      uuid.New(),
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.id+"_"+tt.name, func(t *testing.T) {
			resp, err := ctx.AdminClient.AdminGetOrganization(tt.orgID)
			require.NoError(t, err)
			require.Equal(t, tt.wantStatus, resp.StatusCode, string(resp.RawBody))

			if resp.StatusCode == http.StatusOK {
				assert.Equal(t, tt.orgID, resp.Body.ID)
			}
		})
	}
}

// TestAdminApproveOrganization tests POST /api/v1/admin/organizations/{id}/approve
func TestAdminApproveOrganization(t *testing.T) {
	t.Parallel()
	suite := getSuite(t)
	ctx := fixtures.NewTestContext(t, suite.BaseURL)

	t.Run("ADM-012_approve_pending", func(t *testing.T) {
		// Create pending org
		pendingOrg := fixtures.NewOrganization(t, ctx.AnonClient).Create()

		resp, err := ctx.AdminClient.AdminApproveOrganization(pendingOrg.OrganizationID, nil)
		require.NoError(t, err)
		require.Equal(t, http.StatusNoContent, resp.StatusCode, string(resp.RawBody))

		// Verify status changed
		orgResp, err := ctx.AdminClient.AdminGetOrganization(pendingOrg.OrganizationID)
		require.NoError(t, err)
		assert.Equal(t, "active", orgResp.Body.Status)
	})

	t.Run("ADM-013_already_approved", func(t *testing.T) {
		// ctx.Customer is already approved
		resp, err := ctx.AdminClient.AdminApproveOrganization(ctx.Customer.OrganizationID, nil)
		require.NoError(t, err)
		require.Equal(t, http.StatusConflict, resp.StatusCode)
	})
}

// TestAdminRejectOrganization tests POST /api/v1/admin/organizations/{id}/reject
func TestAdminRejectOrganization(t *testing.T) {
	t.Parallel()
	suite := getSuite(t)
	ctx := fixtures.NewTestContext(t, suite.BaseURL)

	t.Run("ADM-015_reject_pending", func(t *testing.T) {
		// Create pending org
		pendingOrg := fixtures.NewOrganization(t, ctx.AnonClient).Create()

		resp, err := ctx.AdminClient.AdminRejectOrganization(pendingOrg.OrganizationID, "Test rejection reason")
		require.NoError(t, err)
		require.Equal(t, http.StatusNoContent, resp.StatusCode, string(resp.RawBody))
	})

	t.Run("ADM-016_with_reason", func(t *testing.T) {
		pendingOrg := fixtures.NewOrganization(t, ctx.AnonClient).Create()

		reason := "Invalid documentation provided"
		resp, err := ctx.AdminClient.AdminRejectOrganization(pendingOrg.OrganizationID, reason)
		require.NoError(t, err)
		require.Equal(t, http.StatusNoContent, resp.StatusCode, string(resp.RawBody))
	})
}

// TestAdminMarkFraudster tests POST /api/v1/admin/organizations/{id}/mark-fraudster
func TestAdminMarkFraudster(t *testing.T) {
	t.Parallel()
	suite := getSuite(t)
	ctx := fixtures.NewTestContext(t, suite.BaseURL)

	t.Run("ADM-017_mark_fraudster", func(t *testing.T) {
		// Create a new org to mark as fraudster
		org := fixtures.NewActiveOrganization(t, ctx.AnonClient, ctx.AdminClient).Create()

		resp, err := ctx.AdminClient.AdminMarkFraudster(org.OrganizationID, true, "Fraudulent activity detected")
		require.NoError(t, err)
		require.Equal(t, http.StatusNoContent, resp.StatusCode, string(resp.RawBody))

		// Verify org is in fraudsters list (wait for projection sync)
		orgIDStr := org.OrganizationID.String()
		helpers.Wait(t, func() bool {
			fraudstersResp, err := ctx.AdminClient.AdminListFraudsters()
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
	})

	t.Run("ADM-018_with_reason", func(t *testing.T) {
		org := fixtures.NewActiveOrganization(t, ctx.AnonClient, ctx.AdminClient).Create()

		reason := "Multiple fraud reports received"
		resp, err := ctx.AdminClient.AdminMarkFraudster(org.OrganizationID, true, reason)
		require.NoError(t, err)
		require.Equal(t, http.StatusNoContent, resp.StatusCode, string(resp.RawBody))
	})
}

// TestAdminUnmarkFraudster tests POST /api/v1/admin/organizations/{id}/unmark-fraudster
func TestAdminUnmarkFraudster(t *testing.T) {
	t.Parallel()
	suite := getSuite(t)
	ctx := fixtures.NewTestContext(t, suite.BaseURL)

	t.Run("ADM-019_unmark_fraudster", func(t *testing.T) {
		// Create org and mark as fraudster
		org := fixtures.NewActiveOrganization(t, ctx.AnonClient, ctx.AdminClient).Create()

		_, err := ctx.AdminClient.AdminMarkFraudster(org.OrganizationID, true, "Test")
		require.NoError(t, err)

		// Unmark
		resp, err := ctx.AdminClient.AdminUnmarkFraudster(org.OrganizationID, "No longer fraudster")
		require.NoError(t, err)
		require.Equal(t, http.StatusNoContent, resp.StatusCode, string(resp.RawBody))
	})
}

// TestAdminListFraudsters tests GET /api/v1/admin/fraudsters
func TestAdminListFraudsters(t *testing.T) {
	t.Parallel()
	suite := getSuite(t)
	ctx := fixtures.NewTestContext(t, suite.BaseURL)

	t.Run("ADM-020_list_fraudsters", func(t *testing.T) {
		// Create and mark as fraudster
		org := fixtures.NewActiveOrganization(t, ctx.AnonClient, ctx.AdminClient).Create()
		_, err := ctx.AdminClient.AdminMarkFraudster(org.OrganizationID, true, "Test")
		require.NoError(t, err)

		// Wait for projection sync and verify org is in list
		orgIDStr := org.OrganizationID.String()
		helpers.Wait(t, func() bool {
			resp, err := ctx.AdminClient.AdminListFraudsters()
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
	})

	t.Run("ADM-021_empty_list", func(t *testing.T) {
		// Just check endpoint works
		resp, err := ctx.AdminClient.AdminListFraudsters()
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)
	})
}

// TestAdminGetReviews tests GET /api/v1/admin/reviews
func TestAdminGetReviews(t *testing.T) {
	t.Parallel()
	suite := getSuite(t)
	ctx := fixtures.NewTestContext(t, suite.BaseURL)

	t.Run("ADM-022_list_pending_reviews", func(t *testing.T) {
		resp, err := ctx.AdminClient.AdminGetReviews("pending")
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("ADM-023_filter_by_status", func(t *testing.T) {
		resp, err := ctx.AdminClient.AdminGetReviews("pending")
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)

		for _, r := range resp.Body {
			assert.Equal(t, "pending", r.Status)
		}
	})
}

// TestAdminGetReview tests GET /api/v1/admin/reviews/{id}
func TestAdminGetReview(t *testing.T) {
	t.Parallel()
	suite := getSuite(t)
	ctx := fixtures.NewTestContext(t, suite.BaseURL)

	t.Run("ADM-025_nonexistent_review", func(t *testing.T) {
		resp, err := ctx.AdminClient.AdminGetReview(uuid.New())
		require.NoError(t, err)
		require.Equal(t, http.StatusNotFound, resp.StatusCode)
	})
}

// TestAdminWithUserSession tests that user session cannot access admin endpoints
func TestAdminWithUserSession(t *testing.T) {
	t.Parallel()
	suite := getSuite(t)
	ctx := fixtures.NewTestContext(t, suite.BaseURL)

	endpoints := []struct {
		name   string
		call   func() int
	}{
		{
			name: "list organizations",
			call: func() int {
				resp, _ := ctx.Customer.Client.AdminGetOrganizations("")
				return resp.StatusCode
			},
		},
		{
			name: "list fraudsters",
			call: func() int {
				resp, _ := ctx.Customer.Client.AdminListFraudsters()
				return resp.StatusCode
			},
		},
		{
			name: "list reviews",
			call: func() int {
				resp, _ := ctx.Customer.Client.AdminGetReviews("")
				return resp.StatusCode
			},
		},
	}

	for _, ep := range endpoints {
		t.Run(ep.name, func(t *testing.T) {
			status := ep.call()
			assert.Equal(t, http.StatusUnauthorized, status)
		})
	}
}
