package dehistory

import (
	"code.vegaprotocol.io/vega/datanode/config/encoding"
	"code.vegaprotocol.io/vega/datanode/dehistory/initialise"
	"code.vegaprotocol.io/vega/datanode/dehistory/snapshot"
	"code.vegaprotocol.io/vega/datanode/dehistory/store"
	"code.vegaprotocol.io/vega/logging"
)

type Config struct {
	Level         encoding.LogLevel `long:"log-level"`
	Enabled       encoding.Bool     `long:"enabled" description:"set to false to disable decentralized history"`
	WipeOnStartup encoding.Bool     `long:"wipe-on-startup" description:"remove all deHistory state on startup"`

	Publish encoding.Bool `long:"publish" description:"if true this node will create and publish decentralized history segments"`

	Store    store.Config    `group:"Store" namespace:"store"`
	Snapshot snapshot.Config `group:"Snapshot" namespace:"snapshot"`

	Initialise initialise.Config `group:"Initialise" namespace:"initialise"`
}

// NewDefaultConfig creates an instance of the package specific configuration, given a
// pointer to a logger instance to be used for logging within the package.
func NewDefaultConfig() Config {
	return Config{
		Level:         encoding.LogLevel{Level: logging.InfoLevel},
		Enabled:       true,
		WipeOnStartup: true,
		Publish:       true,
		Store:         store.NewDefaultConfig(),
		Snapshot:      snapshot.NewDefaultConfig(),
		Initialise:    initialise.NewDefaultConfig(),
	}
}
