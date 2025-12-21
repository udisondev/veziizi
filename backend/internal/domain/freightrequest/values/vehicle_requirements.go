package values

import "fmt"

// VehicleRequirements represents requirements for the transport vehicle
type VehicleRequirements struct {
	BodyTypes     []BodyType    `json:"body_types"`
	LoadingTypes  []LoadingType `json:"loading_types,omitempty"`
	Capacity      float64       `json:"capacity,omitempty"`
	Volume        float64       `json:"volume,omitempty"`
	Length        float64       `json:"length,omitempty"`
	Width         float64       `json:"width,omitempty"`
	Height        float64       `json:"height,omitempty"`
	RequiresADR   bool          `json:"requires_adr,omitempty"`
	Temperature   *Temperature  `json:"temperature,omitempty"`
}

// Validate validates vehicle requirements
func (v VehicleRequirements) Validate() error {
	if v.Temperature != nil {
		if err := v.Temperature.Validate(); err != nil {
			return fmt.Errorf("temperature: %w", err)
		}
	}
	return nil
}

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
