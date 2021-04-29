package events

import (
	"context"

	types "code.vegaprotocol.io/vega/proto"
	eventspb "code.vegaprotocol.io/vega/proto/events/v1"
)

type RiskFactor struct {
	*Base
	r types.RiskFactor
}

func NewRiskFactorEvent(ctx context.Context, r types.RiskFactor) *RiskFactor {
	cpy := r.DeepClone()
	return &RiskFactor{
		Base: newBase(ctx, RiskFactorEvent),
		r:    *cpy,
	}
}

func (r RiskFactor) MarketID() string {
	return r.r.Market
}

func (r *RiskFactor) RiskFactor() types.RiskFactor {
	return r.r
}

func (r RiskFactor) Proto() types.RiskFactor {
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
