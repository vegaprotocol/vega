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

package broker

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"time"

	eventspb "code.vegaprotocol.io/protos/vega/events/v1"
	"code.vegaprotocol.io/vega/events"

	"github.com/golang/protobuf/proto"
)

type fileEventSource struct {
	eventsFile            string
	timeBetweenBlocks     time.Duration
	sendChannelBufferSize int
}

func NewFileEventSource(file string, timeBetweenBlocks time.Duration,
	sendChannelBufferSize int) (*fileEventSource, error,
) {
	return &fileEventSource{
		eventsFile:            file,
		timeBetweenBlocks:     timeBetweenBlocks,
		sendChannelBufferSize: sendChannelBufferSize,
	}, nil
}

func (e fileEventSource) Listen() error {
	return nil
}

func (e fileEventSource) Receive(ctx context.Context) (<-chan events.Event, <-chan error) {
	eventsCh := make(chan events.Event, e.sendChannelBufferSize)
	errorCh := make(chan error, 1)

	go sendAllEvents(ctx, eventsCh, e.eventsFile, e.timeBetweenBlocks, errorCh)

	return eventsCh, errorCh
}

func sendAllEvents(ctx context.Context, out chan<- events.Event, file string,
	timeBetweenBlocks time.Duration, errorCh chan<- error,
) {
	eventFile, err := os.Open(file)
	defer func() {
		eventFile.Close()
		close(out)
		close(errorCh)
	}()

	if err != nil {
		errorCh <- err
		return
	}

	sizeBytes := make([]byte, 4)
	eventBlock := make([]*eventspb.BusEvent, 0)
	var offset int64 = 0
	currentBlock := ""

	for {
		select {
		case <-ctx.Done():
			return
		default:

			read, err := eventFile.ReadAt(sizeBytes, offset)

			if err == io.EOF {
				// Nothing more to read, send any pending messages. Do not immediately close our
				// output channel, instead sit and wait for our context to be cancelled (e.g. by a
				// shutdown), so as not to trigger a premature exit.
				sendBlock(ctx, out, eventBlock)
				<-ctx.Done()
				return
			}

			if err != nil {
				errorCh <- fmt.Errorf("error whilst reading message size from events file:%w", err)
				return
			}

			offset += int64(read)
			msgSize := binary.BigEndian.Uint32(sizeBytes)
			msgBytes := make([]byte, msgSize)
			read, err = eventFile.ReadAt(msgBytes, offset)
			if err != nil {
				errorCh <- fmt.Errorf("error whilst reading message bytes from events file:%w", err)
				return
			}

			offset += int64(read)

			event := &eventspb.BusEvent{}
			err = proto.Unmarshal(msgBytes, event)
			if err != nil {
				errorCh <- fmt.Errorf("failed to unmarshal bus event: %w", err)
				return
			}

			if event.Block != currentBlock {
				if err := sendBlock(ctx, out, eventBlock); err != nil {
					errorCh <- err
					return
				}
				eventBlock = eventBlock[:0]
				time.Sleep(timeBetweenBlocks)
				currentBlock = event.Block
			}

			eventBlock = append(eventBlock, event)
		}
	}
}

func sendBlock(ctx context.Context, out chan<- events.Event, batch []*eventspb.BusEvent) error {
	for _, busEvent := range batch {
		evt := toEvent(ctx, busEvent)
		// Listen for context cancels, even if we're blocked sending events
		select {
		case out <- evt:
		case <-ctx.Done():
			return ctx.Err()
		}

	}
	return nil
}
