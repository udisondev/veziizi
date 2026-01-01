package fixtures

import (
	"testing"
	"time"

	"codeberg.org/udison/veziizi/backend/e2e/client"
	"github.com/google/uuid"
)

// TestContext holds common test context with pre-created entities.
type TestContext struct {
	T *testing.T

	// Admin client (logged in as platform admin)
	AdminClient *client.Client

	// Customer organization
	Customer *CreatedOrganization

	// Carrier organization
	Carrier *CreatedOrganization

	// Base client without auth (for public endpoints)
	AnonClient *client.Client
}

// NewTestContext creates a context with admin, customer and carrier organizations.
func NewTestContext(t *testing.T, baseURL string) *TestContext {
	t.Helper()

	anonClient := client.New(baseURL)

	// Setup admin
	adminClient := anonClient.Clone()
	// Note: Admin should be created beforehand via make create-admin-dev
	adminLogin, err := adminClient.AdminLogin("admin@veziizi.local", "admin123")
	if err != nil {
		t.Fatalf("failed to login as admin: %v", err)
	}
	if adminLogin.StatusCode != 200 {
		t.Fatalf("admin login failed (run 'make create-admin-dev' first): %s", string(adminLogin.RawBody))
	}

	// Create and approve customer organization
	customer := NewActiveOrganization(t, anonClient, adminClient).Create()

	// Create and approve carrier organization
	carrier := NewActiveOrganization(t, anonClient, adminClient).Create()

	return &TestContext{
		T:           t,
		AdminClient: adminClient,
		Customer:    customer,
		Carrier:     carrier,
		AnonClient:  anonClient,
	}
}

// CreateFreightWithOffer creates a freight request and an offer on it.
func (ctx *TestContext) CreateFreightWithOffer() (*CreatedFreightRequest, *CreatedOffer) {
	ctx.T.Helper()

	// Customer creates freight request
	fr := NewFreightRequest(ctx.T, ctx.Customer.Client).Create()

	// Carrier makes offer
	offer := NewOffer(ctx.T, ctx.Carrier.Client, fr.ID).Create()

	return fr, offer
}

// CreateSelectedOffer creates a freight request with a selected offer.
func (ctx *TestContext) CreateSelectedOffer() (*CreatedFreightRequest, *CreatedOffer) {
	ctx.T.Helper()

	fr, offer := ctx.CreateFreightWithOffer()

	// Customer selects offer
	resp, err := ctx.Customer.Client.SelectOffer(fr.ID, offer.OfferID)
	if err != nil {
		ctx.T.Fatalf("failed to select offer: %v", err)
	}
	if resp.StatusCode != 204 {
		ctx.T.Fatalf("failed to select offer: %s", string(resp.RawBody))
	}

	return fr, offer
}

// CreateConfirmedOrder creates a full order (freight request -> offer -> selection -> confirmation).
func (ctx *TestContext) CreateConfirmedOrder() (*CreatedFreightRequest, *CreatedOffer, uuid.UUID) {
	ctx.T.Helper()

	fr, offer := ctx.CreateSelectedOffer()

	// Carrier confirms offer
	resp, err := ctx.Carrier.Client.ConfirmOffer(fr.ID, offer.OfferID)
	if err != nil {
		ctx.T.Fatalf("failed to confirm offer: %v", err)
	}
	if resp.StatusCode != 204 {
		ctx.T.Fatalf("failed to confirm offer: %s", string(resp.RawBody))
	}

	// Wait for order to be created by worker
	orderID := ctx.waitForOrder(fr.ID)

	return fr, offer, orderID
}

// waitForOrder waits for order to be created for a freight request.
// Uses exponential backoff: 10ms -> 20ms -> 40ms -> ... -> 500ms max.
func (ctx *TestContext) waitForOrder(frID uuid.UUID) uuid.UUID {
	ctx.T.Helper()

	// Exponential backoff for faster response in normal case
	backoff := 10 * time.Millisecond
	maxBackoff := 500 * time.Millisecond
	deadline := time.Now().Add(10 * time.Second)

	for time.Now().Before(deadline) {
		ordersResp, err := ctx.Customer.Client.GetOrders(map[string]string{
			"freight_request_id": frID.String(),
		})
		if err == nil && ordersResp.StatusCode == 200 && len(ordersResp.Body) > 0 {
			return ordersResp.Body[0].ID
		}
		time.Sleep(backoff)
		if backoff < maxBackoff {
			backoff = min(backoff*2, maxBackoff)
		}
	}

	ctx.T.Fatalf("order was not created for freight request %s", frID)
	return uuid.Nil
}

// AddMemberToOrg creates and accepts an invitation, returning the new member's client.
func (ctx *TestContext) AddMemberToOrg(org *CreatedOrganization, role string) *client.Client {
	ctx.T.Helper()

	uniqueID := uuid.New().String()[:8]
	email := "member-" + uniqueID + "@test.local"

	// Create invitation
	invResp, err := org.Client.CreateInvitation(org.OrganizationID, client.CreateInvitationRequest{
		Email: email,
		Role:  role,
	})
	if err != nil {
		ctx.T.Fatalf("failed to create invitation: %v", err)
	}
	if invResp.StatusCode != 201 {
		ctx.T.Fatalf("failed to create invitation: %s", string(invResp.RawBody))
	}

	// Wait for invitation with exponential backoff
	token := invResp.Body.Token
	backoff := 10 * time.Millisecond
	maxBackoff := 200 * time.Millisecond
	deadline := time.Now().Add(5 * time.Second)

	for time.Now().Before(deadline) {
		getResp, err := ctx.AnonClient.GetInvitationByToken(token)
		if err == nil && getResp.StatusCode == 200 {
			break
		}
		time.Sleep(backoff)
		if backoff < maxBackoff {
			backoff = min(backoff*2, maxBackoff)
		}
	}

	// Accept invitation
	name := "Member " + uniqueID
	phone := "+79009876543"
	acceptResp, err := ctx.AnonClient.AcceptInvitation(token, client.AcceptInvitationRequest{
		Password: "password123",
		Name:     &name,
		Phone:    &phone,
	})
	if err != nil {
		ctx.T.Fatalf("failed to accept invitation: %v", err)
	}
	if acceptResp.StatusCode != 200 {
		ctx.T.Fatalf("failed to accept invitation: %s", string(acceptResp.RawBody))
	}

	// Wait for member with exponential backoff
	memberClient := ctx.AnonClient.Clone()
	backoff = 10 * time.Millisecond
	deadline = time.Now().Add(5 * time.Second)

	for time.Now().Before(deadline) {
		loginResp, err := memberClient.Login(email, "password123")
		if err == nil && loginResp.StatusCode == 200 {
			return memberClient
		}
		time.Sleep(backoff)
		if backoff < maxBackoff {
			backoff = min(backoff*2, maxBackoff)
		}
	}

	ctx.T.Fatalf("failed to login as new member after waiting")
	return nil
}

// QuickCustomer creates a new customer organization quickly.
func (ctx *TestContext) QuickCustomer() *CreatedOrganization {
	return NewActiveOrganization(ctx.T, ctx.AnonClient, ctx.AdminClient).Create()
}

// QuickCarrier creates a new carrier organization quickly.
func (ctx *TestContext) QuickCarrier() *CreatedOrganization {
	return NewActiveOrganization(ctx.T, ctx.AnonClient, ctx.AdminClient).Create()
}
