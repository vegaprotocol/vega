package events

import (
	"context"

	types "code.vegaprotocol.io/vega/proto"
)

type Deposit struct {
	*Base
	d types.Deposit
}

func NewDepositEvent(ctx context.Context, d types.Deposit) *Deposit {
	return &Deposit{
		Base: newBase(ctx, DepositEvent),
		d:    d,
	}
}

func (d *Deposit) Deposit() types.Deposit {
	return d.d
}

func (d Deposit) PartyID() string { return d.d.PartyID }
