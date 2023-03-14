// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.VEGA file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package broker

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"google.golang.org/protobuf/proto"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/logging"
)

type FileClient struct {
	log *logging.Logger

	config *FileConfig

	file   *os.File
	mut    sync.RWMutex
	seqNum uint64
}

const (
	NumberOfSeqNumBytes = 8
	NumberOfSizeBytes   = 4

	namedFileClientLogger = "file-client"
)

func NewFileClient(log *logging.Logger, config *FileConfig) (*FileClient, error) {
	log = log.Named(namedFileClientLogger)
	fc := &FileClient{
		log:    log,
		config: config,
	}

	filePath, err := filepath.Abs(config.File)
	if err != nil {
		return nil, fmt.Errorf("unable to determine absolute path of file %s: %w", config.File, err)
	}

	fc.file, err = os.Create(filePath)

	if err != nil {
		return nil, fmt.Errorf("unable to create file %s: %w", filePath, err)
	}

	log.Infof("persisting events to: %s\n", filePath)

	return fc, nil
}

func (fc *FileClient) SendBatch(evts []events.Event) error {
	for _, evt := range evts {
		if err := fc.Send(evt); err != nil {
			return err
		}
	}
	return nil
}

func (fc *FileClient) Send(event events.Event) error {
	fc.mut.RLock()
	defer fc.mut.RUnlock()

	err := WriteToBufferFile(fc.file, fc.seqNum, event)
	fc.seqNum++

	if err != nil {
		return fmt.Errorf("failed to write event to buffer file: %w", err)
	}

	return nil
}

func WriteToBufferFile(bufferFile *os.File, bufferSeqNum uint64, event events.Event) error {
	rawEvent, err := proto.Marshal(event.StreamMessage())
	if err != nil {
		return fmt.Errorf("failed to marshal bus event:%w", err)
	}
	return WriteRawToBufferFile(bufferFile, bufferSeqNum, rawEvent)
}

func WriteRawToBufferFile(bufferFile *os.File, bufferSeqNum uint64, rawEvent []byte) error {
	seqNumBytes := make([]byte, NumberOfSeqNumBytes)
	sizeBytes := make([]byte, NumberOfSizeBytes)

	size := NumberOfSeqNumBytes + uint32(len(rawEvent))

	binary.BigEndian.PutUint64(seqNumBytes, bufferSeqNum)
	binary.BigEndian.PutUint32(sizeBytes, size)
	allBytes := append([]byte{}, sizeBytes...)
	allBytes = append(allBytes, seqNumBytes...)
	allBytes = append(allBytes, rawEvent...)
	_, err := bufferFile.Write(allBytes)
	if err != nil {
		return fmt.Errorf("failed to write to buffer file:%w", err)
	}

	return nil
}

func (fc *FileClient) Close() error {
	return fc.file.Close()
}
