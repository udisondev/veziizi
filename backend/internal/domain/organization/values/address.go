package values

// Address represents organization address as free-form string
// TODO: integrate with DaData or Google Places for validation/autocomplete
type Address string

func (a Address) String() string {
	return string(a)
}
