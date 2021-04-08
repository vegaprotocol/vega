package steps

import (
	"fmt"
	"time"

	"code.vegaprotocol.io/vega/integration/helpers"
)

func TheSystemShouldReturnError(errorHandler *helpers.ErrorHandler, errorMessage string) error {
forLoop:
	for {
		select {
		case <-time.After(2 * time.Second):
			break forLoop
		case e := <-errorHandler.ErrCh():
			switch cause := e.(type) {
			case CancelOrderError:
				if cause.Err.Error() == errorMessage {
					return nil
				}
			case SubmitOrderError:
				if cause.Err.Error() == errorMessage {
					return nil
				}
			default:
			}
		}
	}

	return errNoErrorOccurred(errorMessage)
}

func errNoErrorOccurred(errorMessage string) error {
	return fmt.Errorf("error with message %s not found", errorMessage)
}
