package dto

import (
	types "code.vegaprotocol.io/vega/proto"

	"github.com/golang/protobuf/proto"
	"github.com/shopspring/decimal"
)

type Order struct {
	types.Order
	Price decimal.Decimal
}

func (o *Order) FromProto(po *types.Order) error {
	o.Order = *po
	return o.Price.UnmarshalText(po.Price)
}

func (o *Order) Proto() *types.Order {
	o.Order.Price, _ = o.Price.MarshalText()
	return &o.Order
}

func (o *Order) Marshal() ([]byte, error) {
	priceBytes, err := o.Price.MarshalText()
	if err != nil {
		return nil, err
	}
	o.Order.Price = priceBytes

	return proto.Marshal(&o.Order)
}

func (o *Order) Unmarshal(buf []byte) error {
	o.Order.Reset()
	err := proto.Unmarshal(buf, &o.Order)
	if err != nil {
		return err
	}

	return o.Price.UnmarshalText(o.Order.Price)
}
