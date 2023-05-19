package stoporders_test

import (
	"fmt"
	"testing"

	"code.vegaprotocol.io/vega/core/execution/stoporders"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
)

func TestTrailingStopOrders(t *testing.T) {
	trailing := stoporders.NewTrailingStopOrders()

	trailing.PriceUpdated(num.NewUint(1000))
	trailing.Insert("a", num.DecimalFromFloat(0.1), types.StopOrderTriggerDirectionFallsBelow)
	trailing.Insert("b", num.DecimalFromFloat(0.09), types.StopOrderTriggerDirectionFallsBelow)
	trailing.Insert("c", num.DecimalFromFloat(0.2), types.StopOrderTriggerDirectionFallsBelow)

	trailing.PriceUpdated(num.NewUint(999))
	trailing.Insert("e", num.DecimalFromFloat(0.1), types.StopOrderTriggerDirectionFallsBelow)
	trailing.Insert("f", num.DecimalFromFloat(0.1), types.StopOrderTriggerDirectionFallsBelow)

	fmt.Printf("%s", trailing.DumpFallsBelow())
}
