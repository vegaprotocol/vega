package broker

import (
	"time"

	"code.vegaprotocol.io/data-node/config/encoding"
	"code.vegaprotocol.io/data-node/logging"
)

const namedLogger = "broker"

// Config represents the configuration of the broker.
type Config struct {
	Level                          encoding.LogLevel     `long:"log-level"`
	SocketConfig                   SocketConfig          `group:"Socket" namespace:"socket"`
	SocketServerInboundBufferSize  int                   `long:"socket-server-inbound-buffer-size"`
	SocketServerOutboundBufferSize int                   `long:"socket-server-outbound-buffer-size"`
	FileEventSourceConfig          FileEventSourceConfig `group:"FileEventSourceConfig" namespace:"fileeventsource"`
	UseEventFile                   encoding.Bool         `long:"use-event-file" description:"set to true to source events from a file"`
	PanicOnError                   encoding.Bool         `long:"panic-on-error" description:"if an error occurs on event push the broker will panic, else log the error"`
	BlockProcessingTimeout         encoding.Duration     `long:"block-processing-timeout" description:"The maximum time permitted for a block of events to be processed"`
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
			File:                  "vega.evt",
			TimeBetweenBlocks:     encoding.Duration{Duration: 1 * time.Second},
			SendChannelBufferSize: 1000,
		},
		UseEventFile:           false,
		PanicOnError:           true,
		BlockProcessingTimeout: encoding.Duration{Duration: 30 * time.Second},
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
