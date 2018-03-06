package book

import (
	"container/list"
	"fmt"

	"proto"
)

type wrappedOrder struct {
	order      *pb.Order
	priceLevel *PriceLevel
	elem       *list.Element
	persist    bool
}

func (b *OrderBook) WrapOrder(order *pb.Order) *wrappedOrder {
	return &wrappedOrder{
		order:     order,
		persist:   order.Type == pb.Order_GTC || order.Type == pb.Order_GTT,
	}
}

func (o *wrappedOrder) crossedWith(side pb.Order_Side, price uint64) bool {
	return o.order.GetSide() != side &&
		price > 0 &&
		o.order.Price > 0 &&
		((side == pb.Order_Buy && price >= o.order.Price) ||
			(side == pb.Order_Sell && price <= o.order.Price))
}

func (o *wrappedOrder) update(t *Trade) {
	if t.size > o.order.Remaining {
		panic(fmt.Sprintf("trade.size > order.remaining (o: %v, t: %v)", o, t))
	} else {
		o.order.Remaining -= t.size
	}
}

func (o *wrappedOrder) remove() bool {
	if o.priceLevel == nil {
		return false
	}
	o.priceLevel.volume -= o.order.Remaining
	o.priceLevel.orders.Remove(o.elem)
	o.elem = nil
	o.priceLevel = nil
	return true
}

func OrderString(o *pb.Order) string {
	return fmt.Sprintf(
		"%v %v/%v @%v (%v)",
		o.Side,
		o.Remaining,
		o.Size,
		o.Price,
		o.Party)
}

func (o *wrappedOrder) String() string {
	return OrderString(o.order)
}

func getOrderForSide(side pb.Order_Side, agg, pass *wrappedOrder) *wrappedOrder {
	if agg.order.Side == pass.order.Side {
		panic(fmt.Sprintf("agg.side == pass.side (agg: %v, pass: %v)", agg, pass))
	} else if agg.order.Side == side {
		return agg
	} else { // pass.side == side
		return pass
	}
}
