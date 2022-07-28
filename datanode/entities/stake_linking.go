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
	"math"
	"time"

	v2 "code.vegaprotocol.io/protos/data-node/api/v2"
	eventspb "code.vegaprotocol.io/protos/vega/events/v1"
	"github.com/shopspring/decimal"
)

type StakeLinkingID struct {
	ID
}

func NewStakeLinkingID(id string) StakeLinkingID {
	return StakeLinkingID{
		ID: ID(id),
	}
}

type StakeLinking struct {
	ID                 StakeLinkingID
	StakeLinkingType   StakeLinkingType
	EthereumTimestamp  time.Time
	PartyID            PartyID
	Amount             decimal.Decimal
	StakeLinkingStatus StakeLinkingStatus
	FinalizedAt        time.Time
	TxHash             string
	LogIndex           int64
	EthereumAddress    string
	VegaTime           time.Time
}

func StakeLinkingFromProto(stake *eventspb.StakeLinking, vegaTime time.Time) (*StakeLinking, error) {
	id := NewStakeLinkingID(stake.Id)
	partyID := NewPartyID(stake.Party)
	amount, err := decimal.NewFromString(stake.Amount)
	if err != nil {
		return nil, fmt.Errorf("received invalid staking amount: %s - %w", stake.Amount, err)
	}
	if stake.BlockHeight > math.MaxInt64 {
		return nil, fmt.Errorf("block height is too high: %d", stake.BlockHeight)
	}
	if stake.LogIndex > math.MaxInt64 {
		return nil, fmt.Errorf("log index is too hight: %d", stake.LogIndex)
	}

	logIndex := int64(stake.LogIndex)

	return &StakeLinking{
		ID:                 id,
		StakeLinkingType:   StakeLinkingType(stake.Type),
		EthereumTimestamp:  time.Unix(stake.Ts, 0),
		PartyID:            partyID,
		Amount:             amount,
		StakeLinkingStatus: StakeLinkingStatus(stake.Status),
		FinalizedAt:        time.Unix(0, stake.FinalizedAt),
		TxHash:             stake.TxHash,
		LogIndex:           logIndex,
		EthereumAddress:    stake.EthereumAddress,
		VegaTime:           vegaTime,
	}, nil
}

func (s *StakeLinking) ToProto() *eventspb.StakeLinking {
	return &eventspb.StakeLinking{
		Id:              s.ID.String(),
		Type:            eventspb.StakeLinking_Type(s.StakeLinkingType),
		Ts:              s.EthereumTimestamp.Unix(),
		Party:           s.PartyID.String(),
		Amount:          s.Amount.String(),
		Status:          eventspb.StakeLinking_Status(s.StakeLinkingStatus),
		FinalizedAt:     s.FinalizedAt.UnixNano(),
		TxHash:          s.TxHash,
		LogIndex:        uint64(s.LogIndex),
		EthereumAddress: s.EthereumAddress,
	}
}

func (s StakeLinking) Cursor() *Cursor {
	cursor := StakeLinkingCursor{
		VegaTime: s.VegaTime,
		ID:       s.ID.String(),
	}
	return NewCursor(cursor.String())
}

func (s StakeLinking) ToProtoEdge(_ ...any) (*v2.StakeLinkingEdge, error) {
	return &v2.StakeLinkingEdge{
		Node:   s.ToProto(),
		Cursor: s.Cursor().Encode(),
	}, nil
}

type StakeLinkingCursor struct {
	VegaTime time.Time `json:"vegaTime"`
	ID       string    `json:"id"`
}

func (s StakeLinkingCursor) String() string {
	bs, err := json.Marshal(s)
	if err != nil {
		panic(fmt.Errorf("could not serialize StakeLinkingCursor: %w", err))
	}
	return string(bs)
}

func (s *StakeLinkingCursor) Parse(cursorString string) error {
	if cursorString == "" {
		return nil
	}

	return json.Unmarshal([]byte(cursorString), s)
}
