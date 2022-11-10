package snapshot

import (
	"time"

	"code.vegaprotocol.io/vega/datanode/config/encoding"
)

type Config struct {
	WaitForCreationLockTimeout encoding.Duration `long:"wait-for-creation-lock-timeout" description:"the maximum a caller to create snapshot should have to wait to acquire the creation lock"`
}

// NewDefaultConfig creates an instance of the package specific configuration, given a
// pointer to a logger instance to be used for logging within the package.
func NewDefaultConfig() Config {
	return Config{
		WaitForCreationLockTimeout: encoding.Duration{Duration: 5 * time.Minute},
	}
}
