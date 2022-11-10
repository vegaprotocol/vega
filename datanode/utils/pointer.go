package utils

func ToPtr[T any](val T) *T {
	return &val
}

func FromPtr[T any](ptr *T) (val T) {
	if ptr != nil {
		val = *ptr
	}
	return
}
