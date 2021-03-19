package steps

import (
	"fmt"
	"time"
)

func TheSystemShouldReturnError(errorMessage string) error {
forLoop:
	for {
		select {
		case <-time.After(2 * time.Second):
			break forLoop
		case e := <-errCh:
			if e.Error() == errorMessage {
				return nil
			}
		}
	}

	return errNoErrorOccurred(errorMessage)
}

func errNoErrorOccurred(errorMessage string) error {
	return fmt.Errorf("error with message %s not found", errorMessage)
}
