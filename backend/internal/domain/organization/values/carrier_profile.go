package values

// CarrierProfile contains information for organizations that can be carriers
type CarrierProfile struct {
	Description   string   `json:"description,omitempty"`
	VehicleTypes  []string `json:"vehicle_types,omitempty"`
	Regions       []string `json:"regions,omitempty"`
	HasADR        bool     `json:"has_adr"`
	HasRefrigerator bool   `json:"has_refrigerator"`
}

func NewCarrierProfile() CarrierProfile {
	return CarrierProfile{}
}

func (p CarrierProfile) WithDescription(desc string) CarrierProfile {
	p.Description = desc
	return p
}

func (p CarrierProfile) WithVehicleTypes(types []string) CarrierProfile {
	p.VehicleTypes = types
	return p
}

func (p CarrierProfile) WithRegions(regions []string) CarrierProfile {
	p.Regions = regions
	return p
}

func (p CarrierProfile) WithADR(hasADR bool) CarrierProfile {
	p.HasADR = hasADR
	return p
}

func (p CarrierProfile) WithRefrigerator(has bool) CarrierProfile {
	p.HasRefrigerator = has
	return p
}
