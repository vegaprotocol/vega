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

package service

import (
	"code.vegaprotocol.io/vega/datanode/config/encoding"
	"code.vegaprotocol.io/vega/logging"
)

type MarketDepthConfig struct {
	AmmFullExpansionPercentage float64 `description:"The percentage eitherside of the mid price at which to display acccurate AMM volume"    long:"amm-full-expansion-percentage"`
	AmmEstimatedStepPercentage float64 `description:"The size of the step as a percentage of the mid price at which we aggregate AMM volume" long:"amm-estimated-step-percentage"`
	AmmMaxEstimatedSteps       uint64  `description:"The number of estimate steps to take outside the accurate region"                       long:"amm-max-estimated-steps"`
}

// Config represent the configuration of the service package.
type Config struct {
	Level       encoding.LogLevel `long:"log-level"`
	MarketDepth MarketDepthConfig
}

// NewDefaultConfig creates an instance of the package specific configuration, given a
// pointer to a logger instance to be used for logging within the package.
func NewDefaultConfig() Config {
	return Config{
		Level: encoding.LogLevel{Level: logging.InfoLevel},
		MarketDepth: MarketDepthConfig{
			AmmFullExpansionPercentage: 0.03,
			AmmEstimatedStepPercentage: 2.5,
			AmmMaxEstimatedSteps:       3,
		},
	}
}
