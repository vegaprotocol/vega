package steps

import (
	"fmt"
	"strings"

	types "code.vegaprotocol.io/vega/proto"
)

func formatDiff(expected, got map[string]string) error {
	var expectedStr strings.Builder
	var gotStr strings.Builder
	formatStr := "\n\t%s(%s)"
	for name, value := range expected {
		_, _ = fmt.Fprintf(&expectedStr, formatStr, name, value)
		_, _ = fmt.Fprintf(&gotStr, formatStr, name, got[name])
	}

	return fmt.Errorf("\nexpected:%s\ngot:%s",
		expectedStr.String(),
		gotStr.String(),
	)
}

func errOrderNotFound(reference string, trader string, err error) error {
	return fmt.Errorf("order not found for trader(%s) with reference(%s): %v", trader, reference, err)
}

type CancelOrderError struct {
	reference string
	request   types.OrderCancellation
	Err       error
}

func (c CancelOrderError) Error() string {
	return fmt.Sprintf("failed to cancel order [%v] with reference %s: %v", c.request, c.reference, c.Err)
}

func (c *CancelOrderError) Unwrap() error { return c.Err }

type SubmitOrderError struct {
	reference string
	request   types.Order
	Err       error
}

func (s SubmitOrderError) Error() string {
	return fmt.Sprintf("failed to submit order [%v] with reference %s: %v", s.request, s.reference, s.Err)
}

func (s *SubmitOrderError) Unwrap() error { return s.Err }
