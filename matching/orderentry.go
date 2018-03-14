package matching

import (
	"container/list"
	"fmt"

	"vega/proto"

	"github.com/gogo/protobuf/proto"
	"golang.org/x/crypto/sha3"
)

// Wraps the protobuf Order message for inclusion in the order book
type OrderEntry struct {
	order      *msg.Order
	book       *OrderBook
	side       *OrderBookSide
	priceLevel *PriceLevel
	elem       *list.Element
	persist    bool
	id         string
}

func (o *OrderEntry) GetBook() *OrderBook {
	return o.book
}

// Creates an order entry from an order message
func orderFromMessage(order *msg.Order) *OrderEntry {
	o := &OrderEntry{
		order:   order,
		persist: order.Type == msg.Order_GTC || order.Type == msg.Order_GTT,
	}
	o.id = o.Digest()
	return o
}

// Returns true if the order is crossed (can trade) with the supplied side and price
func (o *OrderEntry) crossedWith(side msg.Side, price uint64) bool {
	return o.order.GetSide() != side &&
		price > 0 &&
		o.order.Price > 0 &&
		((side == msg.Side_Buy && price >= o.order.Price) ||
			(side == msg.Side_Sell && price <= o.order.Price))
}

// Update (remaining size, etc.) for an order that has traded
func (o *OrderEntry) update(trade *Trade) {
	if trade.size > o.order.Remaining {
		panic(fmt.Sprintf("Trade.size > order.remaining (o: %v, newTrade: %v)", o, trade))
	} else {
		o.order.Remaining -= trade.size
	}
}

// Remove an order from the book and update the book metrics
func (o *OrderEntry) remove() *OrderEntry {
	if o.priceLevel == nil {
		return nil
	}
	book := o.book
	delete(book.orders, o.id)
	o.priceLevel.removeOrder(o)
	if !book.config.Quiet {
		fmt.Printf("Removed: %v\n", o)
	}
	return o
}

// Returns the string representation of an order's details
func OrderString(o *msg.Order) string {
	return fmt.Sprintf(
		"%v %v/%v @%v (%v)",
		o.Side,
		o.Remaining,
		o.Size,
		o.Price,
		o.Party)
}

// Returns the string representation of an order including its ID
func (o *OrderEntry) String() string {
	return "[order/" + o.id[-0:5] + "] " + OrderString(o.order)
}

// Calculate the hash (ID) of the order details (as serialised by protobufs)
func (o *OrderEntry) Digest() string {
	bytes, _ := proto.Marshal(o.order)
	hash := make([]byte, 64)
	sha3.ShakeSum256(hash, bytes)
	return fmt.Sprintf("%x", hash)
}

// Work out which of the aggressive & passive orders is the buyer/seller
func getOrderForSide(side msg.Side, agg, pass *OrderEntry) *OrderEntry {
	if agg.order.Side == pass.order.Side {
		panic(fmt.Sprintf("agg.side == pass.side (agg: %v, pass: %v)", agg, pass))
	} else if agg.order.Side == side {
		return agg
	} else { // pass.side == side
		return pass
	}
}
