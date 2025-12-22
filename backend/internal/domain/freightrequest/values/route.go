package values

import "fmt"

// RoutePoint represents a point in the route (can be loading, unloading, or both)
type RoutePoint struct {
	IsLoading   bool `json:"is_loading"`
	IsUnloading bool `json:"is_unloading"`

	// Structured location (new)
	CountryID *int `json:"country_id,omitempty"` // ID from geo_countries
	CityID    *int `json:"city_id,omitempty"`    // ID from geo_cities

	// Legacy address field (kept for backward compatibility with old events)
	Address     string       `json:"address"`
	Coordinates *Coordinates `json:"coordinates,omitempty"`

	DateFrom     string  `json:"date_from"`           // YYYY-MM-DD format
	DateTo       *string `json:"date_to,omitempty"`   // YYYY-MM-DD format
	TimeFrom     *string `json:"time_from,omitempty"` // HH:mm format
	TimeTo       *string `json:"time_to,omitempty"`   // HH:mm format
	ContactName  *string `json:"contact_name,omitempty"`
	ContactPhone *string `json:"contact_phone,omitempty"`
	Comment      *string `json:"comment,omitempty"`
}

// HasStructuredLocation returns true if the point has structured location (country_id + city_id)
func (p RoutePoint) HasStructuredLocation() bool {
	return p.CountryID != nil && p.CityID != nil
}

// Validate validates route point - if contact is provided, both name and phone are required
func (p RoutePoint) Validate() error {
	hasName := p.ContactName != nil && *p.ContactName != ""
	hasPhone := p.ContactPhone != nil && *p.ContactPhone != ""

	if hasName && !hasPhone {
		return fmt.Errorf("contact phone is required when contact name is provided")
	}
	if hasPhone && !hasName {
		return fmt.Errorf("contact name is required when contact phone is provided")
	}
	return nil
}

// Route represents the full route with loading and unloading points
type Route struct {
	Points []RoutePoint `json:"points"`
}

func NewRoute(points []RoutePoint) (Route, error) {
	if len(points) < 2 {
		return Route{}, fmt.Errorf("route must have at least 2 points (loading and unloading)")
	}

	hasLoading := false
	hasUnloading := false
	for i, p := range points {
		if p.IsLoading {
			hasLoading = true
		}
		if p.IsUnloading {
			hasUnloading = true
		}
		if err := p.Validate(); err != nil {
			return Route{}, fmt.Errorf("point %d: %w", i+1, err)
		}
	}

	if !hasLoading {
		return Route{}, fmt.Errorf("route must have at least one loading point")
	}
	if !hasUnloading {
		return Route{}, fmt.Errorf("route must have at least one unloading point")
	}

	return Route{Points: points}, nil
}
