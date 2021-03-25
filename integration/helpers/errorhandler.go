package helpers

type ErrorHandler struct {
	errCh chan error
}

func NewErrorHandler() *ErrorHandler {
	return &ErrorHandler{
		errCh: make(chan error, 100),
	}
}

func (h ErrorHandler) HandleError(err error) {
	h.errCh <- err
}

func (h ErrorHandler) ErrCh() <-chan error {
	return h.errCh
}
