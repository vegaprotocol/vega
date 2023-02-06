// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.VEGA file and at https://www.mariadb.com/bsl11.
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

	"code.vegaprotocol.io/vega/libs/config/encoding"
	"code.vegaprotocol.io/vega/logging"
)

const namedLogger = "broker"

// Config represents the configuration of the broker.
type Config struct {
	Level                    encoding.LogLevel `long:"log-level"`
	Socket                   SocketConfig      `group:"Socket" namespace:"socket"`
	File                     FileConfig        `group:"File" namespace:"file"`
	EventBusClientBufferSize int               `long:"event-bus-client-buffer-size"`
}

// NewDefaultConfig creates an instance of config with default values.
func NewDefaultConfig() Config {
	return Config{
		Level: encoding.LogLevel{Level: logging.InfoLevel},
		Socket: SocketConfig{
			DialTimeout:             encoding.Duration{Duration: 2 * time.Minute},
			DialRetryInterval:       encoding.Duration{Duration: 5 * time.Second},
			SocketQueueTimeout:      encoding.Duration{Duration: 3 * time.Second},
			MaxSendTimeouts:         10,
			EventChannelBufferSize:  10000000,
			SocketChannelBufferSize: 1000000,
			Address:                 "127.0.0.1",
			Port:                    3005,
			Transport:               "tcp",
			Enabled:                 false,
		},
		File: FileConfig{
			Enabled: false,
		},
		EventBusClientBufferSize: 100000,
	}
}

type SocketConfig struct {
	DialTimeout        encoding.Duration `long:"dial-timeout" description:" "`
	DialRetryInterval  encoding.Duration `long:"dial-retry-interval" description:" "`
	SocketQueueTimeout encoding.Duration `long:"socket-queue-timeout" description:" "`

	EventChannelBufferSize  int `long:"event-channel-buffer-size" description:" "`
	SocketChannelBufferSize int `long:"socket-channel-buffer-size" description:" "`

	MaxSendTimeouts int `long:"max-send-timeouts" description:" "`

	Address   string        `long:"address" description:"Data node's address"`
	Port      int           `long:"port" description:"Data node port"`
	Enabled   encoding.Bool `long:"enabled" description:"Enable streaming of bus events over socket"`
	Transport string        `long:"transport" description:"Transport of socket. tcp/inproc are allowed. Default is TCP"`
}

type FileConfig struct {
	Enabled encoding.Bool `long:"enabled" description:"Enable streaming of bus events to a file"`
	File    string        `long:"file" description:"Path of a file to write event log to"`
}
