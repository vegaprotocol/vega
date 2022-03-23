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
	PartyID  PartyID
	NodeID   NodeID
	EpochID  int64
	Amount   decimal.Decimal
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

func DelegationFromProto(pd *eventspb.DelegationBalanceEvent) (Delegation, error) {
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
