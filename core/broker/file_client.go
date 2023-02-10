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
	vgproto "code.vegaprotocol.io/vega/libs/proto"
	"code.vegaprotocol.io/vega/logging"
)

type FileClient struct {
	log *logging.Logger

	config *FileConfig

	file *os.File
	mut  sync.RWMutex
}

const namedFileClientLogger = "file-client"

func NewFileClient(log *logging.Logger, config *FileConfig) (*FileClient, error) {
	fc := &FileClient{
		log:    log.Named(namedFileClientLogger),
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

	busEvent := event.StreamMessage()

	size := uint32(proto.Size(busEvent))
	sizeBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(sizeBytes, size)

	protoBytes, err := vgproto.Marshal(busEvent)
	if err != nil {
		return fmt.Errorf("failed to marshal bus event:" + busEvent.String())
	}

	allBytes := append(sizeBytes, protoBytes...) // nozero
	fc.file.Write(allBytes)
	return nil
}

func (fc *FileClient) Close() error {
	return fc.file.Close()
}
