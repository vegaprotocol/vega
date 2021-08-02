package types

import (
	commandspb "code.vegaprotocol.io/protos/vega/commands/v1"
	"code.vegaprotocol.io/vega/types/num"
)

type Delegate struct {
	NodeID string
	Amount uint64
}

func NewDelegateFromProto(p *commandspb.DelegateSubmission) *Delegate {
	return &Delegate{
		NodeID: p.NodeId,
		Amount: p.Amount,
	}
}

func (d Delegate) IntoProto() *commandspb.DelegateSubmission {
	return &commandspb.DelegateSubmission{
		NodeId: d.NodeID,
		Amount: d.Amount,
	}
}

func (d Delegate) String() string {
	return d.IntoProto().String()
}

type UndelegateAtEpochEnd struct {
	NodeID string
	Amount uint64
}

func NewUndelegateAtEpochEndFromProto(p *commandspb.UndelegateAtEpochEndSubmission) *UndelegateAtEpochEnd {
	return &UndelegateAtEpochEnd{
		NodeID: p.NodeId,
		Amount: p.Amount,
	}
}

func (u UndelegateAtEpochEnd) IntoProto() *commandspb.UndelegateAtEpochEndSubmission {
	return &commandspb.UndelegateAtEpochEndSubmission{
		NodeId: u.NodeID,
		Amount: u.Amount,
	}
}

func (u UndelegateAtEpochEnd) String() string {
	return u.IntoProto().String()
}

// ValidatorData is delegation data for validator
type ValidatorData struct {
	NodeID            string
	StakeByDelegators *num.Uint
	Delegators        map[string]*num.Uint
}
