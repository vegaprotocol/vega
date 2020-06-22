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
	*Base
	store TransferResponseStore
	trs   []*types.TransferResponse
}

func NewTransferResponse(ctx context.Context, store TransferResponseStore) *TransferResponse {
	s := &TransferResponse{
		Base:  newBase(ctx, 0),
		store: store,
		trs:   []*types.TransferResponse{},
	}
	s.running = true
	go s.loop()
	return s
}

func (t *TransferResponse) loop() {
	for {
		select {
		case <-t.ctx.Done():
			t.Halt() // cleanup what we can
			return
		case e := <-t.ch:
			if t.running {
				t.Push(e)
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
