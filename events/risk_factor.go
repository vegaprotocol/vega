package events

import (
	"context"

	"code.vegaprotocol.io/vega/proto"
	eventspb "code.vegaprotocol.io/vega/proto/events/v1"
	"code.vegaprotocol.io/vega/types"
)

type RiskFactor struct {
	*Base
	r types.RiskFactor
}

func NewRiskFactorEvent(ctx context.Context, r types.RiskFactor) *RiskFactor {
	return &RiskFactor{
		Base: newBase(ctx, RiskFactorEvent),
		r:    r,
	}
}

func (r RiskFactor) MarketID() string {
	return r.r.Market
}

func (r *RiskFactor) RiskFactor() types.RiskFactor {
	return r.r
}

func (r RiskFactor) Proto() proto.RiskFactor {
	p := r.r.IntoProto()
	return *p
}

func (r RiskFactor) StreamMessage() *eventspb.BusEvent {
	return &eventspb.BusEvent{
		Id:    r.eventID(),
		Block: r.TraceID(),
		Type:  r.et.ToProto(),
		Event: &eventspb.BusEvent_RiskFactor{
			RiskFactor: r.r.IntoProto(),
		},
	}
}
