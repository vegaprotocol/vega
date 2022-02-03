package broker

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"
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

func (e fileEventSource) listen() error {
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
	defer eventFile.Close()

	if err != nil {
		errorCh <- err
		close(out)
		return
	}

	sizeBytes := make([]byte, 4)
	eventBlock := make([]*eventspb.BusEvent, 0)
	var offset int64 = 0
	currentBlock := ""

	terminateCh := make(chan os.Signal, 1)
	signal.Notify(terminateCh, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)

	for {
		select {
		case <-terminateCh:
			close(out)
			close(errorCh)
			return
		default:

			read, err := eventFile.ReadAt(sizeBytes, offset)

			if err == io.EOF {
				// Nothing more to read, send any pending messages and return
				// Do not close channels, want it to behave the same way as socket based source as much as possible
				sendBlock(ctx, out, eventBlock)
				return
			}

			if err != nil {
				errorCh <- fmt.Errorf("error whilst reading message size from events file:%w", err)
				close(out)
				return
			}

			offset += int64(read)
			msgSize := binary.BigEndian.Uint32(sizeBytes)
			msgBytes := make([]byte, msgSize)
			read, err = eventFile.ReadAt(msgBytes, offset)
			if err != nil {
				errorCh <- fmt.Errorf("error whilst reading message bytes from events file:%w", err)
				close(out)
				return
			}

			offset += int64(read)

			event := &eventspb.BusEvent{}
			err = proto.Unmarshal(msgBytes, event)
			if err != nil {
				errorCh <- fmt.Errorf("failed to unmarshal bus event: %w", err)
				close(out)
				return
			}

			if event.Block != currentBlock {
				sendBlock(ctx, out, eventBlock)
				eventBlock = eventBlock[:0]
				time.Sleep(timeBetweenBlocks)
				currentBlock = event.Block
			}

			eventBlock = append(eventBlock, event)
		}
	}
}

func sendBlock(ctx context.Context, out chan<- events.Event, batch []*eventspb.BusEvent) {
	for _, busEvent := range batch {
		evt := toEvent(ctx, busEvent)
		out <- evt
	}
}
