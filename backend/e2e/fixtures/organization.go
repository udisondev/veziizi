// Package fixtures provides builders for creating test data.
package fixtures

import (
	"fmt"
	"testing"
	"time"

	"github.com/udisondev/veziizi/backend/e2e/client"
	"github.com/google/uuid"
)

// OrganizationBuilder builds organization registration requests.
type OrganizationBuilder struct {
	t        *testing.T
	client   *client.Client
	name     string
	email    string
	phone    string
	inn      string
	country  string
	address  string
	owner    OwnerData
	uniqueID string
}

type OwnerData struct {
	Name     string
	Email    string
	Phone    string
	Password string
}

// CreatedOrganization holds the result of organization creation.
type CreatedOrganization struct {
	OrganizationID uuid.UUID
	MemberID       uuid.UUID
	OwnerEmail     string
	OwnerPassword  string
	Client         *client.Client // Logged-in client for this org
}

// NewOrganization creates a new organization builder with random unique data.
func NewOrganization(t *testing.T, c *client.Client) *OrganizationBuilder {
	uniqueID := uuid.New().String()[:8]
	return &OrganizationBuilder{
		t:        t,
		client:   c,
		uniqueID: uniqueID,
		name:     fmt.Sprintf("Test Org %s", uniqueID),
		email:    fmt.Sprintf("org-%s@test.local", uniqueID),
		phone:    "+79001234567",
		inn:      fmt.Sprintf("12345%s", uniqueID[:5]),
		country:  "RU",
		address:  "Moscow, Test Street 1",
		owner: OwnerData{
			Name:     fmt.Sprintf("Owner %s", uniqueID),
			Email:    fmt.Sprintf("owner-%s@test.local", uniqueID),
			Phone:    "+79001234568",
			Password: "password123",
		},
	}
}

// WithName sets the organization name.
func (b *OrganizationBuilder) WithName(name string) *OrganizationBuilder {
	b.name = name
	return b
}

// WithEmail sets the organization email.
func (b *OrganizationBuilder) WithEmail(email string) *OrganizationBuilder {
	b.email = email
	return b
}

// WithPhone sets the organization phone.
func (b *OrganizationBuilder) WithPhone(phone string) *OrganizationBuilder {
	b.phone = phone
	return b
}

// WithINN sets the organization INN.
func (b *OrganizationBuilder) WithINN(inn string) *OrganizationBuilder {
	b.inn = inn
	return b
}

// WithCountry sets the organization country.
func (b *OrganizationBuilder) WithCountry(country string) *OrganizationBuilder {
	b.country = country
	return b
}

// WithAddress sets the organization address.
func (b *OrganizationBuilder) WithAddress(address string) *OrganizationBuilder {
	b.address = address
	return b
}

// WithOwner sets the owner data.
func (b *OrganizationBuilder) WithOwner(name, email, phone, password string) *OrganizationBuilder {
	b.owner = OwnerData{
		Name:     name,
		Email:    email,
		Phone:    phone,
		Password: password,
	}
	return b
}

// WithOwnerEmail sets only the owner email (uses default for other fields).
func (b *OrganizationBuilder) WithOwnerEmail(email string) *OrganizationBuilder {
	b.owner.Email = email
	return b
}

// WithOwnerPassword sets only the owner password.
func (b *OrganizationBuilder) WithOwnerPassword(password string) *OrganizationBuilder {
	b.owner.Password = password
	return b
}

// Build returns the registration request without creating the organization.
func (b *OrganizationBuilder) Build() client.RegisterOrganizationRequest {
	return client.RegisterOrganizationRequest{
		Name:          b.name,
		Email:         b.email,
		Phone:         b.phone,
		INN:           b.inn,
		Country:       b.country,
		Address:       b.address,
		OwnerName:     b.owner.Name,
		OwnerEmail:    b.owner.Email,
		OwnerPhone:    b.owner.Phone,
		OwnerPassword: b.owner.Password,
	}
}

// Create registers the organization and returns the result.
// Fails the test if registration fails.
func (b *OrganizationBuilder) Create() *CreatedOrganization {
	b.t.Helper()

	req := b.Build()
	resp, err := b.client.RegisterOrganization(req)
	if err != nil {
		b.t.Fatalf("failed to register organization: %v", err)
	}
	if resp.StatusCode != 201 {
		b.t.Fatalf("unexpected status code %d: %s", resp.StatusCode, string(resp.RawBody))
	}

	// Create a new client and login with exponential backoff
	// (event handlers need time to process events into lookup tables)
	orgClient := b.client.Clone()
	var loginResp *client.Response[client.LoginResponse]
	backoff := 10 * time.Millisecond
	maxBackoff := 200 * time.Millisecond
	deadline := time.Now().Add(3 * time.Second)

	for time.Now().Before(deadline) {
		loginResp, err = orgClient.Login(b.owner.Email, b.owner.Password)
		if err != nil {
			b.t.Fatalf("failed to login after registration: %v", err)
		}
		if loginResp.StatusCode == 200 {
			break
		}
		time.Sleep(backoff)
		if backoff < maxBackoff {
			backoff = min(backoff*2, maxBackoff)
		}
	}
	if loginResp.StatusCode != 200 {
		b.t.Fatalf("failed to login after retries: %s", string(loginResp.RawBody))
	}

	return &CreatedOrganization{
		OrganizationID: resp.Body.OrganizationID,
		MemberID:       resp.Body.MemberID,
		OwnerEmail:     b.owner.Email,
		OwnerPassword:  b.owner.Password,
		Client:         orgClient,
	}
}

// CreateWithStatus registers the organization and returns both the result and HTTP status.
// Does not fail the test on error - use for negative testing.
func (b *OrganizationBuilder) CreateWithStatus() (*client.Response[client.RegisterOrganizationResponse], error) {
	b.t.Helper()
	return b.client.RegisterOrganization(b.Build())
}

// ActiveOrganization creates an organization and approves it via admin.
// Requires admin credentials to be set up.
type ActiveOrganizationBuilder struct {
	*OrganizationBuilder
	adminClient *client.Client
}

// NewActiveOrganization creates a builder for an approved organization.
func NewActiveOrganization(t *testing.T, userClient, adminClient *client.Client) *ActiveOrganizationBuilder {
	return &ActiveOrganizationBuilder{
		OrganizationBuilder: NewOrganization(t, userClient),
		adminClient:         adminClient,
	}
}

// Create registers and approves the organization.
func (b *ActiveOrganizationBuilder) Create() *CreatedOrganization {
	b.t.Helper()

	// Create the organization
	org := b.OrganizationBuilder.Create()

	// Approve via admin
	resp, err := b.adminClient.AdminApproveOrganization(org.OrganizationID, nil)
	if err != nil {
		b.t.Fatalf("failed to approve organization: %v", err)
	}
	if resp.StatusCode != 204 {
		b.t.Fatalf("failed to approve organization: %s", string(resp.RawBody))
	}

	return org
}
