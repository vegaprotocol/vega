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
	PartyID  []byte
	NodeID   []byte
	EpochID  int64
	Amount   decimal.Decimal
	VegaTime time.Time
}

func (d *Delegation) PartyHexID() string {
	return Party{ID: d.PartyID}.HexID()
}

func (d *Delegation) NodeHexID() string {
	return Node{ID: d.NodeID}.HexID()
}

func (d Delegation) String() string {
	return fmt.Sprintf("{Epoch: %v, Party: %s, Node: %s, Amount: %v}",
		d.EpochID, d.PartyHexID(), d.NodeHexID(), d.Amount)
}

func (d *Delegation) ToProto() *vega.Delegation {
	protoDelegation := vega.Delegation{
		Party:    d.PartyHexID(),
		NodeId:   d.NodeHexID(),
		EpochSeq: fmt.Sprintf("%v", d.EpochID),
		Amount:   d.Amount.String(),
	}
	return &protoDelegation
}

func DelegationFromProto(pd *eventspb.DelegationBalanceEvent) (Delegation, error) {
	partyID, err := MakePartyID(pd.Party)
	if err != nil {
		return Delegation{}, fmt.Errorf("parsing party id '%v': %w", pd.Party, err)
	}

	nodeID, err := MakeNodeID(pd.NodeId)
	if err != nil {
		return Delegation{}, fmt.Errorf("parsing node id '%v': %w", pd.NodeId, err)
	}

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
		PartyID: partyID,
		NodeID:  nodeID,
		EpochID: epochID,
		Amount:  amount,
	}

	return delegation, nil
}
