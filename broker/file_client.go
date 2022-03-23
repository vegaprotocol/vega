package broker

import (
	"context"
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"code.vegaprotocol.io/vega/events"
	vgproto "code.vegaprotocol.io/vega/libs/proto"
	"code.vegaprotocol.io/vega/logging"
	"google.golang.org/protobuf/proto"
)

type fileClient struct {
	log *logging.Logger

	config *FileConfig

	file *os.File
	mut  sync.RWMutex
}

func newFileClient(ctx context.Context, log *logging.Logger, config *FileConfig) (*fileClient, error) {
	fc := &fileClient{
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

func (fc *fileClient) SendBatch(evts []events.Event) error {
	for _, evt := range evts {
		if err := fc.Send(evt); err != nil {
			return err
		}
	}
	return nil
}

func (fc *fileClient) Send(event events.Event) error {
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
