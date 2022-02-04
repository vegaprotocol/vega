package broker

import (
	"time"

	"code.vegaprotocol.io/data-node/config/encoding"
	"code.vegaprotocol.io/data-node/logging"
)

const namedLogger = "broker"

// Config represents the configuration of the broker.
type Config struct {
	Level                 encoding.LogLevel     `long:"log-level"`
	SocketConfig          SocketConfig          `group:"Socket" namespace:"socket"`
	FileEventSourceConfig FileEventSourceConfig `group:"FileEventSourceConfig" namespace:"fileeventsource"`
	UseEventFile          encoding.Bool         `long:"use-event-file" description:"set to true to source events from a file"`
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
		FileEventSourceConfig: FileEventSourceConfig{
			File:                  "vega.evt",
			TimeBetweenBlocks:     encoding.Duration{Duration: 1 * time.Second},
			SendChannelBufferSize: 1000,
		},
		UseEventFile: false,
	}
}

type FileEventSourceConfig struct {
	File                  string            `long:"file" description:"the event file"`
	TimeBetweenBlocks     encoding.Duration `string:"time-between-blocks" description:"the time between sending blocks"`
	SendChannelBufferSize int               `long:"send-buffer-size" description:"size of channel buffer used to send events to broker "`
}

type SocketConfig struct {
	IP                 string `long:"ip" description:" "`
	Port               int    `long:"port" description:" "`
	MaxReceiveTimeouts int    `long:"max-receive-timeouts"`
	TransportType      string `long:"transport-type"`
}
