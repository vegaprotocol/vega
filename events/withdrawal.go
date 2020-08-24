package events

import (
	"context"

	types "code.vegaprotocol.io/vega/proto"
)

type Withdrawal struct {
	*Base
	w *types.Withdrawal
}

func NewWithdrawalEvent(ctx context.Context, w *types.Withdrawal) *Withdrawal {
	return &Withdrawal{
		Base: newBase(ctx, WithdrawalEvent),
		w:    w,
	}
}

func (w *Withdrawal) Withdrawal() *types.Withdrawal {
	return w.w
}
