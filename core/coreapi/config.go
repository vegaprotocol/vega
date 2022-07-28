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

package coreapi

import (
	"code.vegaprotocol.io/vega/core/config/encoding"
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
