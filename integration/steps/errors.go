package steps

import (
	"fmt"

	types "code.vegaprotocol.io/vega/proto"
)

func errOrderNotFound(reference string, trader string, err error) error {
	return fmt.Errorf("order not found for trader(%s) with reference(%s): %v", trader, reference, err)
}

func errAmendingOrder(o types.Order, err error) error {
	return fmt.Errorf("failed to amend order  for trader(%s) with reference(%s): %v", o.PartyId, o.Reference, err)
}

type CancelOrderError struct {
	reference string
	request   types.OrderCancellation
	Err       error
}

func (c CancelOrderError) Error() string {
	return fmt.Sprintf("failed to cancel order [%v] with reference %s: %v", c.request, c.reference, c.Err)
}

type SubmitOrderError struct {
	reference string
	request   types.Order
	Err       error
}

func (s SubmitOrderError) Error() string {
	return fmt.Sprintf("failed to submit order [%v] with reference %s: %v", s.request, s.reference, s.Err)
}
