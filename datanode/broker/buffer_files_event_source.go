package broker

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"code.vegaprotocol.io/vega/core/events"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"
)

type bufferFileEventSource struct {
	bufferFilesDir        string
	timeBetweenBlocks     time.Duration
	sendChannelBufferSize int
	chainID               string
	archiveFiles          []fs.FileInfo
	currentBlock          string
}

//revive:disable:unexported-return
func NewBufferFilesEventSource(bufferFilesDir string, timeBetweenBlocks time.Duration,
	sendChannelBufferSize int, chainID string) (*bufferFileEventSource, error,
) {
	var archiveFiles []fs.FileInfo
	err := filepath.Walk(bufferFilesDir, func(path string, info fs.FileInfo, err error) error {
		if !info.IsDir() {
			archiveFiles = append(archiveFiles, info)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to walk directory: %w", err)
	}

	// We rely on the name to sort the files in age order, oldest first
	sort.Slice(archiveFiles, func(i, j int) bool {
		return strings.Compare(archiveFiles[i].Name(), archiveFiles[j].Name()) < 0
	})

	return &bufferFileEventSource{
		bufferFilesDir:        bufferFilesDir,
		timeBetweenBlocks:     timeBetweenBlocks,
		sendChannelBufferSize: sendChannelBufferSize,
		chainID:               chainID,
		archiveFiles:          archiveFiles,
	}, nil
}

func (e *bufferFileEventSource) Listen() error {
	return nil
}

func (e *bufferFileEventSource) Send(events.Event) error {
	return nil
}

func (e *bufferFileEventSource) Receive(ctx context.Context) (<-chan events.Event, <-chan error) {
	eventsCh := make(chan events.Event, e.sendChannelBufferSize)
	errorCh := make(chan error, 1)

	go func() {
		for _, eventFile := range e.archiveFiles {
			err := e.sendAllBufferedEventsInFile(ctx, eventsCh, filepath.Join(e.bufferFilesDir, eventFile.Name()),
				e.timeBetweenBlocks, e.chainID)
			if err != nil {
				errorCh <- fmt.Errorf("failed to send events in buffer file: %w", err)
			}
		}
	}()

	return eventsCh, errorCh
}

func (e *bufferFileEventSource) sendAllBufferedEventsInFile(ctx context.Context, out chan<- events.Event, file string,
	timeBetweenBlocks time.Duration, chainID string,
) error {
	eventFile, err := os.Open(file)
	defer func() {
		_ = eventFile.Close()
	}()

	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}

	eventBlock := make([]*eventspb.BusEvent, 0)
	var offset int64

	for {
		select {
		case <-ctx.Done():
			return nil
		default:

			event, _, read, err := readBufferedEvent(eventFile, offset)
			if err != nil {
				return fmt.Errorf("failed to read buffered event:%w", err)
			}

			if read == 0 {
				err = sendBufferedBlock(ctx, out, eventBlock)
				if err != nil {
					return fmt.Errorf("send block failed:%w", err)
				}
				return nil
			}

			offset += int64(read)

			// Buffer files do not necessarily start on block boundaries, to prevent sending part of a block
			// events are ignored until an initial begin block event is encountered
			if len(e.currentBlock) == 0 {
				if event.Type == eventspb.BusEventType_BUS_EVENT_TYPE_BEGIN_BLOCK {
					e.currentBlock = event.Block
				} else {
					continue
				}
			}

			err = checkChainID(chainID, event.ChainId)

			if err != nil {
				return fmt.Errorf("check chain id failed: %w", err)
			}

			if event.Block != e.currentBlock {
				if err := sendBufferedBlock(ctx, out, eventBlock); err != nil {
					return fmt.Errorf("failed to send buffered block: %w", err)
				}
				eventBlock = eventBlock[:0]
				time.Sleep(timeBetweenBlocks)
				e.currentBlock = event.Block
			}

			eventBlock = append(eventBlock, event)
		}
	}
}

func sendBufferedBlock(ctx context.Context, out chan<- events.Event, batch []*eventspb.BusEvent) error {
	for _, busEvent := range batch {
		evt := toEvent(ctx, busEvent)

		select {
		case out <- evt:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return nil
}
