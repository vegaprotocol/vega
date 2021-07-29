package events

import (
	"context"

	proto "code.vegaprotocol.io/protos/vega"
	eventspb "code.vegaprotocol.io/protos/vega/events/v1"
	"code.vegaprotocol.io/data-node/types"
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
	return &eventspb.BusEvent{
		Id:    r.eventID(),
		Block: r.TraceID(),
		Type:  r.et.ToProto(),
		Event: &eventspb.BusEvent_RiskFactor{
			RiskFactor: &r.r,
		},
	}
}
