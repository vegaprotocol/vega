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

	VestingStatsCursor struct {
		VegaTime time.Time
		AtEpoch  uint64
	}

	PartyVestingStats struct {
		RewardBonusMultiplier num.Decimal
		PartyID               string
		VegaTime              time.Time
	}
)

func NewVestingStatsFromProto(vestingStatsProto *eventspb.VestingStatsUpdated, vegaTime time.Time) (*VestingStatsUpdated, error) {
	partyStats := make([]*PartyVestingStats, 0, len(vestingStatsProto.Stats))
	for _, stat := range vestingStatsProto.Stats {
		multiplier, err := num.DecimalFromString(stat.RewardBonusMultiplier)
		if err != nil {
			return nil, fmt.Errorf("could not convert reward bonus multiplier to decimal: %w", err)
		}

		partyStats = append(partyStats, &PartyVestingStats{
			RewardBonusMultiplier: multiplier,
			PartyID:               stat.PartyId,
			VegaTime:              vegaTime,
		})
	}

	return &VestingStatsUpdated{
		AtEpoch:           vestingStatsProto.AtEpoch,
		PartyVestingStats: partyStats,
		VegaTime:          vegaTime,
	}, nil
}
