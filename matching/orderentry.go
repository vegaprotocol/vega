package matching

import (
	"fmt"
	"vega/proto"

	"github.com/golang/protobuf/proto"
	"golang.org/x/crypto/sha3"
)

// Calculate the hash (ID) of the order details (as serialised by protobufs)
func DigestOrderMessage(order *msg.Order) string {
	bytes, _ := proto.Marshal(order)
	hash := make([]byte, 64)
	sha3.ShakeSum256(hash, bytes)
	return fmt.Sprintf("%x", hash)
}

// Work out which of the aggressive & passive orders is the buyer/seller
func getOrderForSide(side msg.Side, agg, pass *msg.Order) *msg.Order {
	if agg.Side == pass.Side {
		panic(fmt.Sprintf("agg.side == pass.side (agg: %v, pass: %v)", agg, pass))
	} else if agg.Side == side {
		return agg
	} else { // pass.side == side
		return pass
	}
}
