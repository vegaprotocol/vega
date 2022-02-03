package broker

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	eventspb "code.vegaprotocol.io/protos/vega/events/v1"
	"code.vegaprotocol.io/vega/events"

	"github.com/golang/protobuf/proto"
)

type fileEventSource struct {
	eventsFile            EventFile
	timeBetweenBlocks     time.Duration
	sendChannelBufferSize int
}

type EventFile interface {
	Open() error
	Close() error
	ReadAt(b []byte, off int64) (n int, err error)
}

func NewFileEventSource(eventsFile EventFile, timeBetweenBlocks time.Duration,
	sendChannelBufferSize int) (*fileEventSource, error,
) {
	return &fileEventSource{
		eventsFile:            eventsFile,
		timeBetweenBlocks:     timeBetweenBlocks,
		sendChannelBufferSize: sendChannelBufferSize,
	}, nil
}

func (e fileEventSource) listen() error {
	return nil
}

func (e fileEventSource) Receive(ctx context.Context) (<-chan events.Event, <-chan error) {
	eventsCh := make(chan events.Event, e.sendChannelBufferSize)
	errorCh := make(chan error)

	go sendAllEvents(ctx, eventsCh, e.eventsFile, e.timeBetweenBlocks, errorCh)

	return eventsCh, errorCh
}

func sendAllEvents(ctx context.Context, out chan<- events.Event, eventFile EventFile,
	timeBetweenBlocks time.Duration, errorCh chan<- error,
) {
	err := eventFile.Open()
	defer eventFile.Close()

	if err != nil {
		errorCh <- err
		return
	}

	sizeBytes := make([]byte, 4)
	msgBytes := make([]byte, 0, 10000)
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
				return
			}

			offset += int64(read)
			msgSize := binary.BigEndian.Uint32(sizeBytes)
			msgBytes = msgBytes[:msgSize]
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

type eventFile struct {
	path    string
	absPath string
	file    *os.File
}

func newEventFile(path string) (*eventFile, error) {
	filePath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("unable to determine absolute path of file %s: %w", path, err)
	}

	return &eventFile{path: path, absPath: filePath}, nil
}

func (e *eventFile) AbsPath() string {
	return e.absPath
}

func (e *eventFile) Open() error {
	file, err := os.Open(e.path)
	if err != nil {
		return err
	}
	e.file = file
	return nil
}

func (e *eventFile) Close() error {
	if e.file == nil {
		return fmt.Errorf("event file has not been opened")
	}
	return e.file.Close()
}

func (e *eventFile) ReadAt(b []byte, off int64) (n int, err error) {
	if e.file == nil {
		return 0, fmt.Errorf("event file has not been opened")
	}
	return e.file.ReadAt(b, off)
}
