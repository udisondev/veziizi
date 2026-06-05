package transport

import "fmt"

// Temperature represents temperature requirements for refrigerated cargo
type Temperature struct {
	Min float64 `json:"min"`
	Max float64 `json:"max"`
}

// Validate validates temperature range
func (t Temperature) Validate() error {
	if t.Min > t.Max {
		return fmt.Errorf("min temperature cannot exceed max temperature")
	}
	return nil
}
