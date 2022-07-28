// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package metrics

import (
	"time"

	"code.vegaprotocol.io/data-node/datanode/config/encoding"
	"code.vegaprotocol.io/data-node/logging"
)

// Config represents the configuration of the metric package
type Config struct {
	Level   encoding.LogLevel `long:"log-level" description:" "`
	Timeout encoding.Duration `long:"timeout" description:" "`
	Port    int               `long:"port" description:" "`
	Path    string            `long:"path" description:" "`
	Enabled encoding.Bool     `long:"enabled" description:" "`
}

// NewDefaultConfig creates an instance of the package specific configuration, given a
// pointer to a logger instance to be used for logging within the package.
func NewDefaultConfig() Config {
	return Config{
		Level:   encoding.LogLevel{Level: logging.InfoLevel},
		Timeout: encoding.Duration{Duration: 5000 * time.Millisecond},

		Port:    2112,
		Path:    "/metrics",
		Enabled: false,
	}
}
