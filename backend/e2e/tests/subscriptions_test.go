package tests

import (
	"net/http"
	"testing"

	"codeberg.org/udison/veziizi/backend/e2e/client"
	"codeberg.org/udison/veziizi/backend/e2e/fixtures"
	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
)

// SubscriptionsSuite combines all subscriptions tests with shared context.
type SubscriptionsSuite struct {
	suite.Suite
	baseURL string
	ctx     *fixtures.TestContext

	// Other organization for access tests
	otherOrg *fixtures.CreatedOrganization

	// Shared subscription for tests
	sharedSubID uuid.UUID
}

func TestSubscriptionsSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(SubscriptionsSuite))
}

func (s *SubscriptionsSuite) SetupSuite() {
	testSuite := getSuite(s.T())
	s.baseURL = testSuite.BaseURL
	s.ctx = fixtures.NewTestContext(s.T(), s.baseURL)
	s.otherOrg = s.ctx.QuickCustomer()

	// Create a shared subscription
	subResp, err := s.ctx.Customer.Client.CreateSubscription(client.CreateSubscriptionRequest{
		Name:     "Shared Test Subscription",
		IsActive: true,
	})
	s.Require().NoError(err)
	s.Require().Equal(http.StatusCreated, subResp.StatusCode)
	s.sharedSubID = subResp.Body.ID
}

// ==================== GET /api/v1/subscriptions ====================

func (s *SubscriptionsSuite) TestSUB001_ListSubscriptions() {
	resp, err := s.ctx.Customer.Client.GetSubscriptions()
	s.Require().NoError(err)
	s.Require().Equal(http.StatusOK, resp.StatusCode, string(resp.RawBody))
}

func (s *SubscriptionsSuite) TestSUB002_EmptyList() {
	// Create a new user who has no subscriptions
	newOrg := s.ctx.QuickCustomer()

	resp, err := newOrg.Client.GetSubscriptions()
	s.Require().NoError(err)
	s.Require().Equal(http.StatusOK, resp.StatusCode)
	s.Assert().Empty(resp.Body)
}

func (s *SubscriptionsSuite) TestSUB003_WithoutAuth() {
	resp, err := s.ctx.AnonClient.GetSubscriptions()
	s.Require().NoError(err)
	s.Require().Equal(http.StatusUnauthorized, resp.StatusCode)
}

// ==================== POST /api/v1/subscriptions ====================

func (s *SubscriptionsSuite) TestSUB004_CreateSubscription() {
	resp, err := s.ctx.Customer.Client.CreateSubscription(client.CreateSubscriptionRequest{
		Name:     "My Subscription",
		IsActive: true,
	})
	s.Require().NoError(err)
	s.Require().Equal(http.StatusCreated, resp.StatusCode, string(resp.RawBody))
	s.Assert().NotEqual(uuid.Nil, resp.Body.ID)
	s.Assert().Equal("My Subscription", resp.Body.Name)
	s.Assert().True(resp.Body.IsActive)
}

func (s *SubscriptionsSuite) TestSUB005_WithFilters() {
	minWeight := 100.0
	maxWeight := 500.0
	resp, err := s.ctx.Customer.Client.CreateSubscription(client.CreateSubscriptionRequest{
		Name:         "Filtered Subscription",
		MinWeight:    &minWeight,
		MaxWeight:    &maxWeight,
		VehicleTypes: []string{"tent"},
		IsActive:     true,
	})
	s.Require().NoError(err)
	s.Require().Equal(http.StatusCreated, resp.StatusCode, string(resp.RawBody))
	s.Assert().NotNil(resp.Body.MinWeight)
	s.Assert().NotNil(resp.Body.MaxWeight)
	s.Assert().Contains(resp.Body.VehicleTypes, "tent")
}

func (s *SubscriptionsSuite) TestSUB006_EmptyName() {
	resp, err := s.ctx.Customer.Client.CreateSubscription(client.CreateSubscriptionRequest{
		Name:     "",
		IsActive: true,
	})
	s.Require().NoError(err)
	s.Require().Equal(http.StatusBadRequest, resp.StatusCode)
}

// ==================== GET /api/v1/subscriptions/{id} ====================

func (s *SubscriptionsSuite) TestSUB007_GetOwnSubscription() {
	resp, err := s.ctx.Customer.Client.GetSubscription(s.sharedSubID)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusOK, resp.StatusCode, string(resp.RawBody))
	s.Assert().Equal(s.sharedSubID, resp.Body.ID)
}

func (s *SubscriptionsSuite) TestSUB008_OtherOrgsSubscription() {
	resp, err := s.otherOrg.Client.GetSubscription(s.sharedSubID)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusForbidden, resp.StatusCode)
}

func (s *SubscriptionsSuite) TestSUB009_NonexistentSubscription() {
	resp, err := s.ctx.Customer.Client.GetSubscription(uuid.New())
	s.Require().NoError(err)
	s.Require().Equal(http.StatusNotFound, resp.StatusCode)
}

// ==================== PUT /api/v1/subscriptions/{id} ====================

func (s *SubscriptionsSuite) TestSUB010_UpdateFilters() {
	// Create a subscription
	subResp, err := s.ctx.Customer.Client.CreateSubscription(client.CreateSubscriptionRequest{
		Name:     "Original Name",
		IsActive: true,
	})
	s.Require().NoError(err)
	s.Require().Equal(http.StatusCreated, subResp.StatusCode)

	// Update it
	minWeight := 200.0
	updateResp, err := s.ctx.Customer.Client.UpdateSubscription(subResp.Body.ID, client.CreateSubscriptionRequest{
		Name:      "Updated Name",
		MinWeight: &minWeight,
		IsActive:  true,
	})
	s.Require().NoError(err)
	s.Require().Equal(http.StatusOK, updateResp.StatusCode, string(updateResp.RawBody))
	s.Assert().Equal("Updated Name", updateResp.Body.Name)
	s.Assert().NotNil(updateResp.Body.MinWeight)
}

// ==================== DELETE /api/v1/subscriptions/{id} ====================

func (s *SubscriptionsSuite) TestSUB011_DeleteOwn() {
	// Create a subscription
	subResp, err := s.ctx.Customer.Client.CreateSubscription(client.CreateSubscriptionRequest{
		Name:     "To Delete",
		IsActive: true,
	})
	s.Require().NoError(err)
	s.Require().Equal(http.StatusCreated, subResp.StatusCode)

	// Delete it
	deleteResp, err := s.ctx.Customer.Client.DeleteSubscription(subResp.Body.ID)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusNoContent, deleteResp.StatusCode, string(deleteResp.RawBody))

	// Verify it's gone
	getResp, err := s.ctx.Customer.Client.GetSubscription(subResp.Body.ID)
	s.Require().NoError(err)
	s.Assert().Equal(http.StatusNotFound, getResp.StatusCode)
}

// ==================== PATCH /api/v1/subscriptions/{id}/active ====================

func (s *SubscriptionsSuite) TestSUB012_Activate() {
	// Create an active subscription
	subResp, err := s.ctx.Customer.Client.CreateSubscription(client.CreateSubscriptionRequest{
		Name:     "Active Subscription",
		IsActive: true,
	})
	s.Require().NoError(err)
	s.Require().Equal(http.StatusCreated, subResp.StatusCode)

	// Deactivate first
	resp, err := s.ctx.Customer.Client.SetSubscriptionActive(subResp.Body.ID, false)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusNoContent, resp.StatusCode, string(resp.RawBody))

	// Activate
	resp2, err := s.ctx.Customer.Client.SetSubscriptionActive(subResp.Body.ID, true)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusNoContent, resp2.StatusCode, string(resp2.RawBody))

	// Verify it's activated
	getResp, err := s.ctx.Customer.Client.GetSubscription(subResp.Body.ID)
	s.Require().NoError(err)
	s.Assert().True(getResp.Body.IsActive)
}

func (s *SubscriptionsSuite) TestSUB013_Deactivate() {
	// Create an active subscription
	subResp, err := s.ctx.Customer.Client.CreateSubscription(client.CreateSubscriptionRequest{
		Name:     "Deactivate Subscription",
		IsActive: true,
	})
	s.Require().NoError(err)
	s.Require().Equal(http.StatusCreated, subResp.StatusCode)
	s.Require().True(subResp.Body.IsActive)

	resp, err := s.ctx.Customer.Client.SetSubscriptionActive(subResp.Body.ID, false)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusNoContent, resp.StatusCode, string(resp.RawBody))

	// Verify it's deactivated
	getResp, err := s.ctx.Customer.Client.GetSubscription(subResp.Body.ID)
	s.Require().NoError(err)
	s.Assert().False(getResp.Body.IsActive)
}

// ==================== Without auth ====================

func (s *SubscriptionsSuite) TestCreateWithoutAuth() {
	resp, err := s.ctx.AnonClient.CreateSubscription(client.CreateSubscriptionRequest{
		Name:     "Test",
		IsActive: true,
	})
	s.Require().NoError(err)
	s.Require().Equal(http.StatusUnauthorized, resp.StatusCode)
}

func (s *SubscriptionsSuite) TestGetWithoutAuth() {
	resp, err := s.ctx.AnonClient.GetSubscription(uuid.New())
	s.Require().NoError(err)
	s.Require().Equal(http.StatusUnauthorized, resp.StatusCode)
}
