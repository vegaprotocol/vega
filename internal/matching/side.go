package matching

import (
	"fmt"

	"code.vegaprotocol.io/vega/internal/logging"
	types "code.vegaprotocol.io/vega/proto"

	"github.com/pkg/errors"
)

var (
	ErrPriceNotFound = errors.New("price-volume pair not found")
)

type OrderBookSide struct {
	log *logging.Logger
	// Config
	levels      []*PriceLevel
	volumePrice map[uint64]uint64
	list        *priceVolList
	proRataMode bool
}

type priceVolList struct {
	prev, next *priceVolList
	idx        map[uint64]*priceVolList // index in head node to quickly find random entries in the list
	key        uint64
	val        uint64
}

func (o *OrderBookSide) addPriceVol(vol, price uint64) *priceVolList {
	var ls *priceVolList
	if o.list == nil {
		o.list = &priceVolList{
			key: price,
			val: vol,
			idx: map[uint64]*priceVolList{},
		}
		o.list.idx[price] = o.list
		return o.list
	}
	// there's already an entry for this price, let's increment the value accordingly
	if node, ok := o.list.idx[price]; ok {
		node.val += vol
		return node
	}
	// new entry needed, create the node, set the idx accordingly, and append to the list
	ls = &priceVolList{
		key: price,
		val: vol,
	}
	o.list.idx[price] = ls
	// ensure idx is set
	ls.idx = o.list.idx
	o.list.append(ls)
	return ls
}

func (o *OrderBookSide) rmPriceVol(vol, price uint64) error {
	node, ok := o.list.idx[price]
	if !ok {
		// this should produce an error!
		return ErrPriceNotFound
	}
	// remove this volume from the price-bracket
	node.val -= vol
	// we don't have anything left at this price point
	if node.val == 0 {
		delete(o.list.idx, price)
		// unlink this node, previous node points to this node's next one, and vice-versa
		if node.prev != nil {
			node.prev.next = node.next
		}
		if node.next != nil {
			node.next.prev = node.prev
		}
	}
	return nil
}

func (l *priceVolList) append(node *priceVolList) {
	var current *priceVolList
	for current = l; current.next != nil; current = current.next {
		if current.key > node.key {
			// insert new node:
			// the previous node is taken from current.prev, the next node is the "current" node
			// meanwhile, the current.prev.next, and current.prev both reference the new node
			node.prev, node.next = current.prev, current
			if current.prev != nil {
				current.prev.next = node
			}
			current.prev = node
			return
		}
	}
	// current node still is lower down, we have to append
	node.prev, current.next = current, node
}

func (s *OrderBookSide) addOrder(o *types.Order, side types.Side) {
	// update the price-volume map
	_ = s.addPriceVol(o.Size, o.Price)
	s.getPriceLevel(o.Price, side).addOrder(o)
}

func (s *OrderBookSide) amendOrder(orderAmended *types.Order) error {
	priceLevelIndex := -1
	orderIndex := -1
	var oldOrder *types.Order

	for idx, priceLevel := range s.levels {
		if priceLevel.price == orderAmended.Price {
			priceLevelIndex = idx
			for j, order := range priceLevel.orders {
				if order.Id == orderAmended.Id {
					orderIndex = j
					oldOrder = order
					break
				}
			}
			break
		}
	}

	if priceLevelIndex == -1 || orderIndex == -1 {
		return types.ErrOrderNotFound
	}

	if oldOrder.PartyID != orderAmended.PartyID {
		return types.ErrOrderAmendFailure
	}

	if oldOrder.Size < orderAmended.Size {
		return types.ErrOrderAmendFailure
	}

	if oldOrder.Reference != orderAmended.Reference {
		return types.ErrOrderAmendFailure
	}
	// remove the old, add the new - it might be more efficient to check for price changes, and if the price remains the same, have an update func (instead of rm + add)
	if err := s.rmPriceVol(oldOrder.Size, oldOrder.Price); err != nil {
		return err
	}
	_ = s.addPriceVol(orderAmended.Size, orderAmended.Price)

	s.levels[priceLevelIndex].orders[orderIndex] = orderAmended
	return nil
}

func (s *OrderBookSide) RemoveOrder(o *types.Order) error {
	//todo: use binary search of expiring price levels (https://gitlab.com/vega-protocol/trading-core/issues/132)
	toDelete := -1
	toRemove := -1
	for idx, priceLevel := range s.levels {
		if priceLevel.price == o.Price {
			for j, order := range priceLevel.orders {
				if order.Id == o.Id {
					toRemove = j
					break
				}
			}
			if toRemove != -1 {
				priceLevel.removeOrder(toRemove)
			}
			if len(priceLevel.orders) == 0 {
				toDelete = idx
			}
			break
		}
	}
	if toDelete != -1 {
		copy(s.levels[toDelete:], s.levels[toDelete+1:])
		s.levels = s.levels[:len(s.levels)-1]

	}
	if toRemove == -1 {
		return types.ErrOrderNotFound
	}
	return s.rmPriceVol(o.Size, o.Price)
}

func (s *OrderBookSide) getPriceLevel(price uint64, side types.Side) *PriceLevel {
	//todo: use binary search of price levels (gitlab.com/vega-protocol/trading-core/issues/90)
	at := -1
	if side == types.Side_Buy {
		// buy side levels should be ordered in descending
		for i, level := range s.levels {
			if level.price > price {
				continue
			}
			if level.price == price {
				return level
			}
			at = i
			break
		}
	} else {
		// sell side levels should be ordered in ascending
		for i, level := range s.levels {
			if level.price < price {
				continue
			}
			if level.price == price {
				return level
			}
			at = i
			break
		}
	}
	level := NewPriceLevel(price, s.proRataMode)
	if at == -1 {
		s.levels = append(s.levels, level)
		return level
	}
	s.levels = append(s.levels[:at], append([]*PriceLevel{level}, s.levels[at:]...)...)
	return level
}

func (s *OrderBookSide) uncross(agg *types.Order) ([]*types.Trade, []*types.Order, uint64) {

	var (
		trades            []*types.Trade
		impactedOrders    []*types.Order
		lastTradedPrice   uint64
		totalVolumeToFill uint64
	)

	if agg.Type == types.Order_FOK {

		if agg.Side == types.Side_Sell {
			for _, level := range s.levels {
				if level.price >= agg.Price {
					totalVolumeToFill += level.volume
				}
			}
		}

		if agg.Side == types.Side_Buy {
			for _, level := range s.levels {
				if level.price <= agg.Price {
					totalVolumeToFill += level.volume
				}
			}
		}

		s.log.Debug(fmt.Sprintf("totalVolumeToFill %d until price %d, remaining %d\n", totalVolumeToFill, agg.Price, agg.Remaining))

		if totalVolumeToFill <= agg.Remaining {
			return trades, impactedOrders, 0
		}
	}

	if agg.Side == types.Side_Sell {
		for _, level := range s.levels {
			// buy side levels are ordered descending
			if level.price >= agg.Price {
				filled, nTrades, nImpact := level.uncross(agg)
				trades = append(trades, nTrades...)
				impactedOrders = append(impactedOrders, nImpact...)
				if filled {
					break
				}
			} else {
				break
			}
		}
	}

	if agg.Side == types.Side_Buy {
		for _, level := range s.levels {
			// sell side levels are ordered ascending
			if level.price <= agg.Price {
				filled, nTrades, nImpact := level.uncross(agg)
				trades = append(trades, nTrades...)
				impactedOrders = append(impactedOrders, nImpact...)
				if filled {
					break
				}
			} else {
				break
			}
		}
	}

	if len(trades) > 0 {
		lastTradedPrice = trades[len(trades)-1].Price
	}
	return trades, impactedOrders, lastTradedPrice
}

func (s *OrderBookSide) getLevels() []*PriceLevel {
	return s.levels
}
