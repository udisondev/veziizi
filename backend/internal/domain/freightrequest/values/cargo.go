package values

// Dimensions represents cargo dimensions in meters
type Dimensions struct {
	Length float64 `json:"length"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}

func (d Dimensions) Volume() float64 {
	return d.Length * d.Width * d.Height
}

// CargoInfo represents information about the cargo
type CargoInfo struct {
	Description string      `json:"description"`
	Weight      float64     `json:"weight"`
	Volume      float64     `json:"volume,omitempty"`
	Dimensions  *Dimensions `json:"dimensions,omitempty"`
	Type        CargoType   `json:"type"`
	ADRClass    ADRClass    `json:"adr_class,omitempty"`
	Quantity    int         `json:"quantity,omitempty"`
}
