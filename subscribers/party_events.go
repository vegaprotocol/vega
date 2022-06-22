// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package subscribers

import (
	"context"
	"sync"

	"code.vegaprotocol.io/data-node/logging"
	types "code.vegaprotocol.io/protos/vega"
	"code.vegaprotocol.io/vega/events"
)

type PE interface {
	events.Event
	Party() types.Party
}

type PartyStore interface {
	SaveBatch(order []types.Party) error
}

type PartySub struct {
	*Base
	mu    sync.Mutex
	store PartyStore
	buf   []types.Party
	log   *logging.Logger
}

func NewPartySub(ctx context.Context, store PartyStore, log *logging.Logger, ack bool) *PartySub {
	a := &PartySub{
		Base:  NewBase(ctx, 10, ack),
		store: store,
		buf:   []types.Party{},
		log:   log,
	}
	if a.isRunning() {
		go a.loop(a.ctx)
	}
	return a
}

func (p *PartySub) loop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			p.Halt()
			return
		case e := <-p.ch:
			if p.isRunning() {
				p.Push(e...)
			}
		}
	}
}

func (p *PartySub) Push(evts ...events.Event) {
	for _, e := range evts {
		switch et := e.(type) {
		case PE:
			party := et.Party()
			p.mu.Lock()
			p.buf = append(p.buf, party)
			p.mu.Unlock()
		case TimeEvent:
			p.flush()
		default:
			p.log.Panic("Unknown event type in party subscriber", logging.String("Type", et.Type().String()))
		}
	}
}

func (*PartySub) Types() []events.Type {
	return []events.Type{
		events.PartyEvent,
		events.TimeUpdate,
	}
}

func (p *PartySub) flush() {
	p.mu.Lock()
	cpy := p.buf
	p.buf = make([]types.Party, 0, cap(cpy))
	p.mu.Unlock()
	_ = p.store.SaveBatch(cpy)
}
