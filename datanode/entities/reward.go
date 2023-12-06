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
	"strconv"
	"time"

	"code.vegaprotocol.io/vega/libs/ptr"
	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"
	"code.vegaprotocol.io/vega/protos/vega"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"

	"github.com/shopspring/decimal"
)

type Reward struct {
	PartyID            PartyID
	AssetID            AssetID
	MarketID           MarketID
	EpochID            int64
	Amount             decimal.Decimal
	QuantumAmount      decimal.Decimal
	PercentOfTotal     float64
	RewardType         string
	Timestamp          time.Time
	TxHash             TxHash
	VegaTime           time.Time
	SeqNum             uint64
	LockedUntilEpochID int64
	GameID             *GameID
	TeamID             *TeamID
}

type RewardTotals struct {
	GameID       GameID
	PartyID      PartyID
	AssetID      AssetID
	MarketID     MarketID
	EpochID      int64
	TeamID       TeamID
	TotalRewards decimal.Decimal
}

func (r Reward) String() string {
	return fmt.Sprintf("{Epoch: %v, Party: %s, Asset: %s, Amount: %v}",
		r.EpochID, r.PartyID, r.AssetID, r.Amount)
}

func (r Reward) ToProto() *vega.Reward {
	var gameID, teamID *string
	if r.GameID != nil && *r.GameID != "" {
		gameID = ptr.From(r.GameID.String())
	}

	if r.TeamID != nil && *r.TeamID != "" {
		teamID = ptr.From(r.TeamID.String())
	}

	protoReward := vega.Reward{
		PartyId:           r.PartyID.String(),
		AssetId:           r.AssetID.String(),
		Epoch:             uint64(r.EpochID),
		Amount:            r.Amount.String(),
		QuantumAmount:     r.QuantumAmount.String(),
		PercentageOfTotal: fmt.Sprintf("%v", r.PercentOfTotal),
		ReceivedAt:        r.Timestamp.UnixNano(),
		MarketId:          r.MarketID.String(),
		RewardType:        r.RewardType,
		LockedUntilEpoch:  uint64(r.LockedUntilEpochID),
		GameId:            gameID,
		TeamId:            teamID,
	}
	return &protoReward
}

func (r Reward) Cursor() *Cursor {
	cursor := RewardCursor{
		PartyID: r.PartyID.String(),
		AssetID: r.AssetID.String(),
		EpochID: r.EpochID,
	}
	return NewCursor(cursor.String())
}

func (r Reward) ToProtoEdge(_ ...any) (*v2.RewardEdge, error) {
	return &v2.RewardEdge{
		Node:   r.ToProto(),
		Cursor: r.Cursor().Encode(),
	}, nil
}

func RewardFromProto(pr eventspb.RewardPayoutEvent, txHash TxHash, vegaTime time.Time, seqNum uint64) (Reward, error) {
	epochID, err := strconv.ParseInt(pr.EpochSeq, 10, 64)
	if err != nil {
		return Reward{}, fmt.Errorf("could not parse epoch %q: %w", pr.EpochSeq, err)
	}

	percentOfTotal, err := strconv.ParseFloat(pr.PercentOfTotalReward, 64)
	if err != nil {
		return Reward{}, fmt.Errorf("could not parse percent of total reward %q: %w", pr.PercentOfTotalReward, err)
	}

	amount, err := decimal.NewFromString(pr.Amount)
	if err != nil {
		return Reward{}, fmt.Errorf("could not parse amount of reward %q: %w", pr.Amount, err)
	}

	quantumAmount, err := decimal.NewFromString(pr.QuantumAmount)
	if err != nil {
		return Reward{}, fmt.Errorf("could not parse the amount of reward %q: %w", pr.QuantumAmount, err)
	}

	marketID := pr.Market
	if marketID == "!" {
		marketID = ""
	}

	lockedUntilEpochID := epochID
	if len(pr.LockedUntilEpoch) > 0 {
		lockedUntilEpochID, err = strconv.ParseInt(pr.LockedUntilEpoch, 10, 64)
		if err != nil {
			return Reward{}, fmt.Errorf("parsing locked until epoch %q: %w", pr.LockedUntilEpoch, err)
		}
	}

	var gameID *GameID
	if pr.GameId != nil {
		gameID = ptr.From(GameID(*pr.GameId))
	}

	reward := Reward{
		PartyID:            PartyID(pr.Party),
		AssetID:            AssetID(pr.Asset),
		EpochID:            epochID,
		Amount:             amount,
		QuantumAmount:      quantumAmount,
		PercentOfTotal:     percentOfTotal,
		Timestamp:          NanosToPostgresTimestamp(pr.Timestamp),
		MarketID:           MarketID(marketID),
		RewardType:         pr.RewardType,
		TxHash:             txHash,
		VegaTime:           vegaTime,
		SeqNum:             seqNum,
		LockedUntilEpochID: lockedUntilEpochID,
		GameID:             gameID,
		// We are not expecting TeamID to be set in the proto from core, but the API will populate it
		// if the reward is for a team game.
	}

	return reward, nil
}

type RewardCursor struct {
	PartyID string `json:"party_id"`
	AssetID string `json:"asset_id"`
	EpochID int64  `json:"epoch_id"`
}

func (rc RewardCursor) String() string {
	bs, err := json.Marshal(rc)
	if err != nil {
		// This should never happen.
		panic(fmt.Errorf("marshalling reward cursor: %w", err))
	}
	return string(bs)
}

func (rc *RewardCursor) Parse(cursorString string) error {
	if cursorString == "" {
		return nil
	}
	return json.Unmarshal([]byte(cursorString), rc)
}
