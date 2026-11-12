package ldutils

// Ptr is a pointer to a copy of the value, for any type.
func Ptr[T any](value T) *T {
	return &value
}

func IsValidString(s any) bool {
	_, isString := s.(string)
	return isString
}
