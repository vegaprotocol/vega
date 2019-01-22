package blockchain

import (
	"github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/assert"
	"testing"
	"vega/msg"
)

func TestDecodeInvalidPayload(t *testing.T) {
	invalidBytes := []byte{10, 20, 30, 40}

	processor := abciProcessor{}
	decodeBytes, cmd, err := processor.txDecode(invalidBytes)

	t.Log(decodeBytes)
	t.Log(cmd)
	t.Log(err)

	assert.Error(t, err)
}

func TestEncodeAndDecodeWithCreateOrderCommand(t *testing.T) {
	order := &msg.Order{
		Id:     "V9-120",
		Market: "BTC/DEC18",
		Party:  "PartyA",
	}

	orderBytes, err := proto.Marshal(order)
	assert.Nil(t, err)

	client := NewClient()
	resultBytes, err := client.txEncode(orderBytes, SubmitOrderCommand)
	assert.Nil(t, err)

	processor := abciProcessor{}
	decodeBytes, cmd, err := processor.txDecode(resultBytes)
	assert.Equal(t, SubmitOrderCommand, cmd)

	resultOrder := &msg.Order{}
	err = proto.Unmarshal(decodeBytes, resultOrder)
	assert.Nil(t, err)

	assert.Equal(t, "V9-120", resultOrder.Id)
	assert.Equal(t, "BTC/DEC18", resultOrder.Market)
	assert.Equal(t, "PartyA", resultOrder.Party)
}
