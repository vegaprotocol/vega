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

package admin

import (
	"os"
	"path"

	"code.vegaprotocol.io/vega/libs/config/encoding"
	"code.vegaprotocol.io/vega/logging"
)

const (
	nvServerNamedLogger   = "admin.non-validator-server"
	vServerNamedLogger    = "admin.validator-server"
	clientNamedLogger     = "admin-client"
	nodeWalletNamedLogger = "admin.node-wallet"
)

// ServerConfig represent the configuration of the server.
type ServerConfig struct {
	SocketPath string `description:"Listen for connection on UNIX socket path <file-path>" long:"socket-path"`
	HTTPPath   string `description:"Http path of the socket HTTP RPC server"               long:"http-path"`
}

// Config represents the configuration of the admin package.
type Config struct {
	Level  encoding.LogLevel `long:"log-level"`
	Server ServerConfig      `group:"Server"   namespace:"server"`
}

// NewDefaultConfig creates an instance of the package specific configuration, given a
// pointer to a logger instance to be used for logging within the package.
func NewDefaultConfig() Config {
	return Config{
		Level: encoding.LogLevel{Level: logging.InfoLevel},
		Server: ServerConfig{
			SocketPath: path.Join(os.TempDir(), "vega.sock"),
			HTTPPath:   "/rpc",
		},
	}
}
