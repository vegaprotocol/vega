package sqlsubscribers

import (
	"context"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/datanode/entities"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"
)

type (
	FeeStatsEvent interface {
		events.Event
		FeeStats() *eventspb.FeeStats
	}

	ReferralFeeStatsStore interface {
		AddFeeStats(ctx context.Context, feeStats *entities.ReferralFeeStats) error
	}

	ReferralFeeStats struct {
		subscriber
		store ReferralFeeStatsStore
	}
)

func NewReferralFeeStats(store ReferralFeeStatsStore) *ReferralFeeStats {
	return &ReferralFeeStats{
		store: store,
	}
}

func (r *ReferralFeeStats) Types() []events.Type {
	return []events.Type{
		events.FeeStatsEvent,
	}
}

func (r *ReferralFeeStats) Push(ctx context.Context, evt events.Event) error {
	switch e := evt.(type) {
	case FeeStatsEvent:
		return r.consumeFeeStatsEvent(ctx, e)
	default:
		return nil
	}
}

func (r *ReferralFeeStats) consumeFeeStatsEvent(ctx context.Context, e FeeStatsEvent) error {
	return r.store.AddFeeStats(ctx, entities.ReferralFeeStatsFromProto(e.FeeStats(), r.vegaTime))
}
