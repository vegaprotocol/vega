// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.DATANODE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package broker

import (
	"context"
	"fmt"
	"sync"

	"code.vegaprotocol.io/vega/core/events"
)

// fanOutEventSource: an event source to fan out an event stream, it is told in advance the number of subscribers to
// expect and only starts publishing events once that number of subscriptions has been received.
type fanOutEventSource struct {
	source                 EventReceiver
	sendChannelBufferSize  int
	expectedNumSubscribers int
	numSubscribers         int
	eventChannels          []chan events.Event
	errorChannels          []chan error
	listening              bool
	mutex                  sync.Mutex
}

//revive:disable:unexported-return
func NewFanOutEventSource(source EventReceiver, sendChannelBufferSize int, expectedNumSubscribers int) *fanOutEventSource {
	return &fanOutEventSource{
		source:                 source,
		sendChannelBufferSize:  sendChannelBufferSize,
		expectedNumSubscribers: expectedNumSubscribers,
	}
}

func (e *fanOutEventSource) Listen() error {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	if !e.listening {
		err := e.source.Listen()
		if err != nil {
			return err
		}
		e.listening = true
	}

	return nil
}

func (e *fanOutEventSource) Receive(ctx context.Context) (<-chan events.Event, <-chan error) {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	eventsCh := make(chan events.Event, e.sendChannelBufferSize)
	errorCh := make(chan error, 1)

	e.eventChannels = append(e.eventChannels, eventsCh)
	e.errorChannels = append(e.errorChannels, errorCh)

	e.numSubscribers++

	if e.numSubscribers > e.expectedNumSubscribers {
		panic("number of subscribers exceeded expected subscriber number")
	}

	// Once the number of subscribers equals the expected number start forwarding events
	if e.numSubscribers == e.expectedNumSubscribers {
		go e.sendEvents(ctx)
	}

	return eventsCh, errorCh
}

func (e *fanOutEventSource) sendEvents(ctx context.Context) {
	srcEventCh, srcErrorCh := e.source.Receive(ctx)

	defer func() {
		for _, evtCh := range e.eventChannels {
			close(evtCh)
		}

		for _, errorCh := range e.errorChannels {
			close(errorCh)
		}
	}()

	var firstBlock int64
	var prevBlock int64
	var prevSeq uint64
	first := true

	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-srcEventCh:
			if !ok {
				return
			}
			// Check that the sequence of events is sane:
			//  - First event can be at any height; but event must have sequence number 1
			//  - Subsequent events must either increase sequence number by 1 OR
			//    Go back to 1 and increase block number by 1
			//  Exception:
			//   - We get two blocks with the first block number; presumably because we have genesis
			//     or checkpoint; and then the first real block.
			if first {
				firstBlock = event.BlockNr()
				first = false
			}

			firstBlockFirstEvent := event.BlockNr() == firstBlock && event.Sequence() == 1
			sameBlockNextEvent := event.BlockNr() == prevBlock && event.Sequence() == prevSeq+1
			nextBlockFirstEvent := event.BlockNr() == prevBlock+1 && event.Sequence() == 1
			firstBlockAgain := event.BlockNr() == firstBlock && prevBlock == firstBlock && event.Sequence() == 1
			ok = firstBlockFirstEvent || sameBlockNextEvent || nextBlockFirstEvent || firstBlockAgain

			if !ok {
				panic(fmt.Sprintf("non-contiguous event stream: last: %d-%d, received: %d-%d",
					prevBlock, prevSeq, event.BlockNr(), event.Sequence()))
			}

			prevBlock, prevSeq = event.BlockNr(), event.Sequence()

			// Now get on with sending event to listeners
			for _, evtCh := range e.eventChannels {
				// Listen for context cancels, even if we're blocked sending events
				select {
				case evtCh <- event:
				case <-ctx.Done():
					return
				}
			}

		case err, ok := <-srcErrorCh:
			if !ok {
				return
			}
			for _, errorCh := range e.errorChannels {
				errorCh <- err
			}
		}
	}
}
