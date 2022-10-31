package dehistory

import (
	"time"

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

	AddSnapshotsToStore  encoding.Bool     `long:"add-snapshots-to-store" description:"if true snapshot data produced by this node will be added to the decentralise history store"`
	AddSnapshotsInterval encoding.Duration `long:"add-snapshots-interval" description:"interval between checking for and adding snapshot data to the decentralised store"`

	Store    store.Config    `group:"Store" namespace:"store"`
	Snapshot snapshot.Config `group:"Snapshot" namespace:"snapshot"`

	Initialise initialise.Config `group:"Initialise" namespace:"initialise"`
}

// NewDefaultConfig creates an instance of the package specific configuration, given a
// pointer to a logger instance to be used for logging within the package.
func NewDefaultConfig() Config {
	return Config{
		Level:                encoding.LogLevel{Level: logging.InfoLevel},
		Enabled:              true,
		WipeOnStartup:        true,
		AddSnapshotsToStore:  true,
		AddSnapshotsInterval: encoding.Duration{Duration: 5 * time.Second},
		Store:                store.NewDefaultConfig(),
		Snapshot:             snapshot.NewDefaultConfig(),
		Initialise:           initialise.NewDefaultConfig(),
	}
}
