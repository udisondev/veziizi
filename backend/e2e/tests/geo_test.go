package tests

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/suite"
	"github.com/udisondev/veziizi/backend/e2e/client"
)

// GeoSuite combines all geo tests with shared context.
type GeoSuite struct {
	suite.Suite
	baseURL string
	c       *client.Client

	// Cached data from initial queries
	validCountryID int
	russiaID       int
	validCityID    int
}

func TestGeoSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(GeoSuite))
}

func (s *GeoSuite) SetupSuite() {
	testSuite := getSuite(s.T())
	s.baseURL = testSuite.BaseURL
	s.c = client.New(s.baseURL)

	// Pre-fetch countries to get valid IDs
	countriesResp, err := s.c.GetCountries()
	s.Require().NoError(err)
	s.Require().Equal(http.StatusOK, countriesResp.StatusCode)
	s.Require().True(len(countriesResp.Body) > 0, "should have at least one country")

	s.validCountryID = countriesResp.Body[0].ID

	// Find Russia
	for _, country := range countriesResp.Body {
		if country.ISOCode == "RU" {
			s.russiaID = country.ID
			break
		}
	}
	s.Require().NotZero(s.russiaID, "Russia should be in countries list")

	// Get a valid city ID
	citiesResp, err := s.c.GetCountryCities(s.validCountryID, "", 1)
	s.Require().NoError(err)
	if citiesResp.StatusCode == http.StatusOK && len(citiesResp.Body) > 0 {
		s.validCityID = citiesResp.Body[0].ID
	}
}

// ==================== GET /api/v1/geo/countries ====================

func (s *GeoSuite) TestGEO001_ListCountries() {
	resp, err := s.c.GetCountries()
	s.Require().NoError(err)
	s.Require().Equal(http.StatusOK, resp.StatusCode, string(resp.RawBody))
	s.Assert().Contains(string(resp.RawBody), "RU", "should contain Russia")
}

func (s *GeoSuite) TestGEO002_PublicAccessWithoutAuth() {
	resp, err := s.c.GetCountries()
	s.Require().NoError(err)
	s.Require().Equal(http.StatusOK, resp.StatusCode, string(resp.RawBody))
}

// ==================== GET /api/v1/geo/countries/{id} ====================

func (s *GeoSuite) TestGEO003_GetExistingCountry() {
	resp, err := s.c.GetCountry(s.validCountryID)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusOK, resp.StatusCode, string(resp.RawBody))
	s.Assert().Equal(s.validCountryID, resp.Body.ID)
	s.Assert().NotEmpty(resp.Body.Name)
	s.Assert().NotEmpty(resp.Body.ISOCode)
}

func (s *GeoSuite) TestGEO004_CountryPublicAccess() {
	resp, err := s.c.GetCountry(s.validCountryID)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusOK, resp.StatusCode, string(resp.RawBody))
}

func (s *GeoSuite) TestGEO005_InvalidCountryID() {
	status, _, err := s.c.GetCountryRaw("abc")
	s.Require().NoError(err)
	s.Require().Equal(http.StatusNotFound, status) // Router will not match pattern [0-9]+
}

func (s *GeoSuite) TestGEO006_NonexistentCountry() {
	resp, err := s.c.GetCountry(999999)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusNotFound, resp.StatusCode)
}

// ==================== GET /api/v1/geo/countries/{id}/cities ====================

func (s *GeoSuite) TestGEO007_ListCitiesForCountry() {
	resp, err := s.c.GetCountryCities(s.russiaID, "", 0)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusOK, resp.StatusCode, string(resp.RawBody))
	s.Assert().Contains(string(resp.RawBody), "[", "should be an array")
}

func (s *GeoSuite) TestGEO008_SearchCitiesByName() {
	resp, err := s.c.GetCountryCities(s.russiaID, "Моск", 0)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusOK, resp.StatusCode, string(resp.RawBody))
	s.Assert().Contains(string(resp.RawBody), "Москва", "should find Moscow")
}

func (s *GeoSuite) TestGEO009_PaginationWithLimit() {
	resp, err := s.c.GetCountryCities(s.russiaID, "", 5)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusOK, resp.StatusCode, string(resp.RawBody))
	s.Assert().LessOrEqual(len(resp.Body), 5, "should respect limit")
}

func (s *GeoSuite) TestGEO010_CitiesPublicAccess() {
	resp, err := s.c.GetCountryCities(s.russiaID, "", 0)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusOK, resp.StatusCode, string(resp.RawBody))
}

// ==================== GET /api/v1/geo/cities/{id} ====================

func (s *GeoSuite) TestGEO012_GetExistingCity() {
	if s.validCityID == 0 {
		s.T().Skip("no valid city ID available")
	}

	resp, err := s.c.GetCity(s.validCityID)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusOK, resp.StatusCode, string(resp.RawBody))
	s.Assert().Equal(s.validCityID, resp.Body.ID)
	s.Assert().NotEmpty(resp.Body.Name)
	s.Assert().NotZero(resp.Body.CountryID)
}

func (s *GeoSuite) TestGEO013_CityPublicAccess() {
	if s.validCityID == 0 {
		s.T().Skip("no valid city ID available")
	}

	resp, err := s.c.GetCity(s.validCityID)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusOK, resp.StatusCode, string(resp.RawBody))
}

func (s *GeoSuite) TestGEO014_NonexistentCity() {
	resp, err := s.c.GetCity(999999999)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusNotFound, resp.StatusCode)
}
