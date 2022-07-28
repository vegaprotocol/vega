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

package genesis

import (
	"code.vegaprotocol.io/vega/core/config/encoding"
	"code.vegaprotocol.io/vega/logging"
)

const namedLogger = "genesis"

type Config struct {
	Level encoding.LogLevel `long:"log-level"`
}

func NewDefaultConfig() Config {
	return Config{
		Level: encoding.LogLevel{Level: logging.InfoLevel},
	}
}
