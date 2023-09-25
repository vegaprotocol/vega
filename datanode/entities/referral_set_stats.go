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

	ReferralFeeStats struct {
		MarketID                 MarketID
		AssetID                  AssetID
		EpochSeq                 uint64
		TotalRewardsPaid         []*eventspb.PartyAmount
		ReferrerRewardsGenerated []*eventspb.ReferrerRewardsGenerated
		RefereesDiscountApplied  []*eventspb.PartyAmount
		VolumeDiscountApplied    []*eventspb.PartyAmount
		VegaTime                 time.Time
	}
)

func ReferralSetStatsFromProto(proto *eventspb.ReferralSetStatsUpdated, vegaTime time.Time) (*ReferralSetStats, error) {
	takerVolume, err := num.DecimalFromString(proto.GetReferralSetRunningNotionalTakerVolume())
	if err != nil {
		return nil, fmt.Errorf("Invalid Running Notional Taker Volume: %v", err)
	}

	return &ReferralSetStats{
		SetID:                                 ReferralSetID(proto.SetId),
		AtEpoch:                               proto.AtEpoch,
		ReferralSetRunningNotionalTakerVolume: takerVolume,
		RefereesStats:                         proto.RefereesStats,
		VegaTime:                              vegaTime,
	}, nil
}

func (rss *ReferralSetStats) ToProto() *v2.ReferralSetStats {
	if rss == nil {
		return nil
	}

	return &v2.ReferralSetStats{
		SetId:                                 rss.SetID.String(),
		AtEpoch:                               rss.AtEpoch,
		ReferralSetRunningNotionalTakerVolume: rss.ReferralSetRunningNotionalTakerVolume.String(),
		RefereesStats:                         rss.RefereesStats,
	}
}

func (ref *ReferralSetRefereeStats) ToProto() *v2.ReferralSetStats {
	stats := eventspb.RefereeStats{
		PartyId:                  ref.PartyID,
		DiscountFactor:           ref.DiscountFactor,
		RewardFactor:             ref.RewardFactor,
		EpochNotionalTakerVolume: ref.ReferralSetRunningNotionalTakerVolume.String(),
	}

	return &v2.ReferralSetStats{
		SetId:                                 ref.SetID.String(),
		AtEpoch:                               ref.AtEpoch,
		ReferralSetRunningNotionalTakerVolume: ref.ReferralSetRunningNotionalTakerVolume.String(),
		RefereesStats:                         []*eventspb.RefereeStats{&stats},
	}
}

func ReferralFeeStatsFromProto(proto *eventspb.FeeStats, vegaTime time.Time) *ReferralFeeStats {
	return &ReferralFeeStats{
		MarketID:                 MarketID(proto.Market),
		AssetID:                  AssetID(proto.Asset),
		EpochSeq:                 proto.EpochSeq,
		TotalRewardsPaid:         proto.TotalRewardsPaid,
		ReferrerRewardsGenerated: proto.ReferrerRewardsGenerated,
		RefereesDiscountApplied:  proto.RefereesDiscountApplied,
		VolumeDiscountApplied:    proto.VolumeDiscountApplied,
		VegaTime:                 vegaTime,
	}
}

func (stats *ReferralFeeStats) ToProto() *eventspb.FeeStats {
	return &eventspb.FeeStats{
		Market:                   stats.MarketID.String(),
		Asset:                    stats.AssetID.String(),
		EpochSeq:                 stats.EpochSeq,
		TotalRewardsPaid:         stats.TotalRewardsPaid,
		ReferrerRewardsGenerated: stats.ReferrerRewardsGenerated,
		RefereesDiscountApplied:  stats.RefereesDiscountApplied,
		VolumeDiscountApplied:    stats.VolumeDiscountApplied,
	}
}
