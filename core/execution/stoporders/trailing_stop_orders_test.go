package stoporders_test

import (
	"testing"

	"code.vegaprotocol.io/vega/core/execution/stoporders"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	"github.com/stretchr/testify/assert"
)

// - A trailing stop order for a 5% drop placed when the price is `50`, followed by a
// price rise to `60` will:
//   - Be triggered by a fall to `57`. (<a name="0014-ORDT-027"
//
// href="#0014-ORDT-027">0014-ORDT-027</a>)
//   - Not be triggered by a fall to `58`. (<a name="0014-ORDT-036"
//
// href="#0014-ORDT-036">0014-ORDT-036</a>)
func TestTrailingAC_0014_ORDT_027_036(t *testing.T) {
	trailing := stoporders.NewTrailingStopOrders()

	// initial price
	trailing.PriceUpdated(num.NewUint(50))
	trailing.Insert("a", num.DecimalFromFloat(0.05), types.StopOrderTriggerDirectionFallsBelow)

	// price move
	trailing.PriceUpdated(num.NewUint(60))

	atPrice, offset, ok := trailing.Exists("a")
	assert.True(t, ok)
	assert.Equal(t, atPrice, num.NewUint(60))
	assert.Equal(t, offset, num.DecimalFromFloat(0.05))

	// move price to 58, nothing happen
	affectedOrders := trailing.PriceUpdated(num.NewUint(58))
	assert.Len(t, affectedOrders, 0)

	affectedOrders = trailing.PriceUpdated(num.NewUint(57))
	assert.Len(t, affectedOrders, 1)

	assert.Equal(t, trailing.Len(types.StopOrderTriggerDirectionFallsBelow), 0)
}

// A trailing stop order for a 5% rise placed when the price is `50`, followed by a drop
// to `40` will:
//   - Be triggered by a rise to `42`. (<a name="0014-ORDT-028"
//
// href="#0014-ORDT-028">0014-ORDT-028</a>)
//   - Not be triggered by a rise to `41`. (<a name="0014-ORDT-029"
//
// href="#0014-ORDT-029">0014-ORDT-029</a>)
func TestTrailingAC_0014_ORDT_827_029(t *testing.T) {
	trailing := stoporders.NewTrailingStopOrders()

	// initial price
	trailing.PriceUpdated(num.NewUint(50))
	trailing.Insert("a", num.DecimalFromFloat(0.05), types.StopOrderTriggerDirectionRisesAbove)

	// price move
	trailing.PriceUpdated(num.NewUint(40))

	atPrice, offset, ok := trailing.Exists("a")
	assert.True(t, ok)
	assert.Equal(t, atPrice, num.NewUint(40))
	assert.Equal(t, offset, num.DecimalFromFloat(0.05))

	// move price to 41, nothing happen
	affectedOrders := trailing.PriceUpdated(num.NewUint(41))
	assert.Len(t, affectedOrders, 0)

	affectedOrders = trailing.PriceUpdated(num.NewUint(42))
	assert.Len(t, affectedOrders, 1)

	assert.Equal(t, trailing.Len(types.StopOrderTriggerDirectionRisesAbove), 0)
}

// A trailing stop order for a 25% drop placed when the price is `50`, followed by a
// price rise to `60`, then to `50`, then another rise to `57` will:
//   - Be triggered by a fall to `45`. (<a name="0014-ORDT-030"
//
// href="#0014-ORDT-030">0014-ORDT-030</a>)
//   - Not be triggered by a fall to `46`. (<a name="0014-ORDT-031"
//
// href="#0014-ORDT-031">0014-ORDT-031</a>)
func TestTrailingAC_0014_ORDT_030_031(t *testing.T) {
	trailing := stoporders.NewTrailingStopOrders()

	// initial price
	trailing.PriceUpdated(num.NewUint(50))
	trailing.Insert("a", num.DecimalFromFloat(0.25), types.StopOrderTriggerDirectionFallsBelow)

	// price move
	affectedOrders := trailing.PriceUpdated(num.NewUint(60))
	assert.Len(t, affectedOrders, 0)

	affectedOrders = trailing.PriceUpdated(num.NewUint(50))
	assert.Len(t, affectedOrders, 0)

	affectedOrders = trailing.PriceUpdated(num.NewUint(57))
	assert.Len(t, affectedOrders, 0)

	atPrice, offset, ok := trailing.Exists("a")
	assert.True(t, ok)
	assert.Equal(t, atPrice, num.NewUint(60))
	assert.Equal(t, offset, num.DecimalFromFloat(0.25))

	// move price to 46, nothing happen
	affectedOrders = trailing.PriceUpdated(num.NewUint(46))
	assert.Len(t, affectedOrders, 0)

	affectedOrders = trailing.PriceUpdated(num.NewUint(45))
	assert.Len(t, affectedOrders, 1)

	assert.Equal(t, trailing.Len(types.StopOrderTriggerDirectionFallsBelow), 0)
}

func TestTrailingStopOrdersMultipleOffsetPerPrice(t *testing.T) {
	trailing := stoporders.NewTrailingStopOrders()

	// initial price
	trailing.PriceUpdated(num.NewUint(50))
	// won't trigger unless it goes bellow 50%
	trailing.Insert("a", num.DecimalFromFloat(0.50), types.StopOrderTriggerDirectionFallsBelow)

	trailing.PriceUpdated(num.NewUint(40))
	// won't trigger unless it goes bellow 50%
	trailing.Insert("b", num.DecimalFromFloat(0.10), types.StopOrderTriggerDirectionFallsBelow)

	// as of no they should be in 2 differen buckets

	atPrice, offset, ok := trailing.Exists("a")
	assert.True(t, ok)
	assert.Equal(t, atPrice, num.NewUint(50))
	assert.Equal(t, offset, num.DecimalFromFloat(0.50))

	atPrice, offset, ok = trailing.Exists("b")
	assert.True(t, ok)
	assert.Equal(t, atPrice, num.NewUint(40))
	assert.Equal(t, offset, num.DecimalFromFloat(0.10))

	affectedOrders := trailing.PriceUpdated(num.NewUint(45))
	assert.Len(t, affectedOrders, 0)

	// ensure a is still in the same bucked
	// b moved to 45

	atPrice, offset, ok = trailing.Exists("a")
	assert.True(t, ok)
	assert.Equal(t, atPrice, num.NewUint(50))
	assert.Equal(t, offset, num.DecimalFromFloat(0.50))

	atPrice, offset, ok = trailing.Exists("b")
	assert.True(t, ok)
	assert.Equal(t, atPrice, num.NewUint(45))
	assert.Equal(t, offset, num.DecimalFromFloat(0.10))

	affectedOrders = trailing.PriceUpdated(num.NewUint(60))
	assert.Len(t, affectedOrders, 0)

	// ensure they are in the same buckets

	atPrice, offset, ok = trailing.Exists("a")
	assert.True(t, ok)
	assert.Equal(t, atPrice, num.NewUint(60))
	assert.Equal(t, offset, num.DecimalFromFloat(0.50))

	atPrice, offset, ok = trailing.Exists("b")
	assert.True(t, ok)
	assert.Equal(t, atPrice, num.NewUint(60))
	assert.Equal(t, offset, num.DecimalFromFloat(0.10))

	// now move prices so b triggers
	affectedOrders = trailing.PriceUpdated(num.NewUint(54))
	assert.Len(t, affectedOrders, 1)
	assert.Equal(t, affectedOrders[0], "b")

	atPrice, offset, ok = trailing.Exists("a")
	assert.True(t, ok)
	assert.Equal(t, atPrice, num.NewUint(60))
	assert.Equal(t, offset, num.DecimalFromFloat(0.50))

	_, _, ok = trailing.Exists("b")
	assert.False(t, ok)

}
