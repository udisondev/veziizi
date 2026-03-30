// Package client provides a typed HTTP client for E2E testing.
package client

import (
	"time"

	"github.com/udisondev/veziizi/backend/internal/interfaces/http/handlers"
	"github.com/google/uuid"
)

// Re-export types from handlers to avoid duplication
type (
	// Support response types
	TicketDetailResponse      = handlers.TicketDetailResponse
	TicketMessageResponse     = handlers.TicketMessageResponse
	AdminTicketDetailResponse = handlers.AdminTicketDetailResponse

	// Support request types
	AddMessageRequest      = handlers.AddMessageRequest
	AdminAddMessageRequest = handlers.AdminAddMessageRequest

	// Organization request types
	BlockMemberRequest = handlers.BlockMemberRequest

	// Admin fraud types
	FraudsterResponse  = handlers.FraudsterResponse
	FraudstersResponse = handlers.FraudstersResponse
)

// Response wraps HTTP response data with status code.
type Response[T any] struct {
	StatusCode int
	Body       T
	RawBody    []byte
	Cookies    map[string]string
}

// ErrorResponse represents an API error response.
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}

// --- Auth Types ---

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// ForgotPasswordRequest is the request for initiating password reset
type ForgotPasswordRequest struct {
	Email string `json:"email"`
}

// ResetPasswordRequest is the request for resetting password with token
type ResetPasswordRequest struct {
	Token       string `json:"token"`
	NewPassword string `json:"new_password"`
}

type LoginResponse struct {
	MemberID       uuid.UUID `json:"member_id"`
	OrganizationID uuid.UUID `json:"organization_id"`
	Email          string    `json:"email"`
	Name           string    `json:"name"`
	Role           string    `json:"role"`
}

type MeResponse struct {
	MemberID       uuid.UUID           `json:"member_id"`
	OrganizationID uuid.UUID           `json:"organization_id"`
	Role           string              `json:"role"`
	Email          string              `json:"email"`
	Name           string              `json:"name"`
	Phone          *string             `json:"phone,omitempty"`
	TelegramID     *int64              `json:"telegram_id,omitempty"`
	Status         string              `json:"status"`
	Organization   OrganizationSummary `json:"organization"`
}

type OrganizationSummary struct {
	ID      uuid.UUID `json:"id"`
	Name    string    `json:"name"`
	Status  string    `json:"status"`
	Country string    `json:"country"`
}

type MemberPublicProfile struct {
	ID             uuid.UUID `json:"id"`
	OrganizationID uuid.UUID `json:"organization_id"`
	Name           string    `json:"name"`
	Email          string    `json:"email"`
	Phone          *string   `json:"phone,omitempty"`
	Role           string    `json:"role"`
	Status         string    `json:"status"`
}

// MemberProfileResponse is returned by GET /api/v1/members/{id}
type MemberProfileResponse struct {
	ID               string    `json:"id"`
	Name             string    `json:"name"`
	Email            string    `json:"email"`
	Phone            *string   `json:"phone,omitempty"`
	Role             string    `json:"role"`
	Status           string    `json:"status"`
	OrganizationID   string    `json:"organization_id"`
	OrganizationName string    `json:"organization_name"`
	CreatedAt        time.Time `json:"created_at"`
}

// --- Organization Types ---

type RegisterOrganizationRequest struct {
	Name          string `json:"name"`
	Email         string `json:"email"`
	Phone         string `json:"phone"`
	INN           string `json:"inn"`
	Country       string `json:"country"`
	Address       string `json:"address"`
	OwnerName     string `json:"owner_name"`
	OwnerEmail    string `json:"owner_email"`
	OwnerPhone    string `json:"owner_phone"`
	OwnerPassword string `json:"owner_password"`
}

type RegisterOrganizationResponse struct {
	OrganizationID uuid.UUID `json:"organization_id"`
	MemberID       uuid.UUID `json:"member_id"`
}

type OrganizationResponse struct {
	ID       uuid.UUID `json:"id"`
	Name     string    `json:"name"`
	Email    string    `json:"email"`
	Phone    string    `json:"phone"`
	INN      string    `json:"inn"`
	Country  string    `json:"country"`
	Address  string    `json:"address"`
	Status   string    `json:"status"`
	IsFraud  bool      `json:"is_fraud"`
	Version  int       `json:"version"`
}

// OrganizationMemberDetail is a member in organization full response
type OrganizationMemberDetail struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	Name      string `json:"name"`
	Phone     string `json:"phone"`
	Role      string `json:"role"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at"`
}

type OrganizationFullResponse struct {
	ID        string                     `json:"id"`
	Name      string                     `json:"name"`
	INN       string                     `json:"inn"`
	LegalName string                     `json:"legal_name"`
	Country   string                     `json:"country"`
	Phone     string                     `json:"phone"`
	Email     string                     `json:"email"`
	Address   string                     `json:"address"`
	Status    string                     `json:"status"`
	Members   []OrganizationMemberDetail `json:"members"`
	CreatedAt string                     `json:"created_at"`
}

type RatingResponse struct {
	OrganizationID  uuid.UUID `json:"organization_id"`
	TotalReviews    int       `json:"total_reviews"`
	PendingReviews  int       `json:"pending_reviews"`
	AverageRating   float64   `json:"average_rating"`
	WeightedAverage float64   `json:"weighted_average"`
}

type CreateInvitationRequest struct {
	Email string  `json:"email"`
	Role  string  `json:"role"`
	Name  *string `json:"name,omitempty"`
	Phone *string `json:"phone,omitempty"`
}

type InvitationResponse struct {
	ID             uuid.UUID  `json:"id"`
	OrganizationID uuid.UUID  `json:"organization_id"`
	Email          string     `json:"email"`
	Role           string     `json:"role"`
	Status         string     `json:"status"`
	Token          string     `json:"token,omitempty"`
	Name           *string    `json:"name,omitempty"`
	Phone          *string    `json:"phone,omitempty"`
	ExpiresAt      time.Time  `json:"expires_at"`
}

type AcceptInvitationRequest struct {
	Password string  `json:"password"`
	Name     *string `json:"name,omitempty"`
	Phone    *string `json:"phone,omitempty"`
}

type AcceptInvitationResponse struct {
	OrganizationID uuid.UUID `json:"organization_id"`
	MemberID       uuid.UUID `json:"member_id"`
}

type ChangeRoleRequest struct {
	Role string `json:"role"`
}

// --- Freight Request Types ---

type RoutePoint struct {
	IsLoading    bool    `json:"is_loading"`
	IsUnloading  bool    `json:"is_unloading"`
	CountryID    *int    `json:"country_id,omitempty"`
	CityID       *int    `json:"city_id,omitempty"`
	Address      string  `json:"address"`
	DateFrom     string  `json:"date_from"`
	DateTo       *string `json:"date_to,omitempty"`
	TimeFrom     *string `json:"time_from,omitempty"`
	TimeTo       *string `json:"time_to,omitempty"`
	ContactName  *string `json:"contact_name,omitempty"`
	ContactPhone *string `json:"contact_phone,omitempty"`
	Comment      *string `json:"comment,omitempty"`
}

type Route struct {
	Points []RoutePoint `json:"points"`
}

type Cargo struct {
	Description string  `json:"description"`
	Weight      float64 `json:"weight"`
	Volume      float64 `json:"volume,omitempty"`
	ADRClass    string  `json:"adr_class,omitempty"`
	Quantity    int     `json:"quantity"`
}

type VehicleRequirements struct {
	VehicleType    string   `json:"vehicle_type"`
	VehicleSubtype string   `json:"vehicle_subtype"`
	LoadingTypes   []string `json:"loading_types,omitempty"`
}

type Money struct {
	Amount   int64  `json:"amount"`
	Currency string `json:"currency"`
}

type Payment struct {
	Price        *Money `json:"price,omitempty"`
	VatType      string `json:"vat_type"`
	Method       string `json:"method"`
	Terms        string `json:"terms"`
	DeferredDays int    `json:"deferred_days,omitempty"`
}

type CreateFreightRequestRequest struct {
	Route               Route               `json:"route"`
	Cargo               Cargo               `json:"cargo"`
	VehicleRequirements VehicleRequirements `json:"vehicle_requirements"`
	Payment             Payment             `json:"payment"`
	Comment             *string             `json:"comment,omitempty"`
	ExpiresAt           *time.Time          `json:"expires_at,omitempty"`
}

type FreightRequestResponse struct {
	ID                  uuid.UUID           `json:"id"`
	RequestNumber       string              `json:"request_number"`
	CustomerOrgID       uuid.UUID           `json:"customer_org_id"`
	CustomerOrgName     string              `json:"customer_org_name,omitempty"`
	MemberID            uuid.UUID           `json:"member_id"`
	MemberName          string              `json:"member_name,omitempty"`
	Route               Route               `json:"route"`
	Cargo               Cargo               `json:"cargo"`
	VehicleRequirements VehicleRequirements `json:"vehicle_requirements"`
	Payment             Payment             `json:"payment"`
	Comment             *string             `json:"comment,omitempty"`
	Status              string              `json:"status"`
	SelectedOfferID     *uuid.UUID          `json:"selected_offer_id,omitempty"`
	OrderID             *uuid.UUID          `json:"order_id,omitempty"`
	Version             int                 `json:"version"`
	FreightVersion      int                 `json:"freight_version"`
	CreatedAt           time.Time           `json:"created_at"`
	ExpiresAt           time.Time           `json:"expires_at"`
}

// FreightRequestListItem is a simplified version for list responses
type FreightRequestListItem struct {
	ID                 uuid.UUID  `json:"id"`
	RequestNumber      int64      `json:"request_number"`
	CustomerOrgID      uuid.UUID  `json:"customer_org_id"`
	Status             string     `json:"status"`
	ExpiresAt          time.Time  `json:"expires_at"`
	CreatedAt          time.Time  `json:"created_at"`
	OriginAddress      *string    `json:"origin_address,omitempty"`
	DestinationAddress *string    `json:"destination_address,omitempty"`
	CargoWeight        *float64   `json:"cargo_weight,omitempty"`
	PriceAmount        *int64     `json:"price_amount,omitempty"`
	PriceCurrency      *string    `json:"price_currency,omitempty"`
	VehicleType        *string    `json:"vehicle_type,omitempty"`
	VehicleSubType     *string    `json:"vehicle_subtype,omitempty"`
	CustomerOrgName    *string    `json:"customer_org_name,omitempty"`
	CustomerMemberID   *uuid.UUID `json:"customer_member_id,omitempty"`
}

// FreightRequestListResponse is the response for list freight requests
type FreightRequestListResponse struct {
	Items      []FreightRequestListItem `json:"items"`
	NextCursor *string                  `json:"next_cursor,omitempty"`
	HasMore    bool                     `json:"has_more"`
}

type CreateOfferRequest struct {
	Price         Money   `json:"price"`
	Comment       string  `json:"comment,omitempty"`
	VATType       string  `json:"vat_type"`
	PaymentMethod string  `json:"payment_method"`
}

// CreateOfferResponse is the response from POST /api/v1/freight-requests/{id}/offers
type CreateOfferResponse struct {
	OfferID uuid.UUID `json:"offer_id"`
}

type OfferResponse struct {
	ID                uuid.UUID  `json:"id"`
	FreightRequestID  uuid.UUID  `json:"freight_request_id"`
	CarrierOrgID      uuid.UUID  `json:"carrier_org_id"`
	CarrierOrgName    string     `json:"carrier_org_name,omitempty"`
	CarrierMemberID   *uuid.UUID `json:"carrier_member_id,omitempty"`
	CarrierMemberName *string    `json:"carrier_member_name,omitempty"`
	Price             float64    `json:"price"`
	Currency          string     `json:"currency"`
	VATType           string     `json:"vat_type"`
	PaymentMethod     string     `json:"payment_method"`
	PaymentTerms      string     `json:"payment_terms"`
	PrepayPercent     *int       `json:"prepay_percent,omitempty"`
	Comment           *string    `json:"comment,omitempty"`
	Status            string     `json:"status"`
	CreatedAt         time.Time  `json:"created_at"`
}

type ReassignRequest struct {
	NewMemberID uuid.UUID `json:"new_member_id"`
}

type CancelRequest struct {
	Reason *string `json:"reason,omitempty"`
}

// --- Freight Request Review Types ---

type LeaveReviewRequest struct {
	Rating  int     `json:"rating"`
	Comment *string `json:"comment,omitempty"`
}

type LeaveReviewResponse struct {
	ReviewID uuid.UUID `json:"review_id"`
}

// --- Organization Review Types ---

type OrgReviewResponse struct {
	ID            string  `json:"id"`
	OrderID       string  `json:"order_id"`
	ReviewerOrgID string  `json:"reviewer_org_id"`
	Rating        int     `json:"rating"`
	Comment       string  `json:"comment"`
	Weight        float64 `json:"weight"`
	CreatedAt     string  `json:"created_at"`
}

type ReviewsListResponse struct {
	Items      []OrgReviewResponse `json:"items"`
	NextCursor string              `json:"next_cursor,omitempty"`
	HasMore    bool                `json:"has_more"`
}

// --- Geo Types ---

type CountryResponse struct {
	ID      int    `json:"id"`
	Name    string `json:"name"`
	NameRU  string `json:"name_ru"`
	ISOCode string `json:"iso2"` // API returns iso2
}

type CityResponse struct {
	ID        int     `json:"id"`
	CountryID int     `json:"country_id"`
	Name      string  `json:"name"`
	NameRU    string  `json:"name_ru"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

// --- Admin Types ---

type AdminLoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type AdminLoginResponse struct {
	ID    uuid.UUID `json:"id"`
	Email string    `json:"email"`
	Name  string    `json:"name"`
}

type PaginatedResponse[T any] struct {
	Items  []T `json:"items"`
	Total  int `json:"total"`
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}

type ApproveOrganizationRequest struct {
	Comment *string `json:"comment,omitempty"`
}

type RejectOrganizationRequest struct {
	Reason string `json:"reason"`
}

// --- Notification Types ---

// CategorySettings настройки для одной категории
type CategorySettings struct {
	InApp    bool `json:"in_app"`
	Telegram bool `json:"telegram"`
}

// EnabledCategories настройки всех категорий
type EnabledCategories map[string]CategorySettings

// TelegramStatusResponse статус Telegram
type TelegramStatusResponse struct {
	Connected   bool    `json:"connected"`
	Username    *string `json:"username,omitempty"`
	ConnectedAt *string `json:"connected_at,omitempty"`
}

type NotificationPreferencesResponse struct {
	MemberID          uuid.UUID              `json:"member_id"`
	EnabledCategories EnabledCategories      `json:"enabled_categories"`
	Telegram          TelegramStatusResponse `json:"telegram"`
}

type UpdatePreferencesRequest struct {
	Categories EnabledCategories `json:"categories"`
}

type InAppNotificationResponse struct {
	ID        uuid.UUID `json:"id"`
	MemberID  uuid.UUID `json:"member_id"`
	Type      string    `json:"type"`
	Title     string    `json:"title"`
	Body      string    `json:"body"`
	Data      any       `json:"data,omitempty"`
	IsRead    bool      `json:"is_read"`
	CreatedAt time.Time `json:"created_at"`
}

// --- Support Types ---

type CreateTicketRequest struct {
	Subject string `json:"subject"`
	Message string `json:"message"`
}

type TicketResponse struct {
	ID           uuid.UUID `json:"id"`
	TicketNumber int64     `json:"ticket_number"`
	Subject      string    `json:"subject"`
	Status       string    `json:"status"`
	CreatedAt    string    `json:"created_at"`
	UpdatedAt    string    `json:"updated_at"`
	ClosedAt     *string   `json:"closed_at,omitempty"`
}

// TicketDetailResponse and TicketMessageResponse are aliased from handlers package above

// AddTicketMessageRequest is an alias for AddMessageRequest for backward compatibility
type AddTicketMessageRequest = AddMessageRequest

type FAQResponse struct {
	ID       string `json:"id"`
	Question string `json:"question"`
	Answer   string `json:"answer"`
	Order    int    `json:"order"`
}

// Subscription types

type SubscriptionResponse struct {
	ID              uuid.UUID                  `json:"id"`
	MemberID        uuid.UUID                  `json:"member_id"`
	Name            string                     `json:"name"`
	MinWeight       *float64                   `json:"min_weight,omitempty"`
	MaxWeight       *float64                   `json:"max_weight,omitempty"`
	MinPrice        *int64                     `json:"min_price,omitempty"`
	MaxPrice        *int64                     `json:"max_price,omitempty"`
	MinVolume       *float64                   `json:"min_volume,omitempty"`
	MaxVolume       *float64                   `json:"max_volume,omitempty"`
	VehicleTypes    []string                   `json:"vehicle_types,omitempty"`
	VehicleSubTypes []string                   `json:"vehicle_subtypes,omitempty"`
	PaymentMethods  []string                   `json:"payment_methods,omitempty"`
	PaymentTerms    []string                   `json:"payment_terms,omitempty"`
	VatTypes        []string                   `json:"vat_types,omitempty"`
	RoutePoints     []RoutePointCriteriaResponse `json:"route_points,omitempty"`
	IsActive        bool                       `json:"is_active"`
	CreatedAt       string                     `json:"created_at"`
	UpdatedAt       string                     `json:"updated_at"`
}

type RoutePointCriteriaResponse struct {
	CountryID   int     `json:"country_id"`
	CountryName *string `json:"country_name,omitempty"`
	CityID      *int    `json:"city_id,omitempty"`
	CityName    *string `json:"city_name,omitempty"`
	Order       int     `json:"order"`
}

type CreateSubscriptionRequest struct {
	Name            string                    `json:"name"`
	MinWeight       *float64                  `json:"min_weight,omitempty"`
	MaxWeight       *float64                  `json:"max_weight,omitempty"`
	MinPrice        *int64                    `json:"min_price,omitempty"`
	MaxPrice        *int64                    `json:"max_price,omitempty"`
	MinVolume       *float64                  `json:"min_volume,omitempty"`
	MaxVolume       *float64                  `json:"max_volume,omitempty"`
	VehicleTypes    []string                  `json:"vehicle_types,omitempty"`
	VehicleSubTypes []string                  `json:"vehicle_subtypes,omitempty"`
	PaymentMethods  []string                  `json:"payment_methods,omitempty"`
	PaymentTerms    []string                  `json:"payment_terms,omitempty"`
	VatTypes        []string                  `json:"vat_types,omitempty"`
	RoutePoints     []RoutePointCriteriaRequest `json:"route_points,omitempty"`
	IsActive        bool                      `json:"is_active"`
}

type RoutePointCriteriaRequest struct {
	CountryID int  `json:"country_id"`
	CityID    *int `json:"city_id,omitempty"`
	Order     int  `json:"order"`
}

type SetActiveRequest struct {
	IsActive bool `json:"is_active"`
}

// Admin Support types

type AdminTicketsListResponse struct {
	Tickets []AdminTicketResponse `json:"tickets"`
	Total   int                   `json:"total"`
}

type AdminTicketResponse struct {
	ID           uuid.UUID `json:"id"`
	TicketNumber int64     `json:"ticket_number"`
	MemberID     uuid.UUID `json:"member_id"`
	OrgID        uuid.UUID `json:"org_id"`
	Subject      string    `json:"subject"`
	Status       string    `json:"status"`
	CreatedAt    string    `json:"created_at"`
	UpdatedAt    string    `json:"updated_at"`
	ClosedAt     *string   `json:"closed_at,omitempty"`
}

// AdminTicketDetailResponse and AdminAddMessageRequest are aliased from handlers package above
