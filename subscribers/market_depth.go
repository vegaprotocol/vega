package subscribers

import (
	"context"
	"fmt"
	"sync"

	"code.vegaprotocol.io/vega/events"
	types "code.vegaprotocol.io/vega/proto"
)

type priceLevel struct {
	price       int64
	totalOrders int64
	totalVolume int64
}

// MarketDepthBuilder is a subscriber of order events
// used to build the live market depth structure
type MarketDepthBuilder struct {
	*Base
	mu         sync.Mutex
	buf        []types.Order
	liveOrders map[string]*types.Order
	buySide    []*priceLevel
	sellSide   []*priceLevel
}

// NewMarketDepthBuilder constructor to create a market depth subscriber
func NewMarketDepthBuilder(ctx context.Context, ack bool) *MarketDepthBuilder {
	mdb := MarketDepthBuilder{
		Base: NewBase(ctx, 10, ack),
		buf:  []types.Order{},
	}
	if mdb.isRunning() {
		go mdb.loop(mdb.ctx)
	}
	return &mdb
}

func (mdb *MarketDepthBuilder) loop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			mdb.Halt()
			return
		case e := <-mdb.ch:
			if mdb.isRunning() {
				mdb.Push(e)
			}
		}
	}
}

// Push takes order messages and applied them to the makret depth structure
func (mdb *MarketDepthBuilder) Push(evts ...events.Event) {
	for _, e := range evts {
		switch te := e.(type) {
		case OE:
			mdb.updateMarketDepth(te.Order())
		}
	}
}

// Types returns all the message types this subscriber wants to receive
func (mdb *MarketDepthBuilder) Types() []events.Type {
	return []events.Type{
		events.OrderEvent,
	}
}

func (mdb *MarketDepthBuilder) orderExists(orderID string) *types.Order {
	return mdb.liveOrders[orderID]
}

func (mdb *MarketDepthBuilder) removeOrder(order *types.Order) {
	// Find the price level

	// Update the values

	// Remove the orderID from the list of live orders
}

func (mdb *MarketDepthBuilder) addOrder(order *types.Order) {

}

func (mdb *MarketDepthBuilder) updateOrder(originalOrder, newOrder *types.Order) {

}

func (mdb *MarketDepthBuilder) updateMarketDepth(order *types.Order) {
	fmt.Println("MDB Order:", order)

	// Do we know about this order already?
	originalOrder := mdb.orderExists(order.Id)
	if originalOrder != nil {
		// Remove the original order values

		// Insert the new order values
	} else {
		// We have a new order, add it to the structure
	}
}

/*****************************************************************************/
/*                 FUNCTIONS TO HELP WITH UNIT TESTING                       */
/*****************************************************************************/

func (mdb *MarketDepthBuilder) GetOrderCount() int {
	return len(mdb.liveOrders)
}

func (mdb *MarketDepthBuilder) GetVolumeAtPrice(price uint64) int {
	return len(mdb.liveOrders)
}

func (mdb *MarketDepthBuilder) GetOrderCountAtPrice(price uint64) int {
	return len(mdb.liveOrders)
}

func (mdb *MarketDepthBuilder) GetPriceLevels() int {
	return mdb.GetBuyPriceLevels() + mdb.GetSellPriceLevels()
}

func (mdb *MarketDepthBuilder) GetBuyPriceLevels() int {
	return len(mdb.buySide)
}

func (mdb *MarketDepthBuilder) GetSellPriceLevels() int {
	return len(mdb.sellSide)
}
