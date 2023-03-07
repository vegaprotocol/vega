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
	"time"

	"code.vegaprotocol.io/vega/datanode/config/encoding"
	"code.vegaprotocol.io/vega/logging"
)

const namedLogger = "broker"

// Config represents the configuration of the broker.
type Config struct {
	Level                          encoding.LogLevel         `long:"log-level"`
	SocketConfig                   SocketConfig              `group:"Socket" namespace:"socket"`
	SocketServerInboundBufferSize  int                       `long:"socket-server-inbound-buffer-size"`
	SocketServerOutboundBufferSize int                       `long:"socket-server-outbound-buffer-size"`
	FileEventSourceConfig          FileEventSourceConfig     `group:"FileEventSourceConfig" namespace:"fileeventsource"`
	UseEventFile                   encoding.Bool             `long:"use-event-file" description:"set to true to source events from a file"`
	PanicOnError                   encoding.Bool             `long:"panic-on-error" description:"if an error occurs on event push the broker will panic, else log the error"`
	UseBufferedEventSource         encoding.Bool             `long:"use-buffered-event-source" description:"if true datanode will buffer events"`
	BufferedEventSourceConfig      BufferedEventSourceConfig `group:"BufferedEventSource" namespace:"bufferedeventsource"`
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
			MaxBufferedEvents:       100_000_000,
			Archive:                 true,
			ArchiveMaximumSizeBytes: 10_000_000_000,
		},
		EventBusClientBufferSize: 100000,
	}
}

type FileEventSourceConfig struct {
	Directory             string            `long:"directory" description:"the directory container the event files"`
	TimeBetweenBlocks     encoding.Duration `string:"time-between-blocks" description:"the time between sending blocks"`
	SendChannelBufferSize int               `long:"send-buffer-size" description:"size of channel buffer used to send events to broker "`
}

type SocketConfig struct {
	IP                 string `long:"ip" description:" "`
	Port               int    `long:"port" description:" "`
	MaxReceiveTimeouts int    `long:"max-receive-timeouts"`
	TransportType      string `long:"transport-type"`
}

type BufferedEventSourceConfig struct {
	EventsPerFile           int   `long:"events-per-file" description:"the number of events to store in a file buffer, set to 0 to disable the buffer"`
	SendChannelBufferSize   int   `long:"send-buffer-size" description:"sink event channel buffer size"`
	MaxBufferedEvents       int   `long:"max-buffered-events" description:"max number of events that can be buffered, after this point events will no longer be buffered"`
	Archive                 bool  `long:"archive" description:"archives event buffer files after they have been read, default false"`
	ArchiveMaximumSizeBytes int64 `long:"archive-maximum-size" description:"the maximum size of the archive directory"`
}
