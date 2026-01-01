package tests

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"codeberg.org/udison/veziizi/backend/e2e/client"
	"codeberg.org/udison/veziizi/backend/e2e/fixtures"
	"codeberg.org/udison/veziizi/backend/e2e/helpers"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// TestGetOrders tests GET /api/v1/orders
func TestGetOrders(t *testing.T) {
	t.Parallel()
	s := getSuite(t)
	ctx := fixtures.NewTestContext(t, s.BaseURL)

	// Create a confirmed order first
	fr, _, orderID := ctx.CreateConfirmedOrder()
	require.NotEqual(t, uuid.Nil, orderID, "order should be created")

	tests := []struct {
		id         string
		name       string
		client     *client.Client
		filters    map[string]string
		wantStatus int
		wantErr    string
		check      func(*testing.T, *client.Response[[]client.OrderResponse])
	}{
		// Happy path
		{
			id:         "ORD-001",
			name:       "list orders",
			client:     ctx.Customer.Client,
			filters:    nil,
			wantStatus: http.StatusOK,
			check: func(t *testing.T, resp *client.Response[[]client.OrderResponse]) {
				assert.True(t, len(resp.Body) >= 1, "should have at least 1 order")
			},
		},
		{
			id:         "ORD-003",
			name:       "filter by customer_org_id",
			client:     ctx.Customer.Client,
			filters:    map[string]string{"customer_org_id": ctx.Customer.OrganizationID.String()},
			wantStatus: http.StatusOK,
			check: func(t *testing.T, resp *client.Response[[]client.OrderResponse]) {
				for _, order := range resp.Body {
					assert.Equal(t, ctx.Customer.OrganizationID, order.CustomerOrgID)
				}
			},
		},
		{
			id:         "ORD-004",
			name:       "filter by carrier_org_id",
			client:     ctx.Carrier.Client,
			filters:    map[string]string{"carrier_org_id": ctx.Carrier.OrganizationID.String()},
			wantStatus: http.StatusOK,
			check: func(t *testing.T, resp *client.Response[[]client.OrderResponse]) {
				for _, order := range resp.Body {
					assert.Equal(t, ctx.Carrier.OrganizationID, order.CarrierOrgID)
				}
			},
		},
		{
			id:         "ORD-005",
			name:       "filter by freight_request_id",
			client:     ctx.Customer.Client,
			filters:    map[string]string{"freight_request_id": fr.ID.String()},
			wantStatus: http.StatusOK,
			check: func(t *testing.T, resp *client.Response[[]client.OrderResponse]) {
				assert.Equal(t, 1, len(resp.Body), "should have exactly 1 order")
				assert.Equal(t, fr.ID, resp.Body[0].FreightRequestID)
			},
		},
		{
			id:         "ORD-006",
			name:       "filter by status",
			client:     ctx.Customer.Client,
			filters:    map[string]string{"status": "active"},
			wantStatus: http.StatusOK,
			check: func(t *testing.T, resp *client.Response[[]client.OrderResponse]) {
				for _, order := range resp.Body {
					assert.Equal(t, "active", order.Status)
				}
			},
		},
		{
			id:         "ORD-007",
			name:       "pagination",
			client:     ctx.Customer.Client,
			filters:    map[string]string{"limit": "1", "offset": "0"},
			wantStatus: http.StatusOK,
			check: func(t *testing.T, resp *client.Response[[]client.OrderResponse]) {
				assert.LessOrEqual(t, len(resp.Body), 1, "should respect limit")
			},
		},

		// Validation errors
		{
			id:         "ORD-008",
			name:       "invalid customer_org_id",
			client:     ctx.Customer.Client,
			filters:    map[string]string{"customer_org_id": "not-a-uuid"},
			wantStatus: http.StatusBadRequest,
			wantErr:    "invalid customer_org_id",
		},
		{
			id:         "ORD-009",
			name:       "invalid limit (0)",
			client:     ctx.Customer.Client,
			filters:    map[string]string{"limit": "0"},
			wantStatus: http.StatusBadRequest,
			wantErr:    "invalid limit",
		},
		{
			id:         "ORD-010",
			name:       "invalid offset (negative)",
			client:     ctx.Customer.Client,
			filters:    map[string]string{"offset": "-1"},
			wantStatus: http.StatusBadRequest,
			wantErr:    "invalid offset",
		},
		{
			id:         "ORD-011",
			name:       "limit > 100",
			client:     ctx.Customer.Client,
			filters:    map[string]string{"limit": "200"},
			wantStatus: http.StatusBadRequest,
			wantErr:    "invalid limit",
		},

		// Auth errors
		{
			id:         "ORD-012",
			name:       "without auth",
			client:     ctx.AnonClient,
			filters:    nil,
			wantStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.id+"_"+tt.name, func(t *testing.T) {
			resp, err := tt.client.GetOrders(tt.filters)
			require.NoError(t, err)
			require.Equal(t, tt.wantStatus, resp.StatusCode, string(resp.RawBody))

			if tt.wantErr != "" {
				assert.Contains(t, strings.ToLower(string(resp.RawBody)), strings.ToLower(tt.wantErr))
			}

			if tt.check != nil && resp.StatusCode == http.StatusOK {
				tt.check(t, resp)
			}
		})
	}
}

// TestGetOrder tests GET /api/v1/orders/{id}
func TestGetOrder(t *testing.T) {
	t.Parallel()
	s := getSuite(t)
	ctx := fixtures.NewTestContext(t, s.BaseURL)

	// Create a confirmed order
	_, _, orderID := ctx.CreateConfirmedOrder()
	require.NotEqual(t, uuid.Nil, orderID)

	// Create another org for access tests
	otherOrg := ctx.QuickCustomer()

	tests := []struct {
		id         string
		name       string
		client     *client.Client
		orderID    uuid.UUID
		useRaw     bool
		rawID      string
		wantStatus int
		wantErr    string
		check      func(*testing.T, *client.Response[client.OrderResponse])
	}{
		// Happy path
		{
			id:         "ORD-014",
			name:       "get as customer",
			client:     ctx.Customer.Client,
			orderID:    orderID,
			wantStatus: http.StatusOK,
			check: func(t *testing.T, resp *client.Response[client.OrderResponse]) {
				assert.Equal(t, orderID, resp.Body.ID)
				assert.Equal(t, ctx.Customer.OrganizationID, resp.Body.CustomerOrgID)
			},
		},
		{
			id:         "ORD-015",
			name:       "get as carrier",
			client:     ctx.Carrier.Client,
			orderID:    orderID,
			wantStatus: http.StatusOK,
			check: func(t *testing.T, resp *client.Response[client.OrderResponse]) {
				assert.Equal(t, orderID, resp.Body.ID)
				assert.Equal(t, ctx.Carrier.OrganizationID, resp.Body.CarrierOrgID)
			},
		},
		{
			id:         "ORD-019",
			name:       "includes org names",
			client:     ctx.Customer.Client,
			orderID:    orderID,
			wantStatus: http.StatusOK,
			check: func(t *testing.T, resp *client.Response[client.OrderResponse]) {
				assert.NotEmpty(t, resp.Body.CustomerOrgName, "should have customer org name")
				assert.NotEmpty(t, resp.Body.CarrierOrgName, "should have carrier org name")
			},
		},
		{
			id:         "ORD-020",
			name:       "includes member names",
			client:     ctx.Customer.Client,
			orderID:    orderID,
			wantStatus: http.StatusOK,
			check: func(t *testing.T, resp *client.Response[client.OrderResponse]) {
				assert.NotEmpty(t, resp.Body.CustomerMemberName, "should have customer member name")
				assert.NotEmpty(t, resp.Body.CarrierMemberName, "should have carrier member name")
			},
		},

		// Validation errors
		{
			id:         "ORD-021",
			name:       "invalid UUID",
			client:     ctx.Customer.Client,
			useRaw:     true,
			rawID:      "not-a-uuid",
			wantStatus: http.StatusBadRequest,
			wantErr:    "invalid id",
		},

		// Auth errors
		{
			id:         "ORD-022",
			name:       "without auth",
			client:     ctx.AnonClient,
			orderID:    orderID,
			wantStatus: http.StatusUnauthorized,
		},

		// Access errors
		{
			id:         "ORD-023",
			name:       "non-participant org",
			client:     otherOrg.Client,
			orderID:    orderID,
			wantStatus: http.StatusForbidden,
			wantErr:    "access denied",
		},

		// Not found
		{
			id:         "ORD-024",
			name:       "nonexistent order",
			client:     ctx.Customer.Client,
			orderID:    uuid.New(),
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.id+"_"+tt.name, func(t *testing.T) {
			if tt.useRaw {
				status, body, err := tt.client.Raw(http.MethodGet, "/api/v1/orders/"+tt.rawID, nil, nil)
				require.NoError(t, err)
				require.Equal(t, tt.wantStatus, status)
				if tt.wantErr != "" {
					assert.Contains(t, strings.ToLower(string(body)), strings.ToLower(tt.wantErr))
				}
				return
			}

			resp, err := tt.client.GetOrder(tt.orderID)
			require.NoError(t, err)
			require.Equal(t, tt.wantStatus, resp.StatusCode, string(resp.RawBody))

			if tt.wantErr != "" {
				assert.Contains(t, strings.ToLower(string(resp.RawBody)), strings.ToLower(tt.wantErr))
			}

			if tt.check != nil && resp.StatusCode == http.StatusOK {
				tt.check(t, resp)
			}
		})
	}
}

// TestSendMessage tests POST /api/v1/orders/{id}/messages
func TestSendMessage(t *testing.T) {
	t.Parallel()
	s := getSuite(t)
	ctx := fixtures.NewTestContext(t, s.BaseURL)

	// Create a confirmed order
	_, _, orderID := ctx.CreateConfirmedOrder()

	otherOrg := ctx.QuickCustomer()

	tests := []struct {
		id         string
		name       string
		client     *client.Client
		orderID    uuid.UUID
		content    string
		wantStatus int
		wantErr    string
	}{
		// Happy path
		{
			id:         "ORD-025",
			name:       "message from customer",
			client:     ctx.Customer.Client,
			orderID:    orderID,
			content:    "Hello from customer",
			wantStatus: http.StatusNoContent,
		},
		{
			id:         "ORD-026",
			name:       "message from carrier",
			client:     ctx.Carrier.Client,
			orderID:    orderID,
			content:    "Hello from carrier",
			wantStatus: http.StatusNoContent,
		},

		// Validation errors
		{
			id:         "ORD-027",
			name:       "empty message",
			client:     ctx.Customer.Client,
			orderID:    orderID,
			content:    "",
			wantStatus: http.StatusBadRequest,
			wantErr:    "message content is empty",
		},

		// Auth errors
		{
			id:         "ORD-029",
			name:       "without auth",
			client:     ctx.AnonClient,
			orderID:    orderID,
			content:    "Test",
			wantStatus: http.StatusUnauthorized,
		},

		// Access errors
		{
			id:         "ORD-030",
			name:       "non-participant",
			client:     otherOrg.Client,
			orderID:    orderID,
			content:    "Test",
			wantStatus: http.StatusForbidden,
			wantErr:    "not an order participant",
		},
	}

	for _, tt := range tests {
		t.Run(tt.id+"_"+tt.name, func(t *testing.T) {
			resp, err := tt.client.SendMessage(tt.orderID, tt.content)
			require.NoError(t, err)
			require.Equal(t, tt.wantStatus, resp.StatusCode, string(resp.RawBody))

			if tt.wantErr != "" {
				assert.Contains(t, strings.ToLower(string(resp.RawBody)), strings.ToLower(tt.wantErr))
			}
		})
	}
}

// OrderActionSuite is a testify suite for order action tests (complete, cancel, review).
// Uses suite pattern to properly handle test context with subtests.
type OrderActionSuite struct {
	suite.Suite
	baseURL  string
	ctx      *fixtures.TestContext
	otherOrg *fixtures.CreatedOrganization
}

func TestOrderActionSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(OrderActionSuite))
}

func (s *OrderActionSuite) SetupSuite() {
	testSuite := getSuite(s.T())
	s.baseURL = testSuite.BaseURL
	// Create context once for all tests
	s.ctx = fixtures.NewTestContext(s.T(), s.baseURL)
	s.otherOrg = s.ctx.QuickCustomer()
}

// createConfirmedOrder creates an order using the current test's T
func (s *OrderActionSuite) createConfirmedOrder() (*fixtures.CreatedFreightRequest, *fixtures.CreatedOffer, uuid.UUID) {
	t := s.T()
	t.Helper()

	fr, offer := s.ctx.CreateFreightWithOffer()

	// Customer selects offer
	resp, err := s.ctx.Customer.Client.SelectOffer(fr.ID, offer.OfferID)
	s.Require().NoError(err)
	s.Require().Equal(204, resp.StatusCode, string(resp.RawBody))

	// Carrier confirms offer
	resp, err = s.ctx.Carrier.Client.ConfirmOffer(fr.ID, offer.OfferID)
	s.Require().NoError(err)
	s.Require().Equal(204, resp.StatusCode, string(resp.RawBody))

	// Wait for order with proper T
	orderID := s.waitForOrder(fr.ID)

	return fr, offer, orderID
}

func (s *OrderActionSuite) waitForOrder(frID uuid.UUID) uuid.UUID {
	s.T().Helper()

	for range 300 {
		ordersResp, err := s.ctx.Customer.Client.GetOrders(map[string]string{
			"freight_request_id": frID.String(),
		})
		if err == nil && ordersResp.StatusCode == 200 && len(ordersResp.Body) > 0 {
			return ordersResp.Body[0].ID
		}
		time.Sleep(100 * time.Millisecond)
	}

	s.T().Fatalf("order was not created for freight request %s", frID)
	return uuid.Nil
}

func (s *OrderActionSuite) createCompletedOrder() uuid.UUID {
	_, _, orderID := s.createConfirmedOrder()

	s.ctx.Customer.Client.CompleteOrder(orderID)
	s.ctx.Carrier.Client.CompleteOrder(orderID)

	time.Sleep(100 * time.Millisecond)
	return orderID
}

// Complete order tests

func (s *OrderActionSuite) TestORD056_CompleteByCustomer() {
	_, _, orderID := s.createConfirmedOrder()

	resp, err := s.ctx.Customer.Client.CompleteOrder(orderID)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusNoContent, resp.StatusCode, string(resp.RawBody))
}

func (s *OrderActionSuite) TestORD057_CompleteByCarrier() {
	_, _, orderID := s.createConfirmedOrder()

	resp, err := s.ctx.Carrier.Client.CompleteOrder(orderID)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusNoContent, resp.StatusCode, string(resp.RawBody))
}

func (s *OrderActionSuite) TestORD058_FullCompletion() {
	_, _, orderID := s.createConfirmedOrder()

	// Customer completes
	resp1, err := s.ctx.Customer.Client.CompleteOrder(orderID)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusNoContent, resp1.StatusCode, string(resp1.RawBody))

	// Carrier completes
	resp2, err := s.ctx.Carrier.Client.CompleteOrder(orderID)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusNoContent, resp2.StatusCode, string(resp2.RawBody))

	// Check order status is completed
	time.Sleep(100 * time.Millisecond)
	orderResp, err := s.ctx.Customer.Client.GetOrder(orderID)
	s.Require().NoError(err)
	s.Assert().Equal("completed", orderResp.Body.Status)
}

func (s *OrderActionSuite) TestORD059_CompleteWithoutAuth() {
	_, _, orderID := s.createConfirmedOrder()

	resp, err := s.ctx.AnonClient.CompleteOrder(orderID)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusUnauthorized, resp.StatusCode)
}

func (s *OrderActionSuite) TestORD060_CompleteNonParticipant() {
	_, _, orderID := s.createConfirmedOrder()

	resp, err := s.otherOrg.Client.CompleteOrder(orderID)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusForbidden, resp.StatusCode)
}

func (s *OrderActionSuite) TestORD062_AlreadyCompletedBySide() {
	_, _, orderID := s.createConfirmedOrder()

	// Complete first time
	resp1, err := s.ctx.Customer.Client.CompleteOrder(orderID)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusNoContent, resp1.StatusCode, string(resp1.RawBody))

	// Try to complete again by same side
	resp2, err := s.ctx.Customer.Client.CompleteOrder(orderID)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusConflict, resp2.StatusCode)
	s.Assert().Contains(strings.ToLower(string(resp2.RawBody)), "already")
}

// Cancel order tests

func (s *OrderActionSuite) TestORD064_CancelByCustomer() {
	_, _, orderID := s.createConfirmedOrder()

	resp, err := s.ctx.Customer.Client.CancelOrder(orderID, helpers.StringPtr("Changed my mind"))
	s.Require().NoError(err)
	s.Require().Equal(http.StatusNoContent, resp.StatusCode, string(resp.RawBody))
}

func (s *OrderActionSuite) TestORD065_CancelByCarrier() {
	_, _, orderID := s.createConfirmedOrder()

	resp, err := s.ctx.Carrier.Client.CancelOrder(orderID, helpers.StringPtr("Cannot fulfill"))
	s.Require().NoError(err)
	s.Require().Equal(http.StatusNoContent, resp.StatusCode, string(resp.RawBody))
}

func (s *OrderActionSuite) TestORD066_CancelWithReason() {
	_, _, orderID := s.createConfirmedOrder()

	reason := "Order cancelled due to pricing issue"
	resp, err := s.ctx.Customer.Client.CancelOrder(orderID, &reason)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusNoContent, resp.StatusCode, string(resp.RawBody))
}

func (s *OrderActionSuite) TestORD067_CancelWithoutAuth() {
	_, _, orderID := s.createConfirmedOrder()

	resp, err := s.ctx.AnonClient.CancelOrder(orderID, nil)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusUnauthorized, resp.StatusCode)
}

func (s *OrderActionSuite) TestORD068_CancelNonParticipant() {
	_, _, orderID := s.createConfirmedOrder()

	resp, err := s.otherOrg.Client.CancelOrder(orderID, nil)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusForbidden, resp.StatusCode)
}

func (s *OrderActionSuite) TestORD069_AlreadyCancelled() {
	_, _, orderID := s.createConfirmedOrder()

	// Cancel first time
	resp1, err := s.ctx.Customer.Client.CancelOrder(orderID, nil)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusNoContent, resp1.StatusCode, string(resp1.RawBody))

	// Try to cancel again
	resp2, err := s.ctx.Customer.Client.CancelOrder(orderID, nil)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusConflict, resp2.StatusCode)
}

func (s *OrderActionSuite) TestORD070_CancelAfterCompletionStarted() {
	_, _, orderID := s.createConfirmedOrder()

	// Customer completes
	resp1, err := s.ctx.Customer.Client.CompleteOrder(orderID)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusNoContent, resp1.StatusCode, string(resp1.RawBody))

	// Try to cancel after completion started
	resp2, err := s.ctx.Carrier.Client.CancelOrder(orderID, nil)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusConflict, resp2.StatusCode)
	s.Assert().Contains(strings.ToLower(string(resp2.RawBody)), "completion")
}

// Review tests

func (s *OrderActionSuite) TestORD071_ReviewFromCustomer() {
	orderID := s.createCompletedOrder()

	resp, err := s.ctx.Customer.Client.LeaveReview(orderID, 5, helpers.StringPtr("Great service!"))
	s.Require().NoError(err)
	s.Require().Equal(http.StatusNoContent, resp.StatusCode, string(resp.RawBody))
}

func (s *OrderActionSuite) TestORD072_ReviewFromCarrier() {
	orderID := s.createCompletedOrder()

	resp, err := s.ctx.Carrier.Client.LeaveReview(orderID, 4, helpers.StringPtr("Good customer"))
	s.Require().NoError(err)
	s.Require().Equal(http.StatusNoContent, resp.StatusCode, string(resp.RawBody))
}

func (s *OrderActionSuite) TestORD073_Rating1To5() {
	for rating := 1; rating <= 5; rating++ {
		s.Run(string(rune('0'+rating)), func() {
			orderID := s.createCompletedOrder()

			resp, err := s.ctx.Customer.Client.LeaveReview(orderID, rating, nil)
			s.Require().NoError(err)
			s.Require().Equal(http.StatusNoContent, resp.StatusCode, "rating %d should be valid", rating)
		})
	}
}

func (s *OrderActionSuite) TestORD075_Rating0() {
	orderID := s.createCompletedOrder()

	resp, err := s.ctx.Customer.Client.LeaveReview(orderID, 0, nil)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusBadRequest, resp.StatusCode)
	s.Assert().Contains(strings.ToLower(string(resp.RawBody)), "rating")
}

func (s *OrderActionSuite) TestORD076_Rating6() {
	orderID := s.createCompletedOrder()

	resp, err := s.ctx.Customer.Client.LeaveReview(orderID, 6, nil)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusBadRequest, resp.StatusCode)
	s.Assert().Contains(strings.ToLower(string(resp.RawBody)), "rating")
}

func (s *OrderActionSuite) TestORD078_ReviewWithoutAuth() {
	orderID := s.createCompletedOrder()

	resp, err := s.ctx.AnonClient.LeaveReview(orderID, 5, nil)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusUnauthorized, resp.StatusCode)
}

func (s *OrderActionSuite) TestORD079_ReviewNonParticipant() {
	orderID := s.createCompletedOrder()

	resp, err := s.otherOrg.Client.LeaveReview(orderID, 5, nil)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusForbidden, resp.StatusCode)
}

func (s *OrderActionSuite) TestORD080_ReviewNotCompletedOrder() {
	_, _, orderID := s.createConfirmedOrder() // Active order

	resp, err := s.ctx.Customer.Client.LeaveReview(orderID, 5, nil)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusConflict, resp.StatusCode)
	s.Assert().Contains(strings.ToLower(string(resp.RawBody)), "finished")
}

func (s *OrderActionSuite) TestORD081_AlreadyReviewed() {
	orderID := s.createCompletedOrder()

	// First review
	resp1, err := s.ctx.Customer.Client.LeaveReview(orderID, 5, nil)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusNoContent, resp1.StatusCode, string(resp1.RawBody))

	// Second review
	resp2, err := s.ctx.Customer.Client.LeaveReview(orderID, 4, nil)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusConflict, resp2.StatusCode)
	s.Assert().Contains(strings.ToLower(string(resp2.RawBody)), "already")
}

// TestMessageOnCancelledOrder tests ORD-031: cannot message on cancelled order
func TestMessageOnCancelledOrder(t *testing.T) {
	t.Parallel()
	s := getSuite(t)
	ctx := fixtures.NewTestContext(t, s.BaseURL)

	_, _, orderID := ctx.CreateConfirmedOrder()

	// Cancel the order
	resp1, err := ctx.Customer.Client.CancelOrder(orderID, nil)
	require.NoError(t, err)
	require.Equal(t, http.StatusNoContent, resp1.StatusCode, string(resp1.RawBody))

	// Try to send message
	resp2, err := ctx.Customer.Client.SendMessage(orderID, "Test message")
	require.NoError(t, err)
	require.Equal(t, http.StatusConflict, resp2.StatusCode)
	assert.Contains(t, strings.ToLower(string(resp2.RawBody)), "cancelled")
}

// TestMessageOnCompletedOrder tests ORD-032: cannot message on completed order
func TestMessageOnCompletedOrder(t *testing.T) {
	t.Parallel()
	s := getSuite(t)
	ctx := fixtures.NewTestContext(t, s.BaseURL)

	_, _, orderID := ctx.CreateConfirmedOrder()

	// Complete the order
	resp1, err := ctx.Customer.Client.CompleteOrder(orderID)
	require.NoError(t, err)
	require.Equal(t, http.StatusNoContent, resp1.StatusCode)

	resp2, err := ctx.Carrier.Client.CompleteOrder(orderID)
	require.NoError(t, err)
	require.Equal(t, http.StatusNoContent, resp2.StatusCode)

	// Wait for order to be in completed status
	helpers.WaitFor(t, func() (bool, bool) {
		orderResp, err := ctx.Customer.Client.GetOrder(orderID)
		if err != nil {
			return false, false
		}
		return orderResp.Body.Status == "completed", orderResp.Body.Status == "completed"
	}, "order should be completed")

	// Try to send message
	resp, err := ctx.Customer.Client.SendMessage(orderID, "Test message")
	require.NoError(t, err)
	require.Equal(t, http.StatusConflict, resp.StatusCode)
	assert.Contains(t, strings.ToLower(string(resp.RawBody)), "completed")
}

// TestOrderOnlyShowsOwnOrders tests ORD-013: filter only shows orders user's org is part of
func TestOrderOnlyShowsOwnOrders(t *testing.T) {
	t.Parallel()
	s := getSuite(t)
	ctx := fixtures.NewTestContext(t, s.BaseURL)

	// Create orders
	ctx.CreateConfirmedOrder()

	// Create another org not involved in any orders
	otherOrg := ctx.QuickCarrier()

	// Get orders for other org - should be empty
	resp, err := otherOrg.Client.GetOrders(nil)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, 0, len(resp.Body), "should not see orders from other orgs")
}
