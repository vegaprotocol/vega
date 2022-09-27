package snapshot

import "code.vegaprotocol.io/vega/datanode/config/encoding"

type Config struct {
	Enabled encoding.Bool `long:"set to false to prevent datanode creating snapshots" description:"set to false to prevent datanode creating snapshots"`

	RemoveSnapshotsOnStartup encoding.Bool `long:"remove-snapshops-on-startup" description:"if true the nodes snapshot directory will be emptied at startup"`

	Publish encoding.Bool `long:"publish" description:"set to true if this node should publish snapshot data"`

	DatabaseSnapshotsPath string `long:"database-snapshot-path" description:"the snapshots path relative the database working directory"`
}

// NewDefaultConfig creates an instance of the package specific configuration, given a
// pointer to a logger instance to be used for logging within the package.
func NewDefaultConfig() Config {
	return Config{
		Enabled:                  false,
		DatabaseSnapshotsPath:    "/snapshots",
		RemoveSnapshotsOnStartup: false,
		Publish:                  false,
	}
}
