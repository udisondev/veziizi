package values

import "fmt"

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
	ADRClass    ADRClass    `json:"adr_class,omitempty"`
	Quantity    int         `json:"quantity"`
}

// Validate validates cargo info
func (c CargoInfo) Validate() error {
	if c.Quantity <= 0 {
		return fmt.Errorf("quantity must be greater than 0")
	}
	return nil
}
