package events

import (
	"context"

	proto "code.vegaprotocol.io/protos/vega"
	eventspb "code.vegaprotocol.io/protos/vega/events/v1"
	"code.vegaprotocol.io/vega/types"
)

type RiskFactor struct {
	*Base
	r proto.RiskFactor
}

func NewRiskFactorEvent(ctx context.Context, r types.RiskFactor) *RiskFactor {
	return &RiskFactor{
		Base: newBase(ctx, RiskFactorEvent),
		r:    *r.IntoProto(),
	}
}

func (r RiskFactor) MarketID() string {
	return r.r.Market
}

func (r *RiskFactor) RiskFactor() proto.RiskFactor {
	return r.r
}

func (r RiskFactor) Proto() proto.RiskFactor {
	return r.r
}

func (r RiskFactor) StreamMessage() *eventspb.BusEvent {
	busEvent := newBusEventFromBase(r.Base)
	busEvent.Event = &eventspb.BusEvent_RiskFactor{
		RiskFactor: &r.r,
	}

	return busEvent
}

func RiskFactorEventFromStream(ctx context.Context, be *eventspb.BusEvent) *RiskFactor {
	return &RiskFactor{
		Base: newBaseFromBusEvent(ctx, RiskFactorEvent, be),
		r:    *be.GetRiskFactor(),
	}
}
