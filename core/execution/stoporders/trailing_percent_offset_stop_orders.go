package stoporders

import (
	"errors"
	"fmt"

	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	"github.com/google/btree"
)

var (
	ErrNoPriceToOffset = errors.New("no price to offset")
)

type ordersAtOffset struct {
	offset num.Decimal
	orders []string
}

func (o *ordersAtOffset) String() string {
	return fmt.Sprintf("%s:%v", o.offset.String(), o.orders)
}

type orderAtOffsetStat struct {
	offset    num.Decimal
	direction types.StopOrderTriggerDirection
}

func lessFuncOrdersAtOffset(a, b *ordersAtOffset) bool {
	return a.offset.LessThan(b.offset)
}

type offsetsAtPrice struct {
	price   *num.Uint
	offsets *btree.BTreeG[*ordersAtOffset]
}

func (o *offsetsAtPrice) String() string {
	return fmt.Sprintf("%s:%s", o.price.String(), dumpTree(o.offsets))
}

func lessFuncOffsetsAtPrice(a, b *offsetsAtPrice) bool {
	return a.price.LT(b.price)
}

type TrailingStopOrders struct {
	lastSeenPrice *num.Uint
	orders        map[string]orderAtOffsetStat
	risesAbove    *btree.BTreeG[*offsetsAtPrice]
	fallsBelow    *btree.BTreeG[*offsetsAtPrice]
}

func NewTrailingStopOrders() *TrailingStopOrders {
	return &TrailingStopOrders{
		lastSeenPrice: num.UintZero(),
		orders:        map[string]orderAtOffsetStat{},
		risesAbove:    btree.NewG(2, lessFuncOffsetsAtPrice),
		fallsBelow:    btree.NewG(2, lessFuncOffsetsAtPrice),
	}
}

func (p *TrailingStopOrders) PriceUpdated(newPrice *num.Uint) []string {
	out := []string{}
	// price increased, move trailing prices buckets
	// for fallsBelow and executed triggers for risesAbove
	if p.lastSeenPrice.LT(newPrice) {
		p.adjustBuckets(p.fallsBelow, newPrice)
		out = p.trigger(p.risesAbove, newPrice)
	} else if p.lastSeenPrice.GT(newPrice) {
		p.adjustBuckets(p.risesAbove, newPrice)
		out = p.trigger(p.fallsBelow, newPrice)
	} else {
		// nothing happened
		return nil
	}

	p.lastSeenPrice = newPrice.Clone()

	return out
}

func (p *TrailingStopOrders) adjustBuckets(
	tree *btree.BTreeG[*offsetsAtPrice],
	price *num.Uint,
) {

}

func (p *TrailingStopOrders) trigger(
	tree *btree.BTreeG[*offsetsAtPrice],
	price *num.Uint,
) []string {
	return nil
}

func (p *TrailingStopOrders) Insert(
	id string, offset num.Decimal, direction types.StopOrderTriggerDirection,
) {
	p.orders[id] = orderAtOffsetStat{offset, direction}

	switch direction {
	case types.StopOrderTriggerDirectionFallsBelow:
		p.insertOrUpdateOffsetAtPrice(p.fallsBelow, id, p.lastSeenPrice.Clone(), offset)
	case types.StopOrderTriggerDirectionRisesAbove:
		p.insertOrUpdateOffsetAtPrice(p.risesAbove, id, p.lastSeenPrice.Clone(), offset)
	}
}

func (p *TrailingStopOrders) insertOrUpdateOffsetAtPrice(
	tree *btree.BTreeG[*offsetsAtPrice],
	id string,
	price *num.Uint,
	offset num.Decimal,
) {
	oap, ok := tree.Get(&offsetsAtPrice{price: price})
	if !ok {
		oap = &offsetsAtPrice{
			price:   price,
			offsets: btree.NewG(2, lessFuncOrdersAtOffset),
		}
	}

	// add to the list
	p.insertOrUpdateOrderAtOffset(oap.offsets, id, offset)

	// finally insert or whatever
	tree.ReplaceOrInsert(oap)
}

func (p *TrailingStopOrders) insertOrUpdateOrderAtOffset(
	tree *btree.BTreeG[*ordersAtOffset],
	id string,
	offset num.Decimal,
) {
	oap, ok := tree.Get(&ordersAtOffset{offset: offset})
	if !ok {
		oap = &ordersAtOffset{offset: offset}
	}

	// add to the list
	oap.orders = append(oap.orders, id)

	// finally insert or whatever
	tree.ReplaceOrInsert(oap)
}

func (p *TrailingStopOrders) Remove(id string) {}

func (p *TrailingStopOrders) DumpRisesAbove() string {
	return dumpTree(p.risesAbove)
}

func (p *TrailingStopOrders) DumpFallsBelow() string {
	return dumpTree(p.fallsBelow)
}
