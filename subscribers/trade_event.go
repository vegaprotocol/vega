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

type TE interface {
	events.Event
	Trade() types.Trade
}

type TradeStore interface {
	SaveBatch([]types.Trade) error
}

type TradeSub struct {
	*Base
	mu    sync.Mutex
	buf   []types.Trade
	store TradeStore
	log   *logging.Logger
}

func NewTradeSub(ctx context.Context, store TradeStore, log *logging.Logger, ack bool) *TradeSub {
	t := &TradeSub{
		Base:  NewBase(ctx, 10, ack),
		buf:   []types.Trade{},
		store: store,
		log:   log,
	}
	if t.isRunning() {
		go t.loop(t.ctx)
	}
	return t
}

func (t *TradeSub) loop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			t.Halt()
			return
		case e := <-t.ch:
			if t.isRunning() {
				t.Push(e...)
			}
		}
	}
}

func (t *TradeSub) Push(evts ...events.Event) {
	if len(evts) == 0 {
		return
	}
	// acquire lock here, so a time event doesn't result in a partial flush
	t.mu.Lock()
	for _, e := range evts {
		switch te := e.(type) {
		case TE:
			t.buf = append(t.buf, te.Trade())
		case TimeEvent:
			t.flush()
		default:
			t.log.Panic("Unknown event type in trade subscriber", logging.String("Type", te.Type().String()))
		}
	}
	t.mu.Unlock()
}

func (t *TradeSub) flush() {
	b := t.buf
	t.buf = make([]types.Trade, 0, cap(b))
	_ = t.store.SaveBatch(b)
}

func (t *TradeSub) Types() []events.Type {
	return []events.Type{
		events.TradeEvent,
		events.TimeUpdate,
	}
}
