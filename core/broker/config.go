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
			DialTimeout:             encoding.Duration{Duration: 96 * time.Hour},
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
	DialTimeout       encoding.Duration `description:" " long:"dial-timeout"`
	DialRetryInterval encoding.Duration `description:" " long:"dial-retry-interval"`

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
