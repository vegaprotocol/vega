package entities

import (
	"time"

	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"
)

type FeesStats struct {
	MarketID                 MarketID
	AssetID                  AssetID
	EpochSeq                 uint64
	TotalRewardsPaid         []*eventspb.PartyAmount
	ReferrerRewardsGenerated []*eventspb.ReferrerRewardsGenerated
	RefereesDiscountApplied  []*eventspb.PartyAmount
	VolumeDiscountApplied    []*eventspb.PartyAmount
	TotalMakerFeesReceived   []*eventspb.PartyAmount
	MakerFeesGenerated       []*eventspb.MakerFeesGenerated
	VegaTime                 time.Time
}

func FeesStatsFromProto(proto *eventspb.FeesStats, vegaTime time.Time) *FeesStats {
	return &FeesStats{
		MarketID:                 MarketID(proto.Market),
		AssetID:                  AssetID(proto.Asset),
		EpochSeq:                 proto.EpochSeq,
		TotalRewardsPaid:         proto.TotalRewardsPaid,
		ReferrerRewardsGenerated: proto.ReferrerRewardsGenerated,
		RefereesDiscountApplied:  proto.RefereesDiscountApplied,
		VolumeDiscountApplied:    proto.VolumeDiscountApplied,
		TotalMakerFeesReceived:   proto.TotalMakerFeesReceived,
		MakerFeesGenerated:       proto.MakerFeesGenerated,
		VegaTime:                 vegaTime,
	}
}

func (stats *FeesStats) ToProto() *eventspb.FeesStats {
	return &eventspb.FeesStats{
		Market:                   stats.MarketID.String(),
		Asset:                    stats.AssetID.String(),
		EpochSeq:                 stats.EpochSeq,
		TotalRewardsPaid:         stats.TotalRewardsPaid,
		ReferrerRewardsGenerated: stats.ReferrerRewardsGenerated,
		RefereesDiscountApplied:  stats.RefereesDiscountApplied,
		VolumeDiscountApplied:    stats.VolumeDiscountApplied,
		TotalMakerFeesReceived:   stats.TotalMakerFeesReceived,
		MakerFeesGenerated:       stats.MakerFeesGenerated,
	}
}