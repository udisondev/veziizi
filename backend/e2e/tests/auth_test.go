package tests

import (
	"net/http"
	"strings"
	"testing"

	"codeberg.org/udison/veziizi/backend/e2e/client"
	"codeberg.org/udison/veziizi/backend/e2e/fixtures"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLogin tests POST /api/v1/auth/login
func TestLogin(t *testing.T) {
	t.Parallel()
	c := getClient(t)

	// Create a test organization to have a user to login with
	org := fixtures.NewOrganization(t, c).Create()

	// Table-driven tests for login scenarios
	tests := []struct {
		id          string // Test ID from E2E_TESTS.md
		name        string
		email       string
		password    string
		wantStatus  int
		wantErr     string
		checkCookie bool
	}{
		// Happy path
		{
			id:          "AUTH-001",
			name:        "successful login",
			email:       org.OwnerEmail,
			password:    org.OwnerPassword,
			wantStatus:  http.StatusOK,
			checkCookie: true,
		},

		// Validation errors (400)
		{
			id:         "AUTH-005",
			name:       "missing email",
			email:      "",
			password:   "password123",
			wantStatus: http.StatusBadRequest,
		},
		{
			id:         "AUTH-006",
			name:       "missing password",
			email:      org.OwnerEmail,
			password:   "",
			wantStatus: http.StatusBadRequest,
		},

		// Auth errors (401)
		{
			id:         "AUTH-007",
			name:       "nonexistent email",
			email:      "nonexistent@test.local",
			password:   "password123",
			wantStatus: http.StatusUnauthorized,
			wantErr:    "invalid credentials",
		},
		{
			id:         "AUTH-008",
			name:       "wrong password",
			email:      org.OwnerEmail,
			password:   "wrongpassword",
			wantStatus: http.StatusUnauthorized,
			wantErr:    "invalid credentials",
		},
		{
			id:         "AUTH-016",
			name:       "empty password",
			email:      org.OwnerEmail,
			password:   "",
			wantStatus: http.StatusBadRequest,
		},

		// Edge cases
		{
			id:         "AUTH-012",
			name:       "SQL injection in email",
			email:      "'; DROP TABLE members--",
			password:   "password123",
			wantStatus: http.StatusUnauthorized, // Should be safely handled
		},
		{
			id:         "AUTH-014",
			name:       "unicode in email",
			email:      "тест@test.com",
			password:   "password123",
			wantStatus: http.StatusUnauthorized, // Email doesn't exist
		},
	}

	for _, tt := range tests {
		t.Run(tt.id+"_"+tt.name, func(t *testing.T) {
			// Use fresh client for each test to avoid session interference
			testClient := c.Clone()

			resp, err := testClient.Login(tt.email, tt.password)
			require.NoError(t, err)
			require.Equal(t, tt.wantStatus, resp.StatusCode, string(resp.RawBody))

			if tt.wantErr != "" {
				assert.Contains(t, strings.ToLower(string(resp.RawBody)), strings.ToLower(tt.wantErr))
			}

			if tt.checkCookie && resp.StatusCode == http.StatusOK {
				// Verify we got a session cookie
				assert.NotEmpty(t, resp.Body.Email, "email in response")
				assert.NotEmpty(t, resp.Body.MemberID.String(), "member_id should be set")
			}
		})
	}
}

// TestLogout tests POST /api/v1/auth/logout
func TestLogout(t *testing.T) {
	t.Parallel()
	c := getClient(t)

	org := fixtures.NewOrganization(t, c).Create()

	tests := []struct {
		id         string
		name       string
		setup      func(*client.Client)
		wantStatus int
	}{
		{
			id:   "AUTH-025",
			name: "successful logout",
			setup: func(c *client.Client) {
				// Login first
				c.Login(org.OwnerEmail, org.OwnerPassword)
			},
			wantStatus: http.StatusNoContent,
		},
		{
			id:         "AUTH-027",
			name:       "logout without session",
			setup:      func(c *client.Client) {},
			wantStatus: http.StatusUnauthorized, // Requires authentication
		},
	}

	for _, tt := range tests {
		t.Run(tt.id+"_"+tt.name, func(t *testing.T) {
			testClient := c.Clone()
			tt.setup(testClient)

			resp, err := testClient.Logout()
			require.NoError(t, err)
			require.Equal(t, tt.wantStatus, resp.StatusCode, string(resp.RawBody))
		})
	}
}

// TestMe tests GET /api/v1/auth/me
func TestMe(t *testing.T) {
	t.Parallel()
	c := getClient(t)

	org := fixtures.NewOrganization(t, c).Create()

	tests := []struct {
		id         string
		name       string
		setup      func(*client.Client)
		wantStatus int
		check      func(*testing.T, *client.Response[client.MeResponse])
	}{
		{
			id:   "AUTH-030",
			name: "get profile when logged in",
			setup: func(c *client.Client) {
				c.Login(org.OwnerEmail, org.OwnerPassword)
			},
			wantStatus: http.StatusOK,
			check: func(t *testing.T, resp *client.Response[client.MeResponse]) {
				assert.Equal(t, org.OwnerEmail, resp.Body.Email, "email")
				assert.Equal(t, "owner", resp.Body.Role, "role")
				assert.NotEmpty(t, resp.Body.OrganizationID.String(), "organization_id")
			},
		},
		{
			id:         "AUTH-036",
			name:       "get profile without auth",
			setup:      func(c *client.Client) {},
			wantStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.id+"_"+tt.name, func(t *testing.T) {
			testClient := c.Clone()
			tt.setup(testClient)

			resp, err := testClient.Me()
			require.NoError(t, err)
			require.Equal(t, tt.wantStatus, resp.StatusCode, string(resp.RawBody))

			if tt.check != nil && resp.StatusCode == http.StatusOK {
				tt.check(t, resp)
			}
		})
	}
}

// TestActionsAfterLogout tests AUTH-029: Actions after logout should fail
func TestActionsAfterLogout(t *testing.T) {
	t.Parallel()
	c := getClient(t)

	org := fixtures.NewOrganization(t, c).Create()

	// Login
	_, err := org.Client.Login(org.OwnerEmail, org.OwnerPassword)
	require.NoError(t, err)

	// Verify logged in
	meResp, _ := org.Client.Me()
	require.Equal(t, http.StatusOK, meResp.StatusCode, string(meResp.RawBody))

	// Logout
	logoutResp, _ := org.Client.Logout()
	require.Equal(t, http.StatusNoContent, logoutResp.StatusCode, string(logoutResp.RawBody))

	// Try to access protected endpoint
	meResp2, _ := org.Client.Me()
	require.Equal(t, http.StatusUnauthorized, meResp2.StatusCode, string(meResp2.RawBody))
}
