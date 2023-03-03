package broker

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"code.vegaprotocol.io/vega/core/broker"

	"code.vegaprotocol.io/vega/datanode/utils"

	"code.vegaprotocol.io/vega/datanode/metrics"
	"code.vegaprotocol.io/vega/logging"
)

type FileBufferedEventSource struct {
	log                   *logging.Logger
	lastBufferedSeqNum    chan uint64
	sendChannelBufferSize int
	source                RawEventReceiver
	bufferFilePath        string
	archiveFilesPath      string
	config                BufferedEventSourceConfig
}

func NewBufferedEventSource(ctx context.Context, log *logging.Logger, config BufferedEventSourceConfig,
	source RawEventReceiver, bufferFilesDir string,
	archiveFilesDir string,
) (*FileBufferedEventSource, error) {
	err := os.RemoveAll(bufferFilesDir)
	if err != nil {
		return nil, fmt.Errorf("failed to remove old buffer files: %w", err)
	}

	err = os.Mkdir(bufferFilesDir, os.ModePerm)
	if err != nil {
		return nil, fmt.Errorf("failed to create buffer file directory: %w", err)
	}

	if config.Archive {
		err = os.MkdirAll(archiveFilesDir, os.ModePerm)
		if err != nil {
			return nil, fmt.Errorf("failed to create buffer file archive directory: %w", err)
		}

		go func() {
			ticker := time.NewTicker(1 * time.Minute)
			defer ticker.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					err := compressUncompressedFilesInDir(archiveFilesDir)
					if err != nil {
						log.Errorf("failed to compress uncompressed file in archive dir: %w", err)
					}

					err = removeOldArchiveFilesIfDirectoryFull(archiveFilesDir, config.ArchiveMaximumSizeBytes)
					if err != nil {
						log.Errorf("failed to remove old files from full archive directory: %w", err)
					}
				}
			}
		}()
	}

	fb := &FileBufferedEventSource{
		log:                log.Named("buffered-event-source"),
		source:             source,
		config:             config,
		lastBufferedSeqNum: make(chan uint64, config.MaxBufferedEvents),
		bufferFilePath:     bufferFilesDir,
		archiveFilesPath:   archiveFilesDir,
	}

	fb.log.Infof("Starting buffered event source with a max buffered event count of %d, and events per buffer file size %d",
		config.MaxBufferedEvents, config.EventsPerFile)

	return fb, nil
}

func (m *FileBufferedEventSource) Listen() error {
	return m.source.Listen()
}

func (m *FileBufferedEventSource) Receive(ctx context.Context) (<-chan []byte, <-chan error) {
	sourceEventCh, sourceErrCh := m.source.Receive(ctx)

	if m.config.EventsPerFile == 0 {
		m.log.Info("events per file is set to 0, disabling event buffer")
		return sourceEventCh, sourceErrCh
	}

	sinkEventCh := make(chan []byte, m.sendChannelBufferSize)
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

func (m *FileBufferedEventSource) writeEventsToBuffer(ctx context.Context, sourceEventCh <-chan []byte,
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
			err = broker.WriteRawToBufferFile(bufferFile, bufferSeqNum, event)
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

func (m *FileBufferedEventSource) readEventsFromBuffer(ctx context.Context, sinkEventCh chan []byte, sinkErrorCh chan error) {
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

			event, bufferSeqNum, read, err := readRawEvent(bufferFile, offset)
			if err != nil {
				sinkErrorCh <- fmt.Errorf("error when reading event from buffer file:%w", err)
				return
			}

			offset += int64(read)

			if event != nil {
				sinkEventCh <- event
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

func (m *FileBufferedEventSource) removeBufferFile(bufferFile *os.File) error {
	err := bufferFile.Close()
	if err != nil {
		return fmt.Errorf("failed to close last event buffer file:%w", err)
	}

	if m.config.Archive {
		err = m.moveBufferFileToArchive(bufferFile.Name())
		if err != nil {
			return fmt.Errorf("failed to move buffer file to archive: %w", err)
		}
	} else {
		err = os.Remove(bufferFile.Name())
		if err != nil {
			return fmt.Errorf("failed to remove event buffer file: %w", err)
		}
	}

	return nil
}

// moveBufferFileToArchive encodes the creation time into the archive file name to ensure that the correct order
// of files can always be determined even if the archive files are copied etc.
func (m *FileBufferedEventSource) moveBufferFileToArchive(bufferFilePath string) error {
	bufferFileName := filepath.Base(bufferFilePath)
	bufferSeqSpan := strings.ReplaceAll(bufferFileName, bufferFileNamePrepend, "")
	timeNowUtc := time.Now().UTC()

	archiveFileName := fmt.Sprintf("%s-%s-%d-seqnumspan%s", bufferFileNamePrepend,
		timeNowUtc.Format("2006-01-02-15-04-05"), timeNowUtc.UnixNano(), bufferSeqSpan)

	archiveFilePath := filepath.Join(m.archiveFilesPath, archiveFileName)

	err := os.Rename(bufferFilePath, archiveFilePath)
	if err != nil {
		return fmt.Errorf("failed to rename file: %w", err)
	}
	return nil
}

func readRawEvent(eventFile *os.File, offset int64) (event []byte, seqNum uint64,
	totalBytesRead uint32, err error,
) {
	sizeBytes := make([]byte, broker.NumberOfSizeBytes)
	read, err := eventFile.ReadAt(sizeBytes, offset)

	if err == io.EOF {
		return nil, 0, 0, nil
	} else if err != nil {
		return nil, 0, 0, fmt.Errorf("error reading message size from events file:%w", err)
	}

	if read < broker.NumberOfSizeBytes {
		return nil, 0, 0, nil
	}

	messageOffset := offset + broker.NumberOfSizeBytes

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

	seqNumBytes := seqNumAndMsgBytes[:broker.NumberOfSeqNumBytes]
	seqNum = binary.BigEndian.Uint64(seqNumBytes)
	msgBytes := seqNumAndMsgBytes[broker.NumberOfSeqNumBytes:]
	totalBytesRead = broker.NumberOfSizeBytes + msgSize

	return msgBytes, seqNum, totalBytesRead, nil
}

const bufferFileNamePrepend = "datanode-buffer"

func (m *FileBufferedEventSource) getBufferFileName(fromSeqNum uint64, toSeqNum uint64) string {
	return fmt.Sprintf("%s/%s-%d-%d.bevt", m.bufferFilePath, bufferFileNamePrepend, fromSeqNum, toSeqNum)
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

func compressUncompressedFilesInDir(dir string) error {
	files, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("failed to read dir: %w", err)
	}

	for _, file := range files {
		if !file.IsDir() {
			if !strings.HasSuffix(file.Name(), "gz") {
				err = compressBufferedEventFile(file.Name(), dir)
				if err != nil {
					return fmt.Errorf("failed to compress file: %w", err)
				}
			}
		}
	}

	return nil
}

func compressBufferedEventFile(bufferFileName string, archiveFilesDir string) error {
	bufferFilePath := filepath.Join(archiveFilesDir, bufferFileName)
	archiveFilePath := filepath.Join(archiveFilesDir, bufferFileName+".gz")

	err := utils.CompressFile(bufferFilePath, archiveFilePath)
	if err != nil {
		return fmt.Errorf("failed to compress buffer file: %w", err)
	}

	err = os.Remove(bufferFilePath)
	if err != nil {
		return fmt.Errorf("failed to remove uncompressed buffer file: %w", err)
	}

	return nil
}

// removeOldArchiveFilesIfDirectoryFull intentionally uses the name of the file to figure out the relative age
// of the file, see moveBufferFileToArchive.
func removeOldArchiveFilesIfDirectoryFull(dir string, maximumDirSizeBytes int64) error {
	var dirSizeBytes int64
	var archiveFiles []fs.FileInfo
	err := filepath.Walk(dir, func(path string, info fs.FileInfo, err error) error {
		if !info.IsDir() {
			dirSizeBytes += info.Size()
			archiveFiles = append(archiveFiles, info)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to walk directory: %w", err)
	}

	if dirSizeBytes > maximumDirSizeBytes {
		sort.Slice(archiveFiles, func(i, j int) bool {
			return strings.Compare(archiveFiles[i].Name(), archiveFiles[j].Name()) < 0
		})

		minimumBytesToRemove := dirSizeBytes - maximumDirSizeBytes

		var bytesRemoved int64
		for _, file := range archiveFiles {
			err := os.Remove(filepath.Join(dir, file.Name()))
			if err != nil {
				return fmt.Errorf("failed to remove file: %w", err)
			}
			bytesRemoved += file.Size()
			if bytesRemoved >= minimumBytesToRemove {
				break
			}
		}
	}

	return nil
}
