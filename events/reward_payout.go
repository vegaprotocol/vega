package events

import (
	"context"
	"fmt"

	eventspb "code.vegaprotocol.io/protos/vega/events/v1"

	"code.vegaprotocol.io/vega/types/num"
)

type RewardPayout struct {
	*Base
	from                 string
	to                   string
	party                string
	epochSeq             string
	asset                string
	percentOfTotalReward string
	amount               *num.Uint
}

func NewRewardPayout(ctx context.Context, from, to, party, epochSeq, asset string, amount *num.Uint, percentOfTotalReward float64) *RewardPayout {
	return &RewardPayout{
		Base:                 newBase(ctx, DelegationBalanceEvent),
		from:                 from,
		to:                   to,
		epochSeq:             epochSeq,
		asset:                asset,
		amount:               amount,
		party:                party,
		percentOfTotalReward: fmt.Sprintf("%f", percentOfTotalReward),
	}
}

func (rp RewardPayout) Proto() eventspb.RewardPayoutEvent {
	return eventspb.RewardPayoutEvent{
		FromAccount:          rp.from,
		ToAccount:            rp.to,
		Party:                rp.party,
		EpochSeq:             rp.epochSeq,
		Asset:                rp.asset,
		Amount:               rp.amount.Uint64(),
		PercentOfTotalReward: rp.percentOfTotalReward,
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
