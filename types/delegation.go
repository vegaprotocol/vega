package types

import (
	commandspb "code.vegaprotocol.io/protos/vega/commands/v1"
	"code.vegaprotocol.io/vega/types/num"
)

type Delegate struct {
	NodeID string
	Amount *num.Uint
}

func NewDelegateFromProto(p *commandspb.DelegateSubmission) *Delegate {
	return &Delegate{
		NodeID: p.NodeId,
		Amount: num.NewUint(p.Amount),
	}
}

func (d Delegate) IntoProto() *commandspb.DelegateSubmission {
	return &commandspb.DelegateSubmission{
		NodeId: d.NodeID,
		Amount: d.Amount.Uint64(),
	}
}

func (d Delegate) String() string {
	return d.IntoProto().String()
}

type Undelegate struct {
	NodeID string
	Amount *num.Uint
	Method string
}

func NewUndelegateFromProto(p *commandspb.UndelegateSubmission) *Undelegate {
	return &Undelegate{
		NodeID: p.NodeId,
		Amount: num.NewUint(p.Amount),
		Method: p.Method.String(),
	}
}

func (u Undelegate) IntoProto() *commandspb.UndelegateSubmission {
	return &commandspb.UndelegateSubmission{
		NodeId: u.NodeID,
		Amount: u.Amount.Uint64(),
		Method: commandspb.UndelegateSubmission_Method(commandspb.UndelegateSubmission_Method_value[u.Method]),
	}
}

func (u Undelegate) String() string {
	return u.IntoProto().String()
}

// ValidatorData is delegation data for validator
type ValidatorData struct {
	NodeID            string
	StakeByDelegators *num.Uint
	SelfStake         *num.Uint
	Delegators        map[string]*num.Uint
}
