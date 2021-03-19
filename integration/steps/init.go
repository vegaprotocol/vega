package steps

var errCh chan error

func init() {
	errCh = make(chan error, 1)
}