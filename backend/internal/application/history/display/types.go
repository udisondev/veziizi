package display

// DisplayField представляет поле для отображения
type DisplayField struct {
	Label string `json:"label"`
	Value string `json:"value"`
	Type  string `json:"type,omitempty"` // text, money, date, status
}

// DiffValue представляет изменение значения
type DiffValue struct {
	Label    string `json:"label"`
	OldValue string `json:"old_value"`
	NewValue string `json:"new_value"`
}

// DisplayView представляет человекочитаемое отображение события
type DisplayView struct {
	Title       string         `json:"title"`
	Description string         `json:"description"`
	Fields      []DisplayField `json:"fields,omitempty"`
	Diffs       []DiffValue    `json:"diffs,omitempty"`
	Icon        string         `json:"icon,omitempty"`
	Severity    string         `json:"severity,omitempty"` // info, success, warning, error
}

// NewDisplayView создаёт DisplayView с заданными title и description
func NewDisplayView(title, description string) DisplayView {
	return DisplayView{
		Title:       title,
		Description: description,
		Severity:    "info",
	}
}

// WithIcon добавляет иконку
func (v DisplayView) WithIcon(icon string) DisplayView {
	v.Icon = icon
	return v
}

// WithSeverity устанавливает severity
func (v DisplayView) WithSeverity(severity string) DisplayView {
	v.Severity = severity
	return v
}

// AddField добавляет поле
func (v *DisplayView) AddField(label, value string) {
	v.Fields = append(v.Fields, DisplayField{
		Label: label,
		Value: value,
		Type:  "text",
	})
}

// AddFieldWithType добавляет поле с типом
func (v *DisplayView) AddFieldWithType(label, value, fieldType string) {
	v.Fields = append(v.Fields, DisplayField{
		Label: label,
		Value: value,
		Type:  fieldType,
	})
}

// AddDiff добавляет diff
func (v *DisplayView) AddDiff(label, oldValue, newValue string) {
	v.Diffs = append(v.Diffs, DiffValue{
		Label:    label,
		OldValue: oldValue,
		NewValue: newValue,
	})
}
