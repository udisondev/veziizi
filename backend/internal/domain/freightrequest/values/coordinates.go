package values

import "fmt"

// Coordinates represents geographical coordinates (latitude, longitude)
type Coordinates struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

func NewCoordinates(lat, lon float64) (Coordinates, error) {
	if lat < -90 || lat > 90 {
		return Coordinates{}, fmt.Errorf("invalid latitude: %f, must be between -90 and 90", lat)
	}
	if lon < -180 || lon > 180 {
		return Coordinates{}, fmt.Errorf("invalid longitude: %f, must be between -180 and 180", lon)
	}
	return Coordinates{Latitude: lat, Longitude: lon}, nil
}

func (c Coordinates) IsZero() bool {
	return c.Latitude == 0 && c.Longitude == 0
}

func (c Coordinates) String() string {
	return fmt.Sprintf("%.6f,%.6f", c.Latitude, c.Longitude)
}
