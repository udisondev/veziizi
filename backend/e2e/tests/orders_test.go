package tests

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"codeberg.org/udison/veziizi/backend/e2e/fixtures"
	"codeberg.org/udison/veziizi/backend/e2e/helpers"
	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
)

// OrdersSuite combines all order-related tests with shared context.
// Organizations are created once in SetupSuite and reused across tests.
type OrdersSuite struct {
	suite.Suite
	baseURL  string
	ctx      *fixtures.TestContext
	otherOrg *fixtures.CreatedOrganization

	// Shared order for read-only tests
	sharedOrderID uuid.UUID
	sharedFR      *fixtures.CreatedFreightRequest

	// Shared completed order for review rejection tests (auth/access tests that don't leave reviews)
	sharedCompletedOrderID uuid.UUID
}

func TestOrdersSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(OrdersSuite))
}

func (s *OrdersSuite) SetupSuite() {
	testSuite := getSuite(s.T())
	s.baseURL = testSuite.BaseURL

	// Create context with Customer and Carrier orgs - done ONCE for all tests
	s.ctx = fixtures.NewTestContext(s.T(), s.baseURL)
	s.otherOrg = s.ctx.QuickCustomer()

	// Create a shared active order for read-only tests
	s.sharedFR, _, s.sharedOrderID = s.ctx.CreateConfirmedOrder()

	// Create a shared completed order for review rejection tests
	_, _, s.sharedCompletedOrderID = s.ctx.CreateConfirmedOrder()
	s.ctx.Customer.Client.CompleteOrder(s.sharedCompletedOrderID)
	s.ctx.Carrier.Client.CompleteOrder(s.sharedCompletedOrderID)
	time.Sleep(50 * time.Millisecond)
}

// ==================== GET /api/v1/orders ====================

func (s *OrdersSuite) TestORD001_ListOrders() {
	resp, err := s.ctx.Customer.Client.GetOrders(nil)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusOK, resp.StatusCode, string(resp.RawBody))
	s.Assert().True(len(resp.Body) >= 1, "should have at least 1 order")
}

func (s *OrdersSuite) TestORD003_FilterByCustomerOrgID() {
	resp, err := s.ctx.Customer.Client.GetOrders(map[string]string{
		"customer_org_id": s.ctx.Customer.OrganizationID.String(),
	})
	s.Require().NoError(err)
	s.Require().Equal(http.StatusOK, resp.StatusCode, string(resp.RawBody))

	for _, order := range resp.Body {
		s.Assert().Equal(s.ctx.Customer.OrganizationID, order.CustomerOrgID)
	}
}

func (s *OrdersSuite) TestORD004_FilterByCarrierOrgID() {
	resp, err := s.ctx.Carrier.Client.GetOrders(map[string]string{
		"carrier_org_id": s.ctx.Carrier.OrganizationID.String(),
	})
	s.Require().NoError(err)
	s.Require().Equal(http.StatusOK, resp.StatusCode, string(resp.RawBody))

	for _, order := range resp.Body {
		s.Assert().Equal(s.ctx.Carrier.OrganizationID, order.CarrierOrgID)
	}
}

func (s *OrdersSuite) TestORD005_FilterByFreightRequestID() {
	resp, err := s.ctx.Customer.Client.GetOrders(map[string]string{
		"freight_request_id": s.sharedFR.ID.String(),
	})
	s.Require().NoError(err)
	s.Require().Equal(http.StatusOK, resp.StatusCode, string(resp.RawBody))
	s.Assert().Equal(1, len(resp.Body), "should have exactly 1 order")
	s.Assert().Equal(s.sharedFR.ID, resp.Body[0].FreightRequestID)
}

func (s *OrdersSuite) TestORD006_FilterByStatus() {
	resp, err := s.ctx.Customer.Client.GetOrders(map[string]string{
		"status": "active",
	})
	s.Require().NoError(err)
	s.Require().Equal(http.StatusOK, resp.StatusCode, string(resp.RawBody))

	for _, order := range resp.Body {
		s.Assert().Equal("active", order.Status)
	}
}

func (s *OrdersSuite) TestORD007_Pagination() {
	resp, err := s.ctx.Customer.Client.GetOrders(map[string]string{
		"limit": "1", "offset": "0",
	})
	s.Require().NoError(err)
	s.Require().Equal(http.StatusOK, resp.StatusCode, string(resp.RawBody))
	s.Assert().LessOrEqual(len(resp.Body), 1, "should respect limit")
}

func (s *OrdersSuite) TestORD008_InvalidCustomerOrgID() {
	resp, err := s.ctx.Customer.Client.GetOrders(map[string]string{
		"customer_org_id": "not-a-uuid",
	})
	s.Require().NoError(err)
	s.Require().Equal(http.StatusBadRequest, resp.StatusCode)
	s.Assert().Contains(strings.ToLower(string(resp.RawBody)), "invalid customer_org_id")
}

func (s *OrdersSuite) TestORD009_InvalidLimit() {
	resp, err := s.ctx.Customer.Client.GetOrders(map[string]string{"limit": "0"})
	s.Require().NoError(err)
	s.Require().Equal(http.StatusBadRequest, resp.StatusCode)
	s.Assert().Contains(strings.ToLower(string(resp.RawBody)), "invalid limit")
}

func (s *OrdersSuite) TestORD010_InvalidOffset() {
	resp, err := s.ctx.Customer.Client.GetOrders(map[string]string{"offset": "-1"})
	s.Require().NoError(err)
	s.Require().Equal(http.StatusBadRequest, resp.StatusCode)
	s.Assert().Contains(strings.ToLower(string(resp.RawBody)), "invalid offset")
}

func (s *OrdersSuite) TestORD011_LimitTooLarge() {
	resp, err := s.ctx.Customer.Client.GetOrders(map[string]string{"limit": "200"})
	s.Require().NoError(err)
	s.Require().Equal(http.StatusBadRequest, resp.StatusCode)
	s.Assert().Contains(strings.ToLower(string(resp.RawBody)), "invalid limit")
}

func (s *OrdersSuite) TestORD012_WithoutAuth() {
	resp, err := s.ctx.AnonClient.GetOrders(nil)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusUnauthorized, resp.StatusCode)
}

func (s *OrdersSuite) TestORD013_OnlyShowsOwnOrders() {
	// Other org should not see orders from Customer/Carrier
	resp, err := s.otherOrg.Client.GetOrders(nil)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusOK, resp.StatusCode)
	s.Assert().Equal(0, len(resp.Body), "should not see orders from other orgs")
}

// ==================== GET /api/v1/orders/{id} ====================

func (s *OrdersSuite) TestORD014_GetAsCustomer() {
	resp, err := s.ctx.Customer.Client.GetOrder(s.sharedOrderID)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusOK, resp.StatusCode, string(resp.RawBody))
	s.Assert().Equal(s.sharedOrderID, resp.Body.ID)
	s.Assert().Equal(s.ctx.Customer.OrganizationID, resp.Body.CustomerOrgID)
}

func (s *OrdersSuite) TestORD015_GetAsCarrier() {
	resp, err := s.ctx.Carrier.Client.GetOrder(s.sharedOrderID)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusOK, resp.StatusCode, string(resp.RawBody))
	s.Assert().Equal(s.sharedOrderID, resp.Body.ID)
	s.Assert().Equal(s.ctx.Carrier.OrganizationID, resp.Body.CarrierOrgID)
}

func (s *OrdersSuite) TestORD019_IncludesOrgNames() {
	resp, err := s.ctx.Customer.Client.GetOrder(s.sharedOrderID)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusOK, resp.StatusCode, string(resp.RawBody))
	s.Assert().NotEmpty(resp.Body.CustomerOrgName, "should have customer org name")
	s.Assert().NotEmpty(resp.Body.CarrierOrgName, "should have carrier org name")
}

func (s *OrdersSuite) TestORD020_IncludesMemberNames() {
	resp, err := s.ctx.Customer.Client.GetOrder(s.sharedOrderID)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusOK, resp.StatusCode, string(resp.RawBody))
	s.Assert().NotEmpty(resp.Body.CustomerMemberName, "should have customer member name")
	s.Assert().NotEmpty(resp.Body.CarrierMemberName, "should have carrier member name")
}

func (s *OrdersSuite) TestORD021_InvalidUUID() {
	status, body, err := s.ctx.Customer.Client.Raw(http.MethodGet, "/api/v1/orders/not-a-uuid", nil, nil)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusBadRequest, status)
	s.Assert().Contains(strings.ToLower(string(body)), "invalid id")
}

func (s *OrdersSuite) TestORD022_GetWithoutAuth() {
	resp, err := s.ctx.AnonClient.GetOrder(s.sharedOrderID)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusUnauthorized, resp.StatusCode)
}

func (s *OrdersSuite) TestORD023_NonParticipantOrg() {
	resp, err := s.otherOrg.Client.GetOrder(s.sharedOrderID)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusForbidden, resp.StatusCode)
	s.Assert().Contains(strings.ToLower(string(resp.RawBody)), "access denied")
}

func (s *OrdersSuite) TestORD024_NonexistentOrder() {
	resp, err := s.ctx.Customer.Client.GetOrder(uuid.New())
	s.Require().NoError(err)
	s.Require().Equal(http.StatusNotFound, resp.StatusCode)
}

// ==================== POST /api/v1/orders/{id}/messages ====================

func (s *OrdersSuite) TestORD025_MessageFromCustomer() {
	resp, err := s.ctx.Customer.Client.SendMessage(s.sharedOrderID, "Hello from customer")
	s.Require().NoError(err)
	s.Require().Equal(http.StatusNoContent, resp.StatusCode, string(resp.RawBody))
}

func (s *OrdersSuite) TestORD026_MessageFromCarrier() {
	resp, err := s.ctx.Carrier.Client.SendMessage(s.sharedOrderID, "Hello from carrier")
	s.Require().NoError(err)
	s.Require().Equal(http.StatusNoContent, resp.StatusCode, string(resp.RawBody))
}

func (s *OrdersSuite) TestORD027_EmptyMessage() {
	resp, err := s.ctx.Customer.Client.SendMessage(s.sharedOrderID, "")
	s.Require().NoError(err)
	s.Require().Equal(http.StatusBadRequest, resp.StatusCode)
	s.Assert().Contains(strings.ToLower(string(resp.RawBody)), "message content is empty")
}

func (s *OrdersSuite) TestORD029_MessageWithoutAuth() {
	resp, err := s.ctx.AnonClient.SendMessage(s.sharedOrderID, "Test")
	s.Require().NoError(err)
	s.Require().Equal(http.StatusUnauthorized, resp.StatusCode)
}

func (s *OrdersSuite) TestORD030_MessageNonParticipant() {
	resp, err := s.otherOrg.Client.SendMessage(s.sharedOrderID, "Test")
	s.Require().NoError(err)
	s.Require().Equal(http.StatusForbidden, resp.StatusCode)
	s.Assert().Contains(strings.ToLower(string(resp.RawBody)), "not an order participant")
}

func (s *OrdersSuite) TestORD031_MessageOnCancelledOrder() {
	_, _, orderID := s.ctx.CreateConfirmedOrder()

	// Cancel the order
	resp1, err := s.ctx.Customer.Client.CancelOrder(orderID, nil)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusNoContent, resp1.StatusCode, string(resp1.RawBody))

	// Try to send message
	resp2, err := s.ctx.Customer.Client.SendMessage(orderID, "Test message")
	s.Require().NoError(err)
	s.Require().Equal(http.StatusConflict, resp2.StatusCode)
	s.Assert().Contains(strings.ToLower(string(resp2.RawBody)), "cancelled")
}

func (s *OrdersSuite) TestORD032_MessageOnCompletedOrder() {
	_, _, orderID := s.ctx.CreateConfirmedOrder()

	// Complete the order
	resp1, err := s.ctx.Customer.Client.CompleteOrder(orderID)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusNoContent, resp1.StatusCode)

	resp2, err := s.ctx.Carrier.Client.CompleteOrder(orderID)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusNoContent, resp2.StatusCode)

	// Wait for order to be in completed status
	helpers.WaitFor(s.T(), func() (bool, bool) {
		orderResp, err := s.ctx.Customer.Client.GetOrder(orderID)
		if err != nil {
			return false, false
		}
		return orderResp.Body.Status == "completed", orderResp.Body.Status == "completed"
	}, "order should be completed")

	// Try to send message
	resp, err := s.ctx.Customer.Client.SendMessage(orderID, "Test message")
	s.Require().NoError(err)
	s.Require().Equal(http.StatusConflict, resp.StatusCode)
	s.Assert().Contains(strings.ToLower(string(resp.RawBody)), "completed")
}

// ==================== POST /api/v1/orders/{id}/complete ====================

func (s *OrdersSuite) TestORD056_CompleteByCustomer() {
	_, _, orderID := s.ctx.CreateConfirmedOrder()

	resp, err := s.ctx.Customer.Client.CompleteOrder(orderID)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusNoContent, resp.StatusCode, string(resp.RawBody))
}

func (s *OrdersSuite) TestORD057_CompleteByCarrier() {
	_, _, orderID := s.ctx.CreateConfirmedOrder()

	resp, err := s.ctx.Carrier.Client.CompleteOrder(orderID)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusNoContent, resp.StatusCode, string(resp.RawBody))
}

func (s *OrdersSuite) TestORD058_FullCompletion() {
	_, _, orderID := s.ctx.CreateConfirmedOrder()

	// Customer completes
	resp1, err := s.ctx.Customer.Client.CompleteOrder(orderID)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusNoContent, resp1.StatusCode, string(resp1.RawBody))

	// Carrier completes
	resp2, err := s.ctx.Carrier.Client.CompleteOrder(orderID)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusNoContent, resp2.StatusCode, string(resp2.RawBody))

	// Check order status is completed
	time.Sleep(50 * time.Millisecond)
	orderResp, err := s.ctx.Customer.Client.GetOrder(orderID)
	s.Require().NoError(err)
	s.Assert().Equal("completed", orderResp.Body.Status)
}

func (s *OrdersSuite) TestORD059_CompleteWithoutAuth() {
	// Use shared order - operation is rejected anyway
	resp, err := s.ctx.AnonClient.CompleteOrder(s.sharedOrderID)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusUnauthorized, resp.StatusCode)
}

func (s *OrdersSuite) TestORD060_CompleteNonParticipant() {
	// Use shared order - operation is rejected anyway
	resp, err := s.otherOrg.Client.CompleteOrder(s.sharedOrderID)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusForbidden, resp.StatusCode)
}

func (s *OrdersSuite) TestORD062_AlreadyCompletedBySide() {
	_, _, orderID := s.ctx.CreateConfirmedOrder()

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

// ==================== POST /api/v1/orders/{id}/cancel ====================

func (s *OrdersSuite) TestORD064_CancelByCustomer() {
	_, _, orderID := s.ctx.CreateConfirmedOrder()

	resp, err := s.ctx.Customer.Client.CancelOrder(orderID, helpers.StringPtr("Changed my mind"))
	s.Require().NoError(err)
	s.Require().Equal(http.StatusNoContent, resp.StatusCode, string(resp.RawBody))
}

func (s *OrdersSuite) TestORD065_CancelByCarrier() {
	_, _, orderID := s.ctx.CreateConfirmedOrder()

	resp, err := s.ctx.Carrier.Client.CancelOrder(orderID, helpers.StringPtr("Cannot fulfill"))
	s.Require().NoError(err)
	s.Require().Equal(http.StatusNoContent, resp.StatusCode, string(resp.RawBody))
}

func (s *OrdersSuite) TestORD066_CancelWithReason() {
	_, _, orderID := s.ctx.CreateConfirmedOrder()

	reason := "Order cancelled due to pricing issue"
	resp, err := s.ctx.Customer.Client.CancelOrder(orderID, &reason)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusNoContent, resp.StatusCode, string(resp.RawBody))
}

func (s *OrdersSuite) TestORD067_CancelWithoutAuth() {
	// Use shared order - operation is rejected anyway
	resp, err := s.ctx.AnonClient.CancelOrder(s.sharedOrderID, nil)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusUnauthorized, resp.StatusCode)
}

func (s *OrdersSuite) TestORD068_CancelNonParticipant() {
	// Use shared order - operation is rejected anyway
	resp, err := s.otherOrg.Client.CancelOrder(s.sharedOrderID, nil)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusForbidden, resp.StatusCode)
}

func (s *OrdersSuite) TestORD069_AlreadyCancelled() {
	_, _, orderID := s.ctx.CreateConfirmedOrder()

	// Cancel first time
	resp1, err := s.ctx.Customer.Client.CancelOrder(orderID, nil)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusNoContent, resp1.StatusCode, string(resp1.RawBody))

	// Try to cancel again
	resp2, err := s.ctx.Customer.Client.CancelOrder(orderID, nil)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusConflict, resp2.StatusCode)
}

func (s *OrdersSuite) TestORD070_CancelAfterCompletionStarted() {
	_, _, orderID := s.ctx.CreateConfirmedOrder()

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

// ==================== POST /api/v1/orders/{id}/review ====================

func (s *OrdersSuite) createCompletedOrder() uuid.UUID {
	_, _, orderID := s.ctx.CreateConfirmedOrder()

	s.ctx.Customer.Client.CompleteOrder(orderID)
	s.ctx.Carrier.Client.CompleteOrder(orderID)

	time.Sleep(50 * time.Millisecond)
	return orderID
}

func (s *OrdersSuite) TestORD071_ReviewFromCustomer() {
	orderID := s.createCompletedOrder()

	resp, err := s.ctx.Customer.Client.LeaveReview(orderID, 5, helpers.StringPtr("Great service!"))
	s.Require().NoError(err)
	s.Require().Equal(http.StatusNoContent, resp.StatusCode, string(resp.RawBody))
}

func (s *OrdersSuite) TestORD072_ReviewFromCarrier() {
	orderID := s.createCompletedOrder()

	resp, err := s.ctx.Carrier.Client.LeaveReview(orderID, 4, helpers.StringPtr("Good customer"))
	s.Require().NoError(err)
	s.Require().Equal(http.StatusNoContent, resp.StatusCode, string(resp.RawBody))
}

func (s *OrdersSuite) TestORD073_Rating1To5() {
	for rating := 1; rating <= 5; rating++ {
		s.Run(string(rune('0'+rating)), func() {
			orderID := s.createCompletedOrder()

			resp, err := s.ctx.Customer.Client.LeaveReview(orderID, rating, nil)
			s.Require().NoError(err)
			s.Require().Equal(http.StatusNoContent, resp.StatusCode, "rating %d should be valid", rating)
		})
	}
}

func (s *OrdersSuite) TestORD075_Rating0() {
	orderID := s.createCompletedOrder()

	resp, err := s.ctx.Customer.Client.LeaveReview(orderID, 0, nil)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusBadRequest, resp.StatusCode)
	s.Assert().Contains(strings.ToLower(string(resp.RawBody)), "rating")
}

func (s *OrdersSuite) TestORD076_Rating6() {
	orderID := s.createCompletedOrder()

	resp, err := s.ctx.Customer.Client.LeaveReview(orderID, 6, nil)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusBadRequest, resp.StatusCode)
	s.Assert().Contains(strings.ToLower(string(resp.RawBody)), "rating")
}

func (s *OrdersSuite) TestORD078_ReviewWithoutAuth() {
	// Use shared completed order - operation is rejected anyway
	resp, err := s.ctx.AnonClient.LeaveReview(s.sharedCompletedOrderID, 5, nil)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusUnauthorized, resp.StatusCode)
}

func (s *OrdersSuite) TestORD079_ReviewNonParticipant() {
	// Use shared completed order - operation is rejected anyway
	resp, err := s.otherOrg.Client.LeaveReview(s.sharedCompletedOrderID, 5, nil)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusForbidden, resp.StatusCode)
}

func (s *OrdersSuite) TestORD080_ReviewNotCompletedOrder() {
	// Use shared active order - it's not completed so review should fail
	resp, err := s.ctx.Customer.Client.LeaveReview(s.sharedOrderID, 5, nil)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusConflict, resp.StatusCode)
	s.Assert().Contains(strings.ToLower(string(resp.RawBody)), "finished")
}

func (s *OrdersSuite) TestORD081_AlreadyReviewed() {
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
