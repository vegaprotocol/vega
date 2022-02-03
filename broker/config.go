package broker

import (
	"code.vegaprotocol.io/data-node/config/encoding"
	"code.vegaprotocol.io/data-node/logging"
)

const namedLogger = "broker"

// Config represents the configuration of the broker.
type Config struct {
	Level           encoding.LogLevel `long:"log-level"`
	SocketConfig    SocketConfig      `group:"Socket" namespace:"socket"`
	FileEventSource FileEventSource   `group:"FileEventSource" namespace:"fileeventsource"`
	UseEventFile    encoding.Bool     `long:"use-event-file" description:"set to true to source events from a file"`
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
		FileEventSource: FileEventSource{
			File:                  "vega.evt",
			TimeBetweenBlocks:     1000,
			SendChannelBufferSize: 1000,
		},
		UseEventFile: false,
	}
}

type FileEventSource struct {
	File                  string
	TimeBetweenBlocks     int `description:"the time between sending blocks in milliseconds "`
	SendChannelBufferSize int `long:"send-buffer-size" description:" "`
}

type SocketConfig struct {
	IP                 string `long:"ip" description:" "`
	Port               int    `long:"port" description:" "`
	MaxReceiveTimeouts int    `long:"max-Receive-timeouts"`
	TransportType      string `long:"transport-type"`
}
