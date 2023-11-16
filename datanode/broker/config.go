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
	"time"

	"code.vegaprotocol.io/vega/datanode/config/encoding"
	"code.vegaprotocol.io/vega/logging"
)

const namedLogger = "broker"

// Config represents the configuration of the broker.
type Config struct {
	Level                          encoding.LogLevel         `long:"log-level"`
	SocketConfig                   SocketConfig              `group:"Socket"                                                                           namespace:"socket"`
	SocketServerInboundBufferSize  int                       `long:"socket-server-inbound-buffer-size"`
	SocketServerOutboundBufferSize int                       `long:"socket-server-outbound-buffer-size"`
	FileEventSourceConfig          FileEventSourceConfig     `group:"FileEventSourceConfig"                                                            namespace:"fileeventsource"`
	UseEventFile                   encoding.Bool             `description:"set to true to source events from a file"                                   long:"use-event-file"`
	PanicOnError                   encoding.Bool             `description:"if an error occurs on event push the broker will panic, else log the error" long:"panic-on-error"`
	UseBufferedEventSource         encoding.Bool             `description:"if true datanode will buffer events"                                        long:"use-buffered-event-source"`
	BufferedEventSourceConfig      BufferedEventSourceConfig `group:"BufferedEventSource"                                                              namespace:"bufferedeventsource"`
	EventBusClientBufferSize       int                       `long:"event-bus-client-buffer-size"`
}

// NewDefaultConfig creates an instance of config with default values.
func NewDefaultConfig() Config {
	return Config{
		Level: encoding.LogLevel{Level: logging.InfoLevel},
		SocketConfig: SocketConfig{
			IP:                 "0.0.0.0",
			Port:               3005,
			MaxReceiveTimeouts: 3,
			TransportType:      "tcp",
		},
		SocketServerInboundBufferSize:  10000,
		SocketServerOutboundBufferSize: 10000,
		FileEventSourceConfig: FileEventSourceConfig{
			Directory:             "events",
			TimeBetweenBlocks:     encoding.Duration{Duration: 1 * time.Second},
			SendChannelBufferSize: 1000,
		},
		UseEventFile:           false,
		PanicOnError:           false,
		UseBufferedEventSource: true,
		BufferedEventSourceConfig: BufferedEventSourceConfig{
			EventsPerFile:           10_000_000,
			SendChannelBufferSize:   10_000,
			Archive:                 true,
			ArchiveMaximumSizeBytes: 1_000_000_000,
		},
		EventBusClientBufferSize: 100000,
	}
}

type FileEventSourceConfig struct {
	Directory             string            `description:"the directory container the event files"               long:"directory"`
	TimeBetweenBlocks     encoding.Duration `description:"the time between sending blocks"                       string:"time-between-blocks"`
	SendChannelBufferSize int               `description:"size of channel buffer used to send events to broker " long:"send-buffer-size"`
}

type SocketConfig struct {
	IP                 string `description:" "             long:"ip"`
	Port               int    `description:" "             long:"port"`
	MaxReceiveTimeouts int    `long:"max-receive-timeouts"`
	TransportType      string `long:"transport-type"`
}

type BufferedEventSourceConfig struct {
	EventsPerFile           int   `description:"the number of events to store in a file buffer, set to 0 to disable the buffer" long:"events-per-file"`
	SendChannelBufferSize   int   `description:"sink event channel buffer size"                                                 long:"send-buffer-size"`
	Archive                 bool  `description:"archives event buffer files after they have been read, default false"           long:"archive"`
	ArchiveMaximumSizeBytes int64 `description:"the maximum size of the archive directory"                                      long:"archive-maximum-size"`
}
