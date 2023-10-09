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

package coreapi

import (
	"code.vegaprotocol.io/vega/libs/config/encoding"
	"code.vegaprotocol.io/vega/logging"
)

const (
	namedLogger = "coreapi"
)

type Config struct {
	LogLevel          encoding.LogLevel
	Accounts          bool
	Assets            bool
	NetworkParameters bool
	NetworkLimits     bool
	Parties           bool
	Validators        bool
	Proposals         bool
	Markets           bool
	Votes             bool
	MarketsData       bool
	PartiesStake      bool
	Delegations       bool
}

func NewDefaultConfig() Config {
	return Config{
		LogLevel:          encoding.LogLevel{Level: logging.InfoLevel},
		Accounts:          true,
		Assets:            true,
		NetworkParameters: true,
		NetworkLimits:     true,
		Parties:           true,
		Validators:        true,
		Markets:           true,
		Proposals:         true,
		Votes:             true,
		MarketsData:       true,
		PartiesStake:      true,
		Delegations:       true,
	}
}
