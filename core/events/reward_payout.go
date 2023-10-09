// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

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
