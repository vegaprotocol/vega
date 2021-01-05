package subscribers

import (
	"context"
	"sync"
	"time"

	"code.vegaprotocol.io/vega/events"
	types "code.vegaprotocol.io/vega/proto/gen/golang"
)

// TransferResponseStore ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/transfer_response_store_mock.go -package mocks code.vegaprotocol.io/vega/subscribers TransferResponseStore
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
}

func NewTransferResponse(ctx context.Context, store TransferResponseStore, ack bool) *TransferResponse {
	s := &TransferResponse{
		Base:  NewBase(ctx, 0, ack),
		store: store,
		trs:   []*types.TransferResponse{},
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
		}
	}
}
