package matching

import (
	"fmt"
	"vega/proto"

	"github.com/golang/protobuf/proto"
	"golang.org/x/crypto/sha3"
)

type Trade struct {
	id    string
	price uint64
	size  uint64
	agg   *msg.Order
	pass  *msg.Order
	buy   *msg.Order
	sell  *msg.Order
	msg   *msg.Trade
}

// Creates a trade of a given size between two orders and updates the order details
func newTrade(agg, pass *msg.Order, size uint64) *Trade {
	var buyer, seller *msg.Order
	if agg.Side == msg.Side_Buy {
		buyer = agg
		seller = pass
 	} else {
 		buyer = pass
 		seller = agg
	}

	if agg.Side == pass.Side {
		panic(fmt.Sprintf("agg.side == pass.side (agg: %v, pass: %v)", agg, pass))
	}

	trade := &Trade{
		price: pass.Price,
		size:  size,
		agg:   agg,
		pass:  pass,
		buy:   buyer,
		sell:  seller,
	}
	trade.id = trade.Digest()

	//log.Printf("Matched: %v\n", trade)
	return trade
}

// Calculate the hash (ID) of the trade details (as serialised by protobufs)
func (t *Trade) Digest() string {
	bytes, _ := proto.Marshal(t.toMessage())
	hash := make([]byte, 64)
	sha3.ShakeSum256(hash, bytes)
	return fmt.Sprintf("%x", hash)
}


// Returns a string representation of a trade
func (t *Trade) String() string {
	var aggressiveAction string
	if t.agg.Side == msg.Side_Buy {
		aggressiveAction = "buys from"
	} else {
		aggressiveAction = "sells to"
	}
	return fmt.Sprintf(
		"[trade/%v] %v %v %v: %v at %v",
		t.id[0:5],
		t.agg.Party,
		aggressiveAction,
		t.pass.Party,
		t.size,
		t.price)
}

// Returns the protobufs message object for a trade
func (t *Trade) toMessage() *msg.Trade {
	if t.msg == nil {
		t.msg = &msg.Trade{
			Price:     t.price,
			Size:      t.size,
			Buyer:     t.buy.Party,
			Seller:    t.sell.Party,
			Aggressor: t.agg.Side,
		}
	}
	return t.msg
}
