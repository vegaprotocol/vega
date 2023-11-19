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
	"time"

	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"
)

type FeesStats struct {
	MarketID                 MarketID
	AssetID                  AssetID
	EpochSeq                 uint64
	TotalRewardsReceived     []*eventspb.PartyAmount
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
		TotalRewardsReceived:     proto.TotalRewardsReceived,
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
		TotalRewardsReceived:     stats.TotalRewardsReceived,
		ReferrerRewardsGenerated: stats.ReferrerRewardsGenerated,
		RefereesDiscountApplied:  stats.RefereesDiscountApplied,
		VolumeDiscountApplied:    stats.VolumeDiscountApplied,
		TotalMakerFeesReceived:   stats.TotalMakerFeesReceived,
		MakerFeesGenerated:       stats.MakerFeesGenerated,
	}
}

type FeesStatsForParty struct {
	AssetID                 AssetID
	TotalRewardsReceived    string
	RefereesDiscountApplied string
	VolumeDiscountApplied   string
	TotalMakerFeesReceived  string
}
