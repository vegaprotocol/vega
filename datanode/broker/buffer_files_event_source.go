package broker

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/libs/proto"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"
)

type bufferFileEventSource struct {
	mu                    sync.Mutex
	bufferFilesDir        string
	timeBetweenBlocks     time.Duration
	sendChannelBufferSize int
	chainID               string
	archiveFiles          []fs.FileInfo
	currentBlock          string
	cbmu                  sync.RWMutex
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

func (e *bufferFileEventSource) Receive(ctx context.Context) (<-chan []byte, <-chan error) {
	eventsCh := make(chan []byte, e.sendChannelBufferSize)
	errorCh := make(chan error, 1)

	go func() {
		for _, eventFile := range e.archiveFiles {
			err := e.sendAllRawEventsInFile(ctx, eventsCh, filepath.Join(e.bufferFilesDir, eventFile.Name()),
				e.timeBetweenBlocks)
			if err != nil {
				errorCh <- fmt.Errorf("failed to send events in buffer file: %w", err)
			}
		}
	}()

	return eventsCh, errorCh
}

func (e *bufferFileEventSource) sendAllRawEventsInFile(ctx context.Context, out chan<- []byte, file string,
	timeBetweenBlocks time.Duration,
) error {
	eventFile, err := os.Open(file)
	defer func() {
		_ = eventFile.Close()
	}()

	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}

	var offset int64

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			rawEvent, _, read, err := readRawEvent(eventFile, offset)
			if err != nil {
				return fmt.Errorf("failed to read buffered event:%w", err)
			}

			if read == 0 {
				return nil
			}

			offset += int64(read)

			// We have to deserialize the busEvent here (even though we output the raw busEvent)
			// to be able to skip the first few events before we get a BeginBlock and to be
			// able to sleep between blocks.
			busEvent := &eventspb.BusEvent{}
			if err := proto.Unmarshal(rawEvent, busEvent); err != nil {
				return fmt.Errorf("failed to unmarshal bus event: %w", err)
			}

			// Buffer files do not necessarily start on block boundaries, to prevent sending part of a block
			// events are ignored until an initial begin block event is encountered
			e.mu.Lock()
			if len(e.currentBlock) == 0 {
				if busEvent.Type == eventspb.BusEventType_BUS_EVENT_TYPE_BEGIN_BLOCK {
					e.currentBlock = busEvent.Block
				} else {
					e.mu.Unlock()
					continue
				}
			}

			// Optional sleep between blocks to mimic running against core
			if busEvent.Block != e.currentBlock {
				time.Sleep(timeBetweenBlocks)
				e.currentBlock = busEvent.Block
			}
			e.mu.Unlock()

			err = sendRawEvent(ctx, out, rawEvent)
			if err != nil {
				return fmt.Errorf("send event failed:%w", err)
			}
		}
	}
	return true
}

func sendRawEvent(ctx context.Context, out chan<- []byte, rawEvent []byte) error {
	select {
	case out <- rawEvent:
	case <-ctx.Done():
		return ctx.Err()
	}
	return nil
}
