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

package notary

import (
	"code.vegaprotocol.io/data-node/config/encoding"
	"code.vegaprotocol.io/data-node/logging"
)

const (
	namedLogger = "notary"
)

// Config represents governance specific configuration
type Config struct {
	// logging level
	Level encoding.LogLevel `long:"log-level"`
}

// NewDefaultConfig creates an instance of the package specific configuration.
func NewDefaultConfig() Config {
	return Config{
		Level: encoding.LogLevel{Level: logging.InfoLevel},
	}
}
