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

	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"
	"code.vegaprotocol.io/vega/protos/vega"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"

	"github.com/shopspring/decimal"
)

type Delegation struct {
	PartyID  PartyID         `json:"party_id"`
	NodeID   NodeID          `json:"node_id"`
	EpochID  int64           `json:"epoch_id"`
	Amount   decimal.Decimal `json:"amount"`
	TxHash   TxHash
	VegaTime time.Time
	SeqNum   uint64
}

func (d Delegation) String() string {
	return fmt.Sprintf("{Epoch: %v, Party: %s, Node: %s, Amount: %v}",
		d.EpochID, d.PartyID, d.NodeID, d.Amount)
}

func (d Delegation) ToProto() *vega.Delegation {
	protoDelegation := vega.Delegation{
		Party:    d.PartyID.String(),
		NodeId:   d.NodeID.String(),
		EpochSeq: fmt.Sprintf("%v", d.EpochID),
		Amount:   d.Amount.String(),
	}
	return &protoDelegation
}

func (d Delegation) Cursor() *Cursor {
	dc := DelegationCursor{
		VegaTime: d.VegaTime,
		PartyID:  d.PartyID,
		NodeID:   d.NodeID,
		EpochID:  d.EpochID,
	}
	return NewCursor(dc.String())
}

func (d Delegation) ToProtoEdge(_ ...any) (*v2.DelegationEdge, error) {
	return &v2.DelegationEdge{
		Node:   d.ToProto(),
		Cursor: d.Cursor().Encode(),
	}, nil
}

func DelegationFromProto(pd *vega.Delegation, txHash TxHash) (Delegation, error) {
	epochID, err := strconv.ParseInt(pd.EpochSeq, 10, 64)
	if err != nil {
		return Delegation{}, fmt.Errorf("parsing epoch '%v': %w", pd.EpochSeq, err)
	}

	amount, err := decimal.NewFromString(pd.Amount)
	if err != nil {
		return Delegation{}, fmt.Errorf("parsing amount of delegation: '%v': %w",
			pd.Amount, err)
	}

	delegation := Delegation{
		PartyID: PartyID(pd.Party),
		NodeID:  NodeID(pd.NodeId),
		EpochID: epochID,
		Amount:  amount,
		TxHash:  txHash,
	}

	return delegation, nil
}

func DelegationFromEventProto(pd *eventspb.DelegationBalanceEvent, txHash TxHash) (Delegation, error) {
	epochID, err := strconv.ParseInt(pd.EpochSeq, 10, 64)
	if err != nil {
		return Delegation{}, fmt.Errorf("parsing epoch '%v': %w", pd.EpochSeq, err)
	}

	amount, err := decimal.NewFromString(pd.Amount)
	if err != nil {
		return Delegation{}, fmt.Errorf("parsing amount of delegation: '%v': %w",
			pd.Amount, err)
	}

	delegation := Delegation{
		PartyID: PartyID(pd.Party),
		NodeID:  NodeID(pd.NodeId),
		EpochID: epochID,
		Amount:  amount,
		TxHash:  txHash,
	}

	return delegation, nil
}

type DelegationCursor struct {
	VegaTime time.Time `json:"vegaTime"`
	PartyID  PartyID   `json:"partyId"`
	NodeID   NodeID    `json:"nodeId"`
	EpochID  int64     `json:"epochId"`
}

func (c DelegationCursor) String() string {
	bs, err := json.Marshal(c)
	if err != nil {
		panic(fmt.Errorf("could not marshal delegation cursor: %w", err))
	}
	return string(bs)
}

func (c *DelegationCursor) Parse(cursorString string) error {
	if cursorString == "" {
		return nil
	}
	return json.Unmarshal([]byte(cursorString), c)
}
