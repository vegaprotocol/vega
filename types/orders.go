//lint:file-ignore ST1003 Ignore underscores in names, this is straigh copied from the proto package to ease introducing the domain types

package types

import (
	proto "code.vegaprotocol.io/data-node/proto/vega"
	commandspb "code.vegaprotocol.io/data-node/proto/vega/commands/v1"
	"code.vegaprotocol.io/data-node/types/num"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

type PriceLevel struct {
	Price          *num.Uint
	NumberOfOrders uint64
	Volume         uint64
}

func NewPriceLevelFromProto(p *proto.PriceLevel) *PriceLevel {
	return &PriceLevel{
		Price:          num.NewUint(p.Price),
		NumberOfOrders: p.NumberOfOrders,
		Volume:         p.Volume,
	}
}

func (p PriceLevel) IntoProto() *proto.PriceLevel {
	return &proto.PriceLevel{
		Price:          p.Price.Uint64(),
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
	OrderId         string
	MarketId        string
	Price           *num.Uint
	SizeDelta       int64
	ExpiresAt       *int64 // timestamp
	TimeInForce     Order_TimeInForce
	PeggedOffset    *int64 // *wrappers.Int64Value
	PeggedReference PeggedReference
}

func NewOrderAmendmentFromProto(p *commandspb.OrderAmendment) *OrderAmendment {
	var (
		price             *num.Uint
		exp, peggedOffset *int64
	)
	if p.Price != nil {
		price = num.NewUint(p.Price.Value)
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
		OrderId:         p.OrderId,
		MarketId:        p.MarketId,
		Price:           price,
		SizeDelta:       p.SizeDelta,
		ExpiresAt:       exp,
		TimeInForce:     p.TimeInForce,
		PeggedOffset:    peggedOffset,
		PeggedReference: p.PeggedReference,
	}
}

func (o OrderAmendment) IntoProto() *commandspb.OrderAmendment {
	r := &commandspb.OrderAmendment{
		OrderId:         o.OrderId,
		MarketId:        o.MarketId,
		SizeDelta:       o.SizeDelta,
		TimeInForce:     o.TimeInForce,
		PeggedReference: o.PeggedReference,
	}
	if o.Price != nil {
		r.Price = &proto.Price{
			Value: o.Price.Uint64(),
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
	if o.TimeInForce == Order_TIME_IN_FORCE_GTT && o.ExpiresAt == nil {
		return OrderError_ORDER_ERROR_CANNOT_AMEND_TO_GTT_WITHOUT_EXPIRYAT
	}

	if o.TimeInForce == Order_TIME_IN_FORCE_GTC && o.ExpiresAt != nil {
		// this is cool, but we need to ensure and expiry is not set
		return OrderError_ORDER_ERROR_CANNOT_HAVE_GTC_AND_EXPIRYAT
	}

	if o.TimeInForce == Order_TIME_IN_FORCE_FOK || o.TimeInForce == Order_TIME_IN_FORCE_IOC {
		// IOC and FOK are not acceptable for amend order
		return OrderError_ORDER_ERROR_CANNOT_AMEND_TO_FOK_OR_IOC
	}

	return nil
}

func (o OrderAmendment) String() string {
	return o.IntoProto().String()
}

func (o OrderAmendment) GetOrderId() string {
	return o.OrderId
}

func (o OrderAmendment) GetMarketId() string {
	return o.MarketId
}
