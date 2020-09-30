package events

import (
	"context"

	types "code.vegaprotocol.io/vega/proto"
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

func (r RiskFactor) Proto() types.RiskFactor {
	return r.r
}

func (r RiskFactor) StreamMessage() types.BusEvent {
	return types.BusEvent{
		ID:   r.eventID(),
		Type: r.et.ToProto(),
		Event: &types.BusEvent_RiskFactor{
			RiskFactor: &r.r,
		},
	}
}
