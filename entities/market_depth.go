package entities

import (
	"errors"
	"sort"

	"code.vegaprotocol.io/protos/vega"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"
)

type PriceLevel struct {
	// Price of the price level
	Price *num.Uint
	// How many orders are at this level
	TotalOrders uint64
	// How much volume is at this level
	TotalVolume uint64
	// What side of the book is this level
	Side types.Side
}

type MarketDepth struct {
	// Which market is this for
	MarketID string
	// All of the orders in the order book
	LiveOrders map[string]*types.Order
	// Just the buy side of the book
	BuySide []*PriceLevel
	// Just the sell side of the book
	SellSide []*PriceLevel
	// All price levels that have changed in the last update
	Changes []*PriceLevel
	// Sequence number is an increment-only value to identify a state
	// of the market depth in time. Used when trying to match updates
	// to a snapshot dump
	SequenceNumber uint64
	// PreviousSequenceNumber is the sequence number of the last published update. 'Changes' include
	// updates from all events with a sequence number > PreviousSequenceNumber and <= SequenceNumber
	PreviousSequenceNumber uint64
}

func (md *MarketDepth) ToProto(limit uint64) *vega.MarketDepth {
	buyLimit := uint64(len(md.BuySide))
	sellLimit := uint64(len(md.SellSide))
	if limit > 0 {
		buyLimit = min(buyLimit, limit)
		sellLimit = min(sellLimit, limit)
	}

	buyPtr := make([]*types.PriceLevel, buyLimit)
	sellPtr := make([]*types.PriceLevel, sellLimit)

	// Copy the data across
	for index, pl := range md.BuySide[:buyLimit] {
		buyPtr[index] = &types.PriceLevel{
			Volume:         pl.TotalVolume,
			NumberOfOrders: pl.TotalOrders,
			Price:          pl.Price.Clone(),
		}
	}

	for index, pl := range md.SellSide[:sellLimit] {
		sellPtr[index] = &types.PriceLevel{
			Volume:         pl.TotalVolume,
			NumberOfOrders: pl.TotalOrders,
			Price:          pl.Price.Clone(),
		}
	}

	return &types.MarketDepth{
		MarketId:       md.MarketID,
		Buy:            types.PriceLevels(buyPtr).IntoProto(),
		Sell:           types.PriceLevels(sellPtr).IntoProto(),
		SequenceNumber: md.SequenceNumber,
	}
}
func (md *MarketDepth) AddOrderUpdate(order *types.Order) {
	// Do we know about this order already?
	originalOrder := md.orderExists(order.ID)
	if originalOrder != nil {
		// Check to see if we are updating the order or removing it
		if order.Status == types.OrderStatusCancelled ||
			order.Status == types.OrderStatusExpired ||
			order.Status == types.OrderStatusStopped ||
			order.Status == types.OrderStatusFilled ||
			order.Status == types.OrderStatusPartiallyFilled ||
			order.Status == types.OrderStatusRejected ||
			order.Status == types.OrderStatusParked {
			md.removeOrder(originalOrder)
		} else {
			md.updateOrder(originalOrder, order)
		}
	} else {
		if order.Remaining > 0 && order.Status == types.OrderStatusActive {
			md.addOrder(order)
		}
	}
}

func (md *MarketDepth) orderExists(orderID string) *types.Order {
	return md.LiveOrders[orderID]
}

func (md *MarketDepth) addOrder(order *types.Order) {
	// Cache the orderID
	orderCopy := order.Clone()
	md.LiveOrders[order.ID] = orderCopy

	// Update the price level
	pl := md.GetPriceLevel(order.Side, order.Price)

	if pl == nil {
		pl = md.createNewPriceLevel(order)
	} else {
		pl.TotalOrders++
		pl.TotalVolume += order.Remaining
	}
	md.Changes = append(md.Changes, pl)
}

func (md *MarketDepth) removeOrder(order *types.Order) error {
	// Find the price level
	pl := md.GetPriceLevel(order.Side, order.Price)

	if pl == nil {
		return errors.New("unknown pricelevel")
	}
	// Update the values
	pl.TotalOrders--
	pl.TotalVolume -= order.Remaining

	// See if we can remove this price level
	if pl.TotalOrders == 0 {
		md.removePriceLevel(order)
	}

	md.Changes = append(md.Changes, pl)

	// Remove the orderID from the list of live orders
	delete(md.LiveOrders, order.ID)
	return nil
}

func (md *MarketDepth) updateOrder(originalOrder, newOrder *types.Order) {
	// If the price is the same, we can update the original order
	if originalOrder.Price.EQ(newOrder.Price) {
		if newOrder.Remaining == 0 {
			md.removeOrder(newOrder)
		} else {
			// Update
			pl := md.GetPriceLevel(originalOrder.Side, originalOrder.Price)
			pl.TotalVolume += newOrder.Remaining - originalOrder.Remaining
			originalOrder.Remaining = newOrder.Remaining
			originalOrder.Size = newOrder.Size
			md.Changes = append(md.Changes, pl)
		}
	} else {
		md.removeOrder(originalOrder)
		if newOrder.Remaining > 0 {
			md.addOrder(newOrder)
		}
	}
}

func (md *MarketDepth) createNewPriceLevel(order *types.Order) *PriceLevel {
	pl := &PriceLevel{
		Price:       order.Price.Clone(),
		TotalOrders: 1,
		TotalVolume: order.Remaining,
		Side:        order.Side,
	}

	if order.Side == types.SideBuy {
		index := sort.Search(len(md.BuySide), func(i int) bool { return md.BuySide[i].Price.LTE(order.Price) })
		if index < len(md.BuySide) {
			// We need to go midslice
			md.BuySide = append(md.BuySide, nil)
			copy(md.BuySide[index+1:], md.BuySide[index:])
			md.BuySide[index] = pl
		} else {
			// We can tag on the end
			md.BuySide = append(md.BuySide, pl)
		}
	} else {
		index := sort.Search(len(md.SellSide), func(i int) bool { return md.SellSide[i].Price.GTE(order.Price) })
		if index < len(md.SellSide) {
			// We need to go midslice
			md.SellSide = append(md.SellSide, nil)
			copy(md.SellSide[index+1:], md.SellSide[index:])
			md.SellSide[index] = pl
		} else {
			// We can tag on the end
			md.SellSide = append(md.SellSide, pl)
		}
	}
	return pl
}

func (md *MarketDepth) GetPriceLevel(side types.Side, price *num.Uint) *PriceLevel {
	var i int
	if side == types.SideBuy {
		// buy side levels should be ordered in descending
		i = sort.Search(len(md.BuySide), func(i int) bool { return md.BuySide[i].Price.LTE(price) })
		if i < len(md.BuySide) && md.BuySide[i].Price.EQ(price) {
			return md.BuySide[i]
		}
	} else {
		// sell side levels should be ordered in ascending
		i = sort.Search(len(md.SellSide), func(i int) bool { return md.SellSide[i].Price.GTE(price) })
		if i < len(md.SellSide) && md.SellSide[i].Price.EQ(price) {
			return md.SellSide[i]
		}
	}
	return nil
}

func (md *MarketDepth) removePriceLevel(order *types.Order) {
	var i int
	if order.Side == types.SideBuy {
		// buy side levels should be ordered in descending
		i = sort.Search(len(md.BuySide), func(i int) bool { return md.BuySide[i].Price.LTE(order.Price) })
		if i < len(md.BuySide) && md.BuySide[i].Price.EQ(order.Price) {
			copy(md.BuySide[i:], md.BuySide[i+1:])
			md.BuySide[len(md.BuySide)-1] = nil
			md.BuySide = md.BuySide[:len(md.BuySide)-1]
		}
	} else {
		// sell side levels should be ordered in ascending
		i = sort.Search(len(md.SellSide), func(i int) bool { return md.SellSide[i].Price.GTE(order.Price) })
		// we found the level just return it.
		if i < len(md.SellSide) && md.SellSide[i].Price.EQ(order.Price) {
			copy(md.SellSide[i:], md.SellSide[i+1:])
			md.SellSide[len(md.SellSide)-1] = nil
			md.SellSide = md.SellSide[:len(md.SellSide)-1]
		}
	}
}

// Returns the min of 2 uint64s
func min(x, y uint64) uint64 {
	if y < x {
		return y
	}
	return x
}
