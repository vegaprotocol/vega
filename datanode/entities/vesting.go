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

package entities

import (
	"fmt"
	"time"

	"code.vegaprotocol.io/vega/libs/num"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"
)

type (
	VestingStatsUpdated struct {
		AtEpoch           uint64
		PartyVestingStats []*PartyVestingStats
		VegaTime          time.Time
	}

	PartyVestingStats struct {
		RewardBonusMultiplier       num.Decimal
		QuantumBalance              num.Decimal
		SummedRewardBonusMultiplier num.Decimal
		SummedQuantumBalance        num.Decimal
		PartyID                     PartyID
		VegaTime                    time.Time
		AtEpoch                     uint64
	}
)

func NewVestingStatsFromProto(vestingStatsProto *eventspb.VestingStatsUpdated, vegaTime time.Time) (*VestingStatsUpdated, error) {
	partyStats := make([]*PartyVestingStats, 0, len(vestingStatsProto.Stats))
	for _, stat := range vestingStatsProto.Stats {
		multiplier, err := num.DecimalFromString(stat.RewardBonusMultiplier)
		if err != nil {
			return nil, fmt.Errorf("could not convert reward bonus multiplier to decimal: %w", err)
		}
		quantumBalance, err := num.DecimalFromString(stat.QuantumBalance)
		if err != nil {
			return nil, fmt.Errorf("could not convert quantum balance to decimal: %w", err)
		}
		summedMultiplier, err := num.DecimalFromString(stat.SummedRewardBonusMultiplier)
		if err != nil {
			return nil, fmt.Errorf("could not convert summed reward bonus multiplier to decimal: %w", err)
		}
		summedQuantumBalance, err := num.DecimalFromString(stat.SummedQuantumBalance)
		if err != nil {
			return nil, fmt.Errorf("could not convert summed quantum balance to decimal: %w", err)
		}

		partyStats = append(partyStats, &PartyVestingStats{
			RewardBonusMultiplier:       multiplier,
			QuantumBalance:              quantumBalance,
			SummedRewardBonusMultiplier: summedMultiplier,
			SummedQuantumBalance:        summedQuantumBalance,
			PartyID:                     PartyID(stat.PartyId),
			VegaTime:                    vegaTime,
			AtEpoch:                     vestingStatsProto.AtEpoch,
		})
	}

	return &VestingStatsUpdated{
		AtEpoch:           vestingStatsProto.AtEpoch,
		PartyVestingStats: partyStats,
		VegaTime:          vegaTime,
	}, nil
}
