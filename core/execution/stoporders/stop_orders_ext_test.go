package stoporders

import (
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	"golang.org/x/exp/slices"
)

func (t *TrailingStopOrders) Len(direction types.StopOrderTriggerDirection) int {
	switch direction {
	case types.StopOrderTriggerDirectionFallsBelow:
		return t.fallsBelow.Len()
	case types.StopOrderTriggerDirectionRisesAbove:
		return t.risesAbove.Len()
	default:
		panic("nope")
	}
}

func (p *PricedStopOrders) Len(direction types.StopOrderTriggerDirection) int {
	switch direction {
	case types.StopOrderTriggerDirectionFallsBelow:
		return p.fallsBelow.Len()
	case types.StopOrderTriggerDirectionRisesAbove:
		return p.risesAbove.Len()
	default:
		panic("nope")
	}
}

func (p *PricedStopOrders) Exists(id string) (atPrice *num.Uint, exists bool) {
	findFn := func(item *ordersAtPrice) bool {
		for _, v := range item.orders {
			if v == id {
				atPrice = item.price.Clone()
				exists = true
				return false
			}
		}

		return true
	}

	p.fallsBelow.Ascend(findFn)

	if !exists {
		p.risesAbove.Ascend(findFn)
	}

	return
}

func (p *TrailingStopOrders) Exists(id string) (atPrice *num.Uint, offset num.Decimal, exists bool) {
	findFnOrder := func(item *ordersAtOffset) bool {
		for _, v := range item.orders {
			if v == id {
				exists = true
				offset = item.offset
				return false
			}
		}

		return true
	}

	findFn := func(item *offsetsAtPrice) bool {
		item.offsets.Ascend(findFnOrder)
		if exists {
			atPrice = item.price.Clone()
			return false
		}

		return true
	}

	p.fallsBelow.Ascend(findFn)

	if !exists {
		p.risesAbove.Ascend(findFn)
	}

	return
}

func (p *Pool) Len() int {
	return len(p.orderToParty)
}

func (p *Pool) Trailing() *TrailingStopOrders {
	return p.trailing
}

func (p *Pool) Priced() *PricedStopOrders {
	return p.priced
}

func (p *PricedStopOrders) Equal(p2 *PricedStopOrders) bool {
	fallsbelowOk, risesAboveOk := true, true

	p.fallsBelow.Ascend(func(item *ordersAtPrice) bool {
		item2, ok := p2.fallsBelow.Get(item)
		if !ok {
			fallsbelowOk = false
			return fallsbelowOk
		}

		slices.Equal(item.orders, item2.orders)

		return fallsbelowOk
	})

	p.risesAbove.Ascend(func(item *ordersAtPrice) bool {
		item2, ok := p2.risesAbove.Get(item)
		if !ok {
			risesAboveOk = false
			return risesAboveOk
		}
		slices.Equal(item.orders, item2.orders)

		return risesAboveOk
	})

	return fallsbelowOk && risesAboveOk
}

func (p *TrailingStopOrders) Equal(p2 *TrailingStopOrders) bool {
	fallsbelowOk, risesAboveOk := true, true

	p.fallsBelow.Ascend(func(item *offsetsAtPrice) bool {
		item2, ok := p2.fallsBelow.Get(item)
		if !ok {
			fallsbelowOk = false
			return fallsbelowOk
		}

		item.offsets.Ascend(func(itemInner *ordersAtOffset) bool {
			itemInner2, ok := item2.offsets.Get(itemInner)
			if !ok {
				fallsbelowOk = false
				return fallsbelowOk
			}
			slices.Equal(itemInner.orders, itemInner2.orders)

			return fallsbelowOk
		})

		return fallsbelowOk
	})

	p.risesAbove.Ascend(func(item *offsetsAtPrice) bool {
		item2, ok := p2.risesAbove.Get(item)
		if !ok {
			fallsbelowOk = false
			return fallsbelowOk
		}

		item.offsets.Ascend(func(itemInner *ordersAtOffset) bool {
			itemInner2, ok := item2.offsets.Get(itemInner)
			if !ok {
				risesAboveOk = false
				return risesAboveOk
			}
			slices.Equal(itemInner.orders, itemInner2.orders)

			return risesAboveOk
		})

		return risesAboveOk
	})

	return fallsbelowOk && risesAboveOk
}

func (p *Pool) Equal(p2 *Pool) bool {
	if !p.trailing.lastSeenPrice.EQ(p2.trailing.lastSeenPrice) {
		return false
	}

	for k, v := range p.orderToParty {
		if v2, ok := p2.orderToParty[k]; !ok {
			return false
		} else if v2 != v {
			return false
		}
	}

	for k, v2 := range p2.orderToParty {
		if v, ok := p.orderToParty[k]; !ok {
			return false
		} else if v2 != v {
			return false
		}
	}

	for partyId, orders := range p.orders {
		other, ok := p2.orders[partyId]
		if !ok {
			return false
		}

		for orderId, order := range orders {
			otherOrder, ok := other[orderId]
			if !ok {
				return false
			}

			if otherOrder.ID != order.ID {
				return false
			}
		}
	}

	for partyId, orders := range p2.orders {
		other, ok := p.orders[partyId]
		if !ok {
			return false
		}

		for orderId, order := range orders {
			otherOrder, ok := other[orderId]
			if !ok {
				return false
			}

			if otherOrder.ID != order.ID {
				return false
			}
		}
	}

	return p.priced.Equal(p2.priced) && p2.priced.Equal(p.priced) && p.trailing.Equal(p2.trailing) && p2.trailing.Equal(p.trailing)
}
