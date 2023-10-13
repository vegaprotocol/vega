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

	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"
)

type PaidLiquidityFeesStatsCursor struct {
	Epoch    uint64
	MarketID string
	AssetID  string
}

func (c PaidLiquidityFeesStatsCursor) ToString() string {
	bs, _ := json.Marshal(c)
	return string(bs)
}

func (c *PaidLiquidityFeesStatsCursor) Parse(cursorString string) error {
	if cursorString == "" {
		return nil
	}
	return json.Unmarshal([]byte(cursorString), c)
}

type PaidLiquidityFeesStats struct {
	MarketID      MarketID
	AssetID       AssetID
	EpochSeq      uint64
	TotalFeesPaid string
	FeesPerParty  []*eventspb.PartyAmount
}

func (s PaidLiquidityFeesStats) Cursor() *Cursor {
	c := PaidLiquidityFeesStatsCursor{
		Epoch:    s.EpochSeq,
		MarketID: string(s.MarketID),
		AssetID:  s.AssetID.String(),
	}
	return NewCursor(c.ToString())
}

func (s PaidLiquidityFeesStats) ToProtoEdge(_ ...any) (*v2.PaidLiquidityFeesEdge, error) {
	return &v2.PaidLiquidityFeesEdge{
		Node:   s.ToProto(),
		Cursor: s.Cursor().Encode(),
	}, nil
}

func (s PaidLiquidityFeesStats) ToProto() *eventspb.PaidLiquidityFeesStats {
	return &eventspb.PaidLiquidityFeesStats{
		Market:           s.MarketID.String(),
		Asset:            s.AssetID.String(),
		EpochSeq:         s.EpochSeq,
		TotalFeesPaid:    s.TotalFeesPaid,
		FeesPaidPerParty: s.FeesPerParty,
	}
}

func PaidLiquidityFeesStatsFromProto(proto *eventspb.PaidLiquidityFeesStats) *PaidLiquidityFeesStats {
	return &PaidLiquidityFeesStats{
		MarketID:      MarketID(proto.Market),
		AssetID:       AssetID(proto.Asset),
		EpochSeq:      proto.EpochSeq,
		TotalFeesPaid: proto.TotalFeesPaid,
		FeesPerParty:  proto.FeesPaidPerParty,
	}
}
