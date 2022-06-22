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

	"code.vegaprotocol.io/data-node/logging"
	types "code.vegaprotocol.io/protos/vega"
	"code.vegaprotocol.io/vega/events"
)

// NME - NewMarketEvent
type NME interface {
	Market() types.Market
}

type MarketStore interface {
	SaveBatch(markets []types.Market) error
}

type Market struct {
	*Base
	store MarketStore
	log   *logging.Logger
}

func NewMarketSub(ctx context.Context, store MarketStore, log *logging.Logger, ack bool) *Market {
	m := &Market{
		Base:  NewBase(ctx, 1, ack),
		store: store,
		log:   log,
	}
	if m.isRunning() {
		go m.loop(m.ctx)
	}
	return m
}

func (m *Market) loop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			m.Halt()
			return
		case e := <-m.ch:
			if m.isRunning() {
				m.Push(e...)
			}
		}
	}
}

func (m *Market) Push(evts ...events.Event) {
	batch := make([]types.Market, 0, len(evts))
	for _, e := range evts {
		switch et := e.(type) {
		case NME:
			batch = append(batch, et.Market())
		default:
			m.log.Panic("Unknown event type in market subscriber", logging.String("Type", et.Type().String()))
		}
	}
	if len(batch) > 0 {
		_ = m.store.SaveBatch(batch)
	}
}

func (m *Market) Types() []events.Type {
	return []events.Type{
		events.MarketCreatedEvent,
	}
}
