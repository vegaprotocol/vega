package matching

import (
	"container/list"
	"fmt"
	"vega/proto"

	"github.com/golang/protobuf/proto"
	"golang.org/x/crypto/sha3"
)

// Wraps the protobuf Order message for inclusion in the order book
type OrderEntry struct {
	order      *msg.Order
	Side msg.Side
	//book       *OrderBook
	//side       *OrderBookSide
	//priceLevel *PriceLevel
	elem       *list.Element
	persist    bool
	dispatchChannels []chan msg.Order
}

func newOrderEntry(orderMessage *msg.Order, dispatchChannels []chan msg.Order) *OrderEntry {
	o := &OrderEntry{
		Side: orderMessage.Side,
		order:   orderMessage,
		persist: orderMessage.Type == msg.Order_GTC || orderMessage.Type == msg.Order_GTT,
		dispatchChannels: dispatchChannels,
	}
	o.order.Id = o.Digest()
	return o
}

//func (o *OrderEntry) GetBook() *OrderBook {
//	return o.book
//}

// Creates an order entry from an order message
//func orderEntryFromMessage(order *msg.Order) *OrderEntry {
//	o := &OrderEntry{
//		order:   order,
//		persist: order.Type == msg.Order_GTC || order.Type == msg.Order_GTT,
//	}
//	order.Id = ""
//	order.Id = o.Digest()
//	return o
//}

// Returns true if the order is crossed (can trade) with the supplied side and price
//func (o *OrderEntry) crossedWith(side msg.Side, price uint64) bool {
//	return o.order.GetSide() != side &&
//		price > 0 &&
//		o.order.Price > 0 &&
//		((side == msg.Side_Buy && price >= o.order.Price) ||
//			(side == msg.Side_Sell && price <= o.order.Price))
//}

// Update (remaining size, etc.) for an order that has traded
func (o *OrderEntry) updateRemaining(tradeSize uint64) {

	o.order.Remaining -= tradeSize

	for _, c := range o.dispatchChannels {
		c <- *o.order
	}
}

// Remove an order from the book and update the book metrics
//func (o *OrderEntry) remove() *OrderEntry {
//	if o.priceLevel == nil {
//		return nil
//	}
//	//book := o.book
//	//delete(book.orders, o.order.Id)
//	removeOrder
//	o.priceLevel.removeOrder(o)
//	if !book.config.Quiet {
//		log.Printf("Removed: %v\n", o)
//	}
//	return o
//}

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
	return "[order/" + o.order.Id[-0:5] + "] " + OrderString(o.order)
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
