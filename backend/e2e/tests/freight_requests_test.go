package tests

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"codeberg.org/udison/veziizi/backend/e2e/client"
	"codeberg.org/udison/veziizi/backend/e2e/fixtures"
	"codeberg.org/udison/veziizi/backend/e2e/helpers"
	"codeberg.org/udison/veziizi/backend/internal/domain/freightrequest/values"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCreateFreightRequest tests POST /api/v1/freight-requests
func TestCreateFreightRequest(t *testing.T) {
	t.Parallel()
	suite := getSuite(t)
	ctx := fixtures.NewTestContext(t, suite.BaseURL)

	tests := []struct {
		id         string
		name       string
		client     *client.Client
		modify     func(*fixtures.FreightRequestBuilder)
		wantStatus int
		wantErr    string
	}{
		// Happy path
		{
			id:         "FR-001",
			name:       "successful creation",
			client:     ctx.Customer.Client,
			modify:     nil,
			wantStatus: http.StatusCreated,
		},
		{
			id:     "FR-002",
			name:   "with comment",
			client: ctx.Customer.Client,
			modify: func(b *fixtures.FreightRequestBuilder) {
				b.WithComment("Urgent delivery needed")
			},
			wantStatus: http.StatusCreated,
		},
		{
			id:     "FR-003",
			name:   "with custom expiry",
			client: ctx.Customer.Client,
			modify: func(b *fixtures.FreightRequestBuilder) {
				b.ExpiresIn(14 * 24 * time.Hour)
			},
			wantStatus: http.StatusCreated,
		},

		// Validation errors
		{
			id:     "FR-006",
			name:   "negative weight",
			client: ctx.Customer.Client,
			modify: func(b *fixtures.FreightRequestBuilder) {
				b.WithWeight(-100)
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			id:     "FR-009",
			name:   "negative price",
			client: ctx.Customer.Client,
			modify: func(b *fixtures.FreightRequestBuilder) {
				b.WithPrice(-1000, values.CurrencyRUB.String())
			},
			wantStatus: http.StatusBadRequest,
		},

		// Auth errors
		{
			id:         "FR-010",
			name:       "without auth",
			client:     ctx.AnonClient,
			modify:     nil,
			wantStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.id+"_"+tt.name, func(t *testing.T) {
			t.Parallel()

			builder := fixtures.NewFreightRequest(t, tt.client)
			if tt.modify != nil {
				tt.modify(builder)
			}

			resp, err := builder.CreateWithStatus()
			require.NoError(t, err)
			require.Equal(t, tt.wantStatus, resp.StatusCode, string(resp.RawBody))

			if tt.wantErr != "" {
				assert.Contains(t, strings.ToLower(string(resp.RawBody)), strings.ToLower(tt.wantErr))
			}

			if resp.StatusCode == http.StatusCreated {
				assert.NotEmpty(t, resp.Body.ID.String(), "id should be set")
			}
		})
	}
}

// TestGetFreightRequests tests GET /api/v1/freight-requests
func TestGetFreightRequests(t *testing.T) {
	t.Parallel()
	suite := getSuite(t)
	ctx := fixtures.NewTestContext(t, suite.BaseURL)

	// Create some freight requests
	fr1 := fixtures.NewFreightRequest(t, ctx.Customer.Client).
		WithWeight(1000).
		WithPrice(50000, values.CurrencyRUB.String()).
		Create()

	fr2 := fixtures.NewFreightRequest(t, ctx.Customer.Client).
		WithWeight(5000).
		WithPrice(100000, values.CurrencyRUB.String()).
		Create()

	tests := []struct {
		id         string
		name       string
		filters    map[string]string
		wantStatus int
		check      func(*testing.T, *client.Response[[]client.FreightRequestResponse])
	}{
		{
			id:         "FR-014",
			name:       "list all",
			filters:    nil,
			wantStatus: http.StatusOK,
			check: func(t *testing.T, resp *client.Response[[]client.FreightRequestResponse]) {
				assert.True(t, len(resp.Body) >= 2, "should have at least 2 requests")
			},
		},
		{
			id:         "FR-016",
			name:       "filter by customer_org_id",
			filters:    map[string]string{"customer_org_id": ctx.Customer.OrganizationID.String()},
			wantStatus: http.StatusOK,
			check: func(t *testing.T, resp *client.Response[[]client.FreightRequestResponse]) {
				for _, fr := range resp.Body {
					assert.Equal(t, ctx.Customer.OrganizationID, fr.CustomerOrgID, "customer_org_id")
				}
			},
		},
		{
			id:         "FR-022",
			name:       "filter by min_weight",
			filters:    map[string]string{"min_weight": "3000"},
			wantStatus: http.StatusOK,
			check: func(t *testing.T, resp *client.Response[[]client.FreightRequestResponse]) {
				for _, fr := range resp.Body {
					assert.True(t, fr.Cargo.Weight >= 3000, "weight should be >= 3000")
				}
			},
		},
		{
			id:         "FR-030",
			name:       "pagination",
			filters:    map[string]string{"limit": "1", "offset": "0"},
			wantStatus: http.StatusOK,
			check: func(t *testing.T, resp *client.Response[[]client.FreightRequestResponse]) {
				assert.True(t, len(resp.Body) <= 1, "should have at most 1 request")
			},
		},
		{
			id:         "FR-031",
			name:       "invalid customer_org_id",
			filters:    map[string]string{"customer_org_id": "not-uuid"},
			wantStatus: http.StatusBadRequest,
		},
	}

	_ = fr1 // silence unused
	_ = fr2

	for _, tt := range tests {
		t.Run(tt.id+"_"+tt.name, func(t *testing.T) {
			// Use authenticated client - freight requests require auth
			resp, err := ctx.Customer.Client.GetFreightRequests(tt.filters)
			require.NoError(t, err)
			require.Equal(t, tt.wantStatus, resp.StatusCode, string(resp.RawBody))

			if tt.check != nil && resp.StatusCode == http.StatusOK {
				tt.check(t, resp)
			}
		})
	}
}

// TestGetFreightRequest tests GET /api/v1/freight-requests/{id}
func TestGetFreightRequest(t *testing.T) {
	t.Parallel()
	suite := getSuite(t)
	ctx := fixtures.NewTestContext(t, suite.BaseURL)

	fr := fixtures.NewFreightRequest(t, ctx.Customer.Client).Create()

	tests := []struct {
		id         string
		name       string
		frID       uuid.UUID
		wantStatus int
	}{
		{
			id:         "FR-036",
			name:       "get existing request",
			frID:       fr.ID,
			wantStatus: http.StatusOK,
		},
		{
			id:         "FR-039",
			name:       "nonexistent request",
			frID:       uuid.New(),
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.id+"_"+tt.name, func(t *testing.T) {
			// Use authenticated client - freight requests require auth
			resp, err := ctx.Customer.Client.GetFreightRequest(tt.frID)
			require.NoError(t, err)
			require.Equal(t, tt.wantStatus, resp.StatusCode, string(resp.RawBody))
		})
	}
}

// TestUpdateFreightRequest tests PATCH /api/v1/freight-requests/{id}
func TestUpdateFreightRequest(t *testing.T) {
	t.Parallel()
	suite := getSuite(t)
	ctx := fixtures.NewTestContext(t, suite.BaseURL)

	fr := fixtures.NewFreightRequest(t, ctx.Customer.Client).Create()

	tests := []struct {
		id         string
		name       string
		client     *client.Client
		frID       uuid.UUID
		wantStatus int
		wantErr    string
	}{
		{
			id:         "FR-041",
			name:       "update by owner",
			client:     ctx.Customer.Client,
			frID:       fr.ID,
			wantStatus: http.StatusNoContent,
		},
		{
			id:         "FR-046",
			name:       "update without auth",
			client:     ctx.AnonClient,
			frID:       fr.ID,
			wantStatus: http.StatusUnauthorized,
		},
		{
			id:         "FR-048",
			name:       "update by different org",
			client:     ctx.Carrier.Client,
			frID:       fr.ID,
			wantStatus: http.StatusForbidden,
			wantErr:    "not freight request owner",
		},
		{
			id:         "FR-049",
			name:       "update nonexistent",
			client:     ctx.Customer.Client,
			frID:       uuid.New(),
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.id+"_"+tt.name, func(t *testing.T) {
			// Build update request
			tomorrow := time.Now().AddDate(0, 0, 1).Format("2006-01-02")
			dayAfter := time.Now().AddDate(0, 0, 2).Format("2006-01-02")
			updateReq := client.CreateFreightRequestRequest{
				Route: client.Route{
					Points: []client.RoutePoint{
						{IsLoading: true, IsUnloading: false, CountryID: intPtr(1), CityID: intPtr(1), Address: "Moscow", DateFrom: tomorrow},
						{IsLoading: false, IsUnloading: true, CountryID: intPtr(1), CityID: intPtr(3), Address: "Kazan", DateFrom: dayAfter},
					},
				},
				Cargo:               client.Cargo{Description: "Updated cargo", Weight: 2000, Quantity: 1},
				VehicleRequirements: client.VehicleRequirements{VehicleType: values.VehicleTypeVan.String(), VehicleSubtype: values.VehicleSubTypeDryVan.String()},
				Payment:             client.Payment{Price: &client.Money{Amount: 60000, Currency: values.CurrencyRUB.String()}, VatType: values.VatTypeIncluded.String(), Method: values.PaymentMethodBankTransfer.String(), Terms: values.PaymentTermsPrepaid.String()},
			}

			resp, err := tt.client.UpdateFreightRequest(tt.frID, updateReq)
			require.NoError(t, err)
			require.Equal(t, tt.wantStatus, resp.StatusCode, string(resp.RawBody))

			if tt.wantErr != "" {
				assert.Contains(t, strings.ToLower(string(resp.RawBody)), strings.ToLower(tt.wantErr))
			}
		})
	}
}

// TestCancelFreightRequest tests DELETE /api/v1/freight-requests/{id}
func TestCancelFreightRequest(t *testing.T) {
	t.Parallel()
	suite := getSuite(t)
	ctx := fixtures.NewTestContext(t, suite.BaseURL)

	t.Run("FR-053_successful_cancel", func(t *testing.T) {
		fr := fixtures.NewFreightRequest(t, ctx.Customer.Client).Create()

		resp, err := ctx.Customer.Client.CancelFreightRequest(fr.ID, helpers.StringPtr("No longer needed"))
		require.NoError(t, err)
		require.Equal(t, http.StatusNoContent, resp.StatusCode, string(resp.RawBody))
	})

	t.Run("FR-055_cancel_without_auth", func(t *testing.T) {
		fr := fixtures.NewFreightRequest(t, ctx.Customer.Client).Create()

		resp, err := ctx.AnonClient.CancelFreightRequest(fr.ID, nil)
		require.NoError(t, err)
		require.Equal(t, http.StatusUnauthorized, resp.StatusCode, string(resp.RawBody))
	})

	t.Run("FR-057_cancel_other_org_request", func(t *testing.T) {
		fr := fixtures.NewFreightRequest(t, ctx.Customer.Client).Create()

		resp, err := ctx.Carrier.Client.CancelFreightRequest(fr.ID, nil)
		require.NoError(t, err)
		require.Equal(t, http.StatusForbidden, resp.StatusCode, string(resp.RawBody))
	})

	t.Run("FR-058_cancel_already_cancelled", func(t *testing.T) {
		fr := fixtures.NewFreightRequest(t, ctx.Customer.Client).Create()

		// Cancel first time
		ctx.Customer.Client.CancelFreightRequest(fr.ID, nil)

		// Try to cancel again
		resp, err := ctx.Customer.Client.CancelFreightRequest(fr.ID, nil)
		require.NoError(t, err)
		require.Equal(t, http.StatusConflict, resp.StatusCode, string(resp.RawBody))
	})
}

// TestCreateOffer tests POST /api/v1/freight-requests/{id}/offers
func TestCreateOffer(t *testing.T) {
	t.Parallel()
	suite := getSuite(t)
	ctx := fixtures.NewTestContext(t, suite.BaseURL)

	fr := fixtures.NewFreightRequest(t, ctx.Customer.Client).Create()

	tests := []struct {
		id         string
		name       string
		client     *client.Client
		frID       uuid.UUID
		wantStatus int
		wantErr    string
	}{
		{
			id:         "FR-065",
			name:       "successful offer",
			client:     ctx.Carrier.Client,
			frID:       fr.ID,
			wantStatus: http.StatusCreated,
		},
		{
			id:         "FR-070",
			name:       "offer on own request",
			client:     ctx.Customer.Client,
			frID:       fr.ID,
			wantStatus: http.StatusBadRequest,
			wantErr:    "cannot make offer on own request",
		},
		{
			id:         "FR-071",
			name:       "offer without auth",
			client:     ctx.AnonClient,
			frID:       fr.ID,
			wantStatus: http.StatusUnauthorized,
		},
		{
			id:         "FR-072",
			name:       "offer on nonexistent request",
			client:     ctx.Carrier.Client,
			frID:       uuid.New(),
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.id+"_"+tt.name, func(t *testing.T) {
			// Use different carrier for each test to avoid duplicate offer error
			var testClient *client.Client
			if tt.client == ctx.Carrier.Client && tt.id != "FR-065" {
				testClient = ctx.QuickCarrier().Client
			} else {
				testClient = tt.client
			}

			resp, err := testClient.CreateOffer(tt.frID, client.CreateOfferRequest{
				Price:         client.Money{Amount: 45000, Currency: values.CurrencyRUB.String()},
				VATType:       values.VatTypeIncluded.String(),
				PaymentMethod: values.PaymentMethodBankTransfer.String(),
			})
			require.NoError(t, err)
			require.Equal(t, tt.wantStatus, resp.StatusCode, string(resp.RawBody))

			if tt.wantErr != "" {
				assert.Contains(t, strings.ToLower(string(resp.RawBody)), strings.ToLower(tt.wantErr))
			}
		})
	}
}

// TestDuplicateOffer tests FR-073: Offer already exists
func TestDuplicateOffer(t *testing.T) {
	t.Parallel()
	suite := getSuite(t)
	ctx := fixtures.NewTestContext(t, suite.BaseURL)

	fr := fixtures.NewFreightRequest(t, ctx.Customer.Client).Create()

	// First offer
	resp1, err := ctx.Carrier.Client.CreateOffer(fr.ID, client.CreateOfferRequest{
		Price:         client.Money{Amount: 45000, Currency: values.CurrencyRUB.String()},
		VATType:       values.VatTypeIncluded.String(),
		PaymentMethod: values.PaymentMethodBankTransfer.String(),
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp1.StatusCode, string(resp1.RawBody))

	// Second offer from same org
	resp2, err := ctx.Carrier.Client.CreateOffer(fr.ID, client.CreateOfferRequest{
		Price:         client.Money{Amount: 40000, Currency: values.CurrencyRUB.String()},
		VATType:       values.VatTypeIncluded.String(),
		PaymentMethod: values.PaymentMethodBankTransfer.String(),
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusConflict, resp2.StatusCode, string(resp2.RawBody))
	assert.Contains(t, strings.ToLower(string(resp2.RawBody)), "already exists")
}

// TestOfferFlow tests the complete offer flow: create -> select -> confirm
func TestOfferFlow(t *testing.T) {
	t.Parallel()
	suite := getSuite(t)
	ctx := fixtures.NewTestContext(t, suite.BaseURL)

	fr := fixtures.NewFreightRequest(t, ctx.Customer.Client).Create()
	offer := fixtures.NewOffer(t, ctx.Carrier.Client, fr.ID).Create()

	t.Run("FR-091_select_offer", func(t *testing.T) {
		resp, err := ctx.Customer.Client.SelectOffer(fr.ID, offer.OfferID)
		require.NoError(t, err)
		require.Equal(t, http.StatusNoContent, resp.StatusCode, string(resp.RawBody))
	})

	t.Run("FR-102_confirm_offer", func(t *testing.T) {
		resp, err := ctx.Carrier.Client.ConfirmOffer(fr.ID, offer.OfferID)
		require.NoError(t, err)
		require.Equal(t, http.StatusNoContent, resp.StatusCode, string(resp.RawBody))
	})

	t.Run("FR-103_order_created", func(t *testing.T) {
		// Wait for order to be created by worker
		assert.Eventually(t, func() bool {
			ordersResp, _ := ctx.Customer.Client.GetOrders(map[string]string{
				"freight_request_id": fr.ID.String(),
			})
			return ordersResp.StatusCode == http.StatusOK && len(ordersResp.Body) > 0
		}, 5*time.Second, 100*time.Millisecond, "order should be created")
	})
}

// TestGetOffers tests GET /api/v1/freight-requests/{id}/offers
func TestGetOffers(t *testing.T) {
	t.Parallel()
	suite := getSuite(t)
	ctx := fixtures.NewTestContext(t, suite.BaseURL)

	fr := fixtures.NewFreightRequest(t, ctx.Customer.Client).Create()
	fixtures.NewOffer(t, ctx.Carrier.Client, fr.ID).Create()

	t.Run("FR-077_customer_sees_all_offers", func(t *testing.T) {
		resp, err := ctx.Customer.Client.GetOffers(fr.ID)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode, string(resp.RawBody))
		assert.True(t, len(resp.Body) >= 1, "should see offers")
	})

	t.Run("FR-078_carrier_sees_own_offer", func(t *testing.T) {
		resp, err := ctx.Carrier.Client.GetOffers(fr.ID)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode, string(resp.RawBody))
		assert.True(t, len(resp.Body) >= 1, "should see own offer")
	})

	t.Run("FR-084_other_org_sees_empty", func(t *testing.T) {
		otherCarrier := ctx.QuickCarrier()
		resp, err := otherCarrier.Client.GetOffers(fr.ID)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode, string(resp.RawBody))
		assert.Equal(t, 0, len(resp.Body), "should see empty list")
	})
}

// Helper function
func intPtr(i int) *int {
	return &i
}
