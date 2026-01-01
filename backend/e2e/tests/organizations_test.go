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

// TestRegisterOrganization tests POST /api/v1/organizations
func TestRegisterOrganization(t *testing.T) {
	t.Parallel()
	c := getClient(t)

	tests := []struct {
		id          string
		name        string
		modify      func(*fixtures.OrganizationBuilder)
		wantStatus  int
		wantErr     string
		checkResult func(*testing.T, *client.Response[client.RegisterOrganizationResponse])
	}{
		// Happy path - different countries
		{
			id:         "ORG-001",
			name:       "register RU organization",
			modify:     func(b *fixtures.OrganizationBuilder) { b.WithCountry("RU") },
			wantStatus: http.StatusCreated,
			checkResult: func(t *testing.T, resp *client.Response[client.RegisterOrganizationResponse]) {
				assert.NotEmpty(t, resp.Body.OrganizationID.String(), "organization_id")
				assert.NotEmpty(t, resp.Body.MemberID.String(), "member_id")
			},
		},
		{
			id:         "ORG-002",
			name:       "register KZ organization",
			modify:     func(b *fixtures.OrganizationBuilder) { b.WithCountry("KZ") },
			wantStatus: http.StatusCreated,
		},
		{
			id:         "ORG-003",
			name:       "register BY organization",
			modify:     func(b *fixtures.OrganizationBuilder) { b.WithCountry("BY") },
			wantStatus: http.StatusCreated,
		},

		// Validation errors (400)
		{
			id:         "ORG-009",
			name:       "invalid country code",
			modify:     func(b *fixtures.OrganizationBuilder) { b.WithCountry("XX") },
			wantStatus: http.StatusBadRequest,
			wantErr:    "invalid country",
		},
		{
			id:         "ORG-010",
			name:       "empty country",
			modify:     func(b *fixtures.OrganizationBuilder) { b.WithCountry("") },
			wantStatus: http.StatusBadRequest,
		},

		// Edge cases
		{
			id:   "ORG-019",
			name: "SQL injection in name",
			modify: func(b *fixtures.OrganizationBuilder) {
				b.WithName("'; DROP TABLE--")
			},
			wantStatus: http.StatusCreated, // Should be safely stored
		},
		{
			id:   "ORG-022",
			name: "unicode in data",
			modify: func(b *fixtures.OrganizationBuilder) {
				b.WithName("中文公司 🚛")
				b.WithAddress("北京市朝阳区")
			},
			wantStatus: http.StatusCreated,
		},
	}

	for _, tt := range tests {
		t.Run(tt.id+"_"+tt.name, func(t *testing.T) {
			t.Parallel()

			builder := fixtures.NewOrganization(t, c)
			if tt.modify != nil {
				tt.modify(builder)
			}

			resp, err := builder.CreateWithStatus()
			require.NoError(t, err)
			require.Equal(t, tt.wantStatus, resp.StatusCode, string(resp.RawBody))

			if tt.wantErr != "" {
				assert.Contains(t, strings.ToLower(string(resp.RawBody)), strings.ToLower(tt.wantErr))
			}

			if tt.checkResult != nil && resp.StatusCode == http.StatusCreated {
				tt.checkResult(t, resp)
			}
		})
	}
}

// TestGetOrganization tests GET /api/v1/organizations/{id}
func TestGetOrganization(t *testing.T) {
	t.Parallel()
	c := getClient(t)

	org := fixtures.NewOrganization(t, c).Create()

	tests := []struct {
		id         string
		name       string
		orgID      string
		wantStatus int
		check      func(*testing.T, *client.Response[client.OrganizationResponse])
	}{
		{
			id:         "ORG-026",
			name:       "get existing organization",
			orgID:      org.OrganizationID.String(),
			wantStatus: http.StatusOK,
			check: func(t *testing.T, resp *client.Response[client.OrganizationResponse]) {
				assert.Equal(t, org.OrganizationID, resp.Body.ID, "id")
				assert.Equal(t, "pending", resp.Body.Status, "status")
			},
		},
		{
			id:         "ORG-028",
			name:       "get without auth (public endpoint)",
			orgID:      org.OrganizationID.String(),
			wantStatus: http.StatusOK,
		},
		{
			id:         "ORG-030",
			name:       "invalid UUID",
			orgID:      "not-a-uuid",
			wantStatus: http.StatusBadRequest,
		},
		{
			id:         "ORG-031",
			name:       "nonexistent organization",
			orgID:      uuid.New().String(),
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.id+"_"+tt.name, func(t *testing.T) {
			testClient := c.Clone()

			var resp *client.Response[client.OrganizationResponse]
			var err error

			orgID, parseErr := uuid.Parse(tt.orgID)
			if parseErr != nil {
				// For invalid UUID tests, use Raw request
				status, body, err := testClient.Raw(http.MethodGet, "/api/v1/organizations/"+tt.orgID, nil, nil)
				require.NoError(t, err)
				require.Equal(t, tt.wantStatus, status, string(body))
				return
			}

			resp, err = testClient.GetOrganization(orgID)
			require.NoError(t, err)
			require.Equal(t, tt.wantStatus, resp.StatusCode, string(resp.RawBody))

			if tt.check != nil && resp.StatusCode == http.StatusOK {
				tt.check(t, resp)
			}
		})
	}
}

// TestGetOrganizationFull tests GET /api/v1/organizations/{id}/full
func TestGetOrganizationFull(t *testing.T) {
	t.Parallel()
	c := getClient(t)

	org := fixtures.NewOrganization(t, c).Create()
	otherOrg := fixtures.NewOrganization(t, c).Create()

	tests := []struct {
		id         string
		name       string
		client     *client.Client
		orgID      uuid.UUID
		wantStatus int
		check      func(*testing.T, *client.Response[client.OrganizationFullResponse])
	}{
		{
			id:         "ORG-034",
			name:       "get own organization with members",
			client:     org.Client,
			orgID:      org.OrganizationID,
			wantStatus: http.StatusOK,
			check: func(t *testing.T, resp *client.Response[client.OrganizationFullResponse]) {
				assert.True(t, len(resp.Body.Members) > 0, "should have members")
			},
		},
		{
			id:         "ORG-036",
			name:       "without auth",
			client:     c,
			orgID:      org.OrganizationID,
			wantStatus: http.StatusUnauthorized,
		},
		{
			id:         "ORG-037",
			name:       "different organization",
			client:     otherOrg.Client,
			orgID:      org.OrganizationID,
			wantStatus: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.id+"_"+tt.name, func(t *testing.T) {
			resp, err := tt.client.GetOrganizationFull(tt.orgID)
			require.NoError(t, err)
			require.Equal(t, tt.wantStatus, resp.StatusCode, string(resp.RawBody))

			if tt.check != nil && resp.StatusCode == http.StatusOK {
				tt.check(t, resp)
			}
		})
	}
}

// TestOrganizationRating tests GET /api/v1/organizations/{id}/rating
func TestOrganizationRating(t *testing.T) {
	t.Parallel()
	c := getClient(t)

	org := fixtures.NewOrganization(t, c).Create()

	tests := []struct {
		id         string
		name       string
		orgID      uuid.UUID
		wantStatus int
		check      func(*testing.T, *client.Response[client.RatingResponse])
	}{
		{
			id:         "ORG-040",
			name:       "rating without reviews",
			orgID:      org.OrganizationID,
			wantStatus: http.StatusOK,
			check: func(t *testing.T, resp *client.Response[client.RatingResponse]) {
				assert.Equal(t, 0, resp.Body.TotalReviews, "total_reviews")
				assert.Equal(t, 0.0, resp.Body.AverageRating, "average_rating")
			},
		},
		{
			id:         "ORG-042",
			name:       "public access (no auth)",
			orgID:      org.OrganizationID,
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.id+"_"+tt.name, func(t *testing.T) {
			resp, err := c.GetOrganizationRating(tt.orgID)
			require.NoError(t, err)
			require.Equal(t, tt.wantStatus, resp.StatusCode, string(resp.RawBody))

			if tt.check != nil && resp.StatusCode == http.StatusOK {
				tt.check(t, resp)
			}
		})
	}
}

// TestCreateInvitation tests POST /api/v1/organizations/{id}/invitations
func TestCreateInvitation(t *testing.T) {
	t.Parallel()
	c := getClient(t)

	org := fixtures.NewOrganization(t, c).Create()
	otherOrg := fixtures.NewOrganization(t, c).Create()

	tests := []struct {
		id         string
		name       string
		client     *client.Client
		orgID      uuid.UUID
		email      string
		role       string
		wantStatus int
		wantErr    string
	}{
		{
			id:         "ORG-052",
			name:       "create administrator invitation",
			client:     org.Client,
			orgID:      org.OrganizationID,
			email:      helpers.RandomEmail(),
			role:       "administrator",
			wantStatus: http.StatusCreated,
		},
		{
			id:         "ORG-053",
			name:       "create employee invitation",
			client:     org.Client,
			orgID:      org.OrganizationID,
			email:      helpers.RandomEmail(),
			role:       "employee",
			wantStatus: http.StatusCreated,
		},
		{
			id:         "ORG-056",
			name:       "invalid role",
			client:     org.Client,
			orgID:      org.OrganizationID,
			email:      helpers.RandomEmail(),
			role:       "superadmin",
			wantStatus: http.StatusBadRequest,
			wantErr:    "invalid role",
		},
		{
			id:         "ORG-060",
			name:       "without auth",
			client:     c,
			orgID:      org.OrganizationID,
			email:      helpers.RandomEmail(),
			role:       "employee",
			wantStatus: http.StatusUnauthorized,
		},
		{
			id:         "ORG-061",
			name:       "different organization",
			client:     otherOrg.Client,
			orgID:      org.OrganizationID,
			email:      helpers.RandomEmail(),
			role:       "employee",
			wantStatus: http.StatusNotFound, // member not found in target org
			wantErr:    "member not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.id+"_"+tt.name, func(t *testing.T) {
			resp, err := tt.client.CreateInvitation(tt.orgID, client.CreateInvitationRequest{
				Email: tt.email,
				Role:  tt.role,
			})
			require.NoError(t, err)
			require.Equal(t, tt.wantStatus, resp.StatusCode, string(resp.RawBody))

			if tt.wantErr != "" {
				assert.Contains(t, strings.ToLower(string(resp.RawBody)), strings.ToLower(tt.wantErr))
			}
		})
	}
}

// TestDuplicateInvitation tests ORG-064: Email already invited
func TestDuplicateInvitation(t *testing.T) {
	t.Parallel()
	c := getClient(t)

	org := fixtures.NewOrganization(t, c).Create()
	email := helpers.RandomEmail()

	// First invitation
	resp1, err := org.Client.CreateInvitation(org.OrganizationID, client.CreateInvitationRequest{
		Email: email,
		Role:  "employee",
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp1.StatusCode, string(resp1.RawBody))

	// Second invitation with same email
	resp2, err := org.Client.CreateInvitation(org.OrganizationID, client.CreateInvitationRequest{
		Email: email,
		Role:  "employee",
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusConflict, resp2.StatusCode, string(resp2.RawBody))
	assert.Contains(t, strings.ToLower(string(resp2.RawBody)), "already invited")
}

// TestAcceptInvitation tests POST /api/v1/invitations/{token}/accept
func TestAcceptInvitation(t *testing.T) {
	t.Parallel()
	c := getClient(t)

	org := fixtures.NewOrganization(t, c).Create()

	// Create invitation for successful accept
	email := helpers.RandomEmail()
	invResp, _ := org.Client.CreateInvitation(org.OrganizationID, client.CreateInvitationRequest{
		Email: email,
		Role:  "employee",
	})
	token := invResp.Body.Token

	// Create second invitation for empty password test
	email2 := helpers.RandomEmail()
	invResp2, _ := org.Client.CreateInvitation(org.OrganizationID, client.CreateInvitationRequest{
		Email: email2,
		Role:  "employee",
	})
	token2 := invResp2.Body.Token

	// Wait for invitations to be available in lookup
	helpers.WaitFor(t, func() (bool, bool) {
		getResp, err := c.GetInvitationByToken(token)
		return err == nil && getResp.StatusCode == 200, err == nil && getResp.StatusCode == 200
	}, "invitation 1 should be available")

	helpers.WaitFor(t, func() (bool, bool) {
		getResp, err := c.GetInvitationByToken(token2)
		return err == nil && getResp.StatusCode == 200, err == nil && getResp.StatusCode == 200
	}, "invitation 2 should be available")

	t.Run("ORG-082_successful_accept", func(t *testing.T) {
		resp, err := c.AcceptInvitation(token, client.AcceptInvitationRequest{
			Password: "password123",
			Name:     helpers.StringPtr("New Member"),
			Phone:    helpers.StringPtr("+79001234567"),
		})
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode, string(resp.RawBody))
	})

	t.Run("ORG-085_empty_password", func(t *testing.T) {
		resp, err := c.AcceptInvitation(token2, client.AcceptInvitationRequest{
			Password: "",
		})
		require.NoError(t, err)
		require.Equal(t, http.StatusBadRequest, resp.StatusCode, string(resp.RawBody))
	})

	t.Run("ORG-088_nonexistent_token", func(t *testing.T) {
		resp, err := c.AcceptInvitation("nonexistent-token", client.AcceptInvitationRequest{
			Password: "password123",
			Name:     helpers.StringPtr("Test"),
			Phone:    helpers.StringPtr("+79001234567"),
		})
		require.NoError(t, err)
		require.Equal(t, http.StatusNotFound, resp.StatusCode, string(resp.RawBody))
	})
}

// TestBlockUnblockMember tests member blocking functionality
func TestBlockUnblockMember(t *testing.T) {
	t.Parallel()
	c := getClient(t)

	org := fixtures.NewOrganization(t, c).Create()

	// Create and accept invitation to have a second member
	email := helpers.RandomEmail()
	invResp, _ := org.Client.CreateInvitation(org.OrganizationID, client.CreateInvitationRequest{
		Email: email,
		Role:  "employee",
	})

	// Wait for invitation to be available in lookup
	token := invResp.Body.Token
	helpers.WaitFor(t, func() (bool, bool) {
		getResp, err := c.GetInvitationByToken(token)
		return err == nil && getResp.StatusCode == 200, err == nil && getResp.StatusCode == 200
	}, "invitation should be available")

	name := "Member to Block"
	phone := "+79001234567"
	acceptResp, _ := c.AcceptInvitation(token, client.AcceptInvitationRequest{
		Password: "password123",
		Name:     &name,
		Phone:    &phone,
	})
	memberID := acceptResp.Body.MemberID

	// Wait for member to be available in lookup
	helpers.WaitFor(t, func() (bool, bool) {
		// Check if member can be found via organization API
		meResp, err := c.Login(email, "password123")
		return err == nil && meResp.StatusCode == 200, err == nil && meResp.StatusCode == 200
	}, "member should be available")

	t.Run("ORG-101_owner_blocks_member", func(t *testing.T) {
		resp, err := org.Client.BlockMember(org.OrganizationID, memberID, "test block reason")
		require.NoError(t, err)
		require.Equal(t, http.StatusNoContent, resp.StatusCode, string(resp.RawBody))
	})

	t.Run("ORG-110_owner_unblocks_member", func(t *testing.T) {
		resp, err := org.Client.UnblockMember(org.OrganizationID, memberID)
		require.NoError(t, err)
		require.Equal(t, http.StatusNoContent, resp.StatusCode, string(resp.RawBody))
	})

	t.Run("ORG-104_cannot_block_self", func(t *testing.T) {
		resp, err := org.Client.BlockMember(org.OrganizationID, org.MemberID, "test block reason")
		require.NoError(t, err)
		require.Equal(t, http.StatusBadRequest, resp.StatusCode, string(resp.RawBody))
		assert.Contains(t, strings.ToLower(string(resp.RawBody)), "cannot block yourself")
	})
}
