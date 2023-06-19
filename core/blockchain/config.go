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

package blockchain

import (
	"time"

	"code.vegaprotocol.io/vega/libs/config/encoding"
	"code.vegaprotocol.io/vega/logging"
)

const (
	ProviderNullChain  = "nullchain"
	ProviderTendermint = "tendermint"
)

// Config represent the configuration of the blockchain package.
type Config struct {
	Level               encoding.LogLevel `long:"log-level"`
	LogTimeDebug        bool              `long:"log-time-debug"`
	LogOrderSubmitDebug bool              `long:"log-order-submit-debug"`
	LogOrderAmendDebug  bool              `long:"log-order-amend-debug"`
	LogOrderCancelDebug bool              `long:"log-order-cancel-debug"`
	ChainProvider       string            `long:"chain-provider"`

	Tendermint TendermintConfig `group:"Tendermint" namespace:"tendermint"`
	Null       NullChainConfig  `group:"NullChain"  namespace:"nullchain"`
}

// NewDefaultConfig creates an instance of the package specific configuration, given a
// pointer to a logger instance to be used for logging within the package.
func NewDefaultConfig() Config {
	return Config{
		Level:               encoding.LogLevel{Level: logging.InfoLevel},
		LogOrderSubmitDebug: true,
		LogTimeDebug:        true,
		ChainProvider:       ProviderTendermint,
		Tendermint:          NewDefaultTendermintConfig(),
		Null:                NewDefaultNullChainConfig(),
	}
}

type TendermintConfig struct {
	Level   encoding.LogLevel `description:" "                             long:"log-level"`
	RPCAddr string            `description:"address of the tendermint rpc" long:"rpc-addr"`
}

// NewDefaultTendermintConfig creates an instance of the package specific configuration, given a
// pointer to a logger instance to be used for logging within the package.
func NewDefaultTendermintConfig() TendermintConfig {
	return TendermintConfig{
		Level: encoding.LogLevel{Level: logging.InfoLevel},
	}
}

type ReplayConfig struct {
	Record     bool   `description:"whether to record block data to a file to allow replaying" long:"record"`
	Replay     bool   `description:"whether to replay any blockdata found in replay-file"      long:"replay"`
	ReplayFile string `description:"path to file of which to write/read replay data"           long:"replay-file"`
}

type NullChainConfig struct {
	Level                encoding.LogLevel `long:"log-level"`
	BlockDuration        encoding.Duration `description:"(default 1s)"                           long:"block-duration"`
	TransactionsPerBlock uint64            `description:"(default 10)"                           long:"transactions-per-block"`
	GenesisFile          string            `description:"path to a tendermint genesis file"      long:"genesis-file"`
	IP                   string            `description:"time-forwarding IP (default localhost)" long:"ip"`
	Port                 int               `description:"time-forwarding port (default 3009)"    long:"port"`
	Replay               ReplayConfig
}

// NewDefaultNullChainConfig creates an instance of the package specific configuration, given a
// pointer to a logger instance to be used for logging within the package.
func NewDefaultNullChainConfig() NullChainConfig {
	return NullChainConfig{
		Level:                encoding.LogLevel{Level: logging.InfoLevel},
		BlockDuration:        encoding.Duration{Duration: time.Second},
		TransactionsPerBlock: 10,
		IP:                   "localhost",
		Port:                 3101,
		Replay:               ReplayConfig{},
	}
}
