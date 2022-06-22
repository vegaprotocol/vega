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
	"time"

	"code.vegaprotocol.io/data-node/logging"
	types "code.vegaprotocol.io/protos/vega"
	"code.vegaprotocol.io/vega/events"
)

// TransferResponseStore ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/transfer_response_store_mock.go -package mocks code.vegaprotocol.io/data-node/subscribers TransferResponseStore
type TransferResponseStore interface {
	SaveBatch([]*types.TransferResponse) error
}

type TimeEvent interface {
	Time() time.Time
}

type TransferResponseEvent interface {
	TransferResponses() []*types.TransferResponse
}

// TransferResponse is a buffer for all transfer responses
// produced during a block by vega
type TransferResponse struct {
	*Base
	mu    sync.Mutex
	store TransferResponseStore
	trs   []*types.TransferResponse
	log   *logging.Logger
}

func NewTransferResponse(ctx context.Context, store TransferResponseStore, log *logging.Logger, ack bool) *TransferResponse {
	s := &TransferResponse{
		Base:  NewBase(ctx, 0, ack),
		store: store,
		trs:   []*types.TransferResponse{},
		log:   log,
	}
	if s.isRunning() {
		go s.loop()
	}
	return s
}

func (t *TransferResponse) loop() {
	for {
		select {
		case <-t.ctx.Done():
			t.Halt() // cleanup what we can
			return
		case e := <-t.ch:
			if t.isRunning() {
				t.Push(e...)
			}
		}
	}
}

func (t *TransferResponse) Types() []events.Type {
	return []events.Type{
		events.TimeUpdate,
		events.TransferResponses,
	}
}

func (t *TransferResponse) flush() error {
	t.mu.Lock()
	trs := t.trs
	t.trs = []*types.TransferResponse{}
	t.mu.Unlock()
	if len(trs) == 0 {
		return nil
	}
	return t.store.SaveBatch(trs)
}

// Push - takes the event pushed by the broker
// in this case, transfer responses are already packaged into a single event
// but this may change over time. In that case, the use of the mutex needs to be updated
func (t *TransferResponse) Push(evts ...events.Event) {
	for _, e := range evts {
		switch te := e.(type) {
		case TimeEvent:
			_ = t.flush()
		case TransferResponseEvent:
			t.mu.Lock()
			t.trs = append(t.trs, te.TransferResponses()...)
			t.mu.Unlock()
		default:
			t.log.Panic("Unknown event type in transfer response subscriber", logging.String("Type", te.Type().String()))
		}
	}
}
