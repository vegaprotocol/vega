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
	"fmt"
	"time"

	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"
)

type (
	VolumeRebateStats struct {
		AtEpoch                  uint64
		PartiesVolumeRebateStats []*eventspb.PartyVolumeRebateStats
		VegaTime                 time.Time
	}

	FlattenVolumeRebateStats struct {
		AtEpoch             uint64
		PartyID             string
		AdditionalRebate    string
		MakerVolumeFraction string
		MakerFeesReceived   string
		VegaTime            time.Time
	}

	VolumeRebateStatsCursor struct {
		VegaTime time.Time
		AtEpoch  uint64
		PartyID  string
	}
)

func (s FlattenVolumeRebateStats) Cursor() *Cursor {
	c := VolumeRebateStatsCursor{
		VegaTime: s.VegaTime,
		AtEpoch:  s.AtEpoch,
		PartyID:  s.PartyID,
	}
	return NewCursor(c.ToString())
}

func (s FlattenVolumeRebateStats) ToProtoEdge(_ ...any) (*v2.VolumeRebateStatsEdge, error) {
	return &v2.VolumeRebateStatsEdge{
		Node:   s.ToProto(),
		Cursor: s.Cursor().Encode(),
	}, nil
}

func (s FlattenVolumeRebateStats) ToProto() *v2.VolumeRebateStats {
	return &v2.VolumeRebateStats{
		AtEpoch:               s.AtEpoch,
		PartyId:               s.PartyID,
		AdditionalMakerRebate: s.AdditionalRebate,
		MakerVolumeFraction:   s.MakerVolumeFraction,
		MakerFeesReceived:     s.MakerFeesReceived,
	}
}

func NewVolumeRebateStatsFromProto(vestingStatsProto *eventspb.VolumeRebateStatsUpdated, vegaTime time.Time) (*VolumeRebateStats, error) {
	return &VolumeRebateStats{
		AtEpoch:                  vestingStatsProto.AtEpoch,
		PartiesVolumeRebateStats: vestingStatsProto.Stats,
		VegaTime:                 vegaTime,
	}, nil
}

func (c *VolumeRebateStatsCursor) ToString() string {
	bs, err := json.Marshal(c)
	if err != nil {
		panic(fmt.Errorf("could not marshal volume rebate stats cursor: %v", err))
	}
	return string(bs)
}

func (c *VolumeRebateStatsCursor) Parse(cursorString string) error {
	if cursorString == "" {
		return nil
	}
	return json.Unmarshal([]byte(cursorString), c)
}
