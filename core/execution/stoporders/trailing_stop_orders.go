package stoporders

import (
	"fmt"

	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	v1 "code.vegaprotocol.io/vega/protos/vega/snapshot/v1"
	"github.com/google/btree"
	"golang.org/x/exp/slices"
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
		orders:     map[string]orderAtOffsetStat{},
		risesAbove: btree.NewG(2, lessFuncOffsetsAtPrice),
		fallsBelow: btree.NewG(2, lessFuncOffsetsAtPrice),
	}
}

func NewTrailingStopOrdersFromProto(p *v1.TrailingStopOrders) *TrailingStopOrders {
	tso := NewTrailingStopOrders()

	if len(p.LastSeenPrice) > 0 {
		var overflow bool
		tso.lastSeenPrice, overflow = num.UintFromString(p.LastSeenPrice, 10)
		if overflow {
			panic("lastSeenPrice should always be valid")
		}
	}

	for _, v := range p.FallsBellow {
		price, overflow := num.UintFromString(v.Price, 10)
		if overflow {
			panic(fmt.Sprintf("invalid uint from snapshot, would overflow: %s", v.Price))
		}
		for _, offset := range v.Offsets {
			off, err := num.DecimalFromString(offset.Offset)
			if err != nil {
				panic(fmt.Sprintf("invalid decimal from snapshot: %s", offset.Offset))
			}
			for _, oid := range offset.Orders {
				tso.insertAtPrice(
					oid, off, price, types.StopOrderTriggerDirectionFallsBelow)
			}
		}
	}

	for _, v := range p.RisesAbove {
		price, overflow := num.UintFromString(v.Price, 10)
		if overflow {
			panic(fmt.Sprintf("invalid uint from snapshot, would overflow: %s", v.Price))
		}
		for _, offset := range v.Offsets {
			off, err := num.DecimalFromString(offset.Offset)
			if err != nil {
				panic(fmt.Sprintf("invalid decimal from snapshot: %s", offset.Offset))
			}
			for _, oid := range offset.Orders {
				tso.insertAtPrice(
					oid, off, price, types.StopOrderTriggerDirectionRisesAbove)
			}
		}
	}

	return tso
}

func (p *TrailingStopOrders) ToProto() *v1.TrailingStopOrders {
	var lastSeenPrice string
	if p.lastSeenPrice != nil {
		lastSeenPrice = p.lastSeenPrice.String()
	}

	return &v1.TrailingStopOrders{
		LastSeenPrice: lastSeenPrice,
		FallsBellow:   p.serialize(p.fallsBelow),
		RisesAbove:    p.serialize(p.risesAbove),
	}
}

func (p *TrailingStopOrders) serialize(
	tree *btree.BTreeG[*offsetsAtPrice],
) []*v1.OffsetsAtPrice {
	out := []*v1.OffsetsAtPrice{}

	tree.Ascend(func(item *offsetsAtPrice) bool {
		offsets := []*v1.OrdersAtOffset{}

		item.offsets.Ascend(func(item *ordersAtOffset) bool {
			offsets = append(offsets, &v1.OrdersAtOffset{
				Offset: item.offset.String(),
				Orders: slices.Clone(item.orders),
			})
			return true
		})

		out = append(out, &v1.OffsetsAtPrice{
			Price:   item.price.String(),
			Offsets: offsets,
		})
		return true
	})

	return out
}

func (p *TrailingStopOrders) PriceUpdated(newPrice *num.Uint) []string {
	// short circuit for very first update,
	// not much to do here, just set the newPrice
	// and move on
	if p.lastSeenPrice == nil {
		p.lastSeenPrice = newPrice.Clone()
		return nil
	}

	var out []string
	// price increased, move trailing prices buckets
	// for fallsBelow and executed triggers for risesAbove
	if p.lastSeenPrice.LT(newPrice) {
		p.adjustBuckets(p.fallsBelow, p.fallsBelow.AscendLessThan, newPrice)
		out = p.trigger(
			p.risesAbove,
			p.risesAbove.Ascend,
			func(a *num.Uint, b *num.Uint) bool { return a.GT(b) },
			func(a num.Decimal, b num.Decimal) bool { return a.GreaterThan(b) },
			func(a num.Decimal, offset num.Decimal) num.Decimal { return a.Add(a.Mul(offset)) },
			newPrice,
		)
	} else if p.lastSeenPrice.GT(newPrice) {
		p.adjustBuckets(p.risesAbove, p.risesAbove.DescendGreaterThan, newPrice)
		out = p.trigger(
			p.fallsBelow,
			p.fallsBelow.Descend,
			func(a *num.Uint, b *num.Uint) bool { return a.LT(b) },
			func(a num.Decimal, b num.Decimal) bool { return a.LessThan(b) },
			func(a num.Decimal, offset num.Decimal) num.Decimal { return a.Sub(a.Mul(offset)) },
			newPrice,
		)
	} else {
		// nothing happened
		return nil
	}

	// remove orders from the mapping
	for _, v := range out {
		delete(p.orders, v)
	}

	p.lastSeenPrice = newPrice.Clone()

	return out
}

func (p *TrailingStopOrders) adjustBuckets(
	tree *btree.BTreeG[*offsetsAtPrice],
	findFn func(*offsetsAtPrice, btree.ItemIteratorG[*offsetsAtPrice]),
	newPrice *num.Uint,
) {
	// first we get all prices to adjust
	item := &offsetsAtPrice{price: newPrice}
	pricesToAdjust := []*num.Uint{}
	findFn(item, func(oap *offsetsAtPrice) bool {
		pricesToAdjust = append(pricesToAdjust, oap.price.Clone())
		return true
	})

	// now for each of them, we pull the orders, and insert them in the new price.
	for _, price := range pricesToAdjust {
		current := &offsetsAtPrice{price: price}
		// no error to check, we just iterated over,
		// impossible we would not find it.
		oap, _ := tree.Get(current)

		// now for each orders of every leaf we can add at the new price
		oap.offsets.Ascend(func(oao *ordersAtOffset) bool {
			for _, order := range oao.orders {
				// update the mapping
				prevOrderAtOffsetStat := p.orders[order]
				p.orders[order] = orderAtOffsetStat{oao.offset, prevOrderAtOffsetStat.direction}
				// insert
				p.insertOrUpdateOffsetAtPrice(
					tree, order, newPrice, oao.offset,
				)
			}
			return true
		})

		// now we delete this one, it's not needed anymore
		_, _ = tree.Delete(current)
	}
}

func (p *TrailingStopOrders) trigger(
	tree *btree.BTreeG[*offsetsAtPrice],
	iterateFn func(btree.ItemIteratorG[*offsetsAtPrice]),
	cmpPriceFn func(*num.Uint, *num.Uint) bool,
	cmpOffsetFn func(num.Decimal, num.Decimal) bool,
	applyOffsetFn func(num.Decimal, num.Decimal) num.Decimal,
	price *num.Uint,
) (orders []string) {
	priceDec := price.ToDecimal()
	toRemovePrices := []*num.Uint{}
	iterateFn(func(item *offsetsAtPrice) bool {
		if cmpPriceFn(item.price, price) {
			// continue but nothing to do here
			return true
		}

		leafPriceDec := item.price.ToDecimal()

		toRemoveOffsets := []num.Decimal{}
		// now in here, we iterate all the
		item.offsets.Ascend(func(item *ordersAtOffset) bool {
			offsetedPrice := applyOffsetFn(leafPriceDec, item.offset)
			if cmpOffsetFn(offsetedPrice, priceDec) {
				// we have still margin, no need to process the others
				return false
			}

			toRemoveOffsets = append(toRemoveOffsets, item.offset)

			return true
		})

		// now remove all
		for _, o := range toRemoveOffsets {
			oao, _ := item.offsets.Delete(&ordersAtOffset{offset: o})
			// add to the list of orders
			orders = append(orders, oao.orders...)
		}

		// now we check if the offsets at the price have been depleted,
		// and add them to the list to eventually remove
		if item.offsets.Len() <= 0 {
			toRemovePrices = append(toRemovePrices, item.price.Clone())
		}

		return true
	})

	// now we remove all depleted prices
	for _, p := range toRemovePrices {
		_, _ = tree.Delete(&offsetsAtPrice{price: p})
	}

	return orders
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

func (p *TrailingStopOrders) insertAtPrice(
	id string, offset num.Decimal, price *num.Uint, direction types.StopOrderTriggerDirection,
) {
	p.orders[id] = orderAtOffsetStat{offset, direction}

	switch direction {
	case types.StopOrderTriggerDirectionFallsBelow:
		p.insertOrUpdateOffsetAtPrice(p.fallsBelow, id, price.Clone(), offset)
	case types.StopOrderTriggerDirectionRisesAbove:
		p.insertOrUpdateOffsetAtPrice(p.risesAbove, id, price.Clone(), offset)
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

func (p *TrailingStopOrders) Remove(id string) error {
	o, ok := p.orders[id]
	if !ok {
		return ErrOrderNotFound
	}

	// we can remove from the map now
	delete(p.orders, id)

	switch o.direction {
	case types.StopOrderTriggerDirectionFallsBelow:
		p.remove(p.fallsBelow, id, o.offset)
	case types.StopOrderTriggerDirectionRisesAbove:
		p.remove(p.risesAbove, id, o.offset)
	}

	return nil
}

func (p *TrailingStopOrders) remove(
	tree *btree.BTreeG[*offsetsAtPrice],
	id string,
	offset num.Decimal,
) {
	var deletePrice *num.Uint
	tree.Ascend(func(item *offsetsAtPrice) bool {
		innerItem := &ordersAtOffset{offset: offset}
		// does that offset exists at that price?
		oao, ok := item.offsets.Get(innerItem)
		if !ok {
			return true // nope, keep moving
		}

		continu := true
		for n, v := range oao.orders {
			// we found our order!
			if v == id {
				oao.orders = slices.Delete(oao.orders, n, n+1)
				continu = false
				break
			}
		}

		if len(oao.orders) <= 0 {
			item.offsets.Delete(innerItem)
		}

		if item.offsets.Len() <= 0 {
			deletePrice = item.price.Clone()
		}

		return continu
	})

	if deletePrice != nil {
		tree.Delete(&offsetsAtPrice{price: deletePrice})
	}
}

func (p *TrailingStopOrders) DumpRisesAbove() string {
	return dumpTree(p.risesAbove)
}

func (p *TrailingStopOrders) DumpFallsBelow() string {
	return dumpTree(p.fallsBelow)
}
