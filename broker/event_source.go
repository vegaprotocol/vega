// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

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
