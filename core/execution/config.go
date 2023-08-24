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

package execution

import (
	"code.vegaprotocol.io/vega/core/fee"
	"code.vegaprotocol.io/vega/core/liquidity/v2"
	"code.vegaprotocol.io/vega/core/matching"
	"code.vegaprotocol.io/vega/core/positions"
	"code.vegaprotocol.io/vega/core/risk"
	"code.vegaprotocol.io/vega/core/settlement"
	"code.vegaprotocol.io/vega/libs/config/encoding"
	"code.vegaprotocol.io/vega/logging"
)

const (
	// namedLogger is the identifier for package and should ideally match the package name
	// this is simply emitted as a hierarchical label e.g. 'api.grpc'.
	namedLogger = "execution"
)

// Config is the configuration of the execution package.
type Config struct {
	Level encoding.LogLevel `long:"log-level"`

	Matching   matching.Config   `group:"Matching"    namespace:"matching"`
	Risk       risk.Config       `group:"Risk"        namespace:"risk"`
	Position   positions.Config  `group:"Position"    namespace:"position"`
	Settlement settlement.Config `group:"Settlement"  namespace:"settlement"`
	Fee        fee.Config        `group:"Fee"         namespace:"fee"`
	Liquidity  liquidity.Config  `group:"LiquidityV2" namespace:"liquidityV2"`
}

// NewDefaultConfig creates an instance of the package specific configuration, given a
// pointer to a logger instance to be used for logging within the package.
func NewDefaultConfig() Config {
	c := Config{
		Level:      encoding.LogLevel{Level: logging.InfoLevel},
		Matching:   matching.NewDefaultConfig(),
		Risk:       risk.NewDefaultConfig(),
		Position:   positions.NewDefaultConfig(),
		Settlement: settlement.NewDefaultConfig(),
		Fee:        fee.NewDefaultConfig(),
		Liquidity:  liquidity.NewDefaultConfig(),
	}
	return c
}
