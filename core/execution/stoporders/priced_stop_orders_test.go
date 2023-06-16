package stoporders_test

import (
	"testing"

	"code.vegaprotocol.io/vega/core/execution/stoporders"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	"github.com/stretchr/testify/assert"
)

func TestPricedStopOrders(t *testing.T) {
	t.Run("remove", testPricedStopOrderRemove)
	t.Run("test trigger price both direction", testPricedStopOrderTriggerPriceBothDirection)
}

func testPricedStopOrderRemove(t *testing.T) {
	priced := stoporders.NewPricedStopOrders()

	// first try an empty one
	assert.EqualError(t, priced.Remove("a"), stoporders.ErrOrderNotFound.Error())

	// now insert one each direction
	priced.Insert("a", num.NewUint(10), types.StopOrderTriggerDirectionFallsBelow)
	priced.Insert("b", num.NewUint(10), types.StopOrderTriggerDirectionRisesAbove)
	priced.Insert("c", num.NewUint(11), types.StopOrderTriggerDirectionFallsBelow)

	// now remove some
	assert.NoError(t, priced.Remove("a"))

	// try again, it should fail
	assert.EqualError(t, priced.Remove("a"), stoporders.ErrOrderNotFound.Error())

	// now remove the last one
	assert.NoError(t, priced.Remove("b"))
	// try again, it should fail
	assert.EqualError(t, priced.Remove("b"), stoporders.ErrOrderNotFound.Error())
}

func testPricedStopOrderTriggerPriceBothDirection(t *testing.T) {
	priced := stoporders.NewPricedStopOrders()

	priced.Insert("a", num.NewUint(10), types.StopOrderTriggerDirectionFallsBelow)
	priced.Insert("b", num.NewUint(11), types.StopOrderTriggerDirectionFallsBelow)
	priced.Insert("c", num.NewUint(12), types.StopOrderTriggerDirectionFallsBelow)
	priced.Insert("d", num.NewUint(13), types.StopOrderTriggerDirectionFallsBelow)
	priced.Insert("e", num.NewUint(14), types.StopOrderTriggerDirectionFallsBelow)
	priced.Insert("f", num.NewUint(15), types.StopOrderTriggerDirectionFallsBelow)

	priced.Insert("a", num.NewUint(10), types.StopOrderTriggerDirectionRisesAbove)
	priced.Insert("b", num.NewUint(11), types.StopOrderTriggerDirectionRisesAbove)
	priced.Insert("c", num.NewUint(12), types.StopOrderTriggerDirectionRisesAbove)
	priced.Insert("d", num.NewUint(13), types.StopOrderTriggerDirectionRisesAbove)
	priced.Insert("e", num.NewUint(14), types.StopOrderTriggerDirectionRisesAbove)
	priced.Insert("f", num.NewUint(15), types.StopOrderTriggerDirectionRisesAbove)

	// Remove them once
	assert.EqualValues(t,
		priced.PriceUpdated(num.NewUint(13)),
		[]string{"f", "e", "a", "b", "c"},
	)

	// try again
	assert.EqualValues(t, priced.PriceUpdated(num.NewUint(13)), []string{})
}
