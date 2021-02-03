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

func (d *Deposit) Push(evts ...events.Event) {
	for _, e := range evts {
		select {
		case <-d.Closed():
			return
		default:
			if wse, ok := e.(DepositEvent); ok {
				d.ch <- wse.Deposit()
			}
		}
	}
}

func (d *Deposit) consume() {
	defer func() { close(d.ch) }()
	for {
		select {
		case <-d.Closed():
			return
		case dep, ok := <-d.ch:
			if !ok {
				// cleanup base
				d.Halt()
				// channel is closed
				return
			}
			d.mu.Lock()
			deposits, ok := d.deposits[dep.PartyId]
			if !ok {
				deposits = map[string]types.Deposit{}
				d.deposits[dep.PartyId] = deposits
			}
			deposits[dep.Id] = dep
			d.mu.Unlock()
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
		if openOnly && dep.Status != types.Deposit_STATUS_OPEN {
			continue
		}
		out = append(out, dep)
	}
	return out
}

func (*Deposit) Types() []events.Type {
	return []events.Type{
		events.DepositEvent,
	}
}
