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
	"github.com/stretchr/testify/suite"
)

// FreightRequestsSuite combines all freight request tests with shared context.
type FreightRequestsSuite struct {
	suite.Suite
	baseURL string
	ctx     *fixtures.TestContext

	// Shared freight request for read-only tests
	sharedFR *fixtures.CreatedFreightRequest
}

func TestFreightRequestsSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(FreightRequestsSuite))
}

func (s *FreightRequestsSuite) SetupSuite() {
	testSuite := getSuite(s.T())
	s.baseURL = testSuite.BaseURL

	// Create context with Customer and Carrier orgs - done ONCE for all tests
	s.ctx = fixtures.NewTestContext(s.T(), s.baseURL)

	// Create a shared freight request for read-only tests
	s.sharedFR = fixtures.NewFreightRequest(s.T(), s.ctx.Customer.Client).Create()
}

// ==================== POST /api/v1/freight-requests ====================

func (s *FreightRequestsSuite) TestFR001_SuccessfulCreation() {
	builder := fixtures.NewFreightRequest(s.T(), s.ctx.Customer.Client)
	resp, err := builder.CreateWithStatus()
	s.Require().NoError(err)
	s.Require().Equal(http.StatusCreated, resp.StatusCode, string(resp.RawBody))
	s.Assert().NotEmpty(resp.Body.ID.String(), "id should be set")
}

func (s *FreightRequestsSuite) TestFR002_WithComment() {
	builder := fixtures.NewFreightRequest(s.T(), s.ctx.Customer.Client).
		WithComment("Urgent delivery needed")
	resp, err := builder.CreateWithStatus()
	s.Require().NoError(err)
	s.Require().Equal(http.StatusCreated, resp.StatusCode, string(resp.RawBody))
}

func (s *FreightRequestsSuite) TestFR003_WithCustomExpiry() {
	builder := fixtures.NewFreightRequest(s.T(), s.ctx.Customer.Client).
		ExpiresIn(14 * 24 * time.Hour)
	resp, err := builder.CreateWithStatus()
	s.Require().NoError(err)
	s.Require().Equal(http.StatusCreated, resp.StatusCode, string(resp.RawBody))
}

func (s *FreightRequestsSuite) TestFR006_NegativeWeight() {
	builder := fixtures.NewFreightRequest(s.T(), s.ctx.Customer.Client).
		WithWeight(-100)
	resp, err := builder.CreateWithStatus()
	s.Require().NoError(err)
	s.Require().Equal(http.StatusBadRequest, resp.StatusCode)
}

func (s *FreightRequestsSuite) TestFR009_NegativePrice() {
	builder := fixtures.NewFreightRequest(s.T(), s.ctx.Customer.Client).
		WithPrice(-1000, values.CurrencyRUB.String())
	resp, err := builder.CreateWithStatus()
	s.Require().NoError(err)
	s.Require().Equal(http.StatusBadRequest, resp.StatusCode)
}

func (s *FreightRequestsSuite) TestFR010_WithoutAuth() {
	builder := fixtures.NewFreightRequest(s.T(), s.ctx.AnonClient)
	resp, err := builder.CreateWithStatus()
	s.Require().NoError(err)
	s.Require().Equal(http.StatusUnauthorized, resp.StatusCode)
}

// ==================== GET /api/v1/freight-requests ====================

func (s *FreightRequestsSuite) TestFR014_ListAll() {
	// Create additional FRs
	fr1 := fixtures.NewFreightRequest(s.T(), s.ctx.Customer.Client).
		WithWeight(1000).
		WithPrice(50000, values.CurrencyRUB.String()).
		Create()
	fr2 := fixtures.NewFreightRequest(s.T(), s.ctx.Customer.Client).
		WithWeight(5000).
		WithPrice(100000, values.CurrencyRUB.String()).
		Create()

	// Wait for projection sync
	helpers.WaitFor(s.T(), func() ([]client.FreightRequestResponse, bool) {
		resp, err := s.ctx.Customer.Client.GetFreightRequests(nil)
		if err != nil || resp.StatusCode != http.StatusOK {
			return nil, false
		}
		// Check if both FRs are in the list
		found1, found2 := false, false
		for _, fr := range resp.Body {
			if fr.ID == fr1.ID {
				found1 = true
			}
			if fr.ID == fr2.ID {
				found2 = true
			}
		}
		if found1 && found2 {
			return resp.Body, true
		}
		return nil, false
	}, "both FRs should appear in list")
}

func (s *FreightRequestsSuite) TestFR016_FilterByCustomerOrgID() {
	resp, err := s.ctx.Customer.Client.GetFreightRequests(map[string]string{
		"customer_org_id": s.ctx.Customer.OrganizationID.String(),
	})
	s.Require().NoError(err)
	s.Require().Equal(http.StatusOK, resp.StatusCode, string(resp.RawBody))

	for _, fr := range resp.Body {
		s.Assert().Equal(s.ctx.Customer.OrganizationID, fr.CustomerOrgID, "customer_org_id")
	}
}

func (s *FreightRequestsSuite) TestFR022_FilterByMinWeight() {
	// Create FR with high weight
	createdFR := fixtures.NewFreightRequest(s.T(), s.ctx.Customer.Client).
		WithWeight(5000).
		Create()

	// Wait for projection sync - the created FR should appear in filtered results
	// Note: List API returns cargo_weight as flat field, client expects nested cargo.weight
	// so we just verify the FR appears in filtered results
	helpers.Wait(s.T(), func() bool {
		resp, err := s.ctx.Customer.Client.GetFreightRequests(map[string]string{
			"min_weight": "3000",
		})
		if err != nil || resp.StatusCode != http.StatusOK {
			return false
		}
		for _, fr := range resp.Body {
			if fr.ID == createdFR.ID {
				return true
			}
		}
		return false
	}, "created FR should appear in min_weight filtered results")
}

func (s *FreightRequestsSuite) TestFR030_Pagination() {
	resp, err := s.ctx.Customer.Client.GetFreightRequests(map[string]string{
		"limit": "1", "offset": "0",
	})
	s.Require().NoError(err)
	s.Require().Equal(http.StatusOK, resp.StatusCode, string(resp.RawBody))
	s.Assert().True(len(resp.Body) <= 1, "should have at most 1 request")
}

func (s *FreightRequestsSuite) TestFR031_InvalidCustomerOrgID() {
	resp, err := s.ctx.Customer.Client.GetFreightRequests(map[string]string{
		"customer_org_id": "not-uuid",
	})
	s.Require().NoError(err)
	s.Require().Equal(http.StatusBadRequest, resp.StatusCode)
}

// ==================== GET /api/v1/freight-requests/{id} ====================

func (s *FreightRequestsSuite) TestFR036_GetExistingRequest() {
	resp, err := s.ctx.Customer.Client.GetFreightRequest(s.sharedFR.ID)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusOK, resp.StatusCode, string(resp.RawBody))
}

func (s *FreightRequestsSuite) TestFR039_NonexistentRequest() {
	resp, err := s.ctx.Customer.Client.GetFreightRequest(uuid.New())
	s.Require().NoError(err)
	s.Require().Equal(http.StatusNotFound, resp.StatusCode)
}

// ==================== PATCH /api/v1/freight-requests/{id} ====================

func (s *FreightRequestsSuite) buildUpdateRequest() client.CreateFreightRequestRequest {
	tomorrow := time.Now().AddDate(0, 0, 1).Format("2006-01-02")
	dayAfter := time.Now().AddDate(0, 0, 2).Format("2006-01-02")
	return client.CreateFreightRequestRequest{
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
}

func (s *FreightRequestsSuite) TestFR041_UpdateByOwner() {
	fr := fixtures.NewFreightRequest(s.T(), s.ctx.Customer.Client).Create()

	resp, err := s.ctx.Customer.Client.UpdateFreightRequest(fr.ID, s.buildUpdateRequest())
	s.Require().NoError(err)
	s.Require().Equal(http.StatusNoContent, resp.StatusCode, string(resp.RawBody))
}

func (s *FreightRequestsSuite) TestFR046_UpdateWithoutAuth() {
	resp, err := s.ctx.AnonClient.UpdateFreightRequest(s.sharedFR.ID, s.buildUpdateRequest())
	s.Require().NoError(err)
	s.Require().Equal(http.StatusUnauthorized, resp.StatusCode)
}

func (s *FreightRequestsSuite) TestFR048_UpdateByDifferentOrg() {
	resp, err := s.ctx.Carrier.Client.UpdateFreightRequest(s.sharedFR.ID, s.buildUpdateRequest())
	s.Require().NoError(err)
	s.Require().Equal(http.StatusForbidden, resp.StatusCode)
	s.Assert().Contains(strings.ToLower(string(resp.RawBody)), "not freight request owner")
}

func (s *FreightRequestsSuite) TestFR049_UpdateNonexistent() {
	resp, err := s.ctx.Customer.Client.UpdateFreightRequest(uuid.New(), s.buildUpdateRequest())
	s.Require().NoError(err)
	s.Require().Equal(http.StatusNotFound, resp.StatusCode)
}

// ==================== DELETE /api/v1/freight-requests/{id} ====================

func (s *FreightRequestsSuite) TestFR053_SuccessfulCancel() {
	fr := fixtures.NewFreightRequest(s.T(), s.ctx.Customer.Client).Create()

	resp, err := s.ctx.Customer.Client.CancelFreightRequest(fr.ID, helpers.StringPtr("No longer needed"))
	s.Require().NoError(err)
	s.Require().Equal(http.StatusNoContent, resp.StatusCode, string(resp.RawBody))
}

func (s *FreightRequestsSuite) TestFR055_CancelWithoutAuth() {
	fr := fixtures.NewFreightRequest(s.T(), s.ctx.Customer.Client).Create()

	resp, err := s.ctx.AnonClient.CancelFreightRequest(fr.ID, nil)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusUnauthorized, resp.StatusCode)
}

func (s *FreightRequestsSuite) TestFR057_CancelOtherOrgRequest() {
	fr := fixtures.NewFreightRequest(s.T(), s.ctx.Customer.Client).Create()

	resp, err := s.ctx.Carrier.Client.CancelFreightRequest(fr.ID, nil)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusForbidden, resp.StatusCode)
}

func (s *FreightRequestsSuite) TestFR058_CancelAlreadyCancelled() {
	fr := fixtures.NewFreightRequest(s.T(), s.ctx.Customer.Client).Create()

	// Cancel first time
	s.ctx.Customer.Client.CancelFreightRequest(fr.ID, nil)

	// Try to cancel again
	resp, err := s.ctx.Customer.Client.CancelFreightRequest(fr.ID, nil)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusConflict, resp.StatusCode)
}

// ==================== POST /api/v1/freight-requests/{id}/offers ====================

func (s *FreightRequestsSuite) TestFR065_SuccessfulOffer() {
	fr := fixtures.NewFreightRequest(s.T(), s.ctx.Customer.Client).Create()

	resp, err := s.ctx.Carrier.Client.CreateOffer(fr.ID, client.CreateOfferRequest{
		Price:         client.Money{Amount: 45000, Currency: values.CurrencyRUB.String()},
		VATType:       values.VatTypeIncluded.String(),
		PaymentMethod: values.PaymentMethodBankTransfer.String(),
	})
	s.Require().NoError(err)
	s.Require().Equal(http.StatusCreated, resp.StatusCode, string(resp.RawBody))
}

func (s *FreightRequestsSuite) TestFR070_OfferOnOwnRequest() {
	fr := fixtures.NewFreightRequest(s.T(), s.ctx.Customer.Client).Create()

	resp, err := s.ctx.Customer.Client.CreateOffer(fr.ID, client.CreateOfferRequest{
		Price:         client.Money{Amount: 45000, Currency: values.CurrencyRUB.String()},
		VATType:       values.VatTypeIncluded.String(),
		PaymentMethod: values.PaymentMethodBankTransfer.String(),
	})
	s.Require().NoError(err)
	s.Require().Equal(http.StatusBadRequest, resp.StatusCode)
	s.Assert().Contains(strings.ToLower(string(resp.RawBody)), "cannot make offer on own request")
}

func (s *FreightRequestsSuite) TestFR071_OfferWithoutAuth() {
	resp, err := s.ctx.AnonClient.CreateOffer(s.sharedFR.ID, client.CreateOfferRequest{
		Price:         client.Money{Amount: 45000, Currency: values.CurrencyRUB.String()},
		VATType:       values.VatTypeIncluded.String(),
		PaymentMethod: values.PaymentMethodBankTransfer.String(),
	})
	s.Require().NoError(err)
	s.Require().Equal(http.StatusUnauthorized, resp.StatusCode)
}

func (s *FreightRequestsSuite) TestFR072_OfferOnNonexistentRequest() {
	resp, err := s.ctx.Carrier.Client.CreateOffer(uuid.New(), client.CreateOfferRequest{
		Price:         client.Money{Amount: 45000, Currency: values.CurrencyRUB.String()},
		VATType:       values.VatTypeIncluded.String(),
		PaymentMethod: values.PaymentMethodBankTransfer.String(),
	})
	s.Require().NoError(err)
	s.Require().Equal(http.StatusNotFound, resp.StatusCode)
}

func (s *FreightRequestsSuite) TestFR073_DuplicateOffer() {
	fr := fixtures.NewFreightRequest(s.T(), s.ctx.Customer.Client).Create()

	// First offer
	resp1, err := s.ctx.Carrier.Client.CreateOffer(fr.ID, client.CreateOfferRequest{
		Price:         client.Money{Amount: 45000, Currency: values.CurrencyRUB.String()},
		VATType:       values.VatTypeIncluded.String(),
		PaymentMethod: values.PaymentMethodBankTransfer.String(),
	})
	s.Require().NoError(err)
	s.Require().Equal(http.StatusCreated, resp1.StatusCode, string(resp1.RawBody))

	// Second offer from same org
	resp2, err := s.ctx.Carrier.Client.CreateOffer(fr.ID, client.CreateOfferRequest{
		Price:         client.Money{Amount: 40000, Currency: values.CurrencyRUB.String()},
		VATType:       values.VatTypeIncluded.String(),
		PaymentMethod: values.PaymentMethodBankTransfer.String(),
	})
	s.Require().NoError(err)
	s.Require().Equal(http.StatusConflict, resp2.StatusCode)
	s.Assert().Contains(strings.ToLower(string(resp2.RawBody)), "already exists")
}

// ==================== GET /api/v1/freight-requests/{id}/offers ====================

func (s *FreightRequestsSuite) TestFR077_CustomerSeesAllOffers() {
	fr := fixtures.NewFreightRequest(s.T(), s.ctx.Customer.Client).Create()
	fixtures.NewOffer(s.T(), s.ctx.Carrier.Client, fr.ID).Create()

	resp, err := s.ctx.Customer.Client.GetOffers(fr.ID)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusOK, resp.StatusCode, string(resp.RawBody))
	s.Assert().True(len(resp.Body) >= 1, "should see offers")
}

func (s *FreightRequestsSuite) TestFR078_CarrierSeesOwnOffer() {
	fr := fixtures.NewFreightRequest(s.T(), s.ctx.Customer.Client).Create()
	fixtures.NewOffer(s.T(), s.ctx.Carrier.Client, fr.ID).Create()

	resp, err := s.ctx.Carrier.Client.GetOffers(fr.ID)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusOK, resp.StatusCode, string(resp.RawBody))
	s.Assert().True(len(resp.Body) >= 1, "should see own offer")
}

func (s *FreightRequestsSuite) TestFR084_OtherOrgSeesEmpty() {
	fr := fixtures.NewFreightRequest(s.T(), s.ctx.Customer.Client).Create()
	fixtures.NewOffer(s.T(), s.ctx.Carrier.Client, fr.ID).Create()

	otherCarrier := s.ctx.QuickCarrier()
	resp, err := otherCarrier.Client.GetOffers(fr.ID)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusOK, resp.StatusCode, string(resp.RawBody))
	s.Assert().Equal(0, len(resp.Body), "should see empty list")
}

// ==================== Offer Flow Tests ====================

func (s *FreightRequestsSuite) TestFR091_SelectOffer() {
	fr := fixtures.NewFreightRequest(s.T(), s.ctx.Customer.Client).Create()
	offer := fixtures.NewOffer(s.T(), s.ctx.Carrier.Client, fr.ID).Create()

	resp, err := s.ctx.Customer.Client.SelectOffer(fr.ID, offer.OfferID)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusNoContent, resp.StatusCode, string(resp.RawBody))
}

func (s *FreightRequestsSuite) TestFR102_ConfirmOffer() {
	fr := fixtures.NewFreightRequest(s.T(), s.ctx.Customer.Client).Create()
	offer := fixtures.NewOffer(s.T(), s.ctx.Carrier.Client, fr.ID).Create()

	// Select first
	resp1, err := s.ctx.Customer.Client.SelectOffer(fr.ID, offer.OfferID)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusNoContent, resp1.StatusCode)

	// Then confirm
	resp2, err := s.ctx.Carrier.Client.ConfirmOffer(fr.ID, offer.OfferID)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusNoContent, resp2.StatusCode, string(resp2.RawBody))
}

func (s *FreightRequestsSuite) TestFR103_OrderCreated() {
	fr := fixtures.NewFreightRequest(s.T(), s.ctx.Customer.Client).Create()
	offer := fixtures.NewOffer(s.T(), s.ctx.Carrier.Client, fr.ID).Create()

	// Select and confirm
	s.ctx.Customer.Client.SelectOffer(fr.ID, offer.OfferID)
	s.ctx.Carrier.Client.ConfirmOffer(fr.ID, offer.OfferID)

	// Wait for order to be created by worker
	s.Assert().Eventually(func() bool {
		ordersResp, _ := s.ctx.Customer.Client.GetOrders(map[string]string{
			"freight_request_id": fr.ID.String(),
		})
		return ordersResp.StatusCode == http.StatusOK && len(ordersResp.Body) > 0
	}, 5*time.Second, 50*time.Millisecond, "order should be created")
}

// ==================== POST /api/v1/freight-requests/{id}/offers/{offerId}/unselect ====================

func (s *FreightRequestsSuite) TestFR110_UnselectOffer_Success() {
	fr := fixtures.NewFreightRequest(s.T(), s.ctx.Customer.Client).Create()
	offer := fixtures.NewOffer(s.T(), s.ctx.Carrier.Client, fr.ID).Create()

	// Select offer
	selectResp, err := s.ctx.Customer.Client.SelectOffer(fr.ID, offer.OfferID)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusNoContent, selectResp.StatusCode)

	// Unselect offer
	resp, err := s.ctx.Customer.Client.UnselectOffer(fr.ID, offer.OfferID, nil)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusNoContent, resp.StatusCode, string(resp.RawBody))

	// Verify FR status back to published
	frResp, err := s.ctx.Customer.Client.GetFreightRequest(fr.ID)
	s.Require().NoError(err)
	s.Assert().Equal("published", frResp.Body.Status)
}

func (s *FreightRequestsSuite) TestFR111_UnselectOffer_WithReason() {
	fr := fixtures.NewFreightRequest(s.T(), s.ctx.Customer.Client).Create()
	offer := fixtures.NewOffer(s.T(), s.ctx.Carrier.Client, fr.ID).Create()

	// Select offer
	s.ctx.Customer.Client.SelectOffer(fr.ID, offer.OfferID)

	// Unselect with reason
	reason := "Found better offer"
	resp, err := s.ctx.Customer.Client.UnselectOffer(fr.ID, offer.OfferID, &reason)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusNoContent, resp.StatusCode, string(resp.RawBody))
}

func (s *FreightRequestsSuite) TestFR112_UnselectOffer_Unauthorized() {
	fr := fixtures.NewFreightRequest(s.T(), s.ctx.Customer.Client).Create()
	offer := fixtures.NewOffer(s.T(), s.ctx.Carrier.Client, fr.ID).Create()

	// Select offer
	s.ctx.Customer.Client.SelectOffer(fr.ID, offer.OfferID)

	// Try to unselect without auth
	resp, err := s.ctx.AnonClient.UnselectOffer(fr.ID, offer.OfferID, nil)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusUnauthorized, resp.StatusCode)
}

func (s *FreightRequestsSuite) TestFR113_UnselectOffer_NotOwner() {
	fr := fixtures.NewFreightRequest(s.T(), s.ctx.Customer.Client).Create()
	offer := fixtures.NewOffer(s.T(), s.ctx.Carrier.Client, fr.ID).Create()

	// Select offer
	s.ctx.Customer.Client.SelectOffer(fr.ID, offer.OfferID)

	// Try to unselect by carrier (not owner)
	resp, err := s.ctx.Carrier.Client.UnselectOffer(fr.ID, offer.OfferID, nil)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusForbidden, resp.StatusCode)
}

func (s *FreightRequestsSuite) TestFR114_UnselectOffer_NotSelected() {
	fr := fixtures.NewFreightRequest(s.T(), s.ctx.Customer.Client).Create()
	offer := fixtures.NewOffer(s.T(), s.ctx.Carrier.Client, fr.ID).Create()

	// Try to unselect without selecting first (FR is in published status)
	resp, err := s.ctx.Customer.Client.UnselectOffer(fr.ID, offer.OfferID, nil)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusConflict, resp.StatusCode)
}

func (s *FreightRequestsSuite) TestFR115_UnselectOffer_WrongOffer() {
	fr := fixtures.NewFreightRequest(s.T(), s.ctx.Customer.Client).Create()

	// Create two offers from different carriers
	offer1 := fixtures.NewOffer(s.T(), s.ctx.Carrier.Client, fr.ID).Create()
	carrier2 := s.ctx.QuickCarrier()
	offer2 := fixtures.NewOffer(s.T(), carrier2.Client, fr.ID).Create()

	// Select offer1
	s.ctx.Customer.Client.SelectOffer(fr.ID, offer1.OfferID)

	// Try to unselect offer2 (not the selected one)
	resp, err := s.ctx.Customer.Client.UnselectOffer(fr.ID, offer2.OfferID, nil)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusConflict, resp.StatusCode)
}

func (s *FreightRequestsSuite) TestFR116_UnselectOffer_NonexistentFR() {
	resp, err := s.ctx.Customer.Client.UnselectOffer(uuid.New(), uuid.New(), nil)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusNotFound, resp.StatusCode)
}

func (s *FreightRequestsSuite) TestFR117_UnselectOffer_ThenSelectAnother() {
	fr := fixtures.NewFreightRequest(s.T(), s.ctx.Customer.Client).Create()

	// Create two offers
	offer1 := fixtures.NewOffer(s.T(), s.ctx.Carrier.Client, fr.ID).Create()
	carrier2 := s.ctx.QuickCarrier()
	offer2 := fixtures.NewOffer(s.T(), carrier2.Client, fr.ID).Create()

	// Select offer1
	s.ctx.Customer.Client.SelectOffer(fr.ID, offer1.OfferID)

	// Unselect offer1
	resp1, err := s.ctx.Customer.Client.UnselectOffer(fr.ID, offer1.OfferID, nil)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusNoContent, resp1.StatusCode)

	// Select offer2
	resp2, err := s.ctx.Customer.Client.SelectOffer(fr.ID, offer2.OfferID)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusNoContent, resp2.StatusCode)

	// Verify FR status is selected
	frResp, _ := s.ctx.Customer.Client.GetFreightRequest(fr.ID)
	s.Assert().Equal("selected", frResp.Body.Status)
}

// ==================== Cancel from selected status ====================

func (s *FreightRequestsSuite) TestFR120_CancelSelected_Success() {
	fr := fixtures.NewFreightRequest(s.T(), s.ctx.Customer.Client).Create()
	offer := fixtures.NewOffer(s.T(), s.ctx.Carrier.Client, fr.ID).Create()

	// Select offer
	s.ctx.Customer.Client.SelectOffer(fr.ID, offer.OfferID)

	// Cancel FR from selected status
	resp, err := s.ctx.Customer.Client.CancelFreightRequest(fr.ID, helpers.StringPtr("Changed plans"))
	s.Require().NoError(err)
	s.Require().Equal(http.StatusNoContent, resp.StatusCode, string(resp.RawBody))

	// Verify FR is cancelled
	frResp, _ := s.ctx.Customer.Client.GetFreightRequest(fr.ID)
	s.Assert().Equal("cancelled", frResp.Body.Status)
}

func (s *FreightRequestsSuite) TestFR121_CancelSelected_OfferRejected() {
	fr := fixtures.NewFreightRequest(s.T(), s.ctx.Customer.Client).Create()
	offer := fixtures.NewOffer(s.T(), s.ctx.Carrier.Client, fr.ID).Create()

	// Select offer
	s.ctx.Customer.Client.SelectOffer(fr.ID, offer.OfferID)

	// Cancel FR
	s.ctx.Customer.Client.CancelFreightRequest(fr.ID, nil)

	// Verify offer is rejected
	helpers.Wait(s.T(), func() bool {
		offersResp, err := s.ctx.Customer.Client.GetOffers(fr.ID)
		if err != nil || offersResp.StatusCode != http.StatusOK {
			return false
		}
		for _, o := range offersResp.Body {
			if o.ID == offer.OfferID {
				return o.Status == "rejected"
			}
		}
		return false
	}, "offer should be rejected after FR cancellation")
}

func (s *FreightRequestsSuite) TestFR122_CancelSelected_PendingOffersRejected() {
	fr := fixtures.NewFreightRequest(s.T(), s.ctx.Customer.Client).Create()

	// Create multiple offers
	offer1 := fixtures.NewOffer(s.T(), s.ctx.Carrier.Client, fr.ID).Create()
	carrier2 := s.ctx.QuickCarrier()
	offer2 := fixtures.NewOffer(s.T(), carrier2.Client, fr.ID).Create()

	// Select offer1
	s.ctx.Customer.Client.SelectOffer(fr.ID, offer1.OfferID)

	// Cancel FR
	s.ctx.Customer.Client.CancelFreightRequest(fr.ID, nil)

	// Verify all offers are rejected
	helpers.Wait(s.T(), func() bool {
		offersResp, err := s.ctx.Customer.Client.GetOffers(fr.ID)
		if err != nil || offersResp.StatusCode != http.StatusOK {
			return false
		}
		for _, o := range offersResp.Body {
			if o.ID == offer1.OfferID || o.ID == offer2.OfferID {
				if o.Status != "rejected" {
					return false
				}
			}
		}
		return true
	}, "all offers should be rejected after FR cancellation")
}

// ==================== List filters ====================

func (s *FreightRequestsSuite) TestFR130_FilterByMinVolume() {
	fr := fixtures.NewFreightRequest(s.T(), s.ctx.Customer.Client).
		WithVolume(15.0).
		Create()

	helpers.Wait(s.T(), func() bool {
		resp, err := s.ctx.Customer.Client.GetFreightRequests(map[string]string{
			"min_volume": "10.0",
		})
		if err != nil || resp.StatusCode != http.StatusOK {
			return false
		}
		for _, item := range resp.Body {
			if item.ID == fr.ID {
				return true
			}
		}
		return false
	}, "FR with volume 15 should appear in min_volume=10 filter")
}

func (s *FreightRequestsSuite) TestFR131_FilterByMaxVolume() {
	fr := fixtures.NewFreightRequest(s.T(), s.ctx.Customer.Client).
		WithVolume(5.0).
		Create()

	helpers.Wait(s.T(), func() bool {
		resp, err := s.ctx.Customer.Client.GetFreightRequests(map[string]string{
			"max_volume": "10.0",
		})
		if err != nil || resp.StatusCode != http.StatusOK {
			return false
		}
		for _, item := range resp.Body {
			if item.ID == fr.ID {
				return true
			}
		}
		return false
	}, "FR with volume 5 should appear in max_volume=10 filter")
}

func (s *FreightRequestsSuite) TestFR132_FilterByPaymentMethods() {
	fr := fixtures.NewFreightRequest(s.T(), s.ctx.Customer.Client).
		WithPayment(50000, values.CurrencyRUB.String(), values.VatTypeIncluded.String(),
			values.PaymentMethodCash.String(), values.PaymentTermsPrepaid.String()).
		Create()

	helpers.Wait(s.T(), func() bool {
		resp, err := s.ctx.Customer.Client.GetFreightRequests(map[string]string{
			"payment_methods": values.PaymentMethodCash.String(),
		})
		if err != nil || resp.StatusCode != http.StatusOK {
			return false
		}
		for _, item := range resp.Body {
			if item.ID == fr.ID {
				return true
			}
		}
		return false
	}, "FR with cash payment should appear in payment_methods filter")
}

func (s *FreightRequestsSuite) TestFR133_FilterByPaymentTerms() {
	fr := fixtures.NewFreightRequest(s.T(), s.ctx.Customer.Client).
		WithPayment(50000, values.CurrencyRUB.String(), values.VatTypeIncluded.String(),
			values.PaymentMethodBankTransfer.String(), values.PaymentTermsDeferred.String()).
		Create()

	helpers.Wait(s.T(), func() bool {
		resp, err := s.ctx.Customer.Client.GetFreightRequests(map[string]string{
			"payment_terms": values.PaymentTermsDeferred.String(),
		})
		if err != nil || resp.StatusCode != http.StatusOK {
			return false
		}
		for _, item := range resp.Body {
			if item.ID == fr.ID {
				return true
			}
		}
		return false
	}, "FR with deferred terms should appear in payment_terms filter")
}

func (s *FreightRequestsSuite) TestFR134_FilterByVatTypes() {
	fr := fixtures.NewFreightRequest(s.T(), s.ctx.Customer.Client).
		WithPayment(50000, values.CurrencyRUB.String(), values.VatTypeExcluded.String(),
			values.PaymentMethodBankTransfer.String(), values.PaymentTermsPrepaid.String()).
		Create()

	helpers.Wait(s.T(), func() bool {
		resp, err := s.ctx.Customer.Client.GetFreightRequests(map[string]string{
			"vat_types": values.VatTypeExcluded.String(),
		})
		if err != nil || resp.StatusCode != http.StatusOK {
			return false
		}
		for _, item := range resp.Body {
			if item.ID == fr.ID {
				return true
			}
		}
		return false
	}, "FR with excluded VAT should appear in vat_types filter")
}

func (s *FreightRequestsSuite) TestFR135_FilterCombined() {
	fr := fixtures.NewFreightRequest(s.T(), s.ctx.Customer.Client).
		WithVolume(20.0).
		WithPayment(50000, values.CurrencyRUB.String(), values.VatTypeIncluded.String(),
			values.PaymentMethodBankTransfer.String(), values.PaymentTermsPrepaid.String()).
		Create()

	helpers.Wait(s.T(), func() bool {
		resp, err := s.ctx.Customer.Client.GetFreightRequests(map[string]string{
			"min_volume":      "15.0",
			"payment_methods": values.PaymentMethodBankTransfer.String(),
			"vat_types":       values.VatTypeIncluded.String(),
		})
		if err != nil || resp.StatusCode != http.StatusOK {
			return false
		}
		for _, item := range resp.Body {
			if item.ID == fr.ID {
				return true
			}
		}
		return false
	}, "FR should appear with combined filters")
}

// Helper function
func intPtr(i int) *int {
	return &i
}
