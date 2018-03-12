package matching

import (
	"fmt"

	"proto"

	"github.com/golang/protobuf/proto"
	"golang.org/x/crypto/sha3"
)

type Trade struct {
	id    string
	price uint64
	size  uint64
	agg   *OrderEntry
	pass  *OrderEntry
	buy   *OrderEntry
	sell  *OrderEntry
	msg   *msg.Trade
}

// Returns the min of 2 uint64s
func min(x, y uint64) uint64 {
	if y < x {
		return y
	}
	return x
}

// Creates a trade of a given size between two orders and updates the order details
func newTrade(agg, pass *OrderEntry, size uint64) *Trade {
	trade := &Trade{
		price: pass.order.Price,
		size:  size,
		agg:   agg,
		pass:  pass,
		buy:   getOrderForSide(msg.Side_Buy, agg, pass),
		sell:  getOrderForSide(msg.Side_Sell, agg, pass),
	}
	pass.update(trade)
	agg.update(trade)
	trade.id = trade.Digest()
	return trade
}

// Returns a string representation of a trade
func (t Trade) String() string {
	var aggressiveAction string
	if t.agg.order.Side == msg.Side_Buy {
		aggressiveAction = "buys from"
	} else {
		aggressiveAction = "sells to"
	}
	return fmt.Sprintf(
		"[trade/%v] %v %v %v: %v at %v",
		t.id[0:5],
		t.agg.order.Party,
		aggressiveAction,
		t.pass.order.Party,
		t.size,
		t.price)
}

// Returns the protobufs message object for a trade
func (t *Trade) toMessage() *msg.Trade {
	if t.msg == nil {
		t.msg = &msg.Trade{
			Price:     t.price,
			Size:      t.size,
			Buyer:     t.buy.order.Party,
			Seller:    t.sell.order.Party,
			Aggressor: t.agg.order.Side,
		}
	}
	return t.msg
}

// Calculate the hash (ID) of the trade details (as serialised by protobufs)
func (t *Trade) Digest() string {
	bytes, _ := proto.Marshal(t.toMessage())
	hash := make([]byte, 64)
	sha3.ShakeSum256(hash, bytes)
	return fmt.Sprintf("%x", hash)
}
