package subscribers

import (
	"context"
	"time"

	"code.vegaprotocol.io/vega/events"
	types "code.vegaprotocol.io/vega/proto"
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
	ctx     context.Context
	ch      chan events.Event
	sCh     chan struct{}
	running bool
	store   TransferResponseStore
	trs     []*types.TransferResponse
}

func NewTransferResponse(ctx context.Context, store TransferResponseStore) *TransferResponse {
	s := &TransferResponse{
		ctx:     ctx,
		ch:      make(chan events.Event),
		sCh:     make(chan struct{}),
		running: true,
		store:   store,
		trs:     []*types.TransferResponse{},
	}
	go s.loop()
	return s
}

func (t *TransferResponse) Pause() {
	if t.running {
		t.running = false
		close(t.sCh)
	}
}

func (t *TransferResponse) Resume() {
	if !t.running {
		t.running = true
		t.sCh = make(chan struct{})
	}
}

func (t *TransferResponse) loop() {
	for {
		select {
		case <-t.ctx.Done():
			return
		case e := <-t.ch:
			if t.running {
				t.Push(e)
			}
		}
	}
}

func (t *TransferResponse) Skip() <-chan struct{} {
	return t.sCh
}

func (t *TransferResponse) Closed() <-chan struct{} {
	return t.ctx.Done()
}

func (t *TransferResponse) C() chan<- events.Event {
	return t.ch
}

func (t *TransferResponse) Types() []events.Type {
	return []events.Type{
		events.TimeUpdate,
		events.TransferResponses,
	}
}

func (t *TransferResponse) flush() error {
	trs := t.trs
	t.trs = []*types.TransferResponse{}
	if len(trs) == 0 {
		return nil
	}
	return t.store.SaveBatch(trs)
}

// Push - takes the event pushed by the broker
func (t *TransferResponse) Push(e events.Event) {
	switch te := e.(type) {
	case TimeEvent:
		_ = t.flush()
	case TransferResponseEvent:
		t.trs = append(t.trs, te.TransferResponses()...)
	}
}
