// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

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
