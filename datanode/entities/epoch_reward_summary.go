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

	"code.vegaprotocol.io/vega/libs/num"
	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"
	"code.vegaprotocol.io/vega/protos/vega"
)

type EpochRewardSummary struct {
	AssetID    AssetID
	MarketID   MarketID
	RewardType string
	EpochID    uint64
	Amount     num.Decimal
}

func (r *EpochRewardSummary) ToProto() *vega.EpochRewardSummary {
	protoRewardSummary := vega.EpochRewardSummary{
		AssetId:    r.AssetID.String(),
		MarketId:   r.MarketID.String(),
		RewardType: r.RewardType,
		Epoch:      r.EpochID,
		Amount:     r.Amount.String(),
	}
	return &protoRewardSummary
}

func (r EpochRewardSummary) Cursor() *Cursor {
	cursor := EpochRewardSummaryCursor{
		AssetID:    r.AssetID.String(),
		MarketID:   r.MarketID.String(),
		RewardType: r.RewardType,
		EpochID:    r.EpochID,
		Amount:     r.Amount.String(),
	}
	return NewCursor(cursor.String())
}

func (r EpochRewardSummary) ToProtoEdge(_ ...any) (*v2.EpochRewardSummaryEdge, error) {
	return &v2.EpochRewardSummaryEdge{
		Node:   r.ToProto(),
		Cursor: r.Cursor().Encode(),
	}, nil
}

type EpochRewardSummaryCursor struct {
	EpochID    uint64 `json:"epoch_id"`
	AssetID    string `json:"asset_id"`
	MarketID   string `json:"market_id"`
	RewardType string `json:"reward_type"`
	Amount     string `json:"amount"`
}

func (rc EpochRewardSummaryCursor) String() string {
	bs, err := json.Marshal(rc)
	if err != nil {
		// This should never happen.
		panic(fmt.Errorf("marshalling epoch reward summary cursor: %w", err))
	}
	return string(bs)
}

func (rc *EpochRewardSummaryCursor) Parse(cursorString string) error {
	if cursorString == "" {
		return nil
	}
	return json.Unmarshal([]byte(cursorString), rc)
}
