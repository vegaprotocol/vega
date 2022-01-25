package events

import (
	"context"
	"fmt"

	eventspb "code.vegaprotocol.io/protos/vega/events/v1"

	"code.vegaprotocol.io/vega/types/num"
)

type RewardPayout struct {
	*Base
	Party                   string
	EpochSeq                string
	Asset                   string
	PercentageOfTotalReward string
	Amount                  *num.Uint
	Timestamp               int64
}

func NewRewardPayout(ctx context.Context, timestamp int64, party, epochSeq string, asset string, amount *num.Uint, percentageOfTotalReward float64) *RewardPayout {
	return &RewardPayout{
		Base:                    newBase(ctx, RewardPayoutEvent),
		Party:                   party,
		EpochSeq:                epochSeq,
		Asset:                   asset,
		PercentageOfTotalReward: fmt.Sprintf("%f", percentageOfTotalReward),
		Amount:                  amount,
		Timestamp:               timestamp,
	}
}

func (rp RewardPayout) RewardPayoutEvent() eventspb.RewardPayoutEvent {
	return rp.Proto()
}

func (rp RewardPayout) Proto() eventspb.RewardPayoutEvent {
	return eventspb.RewardPayoutEvent{
		Party:                rp.Party,
		EpochSeq:             rp.EpochSeq,
		Asset:                rp.Asset,
		Amount:               rp.Amount.String(),
		PercentOfTotalReward: rp.PercentageOfTotalReward,
		Timestamp:            rp.Timestamp,
	}
}

func (rp RewardPayout) StreamMessage() *eventspb.BusEvent {
	p := rp.Proto()
	busEvent := newBusEventFromBase(rp.Base)
	busEvent.Event = &eventspb.BusEvent_RewardPayout{
		RewardPayout: &p,
	}

	return busEvent
}

func RewardPayoutEventFromStream(ctx context.Context, be *eventspb.BusEvent) *RewardPayout {
	rp := be.GetRewardPayout()
	if rp == nil {
		return nil
	}

	amount, _ := num.UintFromString(rp.Amount, 10)
	return &RewardPayout{
		Base:                    newBaseFromBusEvent(ctx, RewardPayoutEvent, be),
		Party:                   rp.Party,
		EpochSeq:                rp.EpochSeq,
		Asset:                   rp.Asset,
		PercentageOfTotalReward: rp.PercentOfTotalReward,
		Amount:                  amount,
		Timestamp:               rp.Timestamp,
	}
}
