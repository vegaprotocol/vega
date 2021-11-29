package blockchain

import (
	"time"

	"code.vegaprotocol.io/vega/config/encoding"
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
	Null       NullChainConfig  `group:"NullChain" namespace:"nullchain"`
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
	Level          encoding.LogLevel `long:"log-level" description:" "`
	LogTimeDebug   encoding.Bool     `long:"log-level-time-debug" description:" "`
	ClientAddr     string            `long:"client-addr" description:" "`
	ClientEndpoint string            `long:"client-endpoint" description:" "`
	ServerPort     int               `long:"server-port" description:" "`
	ServerAddr     string            `long:"server-addr" description:" "`
	ABCIRecordDir  string            `long:"abci-record-dir" description:"ABCI recording directory. If set, it will record ABCI operations into <dir>/abci-record-<now()>."`
	ABCIReplayFile string            `long:"abci-replay-file" description:"ABCI replaying file. If set, it will replay ABCI operations from this file."`
}

// NewDefaultTendermintConfig creates an instance of the package specific configuration, given a
// pointer to a logger instance to be used for logging within the package.
func NewDefaultTendermintConfig() TendermintConfig {
	return TendermintConfig{
		Level:          encoding.LogLevel{Level: logging.InfoLevel},
		ServerPort:     26658,
		ServerAddr:     "localhost",
		ClientAddr:     "tcp://0.0.0.0:26657",
		ClientEndpoint: "/websocket",
		LogTimeDebug:   true,

		// Both empty mean that neither record or replay will be activated
		ABCIRecordDir:  "",
		ABCIReplayFile: "",
	}
}

type NullChainConfig struct {
	Level                encoding.LogLevel `long:"log-level"`
	BlockDuration        encoding.Duration `long:"block-duration" description:"(default 1s)"`
	TransactionsPerBlock uint64            `long:"transactions-per-block" description:"(default 10)"`
	GenesisFile          string            `long:"genesis-file" description:"path to a tendermint genesis file"`
	IP                   string            `long:"ip" description:"time-forwarding IP (default localhost)"`
	Port                 int               `long:"port" description:"time-forwarding port (default 3009)"`
}

// NewDefaultNullChainConfig creates an instance of the package specific configuration, given a
// pointer to a logger instance to be used for logging within the package.
func NewDefaultNullChainConfig() NullChainConfig {
	return NullChainConfig{
		Level:                encoding.LogLevel{Level: logging.InfoLevel},
		BlockDuration:        encoding.Duration{Duration: time.Second},
		TransactionsPerBlock: 10,
		IP:                   "localhost",
		Port:                 3009,
	}
}
