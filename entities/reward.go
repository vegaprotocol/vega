// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package entities

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	v2 "code.vegaprotocol.io/protos/data-node/api/v2"
	"code.vegaprotocol.io/protos/vega"
	eventspb "code.vegaprotocol.io/protos/vega/events/v1"
	"github.com/shopspring/decimal"
)

type Reward struct {
	PartyID        PartyID
	AssetID        AssetID
	MarketID       MarketID
	EpochID        int64
	Amount         decimal.Decimal
	PercentOfTotal float64
	RewardType     string
	Timestamp      time.Time
	VegaTime       time.Time
}

func (r Reward) String() string {
	return fmt.Sprintf("{Epoch: %v, Party: %s, Asset: %s, Amount: %v}",
		r.EpochID, r.PartyID, r.AssetID, r.Amount)
}

func (r *Reward) ToProto() *vega.Reward {
	protoReward := vega.Reward{
		PartyId:           r.PartyID.String(),
		AssetId:           r.AssetID.String(),
		Epoch:             uint64(r.EpochID),
		Amount:            r.Amount.String(),
		PercentageOfTotal: fmt.Sprintf("%v", r.PercentOfTotal),
		ReceivedAt:        r.Timestamp.UnixNano(),
		MarketId:          r.MarketID.String(),
		RewardType:        r.RewardType,
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

func (r Reward) ToProtoEdge(_ ...any) *v2.RewardEdge {
	return &v2.RewardEdge{
		Node:   r.ToProto(),
		Cursor: r.Cursor().Encode(),
	}
}

func RewardFromProto(pr eventspb.RewardPayoutEvent, vegaTime time.Time) (Reward, error) {
	epochID, err := strconv.ParseInt(pr.EpochSeq, 10, 64)
	if err != nil {
		return Reward{}, fmt.Errorf("parsing epoch '%v': %w", pr.EpochSeq, err)
	}

	percentOfTotal, err := strconv.ParseFloat(pr.PercentOfTotalReward, 64)
	if err != nil {
		return Reward{}, fmt.Errorf("parsing percent of total reward '%v': %w",
			pr.PercentOfTotalReward, err)
	}

	amount, err := decimal.NewFromString(pr.Amount)
	if err != nil {
		return Reward{}, fmt.Errorf("parsing amount of reward: '%v': %w",
			pr.Amount, err)
	}

	reward := Reward{
		PartyID:        NewPartyID(pr.Party),
		AssetID:        NewAssetID(pr.Asset),
		EpochID:        epochID,
		Amount:         amount,
		PercentOfTotal: percentOfTotal,
		Timestamp:      NanosToPostgresTimestamp(pr.Timestamp),
		MarketID:       NewMarketID(pr.Market),
		RewardType:     pr.RewardType,
		VegaTime:       vegaTime,
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
