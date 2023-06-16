package stoporders

import (
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
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
