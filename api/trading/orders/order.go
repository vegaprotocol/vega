package orders

import (
	"encoding/json"
	"encoding/base64"
)

type Order struct {
	Market    string   `xml:"market" json:"market" binding:"required"`
	Party     string   `xml:"party" json:"party"`
	Side      int32    `xml:"side" json:"side"`
	Price     uint64   `xml:"price" json:"price"`
	Size      uint64   `xml:"size" json:"size" `
	Remaining uint64   `xml:"remaining" json:"remaining"`
	Timestamp uint64   `xml:"timestamp" json:"timestamp"`
	Type      int      `xml:"type" json:"type"`
}

func NewOrder(
	market string,
	party string,
	side int32,
	price uint64,
	size uint64,
	remaining uint64,
	timestamp uint64,
	tradeType int,
) Order {
	return Order {
		market,
		party,
		side,
		price,
		size,
		remaining,
		timestamp,
		tradeType,
	}
}

func (o *Order) Json() ([]byte, error) {
	return json.Marshal(o)
}

func (o *Order) JsonWithEncoding() (string, error) {
	json, err := o.Json()
	if err != nil {
		return "", err
	}
	encoded := base64.StdEncoding.EncodeToString(json)
	return encoded, err
}

