package config

import (
	"code.vegaprotocol.io/vega/accounts"
	"code.vegaprotocol.io/vega/api"
	"code.vegaprotocol.io/vega/blockchain"
	"code.vegaprotocol.io/vega/candles"
	"code.vegaprotocol.io/vega/collateral"
	"code.vegaprotocol.io/vega/execution"
	"code.vegaprotocol.io/vega/gateway"
	"code.vegaprotocol.io/vega/governance"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/markets"
	"code.vegaprotocol.io/vega/matching"
	"code.vegaprotocol.io/vega/metrics"
	"code.vegaprotocol.io/vega/monitoring"
	"code.vegaprotocol.io/vega/orders"
	"code.vegaprotocol.io/vega/parties"
	"code.vegaprotocol.io/vega/positions"
	"code.vegaprotocol.io/vega/pprof"
	"code.vegaprotocol.io/vega/risk"
	"code.vegaprotocol.io/vega/settlement"
	"code.vegaprotocol.io/vega/storage"
	"code.vegaprotocol.io/vega/trades"
	"code.vegaprotocol.io/vega/transfers"
	"code.vegaprotocol.io/vega/vegatime"
)

// Config ties together all other application configuration types.
type Config struct {
	API        api.Config
	Accounts   accounts.Config
	Blockchain blockchain.Config
	Candles    candles.Config
	Collateral collateral.Config
	Execution  execution.Config
	Logging    logging.Config
	Matching   matching.Config
	Markets    markets.Config
	Orders     orders.Config
	Parties    parties.Config
	Position   positions.Config
	Risk       risk.Config
	Settlement settlement.Config
	Storage    storage.Config
	Trades     trades.Config
	Time       vegatime.Config
	Monitoring monitoring.Config
	Gateway    gateway.Config
	Metrics    metrics.Config
	Transfers  transfers.Config
	Governance governance.Config

	Pprof          pprof.Config
	GatewayEnabled bool
	StoresEnabled  bool
	UlimitNOFile   uint64 `tomlcp:"Set the max number of open files (see: ulimit -n)"`
}

// NewDefaultConfig returns a set of default configs for all vega packages, as specified at the per package
// config level, if there is an error initialising any of the configs then this is returned.
func NewDefaultConfig(defaultStoreDirPath string) Config {
	return Config{
		Trades:         trades.NewDefaultConfig(),
		Blockchain:     blockchain.NewDefaultConfig(),
		Execution:      execution.NewDefaultConfig(defaultStoreDirPath),
		API:            api.NewDefaultConfig(),
		Accounts:       accounts.NewDefaultConfig(),
		Orders:         orders.NewDefaultConfig(),
		Time:           vegatime.NewDefaultConfig(),
		Markets:        markets.NewDefaultConfig(),
		Matching:       matching.NewDefaultConfig(),
		Parties:        parties.NewDefaultConfig(),
		Candles:        candles.NewDefaultConfig(),
		Risk:           risk.NewDefaultConfig(),
		Storage:        storage.NewDefaultConfig(defaultStoreDirPath),
		Pprof:          pprof.NewDefaultConfig(),
		Monitoring:     monitoring.NewDefaultConfig(),
		Logging:        logging.NewDefaultConfig(),
		Gateway:        gateway.NewDefaultConfig(),
		Position:       positions.NewDefaultConfig(),
		Settlement:     settlement.NewDefaultConfig(),
		Collateral:     collateral.NewDefaultConfig(),
		Metrics:        metrics.NewDefaultConfig(),
		Transfers:      transfers.NewDefaultConfig(),
		Governance:     governance.NewDefaultConfig(),
		GatewayEnabled: true,
		StoresEnabled:  true,
		UlimitNOFile:   8192,
	}
}
