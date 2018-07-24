package blockchain

import (
	"testing"
	"vega/msg"
	"github.com/gogo/protobuf/proto"
	"github.com/stretchr/testify/assert"
)

func TestVegaTxEncodeAndDecodeWithCreateOrderCommand(t *testing.T) {
	order := &msg.Order{
		Id: "V9-120",
		Market: "BTC/DEC18",
		Party: "PartyA",
	}

	orderBytes, err := proto.Marshal(order)
	assert.Nil(t, err)

	resultBytes, err := VegaTxEncode(orderBytes, CreateOrderCommand)
	assert.Nil(t, err)

	decodeBytes, cmd, err := VegaTxDecode(resultBytes)
	assert.Equal(t, CreateOrderCommand, cmd)

	resultOrder := &msg.Order{}
	err = proto.Unmarshal(decodeBytes, resultOrder)
	assert.Nil(t, err)

	assert.Equal(t, "V9-120", resultOrder.Id)
	assert.Equal(t, "BTC/DEC18", resultOrder.Market)
	assert.Equal(t, "PartyA", resultOrder.Party)
}

