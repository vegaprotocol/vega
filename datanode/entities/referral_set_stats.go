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
	"encoding/json"
	"time"

	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"
	"code.vegaprotocol.io/vega/protos/vega"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"
)

type (
	ReferralSetStats struct {
		SetID                                 ReferralSetID
		AtEpoch                               uint64
		WasEligible                           bool
		ReferralSetRunningNotionalTakerVolume string
		ReferrerTakerVolume                   string
		RefereesStats                         []*eventspb.RefereeStats
		VegaTime                              time.Time
		RewardFactors                         *vega.RewardFactors
		RewardsMultiplier                     string
		RewardsFactorsMultiplier              *vega.RewardFactors
	}

	FlattenReferralSetStats struct {
		SetID                                 ReferralSetID
		AtEpoch                               uint64
		WasEligible                           bool
		ReferralSetRunningNotionalTakerVolume string
		ReferrerTakerVolume                   string
		VegaTime                              time.Time
		PartyID                               string
		DiscountFactors                       *vega.DiscountFactors
		EpochNotionalTakerVolume              string
		RewardFactors                         *vega.RewardFactors
		RewardsMultiplier                     string
		RewardsFactorsMultiplier              *vega.RewardFactors
	}

	ReferralSetStatsCursor struct {
		VegaTime time.Time
		AtEpoch  uint64
		SetID    string
		PartyID  string
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
		ReferrerTakerVolume:                   s.ReferrerTakerVolume,
		PartyId:                               s.PartyID,
		DiscountFactors:                       s.DiscountFactors,
		RewardFactors:                         s.RewardFactors,
		EpochNotionalTakerVolume:              s.EpochNotionalTakerVolume,
		RewardsMultiplier:                     s.RewardsMultiplier,
		RewardsFactorsMultiplier:              s.RewardsFactorsMultiplier,
		WasEligible:                           s.WasEligible,
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
		WasEligible:                           proto.WasEligible,
		ReferralSetRunningNotionalTakerVolume: proto.ReferralSetRunningNotionalTakerVolume,
		ReferrerTakerVolume:                   proto.ReferrerTakerVolume,
		RefereesStats:                         proto.RefereesStats,
		VegaTime:                              vegaTime,
		RewardFactors:                         proto.RewardFactors,
		RewardsMultiplier:                     proto.RewardsMultiplier,
		RewardsFactorsMultiplier:              proto.RewardFactorsMultiplier,
	}, nil
}
