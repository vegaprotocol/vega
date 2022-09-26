package ptr

func From[T any](t T) *T {
	return &t
}
