package datastore

import (
	"testing"
	"vega/msg"

	"github.com/stretchr/testify/assert"
)

func TestOrderModelFromProtoMessage(t *testing.T) {

	in := &msg.Order{
		Id:         "d41d8cd98f00b204e9800998ecf8427e",
		Market:     "market",
		Party:      "party",
		Side:       1,
		Price:      10,
		Size:       1,
		Remaining:  50,
		Type:       1,
		Timestamp:  1,
	}

	out := &Order{
		Order: msg.Order{
			Id:         "d41d8cd98f00b204e9800998ecf8427e",
			Market:     "market",
			Party:      "party",
			Side:       1,
			Price:      10,
			Size:       1,
			Remaining:  50,
			Type:       1,
			Timestamp:  1,
		},
	}

	order := NewOrderFromProtoMessage(in)
	assert.Equal(t, out, order)
}

func TestOrderModelToProtoMessage(t *testing.T) {

	in := &Order{
		Order: msg.Order{
			Id:         "d41d8cd98f00b204e9800998ecf8427e",
			Market:     "market",
			Party:      "party",
			Side:       1,
			Price:      10,
			Size:       1,
			Remaining:  50,
			Type:       1,
			Timestamp:  1,
		},
	}

	out := &msg.Order{
		Id:         "d41d8cd98f00b204e9800998ecf8427e",
		Market:     "market",
		Party:      "party",
		Side:       1,
		Price:      10,
		Size:       1,
		Remaining:  50,
		Type:       1,
		Timestamp:  1,
	}

	order := in.ToProtoMessage()
	assert.Equal(t, out, order)
}

func TestTradeModelFromProtoMessage(t *testing.T) {
	in := &msg.Trade{
		Id:        "d41d8cd98f00b204e9800998ecf8427e",
		Market:    "market",
		Price:     50,
		Size:      1000,
		Buyer:     "buyer",
		Seller:    "seller",
		Aggressor: 1,
		Timestamp: 3,
	}

	out := &Trade{
		Trade: msg.Trade{
			Id:        "d41d8cd98f00b204e9800998ecf8427e",
			Market:    "market",
			Price:     50,
			Size:      1000,
			Buyer:     "buyer",
			Seller:    "seller",
			Aggressor: 1,
			Timestamp: 3,
		},
		AggressiveOrderId: "035ed2311b96d2a65ec6a6fe71046c1",
		PassiveOrderId:    "035ed2311b96d2a65ec6a6fe71046c14",
	}

	trade := NewTradeFromProtoMessage(in, "035ed2311b96d2a65ec6a6fe71046c1", "035ed2311b96d2a65ec6a6fe71046c14")
	assert.Equal(t, out, trade)
}

func TestTradeModelToProtoMessage(t *testing.T) {
	in := &Trade{
		Trade: msg.Trade{
			Id:        "d41d8cd98f00b204e9800998ecf8427e",
			Market:    "market",
			Price:     50,
			Size:      1000,
			Buyer:     "buyer",
			Seller:    "seller",
			Aggressor: 1,
			Timestamp: 3,
		},
		PassiveOrderId:    "035ed2311b96d2a65ec6a6fe71046c14",
		AggressiveOrderId: "035ed2311b96d2a65ec6a6fe71046c1",
	}

	out := &msg.Trade{
		Id:        "d41d8cd98f00b204e9800998ecf8427e",
		Market:    "market",
		Price:     50,
		Size:      1000,
		Buyer:     "buyer",
		Seller:    "seller",
		Aggressor: 1,
		Timestamp: 3,
	}

	trade := in.ToProtoMessage()
	assert.Equal(t, out, trade)
}
