package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Client is a typed HTTP client for E2E testing.
type Client struct {
	BaseURL    string
	HTTPClient *http.Client
	cookies    []*http.Cookie
}

// New creates a new API client with the given base URL.
func New(baseURL string) *Client {
	jar, _ := cookiejar.New(nil)
	return &Client{
		BaseURL: baseURL,
		HTTPClient: &http.Client{
			Timeout: 10 * time.Second,
			Jar:     jar,
		},
	}
}

// Clone creates a new client with fresh cookies (for parallel tests).
func (c *Client) Clone() *Client {
	jar, _ := cookiejar.New(nil)
	return &Client{
		BaseURL: c.BaseURL,
		HTTPClient: &http.Client{
			Timeout: 10 * time.Second,
			Jar:     jar,
		},
	}
}

// do performs an HTTP request and decodes the response.
func (c *Client) do(method, path string, body any, headers map[string]string) (*http.Response, []byte, error) {
	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequest(method, c.BaseURL+path, reqBody)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Requested-With", "XMLHttpRequest") // CSRF header

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to perform request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Store cookies
	c.cookies = resp.Cookies()

	return resp, respBody, nil
}

// doRequest performs a request and returns a typed response.
func doRequest[T any](c *Client, method, path string, body any, headers map[string]string) (*Response[T], error) {
	resp, respBody, err := c.do(method, path, body, headers)
	if err != nil {
		return nil, err
	}

	result := &Response[T]{
		StatusCode: resp.StatusCode,
		RawBody:    respBody,
		Cookies:    make(map[string]string),
	}

	for _, cookie := range resp.Cookies() {
		result.Cookies[cookie.Name] = cookie.Value
	}

	if len(respBody) > 0 && resp.StatusCode < 400 {
		if err := json.Unmarshal(respBody, &result.Body); err != nil {
			// Not an error - some responses have no body
		}
	}

	return result, nil
}

// --- Auth Methods ---

// Login authenticates a user and stores session cookie.
func (c *Client) Login(email, password string) (*Response[LoginResponse], error) {
	return doRequest[LoginResponse](c, http.MethodPost, "/api/v1/auth/login", LoginRequest{
		Email:    email,
		Password: password,
	}, nil)
}

// Logout clears the session.
func (c *Client) Logout() (*Response[struct{}], error) {
	return doRequest[struct{}](c, http.MethodPost, "/api/v1/auth/logout", nil, nil)
}

// Me returns current user info.
func (c *Client) Me() (*Response[MeResponse], error) {
	return doRequest[MeResponse](c, http.MethodGet, "/api/v1/auth/me", nil, nil)
}

// ForgotPassword requests a password reset email.
func (c *Client) ForgotPassword(email string) (*Response[struct{}], error) {
	return doRequest[struct{}](c, http.MethodPost, "/api/v1/auth/forgot-password", ForgotPasswordRequest{
		Email: email,
	}, nil)
}

// ValidateResetToken validates a password reset token.
func (c *Client) ValidateResetToken(token string) (*Response[struct{}], error) {
	return doRequest[struct{}](c, http.MethodGet, "/api/v1/auth/reset-password/"+token, nil, nil)
}

// ResetPassword resets password using token.
func (c *Client) ResetPassword(token, newPassword string) (*Response[struct{}], error) {
	return doRequest[struct{}](c, http.MethodPost, "/api/v1/auth/reset-password", ResetPasswordRequest{
		Token:       token,
		NewPassword: newPassword,
	}, nil)
}

// GetMember returns a member's public profile.
func (c *Client) GetMember(id uuid.UUID) (*Response[MemberPublicProfile], error) {
	return doRequest[MemberPublicProfile](c, http.MethodGet, "/api/v1/members/"+id.String(), nil, nil)
}

// GetMemberProfile returns full member profile with organization details.
func (c *Client) GetMemberProfile(id uuid.UUID) (*Response[MemberProfileResponse], error) {
	return doRequest[MemberProfileResponse](c, http.MethodGet, "/api/v1/members/"+id.String(), nil, nil)
}

// --- Organization Methods ---

// RegisterOrganization creates a new organization with owner.
func (c *Client) RegisterOrganization(req RegisterOrganizationRequest) (*Response[RegisterOrganizationResponse], error) {
	return doRequest[RegisterOrganizationResponse](c, http.MethodPost, "/api/v1/organizations", req, nil)
}

// GetOrganization returns public organization info.
func (c *Client) GetOrganization(id uuid.UUID) (*Response[OrganizationResponse], error) {
	return doRequest[OrganizationResponse](c, http.MethodGet, "/api/v1/organizations/"+id.String(), nil, nil)
}

// GetOrganizationFull returns full organization info with members.
func (c *Client) GetOrganizationFull(id uuid.UUID) (*Response[OrganizationFullResponse], error) {
	return doRequest[OrganizationFullResponse](c, http.MethodGet, "/api/v1/organizations/"+id.String()+"/full", nil, nil)
}

// GetOrganizationRating returns organization rating.
func (c *Client) GetOrganizationRating(id uuid.UUID) (*Response[RatingResponse], error) {
	return doRequest[RatingResponse](c, http.MethodGet, "/api/v1/organizations/"+id.String()+"/rating", nil, nil)
}

// GetOrganizationReviews returns organization reviews.
func (c *Client) GetOrganizationReviews(id uuid.UUID, limit int, cursor string) (*Response[ReviewsListResponse], error) {
	path := fmt.Sprintf("/api/v1/organizations/%s/reviews?limit=%d", id, limit)
	if cursor != "" {
		path += "&cursor=" + cursor
	}
	return doRequest[ReviewsListResponse](c, http.MethodGet, path, nil, nil)
}

// CreateInvitation creates a new invitation.
func (c *Client) CreateInvitation(orgID uuid.UUID, req CreateInvitationRequest) (*Response[InvitationResponse], error) {
	return doRequest[InvitationResponse](c, http.MethodPost, "/api/v1/organizations/"+orgID.String()+"/invitations", req, nil)
}

// GetInvitations returns organization invitations.
func (c *Client) GetInvitations(orgID uuid.UUID) (*Response[[]InvitationResponse], error) {
	return doRequest[[]InvitationResponse](c, http.MethodGet, "/api/v1/organizations/"+orgID.String()+"/invitations", nil, nil)
}

// CancelInvitation cancels an invitation.
func (c *Client) CancelInvitation(orgID, invitationID uuid.UUID) (*Response[struct{}], error) {
	return doRequest[struct{}](c, http.MethodDelete, "/api/v1/organizations/"+orgID.String()+"/invitations/"+invitationID.String(), nil, nil)
}

// GetInvitationByToken returns invitation by token.
func (c *Client) GetInvitationByToken(token string) (*Response[InvitationResponse], error) {
	return doRequest[InvitationResponse](c, http.MethodGet, "/api/v1/invitations/"+token, nil, nil)
}

// AcceptInvitation accepts an invitation.
func (c *Client) AcceptInvitation(token string, req AcceptInvitationRequest) (*Response[AcceptInvitationResponse], error) {
	return doRequest[AcceptInvitationResponse](c, http.MethodPost, "/api/v1/invitations/"+token+"/accept", req, nil)
}

// ChangeMemberRole changes a member's role.
func (c *Client) ChangeMemberRole(orgID, memberID uuid.UUID, role string) (*Response[struct{}], error) {
	return doRequest[struct{}](c, http.MethodPatch, "/api/v1/organizations/"+orgID.String()+"/members/"+memberID.String()+"/role", ChangeRoleRequest{Role: role}, nil)
}

// BlockMember blocks a member.
func (c *Client) BlockMember(orgID, memberID uuid.UUID, reason string) (*Response[struct{}], error) {
	return doRequest[struct{}](c, http.MethodPost, "/api/v1/organizations/"+orgID.String()+"/members/"+memberID.String()+"/block", BlockMemberRequest{Reason: reason}, nil)
}

// UnblockMember unblocks a member.
func (c *Client) UnblockMember(orgID, memberID uuid.UUID) (*Response[struct{}], error) {
	return doRequest[struct{}](c, http.MethodPost, "/api/v1/organizations/"+orgID.String()+"/members/"+memberID.String()+"/unblock", nil, nil)
}

// UpdateMemberInfo updates member profile information.
func (c *Client) UpdateMemberInfo(orgID, memberID uuid.UUID, req UpdateMemberInfoRequest) (*Response[struct{}], error) {
	return doRequest[struct{}](c, http.MethodPatch, "/api/v1/organizations/"+orgID.String()+"/members/"+memberID.String()+"/info", req, nil)
}

// --- Freight Request Methods ---

// CreateFreightRequest creates a new freight request.
func (c *Client) CreateFreightRequest(req CreateFreightRequestRequest) (*Response[struct{ ID uuid.UUID }], error) {
	return doRequest[struct{ ID uuid.UUID }](c, http.MethodPost, "/api/v1/freight-requests", req, nil)
}

// GetFreightRequests returns freight requests with optional filters.
func (c *Client) GetFreightRequests(filters map[string]string) (*Response[FreightRequestListResponse], error) {
	path := "/api/v1/freight-requests"
	if len(filters) > 0 {
		params := make([]string, 0, len(filters))
		for k, v := range filters {
			params = append(params, k+"="+v)
		}
		path += "?" + strings.Join(params, "&")
	}
	return doRequest[FreightRequestListResponse](c, http.MethodGet, path, nil, nil)
}

// GetFreightRequest returns a freight request by ID.
func (c *Client) GetFreightRequest(id uuid.UUID) (*Response[FreightRequestResponse], error) {
	return doRequest[FreightRequestResponse](c, http.MethodGet, "/api/v1/freight-requests/"+id.String(), nil, nil)
}

// UpdateFreightRequest updates a freight request.
func (c *Client) UpdateFreightRequest(id uuid.UUID, req CreateFreightRequestRequest) (*Response[struct{}], error) {
	return doRequest[struct{}](c, http.MethodPatch, "/api/v1/freight-requests/"+id.String(), req, nil)
}

// CancelFreightRequest cancels a freight request.
func (c *Client) CancelFreightRequest(id uuid.UUID, reason *string) (*Response[struct{}], error) {
	return doRequest[struct{}](c, http.MethodDelete, "/api/v1/freight-requests/"+id.String(), CancelRequest{Reason: reason}, nil)
}

// ReassignFreightRequest reassigns the responsible member.
func (c *Client) ReassignFreightRequest(id uuid.UUID, newMemberID uuid.UUID) (*Response[struct{}], error) {
	return doRequest[struct{}](c, http.MethodPost, "/api/v1/freight-requests/"+id.String()+"/reassign", ReassignRequest{NewMemberID: newMemberID}, nil)
}

// --- Offer Methods ---

// CreateOffer creates an offer on a freight request.
func (c *Client) CreateOffer(frID uuid.UUID, req CreateOfferRequest) (*Response[CreateOfferResponse], error) {
	return doRequest[CreateOfferResponse](c, http.MethodPost, "/api/v1/freight-requests/"+frID.String()+"/offers", req, nil)
}

// GetOffers returns offers on a freight request.
func (c *Client) GetOffers(frID uuid.UUID) (*Response[[]OfferResponse], error) {
	return doRequest[[]OfferResponse](c, http.MethodGet, "/api/v1/freight-requests/"+frID.String()+"/offers", nil, nil)
}

// WithdrawOffer withdraws an offer.
func (c *Client) WithdrawOffer(frID, offerID uuid.UUID, reason *string) (*Response[struct{}], error) {
	return doRequest[struct{}](c, http.MethodDelete, "/api/v1/freight-requests/"+frID.String()+"/offers/"+offerID.String(), CancelRequest{Reason: reason}, nil)
}

// SelectOffer selects an offer.
func (c *Client) SelectOffer(frID, offerID uuid.UUID) (*Response[struct{}], error) {
	return doRequest[struct{}](c, http.MethodPost, "/api/v1/freight-requests/"+frID.String()+"/offers/"+offerID.String()+"/select", nil, nil)
}

// RejectOffer rejects an offer.
func (c *Client) RejectOffer(frID, offerID uuid.UUID, reason *string) (*Response[struct{}], error) {
	return doRequest[struct{}](c, http.MethodPost, "/api/v1/freight-requests/"+frID.String()+"/offers/"+offerID.String()+"/reject", CancelRequest{Reason: reason}, nil)
}

// ConfirmOffer confirms an offer (creates order).
func (c *Client) ConfirmOffer(frID, offerID uuid.UUID) (*Response[struct{}], error) {
	return doRequest[struct{}](c, http.MethodPost, "/api/v1/freight-requests/"+frID.String()+"/offers/"+offerID.String()+"/confirm", nil, nil)
}

// DeclineOffer declines an offer after selection.
func (c *Client) DeclineOffer(frID, offerID uuid.UUID, reason *string) (*Response[struct{}], error) {
	return doRequest[struct{}](c, http.MethodPost, "/api/v1/freight-requests/"+frID.String()+"/offers/"+offerID.String()+"/decline", CancelRequest{Reason: reason}, nil)
}

// UnselectOffer cancels selection of an offer (returns it to pending).
func (c *Client) UnselectOffer(frID, offerID uuid.UUID, reason *string) (*Response[struct{}], error) {
	return doRequest[struct{}](c, http.MethodPost, "/api/v1/freight-requests/"+frID.String()+"/offers/"+offerID.String()+"/unselect", CancelRequest{Reason: reason}, nil)
}

// GetMyOffers returns offers made by current organization.
func (c *Client) GetMyOffers(status string, limit, offset int) (*Response[[]OfferResponse], error) {
	path := fmt.Sprintf("/api/v1/offers?limit=%d&offset=%d", limit, offset)
	if status != "" {
		path += "&status=" + status
	}
	return doRequest[[]OfferResponse](c, http.MethodGet, path, nil, nil)
}

// --- Freight Request Completion & Review Methods ---

// CompleteFreightRequest marks the freight request as completed by current organization.
func (c *Client) CompleteFreightRequest(frID uuid.UUID) (*Response[struct{}], error) {
	return doRequest[struct{}](c, http.MethodPost, "/api/v1/freight-requests/"+frID.String()+"/complete", nil, nil)
}

// LeaveFreightRequestReview leaves a review on a completed freight request.
func (c *Client) LeaveFreightRequestReview(frID uuid.UUID, rating int, comment *string) (*Response[LeaveReviewResponse], error) {
	return doRequest[LeaveReviewResponse](c, http.MethodPost, "/api/v1/freight-requests/"+frID.String()+"/review", LeaveReviewRequest{Rating: rating, Comment: comment}, nil)
}

// EditFreightRequestReview edits an existing review (within 24h window).
func (c *Client) EditFreightRequestReview(frID uuid.UUID, rating int, comment *string) (*Response[struct{}], error) {
	return doRequest[struct{}](c, http.MethodPatch, "/api/v1/freight-requests/"+frID.String()+"/review", LeaveReviewRequest{Rating: rating, Comment: comment}, nil)
}

// CancelFreightRequestAfterConfirmed cancels a confirmed freight request.
func (c *Client) CancelFreightRequestAfterConfirmed(frID uuid.UUID, reason *string) (*Response[struct{}], error) {
	return doRequest[struct{}](c, http.MethodPost, "/api/v1/freight-requests/"+frID.String()+"/cancel-confirmed", CancelRequest{Reason: reason}, nil)
}

// ReassignCarrierMember reassigns the responsible carrier member.
func (c *Client) ReassignCarrierMember(frID uuid.UUID, newMemberID uuid.UUID) (*Response[struct{}], error) {
	return doRequest[struct{}](c, http.MethodPost, "/api/v1/freight-requests/"+frID.String()+"/reassign-carrier", ReassignRequest{NewMemberID: newMemberID}, nil)
}

// --- Geo Methods ---

// GetCountries returns all countries.
func (c *Client) GetCountries() (*Response[[]CountryResponse], error) {
	return doRequest[[]CountryResponse](c, http.MethodGet, "/api/v1/geo/countries", nil, nil)
}

// GetCountry returns a country by ID.
func (c *Client) GetCountry(id int) (*Response[CountryResponse], error) {
	return doRequest[CountryResponse](c, http.MethodGet, fmt.Sprintf("/api/v1/geo/countries/%d", id), nil, nil)
}

// GetCountryRaw returns a country by raw string ID (for testing invalid IDs).
func (c *Client) GetCountryRaw(id string) (int, []byte, error) {
	return c.Raw(http.MethodGet, "/api/v1/geo/countries/"+id, nil, nil)
}

// GetCountryCities returns cities for a country with optional search.
func (c *Client) GetCountryCities(countryID int, search string, limit int) (*Response[[]CityResponse], error) {
	path := fmt.Sprintf("/api/v1/geo/countries/%d/cities", countryID)
	params := []string{}
	if search != "" {
		params = append(params, "search="+search)
	}
	if limit > 0 {
		params = append(params, fmt.Sprintf("limit=%d", limit))
	}
	if len(params) > 0 {
		path += "?"
		for i, p := range params {
			if i > 0 {
				path += "&"
			}
			path += p
		}
	}
	return doRequest[[]CityResponse](c, http.MethodGet, path, nil, nil)
}

// GetCity returns a city by ID.
func (c *Client) GetCity(id int) (*Response[CityResponse], error) {
	return doRequest[CityResponse](c, http.MethodGet, fmt.Sprintf("/api/v1/geo/cities/%d", id), nil, nil)
}

// GetCityRaw returns a city by raw string ID (for testing invalid IDs).
func (c *Client) GetCityRaw(id string) (int, []byte, error) {
	return c.Raw(http.MethodGet, "/api/v1/geo/cities/"+id, nil, nil)
}

// GetCities returns cities with optional filters (deprecated, use GetCountryCities).
func (c *Client) GetCities(countryID *int, search string) (*Response[[]CityResponse], error) {
	if countryID != nil {
		return c.GetCountryCities(*countryID, search, 0)
	}
	return doRequest[[]CityResponse](c, http.MethodGet, "/api/v1/geo/cities", nil, nil)
}

// --- History Methods ---

// HistoryEntry represents a single history event.
type HistoryEntry struct {
	EventType   string `json:"event_type"`
	Description string `json:"description"`
	Timestamp   string `json:"timestamp"`
}

// HistoryPage represents a paginated history response.
type HistoryPage struct {
	Items  []HistoryEntry `json:"items"`
	Total  int            `json:"total"`
	Limit  int            `json:"limit"`
	Offset int            `json:"offset"`
}

// GetOrganizationHistory returns history for an organization.
func (c *Client) GetOrganizationHistory(id uuid.UUID, limit, offset int) (*Response[HistoryPage], error) {
	path := fmt.Sprintf("/api/v1/organizations/%s/history?limit=%d&offset=%d", id, limit, offset)
	return doRequest[HistoryPage](c, http.MethodGet, path, nil, nil)
}

// GetFreightRequestHistory returns history for a freight request.
func (c *Client) GetFreightRequestHistory(id uuid.UUID, limit, offset int) (*Response[HistoryPage], error) {
	path := fmt.Sprintf("/api/v1/freight-requests/%s/history?limit=%d&offset=%d", id, limit, offset)
	return doRequest[HistoryPage](c, http.MethodGet, path, nil, nil)
}

// --- Admin Methods ---

// AdminLogin authenticates an admin.
func (c *Client) AdminLogin(email, password string) (*Response[AdminLoginResponse], error) {
	return doRequest[AdminLoginResponse](c, http.MethodPost, "/api/v1/admin/auth/login", AdminLoginRequest{
		Email:    email,
		Password: password,
	}, nil)
}

// AdminLogout logs out admin.
func (c *Client) AdminLogout() (*Response[struct{}], error) {
	return doRequest[struct{}](c, http.MethodPost, "/api/v1/admin/auth/logout", nil, nil)
}

// AdminMe returns current admin info.
func (c *Client) AdminMe() (*Response[AdminMeResponse], error) {
	return doRequest[AdminMeResponse](c, http.MethodGet, "/api/v1/admin/auth/me", nil, nil)
}

// AdminGetOrganizations returns organizations with optional status filter.
func (c *Client) AdminGetOrganizations(status string) (*Response[[]OrganizationResponse], error) {
	path := "/api/v1/admin/organizations"
	if status != "" {
		path += "?status=" + status
	}
	return doRequest[[]OrganizationResponse](c, http.MethodGet, path, nil, nil)
}

// AdminGetPendingOrganizations returns pending organizations.
func (c *Client) AdminGetPendingOrganizations() (*Response[[]OrganizationResponse], error) {
	return c.AdminGetOrganizations("pending")
}

// AdminGetOrganization returns organization details.
func (c *Client) AdminGetOrganization(id uuid.UUID) (*Response[OrganizationFullResponse], error) {
	return doRequest[OrganizationFullResponse](c, http.MethodGet, "/api/v1/admin/organizations/"+id.String(), nil, nil)
}

// AdminApproveOrganization approves an organization.
func (c *Client) AdminApproveOrganization(id uuid.UUID, comment *string) (*Response[struct{}], error) {
	return doRequest[struct{}](c, http.MethodPost, "/api/v1/admin/organizations/"+id.String()+"/approve", ApproveOrganizationRequest{Comment: comment}, nil)
}

// AdminRejectOrganization rejects an organization.
func (c *Client) AdminRejectOrganization(id uuid.UUID, reason string) (*Response[struct{}], error) {
	return doRequest[struct{}](c, http.MethodPost, "/api/v1/admin/organizations/"+id.String()+"/reject", RejectOrganizationRequest{Reason: reason}, nil)
}

// AdminMarkFraudster marks organization as fraudster.
func (c *Client) AdminMarkFraudster(id uuid.UUID, isConfirmed bool, reason string) (*Response[struct{}], error) {
	return doRequest[struct{}](c, http.MethodPost, "/api/v1/admin/organizations/"+id.String()+"/mark-fraudster", struct {
		IsConfirmed bool   `json:"is_confirmed"`
		Reason      string `json:"reason"`
	}{IsConfirmed: isConfirmed, Reason: reason}, nil)
}

// AdminUnmarkFraudster removes fraudster mark from organization.
func (c *Client) AdminUnmarkFraudster(id uuid.UUID, reason string) (*Response[struct{}], error) {
	return doRequest[struct{}](c, http.MethodPost, "/api/v1/admin/organizations/"+id.String()+"/unmark-fraudster", struct {
		Reason string `json:"reason"`
	}{Reason: reason}, nil)
}

// AdminListFraudsters returns all organizations marked as fraudsters.
func (c *Client) AdminListFraudsters() (*Response[FraudstersResponse], error) {
	return doRequest[FraudstersResponse](c, http.MethodGet, "/api/v1/admin/fraudsters", nil, nil)
}

// AdminReviewResponse represents a review in admin context.
type AdminReviewResponse struct {
	ID             uuid.UUID `json:"id"`
	OrderID        uuid.UUID `json:"order_id"`
	ReviewerOrgID  uuid.UUID `json:"reviewer_org_id"`
	ReviewerOrgName string   `json:"reviewer_org_name"`
	TargetOrgID    uuid.UUID `json:"target_org_id"`
	TargetOrgName  string    `json:"target_org_name"`
	Rating         int       `json:"rating"`
	Comment        string    `json:"comment"`
	Status         string    `json:"status"`
	CreatedAt      string    `json:"created_at"`
}

// AdminGetReviews returns reviews for moderation.
func (c *Client) AdminGetReviews(status string) (*Response[[]AdminReviewResponse], error) {
	path := "/api/v1/admin/reviews"
	if status != "" {
		path += "?status=" + status
	}
	return doRequest[[]AdminReviewResponse](c, http.MethodGet, path, nil, nil)
}

// AdminGetReview returns a single review.
func (c *Client) AdminGetReview(id uuid.UUID) (*Response[AdminReviewResponse], error) {
	return doRequest[AdminReviewResponse](c, http.MethodGet, "/api/v1/admin/reviews/"+id.String(), nil, nil)
}

// AdminApproveReview approves a review.
func (c *Client) AdminApproveReview(id uuid.UUID) (*Response[struct{}], error) {
	return doRequest[struct{}](c, http.MethodPost, "/api/v1/admin/reviews/"+id.String()+"/approve", nil, nil)
}

// AdminRejectReview rejects a review.
func (c *Client) AdminRejectReview(id uuid.UUID, reason string) (*Response[struct{}], error) {
	return doRequest[struct{}](c, http.MethodPost, "/api/v1/admin/reviews/"+id.String()+"/reject", struct{ Reason string }{Reason: reason}, nil)
}

// --- Notification Methods ---

// GetNotificationPreferences returns notification preferences.
func (c *Client) GetNotificationPreferences() (*Response[NotificationPreferencesResponse], error) {
	return doRequest[NotificationPreferencesResponse](c, http.MethodGet, "/api/v1/notifications/preferences", nil, nil)
}

// UpdateNotificationPreferences updates notification preferences.
func (c *Client) UpdateNotificationPreferences(req UpdatePreferencesRequest) (*Response[struct{}], error) {
	return doRequest[struct{}](c, http.MethodPatch, "/api/v1/notifications/preferences", req, nil)
}

// GetInAppNotifications returns in-app notifications.
func (c *Client) GetInAppNotifications(unreadOnly bool, limit, offset int) (*Response[PaginatedResponse[InAppNotificationResponse]], error) {
	path := fmt.Sprintf("/api/v1/notifications?limit=%d&offset=%d", limit, offset)
	if unreadOnly {
		path += "&unread_only=true"
	}
	return doRequest[PaginatedResponse[InAppNotificationResponse]](c, http.MethodGet, path, nil, nil)
}

// MarkNotificationRead marks a notification as read.
func (c *Client) MarkNotificationRead(id uuid.UUID) (*Response[struct{}], error) {
	return doRequest[struct{}](c, http.MethodPost, "/api/v1/notifications/"+id.String()+"/read", nil, nil)
}

// MarkAllNotificationsRead marks all notifications as read.
func (c *Client) MarkAllNotificationsRead() (*Response[struct{}], error) {
	return doRequest[struct{}](c, http.MethodPost, "/api/v1/notifications/read-all", nil, nil)
}

// UnreadCountResponse represents unread count response.
type UnreadCountResponse struct {
	Unread int `json:"unread"`
}

// GetUnreadCount returns count of unread notifications.
func (c *Client) GetUnreadCount() (*Response[UnreadCountResponse], error) {
	return doRequest[UnreadCountResponse](c, http.MethodGet, "/api/v1/notifications/unread-count", nil, nil)
}

// MarkNotificationsRead marks specific notifications as read.
func (c *Client) MarkNotificationsRead(ids []uuid.UUID) (*Response[struct{}], error) {
	return doRequest[struct{}](c, http.MethodPost, "/api/v1/notifications/read", struct {
		IDs []uuid.UUID `json:"ids"`
	}{IDs: ids}, nil)
}

// LinkCodeResponse represents telegram link code response.
type LinkCodeResponse struct {
	Code        string `json:"code"`
	BotUsername string `json:"bot_username,omitempty"`
}

// GetTelegramLinkCode generates a code for linking Telegram.
func (c *Client) GetTelegramLinkCode() (*Response[LinkCodeResponse], error) {
	return doRequest[LinkCodeResponse](c, http.MethodPost, "/api/v1/notifications/telegram/link-code", nil, nil)
}

// DisconnectTelegram disconnects Telegram from account.
func (c *Client) DisconnectTelegram() (*Response[struct{}], error) {
	return doRequest[struct{}](c, http.MethodDelete, "/api/v1/notifications/telegram", nil, nil)
}

// SetEmail sets email for notifications.
func (c *Client) SetEmail(email string) (*Response[struct{}], error) {
	return doRequest[struct{}](c, http.MethodPost, "/api/v1/notifications/email", struct {
		Email string `json:"email"`
	}{Email: email}, nil)
}

// DisconnectEmail removes email notification.
func (c *Client) DisconnectEmail() (*Response[struct{}], error) {
	return doRequest[struct{}](c, http.MethodDelete, "/api/v1/notifications/email", nil, nil)
}

// SetMarketingConsent updates marketing email consent.
func (c *Client) SetMarketingConsent(consent bool) (*Response[struct{}], error) {
	return doRequest[struct{}](c, http.MethodPatch, "/api/v1/notifications/email/marketing", struct {
		Consent bool `json:"consent"`
	}{Consent: consent}, nil)
}

// ResendEmailVerification requests email verification resend.
func (c *Client) ResendEmailVerification() (*Response[struct{}], error) {
	return doRequest[struct{}](c, http.MethodPost, "/api/v1/notifications/email/resend-verification", nil, nil)
}

// VerifyEmailByToken verifies email with token.
func (c *Client) VerifyEmailByToken(token string) (*Response[struct{}], error) {
	return doRequest[struct{}](c, http.MethodPost, "/api/v1/notifications/email/verify", struct {
		Token string `json:"token"`
	}{Token: token}, nil)
}

// --- Support Methods ---

// CreateTicket creates a support ticket.
func (c *Client) CreateTicket(subject, message string) (*Response[TicketDetailResponse], error) {
	return doRequest[TicketDetailResponse](c, http.MethodPost, "/api/v1/support/tickets", CreateTicketRequest{
		Subject: subject,
		Message: message,
	}, nil)
}

// GetMyTickets returns current user's tickets.
func (c *Client) GetMyTickets() (*Response[[]TicketResponse], error) {
	return doRequest[[]TicketResponse](c, http.MethodGet, "/api/v1/support/tickets", nil, nil)
}

// GetTicket returns a ticket by ID.
func (c *Client) GetTicket(id uuid.UUID) (*Response[TicketDetailResponse], error) {
	return doRequest[TicketDetailResponse](c, http.MethodGet, "/api/v1/support/tickets/"+id.String(), nil, nil)
}

// AddTicketMessage adds a message to a ticket.
func (c *Client) AddTicketMessage(ticketID uuid.UUID, content string) (*Response[struct{}], error) {
	return doRequest[struct{}](c, http.MethodPost, "/api/v1/support/tickets/"+ticketID.String()+"/messages", AddMessageRequest{Content: content}, nil)
}

// CloseTicket closes a ticket.
func (c *Client) CloseTicket(ticketID uuid.UUID) (*Response[struct{}], error) {
	return doRequest[struct{}](c, http.MethodPost, "/api/v1/support/tickets/"+ticketID.String()+"/close", nil, nil)
}

// ReopenTicket reopens a closed ticket.
func (c *Client) ReopenTicket(ticketID uuid.UUID) (*Response[struct{}], error) {
	return doRequest[struct{}](c, http.MethodPost, "/api/v1/support/tickets/"+ticketID.String()+"/reopen", nil, nil)
}

// GetFAQ returns FAQ items.
func (c *Client) GetFAQ() (*Response[[]FAQResponse], error) {
	return doRequest[[]FAQResponse](c, http.MethodGet, "/api/v1/support/faq", nil, nil)
}

// GetMyTicketsFiltered returns current user's tickets with optional status filter.
func (c *Client) GetMyTicketsFiltered(status string) (*Response[[]TicketResponse], error) {
	path := "/api/v1/support/tickets"
	if status != "" {
		path += "?status=" + status
	}
	return doRequest[[]TicketResponse](c, http.MethodGet, path, nil, nil)
}

// --- Subscriptions Methods ---

// GetSubscriptions returns current user's subscriptions.
func (c *Client) GetSubscriptions() (*Response[[]SubscriptionResponse], error) {
	return doRequest[[]SubscriptionResponse](c, http.MethodGet, "/api/v1/subscriptions", nil, nil)
}

// CreateSubscription creates a new subscription.
func (c *Client) CreateSubscription(req CreateSubscriptionRequest) (*Response[SubscriptionResponse], error) {
	return doRequest[SubscriptionResponse](c, http.MethodPost, "/api/v1/subscriptions", req, nil)
}

// GetSubscription returns a subscription by ID.
func (c *Client) GetSubscription(id uuid.UUID) (*Response[SubscriptionResponse], error) {
	return doRequest[SubscriptionResponse](c, http.MethodGet, "/api/v1/subscriptions/"+id.String(), nil, nil)
}

// UpdateSubscription updates a subscription.
func (c *Client) UpdateSubscription(id uuid.UUID, req CreateSubscriptionRequest) (*Response[SubscriptionResponse], error) {
	return doRequest[SubscriptionResponse](c, http.MethodPut, "/api/v1/subscriptions/"+id.String(), req, nil)
}

// DeleteSubscription deletes a subscription.
func (c *Client) DeleteSubscription(id uuid.UUID) (*Response[struct{}], error) {
	return doRequest[struct{}](c, http.MethodDelete, "/api/v1/subscriptions/"+id.String(), nil, nil)
}

// SetSubscriptionActive sets subscription active status.
func (c *Client) SetSubscriptionActive(id uuid.UUID, isActive bool) (*Response[struct{}], error) {
	return doRequest[struct{}](c, http.MethodPatch, "/api/v1/subscriptions/"+id.String()+"/active", SetActiveRequest{IsActive: isActive}, nil)
}

// --- Admin Support Methods ---

// AdminListTickets returns all support tickets for admin.
func (c *Client) AdminListTickets(status string, limit, offset int) (*Response[AdminTicketsListResponse], error) {
	path := fmt.Sprintf("/api/v1/admin/support/tickets?limit=%d&offset=%d", limit, offset)
	if status != "" {
		path += "&status=" + status
	}
	return doRequest[AdminTicketsListResponse](c, http.MethodGet, path, nil, nil)
}

// AdminGetTicket returns a ticket with messages for admin.
func (c *Client) AdminGetTicket(id uuid.UUID) (*Response[AdminTicketDetailResponse], error) {
	return doRequest[AdminTicketDetailResponse](c, http.MethodGet, "/api/v1/admin/support/tickets/"+id.String(), nil, nil)
}

// AdminAddTicketMessage adds an admin message to a ticket.
func (c *Client) AdminAddTicketMessage(ticketID uuid.UUID, content string) (*Response[struct{}], error) {
	return doRequest[struct{}](c, http.MethodPost, "/api/v1/admin/support/tickets/"+ticketID.String()+"/messages", AdminAddMessageRequest{Content: content}, nil)
}

// AdminCloseTicket closes a ticket.
func (c *Client) AdminCloseTicket(ticketID uuid.UUID) (*Response[struct{}], error) {
	return doRequest[struct{}](c, http.MethodPost, "/api/v1/admin/support/tickets/"+ticketID.String()+"/close", nil, nil)
}

// Raw performs a raw HTTP request (for edge cases).
func (c *Client) Raw(method, path string, body any, headers map[string]string) (int, []byte, error) {
	resp, respBody, err := c.do(method, path, body, headers)
	if err != nil {
		return 0, nil, err
	}
	return resp.StatusCode, respBody, nil
}
