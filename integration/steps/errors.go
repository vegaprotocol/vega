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