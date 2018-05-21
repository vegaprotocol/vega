package orders

import (
	"encoding/json"
	"encoding/base64"
)

type Order struct {
	Market    string
	Party     string
	Side      int32
	Price     uint64
	Size      uint64
	Remaining uint64
	Timestamp uint64
	Type      int
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

