package types

import (
	"errors"

	"code.vegaprotocol.io/protos/vega"
	commandspb "code.vegaprotocol.io/protos/vega/commands/v1"
	"code.vegaprotocol.io/vega/types/num"
)

type Delegation struct {
	Party    string
	NodeID   string
	Amount   *num.Uint
	EpochSeq string
}

func DelegationFromProto(d *vega.Delegation) *Delegation {
	amt, _ := num.UintFromString(d.Amount, 10)
	return &Delegation{
		Party:    d.Party,
		NodeID:   d.NodeId,
		Amount:   amt,
		EpochSeq: d.EpochSeq,
	}
}

func (d Delegation) IntoProto() *vega.Delegation {
	return &vega.Delegation{
		Party:    d.Party,
		NodeId:   d.NodeID,
		Amount:   num.UintToString(d.Amount),
		EpochSeq: d.EpochSeq,
	}
}

type Delegate struct {
	NodeID string
	Amount *num.Uint
}

func NewDelegateFromProto(p *commandspb.DelegateSubmission) (*Delegate, error) {
	amount := num.Zero()
	if len(p.Amount) > 0 {
		var overflowed bool
		amount, overflowed = num.UintFromString(p.Amount, 10)
		if overflowed {
			return nil, errors.New("invalid amount")
		}
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
	amount, overflowed := num.UintFromString(p.Amount, 10)
	if overflowed {
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

// ValidatorData is delegation data for validator.
type ValidatorData struct {
	NodeID            string
	PubKey            string
	StakeByDelegators *num.Uint
	SelfStake         *num.Uint
	Delegators        map[string]*num.Uint
	TmPubKey          string
}

// ValidatorVotingPower is the scaled voting power for the given tm key.
type ValidatorVotingPower struct {
	TmPubKey    string
	VotingPower int64
}
