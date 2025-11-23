package utils

// DefaultValue returns def if val is the zero value of its type.
func DefaultValue[T comparable](val T, def T) T {
	var zero T
	if val == zero {
		return def
	}
	return val
}
