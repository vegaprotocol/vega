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
