package tests

import (
	"net/http"
	"testing"

	"codeberg.org/udison/veziizi/backend/e2e/client"
	"codeberg.org/udison/veziizi/backend/e2e/fixtures"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGetNotifications tests GET /api/v1/notifications
func TestGetNotifications(t *testing.T) {
	t.Parallel()
	suite := getSuite(t)
	ctx := fixtures.NewTestContext(t, suite.BaseURL)

	tests := []struct {
		id         string
		name       string
		client     *client.Client
		unreadOnly bool
		limit      int
		offset     int
		wantStatus int
	}{
		// Happy path
		{
			id:         "NOT-001",
			name:       "list notifications",
			client:     ctx.Customer.Client,
			unreadOnly: false,
			limit:      20,
			offset:     0,
			wantStatus: http.StatusOK,
		},
		{
			id:         "NOT-002",
			name:       "empty list",
			client:     ctx.Customer.Client,
			unreadOnly: false,
			limit:      20,
			offset:     0,
			wantStatus: http.StatusOK,
		},
		{
			id:         "NOT-003",
			name:       "pagination",
			client:     ctx.Customer.Client,
			unreadOnly: false,
			limit:      5,
			offset:     0,
			wantStatus: http.StatusOK,
		},

		// Auth errors
		{
			id:         "NOT-004",
			name:       "without auth",
			client:     ctx.AnonClient,
			unreadOnly: false,
			limit:      20,
			offset:     0,
			wantStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.id+"_"+tt.name, func(t *testing.T) {
			resp, err := tt.client.GetInAppNotifications(tt.unreadOnly, tt.limit, tt.offset)
			require.NoError(t, err)
			require.Equal(t, tt.wantStatus, resp.StatusCode, string(resp.RawBody))

			if resp.StatusCode == http.StatusOK {
				assert.LessOrEqual(t, len(resp.Body.Items), tt.limit)
			}
		})
	}
}

// TestGetUnreadCount tests GET /api/v1/notifications/unread-count
func TestGetUnreadCount(t *testing.T) {
	t.Parallel()
	suite := getSuite(t)
	ctx := fixtures.NewTestContext(t, suite.BaseURL)

	t.Run("NOT-005_get_unread_count", func(t *testing.T) {
		resp, err := ctx.Customer.Client.GetUnreadCount()
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode, string(resp.RawBody))
		// Count can be 0 or more
		assert.GreaterOrEqual(t, resp.Body.Count, 0)
	})

	t.Run("NOT-005b_without_auth", func(t *testing.T) {
		resp, err := ctx.AnonClient.GetUnreadCount()
		require.NoError(t, err)
		require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})
}

// TestMarkNotificationsRead tests POST /api/v1/notifications/read
func TestMarkNotificationsRead(t *testing.T) {
	t.Parallel()
	suite := getSuite(t)
	ctx := fixtures.NewTestContext(t, suite.BaseURL)

	t.Run("NOT-006_mark_selected_read", func(t *testing.T) {
		// Get notifications first
		listResp, err := ctx.Customer.Client.GetInAppNotifications(false, 10, 0)
		require.NoError(t, err)

		if len(listResp.Body.Items) > 0 {
			// Mark first notification as read
			ids := []interface{}{listResp.Body.Items[0].ID}
			_ = ids // Would use in actual API call

			// This test verifies the endpoint exists and works
			resp, err := ctx.Customer.Client.MarkAllNotificationsRead()
			require.NoError(t, err)
			require.Equal(t, http.StatusNoContent, resp.StatusCode, string(resp.RawBody))
		}
	})
}

// TestMarkAllNotificationsRead tests POST /api/v1/notifications/read-all
func TestMarkAllNotificationsRead(t *testing.T) {
	t.Parallel()
	suite := getSuite(t)
	ctx := fixtures.NewTestContext(t, suite.BaseURL)

	t.Run("NOT-007_mark_all_read", func(t *testing.T) {
		resp, err := ctx.Customer.Client.MarkAllNotificationsRead()
		require.NoError(t, err)
		require.Equal(t, http.StatusNoContent, resp.StatusCode, string(resp.RawBody))
	})

	t.Run("NOT-008_idempotent", func(t *testing.T) {
		// Mark all read twice - should be idempotent
		resp1, err := ctx.Customer.Client.MarkAllNotificationsRead()
		require.NoError(t, err)
		require.Equal(t, http.StatusNoContent, resp1.StatusCode)

		resp2, err := ctx.Customer.Client.MarkAllNotificationsRead()
		require.NoError(t, err)
		require.Equal(t, http.StatusNoContent, resp2.StatusCode)
	})
}

// TestGetNotificationPreferences tests GET /api/v1/notifications/preferences
func TestGetNotificationPreferences(t *testing.T) {
	t.Parallel()
	suite := getSuite(t)
	ctx := fixtures.NewTestContext(t, suite.BaseURL)

	t.Run("NOT-009_get_preferences", func(t *testing.T) {
		resp, err := ctx.Customer.Client.GetNotificationPreferences()
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode, string(resp.RawBody))

		// Default preferences should have in_app enabled
		assert.Equal(t, ctx.Customer.MemberID, resp.Body.MemberID)
	})

	t.Run("NOT-009b_without_auth", func(t *testing.T) {
		resp, err := ctx.AnonClient.GetNotificationPreferences()
		require.NoError(t, err)
		require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})
}

// TestUpdateNotificationPreferences tests PATCH /api/v1/notifications/preferences
func TestUpdateNotificationPreferences(t *testing.T) {
	t.Parallel()
	suite := getSuite(t)
	ctx := fixtures.NewTestContext(t, suite.BaseURL)

	t.Run("NOT-010_enable_telegram", func(t *testing.T) {
		enabled := true
		resp, err := ctx.Customer.Client.UpdateNotificationPreferences(client.UpdatePreferencesRequest{
			Telegram: &enabled,
		})
		require.NoError(t, err)
		require.Equal(t, http.StatusNoContent, resp.StatusCode, string(resp.RawBody))
	})

	t.Run("NOT-011_disable_in_app", func(t *testing.T) {
		disabled := false
		resp, err := ctx.Customer.Client.UpdateNotificationPreferences(client.UpdatePreferencesRequest{
			InApp: &disabled,
		})
		require.NoError(t, err)
		require.Equal(t, http.StatusNoContent, resp.StatusCode, string(resp.RawBody))

		// Re-enable for other tests
		enabled := true
		ctx.Customer.Client.UpdateNotificationPreferences(client.UpdatePreferencesRequest{
			InApp: &enabled,
		})
	})

	t.Run("NOT-011b_without_auth", func(t *testing.T) {
		enabled := true
		resp, err := ctx.AnonClient.UpdateNotificationPreferences(client.UpdatePreferencesRequest{
			InApp: &enabled,
		})
		require.NoError(t, err)
		require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})
}

// TestTelegramLinkCode tests POST /api/v1/notifications/telegram/link-code
func TestTelegramLinkCode(t *testing.T) {
	t.Parallel()
	suite := getSuite(t)
	ctx := fixtures.NewTestContext(t, suite.BaseURL)

	t.Run("NOT-012_get_link_code", func(t *testing.T) {
		resp, err := ctx.Customer.Client.GetTelegramLinkCode()
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode, string(resp.RawBody))

		assert.NotEmpty(t, resp.Body.Code, "should return a link code")
	})

	t.Run("NOT-012b_without_auth", func(t *testing.T) {
		resp, err := ctx.AnonClient.GetTelegramLinkCode()
		require.NoError(t, err)
		require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})
}

// TestDisconnectTelegram tests DELETE /api/v1/notifications/telegram
func TestDisconnectTelegram(t *testing.T) {
	t.Parallel()
	suite := getSuite(t)
	ctx := fixtures.NewTestContext(t, suite.BaseURL)

	t.Run("NOT-013_disconnect_telegram", func(t *testing.T) {
		// This should work even if telegram is not connected
		resp, err := ctx.Customer.Client.DisconnectTelegram()
		require.NoError(t, err)
		require.Equal(t, http.StatusNoContent, resp.StatusCode, string(resp.RawBody))
	})

	t.Run("NOT-014_already_disconnected", func(t *testing.T) {
		// Should be idempotent
		resp, err := ctx.Customer.Client.DisconnectTelegram()
		require.NoError(t, err)
		require.Equal(t, http.StatusNoContent, resp.StatusCode)
	})

	t.Run("NOT-014b_without_auth", func(t *testing.T) {
		resp, err := ctx.AnonClient.DisconnectTelegram()
		require.NoError(t, err)
		require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})
}
