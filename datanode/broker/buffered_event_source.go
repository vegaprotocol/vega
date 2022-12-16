package broker

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/datanode/metrics"
	"code.vegaprotocol.io/vega/logging"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"

	"github.com/golang/protobuf/proto"
)

type FileBufferedEventSource struct {
	log                   *logging.Logger
	lastBufferedSeqNum    chan uint64
	sendChannelBufferSize int
	source                EventReceiver
	bufferFilePath        string
	config                BufferedEventSourceConfig
}

const (
	numberOfSeqNumBytes = 8
	numberOfSizeBytes   = 4
)

func NewBufferedEventSource(log *logging.Logger, config BufferedEventSourceConfig, source EventReceiver, bufferFilePath string) (*FileBufferedEventSource, error) {
	err := os.RemoveAll(bufferFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to remove old buffer files:%w", err)
	}

	err = os.Mkdir(bufferFilePath, os.ModePerm)
	if err != nil {
		return nil, fmt.Errorf("failed to create buffer file directory:%w", err)
	}

	files, _ := ioutil.ReadDir(bufferFilePath)
	for _, file := range files {
		os.Remove(filepath.Join(bufferFilePath, file.Name()))
	}

	fb := &FileBufferedEventSource{
		log:                log.Named("buffered-event-source"),
		source:             source,
		config:             config,
		lastBufferedSeqNum: make(chan uint64, config.MaxBufferedEvents),
		bufferFilePath:     bufferFilePath,
	}

	fb.log.Infof("Starting buffered event source with a max buffered event count of %d, and events per buffer file size %d",
		config.MaxBufferedEvents, config.EventsPerFile)

	return fb, nil
}

func (m *FileBufferedEventSource) Listen() error {
	return m.source.Listen()
}

func (m *FileBufferedEventSource) Receive(ctx context.Context) (<-chan events.Event, <-chan error) {
	sourceEventCh, sourceErrCh := m.source.Receive(ctx)

	if m.config.EventsPerFile == 0 {
		m.log.Info("events per file is set to 0, disabling event buffer")
		return sourceEventCh, sourceErrCh
	}

	sinkEventCh := make(chan events.Event, m.sendChannelBufferSize)
	sinkErrorCh := make(chan error, 1)

	ctxWithCancel, cancel := context.WithCancel(ctx)
	go func() {
		m.writeEventsToBuffer(ctx, sourceEventCh, sourceErrCh, sinkErrorCh)
		cancel()
	}()

	go func() {
		m.readEventsFromBuffer(ctxWithCancel, sinkEventCh, sinkErrorCh)
	}()

	return sinkEventCh, sinkErrorCh
}

func (m *FileBufferedEventSource) writeEventsToBuffer(ctx context.Context, sourceEventCh <-chan events.Event,
	sourceErrCh <-chan error, sinkErrorCh chan error,
) {
	bufferSeqNum := uint64(0)

	var bufferFile *os.File
	defer func() {
		if bufferFile != nil {
			err := bufferFile.Close()
			if err != nil {
				m.log.Errorf("failed to close event buffer file:%w", err)
			}
		}
	}()

	var err error
	for {
		select {
		case event, ok := <-sourceEventCh:
			if !ok {
				return
			}
			if bufferSeqNum%uint64(m.config.EventsPerFile) == 0 {
				bufferFile, err = m.rollBufferFile(bufferFile, bufferSeqNum)
				if err != nil {
					sinkErrorCh <- fmt.Errorf("failed to roll buffer file:%w", err)
				}
			}

			bufferSeqNum++
			err = writeToBuffer(bufferFile, bufferSeqNum, event)
			metrics.EventBufferWrittenCountInc()

			if err != nil {
				sinkErrorCh <- fmt.Errorf("failed to write events to buffer:%w", err)
			}

			select {
			case m.lastBufferedSeqNum <- bufferSeqNum:
			default:
			loop:
				for {
					select {
					case <-m.lastBufferedSeqNum:
					case <-ctx.Done():
						return
					default:
						break loop
					}
				}
				m.lastBufferedSeqNum <- bufferSeqNum
			}

		case srcErr, ok := <-sourceErrCh:
			if !ok {
				return
			}
			sinkErrorCh <- srcErr
		case <-ctx.Done():
			return
		}
	}
}

func (m *FileBufferedEventSource) readEventsFromBuffer(ctx context.Context, sinkEventCh chan events.Event, sinkErrorCh chan error) {
	var offset int64
	var lastBufferSeqNum uint64
	var lastSentBufferSeqNum uint64
	var err error

	var bufferFile *os.File

	defer func() {
		if bufferFile != nil {
			err = bufferFile.Close()
			if err != nil {
				m.log.Errorf("failed to close event buffer file:%w", err)
			}
		}
		close(sinkEventCh)
		close(sinkErrorCh)
	}()

	for {
		if ctx.Err() != nil {
			return
		}

		if lastBufferSeqNum > lastSentBufferSeqNum {
			if bufferFile == nil {
				offset = 0
				bufferFile, err = m.openBufferFile(lastSentBufferSeqNum+1, lastSentBufferSeqNum+uint64(m.config.EventsPerFile))
				if err != nil {
					sinkErrorCh <- fmt.Errorf("failed to open buffer file:%w", err)
					return
				}
			}

			event, bufferSeqNum, read, err := readEvent(bufferFile, offset)
			if err != nil {
				sinkErrorCh <- fmt.Errorf("error when reading event from buffer file:%w", err)
				return
			}

			offset += int64(read)

			if event != nil {
				evt := toEvent(ctx, event)
				sinkEventCh <- evt
				metrics.EventBufferReadCountInc()
				lastSentBufferSeqNum = bufferSeqNum

				if lastSentBufferSeqNum%uint64(m.config.EventsPerFile) == 0 {
					if err = m.removeBufferFile(bufferFile); err != nil {
						sinkErrorCh <- fmt.Errorf("failed to remove buffer file:%w", err)
					}
					bufferFile = nil
				}
			} else {
				// Time for the buffer write to complete if we were unable to read a complete event for a given seq num
				time.Sleep(10 * time.Millisecond)
			}
		} else {
			// Wait until told there is new data written to the buffer
			select {
			case <-ctx.Done():
				return
			case bufferSeqNum := <-m.lastBufferedSeqNum:
				lastBufferSeqNum = bufferSeqNum
			}
		}
	}
}

func (m *FileBufferedEventSource) rollBufferFile(currentBufferFile *os.File, seqNum uint64) (*os.File, error) {
	if currentBufferFile != nil {
		err := currentBufferFile.Close()
		if err != nil {
			return nil, fmt.Errorf("unable to create new buffer file, failed to close current events buffer file:%w", err)
		}
	}

	newBufferFile, err := m.createFile(seqNum+1, seqNum+uint64(m.config.EventsPerFile))
	if err != nil {
		return nil, fmt.Errorf("failed to create events buffer file:%w", err)
	}
	return newBufferFile, nil
}

func writeToBuffer(bufferFile *os.File, bufferSeqNum uint64, event events.Event) error {
	e := event.StreamMessage()

	seqNumBytes := make([]byte, numberOfSeqNumBytes)
	sizeBytes := make([]byte, numberOfSizeBytes)

	size := numberOfSeqNumBytes + uint32(proto.Size(e))
	protoBytes, err := proto.Marshal(e)
	if err != nil {
		return fmt.Errorf("failed to marshal bus event:%w", err)
	}

	binary.BigEndian.PutUint64(seqNumBytes, bufferSeqNum)
	binary.BigEndian.PutUint32(sizeBytes, size)
	allBytes := append([]byte{}, sizeBytes...)
	allBytes = append(allBytes, seqNumBytes...)
	allBytes = append(allBytes, protoBytes...)
	_, err = bufferFile.Write(allBytes)
	if err != nil {
		return fmt.Errorf("failed to write to buffer file:%w", err)
	}

	return nil
}

func (m *FileBufferedEventSource) removeBufferFile(bufferFile *os.File) error {
	err := bufferFile.Close()
	if err != nil {
		return fmt.Errorf("failed to close last event buffer file:%w", err)
	}

	err = os.Remove(bufferFile.Name())
	if err != nil {
		return fmt.Errorf("failed to remove event buffer file:%w", err)
	}
	return nil
}

func readEvent(eventFile *os.File, offset int64) (event *eventspb.BusEvent, seqNum uint64,
	totalBytesRead uint32, err error,
) {
	sizeBytes := make([]byte, numberOfSizeBytes)
	read, err := eventFile.ReadAt(sizeBytes, offset)

	if err == io.EOF {
		return nil, 0, 0, nil
	} else if err != nil {
		return nil, 0, 0, fmt.Errorf("error reading message size from events file:%w", err)
	}

	if read < numberOfSizeBytes {
		return nil, 0, 0, nil
	}

	messageOffset := offset + numberOfSizeBytes

	msgSize := binary.BigEndian.Uint32(sizeBytes)
	seqNumAndMsgBytes := make([]byte, msgSize)
	read, err = eventFile.ReadAt(seqNumAndMsgBytes, messageOffset)
	if err == io.EOF {
		return nil, 0, 0, nil
	} else if err != nil {
		return nil, 0, 0, fmt.Errorf("error reading message bytes from events file:%w", err)
	}

	if read < int(msgSize) {
		return nil, 0, 0, nil
	}

	seqNumBytes := seqNumAndMsgBytes[:numberOfSeqNumBytes]
	seqNum = binary.BigEndian.Uint64(seqNumBytes)

	event = &eventspb.BusEvent{}
	msgBytes := seqNumAndMsgBytes[numberOfSeqNumBytes:]
	err = proto.Unmarshal(msgBytes, event)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("failed to unmarshal bus event: %w", err)
	}
	totalBytesRead = numberOfSizeBytes + msgSize

	return event, seqNum, totalBytesRead, nil
}

func (m *FileBufferedEventSource) getBufferFileName(fromSeqNum uint64, toSeqNum uint64) string {
	return fmt.Sprintf("%s/datanode-buffer-%d-%d", m.bufferFilePath, fromSeqNum, toSeqNum)
}

func (m *FileBufferedEventSource) createFile(fromSeqNum uint64, toSeqNum uint64) (*os.File, error) {
	bufferFileName := m.getBufferFileName(fromSeqNum, toSeqNum)
	bufferFile, err := os.Create(bufferFileName)
	if err != nil {
		return nil, fmt.Errorf("failed to create buffer file: %s :%w", bufferFileName, err)
	}
	return bufferFile, err
}

func (m *FileBufferedEventSource) openBufferFile(fromSeqNum uint64, toSeqNum uint64) (*os.File, error) {
	bufferFileName := m.getBufferFileName(fromSeqNum, toSeqNum)
	bufferFile, err := os.Open(bufferFileName)
	if err != nil {
		return nil, fmt.Errorf("failed to open buffer file: %s :%w", bufferFileName, err)
	}
	return bufferFile, nil
}
