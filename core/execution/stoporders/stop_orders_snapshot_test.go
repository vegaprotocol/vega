package stoporders_test

import (
	"testing"

	"code.vegaprotocol.io/vega/core/execution/stoporders"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/libs/proto"
	"code.vegaprotocol.io/vega/logging"
	v1 "code.vegaprotocol.io/vega/protos/vega/snapshot/v1"
	"github.com/stretchr/testify/assert"
)

func TestStopOrdersSnapshot(t *testing.T) {
	log := logging.NewTestLogger()
	// create the pool add the price
	pool := stoporders.New(log)
	pool.PriceUpdated(num.NewUint(50))

	pool.Insert(newPricedStopOrder("a", "p1", "b", num.NewUint(40), types.StopOrderTriggerDirectionFallsBelow))
	pool.Insert(newTrailingStopOrder("b", "p1", "a", num.MustDecimalFromString("0.5"), types.StopOrderTriggerDirectionRisesAbove))

	// move the price a little
	pool.PriceUpdated(num.NewUint(51))
	pool.Insert(newPricedStopOrder("c", "p2", "d", num.NewUint(70), types.StopOrderTriggerDirectionRisesAbove))
	pool.Insert(newTrailingStopOrder("d", "p2", "c", num.MustDecimalFromString("0.5"), types.StopOrderTriggerDirectionFallsBelow))

	// move the price a little more
	pool.PriceUpdated(num.NewUint(52))
	pool.Insert(newPricedStopOrder("e", "p3", "f", num.NewUint(35), types.StopOrderTriggerDirectionFallsBelow))
	pool.Insert(newTrailingStopOrder("f", "p3", "e", num.MustDecimalFromString("0.5"), types.StopOrderTriggerDirectionRisesAbove))

	pool.Insert(newPricedStopOrder("h", "p2", "", num.NewUint(34), types.StopOrderTriggerDirectionFallsBelow))
	// same with new offset
	pool.Insert(newTrailingStopOrder("i", "p2", "", num.MustDecimalFromString("0.2"), types.StopOrderTriggerDirectionRisesAbove))

	// now we get the protos
	serialized := pool.ToProto()

	buf, err := proto.Marshal(serialized)
	assert.NoError(t, err)

	deserialized := &v1.StopOrders{}
	err = proto.Unmarshal(buf, deserialized)
	assert.NoError(t, err)

	pool2 := stoporders.NewFromProto(log, deserialized)
	assert.True(t, pool.Equal(pool2))
}
