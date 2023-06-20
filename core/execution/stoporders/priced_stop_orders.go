package stoporders

import (
	"fmt"

	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	v1 "code.vegaprotocol.io/vega/protos/vega/snapshot/v1"
	"golang.org/x/exp/slices"

	"github.com/google/btree"
)

type ordersAtPrice struct {
	price  *num.Uint
	orders []string
}

func (o *ordersAtPrice) String() string {
	return fmt.Sprintf("%s:%v", o.price.String(), o.orders)
}

func lessFuncOrdersAtPrice(a, b *ordersAtPrice) bool {
	return a.price.LT(b.price)
}

type orderAtPriceStat struct {
	price     *num.Uint
	direction types.StopOrderTriggerDirection
}

type PricedStopOrders struct {
	// mapping table for stop order ID
	// help finding them back in the trees.
	orders map[string]orderAtPriceStat

	fallsBelow *btree.BTreeG[*ordersAtPrice]
	risesAbove *btree.BTreeG[*ordersAtPrice]
}

func NewPricedStopOrders() *PricedStopOrders {
	return &PricedStopOrders{
		orders:     map[string]orderAtPriceStat{},
		fallsBelow: btree.NewG(2, lessFuncOrdersAtPrice),
		risesAbove: btree.NewG(2, lessFuncOrdersAtPrice),
	}
}

func NewPricedStopOrdersFromProto(p *v1.PricedStopOrders) *PricedStopOrders {
	pso := NewPricedStopOrders()

	for _, v := range p.FallsBellow {
		for _, oid := range v.Orders {
			price, overflow := num.UintFromString(v.Price, 10)
			if overflow {
				panic(fmt.Sprintf("invalid uint from snapshot, would overflow: %s", v.Price))
			}
			pso.Insert(oid, price, types.StopOrderTriggerDirectionFallsBelow)
		}
	}

	for _, v := range p.RisesAbove {
		for _, oid := range v.Orders {
			price, overflow := num.UintFromString(v.Price, 10)
			if overflow {
				panic(fmt.Sprintf("invalid uint from snapshot, would overflow: %s", v.Price))
			}
			pso.Insert(oid, price, types.StopOrderTriggerDirectionRisesAbove)
		}
	}

	return pso
}

func (p *PricedStopOrders) ToProto() *v1.PricedStopOrders {
	return &v1.PricedStopOrders{
		FallsBellow: p.serialize(p.fallsBelow),
		RisesAbove:  p.serialize(p.risesAbove),
	}
}

func (p *PricedStopOrders) serialize(tree *btree.BTreeG[*ordersAtPrice]) []*v1.OrdersAtPrice {
	out := []*v1.OrdersAtPrice{}

	tree.Ascend(func(item *ordersAtPrice) bool {
		out = append(out, &v1.OrdersAtPrice{
			Price:  item.price.String(),
			Orders: slices.Clone(item.orders),
		})
		return true
	})

	return out
}

func (p *PricedStopOrders) PriceUpdated(newPrice *num.Uint) []string {
	// first remove if price fallsBelow
	orderIDs := p.trigger(
		p.fallsBelow,
		p.fallsBelow.DescendGreaterThan,
		newPrice,
	)

	// then if it rises above?
	orderIDs = append(orderIDs,
		p.trigger(
			p.risesAbove,
			p.risesAbove.AscendLessThan,
			newPrice,
		)...,
	)

	// here we can cleanup the mapping table as well
	for _, v := range orderIDs {
		delete(p.orders, v)
	}

	return orderIDs
}

func (p *PricedStopOrders) trigger(
	tree *btree.BTreeG[*ordersAtPrice],
	findFn func(pivot *ordersAtPrice, iterator btree.ItemIteratorG[*ordersAtPrice]),
	newPrice *num.Uint,
) []string {
	orderIDs := []string{}
	toDelete := []*num.Uint{}
	findFn(&ordersAtPrice{price: newPrice}, func(item *ordersAtPrice) bool {
		orderIDs = append(orderIDs, item.orders...)
		toDelete = append(toDelete, item.price)
		return true
	})

	// now we delete all the unused tree item
	for _, p := range toDelete {
		tree.Delete(&ordersAtPrice{price: p})
	}

	return orderIDs
}

func (p *PricedStopOrders) Insert(
	id string, price *num.Uint, direction types.StopOrderTriggerDirection,
) {
	p.orders[id] = orderAtPriceStat{price.Clone(), direction}

	switch direction {
	case types.StopOrderTriggerDirectionFallsBelow:
		p.insertOrUpdate(p.fallsBelow, id, price.Clone())
	case types.StopOrderTriggerDirectionRisesAbove:
		p.insertOrUpdate(p.risesAbove, id, price.Clone())
	}
}

func (p *PricedStopOrders) insertOrUpdate(
	tree *btree.BTreeG[*ordersAtPrice], id string, price *num.Uint,
) {
	oap, ok := tree.Get(&ordersAtPrice{price: price})
	if !ok {
		oap = &ordersAtPrice{price: price}
	}

	// add to the list
	oap.orders = append(oap.orders, id)

	// finally insert or whatever
	tree.ReplaceOrInsert(oap)
}

func (p *PricedStopOrders) Remove(id string) error {
	oaps, ok := p.orders[id]
	if !ok {
		return ErrOrderNotFound
	}

	delete(p.orders, id)

	var err error
	switch oaps.direction {
	case types.StopOrderTriggerDirectionFallsBelow:
		err = p.removeAndMaybeDelete(p.fallsBelow, id, oaps.price)
	case types.StopOrderTriggerDirectionRisesAbove:
		err = p.removeAndMaybeDelete(p.risesAbove, id, oaps.price)
	}

	return err
}

func (p *PricedStopOrders) removeAndMaybeDelete(
	tree *btree.BTreeG[*ordersAtPrice], id string, price *num.Uint,
) error {
	// just declare it first, we may reuse it by the end
	item := &ordersAtPrice{price: price}

	oap, ok := tree.Get(item)
	if !ok {
		return ErrPriceNotFound
	}

	before := len(oap.orders)

	for n, v := range oap.orders {
		// this is our ID
		if v == id {
			oap.orders = slices.Delete(oap.orders, n, n+1)
			break
		}
	}

	// didn't found the order, we can just panic it should never happen
	if before == len(oap.orders) {
		panic("order not in tree but in mapping table")
	}

	// now if the len is 0, we probably don't need that
	// price level anymore
	if len(oap.orders) <= 0 {
		tree.Delete(item)
	}

	return nil
}

func (p *PricedStopOrders) DumpRisesAbove() string {
	return dumpTree(p.risesAbove)
}

func (p *PricedStopOrders) DumpFallsBelow() string {
	return dumpTree(p.fallsBelow)
}
