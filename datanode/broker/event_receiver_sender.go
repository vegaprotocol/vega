// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

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
