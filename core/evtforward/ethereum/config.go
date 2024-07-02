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

package ethereum

import (
	"time"

	"code.vegaprotocol.io/vega/libs/config/encoding"
	"code.vegaprotocol.io/vega/logging"
)

const (
	defaultHeartbeatInterval       = 1 * time.Hour
	defaultDurationBetweenTwoRetry = 20 * time.Second
	maxEthereumBlocks              = 10000 // chosen because one of the validators wanted to use quicknode, and this is their limit
)

type Config struct {
	// Level specifies the logging level of the Ethereum implementation of the
	// Event Forwarder.
	Level                                   encoding.LogLevel `long:"log-level"`
	MaxEthereumBlocks                       uint64            `long:"max-ethereum-blocks"`
	PollEventRetryDuration                  encoding.Duration
	ChainID                                 string
	SkipClientVerification                  bool
	HeartbeatIntervalForTestOnlyDoNotChange encoding.Duration
}

func NewDefaultConfig() Config {
	return Config{
		Level:                                   encoding.LogLevel{Level: logging.InfoLevel},
		PollEventRetryDuration:                  encoding.Duration{Duration: defaultDurationBetweenTwoRetry},
		MaxEthereumBlocks:                       maxEthereumBlocks,
		HeartbeatIntervalForTestOnlyDoNotChange: encoding.Duration{Duration: defaultHeartbeatInterval},
	}
}

func (c *Config) setDefaults() {
	if c.MaxEthereumBlocks == 0 {
		c.MaxEthereumBlocks = maxEthereumBlocks
	}

	if c.PollEventRetryDuration.Duration == 0 {
		c.PollEventRetryDuration.Duration = defaultDurationBetweenTwoRetry
	}

	if c.HeartbeatIntervalForTestOnlyDoNotChange.Duration == 0 {
		c.HeartbeatIntervalForTestOnlyDoNotChange.Duration = defaultHeartbeatInterval
	}
}
