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
	trailing.Insert("a", num.DecimalFromFloat(0.1), types.StopOrderTriggerDirectionRisesAbove)
	trailing.Insert("b", num.DecimalFromFloat(0.09), types.StopOrderTriggerDirectionRisesAbove)
	trailing.Insert("c", num.DecimalFromFloat(0.2), types.StopOrderTriggerDirectionRisesAbove)

	trailing.PriceUpdated(num.NewUint(999))
	trailing.Insert("e", num.DecimalFromFloat(0.1), types.StopOrderTriggerDirectionRisesAbove)
	trailing.Insert("f", num.DecimalFromFloat(0.1), types.StopOrderTriggerDirectionRisesAbove)

	fmt.Printf("%s", trailing.DumpFallsBelow())
}

func TestTrailingStopOrders2(t *testing.T) {
	trailing := stoporders.NewTrailingStopOrders()

	trailing.PriceUpdated(num.NewUint(1000))
	trailing.Insert("a", num.DecimalFromFloat(0.1), types.StopOrderTriggerDirectionRisesAbove)

	fmt.Printf("%s\n", trailing.DumpRisesAbove())

	trailing.PriceUpdated(num.NewUint(950))
	fmt.Printf("%s\n", trailing.DumpRisesAbove())

	trailing.PriceUpdated(num.NewUint(1020))
	fmt.Printf("%s\n", trailing.DumpRisesAbove())

	trailing.PriceUpdated(num.NewUint(930))
	fmt.Printf("%s\n", trailing.DumpRisesAbove())

	trailing.PriceUpdated(num.NewUint(1000))
	fmt.Printf("%s\n", trailing.DumpRisesAbove())

	trailing.PriceUpdated(num.NewUint(920))
	fmt.Printf("%s\n", trailing.DumpRisesAbove())

	trailing.PriceUpdated(num.NewUint(960))
	fmt.Printf("%s\n", trailing.DumpRisesAbove())

	trailing.PriceUpdated(num.NewUint(1100))
	fmt.Printf("%s\n", trailing.DumpRisesAbove())
}

func TestTrailingStopOrders3(t *testing.T) {
	trailing := stoporders.NewTrailingStopOrders()

	trailing.PriceUpdated(num.NewUint(1000))
	trailing.Insert("a", num.DecimalFromFloat(0.1), types.StopOrderTriggerDirectionFallsBelow)

	fmt.Printf("%s\n", trailing.DumpFallsBelow())

	trailing.PriceUpdated(num.NewUint(1050))
	fmt.Printf("%s\n", trailing.DumpFallsBelow())

	trailing.PriceUpdated(num.NewUint(980))
	fmt.Printf("%s\n", trailing.DumpFallsBelow())

	trailing.PriceUpdated(num.NewUint(1060))
	fmt.Printf("%s\n", trailing.DumpFallsBelow())

	trailing.PriceUpdated(num.NewUint(1000))
	fmt.Printf("%s\n", trailing.DumpFallsBelow())

	trailing.PriceUpdated(num.NewUint(1070))
	fmt.Printf("%s\n", trailing.DumpFallsBelow())

	trailing.PriceUpdated(num.NewUint(1050))
	fmt.Printf("%s\n", trailing.DumpFallsBelow())

	trailing.PriceUpdated(num.NewUint(900))
	fmt.Printf("%s\n", trailing.DumpFallsBelow())
}

func TestTrailingStopOrders4(t *testing.T) {
	trailing := stoporders.NewTrailingStopOrders()

	trailing.PriceUpdated(num.NewUint(930))
	trailing.Insert("a", num.DecimalFromFloat(0.3), types.StopOrderTriggerDirectionRisesAbove)

	trailing.PriceUpdated(num.NewUint(1000))
	trailing.Insert("b", num.DecimalFromFloat(0.2), types.StopOrderTriggerDirectionRisesAbove)

	trailing.PriceUpdated(num.NewUint(1050))
	trailing.Insert("c", num.DecimalFromFloat(0.10), types.StopOrderTriggerDirectionRisesAbove)

	trailing.Insert("d", num.DecimalFromFloat(0.15), types.StopOrderTriggerDirectionRisesAbove)

	trailing.PriceUpdated(num.NewUint(1100))
	trailing.Insert("e", num.DecimalFromFloat(0.05), types.StopOrderTriggerDirectionRisesAbove)

	trailing.PriceUpdated(num.NewUint(1101))
	trailing.PriceUpdated(num.NewUint(1170))

	fmt.Printf("%s\n", trailing.DumpRisesAbove())
}
