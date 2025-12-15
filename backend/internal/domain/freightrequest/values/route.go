package values

import "fmt"

// RoutePoint represents a point in the route (can be loading, unloading, or both)
type RoutePoint struct {
	IsLoading    bool         `json:"is_loading"`
	IsUnloading  bool         `json:"is_unloading"`
	Address      string       `json:"address"`
	Coordinates  *Coordinates `json:"coordinates,omitempty"`
	DateFrom     string       `json:"date_from"`           // YYYY-MM-DD format
	DateTo       *string      `json:"date_to,omitempty"`   // YYYY-MM-DD format
	TimeFrom     *string      `json:"time_from,omitempty"` // HH:mm format
	TimeTo       *string      `json:"time_to,omitempty"`   // HH:mm format
	ContactName  *string      `json:"contact_name,omitempty"`
	ContactPhone *string      `json:"contact_phone,omitempty"`
	Comment      *string      `json:"comment,omitempty"`
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
	for _, p := range points {
		if p.IsLoading {
			hasLoading = true
		}
		if p.IsUnloading {
			hasUnloading = true
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
