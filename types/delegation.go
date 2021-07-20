//lint:file-ignore ST1003 Ignore underscores in names, this is straight copied from the proto package to ease introducing the domain types

package types

import (
	commandspb "code.vegaprotocol.io/vega/proto/commands/v1"
)

type Delegate struct {
	NodeId string
	Amount uint64
}

func NewDelegateFromProto(p *commandspb.DelegateSubmission) *Delegate {
	return &Delegate{
		NodeId: p.NodeId,
		Amount: p.Amount,
	}
}

func (d Delegate) IntoProto() *commandspb.DelegateSubmission {
	return &commandspb.DelegateSubmission{
		NodeId: d.NodeId,
		Amount: d.Amount,
	}
}

func (d Delegate) String() string {
	return d.IntoProto().String()
}

type UndelegateAtEpochEnd struct {
	NodeId string
	Amount uint64
}

func NewUndelegateAtEpochEndFromProto(p *commandspb.UndelegateAtEpochEndSubmission) *UndelegateAtEpochEnd {
	return &UndelegateAtEpochEnd{
		NodeId: p.NodeId,
		Amount: p.Amount,
	}
}

func (u UndelegateAtEpochEnd) IntoProto() *commandspb.UndelegateAtEpochEndSubmission {
	return &commandspb.UndelegateAtEpochEndSubmission{
		NodeId: u.NodeId,
		Amount: u.Amount,
	}
}

func (u UndelegateAtEpochEnd) String() string {
	return u.IntoProto().String()
}
