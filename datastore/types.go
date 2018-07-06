package datastore

import "vega/proto"

type Trade struct {
	msg.Trade
	OrderId string
}

func NewTradeFromProtoMessage(m *msg.Trade, orderId string) *Trade {
	return &Trade{
		Trade:   *m,
		OrderId: orderId,
	}
}

func (tr *Trade) ToProtoMessage() *msg.Trade {
	return &msg.Trade{
		Id:        tr.Id,
		Market:    tr.Market,
		Price:     tr.Price,
		Size:      tr.Size,
		Buyer:     tr.Buyer,
		Seller:    tr.Seller,
		Aggressor: tr.Aggressor,
		Timestamp: tr.Timestamp,
	}
}

type Order struct {
	msg.Order
}

func NewOrderFromProtoMessage(m *msg.Order) *Order {
	return &Order{
		Order: *m,
	}
}

func (or *Order) ToProtoMessage() *msg.Order {
	return &msg.Order{
		Id:        or.Id,
		Market:    or.Market,
		Party:     or.Party,
		Price:     or.Price,
		Size:      or.Size,
		Remaining: or.Remaining,
		Timestamp: or.Timestamp,
		Side:      or.Side,
		Type:      or.Type,
	}
}
