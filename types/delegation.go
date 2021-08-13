package types

import (
	"errors"

	commandspb "code.vegaprotocol.io/protos/vega/commands/v1"
	"code.vegaprotocol.io/vega/types/num"
)

type Delegate struct {
	NodeID string
	Amount *num.Uint
}

func NewDelegateFromProto(p *commandspb.DelegateSubmission) (*Delegate, error) {
	amount, ok := num.UintFromString(p.Amount, 10)
	if !ok {
		return nil, errors.New("invalid amount")
	}
	return &Delegate{
		NodeID: p.NodeId,
		Amount: amount,
	}, nil
}

func (d Delegate) IntoProto() *commandspb.DelegateSubmission {
	return &commandspb.DelegateSubmission{
		NodeId: d.NodeID,
		Amount: num.UintToString(d.Amount),
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

func NewUndelegateFromProto(p *commandspb.UndelegateSubmission) (*Undelegate, error) {
	amount, ok := num.UintFromString(p.Amount, 10)
	if !ok {
		return nil, errors.New("invalid amount")
	}
	return &Undelegate{
		NodeID: p.NodeId,
		Amount: amount,
		Method: p.Method.String(),
	}, nil
}

func (u Undelegate) IntoProto() *commandspb.UndelegateSubmission {
	return &commandspb.UndelegateSubmission{
		NodeId: u.NodeID,
		Amount: num.UintToString(u.Amount),
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
