package ptr

func From[T any](t T) *T {
	return &t
}

func UnBox[T any](v *T) (out T) {
	if v != nil {
		return *v
	}
	return
}
