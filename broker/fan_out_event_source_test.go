package broker_test

import (
	"code.vegaprotocol.io/data-node/broker"
	eventspb "code.vegaprotocol.io/protos/vega/events/v1"
	"code.vegaprotocol.io/vega/events"
	"context"
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestEventFanOut(t *testing.T) {

	tes := &testEventSource{
		eventsCh: make(chan events.Event),
		errorsCh: make(chan error),
	}

	fos := broker.NewFanOutEventSource(tes, 20, 2)

	evtCh1, _ := fos.Receive(context.Background())
	evtCh2, _ := fos.Receive(context.Background())

	tes.eventsCh <- &testEvent{traceId: "1"}
	tes.eventsCh <- &testEvent{traceId: "2"}

	assert.Equal(t, &testEvent{traceId: "1"}, <-evtCh1)
	assert.Equal(t, &testEvent{traceId: "1"}, <-evtCh2)

	assert.Equal(t, &testEvent{traceId: "2"}, <-evtCh1)
	assert.Equal(t, &testEvent{traceId: "2"}, <-evtCh2)

}

func TestCloseChannelsAndExitWithError(t *testing.T) {

	tes := &testEventSource{
		eventsCh: make(chan events.Event),
		errorsCh: make(chan error, 1),
	}

	fos := broker.NewFanOutEventSource(tes, 20, 2)

	evtCh1, errCh1 := fos.Receive(context.Background())
	evtCh2, errCh2 := fos.Receive(context.Background())

	tes.eventsCh <- &testEvent{traceId: "1"}
	assert.Equal(t, &testEvent{traceId: "1"}, <-evtCh1)
	assert.Equal(t, &testEvent{traceId: "1"}, <-evtCh2)

	tes.errorsCh <- fmt.Errorf("e1")
	close(tes.eventsCh)

	assert.Equal(t, fmt.Errorf("e1"), <-errCh1)
	assert.Equal(t, fmt.Errorf("e1"), <-errCh2)

	_, ok := <-evtCh1
	assert.False(t, ok, "channel should be closed")
	_, ok = <-evtCh2
	assert.False(t, ok, "channel should be closed")

}

func TestPanicOnInvalidSubscriberNumber(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatalf("expected panic")
		}
	}()

	tes := &testEventSource{
		eventsCh: make(chan events.Event),
		errorsCh: make(chan error),
	}

	fos := broker.NewFanOutEventSource(tes, 20, 2)

	fos.Receive(context.Background())
	fos.Receive(context.Background())
	fos.Receive(context.Background())
}

type testEventSource struct {
	eventsCh chan events.Event
	errorsCh chan error
}

func (te *testEventSource) Listen() error {
	return nil
}

func (te *testEventSource) Receive(ctx context.Context) (<-chan events.Event, <-chan error) {
	return te.eventsCh, te.errorsCh
}

type testEvent struct {
	traceId string
}

func (t *testEvent) Type() events.Type {
	panic("implement me")
}

func (t *testEvent) Context() context.Context {
	panic("implement me")
}

func (t *testEvent) TraceID() string {
	panic("implement me")
}

func (t *testEvent) TxHash() string {
	panic("implement me")
}

func (t *testEvent) ChainID() string {
	panic("implement me")
}

func (t *testEvent) Sequence() uint64 {
	panic("implement me")
}

func (t *testEvent) SetSequenceID(s uint64) {
	panic("implement me")
}

func (t *testEvent) BlockNr() int64 {
	panic("implement me")
}

func (t *testEvent) StreamMessage() *eventspb.BusEvent {
	panic("implement me")
}
