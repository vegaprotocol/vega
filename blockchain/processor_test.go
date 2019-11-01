package blockchain

import (
	"testing"

	types "code.vegaprotocol.io/vega/proto"
	"github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/assert"
)

func TestDecodeInvalidPayload(t *testing.T) {
	invalidBytes := []byte{10, 20, 30, 40}
	decodeBytes, cmd, err := txDecode(invalidBytes)

	t.Log(decodeBytes)
	t.Log(cmd)
	t.Log(err)

	assert.Error(t, err)
}

func TestEncodeAndDecodeWithCreateOrderCommand(t *testing.T) {
	order := &types.Order{
		Id:       "V9-120",
		MarketID: "BTC/DEC18",
		PartyID:  "PartyA",
	}

	orderBytes, err := proto.Marshal(order)
	assert.Nil(t, err)

	resultBytes, err := txEncode(orderBytes, SubmitOrderCommand)
	assert.Nil(t, err)

	decodeBytes, cmd, err := txDecode(resultBytes)
	assert.Equal(t, SubmitOrderCommand, cmd)

	resultOrder := &types.Order{}
	err = proto.Unmarshal(decodeBytes, resultOrder)
	assert.Nil(t, err)

	assert.Equal(t, "V9-120", resultOrder.Id)
	assert.Equal(t, "BTC/DEC18", resultOrder.MarketID)
	assert.Equal(t, "PartyA", resultOrder.PartyID)
}
