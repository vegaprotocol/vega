package subscribers

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"

	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/logging"
	types "code.vegaprotocol.io/vega/proto"
)

type priceLevel struct {
	// Price of the price level
	price uint64
	// How many orders are at this level
	totalOrders uint64
	// How much volume is at this level
	totalVolume uint64
	// What side of the book is this level
	side types.Side
}

// MarketDepth holds all the details about a single markets MarketDepth
type MarketDepth struct {
	// Which market is this for
	marketID string
	// All of the orders in the order book
	liveOrders map[string]*types.Order
	// Just the buy side of the book
	buySide []*priceLevel
	// Just the sell side of the book
	sellSide []*priceLevel
	// All price levels that have changed in the last update
	changes []*priceLevel
	// Sequence number is an increment-only value to identify a state
	// of the market depth in time. Used when trying to match updates
	// to a snapshot dump
	sequenceNumber uint64
}

// MarketDepthBuilder is a subscriber of order events
// used to build the live market depth structure
type MarketDepthBuilder struct {
	*Base
	mu sync.RWMutex
	// Map of all the markets to their market depth
	marketDepths map[string]*MarketDepth
	// Incrementing counter for subscriberID
	subscriberID uint64
	// Map of subscriberIds to their channels
	subscribers map[uint64]chan<- *types.MarketDepthUpdate
	// Logger
	log *logging.Logger
}

// NewMarketDepthBuilder constructor to create a market depth subscriber
func NewMarketDepthBuilder(ctx context.Context, log *logging.Logger, ack bool) *MarketDepthBuilder {
	mdb := MarketDepthBuilder{
		Base:         NewBase(ctx, 10, ack),
		log:          log,
		marketDepths: map[string]*MarketDepth{},
		subscribers:  map[uint64]chan<- *types.MarketDepthUpdate{},
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
				mdb.Push(e...)
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

func (md *MarketDepth) orderExists(orderID string) *types.Order {
	return md.liveOrders[orderID]
}

func (md *MarketDepth) removeOrder(order *types.Order) error {
	// Find the price level
	pl := md.getPriceLevel(order.Side, order.Price)

	if pl == nil {
		return errors.New("unknown pricelevel")
	}
	// Update the values
	pl.totalOrders--
	pl.totalVolume -= order.Remaining

	// See if we can remove this price level
	if pl.totalOrders == 0 {
		md.removePriceLevel(order)
	}

	md.changes = append(md.changes, pl)

	// Remove the orderID from the list of live orders
	delete(md.liveOrders, order.Id)
	return nil
}

func (md *MarketDepth) createNewPriceLevel(order *types.Order) *priceLevel {
	pl := &priceLevel{
		price:       order.Price,
		totalOrders: 1,
		totalVolume: order.Remaining,
		side:        order.Side,
	}

	if order.Side == types.Side_SIDE_BUY {
		index := sort.Search(len(md.buySide), func(i int) bool { return md.buySide[i].price <= order.Price })
		if index < len(md.buySide) {
			// We need to go midslice
			md.buySide = append(md.buySide, nil)
			copy(md.buySide[index+1:], md.buySide[index:])
			md.buySide[index] = pl
		} else {
			// We can tag on the end
			md.buySide = append(md.buySide, pl)
		}
	} else {
		index := sort.Search(len(md.sellSide), func(i int) bool { return md.sellSide[i].price >= order.Price })
		if index < len(md.sellSide) {
			// We need to go midslice
			md.sellSide = append(md.sellSide, nil)
			copy(md.sellSide[index+1:], md.sellSide[index:])
			md.sellSide[index] = pl
		} else {
			// We can tag on the end
			md.sellSide = append(md.sellSide, pl)
		}
	}
	return pl
}

func (md *MarketDepth) addOrder(order *types.Order) {
	// Cache the orderID
	orderCopy := *order
	md.liveOrders[order.Id] = &orderCopy

	// Update the price level
	pl := md.getPriceLevel(order.Side, order.Price)

	if pl == nil {
		pl = md.createNewPriceLevel(order)
	} else {
		pl.totalOrders++
		pl.totalVolume += order.Remaining
	}
	md.changes = append(md.changes, pl)
}

func (md *MarketDepth) updateOrder(originalOrder, newOrder *types.Order) {
	// If the price is the same, we can update the original order
	if originalOrder.Price == newOrder.Price {
		if newOrder.Remaining == 0 {
			md.removeOrder(newOrder)
		} else {
			// Update
			pl := md.getPriceLevel(originalOrder.Side, originalOrder.Price)
			pl.totalVolume += newOrder.Remaining - originalOrder.Remaining
			originalOrder.Remaining = newOrder.Remaining
			originalOrder.Size = newOrder.Size
			md.changes = append(md.changes, pl)
		}
	} else {
		md.removeOrder(originalOrder)
		if newOrder.Remaining > 0 {
			md.addOrder(newOrder)
		}
	}
}

func (md *MarketDepth) getPriceLevel(side types.Side, price uint64) *priceLevel {
	var i int
	if side == types.Side_SIDE_BUY {
		// buy side levels should be ordered in descending
		i = sort.Search(len(md.buySide), func(i int) bool { return md.buySide[i].price <= price })
		if i < len(md.buySide) && md.buySide[i].price == price {
			return md.buySide[i]
		}
	} else {
		// sell side levels should be ordered in ascending
		i = sort.Search(len(md.sellSide), func(i int) bool { return md.sellSide[i].price >= price })
		if i < len(md.sellSide) && md.sellSide[i].price == price {
			return md.sellSide[i]
		}
	}
	return nil
}

func (md *MarketDepth) removePriceLevel(order *types.Order) {
	var i int
	if order.Side == types.Side_SIDE_BUY {
		// buy side levels should be ordered in descending
		i = sort.Search(len(md.buySide), func(i int) bool { return md.buySide[i].price <= order.Price })
		if i < len(md.buySide) && md.buySide[i].price == order.Price {
			copy(md.buySide[i:], md.buySide[i+1:])
			md.buySide[len(md.buySide)-1] = nil
			md.buySide = md.buySide[:len(md.buySide)-1]
		}
	} else {
		// sell side levels should be ordered in ascending
		i = sort.Search(len(md.sellSide), func(i int) bool { return md.sellSide[i].price >= order.Price })
		// we found the level just return it.
		if i < len(md.sellSide) && md.sellSide[i].price == order.Price {
			copy(md.sellSide[i:], md.sellSide[i+1:])
			md.sellSide[len(md.sellSide)-1] = nil
			md.sellSide = md.sellSide[:len(md.sellSide)-1]
		}
	}
}

func (mdb *MarketDepthBuilder) updateMarketDepth(order *types.Order) {
	mdb.mu.Lock()
	defer mdb.mu.Unlock()

	// Non persistent and network orders do not matter
	if order.Type == types.Order_TYPE_MARKET ||
		order.TimeInForce == types.Order_TIME_IN_FORCE_FOK ||
		order.TimeInForce == types.Order_TIME_IN_FORCE_IOC {
		return
	}

	// Orders that where not valid are ignored
	if order.Status == types.Order_STATUS_UNSPECIFIED ||
		order.Status == types.Order_STATUS_REJECTED {
		return
	}

	// See if we already have a MarketDepth item for this market
	md := mdb.marketDepths[order.MarketId]
	if md == nil {
		// First time we have an update for this market
		// so we need to create a new MarketDepth
		md = &MarketDepth{marketID: order.MarketId,
			liveOrders: map[string]*types.Order{}}
		mdb.marketDepths[order.MarketId] = md
	}

	md.sequenceNumber++

	// Initialise changes slice ready for new items
	md.changes = []*priceLevel{}

	// Do we know about this order already?
	originalOrder := md.orderExists(order.Id)
	if originalOrder != nil {
		// Check to see if we are updating the order or removing it
		if order.Status == types.Order_STATUS_CANCELLED ||
			order.Status == types.Order_STATUS_EXPIRED ||
			order.Status == types.Order_STATUS_STOPPED ||
			order.Status == types.Order_STATUS_FILLED ||
			order.Status == types.Order_STATUS_PARTIALLY_FILLED ||
			order.Status == types.Order_STATUS_PARKED {
			md.removeOrder(originalOrder)
		} else {
			md.updateOrder(originalOrder, order)
		}
	} else {
		if order.Remaining > 0 && order.Status == types.Order_STATUS_ACTIVE {
			md.addOrder(order)
		}
	}

	buyPtr := []*types.PriceLevel{}
	sellPtr := []*types.PriceLevel{}

	// Send out market depth updates to any listeners
	for _, pl := range md.changes {
		if pl.side == types.Side_SIDE_BUY {
			buyPtr = append(buyPtr, &types.PriceLevel{
				Price:          pl.price,
				NumberOfOrders: pl.totalOrders,
				Volume:         pl.totalVolume,
			})
		} else {
			sellPtr = append(sellPtr, &types.PriceLevel{
				Price:          pl.price,
				NumberOfOrders: pl.totalOrders,
				Volume:         pl.totalVolume,
			})
		}
	}

	marketDepthUpdate := &types.MarketDepthUpdate{
		MarketId:       order.MarketId,
		Buy:            buyPtr,
		Sell:           sellPtr,
		SequenceNumber: md.sequenceNumber,
	}

	for _, channel := range mdb.subscribers {
		channel <- marketDepthUpdate
	}

	// Clear the list of changes
	md.changes = make([]*priceLevel, 0, len(md.changes))
}

// Returns the min of 2 uint64s
func min(x, y uint64) uint64 {
	if y < x {
		return y
	}
	return x
}

// GetMarketDepth builds up the structure to be sent out to any market depth listeners
func (mdb *MarketDepthBuilder) GetMarketDepth(ctx context.Context, market string, limit uint64) (*types.MarketDepth, error) {
	mdb.mu.RLock()
	defer mdb.mu.RUnlock()
	md, ok := mdb.marketDepths[market]
	if !ok || md == nil {
		// When a market is new with no orders there will not be any market depth/order book
		// so we do not need to try and calculate the depth cumulative volumes etc
		return &types.MarketDepth{
			MarketId: market,
			Buy:      []*types.PriceLevel{},
			Sell:     []*types.PriceLevel{},
		}, nil
	}

	buyLimit := uint64(len(md.buySide))
	sellLimit := uint64(len(md.sellSide))
	if limit > 0 {
		buyLimit = min(buyLimit, limit)
		sellLimit = min(sellLimit, limit)
	}

	buyPtr := make([]*types.PriceLevel, buyLimit)
	sellPtr := make([]*types.PriceLevel, sellLimit)

	// Copy the data across
	for index, pl := range md.buySide[:buyLimit] {
		buyPtr[index] = &types.PriceLevel{Volume: pl.totalVolume,
			NumberOfOrders: pl.totalOrders,
			Price:          pl.price}
	}

	for index, pl := range md.sellSide[:sellLimit] {
		sellPtr[index] = &types.PriceLevel{Volume: pl.totalVolume,
			NumberOfOrders: pl.totalOrders,
			Price:          pl.price}
	}

	return &types.MarketDepth{
		MarketId:       market,
		Buy:            buyPtr,
		Sell:           sellPtr,
		SequenceNumber: md.sequenceNumber,
	}, nil
}

/*****************************************************************************/
/*                 FUNCTIONS TO HELP WITH UNIT TESTING                       */
/*****************************************************************************/

// GetOrderCount returns the number of live orders for the given market
func (mdb *MarketDepthBuilder) GetOrderCount(market string) int64 {
	var liveOrders int64
	var bookOrders uint64
	md := mdb.marketDepths[market]
	if md != nil {
		liveOrders = int64(len(md.liveOrders))

		for _, pl := range md.buySide {
			bookOrders += pl.totalOrders
		}

		for _, pl := range md.sellSide {
			bookOrders += pl.totalOrders
		}

		if liveOrders != int64(bookOrders) {
			return -1
		}
		return liveOrders
	}
	return 0
}

// GetVolumeAtPrice returns the order volume at the given price level
func (mdb *MarketDepthBuilder) GetVolumeAtPrice(market string, side types.Side, price uint64) uint64 {
	md := mdb.marketDepths[market]
	if md != nil {
		pl := md.getPriceLevel(side, price)
		if pl == nil {
			return 0
		}
		return pl.totalVolume
	}
	return 0
}

// GetTotalVolume returns the total volume in the order book
func (mdb *MarketDepthBuilder) GetTotalVolume(market string) int64 {
	var volume int64
	md := mdb.marketDepths[market]
	if md != nil {
		for _, pl := range md.buySide {
			volume += int64(pl.totalVolume)
		}

		for _, pl := range md.sellSide {
			volume += int64(pl.totalVolume)
		}
		return volume
	}
	return 0
}

// GetOrderCountAtPrice returns the number of orders at the given price level
func (mdb *MarketDepthBuilder) GetOrderCountAtPrice(market string, side types.Side, price uint64) uint64 {
	md := mdb.marketDepths[market]
	if md != nil {
		pl := md.getPriceLevel(side, price)
		if pl == nil {
			return 0
		}
		return pl.totalOrders
	}
	return 0
}

// GetPriceLevels returns the number of non empty price levels
func (mdb *MarketDepthBuilder) GetPriceLevels(market string) int {
	return mdb.GetBuyPriceLevels(market) + mdb.GetSellPriceLevels(market)
}

// GetBestBidPrice returns the highest bid price in the book
func (mdb *MarketDepthBuilder) GetBestBidPrice(market string) uint64 {
	md := mdb.marketDepths[market]
	if md != nil {
		if len(md.buySide) > 0 {
			return md.buySide[0].price
		}
	}
	return 0
}

// GetBestAskPrice returns the highest bid price in the book
func (mdb *MarketDepthBuilder) GetBestAskPrice(market string) uint64 {
	md := mdb.marketDepths[market]
	if md != nil {
		if len(md.sellSide) > 0 {
			return md.sellSide[0].price
		}
	}
	return 0
}

// GetBuyPriceLevels returns the number of non empty buy price levels
func (mdb *MarketDepthBuilder) GetBuyPriceLevels(market string) int {
	md := mdb.marketDepths[market]
	if md != nil {
		return len(md.buySide)
	}
	return 0
}

// GetSellPriceLevels returns the number of non empty sell price levels
func (mdb *MarketDepthBuilder) GetSellPriceLevels(market string) int {
	md := mdb.marketDepths[market]
	if md != nil {
		return len(md.sellSide)
	}
	return 0
}

// Subscribe allows a client to register for updates of the market depth book
func (mdb *MarketDepthBuilder) Subscribe(updates chan<- *types.MarketDepthUpdate) uint64 {
	mdb.mu.Lock()
	defer mdb.mu.Unlock()

	mdb.subscriberID++
	mdb.subscribers[mdb.subscriberID] = updates

	return mdb.subscriberID
}

// Unsubscribe allows the client to unregister interest in market depth updates
func (mdb *MarketDepthBuilder) Unsubscribe(id uint64) error {
	mdb.mu.Lock()
	defer mdb.mu.Unlock()

	if len(mdb.subscribers) == 0 {
		return nil
	}

	if _, exists := mdb.subscribers[id]; exists {
		delete(mdb.subscribers, id)
		return nil
	}

	return fmt.Errorf("subscriber to market depth updates does not exist with id: %d", id)
}
