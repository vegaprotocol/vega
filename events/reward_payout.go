package events

import (
	"context"
	"fmt"

	eventspb "code.vegaprotocol.io/protos/vega/events/v1"

	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"
)

type RewardPayout struct {
	*Base
	party                   string
	epochSeq                string
	asset                   *types.Asset
	percentageOfTotalReward string
	amount                  *num.Uint
	timestamp               int64
}

func NewRewardPayout(ctx context.Context, timestamp int64, party, epochSeq string, asset *types.Asset, amount *num.Uint, percentageOfTotalReward float64) *RewardPayout {
	return &RewardPayout{
		Base:                    newBase(ctx, RewardPayoutEvent),
		party:                   party,
		epochSeq:                epochSeq,
		asset:                   asset,
		percentageOfTotalReward: fmt.Sprintf("%f", percentageOfTotalReward),
		amount:                  amount,
		timestamp:               timestamp,
	}
}

func (rp RewardPayout) Proto() eventspb.RewardPayoutEvent {
	return eventspb.RewardPayoutEvent{
		Party:                rp.party,
		EpochSeq:             rp.epochSeq,
		Asset:                rp.asset.IntoProto(),
		Amount:               rp.amount.String(),
		PercentOfTotalReward: rp.percentageOfTotalReward,
		Timestamp:            rp.timestamp,
	}
}

func (rp RewardPayout) StreamMessage() *eventspb.BusEvent {
	p := rp.Proto()
	return &eventspb.BusEvent{
		Id:    rp.eventID(),
		Block: rp.TraceID(),
		Type:  rp.et.ToProto(),
		Event: &eventspb.BusEvent_RewardPayout{
			RewardPayout: &p,
		},
	}
}

func RewardPayoutEventFromStream(ctx context.Context, be *eventspb.BusEvent) *RewardPayout {
	rp := be.GetRewardPayout()
	if rp == nil {
		return nil
	}

	amount, ok := num.UintFromString(rp.Amount, 10)
	if !ok {
		amount = num.Zero()
	}

	return &RewardPayout{
		Base:                    newBaseFromStream(ctx, RewardPayoutEvent, be),
		party:                   rp.Party,
		epochSeq:                rp.EpochSeq,
		asset:                   types.AssetFromProto(rp.Asset),
		percentageOfTotalReward: rp.PercentOfTotalReward,
		amount:                  amount,
	}
}
