package stoporders_test

import (
	"testing"
	"time"

	"code.vegaprotocol.io/vega/core/execution/stoporders"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/logging"

	"github.com/stretchr/testify/assert"
)

func TestSingleStopOrders(t *testing.T) {
	pool := stoporders.New(logging.NewTestLogger())

	pool.PriceUpdated(num.NewUint(50))

	// this is going to trigger when going up 60
	pool.Insert(newPricedStopOrder("a", "p1", "", num.NewUint(40), types.StopOrderTriggerDirectionFallsBelow))
	pool.Insert(newPricedStopOrder("b", "p1", "", num.NewUint(57), types.StopOrderTriggerDirectionRisesAbove))

	// this will be triggered when going from 60 to 57, and triggre the falls below
	pool.Insert(newTrailingStopOrder("c", "p2", "", num.MustDecimalFromString("0.05"), types.StopOrderTriggerDirectionFallsBelow))
	pool.Insert(newTrailingStopOrder("d", "p2", "", num.MustDecimalFromString("0.5"), types.StopOrderTriggerDirectionRisesAbove))

	// mixing around both, will be triggered by the end
	pool.Insert(newPricedStopOrder("e", "p2", "", num.NewUint(40), types.StopOrderTriggerDirectionFallsBelow))
	pool.Insert(newTrailingStopOrder("f", "p2", "", num.MustDecimalFromString("0.5"), types.StopOrderTriggerDirectionRisesAbove))

	assert.Equal(t, pool.Len(), 6)

	// move the price a little, nothing should happen.
	triggeredOrders, cancelledOrders := pool.PriceUpdated(num.NewUint(55))
	assert.Len(t, triggeredOrders, 0)
	assert.Len(t, cancelledOrders, 0)

	t.Run("price move triggers priced stop order", func(t *testing.T) {
		// move the price so the priced stop order is triggered, this should return both
		triggeredOrders, cancelledOrders = pool.PriceUpdated(num.NewUint(60))
		assert.Len(t, triggeredOrders, 1)
		assert.Len(t, cancelledOrders, 0)
		assert.Equal(t, pool.Len(), 5)
		assert.Equal(t, triggeredOrders[0].Status, types.StopOrderStatusTriggered)
		assert.Equal(t, triggeredOrders[0].ID, "b")
	})

	t.Run("trying to remove a triggered order returns an error", func(t *testing.T) {
		// try to remove it now. no errors, the party have no orders anymore
		affectedOrders, err := pool.Cancel("p1", "b")
		assert.Len(t, affectedOrders, 0)
		assert.EqualError(t, err, "stop order not found")
	})

	t.Run("price update trigger trailing order", func(t *testing.T) {
		// move the price so the trailing order get triggered
		triggeredOrders, cancelledOrders = pool.PriceUpdated(num.NewUint(57))
		assert.Len(t, triggeredOrders, 1)
		assert.Len(t, cancelledOrders, 0)
		assert.Equal(t, pool.Len(), 4)
		assert.Equal(t, triggeredOrders[0].Status, types.StopOrderStatusTriggered)
		assert.Equal(t, triggeredOrders[0].ID, "c")
	})

	t.Run("trying to remove a triggered order returns an error", func(t *testing.T) {
		// try to remove it now. no errors, the party have no orders anymore
		affectedOrders, err := pool.Cancel("p2", "c")
		assert.Len(t, affectedOrders, 0)
		assert.EqualError(t, err, "stop order not found")
	})

	t.Run("price update trigger trailing order/priced order", func(t *testing.T) {
		// move the price so the trailing order get triggered
		triggeredOrders, cancelledOrders = pool.PriceUpdated(num.NewUint(75))
		assert.Len(t, triggeredOrders, 2)
		assert.Len(t, cancelledOrders, 0)
		assert.Equal(t, pool.Len(), 2)
		assert.Equal(t, triggeredOrders[0].Status, types.StopOrderStatusTriggered)
		assert.Equal(t, triggeredOrders[0].ID, "d")
		assert.Equal(t, triggeredOrders[1].Status, types.StopOrderStatusTriggered)
		assert.Equal(t, triggeredOrders[1].ID, "f")
	})
}

func TestCancelStopOrders(t *testing.T) {
	pool := stoporders.New(logging.NewTestLogger())

	pool.PriceUpdated(num.NewUint(50))

	pool.Insert(newPricedStopOrder("a", "p1", "", num.NewUint(40), types.StopOrderTriggerDirectionFallsBelow))
	pool.Insert(newPricedStopOrder("b", "p1", "", num.NewUint(57), types.StopOrderTriggerDirectionRisesAbove))

	pool.Insert(newTrailingStopOrder("c", "p2", "", num.MustDecimalFromString("0.05"), types.StopOrderTriggerDirectionFallsBelow))
	pool.Insert(newTrailingStopOrder("d", "p2", "", num.MustDecimalFromString("0.5"), types.StopOrderTriggerDirectionRisesAbove))

	pool.Insert(newPricedStopOrder("e", "p2", "f", num.NewUint(40), types.StopOrderTriggerDirectionFallsBelow))
	pool.Insert(newTrailingStopOrder("f", "p2", "e", num.MustDecimalFromString("0.5"), types.StopOrderTriggerDirectionRisesAbove))

	pool.Insert(newPricedStopOrder("h", "p2", "i", num.NewUint(40), types.StopOrderTriggerDirectionFallsBelow))
	pool.Insert(newTrailingStopOrder("i", "p2", "h", num.MustDecimalFromString("0.5"), types.StopOrderTriggerDirectionRisesAbove))

	// a party with no order returns no error
	affectedOrders, err := pool.Cancel("p3", "")
	assert.NoError(t, err)
	assert.Len(t, affectedOrders, 0)

	// remove one order, not OCO
	affectedOrders, err = pool.Cancel("p1", "b")
	assert.NoError(t, err)
	assert.Len(t, affectedOrders, 1)
	assert.Equal(t, affectedOrders[0].ID, "b")
	assert.Equal(t, affectedOrders[0].Status, types.StopOrderStatusCancelled)
	assert.Equal(t, 7, pool.Len())

	// remove one order, OCO, to returned
	affectedOrders, err = pool.Cancel("p2", "f")
	assert.NoError(t, err)
	assert.Len(t, affectedOrders, 2)
	assert.Equal(t, affectedOrders[0].ID, "f")
	assert.Equal(t, affectedOrders[0].Status, types.StopOrderStatusCancelled)
	assert.Equal(t, affectedOrders[1].ID, "e")
	assert.Equal(t, affectedOrders[1].Status, types.StopOrderStatusCancelled)
	assert.Equal(t, 5, pool.Len())

	// remove all for party
	affectedOrders, err = pool.Cancel("p2", "")
	assert.NoError(t, err)
	assert.Len(t, affectedOrders, 4)
	assert.Equal(t, affectedOrders[0].ID, "c")
	assert.Equal(t, affectedOrders[0].Status, types.StopOrderStatusCancelled)
	assert.Equal(t, affectedOrders[1].ID, "d")
	assert.Equal(t, affectedOrders[1].Status, types.StopOrderStatusCancelled)
	assert.Equal(t, affectedOrders[2].ID, "h")
	assert.Equal(t, affectedOrders[2].Status, types.StopOrderStatusCancelled)
	assert.Equal(t, affectedOrders[3].ID, "i")
	assert.Equal(t, affectedOrders[3].Status, types.StopOrderStatusCancelled)
	assert.Equal(t, 1, pool.Len())

	// ensure the actual trees are cleaned up
	assert.Equal(t, 0, pool.Trailing().Len(types.StopOrderTriggerDirectionFallsBelow))
	assert.Equal(t, 0, pool.Trailing().Len(types.StopOrderTriggerDirectionRisesAbove))
	assert.Equal(t, 1, pool.Priced().Len(types.StopOrderTriggerDirectionFallsBelow))
	assert.Equal(t, 0, pool.Priced().Len(types.StopOrderTriggerDirectionRisesAbove))
}

func TestRemoveExpiredStopOrders(t *testing.T) {
	pool := stoporders.New(logging.NewTestLogger())

	pool.PriceUpdated(num.NewUint(50))

	pool.Insert(newPricedStopOrder("a", "p1", "", num.NewUint(40), types.StopOrderTriggerDirectionFallsBelow))
	pool.Insert(newPricedStopOrder("b", "p1", "", num.NewUint(57), types.StopOrderTriggerDirectionRisesAbove))

	pool.Insert(newTrailingStopOrder("c", "p2", "", num.MustDecimalFromString("0.05"), types.StopOrderTriggerDirectionFallsBelow))
	pool.Insert(newTrailingStopOrder("d", "p2", "", num.MustDecimalFromString("0.5"), types.StopOrderTriggerDirectionRisesAbove))

	pool.Insert(newPricedStopOrder("e", "p2", "f", num.NewUint(40), types.StopOrderTriggerDirectionFallsBelow))
	pool.Insert(newTrailingStopOrder("f", "p2", "e", num.MustDecimalFromString("0.5"), types.StopOrderTriggerDirectionRisesAbove))

	pool.Insert(newPricedStopOrder("h", "p2", "i", num.NewUint(40), types.StopOrderTriggerDirectionFallsBelow))
	pool.Insert(newTrailingStopOrder("i", "p2", "h", num.MustDecimalFromString("0.5"), types.StopOrderTriggerDirectionRisesAbove))

	assert.Equal(t, 1, pool.Trailing().Len(types.StopOrderTriggerDirectionFallsBelow))
	assert.Equal(t, 1, pool.Trailing().Len(types.StopOrderTriggerDirectionRisesAbove))
	assert.Equal(t, 1, pool.Priced().Len(types.StopOrderTriggerDirectionFallsBelow))
	assert.Equal(t, 1, pool.Priced().Len(types.StopOrderTriggerDirectionRisesAbove))

	// expire b and f, should return 3 orders
	affectedOrders := pool.RemoveExpired([]string{"b", "f"})
	assert.Len(t, affectedOrders, 3)
	assert.Equal(t, affectedOrders[0].ID, "b")
	assert.Equal(t, affectedOrders[0].Status, types.StopOrderStatusExpired)
	assert.Equal(t, affectedOrders[1].ID, "e")
	assert.Equal(t, affectedOrders[1].Status, types.StopOrderStatusExpired)
	assert.Equal(t, affectedOrders[2].ID, "f")
	assert.Equal(t, affectedOrders[2].Status, types.StopOrderStatusExpired)

	// ensure the actual trees are cleaned up
	assert.Equal(t, 1, pool.Trailing().Len(types.StopOrderTriggerDirectionFallsBelow))
	assert.Equal(t, 1, pool.Trailing().Len(types.StopOrderTriggerDirectionRisesAbove))
	assert.Equal(t, 1, pool.Priced().Len(types.StopOrderTriggerDirectionFallsBelow))
	assert.Equal(t, 0, pool.Priced().Len(types.StopOrderTriggerDirectionRisesAbove))
}

func TestCannotSubmitSameOrderTwice(t *testing.T) {
	pool := stoporders.New(logging.NewTestLogger())

	pool.PriceUpdated(num.NewUint(50))
	pool.Insert(newPricedStopOrder("a", "p1", "b", num.NewUint(40), types.StopOrderTriggerDirectionFallsBelow))
	assert.Panics(t, func() {
		pool.Insert(newPricedStopOrder("a", "p1", "b", num.NewUint(40), types.StopOrderTriggerDirectionFallsBelow))
	})
}

func TestOCOStopOrders(t *testing.T) {
	pool := stoporders.New(logging.NewTestLogger())

	pool.PriceUpdated(num.NewUint(50))

	// this is going to trigger when going up 60, and cancel a
	pool.Insert(newPricedStopOrder("a", "p1", "b", num.NewUint(40), types.StopOrderTriggerDirectionFallsBelow))
	pool.Insert(newPricedStopOrder("b", "p1", "a", num.NewUint(57), types.StopOrderTriggerDirectionRisesAbove))

	// this will be triggered when going from 60 to 57, and triggre the falls below + cancel d
	pool.Insert(newTrailingStopOrder("c", "p2", "d", num.MustDecimalFromString("0.05"), types.StopOrderTriggerDirectionFallsBelow))
	pool.Insert(newTrailingStopOrder("d", "p2", "c", num.MustDecimalFromString("0.5"), types.StopOrderTriggerDirectionRisesAbove))

	// mixing around both, will be triggered by the end
	pool.Insert(newPricedStopOrder("e", "p2", "f", num.NewUint(40), types.StopOrderTriggerDirectionFallsBelow))
	pool.Insert(newTrailingStopOrder("f", "p2", "e", num.MustDecimalFromString("0.5"), types.StopOrderTriggerDirectionRisesAbove))

	assert.Equal(t, pool.Len(), 6)

	// move the price a little, nothing should happen.
	triggeredOrders, cancelledOrders := pool.PriceUpdated(num.NewUint(55))
	assert.Len(t, triggeredOrders, 0)
	assert.Len(t, cancelledOrders, 0)

	t.Run("price move triggers priced stop order", func(t *testing.T) {
		// move the price so the priced stop order is triggered, this should return both
		triggeredOrders, cancelledOrders = pool.PriceUpdated(num.NewUint(60))
		assert.Len(t, triggeredOrders, 1)
		assert.Len(t, cancelledOrders, 1)
		assert.Equal(t, pool.Len(), 4)
		assert.Equal(t, triggeredOrders[0].Status, types.StopOrderStatusTriggered)
		assert.Equal(t, cancelledOrders[0].Status, types.StopOrderStatusStopped)
		assert.Equal(t, triggeredOrders[0].ID, "b")
		assert.Equal(t, cancelledOrders[0].ID, "a")
	})

	// try to remove it now. no errors, the party have no orders anymore
	t.Run("removing when party have submitted nothing returns no error", func(t *testing.T) {
		affectedOrders, err := pool.Cancel("p1", "a")
		assert.Len(t, affectedOrders, 0)
		assert.NoError(t, err)
	})

	t.Run("price update trigger OCO trailing order", func(t *testing.T) {
		// move the price so the trailing order get triggered
		triggeredOrders, cancelledOrders = pool.PriceUpdated(num.NewUint(57))
		assert.Len(t, triggeredOrders, 1)
		assert.Len(t, cancelledOrders, 1)
		assert.Equal(t, pool.Len(), 2)
		assert.Equal(t, triggeredOrders[0].Status, types.StopOrderStatusTriggered)
		assert.Equal(t, cancelledOrders[0].Status, types.StopOrderStatusStopped)
		assert.Equal(t, triggeredOrders[0].ID, "c")
		assert.Equal(t, cancelledOrders[0].ID, "d")
	})

	t.Run("trying to remove a triggered order returns an error", func(t *testing.T) {
		// try to remove it now. no errors, the party have no orders anymore
		affectedOrders, err := pool.Cancel("p2", "c")
		assert.Len(t, affectedOrders, 0)
		assert.EqualError(t, err, "stop order not found")
	})

	t.Run("price update trigger OCO trailing order/priced order", func(t *testing.T) {
		// move the price so the trailing order get triggered
		triggeredOrders, cancelledOrders = pool.PriceUpdated(num.NewUint(75))
		assert.Len(t, triggeredOrders, 1)
		assert.Len(t, cancelledOrders, 1)
		assert.Equal(t, pool.Len(), 0)
		assert.Equal(t, triggeredOrders[0].Status, types.StopOrderStatusTriggered)
		assert.Equal(t, cancelledOrders[0].Status, types.StopOrderStatusStopped)
		assert.Equal(t, triggeredOrders[0].ID, "f")
		assert.Equal(t, cancelledOrders[0].ID, "e")
	})
}

func newPricedStopOrder(
	id, party, ocoLinkID string,
	price *num.Uint,
	direction types.StopOrderTriggerDirection,
) *types.StopOrder {
	return &types.StopOrder{
		ID:        id,
		Party:     party,
		OCOLinkID: ocoLinkID,
		Trigger:   types.NewPriceStopOrderTrigger(direction, price),
		Expiry:    &types.StopOrderExpiry{}, // no expiry, not important here
		CreatedAt: time.Now(),
		UpdatedAt: time.Now().Add(10 * time.Second),
		Status:    types.StopOrderStatusPending,
		OrderSubmission: &types.OrderSubmission{
			MarketID:    "some",
			Type:        types.OrderTypeMarket,
			ReduceOnly:  true,
			Size:        10,
			TimeInForce: types.OrderTimeInForceIOC,
			Side:        types.SideBuy,
		},
	}
}

//nolint:unparam
func newTrailingStopOrder(
	id, party, ocoLinkID string,
	offset num.Decimal,
	direction types.StopOrderTriggerDirection,
) *types.StopOrder {
	return &types.StopOrder{
		ID:        id,
		Party:     party,
		OCOLinkID: ocoLinkID,
		Trigger:   types.NewTrailingStopOrderTrigger(direction, offset),
		Expiry:    &types.StopOrderExpiry{}, // no expiry, not important here
		CreatedAt: time.Now(),
		UpdatedAt: time.Now().Add(10 * time.Second),
		Status:    types.StopOrderStatusPending,
		OrderSubmission: &types.OrderSubmission{
			MarketID:    "some",
			Type:        types.OrderTypeMarket,
			ReduceOnly:  true,
			Size:        10,
			TimeInForce: types.OrderTimeInForceIOC,
			Side:        types.SideBuy,
		},
	}
}
