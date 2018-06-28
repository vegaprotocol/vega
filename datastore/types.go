package datastore

import "vega/proto"

type Trade struct {
	ID        string
	Market    string
	Price     uint64
	Size      uint64
	Buyer     string
	Seller    string
	Side      int32
	Timestamp uint64
	OrderID   string
}

func (tr *Trade) FromProtoMessage(m msg.Trade, orderID string) *Trade {
	return &Trade{
		ID:        m.Id,
		Market:    m.Market,
		Price:     m.Price,
		Size:      m.Size,
		Buyer:     m.Buyer,
		Seller:    m.Seller,
		Side:      int32(m.Aggressor),
		Timestamp: m.Timestamp,
		OrderID:   orderID,
	}
}

func (tr *Trade) ToProtoMessage() *msg.Trade {
	return &msg.Trade{
		Id:        tr.ID,
		Market:    tr.Market,
		Price:     tr.Price,
		Size:      tr.Size,
		Buyer:     tr.Buyer,
		Seller:    tr.Seller,
		Aggressor: msg.Side(tr.Side),
		Timestamp: tr.Timestamp,
	}
}

type Order struct {
	ID        string
	Market    string
	Party     string
	Side      int32
	Remaining uint64
	Price     uint64
	Size      uint64
	Type      int32
	Timestamp uint64
	Status    int32
}

func (or *Order) FromProtoMessage(m msg.Order) *Order {
	return &Order{
		ID:        m.Id,
		Market:    m.Market,
		Party:     m.Party,
		Price:     m.Price,
		Size:      m.Size,
		Remaining: m.Remaining,
		Timestamp: m.Timestamp,
		Side:      int32(m.Side),
		Type:      int32(m.Type),
		Status:    int32(m.Status),
	}
}

func (or *Order) ToProtoMessage() *msg.Order {
	return &msg.Order{
		Id:        or.ID,
		Market:    or.Market,
		Party:     or.Party,
		Price:     or.Price,
		Size:      or.Size,
		Remaining: or.Remaining,
		Timestamp: or.Timestamp,
		Side:      msg.Side(or.Side),
		Status:    msg.Order_Status(or.Status),
		Type:      msg.Order_Type(or.Type),
	}
}