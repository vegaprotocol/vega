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

package evtforward

import (
	"time"

	"code.vegaprotocol.io/vega/core/datasource/external/ethcall"
	"code.vegaprotocol.io/vega/core/evtforward/ethereum"
	"code.vegaprotocol.io/vega/libs/config/encoding"
	"code.vegaprotocol.io/vega/logging"
)

const (
	forwarderLogger = "forwarder"
	// how often the Event Forwarder needs to select a node to send the event
	// if nothing was received.
	defaultRetryRate = 10 * time.Second

	DefaultKeepHashesDuration = 24 * 2 * time.Hour
)

// Config represents governance specific configuration.
type Config struct {
	// Level specifies the logging level of the Event Forwarder engine.
	Level                                    encoding.LogLevel `long:"log-level"`
	RetryRate                                encoding.Duration `long:"retry-rate"`
	KeepHashesDurationForTestOnlyDoNotChange encoding.Duration
	// a list of allowlisted blockchain queue public keys
	BlockchainQueueAllowlist []string `description:" " long:"blockchain-queue-allowlist"`
	// Ethereum groups the configuration related to Ethereum implementation of
	// the Event Forwarder.
	Ethereum ethereum.Config `group:"Ethereum" namespace:"ethereum"`
	EthCall  ethcall.Config  `group:"EthCall"  namespace:"ethcall"`
}

// NewDefaultConfig creates an instance of the package specific configuration.
func NewDefaultConfig() Config {
	return Config{
		Level:     encoding.LogLevel{Level: logging.InfoLevel},
		RetryRate: encoding.Duration{Duration: defaultRetryRate},
		KeepHashesDurationForTestOnlyDoNotChange: encoding.Duration{
			Duration: DefaultKeepHashesDuration,
		},
		BlockchainQueueAllowlist: []string{},
		Ethereum:                 ethereum.NewDefaultConfig(),
		EthCall:                  ethcall.NewDefaultConfig(),
	}
}

// NewDefaultSecondaryConfig creates an instance of the package specific configuration.
func NewDefaultSecondaryConfig() Config {
	const maxEthereumBlocks = 499

	cfg := Config{
		Level:     encoding.LogLevel{Level: logging.InfoLevel},
		RetryRate: encoding.Duration{Duration: defaultRetryRate},
		KeepHashesDurationForTestOnlyDoNotChange: encoding.Duration{
			Duration: DefaultKeepHashesDuration,
		},
		BlockchainQueueAllowlist: []string{},
		Ethereum:                 ethereum.NewDefaultConfig(),
		EthCall:                  ethcall.NewDefaultConfig(),
	}

	cfg.Ethereum.MaxEthereumBlocks = maxEthereumBlocks

	return cfg
}
