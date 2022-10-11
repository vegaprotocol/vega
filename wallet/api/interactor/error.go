package interactor

import "fmt"

func TraceIDMismatchError(expected, got string) error {
	return fmt.Errorf("the trace IDs between the request and the response mismatch: expected %q, got %q", expected, got)
}

func WrongResponseTypeError(expected, got InteractionName) error {
	return fmt.Errorf("the received response does not match the expected response type: expected %q, got %q", string(expected), string(got))
}

func InvalidResponsePayloadError(name InteractionName) error {
	return fmt.Errorf("the received response has not a valid %q payload", string(name))
}
