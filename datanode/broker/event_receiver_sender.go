// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.DATANODE file and at https://www.mariadb.com/bsl11.
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

	"code.vegaprotocol.io/vega/logging"
)

func NewEventReceiverSender(config Config, log *logging.Logger, chainID string) (rawEventReceiverSender, error) {
	var eventsource rawEventReceiverSender
	var err error
	if config.UseEventFile {
		absPath, err := filepath.Abs(config.FileEventSourceConfig.Directory)
		if err != nil {
			return nil, fmt.Errorf("unable to determine absolute path of file %s: %w", config.FileEventSourceConfig.Directory, err)
		}

		log.Infof("using buffer files event source, event files directory: %s", absPath)
		eventsource, err = NewBufferFilesEventSource(absPath, config.FileEventSourceConfig.TimeBetweenBlocks.Duration,
			config.FileEventSourceConfig.SendChannelBufferSize, chainID)

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
