package steps

import "fmt"

func errUnableToPlaceOrder(trader, reference string, err error) error {
	return fmt.Errorf("unable to place order for trader(%s) and reference(%s): %s",
		trader, reference, err.Error(),
	)
}
