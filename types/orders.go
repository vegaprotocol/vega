//lint:file-ignore ST1003 Ignore underscores in names, this is straigh copied from the proto package to ease introducing the domain types

package types

import (
	"errors"

	proto "code.vegaprotocol.io/protos/vega"
	commandspb "code.vegaprotocol.io/protos/vega/commands/v1"
	"code.vegaprotocol.io/vega/types/num"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

type PriceLevel struct {
	Price          *num.Uint
	NumberOfOrders uint64
	Volume         uint64
}

func NewPriceLevelFromProto(p *proto.PriceLevel) (*PriceLevel, error) {
	price, overflowed := num.UintFromString(p.Price, 10)
	if overflowed {
		return nil, errors.New("invalid amount")
	}
	return &PriceLevel{
		Price:          price,
		NumberOfOrders: p.NumberOfOrders,
		Volume:         p.Volume,
	}, nil
}

func (p PriceLevel) IntoProto() *proto.PriceLevel {
	return &proto.PriceLevel{
		Price:          num.UintToString(p.Price),
		NumberOfOrders: p.NumberOfOrders,
		Volume:         p.Volume,
	}
}

type PriceLevels []*PriceLevel

func (p PriceLevels) IntoProto() []*proto.PriceLevel {
	out := make([]*proto.PriceLevel, 0, len(p))
	for _, v := range p {
		out = append(out, v.IntoProto())
	}
	return out
}

type OrderAmendment struct {
	OrderID         string
	MarketID        string
	Price           *num.Uint
	SizeDelta       int64
	ExpiresAt       *int64 // timestamp
	TimeInForce     OrderTimeInForce
	PeggedOffset    *int64 // *wrappers.Int64Value
	PeggedReference PeggedReference
}

func NewOrderAmendmentFromProto(p *commandspb.OrderAmendment) (*OrderAmendment, error) {
	var (
		price             *num.Uint
		exp, peggedOffset *int64
	)
	if p.Price != nil {
		overflowed := false
		price, overflowed = num.UintFromString(p.Price.Value, 10)
		if overflowed {
			return nil, errors.New("invalid amount")
		}
	}
	if p.ExpiresAt != nil {
		e := p.ExpiresAt.Value
		exp = &e
	}
	if p.PeggedOffset != nil {
		po := p.PeggedOffset.Value
		peggedOffset = &po
	}
	return &OrderAmendment{
		OrderID:         p.OrderId,
		MarketID:        p.MarketId,
		Price:           price,
		SizeDelta:       p.SizeDelta,
		ExpiresAt:       exp,
		TimeInForce:     p.TimeInForce,
		PeggedOffset:    peggedOffset,
		PeggedReference: p.PeggedReference,
	}, nil
}

func (o OrderAmendment) IntoProto() *commandspb.OrderAmendment {
	r := &commandspb.OrderAmendment{
		OrderId:         o.OrderID,
		MarketId:        o.MarketID,
		SizeDelta:       o.SizeDelta,
		TimeInForce:     o.TimeInForce,
		PeggedReference: o.PeggedReference,
	}
	if o.Price != nil {
		r.Price = &proto.Price{
			Value: num.UintToString(o.Price),
		}
	}
	if o.ExpiresAt != nil {
		r.ExpiresAt = &proto.Timestamp{
			Value: *o.ExpiresAt,
		}
	}
	if o.PeggedOffset != nil {
		r.PeggedOffset = &wrapperspb.Int64Value{
			Value: *o.PeggedOffset,
		}
	}
	return r
}

// Validate santiy-checks the order amendment as-is, the market will further validate the amendment
// based on the order it's actually trying to amend
func (o OrderAmendment) Validate() error {
	// check TIME_IN_FORCE and expiry
	if o.TimeInForce == OrderTimeInForceGTT && o.ExpiresAt == nil {
		return OrderErrorCannotAmendToGTTWithoutExpiryAt
	}

	if o.TimeInForce == OrderTimeInForceGTC && o.ExpiresAt != nil {
		// this is cool, but we need to ensure and expiry is not set
		return OrderErrorCannotHaveGTCAndExpiryAt
	}

	if o.TimeInForce == OrderTimeInForceFOK || o.TimeInForce == OrderTimeInForceIOC {
		// IOC and FOK are not acceptable for amend order
		return OrderErrorCannotAmendToFOKOrIOC
	}

	return nil
}

func (o OrderAmendment) String() string {
	return o.IntoProto().String()
}

func (o OrderAmendment) GetOrderId() string {
	return o.OrderID
}

func (o OrderAmendment) GetMarketId() string {
	return o.MarketID
}
