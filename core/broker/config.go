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
	Socket                   SocketConfig      `group:"Socket"                      namespace:"socket"`
	File                     FileConfig        `group:"File"                        namespace:"file"`
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
	DialTimeout        encoding.Duration `description:" " long:"dial-timeout"`
	DialRetryInterval  encoding.Duration `description:" " long:"dial-retry-interval"`
	SocketQueueTimeout encoding.Duration `description:" " long:"socket-queue-timeout"`

	EventChannelBufferSize  int `description:" " long:"event-channel-buffer-size"`
	SocketChannelBufferSize int `description:" " long:"socket-channel-buffer-size"`

	MaxSendTimeouts int `description:" " long:"max-send-timeouts"`

	Address   string        `description:"Data node's address"                                         long:"address"`
	Port      int           `description:"Data node port"                                              long:"port"`
	Enabled   encoding.Bool `description:"Enable streaming of bus events over socket"                  long:"enabled"`
	Transport string        `description:"Transport of socket. tcp/inproc are allowed. Default is TCP" long:"transport"`
}

type FileConfig struct {
	Enabled encoding.Bool `description:"Enable streaming of bus events to a file" long:"enabled"`
	File    string        `description:"Path of a file to write event log to"     long:"file"`
}
