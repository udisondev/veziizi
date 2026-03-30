package tests

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/udisondev/veziizi/backend/e2e/client"
	"github.com/udisondev/veziizi/backend/e2e/fixtures"
	"github.com/udisondev/veziizi/backend/e2e/helpers"
	"github.com/udisondev/veziizi/backend/internal/domain/freightrequest/values"
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
	helpers.WaitFor(s.T(), func() ([]client.FreightRequestListItem, bool) {
		resp, err := s.ctx.Customer.Client.GetFreightRequests(nil)
		if err != nil || resp.StatusCode != http.StatusOK {
			return nil, false
		}
		// Check if both FRs are in the list
		found1, found2 := false, false
		for _, fr := range resp.Body.Items {
			if fr.ID == fr1.ID {
				found1 = true
			}
			if fr.ID == fr2.ID {
				found2 = true
			}
		}
		if found1 && found2 {
			return resp.Body.Items, true
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

	for _, fr := range resp.Body.Items {
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
		for _, fr := range resp.Body.Items {
			if fr.ID == createdFR.ID {
				return true
			}
		}
		return false
	}, "created FR should appear in min_weight filtered results")
}

func (s *FreightRequestsSuite) TestFR030_Pagination() {
	resp, err := s.ctx.Customer.Client.GetFreightRequests(map[string]string{
		"limit": "1",
	})
	s.Require().NoError(err)
	s.Require().Equal(http.StatusOK, resp.StatusCode, string(resp.RawBody))
	s.Assert().True(len(resp.Body.Items) <= 1, "should have at most 1 request")
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

func (s *FreightRequestsSuite) TestFR103_ConfirmedStatus() {
	fr := fixtures.NewFreightRequest(s.T(), s.ctx.Customer.Client).Create()
	offer := fixtures.NewOffer(s.T(), s.ctx.Carrier.Client, fr.ID).Create()

	// Select and confirm
	s.ctx.Customer.Client.SelectOffer(fr.ID, offer.OfferID)
	s.ctx.Carrier.Client.ConfirmOffer(fr.ID, offer.OfferID)

	// Verify FR status changes to confirmed
	helpers.WaitFor(s.T(), func() (string, bool) {
		frResp, err := s.ctx.Customer.Client.GetFreightRequest(fr.ID)
		if err != nil || frResp.StatusCode != http.StatusOK {
			return "", false
		}
		return frResp.Body.Status, frResp.Body.Status == "confirmed"
	}, "FR should be in confirmed status after offer confirmation")
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
		for _, item := range resp.Body.Items {
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
		for _, item := range resp.Body.Items {
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
		for _, item := range resp.Body.Items {
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
		for _, item := range resp.Body.Items {
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
		for _, item := range resp.Body.Items {
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
		for _, item := range resp.Body.Items {
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

func stringPtr(s string) *string {
	return &s
}

// ============================================================================
// COMPLETION TESTS (TestFR200-206)
// ============================================================================

func (s *FreightRequestsSuite) TestFR200_Complete_CustomerFirst() {
	fr, _ := s.ctx.CreateConfirmedFreightRequest()

	// Customer completes first
	resp, err := s.ctx.Customer.Client.CompleteFreightRequest(fr.ID)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusNoContent, resp.StatusCode)

	// Verify FR status is partially_completed
	helpers.Wait(s.T(), func() bool {
		frResp, err := s.ctx.Customer.Client.GetFreightRequest(fr.ID)
		return err == nil && frResp.StatusCode == http.StatusOK && frResp.Body.Status == "partially_completed"
	}, "FR should be partially_completed after customer completes")
}

func (s *FreightRequestsSuite) TestFR201_Complete_CarrierFirst() {
	fr, _ := s.ctx.CreateConfirmedFreightRequest()

	// Carrier completes first
	resp, err := s.ctx.Carrier.Client.CompleteFreightRequest(fr.ID)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusNoContent, resp.StatusCode)

	// Verify FR status is partially_completed
	helpers.Wait(s.T(), func() bool {
		frResp, err := s.ctx.Customer.Client.GetFreightRequest(fr.ID)
		return err == nil && frResp.StatusCode == http.StatusOK && frResp.Body.Status == "partially_completed"
	}, "FR should be partially_completed after carrier completes")
}

func (s *FreightRequestsSuite) TestFR202_Complete_BothSides() {
	fr, _ := s.ctx.CreateConfirmedFreightRequest()

	// Customer completes
	resp, err := s.ctx.Customer.Client.CompleteFreightRequest(fr.ID)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusNoContent, resp.StatusCode)

	// Carrier completes
	resp, err = s.ctx.Carrier.Client.CompleteFreightRequest(fr.ID)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusNoContent, resp.StatusCode)

	// Verify FR status is completed
	helpers.Wait(s.T(), func() bool {
		frResp, err := s.ctx.Customer.Client.GetFreightRequest(fr.ID)
		return err == nil && frResp.StatusCode == http.StatusOK && frResp.Body.Status == "completed"
	}, "FR should be completed after both sides complete")
}

func (s *FreightRequestsSuite) TestFR203_Complete_AlreadyCompleted() {
	completed := s.ctx.CreatePartiallyCompletedByCustomer()

	// Customer tries to complete again
	resp, err := s.ctx.Customer.Client.CompleteFreightRequest(completed.FreightRequest.ID)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusConflict, resp.StatusCode, "should return 409 when already completed by this party")
}

func (s *FreightRequestsSuite) TestFR204_Complete_NotConfirmed() {
	fr, _ := s.ctx.CreateSelectedOffer()

	// Try to complete before confirmed
	resp, err := s.ctx.Customer.Client.CompleteFreightRequest(fr.ID)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusConflict, resp.StatusCode, "should return 409 when FR is not confirmed")
}

func (s *FreightRequestsSuite) TestFR205_Complete_NotParticipant() {
	fr, _ := s.ctx.CreateConfirmedFreightRequest()

	// Third party tries to complete
	thirdParty := s.ctx.QuickCarrier()
	resp, err := thirdParty.Client.CompleteFreightRequest(fr.ID)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusForbidden, resp.StatusCode, "should return 403 for non-participant")
}

func (s *FreightRequestsSuite) TestFR206_Complete_WithoutAuth() {
	fr, _ := s.ctx.CreateConfirmedFreightRequest()

	resp, err := s.ctx.AnonClient.CompleteFreightRequest(fr.ID)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusUnauthorized, resp.StatusCode)
}

// ============================================================================
// REVIEW TESTS (TestFR210-216)
// ============================================================================

func (s *FreightRequestsSuite) TestFR210_Review_CustomerLeaves() {
	completed := s.ctx.CreateFullyCompletedFreightRequest()

	comment := "Great carrier!"
	resp, err := s.ctx.Customer.Client.LeaveFreightRequestReview(completed.FreightRequest.ID, 5, &comment)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusCreated, resp.StatusCode, "customer should be able to leave review")
	s.Require().NotEqual(uuid.Nil, resp.Body.ReviewID, "should return review ID")
}

func (s *FreightRequestsSuite) TestFR211_Review_CarrierLeaves() {
	completed := s.ctx.CreateFullyCompletedFreightRequest()

	comment := "Great customer!"
	resp, err := s.ctx.Carrier.Client.LeaveFreightRequestReview(completed.FreightRequest.ID, 4, &comment)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusCreated, resp.StatusCode, "carrier should be able to leave review")
	s.Require().NotEqual(uuid.Nil, resp.Body.ReviewID, "should return review ID")
}

func (s *FreightRequestsSuite) TestFR212_Review_BothLeave() {
	completed := s.ctx.CreateFullyCompletedFreightRequest()

	// Customer leaves review
	customerComment := "Great carrier!"
	resp, err := s.ctx.Customer.Client.LeaveFreightRequestReview(completed.FreightRequest.ID, 5, &customerComment)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusCreated, resp.StatusCode)

	// Carrier leaves review
	carrierComment := "Great customer!"
	resp, err = s.ctx.Carrier.Client.LeaveFreightRequestReview(completed.FreightRequest.ID, 4, &carrierComment)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusCreated, resp.StatusCode)
}

func (s *FreightRequestsSuite) TestFR213_Review_BeforeComplete() {
	fr, _ := s.ctx.CreateConfirmedFreightRequest()

	// Try to leave review before completing
	comment := "Test"
	resp, err := s.ctx.Customer.Client.LeaveFreightRequestReview(fr.ID, 5, &comment)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusConflict, resp.StatusCode, "should return 409 when not completed")
}

func (s *FreightRequestsSuite) TestFR214_Review_Duplicate() {
	completed := s.ctx.CreateFullyCompletedFreightRequest()

	// Leave first review
	comment := "First review"
	resp, err := s.ctx.Customer.Client.LeaveFreightRequestReview(completed.FreightRequest.ID, 5, &comment)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusCreated, resp.StatusCode)

	// Try to leave second review
	comment2 := "Second review"
	resp, err = s.ctx.Customer.Client.LeaveFreightRequestReview(completed.FreightRequest.ID, 4, &comment2)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusConflict, resp.StatusCode, "should return 409 for duplicate review")
}

func (s *FreightRequestsSuite) TestFR215_Review_InvalidRating() {
	completed := s.ctx.CreateFullyCompletedFreightRequest()

	// Rating 0 is invalid
	resp, err := s.ctx.Customer.Client.LeaveFreightRequestReview(completed.FreightRequest.ID, 0, nil)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusBadRequest, resp.StatusCode, "rating 0 should be invalid")

	// Rating 6 is invalid
	resp, err = s.ctx.Customer.Client.LeaveFreightRequestReview(completed.FreightRequest.ID, 6, nil)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusBadRequest, resp.StatusCode, "rating 6 should be invalid")
}

func (s *FreightRequestsSuite) TestFR216_Review_WithoutAuth() {
	completed := s.ctx.CreateFullyCompletedFreightRequest()

	resp, err := s.ctx.AnonClient.LeaveFreightRequestReview(completed.FreightRequest.ID, 5, nil)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusUnauthorized, resp.StatusCode)
}

// ============================================================================
// EDIT REVIEW TESTS (TestFR220-222)
// ============================================================================

func (s *FreightRequestsSuite) TestFR220_EditReview_Success() {
	completed := s.ctx.CreateFullyCompletedFreightRequest()

	// Leave review
	comment := "Initial comment"
	resp, err := s.ctx.Customer.Client.LeaveFreightRequestReview(completed.FreightRequest.ID, 4, &comment)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusCreated, resp.StatusCode)

	// Edit review (within 24h window)
	newComment := "Updated comment"
	editResp, err := s.ctx.Customer.Client.EditFreightRequestReview(completed.FreightRequest.ID, 5, &newComment)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusNoContent, editResp.StatusCode, "should be able to edit review within 24h")
}

func (s *FreightRequestsSuite) TestFR221_EditReview_NotOwner() {
	completed := s.ctx.CreateFullyCompletedFreightRequest()

	// Customer leaves review
	comment := "Customer review"
	resp, err := s.ctx.Customer.Client.LeaveFreightRequestReview(completed.FreightRequest.ID, 4, &comment)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusCreated, resp.StatusCode)

	// Carrier tries to edit customer's review
	newComment := "Hacked!"
	editResp, err := s.ctx.Carrier.Client.EditFreightRequestReview(completed.FreightRequest.ID, 1, &newComment)
	s.Require().NoError(err)
	// Should fail - either 403 (forbidden) or 404 (not found for this org)
	s.Require().True(editResp.StatusCode == http.StatusForbidden || editResp.StatusCode == http.StatusNotFound,
		"should not allow editing other's review, got %d", editResp.StatusCode)
}

func (s *FreightRequestsSuite) TestFR222_EditReview_NotExists() {
	completed := s.ctx.CreateFullyCompletedFreightRequest()

	// Try to edit non-existent review
	newComment := "Test"
	editResp, err := s.ctx.Customer.Client.EditFreightRequestReview(completed.FreightRequest.ID, 5, &newComment)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusNotFound, editResp.StatusCode, "should return 404 for non-existent review")
}

// ============================================================================
// CANCEL AFTER CONFIRMED TESTS (TestFR230-233)
// ============================================================================

func (s *FreightRequestsSuite) TestFR230_CancelConfirmed_ByCustomer() {
	fr, _ := s.ctx.CreateConfirmedFreightRequest()

	reason := "Plans changed"
	resp, err := s.ctx.Customer.Client.CancelFreightRequestAfterConfirmed(fr.ID, &reason)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusNoContent, resp.StatusCode)

	// Verify FR status is cancelled_after_confirmed
	helpers.Wait(s.T(), func() bool {
		frResp, err := s.ctx.Customer.Client.GetFreightRequest(fr.ID)
		return err == nil && frResp.StatusCode == http.StatusOK && frResp.Body.Status == "cancelled_after_confirmed"
	}, "FR should be cancelled_after_confirmed")
}

func (s *FreightRequestsSuite) TestFR231_CancelConfirmed_ByCarrier() {
	fr, _ := s.ctx.CreateConfirmedFreightRequest()

	reason := "Cannot fulfill"
	resp, err := s.ctx.Carrier.Client.CancelFreightRequestAfterConfirmed(fr.ID, &reason)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusNoContent, resp.StatusCode)

	// Verify FR status is cancelled_after_confirmed
	helpers.Wait(s.T(), func() bool {
		frResp, err := s.ctx.Customer.Client.GetFreightRequest(fr.ID)
		return err == nil && frResp.StatusCode == http.StatusOK && frResp.Body.Status == "cancelled_after_confirmed"
	}, "FR should be cancelled_after_confirmed")
}

func (s *FreightRequestsSuite) TestFR232_CancelConfirmed_NotConfirmed() {
	fr, _ := s.ctx.CreateSelectedOffer()

	reason := "Test"
	resp, err := s.ctx.Customer.Client.CancelFreightRequestAfterConfirmed(fr.ID, &reason)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusConflict, resp.StatusCode, "should return 409 when FR is not confirmed")
}

func (s *FreightRequestsSuite) TestFR233_CancelConfirmed_NotParticipant() {
	fr, _ := s.ctx.CreateConfirmedFreightRequest()

	// Third party tries to cancel
	thirdParty := s.ctx.QuickCarrier()
	reason := "Malicious cancel"
	resp, err := thirdParty.Client.CancelFreightRequestAfterConfirmed(fr.ID, &reason)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusForbidden, resp.StatusCode, "should return 403 for non-participant")
}

// ============================================================================
// REASSIGN CARRIER MEMBER TESTS (TestFR240-242)
// ============================================================================

func (s *FreightRequestsSuite) TestFR240_ReassignCarrier_Success() {
	s.T().Skip("TODO: Fix projection sync timing issues")
	fr, _ := s.ctx.CreateConfirmedFreightRequest()

	// Add new member to carrier organization
	newMemberClient := s.ctx.AddMemberToOrg(s.ctx.Carrier, "administrator")
	meResp, err := newMemberClient.Me()
	s.Require().NoError(err)
	s.Require().Equal(http.StatusOK, meResp.StatusCode)
	newMemberID := meResp.Body.MemberID

	// Reassign to new member
	resp, err := s.ctx.Carrier.Client.ReassignCarrierMember(fr.ID, newMemberID)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusNoContent, resp.StatusCode, "should be able to reassign carrier member")
}

func (s *FreightRequestsSuite) TestFR241_ReassignCarrier_NotOwner() {
	s.T().Skip("TODO: Fix projection sync timing issues")
	fr, _ := s.ctx.CreateConfirmedFreightRequest()

	// Add new member to carrier as employee (not owner/admin)
	employeeClient := s.ctx.AddMemberToOrg(s.ctx.Carrier, "employee")
	meResp, err := employeeClient.Me()
	s.Require().NoError(err)
	s.Require().Equal(http.StatusOK, meResp.StatusCode)
	employeeID := meResp.Body.MemberID

	// Employee tries to reassign (should fail)
	resp, err := employeeClient.ReassignCarrierMember(fr.ID, employeeID)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusForbidden, resp.StatusCode, "employee should not be able to reassign")
}

func (s *FreightRequestsSuite) TestFR242_ReassignCarrier_NotConfirmed() {
	s.T().Skip("TODO: Fix projection sync timing issues")
	fr, _ := s.ctx.CreateSelectedOffer()

	// Add new member to carrier organization
	newMemberClient := s.ctx.AddMemberToOrg(s.ctx.Carrier, "administrator")
	meResp, err := newMemberClient.Me()
	s.Require().NoError(err)
	newMemberID := meResp.Body.MemberID

	// Try to reassign before confirmed
	resp, err := s.ctx.Carrier.Client.ReassignCarrierMember(fr.ID, newMemberID)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusConflict, resp.StatusCode, "should return 409 when FR is not confirmed")
}
