package subscribers

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"

	"code.vegaprotocol.io/vega/events"
	types "code.vegaprotocol.io/vega/proto"
)

type priceLevel struct {
	price       uint64
	totalOrders uint64
	totalVolume uint64
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
		Base:       NewBase(ctx, 10, ack),
		buf:        []types.Order{},
		liveOrders: map[string]*types.Order{},
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

func (mdb *MarketDepthBuilder) removeOrder(order *types.Order) error {
	// Find the price level
	pl := mdb.getPriceLevel(order.Side, order.Price)

	if pl == nil {
		fmt.Println("Unable to find price level for order:", order)
		return errors.New("Unknown pricelevel")
	}
	// Update the values
	pl.totalOrders--
	pl.totalVolume -= order.Remaining

	// See if we can remove this price level
	if pl.totalOrders == 0 {
		mdb.removePriceLevel(order)
	}

	// Remove the orderID from the list of live orders
	delete(mdb.liveOrders, order.Id)
	return nil
}

func (mdb *MarketDepthBuilder) createNewPriceLevel(order *types.Order) *priceLevel {
	pl := &priceLevel{
		price:       order.Price,
		totalOrders: 1,
		totalVolume: order.Remaining,
	}

	if order.Side == types.Side_SIDE_BUY {
		index := sort.Search(len(mdb.buySide), func(i int) bool { return mdb.buySide[i].price <= order.Price })
		if index < len(mdb.buySide) {
			// We need to go midslice
			mdb.buySide = append(mdb.buySide, nil)
			copy(mdb.buySide[index+1:], mdb.buySide[index:])
			mdb.buySide[index] = pl
		} else {
			// We can tag on the end
			mdb.buySide = append(mdb.buySide, pl)
		}
	} else {
		index := sort.Search(len(mdb.sellSide), func(i int) bool { return mdb.sellSide[i].price >= order.Price })
		if index < len(mdb.sellSide) {
			// We need to go midslice
			mdb.sellSide = append(mdb.sellSide, nil)
			copy(mdb.sellSide[index+1:], mdb.sellSide[index:])
			mdb.sellSide[index] = pl
		} else {
			// We can tag on the end
			mdb.sellSide = append(mdb.sellSide, pl)
		}
	}
	return pl
}

func (mdb *MarketDepthBuilder) addOrder(order *types.Order) {
	// Cache the orderID
	mdb.liveOrders[order.Id] = order

	// Update the price level
	pl := mdb.getPriceLevel(order.Side, order.Price)

	if pl == nil {
		pl = mdb.createNewPriceLevel(order)
	} else {
		pl.totalOrders++
		pl.totalVolume += order.Remaining
	}
}

func (mdb *MarketDepthBuilder) updateOrder(originalOrder, newOrder *types.Order) {
	// If the price is the same, we can update the original order
	if originalOrder.Price == newOrder.Price {
		// Update
		pl := mdb.getPriceLevel(originalOrder.Side, originalOrder.Price)
		pl.totalVolume += (newOrder.Remaining - originalOrder.Remaining)

		if newOrder.Remaining == 0 {
			mdb.removeOrder(newOrder)
			pl.totalOrders -= 1
		}

		if pl.totalOrders == 0 {
			mdb.removePriceLevel(newOrder)
		}
	} else {
		mdb.removeOrder(originalOrder)
		mdb.addOrder(newOrder)
	}
}

func (mdb *MarketDepthBuilder) getPriceLevel(side types.Side, price uint64) *priceLevel {
	var i int
	if side == types.Side_SIDE_BUY {
		// buy side levels should be ordered in descending
		i = sort.Search(len(mdb.buySide), func(i int) bool { return mdb.buySide[i].price <= price })
		if i < len(mdb.buySide) && mdb.buySide[i].price == price {
			return mdb.buySide[i]
		}
	} else {
		// sell side levels should be ordered in ascending
		i = sort.Search(len(mdb.sellSide), func(i int) bool { return mdb.sellSide[i].price >= price })
		if i < len(mdb.sellSide) && mdb.sellSide[i].price == price {
			return mdb.sellSide[i]
		}
	}
	return nil
}

func (mdb *MarketDepthBuilder) removePriceLevel(order *types.Order) {
	var i int
	if order.Side == types.Side_SIDE_BUY {
		// buy side levels should be ordered in descending
		i = sort.Search(len(mdb.buySide), func(i int) bool { return mdb.buySide[i].price == order.Price })
		if i < len(mdb.buySide) && mdb.buySide[i].price == order.Price {
			copy(mdb.buySide[i:], mdb.buySide[i+1:])
			mdb.buySide[len(mdb.buySide)-1] = nil
			mdb.buySide = mdb.buySide[:len(mdb.buySide)-1]
		}
	} else {
		// sell side levels should be ordered in ascending
		i = sort.Search(len(mdb.sellSide), func(i int) bool { return mdb.sellSide[i].price == order.Price })
		// we found the level just return it.
		if i < len(mdb.sellSide) && mdb.sellSide[i].price == order.Price {
			copy(mdb.sellSide[i:], mdb.sellSide[i+1:])
			mdb.sellSide[len(mdb.sellSide)-1] = nil
			mdb.sellSide = mdb.sellSide[:len(mdb.sellSide)-1]
		}
	}
}

func (mdb *MarketDepthBuilder) updateMarketDepth(order *types.Order) {
	// Non persistent and network orders do not matter
	if order.Type == types.Order_TYPE_MARKET ||
		order.TimeInForce == types.Order_TIF_FOK ||
		order.TimeInForce == types.Order_TIF_IOC {
		return
	}

	// Orders that where not valid are ignored
	if order.Status == types.Order_STATUS_INVALID ||
		order.Status == types.Order_STATUS_REJECTED {
		return
	}

	// Do we know about this order already?
	originalOrder := mdb.orderExists(order.Id)
	if originalOrder != nil {
		// Check to see if we are updating the order of removing it
		if order.Status == types.Order_STATUS_CANCELLED ||
			order.Status == types.Order_STATUS_EXPIRED ||
			order.Status == types.Order_STATUS_STOPPED {
			mdb.removeOrder(order)
		} else {
			mdb.updateOrder(originalOrder, order)
		}
	} else {
		if order.Remaining > 0 {
			mdb.addOrder(order)
		}
	}
}

/*****************************************************************************/
/*                 FUNCTIONS TO HELP WITH UNIT TESTING                       */
/*****************************************************************************/

func (mdb *MarketDepthBuilder) GetOrderCount() int {
	return len(mdb.liveOrders)
}

func (mdb *MarketDepthBuilder) GetVolumeAtPrice(side types.Side, price uint64) uint64 {
	pl := mdb.getPriceLevel(side, price)
	if pl == nil {
		return 0
	}
	return pl.totalVolume
}

func (mdb *MarketDepthBuilder) GetOrderCountAtPrice(side types.Side, price uint64) uint64 {
	pl := mdb.getPriceLevel(side, price)
	if pl == nil {
		return 0
	}
	return pl.totalOrders
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
