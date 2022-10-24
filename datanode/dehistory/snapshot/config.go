package snapshot

import (
	"time"

	"code.vegaprotocol.io/vega/datanode/config/encoding"
)

type Config struct {
	PanicOnSnapshotCreationError encoding.Bool `long:"panic-on-snapshot-creation-error" description:""`

	DatabaseSnapshotsCopyToPath   string `long:"database-snapshot-copy-to-path" description:"the snapshots copy to path relative to the database working directory"`
	DatabaseSnapshotsCopyFromPath string `long:"database-snapshot-copy-from-path" description:"the snapshots copy from path relative to the database working directory"`

	WaitForCreationLockTimeout encoding.Duration `long:"wait-for-creation-lock-timeout" description:"the maximum a caller to create snapshot should have to wait to acquire the creation lock"`
}

// NewDefaultConfig creates an instance of the package specific configuration, given a
// pointer to a logger instance to be used for logging within the package.
func NewDefaultConfig() Config {
	return Config{
		DatabaseSnapshotsCopyToPath:   "/snapshotsCopyTo",
		DatabaseSnapshotsCopyFromPath: "/snapshotsCopyFrom",
		WaitForCreationLockTimeout:    encoding.Duration{Duration: 5 * time.Second},
		PanicOnSnapshotCreationError:  true,
	}
}
