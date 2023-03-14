// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.DATANODE file and at https://www.mariadb.com/bsl11.
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
