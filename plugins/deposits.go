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
	ErrNoDepositForID = errors.New("no deposit for id")
)

type DepositEvent interface {
	events.Event
	Deposit() types.Deposit
}

type Deposit struct {
	*subscribers.Base

	// FIXME(jeremy): add some reference mapping here later on
	// party -> deposit id -> dephdraal
	deposits map[string]map[string]types.Deposit
	mu       sync.RWMutex
	ch       chan types.Deposit
}

func NewDeposit(ctx context.Context) *Deposit {
	w := &Deposit{
		Base:     subscribers.NewBase(ctx, 10, true),
		deposits: map[string]map[string]types.Deposit{},
		ch:       make(chan types.Deposit, 100),
	}

	go w.consume()
	return w
}

func (w *Deposit) Push(evts ...events.Event) {
	for _, e := range evts {
		select {
		case <-w.Closed():
			return
		default:
			if wse, ok := e.(DepositEvent); ok {
				w.ch <- wse.Deposit()
			}
		}
	}
}

func (w *Deposit) consume() {
	defer func() { close(w.ch) }()
	for {
		select {
		case <-w.Closed():
			return
		case dep, ok := <-w.ch:
			if !ok {
				// cleanup base
				w.Halt()
				// channel is closed
				return
			}
			w.mu.Lock()
			deposits, ok := w.deposits[dep.PartyID]
			if !ok {
				deposits = map[string]types.Deposit{}
				w.deposits[dep.PartyID] = deposits
			}
			deposits[dep.Id] = dep
			w.mu.Unlock()
		}
	}
}

func (d *Deposit) GetByID(id string) (types.Deposit, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	// FIXME(jeremy): this is very naive, and will require
	// a lookup table over the dephposit id -> party
	for _, deposits := range d.deposits {
		for did, deposit := range deposits {
			if did == id {
				return deposit, nil
			}
		}
	}
	return types.Deposit{}, ErrNoDepositForID
}

func (d *Deposit) GetByParty(party string, openOnly bool) []types.Deposit {
	d.mu.RLock()
	defer d.mu.RUnlock()
	out := []types.Deposit{}
	deposits := d.deposits[party]
	for _, dep := range deposits {
		if openOnly && dep.Status != types.Deposit_DEPOSIT_STATUS_OPEN {
			continue
		}
		out = append(out, dep)
	}
	return out
}

func (n *Deposit) Types() []events.Type {
	return []events.Type{
		events.DepositEvent,
	}
}
