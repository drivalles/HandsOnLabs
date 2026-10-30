package utils

func PtrString(s string) *string {
	return &s
}

func PtrBool(s bool) *bool {
	return &s
}

func PtrInt(i int) *int {
	return &i
}

func PtrFloat32(i float32) *float32 {
	return &i
}

func PtrInt64(i int64) *int64 {
	return &i
}

func PtrUint(u uint) *uint {
	return &u
}

// Ptr is a pointer to a copy of the value, for any type.
func Ptr[T any](value T) *T {
	return &value
}

func IsValidString(s any) bool {
	_, isString := s.(string)
	return isString
}
