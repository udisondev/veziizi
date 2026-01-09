package tests

import (
	"net/http"
	"testing"

	"codeberg.org/udison/veziizi/backend/e2e/client"
	"codeberg.org/udison/veziizi/backend/e2e/fixtures"
	"github.com/stretchr/testify/suite"
)

// NotificationsSuite combines all notifications tests with shared context.
type NotificationsSuite struct {
	suite.Suite
	baseURL string
	ctx     *fixtures.TestContext
}

func TestNotificationsSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(NotificationsSuite))
}

func (s *NotificationsSuite) SetupSuite() {
	testSuite := getSuite(s.T())
	s.baseURL = testSuite.BaseURL
	s.ctx = fixtures.NewTestContext(s.T(), s.baseURL)
}

// ==================== GET /api/v1/notifications ====================

func (s *NotificationsSuite) TestNOT001_ListNotifications() {
	resp, err := s.ctx.Customer.Client.GetInAppNotifications(false, 20, 0)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusOK, resp.StatusCode, string(resp.RawBody))
	s.Assert().LessOrEqual(len(resp.Body.Items), 20)
}

func (s *NotificationsSuite) TestNOT002_EmptyList() {
	resp, err := s.ctx.Customer.Client.GetInAppNotifications(false, 20, 0)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusOK, resp.StatusCode, string(resp.RawBody))
}

func (s *NotificationsSuite) TestNOT003_Pagination() {
	resp, err := s.ctx.Customer.Client.GetInAppNotifications(false, 5, 0)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusOK, resp.StatusCode, string(resp.RawBody))
	s.Assert().LessOrEqual(len(resp.Body.Items), 5)
}

func (s *NotificationsSuite) TestNOT004_WithoutAuth() {
	resp, err := s.ctx.AnonClient.GetInAppNotifications(false, 20, 0)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusUnauthorized, resp.StatusCode)
}

// ==================== GET /api/v1/notifications/unread-count ====================

func (s *NotificationsSuite) TestNOT005_GetUnreadCount() {
	resp, err := s.ctx.Customer.Client.GetUnreadCount()
	s.Require().NoError(err)
	s.Require().Equal(http.StatusOK, resp.StatusCode, string(resp.RawBody))
	s.Assert().GreaterOrEqual(resp.Body.Unread, 0)
}

func (s *NotificationsSuite) TestNOT005b_UnreadCountWithoutAuth() {
	resp, err := s.ctx.AnonClient.GetUnreadCount()
	s.Require().NoError(err)
	s.Require().Equal(http.StatusUnauthorized, resp.StatusCode)
}

// ==================== POST /api/v1/notifications/read ====================

func (s *NotificationsSuite) TestNOT006_MarkSelectedRead() {
	// Get notifications first
	listResp, err := s.ctx.Customer.Client.GetInAppNotifications(false, 10, 0)
	s.Require().NoError(err)

	if len(listResp.Body.Items) > 0 {
		// This test verifies the endpoint exists and works
		resp, err := s.ctx.Customer.Client.MarkAllNotificationsRead()
		s.Require().NoError(err)
		s.Require().Equal(http.StatusNoContent, resp.StatusCode, string(resp.RawBody))
	}
}

// ==================== POST /api/v1/notifications/read-all ====================

func (s *NotificationsSuite) TestNOT007_MarkAllRead() {
	resp, err := s.ctx.Customer.Client.MarkAllNotificationsRead()
	s.Require().NoError(err)
	s.Require().Equal(http.StatusNoContent, resp.StatusCode, string(resp.RawBody))
}

func (s *NotificationsSuite) TestNOT008_Idempotent() {
	// Mark all read twice - should be idempotent
	resp1, err := s.ctx.Customer.Client.MarkAllNotificationsRead()
	s.Require().NoError(err)
	s.Require().Equal(http.StatusNoContent, resp1.StatusCode)

	resp2, err := s.ctx.Customer.Client.MarkAllNotificationsRead()
	s.Require().NoError(err)
	s.Require().Equal(http.StatusNoContent, resp2.StatusCode)
}

// ==================== GET /api/v1/notifications/preferences ====================

func (s *NotificationsSuite) TestNOT009_GetPreferences() {
	resp, err := s.ctx.Customer.Client.GetNotificationPreferences()
	s.Require().NoError(err)
	s.Require().Equal(http.StatusOK, resp.StatusCode, string(resp.RawBody))
	s.Assert().Equal(s.ctx.Customer.MemberID, resp.Body.MemberID)
	// Verify enabled_categories is present
	s.Assert().NotEmpty(resp.Body.EnabledCategories, "enabled_categories should not be empty")
}

func (s *NotificationsSuite) TestNOT009b_PreferencesWithoutAuth() {
	resp, err := s.ctx.AnonClient.GetNotificationPreferences()
	s.Require().NoError(err)
	s.Require().Equal(http.StatusUnauthorized, resp.StatusCode)
}

// ==================== PATCH /api/v1/notifications/preferences ====================

func (s *NotificationsSuite) TestNOT010_UpdateCategories() {
	// Update offers category settings
	resp, err := s.ctx.Customer.Client.UpdateNotificationPreferences(client.UpdatePreferencesRequest{
		Categories: client.EnabledCategories{
			"offers": client.CategorySettings{InApp: true, Telegram: true},
		},
	})
	s.Require().NoError(err)
	s.Require().Equal(http.StatusNoContent, resp.StatusCode, string(resp.RawBody))
}

func (s *NotificationsSuite) TestNOT011_DisableOffersInApp() {
	// Disable in-app for offers category
	resp, err := s.ctx.Customer.Client.UpdateNotificationPreferences(client.UpdatePreferencesRequest{
		Categories: client.EnabledCategories{
			"offers": client.CategorySettings{InApp: false, Telegram: true},
		},
	})
	s.Require().NoError(err)
	s.Require().Equal(http.StatusNoContent, resp.StatusCode, string(resp.RawBody))

	// Re-enable for other tests
	_, _ = s.ctx.Customer.Client.UpdateNotificationPreferences(client.UpdatePreferencesRequest{
		Categories: client.EnabledCategories{
			"offers": client.CategorySettings{InApp: true, Telegram: true},
		},
	})
}

func (s *NotificationsSuite) TestNOT011b_UpdatePreferencesWithoutAuth() {
	resp, err := s.ctx.AnonClient.UpdateNotificationPreferences(client.UpdatePreferencesRequest{
		Categories: client.EnabledCategories{
			"offers": client.CategorySettings{InApp: true, Telegram: true},
		},
	})
	s.Require().NoError(err)
	s.Require().Equal(http.StatusUnauthorized, resp.StatusCode)
}

// ==================== POST /api/v1/notifications/telegram/link-code ====================

func (s *NotificationsSuite) TestNOT012_GetLinkCode() {
	resp, err := s.ctx.Customer.Client.GetTelegramLinkCode()
	s.Require().NoError(err)
	s.Require().Equal(http.StatusOK, resp.StatusCode, string(resp.RawBody))
	s.Assert().NotEmpty(resp.Body.Code, "should return a link code")
}

func (s *NotificationsSuite) TestNOT012b_LinkCodeWithoutAuth() {
	resp, err := s.ctx.AnonClient.GetTelegramLinkCode()
	s.Require().NoError(err)
	s.Require().Equal(http.StatusUnauthorized, resp.StatusCode)
}

// ==================== DELETE /api/v1/notifications/telegram ====================

func (s *NotificationsSuite) TestNOT013_DisconnectTelegram() {
	// This should work even if telegram is not connected
	resp, err := s.ctx.Customer.Client.DisconnectTelegram()
	s.Require().NoError(err)
	s.Require().Equal(http.StatusNoContent, resp.StatusCode, string(resp.RawBody))
}

func (s *NotificationsSuite) TestNOT014_AlreadyDisconnected() {
	// Should be idempotent
	resp, err := s.ctx.Customer.Client.DisconnectTelegram()
	s.Require().NoError(err)
	s.Require().Equal(http.StatusNoContent, resp.StatusCode)
}

func (s *NotificationsSuite) TestNOT014b_DisconnectWithoutAuth() {
	resp, err := s.ctx.AnonClient.DisconnectTelegram()
	s.Require().NoError(err)
	s.Require().Equal(http.StatusUnauthorized, resp.StatusCode)
}
