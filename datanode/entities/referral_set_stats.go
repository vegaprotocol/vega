package entities

import (
	"fmt"
	"time"

	"code.vegaprotocol.io/vega/libs/num"
	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"
)

type (
	ReferralSetStats struct {
		SetID                                 ReferralSetID
		AtEpoch                               uint64
		ReferralSetRunningNotionalTakerVolume num.Decimal
		RefereesStats                         []*eventspb.RefereeStats
		VegaTime                              time.Time
	}

	ReferralSetStatsCursor struct {
		VegaTime time.Time
		SetID    ReferralSetID
		AtEpoch  uint64
	}

	ReferralSetRefereeStats struct {
		SetID                                 ReferralSetID
		AtEpoch                               uint64
		ReferralSetRunningNotionalTakerVolume num.Decimal
		PartyID                               string
		DiscountFactor                        string
		RewardFactor                          string
		VegaTime                              time.Time
	}
)

func ReferralSetStatsFromProto(protos *eventspb.ReferralSetStatsUpdated, vegaTime time.Time) (*ReferralSetStats, error) {
	takerVolume, err := num.DecimalFromString(protos.GetReferralSetRunningNotionalTakerVolume())
	if err != nil {
		return nil, fmt.Errorf("Invalid Running Notional Taker Volume: %v", err)
	}

	return &ReferralSetStats{
		SetID:                                 ReferralSetID(protos.SetId),
		AtEpoch:                               protos.AtEpoch,
		ReferralSetRunningNotionalTakerVolume: takerVolume,
		RefereesStats:                         nil,
		VegaTime:                              vegaTime,
	}, nil
}

func (rss *ReferralSetStats) ToProto() *v2.ReferralSetStats {
	return &v2.ReferralSetStats{
		SetId:                                 rss.SetID.String(),
		AtEpoch:                               rss.AtEpoch,
		ReferralSetRunningNotionalTakerVolume: rss.ReferralSetRunningNotionalTakerVolume.String(),
		RefereesStats:                         rss.RefereesStats,
	}
}

func (ref *ReferralSetRefereeStats) ToProto() *v2.ReferralSetStats {
	stats := eventspb.RefereeStats{
		PartyId:        ref.PartyID,
		DiscountFactor: ref.DiscountFactor,
		RewardFactor:   ref.RewardFactor,
	}

	return &v2.ReferralSetStats{
		SetId:                                 ref.SetID.String(),
		AtEpoch:                               ref.AtEpoch,
		ReferralSetRunningNotionalTakerVolume: ref.ReferralSetRunningNotionalTakerVolume.String(),
		RefereesStats:                         []*eventspb.RefereeStats{&stats},
	}
}
