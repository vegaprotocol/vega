package broker_test

import (
	"context"
	"fmt"
	"testing"

	"code.vegaprotocol.io/data-node/broker"
	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/types"

	"github.com/stretchr/testify/assert"
)

func TestEventFanOut(t *testing.T) {
	tes := &testEventSource{
		eventsCh: make(chan events.Event),
		errorsCh: make(chan error),
	}

	fos := broker.NewFanOutEventSource(tes, 20, 2)

	evtCh1, _ := fos.Receive(context.Background())
	evtCh2, _ := fos.Receive(context.Background())

	tes.eventsCh <- events.NewAssetEvent(context.Background(), types.Asset{ID: "a1"})
	tes.eventsCh <- events.NewAssetEvent(context.Background(), types.Asset{ID: "a2"})

	assert.Equal(t, events.NewAssetEvent(context.Background(), types.Asset{ID: "a1"}), <-evtCh1)
	assert.Equal(t, events.NewAssetEvent(context.Background(), types.Asset{ID: "a1"}), <-evtCh2)

	assert.Equal(t, events.NewAssetEvent(context.Background(), types.Asset{ID: "a2"}), <-evtCh1)
	assert.Equal(t, events.NewAssetEvent(context.Background(), types.Asset{ID: "a2"}), <-evtCh2)
}

func TestCloseChannelsAndExitWithError(t *testing.T) {
	tes := &testEventSource{
		eventsCh: make(chan events.Event),
		errorsCh: make(chan error, 1),
	}

	fos := broker.NewFanOutEventSource(tes, 20, 2)

	evtCh1, errCh1 := fos.Receive(context.Background())
	evtCh2, errCh2 := fos.Receive(context.Background())

	tes.eventsCh <- events.NewAssetEvent(context.Background(), types.Asset{ID: "a1"})
	assert.Equal(t, events.NewAssetEvent(context.Background(), types.Asset{ID: "a1"}), <-evtCh1)
	assert.Equal(t, events.NewAssetEvent(context.Background(), types.Asset{ID: "a1"}), <-evtCh2)

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

func TestListenOnlyCalledOnceOnSource(t *testing.T) {
	tes := &testEventSource{
		eventsCh: make(chan events.Event),
		errorsCh: make(chan error),
	}

	fos := broker.NewFanOutEventSource(tes, 20, 2)
	fos.Listen()
	fos.Listen()
	fos.Listen()

	assert.Equal(t, 1, tes.listenCount)
}

type testEventSource struct {
	eventsCh    chan events.Event
	errorsCh    chan error
	listenCount int
}

func (te *testEventSource) Listen() error {
	te.listenCount++
	return nil
}

func (te *testEventSource) Receive(ctx context.Context) (<-chan events.Event, <-chan error) {
	return te.eventsCh, te.errorsCh
}
