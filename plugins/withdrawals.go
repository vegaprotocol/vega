package plugins

import (
	"context"
	"sync"

	"code.vegaprotocol.io/vega/events"
	types "code.vegaprotocol.io/vega/proto"
	"code.vegaprotocol.io/vega/subscribers"

	"github.com/pkg/errors"
)

var (
	ErrNoWithdrawalForID = errors.New("no withdrawal for id")
)

type WithdrawalEvent interface {
	events.Event
	Withdrawal() types.Withdrawal
}

type Withdrawal struct {
	*subscribers.Base

	// FIXME(jeremy): add some reference mapping here later on
	// party -> withdrawal id -> withdraal
	withdrawals map[string]map[string]types.Withdrawal
	mu          sync.RWMutex
	ch          chan types.Withdrawal
}

func NewWithdrawal(ctx context.Context) *Withdrawal {
	w := &Withdrawal{
		Base:        subscribers.NewBase(ctx, 10, true),
		withdrawals: map[string]map[string]types.Withdrawal{},
		ch:          make(chan types.Withdrawal, 100),
	}

	go w.consume()
	return w
}

func (w *Withdrawal) Push(evts ...events.Event) {
	for _, e := range evts {
		select {
		case <-w.Closed():
			return
		default:
			if wse, ok := e.(WithdrawalEvent); ok {
				w.ch <- wse.Withdrawal()
			}
		}
	}
}

func (w *Withdrawal) consume() {
	defer func() { close(w.ch) }()
	for {
		select {
		case <-w.Closed():
			return
		case wit, ok := <-w.ch:
			if !ok {
				// cleanup base
				w.Halt()
				// channel is closed
				return
			}
			w.mu.Lock()
			withdrawals, ok := w.withdrawals[wit.PartyID]
			if !ok {
				withdrawals = map[string]types.Withdrawal{}
				w.withdrawals[wit.PartyID] = withdrawals
			}
			withdrawals[wit.Id] = wit
			w.mu.Unlock()
		}
	}
}

func (w *Withdrawal) GetByID(id string) (types.Withdrawal, error) {
	w.mu.RLock()
	defer w.mu.RUnlock()
	// FIXME(jeremy): this is very naive, and will require
	// a lookup table over the withdrwal id -> party
	for _, withdrawals := range w.withdrawals {
		for wid, withdrawal := range withdrawals {
			if wid == id {
				return withdrawal, nil
			}
		}
	}
	return types.Withdrawal{}, ErrNoWithdrawalForID
}

func (w *Withdrawal) GetByParty(party string, openOnly bool) []types.Withdrawal {
	w.mu.RLock()
	defer w.mu.RUnlock()
	out := []types.Withdrawal{}
	withdrawals := w.withdrawals[party]
	for _, w := range withdrawals {
		if openOnly && w.Status != types.Withdrawal_STATUS_OPEN {
			continue
		}
		out = append(out, w)
	}
	return out
}

func (*Withdrawal) Types() []events.Type {
	return []events.Type{
		events.WithdrawalEvent,
	}
}
