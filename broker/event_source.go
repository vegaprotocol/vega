package broker

import (
	"fmt"
	"path/filepath"

	"code.vegaprotocol.io/data-node/logging"
)

func NewEventSource(config Config, log *logging.Logger) (eventSource, error) {
	var eventsource eventSource
	var err error
	if config.UseEventFile {

		absPath, err := filepath.Abs(config.FileEventSourceConfig.File)
		if err != nil {
			return nil, fmt.Errorf("unable to determine absolute path of file %s: %w", config.FileEventSourceConfig.File, err)
		}

		log.Infof("using file event source, event file: %s", absPath)
		eventsource, err = NewFileEventSource(absPath, config.FileEventSourceConfig.TimeBetweenBlocks.Duration,
			config.FileEventSourceConfig.SendChannelBufferSize)

		if err != nil {
			return nil, fmt.Errorf("failed to create file event source:%w", err)
		}

	} else {
		eventsource, err = newSocketServer(log, &config)
		if err != nil {
			return nil, fmt.Errorf("failed to initialise underlying socket receiver: %w", err)
		}
	}
	return eventsource, nil
}
