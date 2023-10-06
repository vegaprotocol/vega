package entities

import (
	"encoding/json"
	"time"

	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"
)

type (
	ReferralSetStats struct {
		SetID                                 ReferralSetID
		AtEpoch                               uint64
		ReferralSetRunningNotionalTakerVolume string
		RefereesStats                         []*eventspb.RefereeStats
		VegaTime                              time.Time
		RewardFactor                          string
	}

	FlattenReferralSetStats struct {
		AtEpoch                               uint64
		ReferralSetRunningNotionalTakerVolume string
		VegaTime                              time.Time
		PartyID                               string
		DiscountFactor                        string
		EpochNotionalTakerVolume              string
		RewardFactor                          string
	}

	ReferralSetStatsCursor struct {
		VegaTime time.Time
		AtEpoch  uint64
		PartyID  string
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

func (s FlattenReferralSetStats) Cursor() *Cursor {
	c := ReferralSetStatsCursor{
		VegaTime: s.VegaTime,
		AtEpoch:  s.AtEpoch,
		PartyID:  s.PartyID,
	}
	return NewCursor(c.ToString())
}

func (s FlattenReferralSetStats) ToProtoEdge(_ ...any) (*v2.ReferralSetStatsEdge, error) {
	return &v2.ReferralSetStatsEdge{
		Node:   s.ToProto(),
		Cursor: s.Cursor().Encode(),
	}, nil
}

func (s FlattenReferralSetStats) ToProto() *v2.ReferralSetStats {
	return &v2.ReferralSetStats{
		AtEpoch:                               s.AtEpoch,
		ReferralSetRunningNotionalTakerVolume: s.ReferralSetRunningNotionalTakerVolume,
		PartyId:                               s.PartyID,
		DiscountFactor:                        s.DiscountFactor,
		RewardFactor:                          s.RewardFactor,
		EpochNotionalTakerVolume:              s.EpochNotionalTakerVolume,
	}
}

func (c ReferralSetStatsCursor) ToString() string {
	bs, _ := json.Marshal(c)
	return string(bs)
}

func (c *ReferralSetStatsCursor) Parse(cursorString string) error {
	if cursorString == "" {
		return nil
	}
	return json.Unmarshal([]byte(cursorString), c)
}

func ReferralSetStatsFromProto(proto *eventspb.ReferralSetStatsUpdated, vegaTime time.Time) (*ReferralSetStats, error) {
	return &ReferralSetStats{
		SetID:                                 ReferralSetID(proto.SetId),
		AtEpoch:                               proto.AtEpoch,
		ReferralSetRunningNotionalTakerVolume: proto.ReferralSetRunningNotionalTakerVolume,
		RefereesStats:                         proto.RefereesStats,
		VegaTime:                              vegaTime,
		RewardFactor:                          proto.RewardFactor,
	}, nil
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
