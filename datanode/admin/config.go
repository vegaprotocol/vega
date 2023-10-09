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

package admin

import (
	"os"
	"path"

	"code.vegaprotocol.io/vega/libs/config/encoding"
	"code.vegaprotocol.io/vega/logging"
)

const namedLogger = "admin.server"

// ServerConfig contains the configuration for the admin server.
type ServerConfig struct {
	SocketPath string `description:"Listen for connection on UNIX socket path <file-path>" long:"socket-path"`
	HTTPPath   string `description:"Http path of the socket HTTP RPC server"               long:"http-path"`
}

// Config represents the configuration of the admin package.
type Config struct {
	Level  encoding.LogLevel `long:"log-level"`
	Server ServerConfig      `group:"Server"   namespace:"server"`
}

// NewDefaultConfig creates an instance of the package specific configuration.
func NewDefaultConfig() Config {
	return Config{
		Level: encoding.LogLevel{Level: logging.InfoLevel},
		Server: ServerConfig{
			SocketPath: path.Join(os.TempDir(), "datanode.sock"),
			HTTPPath:   "/datanode/rpc",
		},
	}
}
