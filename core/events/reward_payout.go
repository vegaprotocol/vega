// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.VEGA file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package events

import (
	"context"

	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	vegapb "code.vegaprotocol.io/vega/protos/vega"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"
)

type RewardPayout struct {
	*Base
	Party                   string
	EpochSeq                string
	Asset                   string
	Market                  string
	PercentageOfTotalReward string
	Amount                  *num.Uint
	Timestamp               int64
	RewardType              types.AccountType
}

func NewRewardPayout(ctx context.Context, timestamp int64, party, epochSeq, asset string, amount *num.Uint, percentageOfTotalReward num.Decimal, rewardType types.AccountType, market string) *RewardPayout {
	return &RewardPayout{
		Base:                    newBase(ctx, RewardPayoutEvent),
		Party:                   party,
		EpochSeq:                epochSeq,
		Asset:                   asset,
		PercentageOfTotalReward: percentageOfTotalReward.String(),
		Amount:                  amount,
		Timestamp:               timestamp,
		RewardType:              rewardType,
		Market:                  market,
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
		RewardType:           vegapb.AccountType_name[int32(rp.RewardType)],
		Market:               rp.Market,
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
		Market:                  rp.Market,
		RewardType:              types.AccountType(vegapb.AccountType_value[rp.RewardType]),
	}
}
