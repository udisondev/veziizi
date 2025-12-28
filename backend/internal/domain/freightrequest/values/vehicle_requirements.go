package values

import "fmt"

// VehicleRequirements represents requirements for the transport vehicle
type VehicleRequirements struct {
	VehicleType    VehicleType    `json:"vehicle_type"`
	VehicleSubType VehicleSubType `json:"vehicle_subtype"`
	LoadingTypes   []LoadingType  `json:"loading_types,omitempty"`
	Capacity       float64        `json:"capacity,omitempty"`
	Volume         float64        `json:"volume,omitempty"`
	Length         float64        `json:"length,omitempty"`
	Width          float64        `json:"width,omitempty"`
	Height         float64        `json:"height,omitempty"`
	RequiresADR    bool           `json:"requires_adr,omitempty"`
	Temperature    *Temperature   `json:"temperature,omitempty"`
	Thermograph    bool           `json:"thermograph,omitempty"` // устройство фиксации температуры в пути
}

// Validate validates vehicle requirements
func (v VehicleRequirements) Validate() error {
	if v.VehicleType == "" {
		return fmt.Errorf("vehicle_type is required")
	}
	if !v.VehicleType.IsValid() {
		return fmt.Errorf("invalid vehicle_type: %s", v.VehicleType)
	}

	if v.VehicleSubType == "" {
		return fmt.Errorf("vehicle_subtype is required")
	}
	if !v.VehicleSubType.IsValid() {
		return fmt.Errorf("invalid vehicle_subtype: %s", v.VehicleSubType)
	}

	if !IsValidSubtypeForType(v.VehicleType, v.VehicleSubType) {
		return fmt.Errorf("vehicle_subtype %s is not valid for vehicle_type %s", v.VehicleSubType, v.VehicleType)
	}

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
