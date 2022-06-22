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
	"fmt"
	"strconv"
	"time"

	"code.vegaprotocol.io/protos/vega"
	eventspb "code.vegaprotocol.io/protos/vega/events/v1"
	"github.com/shopspring/decimal"
)

type Delegation struct {
	PartyID  PartyID         `json:"party_id"`
	NodeID   NodeID          `json:"node_id"`
	EpochID  int64           `json:"epoch_id"`
	Amount   decimal.Decimal `json:"amount"`
	VegaTime time.Time
}

func (d Delegation) String() string {
	return fmt.Sprintf("{Epoch: %v, Party: %s, Node: %s, Amount: %v}",
		d.EpochID, d.PartyID, d.NodeID, d.Amount)
}

func (d *Delegation) ToProto() *vega.Delegation {
	protoDelegation := vega.Delegation{
		Party:    d.PartyID.String(),
		NodeId:   d.NodeID.String(),
		EpochSeq: fmt.Sprintf("%v", d.EpochID),
		Amount:   d.Amount.String(),
	}
	return &protoDelegation
}

func DelegationFromProto(pd *vega.Delegation) (Delegation, error) {
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
		PartyID: NewPartyID(pd.Party),
		NodeID:  NewNodeID(pd.NodeId),
		EpochID: epochID,
		Amount:  amount,
	}

	return delegation, nil
}

func DelegationFromEventProto(pd *eventspb.DelegationBalanceEvent) (Delegation, error) {
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
		PartyID: NewPartyID(pd.Party),
		NodeID:  NewNodeID(pd.NodeId),
		EpochID: epochID,
		Amount:  amount,
	}

	return delegation, nil
}
