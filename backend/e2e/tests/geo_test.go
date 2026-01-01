package tests

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGetCountries tests GET /api/v1/geo/countries
func TestGetCountries(t *testing.T) {
	t.Parallel()
	c := getClient(t)

	tests := []struct {
		id         string
		name       string
		wantStatus int
		check      func(*testing.T, []byte)
	}{
		{
			id:         "GEO-001",
			name:       "list countries",
			wantStatus: http.StatusOK,
			check: func(t *testing.T, body []byte) {
				// Should have at least one country (seeded)
				assert.Contains(t, string(body), "RU", "should contain Russia")
			},
		},
		{
			id:         "GEO-002",
			name:       "public access without auth",
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.id+"_"+tt.name, func(t *testing.T) {
			resp, err := c.GetCountries()
			require.NoError(t, err)
			require.Equal(t, tt.wantStatus, resp.StatusCode, string(resp.RawBody))

			if tt.check != nil && resp.StatusCode == http.StatusOK {
				tt.check(t, resp.RawBody)
			}
		})
	}
}

// TestGetCountry tests GET /api/v1/geo/countries/{id}
func TestGetCountry(t *testing.T) {
	t.Parallel()
	c := getClient(t)

	// First get list of countries to get a valid ID
	countriesResp, err := c.GetCountries()
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, countriesResp.StatusCode)
	require.True(t, len(countriesResp.Body) > 0, "should have at least one country")

	validCountryID := countriesResp.Body[0].ID

	tests := []struct {
		id         string
		name       string
		countryID  int
		useRaw     bool
		rawID      string
		wantStatus int
	}{
		{
			id:         "GEO-003",
			name:       "get existing country",
			countryID:  validCountryID,
			wantStatus: http.StatusOK,
		},
		{
			id:         "GEO-004",
			name:       "public access without auth",
			countryID:  validCountryID,
			wantStatus: http.StatusOK,
		},
		{
			id:         "GEO-005",
			name:       "invalid country ID (non-numeric)",
			useRaw:     true,
			rawID:      "abc",
			wantStatus: http.StatusNotFound, // Router will not match pattern [0-9]+
		},
		{
			id:         "GEO-006",
			name:       "nonexistent country",
			countryID:  999999,
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.id+"_"+tt.name, func(t *testing.T) {
			if tt.useRaw {
				status, _, err := c.GetCountryRaw(tt.rawID)
				require.NoError(t, err)
				require.Equal(t, tt.wantStatus, status)
				return
			}

			resp, err := c.GetCountry(tt.countryID)
			require.NoError(t, err)
			require.Equal(t, tt.wantStatus, resp.StatusCode, string(resp.RawBody))

			if resp.StatusCode == http.StatusOK {
				assert.Equal(t, tt.countryID, resp.Body.ID)
				assert.NotEmpty(t, resp.Body.Name)
				assert.NotEmpty(t, resp.Body.ISOCode)
			}
		})
	}
}

// TestGetCountryCities tests GET /api/v1/geo/countries/{id}/cities
func TestGetCountryCities(t *testing.T) {
	t.Parallel()
	c := getClient(t)

	// Get Russia country ID (should be seeded)
	countriesResp, err := c.GetCountries()
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, countriesResp.StatusCode)

	var russiaID int
	for _, country := range countriesResp.Body {
		if country.ISOCode == "RU" {
			russiaID = country.ID
			break
		}
	}
	require.NotZero(t, russiaID, "Russia should be in countries list")

	tests := []struct {
		id         string
		name       string
		countryID  int
		search     string
		limit      int
		wantStatus int
		check      func(*testing.T, []byte)
	}{
		{
			id:         "GEO-007",
			name:       "list cities for country",
			countryID:  russiaID,
			wantStatus: http.StatusOK,
			check: func(t *testing.T, body []byte) {
				assert.Contains(t, string(body), "[", "should be an array")
			},
		},
		{
			id:         "GEO-008",
			name:       "search cities by name",
			countryID:  russiaID,
			search:     "Моск",
			wantStatus: http.StatusOK,
			check: func(t *testing.T, body []byte) {
				// Moscow should be found when searching for "Моск"
				assert.Contains(t, string(body), "Москва", "should find Moscow")
			},
		},
		{
			id:         "GEO-009",
			name:       "pagination with limit",
			countryID:  russiaID,
			limit:      5,
			wantStatus: http.StatusOK,
		},
		{
			id:         "GEO-010",
			name:       "public access without auth",
			countryID:  russiaID,
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.id+"_"+tt.name, func(t *testing.T) {
			resp, err := c.GetCountryCities(tt.countryID, tt.search, tt.limit)
			require.NoError(t, err)
			require.Equal(t, tt.wantStatus, resp.StatusCode, string(resp.RawBody))

			if tt.check != nil && resp.StatusCode == http.StatusOK {
				tt.check(t, resp.RawBody)
			}

			// Check limit is respected
			if tt.limit > 0 && resp.StatusCode == http.StatusOK {
				assert.LessOrEqual(t, len(resp.Body), tt.limit, "should respect limit")
			}
		})
	}
}

// TestGetCity tests GET /api/v1/geo/cities/{id}
func TestGetCity(t *testing.T) {
	t.Parallel()
	c := getClient(t)

	// Get a valid city ID first
	countriesResp, err := c.GetCountries()
	require.NoError(t, err)
	require.True(t, len(countriesResp.Body) > 0)

	// Get cities for first country
	citiesResp, err := c.GetCountryCities(countriesResp.Body[0].ID, "", 1)
	require.NoError(t, err)

	var validCityID int
	if citiesResp.StatusCode == http.StatusOK && len(citiesResp.Body) > 0 {
		validCityID = citiesResp.Body[0].ID
	}

	tests := []struct {
		id         string
		name       string
		cityID     int
		useRaw     bool
		rawID      string
		wantStatus int
		skip       bool
	}{
		{
			id:         "GEO-012",
			name:       "get existing city",
			cityID:     validCityID,
			wantStatus: http.StatusOK,
			skip:       validCityID == 0,
		},
		{
			id:         "GEO-013",
			name:       "public access without auth",
			cityID:     validCityID,
			wantStatus: http.StatusOK,
			skip:       validCityID == 0,
		},
		{
			id:         "GEO-014",
			name:       "nonexistent city",
			cityID:     999999999,
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.id+"_"+tt.name, func(t *testing.T) {
			if tt.skip {
				t.Skip("no valid city ID available")
			}

			if tt.useRaw {
				status, _, err := c.GetCityRaw(tt.rawID)
				require.NoError(t, err)
				require.Equal(t, tt.wantStatus, status)
				return
			}

			resp, err := c.GetCity(tt.cityID)
			require.NoError(t, err)
			require.Equal(t, tt.wantStatus, resp.StatusCode, string(resp.RawBody))

			if resp.StatusCode == http.StatusOK {
				assert.Equal(t, tt.cityID, resp.Body.ID)
				assert.NotEmpty(t, resp.Body.Name)
				assert.NotZero(t, resp.Body.CountryID)
			}
		})
	}
}
