package subscribers

import (
	"context"
	"sync"

	"code.vegaprotocol.io/data-node/events"
	"code.vegaprotocol.io/data-node/logging"
	types "code.vegaprotocol.io/protos/vega"
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
	log   *logging.Logger
}

func NewAccountSub(ctx context.Context, store AccountStore, log *logging.Logger, ack bool) *AccountSub {
	a := &AccountSub{
		Base:  NewBase(ctx, 10, ack),
		store: store,
		buf:   map[string]*types.Account{},
		log:   log,
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
				a.Push(e...)
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
		default:
			a.log.Panic("Unknown event type in account subscriber", logging.String("Type", et.Type().String()))
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
