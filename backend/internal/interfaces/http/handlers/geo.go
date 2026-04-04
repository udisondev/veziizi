package handlers

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/udisondev/veziizi/backend/internal/infrastructure/projections"
)

// RegisterRoutes registers geo-related routes
func (h *GeoHandler) RegisterRoutes(r chi.Router) {
	// Geo routes (public, no auth required)
	r.Route("/api/v1/geo", func(r chi.Router) {
		r.Get("/countries", h.ListCountries)
		r.Get("/countries/{id:[0-9]+}", h.GetCountry)
		r.Get("/countries/{id:[0-9]+}/cities", h.ListCities)
		r.Get("/cities/{id:[0-9]+}", h.GetCity)
	})
}

// GeoHandler handles geo-related HTTP requests (countries and cities)
type GeoHandler struct {
	projection *projections.GeoProjection
}

// NewGeoHandler creates a new GeoHandler
func NewGeoHandler(projection *projections.GeoProjection) *GeoHandler {
	return &GeoHandler{
		projection: projection,
	}
}

// ListCountries returns all countries
// GET /api/v1/geo/countries
func (h *GeoHandler) ListCountries(w http.ResponseWriter, r *http.Request) {
	countries, err := h.projection.ListCountries(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list countries")
		return
	}

	writeJSON(w, http.StatusOK, countries)
}

// GetCountry returns a country by ID
// GET /api/v1/geo/countries/{id}
func (h *GeoHandler) GetCountry(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid country id")
		return
	}

	country, err := h.projection.GetCountry(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "country not found")
		return
	}

	writeJSON(w, http.StatusOK, country)
}

// Countries that should show only cities with Cyrillic names
var cyrillicCountries = map[string]bool{
	"RU": true, // Russia
	"UA": true, // Ukraine
	"BY": true, // Belarus
	"KZ": true, // Kazakhstan
	"KG": true, // Kyrgyzstan
	"TJ": true, // Tajikistan
	"UZ": true, // Uzbekistan
	"TM": true, // Turkmenistan
	"AM": true, // Armenia
	"AZ": true, // Azerbaijan
	"GE": true, // Georgia
	"MD": true, // Moldova
}

// ListCities returns cities for a country with optional search
// GET /api/v1/geo/countries/{id}/cities?search=москва&limit=20
func (h *GeoHandler) ListCities(w http.ResponseWriter, r *http.Request) {
	countryID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid country id")
		return
	}

	// Parse query params
	search := r.URL.Query().Get("search")
	limit := 20
	const maxGeoLimit = 100
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= maxGeoLimit {
			limit = l
		}
	}

	// Check if this is a Cyrillic country - show only cities with Russian translations
	var opts *projections.SearchCitiesOptions
	country, err := h.projection.GetCountry(r.Context(), countryID)
	if err == nil && cyrillicCountries[country.ISO2] {
		opts = &projections.SearchCitiesOptions{
			OnlyWithTranslation: true,
		}
	}

	cities, err := h.projection.SearchCities(r.Context(), countryID, search, limit, opts)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to search cities")
		return
	}

	writeJSON(w, http.StatusOK, cities)
}

// GetCity returns a city by ID with country info
// GET /api/v1/geo/cities/{id}
func (h *GeoHandler) GetCity(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid city id")
		return
	}

	city, err := h.projection.GetCity(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "city not found")
		return
	}

	writeJSON(w, http.StatusOK, city)
}
