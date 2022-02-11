package broker

import (
	"code.vegaprotocol.io/vega/events"
	"context"
	"sync"
)

type fanOutEventSource struct {
	source                 eventSource
	sendChannelBufferSize  int
	expectedNumSubscribers int
	numSubscribers         int
	eventChannels          []chan events.Event
	errorChannels          []chan error
	receiveLock            sync.Mutex
}

func NewFanOutEventSource(source eventSource, sendChannelBufferSize int, expectedNumSubscribers int) *fanOutEventSource {
	return &fanOutEventSource{
		source:                 source,
		sendChannelBufferSize:  sendChannelBufferSize,
		expectedNumSubscribers: expectedNumSubscribers,
	}
}

func (e *fanOutEventSource) Listen() error {
	return e.source.Listen()
}

func (e *fanOutEventSource) Receive(ctx context.Context) (<-chan events.Event, <-chan error) {
	e.receiveLock.Lock()
	defer e.receiveLock.Unlock()

	eventsCh := make(chan events.Event, e.sendChannelBufferSize)
	errorCh := make(chan error, 1)

	e.eventChannels = append(e.eventChannels, eventsCh)
	e.errorChannels = append(e.errorChannels, errorCh)

	e.numSubscribers++

	if e.numSubscribers > e.expectedNumSubscribers {
		panic("number of subscribers exceeded expected subscriber count")
	}

	// Once the number of subscribers equals the expected number start forwarding events
	if e.numSubscribers == e.expectedNumSubscribers {
		go e.sendEvents(ctx)
	}

	return eventsCh, errorCh
}

func (e *fanOutEventSource) sendEvents(ctx context.Context) {
	srcEventCh, srcErrorCh := e.source.Receive(ctx)

	for event := range srcEventCh {
		for _, evtCh := range e.eventChannels {
			evtCh <- event
		}
	}

	for _, evtCh := range e.eventChannels {
		close(evtCh)
	}

	select {
	case err := <-srcErrorCh:
		for _, errorCh := range e.errorChannels {
			errorCh <- err
		}
	default:
		// Do nothing, continue
	}

	for _, errorCh := range e.errorChannels {
		close(errorCh)
	}
}
