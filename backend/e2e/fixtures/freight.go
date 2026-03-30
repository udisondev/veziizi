package fixtures

import (
	"testing"
	"time"

	"github.com/udisondev/veziizi/backend/e2e/client"
	"github.com/udisondev/veziizi/backend/internal/domain/freightrequest/values"
	"github.com/google/uuid"
)

// FreightRequestBuilder builds freight request creation requests.
type FreightRequestBuilder struct {
	t       *testing.T
	client  *client.Client
	route   client.Route
	cargo   client.Cargo
	vehicle client.VehicleRequirements
	payment client.Payment
	comment *string
	expires *time.Time
}

// CreatedFreightRequest holds the result of freight request creation.
type CreatedFreightRequest struct {
	ID       uuid.UUID
	Client   *client.Client // Client that created the request
	OrgID    uuid.UUID
	MemberID uuid.UUID
}

// NewFreightRequest creates a new freight request builder with default values.
func NewFreightRequest(t *testing.T, c *client.Client) *FreightRequestBuilder {
	// Default date is tomorrow
	tomorrow := time.Now().AddDate(0, 0, 1).Format("2006-01-02")
	dayAfter := time.Now().AddDate(0, 0, 2).Format("2006-01-02")

	return &FreightRequestBuilder{
		t:      t,
		client: c,
		route: client.Route{
			Points: []client.RoutePoint{
				{
					IsLoading:   true,
					IsUnloading: false,
					CountryID:   intPtr(1),
					CityID:      intPtr(1),
					Address:     "Moscow, Test Street 1",
					DateFrom:    tomorrow,
				},
				{
					IsLoading:   false,
					IsUnloading: true,
					CountryID:   intPtr(1),
					CityID:      intPtr(2),
					Address:     "Saint Petersburg, Test Street 2",
					DateFrom:    dayAfter,
				},
			},
		},
		cargo: client.Cargo{
			Description: "Test cargo",
			Weight:      1000,
			Volume:      10,
			Quantity:    1,
		},
		vehicle: client.VehicleRequirements{
			VehicleType:    values.VehicleTypeVan.String(),
			VehicleSubtype: values.VehicleSubTypeDryVan.String(),
		},
		payment: client.Payment{
			Price:   &client.Money{Amount: 50000, Currency: values.CurrencyRUB.String()},
			VatType: values.VatTypeIncluded.String(),
			Method:  values.PaymentMethodBankTransfer.String(),
			Terms:   values.PaymentTermsPrepaid.String(),
		},
	}
}

// WithRoute sets the route with loading and unloading points.
func (b *FreightRequestBuilder) WithRoute(loading, unloading client.RoutePoint) *FreightRequestBuilder {
	// Ensure proper flags
	loading.IsLoading = true
	loading.IsUnloading = false
	unloading.IsLoading = false
	unloading.IsUnloading = true
	b.route = client.Route{Points: []client.RoutePoint{loading, unloading}}
	return b
}

// WithRoutePoints sets custom route points.
func (b *FreightRequestBuilder) WithRoutePoints(points []client.RoutePoint) *FreightRequestBuilder {
	b.route = client.Route{Points: points}
	return b
}

// WithCargo sets the cargo.
func (b *FreightRequestBuilder) WithCargo(description string, weight float64) *FreightRequestBuilder {
	b.cargo.Description = description
	b.cargo.Weight = weight
	return b
}

// WithWeight sets only the weight.
func (b *FreightRequestBuilder) WithWeight(weight float64) *FreightRequestBuilder {
	b.cargo.Weight = weight
	return b
}

// WithVolume sets the volume.
func (b *FreightRequestBuilder) WithVolume(volume float64) *FreightRequestBuilder {
	b.cargo.Volume = volume
	return b
}

// WithQuantity sets the quantity.
func (b *FreightRequestBuilder) WithQuantity(quantity int) *FreightRequestBuilder {
	b.cargo.Quantity = quantity
	return b
}

// WithVehicleType sets the vehicle type.
func (b *FreightRequestBuilder) WithVehicleType(vehicleType string) *FreightRequestBuilder {
	b.vehicle.VehicleType = vehicleType
	return b
}

// WithVehicleSubtype sets the vehicle subtype.
func (b *FreightRequestBuilder) WithVehicleSubtype(subtype string) *FreightRequestBuilder {
	b.vehicle.VehicleSubtype = subtype
	return b
}

// WithPrice sets the price.
func (b *FreightRequestBuilder) WithPrice(amount int64, currency string) *FreightRequestBuilder {
	b.payment.Price = &client.Money{Amount: amount, Currency: currency}
	return b
}

// WithPayment sets full payment info.
func (b *FreightRequestBuilder) WithPayment(amount int64, currency, vatType, method, terms string) *FreightRequestBuilder {
	b.payment = client.Payment{
		Price:   &client.Money{Amount: amount, Currency: currency},
		VatType: vatType,
		Method:  method,
		Terms:   terms,
	}
	return b
}

// WithComment sets the comment.
func (b *FreightRequestBuilder) WithComment(comment string) *FreightRequestBuilder {
	b.comment = &comment
	return b
}

// WithExpiresAt sets the expiration time.
func (b *FreightRequestBuilder) WithExpiresAt(t time.Time) *FreightRequestBuilder {
	b.expires = &t
	return b
}

// ExpiresIn sets expiration relative to now.
func (b *FreightRequestBuilder) ExpiresIn(d time.Duration) *FreightRequestBuilder {
	t := time.Now().Add(d)
	b.expires = &t
	return b
}

// Build returns the request without creating it.
func (b *FreightRequestBuilder) Build() client.CreateFreightRequestRequest {
	return client.CreateFreightRequestRequest{
		Route:               b.route,
		Cargo:               b.cargo,
		VehicleRequirements: b.vehicle,
		Payment:             b.payment,
		Comment:             b.comment,
		ExpiresAt:           b.expires,
	}
}

// Create creates the freight request and returns the result.
func (b *FreightRequestBuilder) Create() *CreatedFreightRequest {
	b.t.Helper()

	// Get current user info to capture org/member IDs
	meResp, err := b.client.Me()
	if err != nil {
		b.t.Fatalf("failed to get current user: %v", err)
	}
	if meResp.StatusCode != 200 {
		b.t.Fatalf("failed to get current user: %s", string(meResp.RawBody))
	}

	req := b.Build()
	resp, err := b.client.CreateFreightRequest(req)
	if err != nil {
		b.t.Fatalf("failed to create freight request: %v", err)
	}
	if resp.StatusCode != 201 {
		b.t.Fatalf("unexpected status code %d: %s", resp.StatusCode, string(resp.RawBody))
	}

	return &CreatedFreightRequest{
		ID:       resp.Body.ID,
		Client:   b.client,
		OrgID:    meResp.Body.OrganizationID,
		MemberID: meResp.Body.MemberID,
	}
}

// CreateWithStatus creates and returns both result and status (for negative testing).
func (b *FreightRequestBuilder) CreateWithStatus() (*client.Response[struct{ ID uuid.UUID }], error) {
	b.t.Helper()
	return b.client.CreateFreightRequest(b.Build())
}

// OfferBuilder builds offer creation requests.
type OfferBuilder struct {
	t                *testing.T
	client           *client.Client
	freightRequestID uuid.UUID
	priceAmount      int64
	currency         string
	vatType          string
	paymentMethod    string
	comment          string
}

// CreatedOffer holds the result of offer creation.
type CreatedOffer struct {
	OfferID          uuid.UUID
	FreightRequestID uuid.UUID
	Client           *client.Client
	OrgID            uuid.UUID
	MemberID         uuid.UUID
}

// NewOffer creates a new offer builder.
func NewOffer(t *testing.T, c *client.Client, freightRequestID uuid.UUID) *OfferBuilder {
	return &OfferBuilder{
		t:                t,
		client:           c,
		freightRequestID: freightRequestID,
		priceAmount:      45000,
		currency:         values.CurrencyRUB.String(),
		vatType:          values.VatTypeIncluded.String(),
		paymentMethod:    values.PaymentMethodBankTransfer.String(),
	}
}

// WithPrice sets the price amount.
func (b *OfferBuilder) WithPrice(price int64) *OfferBuilder {
	b.priceAmount = price
	return b
}

// WithCurrency sets the currency.
func (b *OfferBuilder) WithCurrency(currency string) *OfferBuilder {
	b.currency = currency
	return b
}

// WithVATType sets the VAT type.
func (b *OfferBuilder) WithVATType(vatType string) *OfferBuilder {
	b.vatType = vatType
	return b
}

// WithPaymentMethod sets the payment method.
func (b *OfferBuilder) WithPaymentMethod(method string) *OfferBuilder {
	b.paymentMethod = method
	return b
}

// WithComment sets the comment.
func (b *OfferBuilder) WithComment(comment string) *OfferBuilder {
	b.comment = comment
	return b
}

// Build returns the request without creating it.
func (b *OfferBuilder) Build() client.CreateOfferRequest {
	return client.CreateOfferRequest{
		Price:         client.Money{Amount: b.priceAmount, Currency: b.currency},
		Comment:       b.comment,
		VATType:       b.vatType,
		PaymentMethod: b.paymentMethod,
	}
}

// Create creates the offer and returns the result.
func (b *OfferBuilder) Create() *CreatedOffer {
	b.t.Helper()

	// Get current user info
	meResp, err := b.client.Me()
	if err != nil {
		b.t.Fatalf("failed to get current user: %v", err)
	}
	if meResp.StatusCode != 200 {
		b.t.Fatalf("failed to get current user: %s", string(meResp.RawBody))
	}

	req := b.Build()
	resp, err := b.client.CreateOffer(b.freightRequestID, req)
	if err != nil {
		b.t.Fatalf("failed to create offer: %v", err)
	}
	if resp.StatusCode != 201 {
		b.t.Fatalf("unexpected status code %d: %s", resp.StatusCode, string(resp.RawBody))
	}

	return &CreatedOffer{
		OfferID:          resp.Body.OfferID,
		FreightRequestID: b.freightRequestID,
		Client:           b.client,
		OrgID:            meResp.Body.OrganizationID,
		MemberID:         meResp.Body.MemberID,
	}
}

// CreateWithStatus creates and returns both result and status.
func (b *OfferBuilder) CreateWithStatus() (*client.Response[client.CreateOfferResponse], error) {
	b.t.Helper()
	return b.client.CreateOffer(b.freightRequestID, b.Build())
}

// Helper functions
func intPtr(i int) *int {
	return &i
}

func float64Ptr(f float64) *float64 {
	return &f
}

func stringPtr(s string) *string {
	return &s
}
