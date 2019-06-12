package config

import (
	"code.vegaprotocol.io/vega/internal/api"
	"code.vegaprotocol.io/vega/internal/auth"
	"code.vegaprotocol.io/vega/internal/blockchain"
	"code.vegaprotocol.io/vega/internal/collateral"
	"code.vegaprotocol.io/vega/internal/execution"
	"code.vegaprotocol.io/vega/internal/gateway"
	"code.vegaprotocol.io/vega/internal/logging"
	"code.vegaprotocol.io/vega/internal/matching"
	"code.vegaprotocol.io/vega/internal/metrics"
	"code.vegaprotocol.io/vega/internal/monitoring"
	"code.vegaprotocol.io/vega/internal/positions"
	"code.vegaprotocol.io/vega/internal/pprof"
	"code.vegaprotocol.io/vega/internal/settlement"
	storcfg "code.vegaprotocol.io/vega/internal/storage/config"
	"code.vegaprotocol.io/vega/internal/vegatime"
)

// Config ties together all other application configuration types.
type Config struct {
	API        api.Config
	Accounts   storcfg.AccountsConfig
	Blockchain blockchain.Config
	Candles    storcfg.CandlesConfig
	Collateral collateral.Config
	Execution  execution.Config
	Logging    logging.Config
	Matching   matching.Config
	Markets    storcfg.MarketsConfig
	Orders     storcfg.OrdersConfig
	Parties    storcfg.PartiesConfig
	Position   positions.Config
	Risk       storcfg.RiskConfig
	Settlement settlement.Config
	Trades     storcfg.TradesConfig
	Time       vegatime.Config
	Monitoring monitoring.Config
	Gateway    gateway.Config
	Auth       auth.Config
	Metrics    metrics.Config

	Pprof          pprof.Config
	GatewayEnabled bool
}

// NewDefaultConfig returns a set of default configs for all vega packages, as specified at the per package
// config level, if there is an error initialising any of the configs then this is returned.
func NewDefaultConfig(defaultStoreDirPath string) Config {
	return Config{
		Trades:         storcfg.NewDefaultTradesConfig(defaultStoreDirPath),
		Blockchain:     blockchain.NewDefaultConfig(),
		Execution:      execution.NewDefaultConfig(defaultStoreDirPath),
		API:            api.NewDefaultConfig(),
		Accounts:       storcfg.NewDefaultAccountsConfig(defaultStoreDirPath),
		Orders:         storcfg.NewDefaultOrdersConfig(defaultStoreDirPath),
		Time:           vegatime.NewDefaultConfig(),
		Markets:        storcfg.NewDefaultMarketsConfig(defaultStoreDirPath),
		Matching:       matching.NewDefaultConfig(),
		Parties:        storcfg.NewDefaultPartiesConfig(defaultStoreDirPath),
		Candles:        storcfg.NewDefaultCandlesConfig(defaultStoreDirPath),
		Risk:           storcfg.NewDefaultRiskConfig(defaultStoreDirPath),
		Pprof:          pprof.NewDefaultConfig(),
		Monitoring:     monitoring.NewDefaultConfig(),
		Logging:        logging.NewDefaultConfig(),
		Gateway:        gateway.NewDefaultConfig(),
		Position:       positions.NewDefaultConfig(),
		Settlement:     settlement.NewDefaultConfig(),
		Collateral:     collateral.NewDefaultConfig(),
		Auth:           auth.NewDefaultConfig(),
		Metrics:        metrics.NewDefaultConfig(),
		GatewayEnabled: true,
	}
}
