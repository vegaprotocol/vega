package subscribers

import (
	"context"
	"sync"

	"code.vegaprotocol.io/vega/events"
	types "code.vegaprotocol.io/vega/proto"
)

type AE interface {
	events.Event
	Account() types.Account
}

type AccountStore interface {
	SaveBatch([]*types.Account) error
}

type AccountSub struct {
	*Base
	store AccountStore
	mu    sync.Mutex
	buf   map[string]*types.Account
}

func NewAccountSub(ctx context.Context, store AccountStore, ack bool) *AccountSub {
	a := &AccountSub{
		Base:  NewBase(ctx, 10, ack),
		store: store,
		buf:   map[string]*types.Account{},
	}
	if a.isRunning() {
		go a.loop(a.ctx)
	}
	return a
}

func (a *AccountSub) loop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			a.Halt()
			return
		case e := <-a.ch:
			if a.isRunning() {
				a.Push(e)
			}
		}
	}
}

func (a *AccountSub) Push(evts ...events.Event) {
	if len(evts) == 0 {
		return
	}
	// lock now, this could be a batch in the future
	a.mu.Lock()
	for _, e := range evts {
		switch et := e.(type) {
		case AE:
			acc := et.Account()
			k := acc.Id
			acc.Id = ""
			a.buf[k] = &acc
		case TimeEvent:
			a.flush()
		}
	}
	a.mu.Unlock()
}

func (a *AccountSub) Types() []events.Type {
	return []events.Type{
		events.AccountEvent,
		events.TimeUpdate,
	}
}

func (a *AccountSub) flush() {
	batch := make([]*types.Account, 0, len(a.buf))
	for _, acc := range a.buf {
		batch = append(batch, acc)
	}
	a.buf = map[string]*types.Account{}
	_ = a.store.SaveBatch(batch)
}
