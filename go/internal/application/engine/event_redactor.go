package engine

// EventRedactor masks sensitive values in event data before dispatch to sinks.
type EventRedactor struct{}

// NewEventRedactor creates a new EventRedactor.
func NewEventRedactor() *EventRedactor {
	return &EventRedactor{}
}

// MaskValue masks a single string value.
// Strings longer than 8 chars show first 3 + "***" + last 3.
// Shorter strings are fully masked as "***".
func (r *EventRedactor) MaskValue(s string) string {
	if len(s) > 8 {
		return s[:3] + "***" + s[len(s)-3:]
	}
	return "***"
}

// RedactMap masks all string values in a map recursively.
// Non-string values (numbers, bools) are preserved.
func (r *EventRedactor) RedactMap(m map[string]any) map[string]any {
	if m == nil {
		return nil
	}
	result := make(map[string]any, len(m))
	for k, v := range m {
		switch val := v.(type) {
		case string:
			result[k] = r.MaskValue(val)
		case map[string]any:
			result[k] = r.RedactMap(val)
		default:
			result[k] = v
		}
	}
	return result
}
