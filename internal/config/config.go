package config

import (
	"code.vegaprotocol.io/vega/internal/api"
	"code.vegaprotocol.io/vega/internal/blockchain"
	"code.vegaprotocol.io/vega/internal/candles"
	"code.vegaprotocol.io/vega/internal/execution"
	"code.vegaprotocol.io/vega/internal/logging"
	"code.vegaprotocol.io/vega/internal/markets"
	"code.vegaprotocol.io/vega/internal/monitoring"
	"code.vegaprotocol.io/vega/internal/orders"
	"code.vegaprotocol.io/vega/internal/parties"
	"code.vegaprotocol.io/vega/internal/pprof"
	"code.vegaprotocol.io/vega/internal/risk"
	"code.vegaprotocol.io/vega/internal/storage"
	"code.vegaprotocol.io/vega/internal/trades"
	"code.vegaprotocol.io/vega/internal/vegatime"
)

// Config ties together all other application configuration types.
type Config struct {
	log        *logging.Logger
	API        api.Config
	Blockchain blockchain.Config
	Candles    candles.Config
	Execution  execution.Config
	Logging    logging.Config
	Markets    markets.Config
	Orders     orders.Config
	Parties    parties.Config
	Risk       risk.Config
	Storage    storage.Config
	Trades     trades.Config
	Time       vegatime.Config
	Monitoring monitoring.Config

	Pprof pprof.Config
}

// NewDefaultConfig returns a set of default configs for all vega packages, as specified at the per package
// config level, if there is an error initialising any of the configs then this is returned.
func NewDefaultConfig(log *logging.Logger, defaultStoreDirPath string) Config {
	return Config{
		log:        log,
		Trades:     trades.NewDefaultConfig(log),
		Blockchain: blockchain.NewDefaultConfig(log),
		Execution:  execution.NewDefaultConfig(log, defaultStoreDirPath),
		API:        api.NewDefaultConfig(log),
		Orders:     orders.NewDefaultConfig(log),
		Time:       vegatime.NewDefaultConfig(log),
		Markets:    markets.NewDefaultConfig(log),
		Parties:    parties.NewDefaultConfig(log),
		Candles:    candles.NewDefaultConfig(log),
		Storage:    storage.NewDefaultConfig(log, defaultStoreDirPath),
		Risk:       risk.NewDefaultConfig(log),
		Pprof:      pprof.NewDefaultConfig(log),
		Monitoring: monitoring.NewDefaultConfig(log),
		Logging:    logging.NewDefaultConfig(),
	}
}

// ResetLoggers will re-create loggers of all config instances.
func (c *Config) ResetLoggers(oldLogEnv string) {
	newLogEnv := c.Logging.Environment
	if oldLogEnv == newLogEnv {
		return
	}
	c.log = logging.NewLoggerFromConfig(c.Logging)

	/*
		c.log.Info("Logging environment set", logging.String("environment", newLogEnv))

		c.API.SetLogger(c.log)
		c.Blockchain.SetLogger(c.log)
		c.Candles.SetLogger(c.log)
		c.Execution.SetLogger(c.log)
		c.Markets.SetLogger(c.log)
		c.Monitoring.SetLogger(c.log)
		c.Orders.SetLogger(c.log)
		c.Parties.SetLogger(c.log)
		c.Pprof.SetLogger(c.log)
		c.Risk.SetLogger(c.log)
		c.Storage.SetLogger(c.log)
		c.Time.SetLogger(c.log)
		c.Trades.SetLogger(c.log)
		// Any new package configs with a logger should be added here, in alphabetical order.

	*/
}

func (c *Config) updateLoggers() {
	// We need to call UpdateLogger on each config instance so that
	// the zap core is updated to the new logging level.
	/*
		c.Trades.UpdateLogger()
		c.Blockchain.UpdateLogger()
		c.Execution.UpdateLogger()
		c.API.UpdateLogger()
		c.Orders.UpdateLogger()
		c.Time.UpdateLogger()
		c.Markets.UpdateLogger()
		c.Parties.UpdateLogger()
		c.Candles.UpdateLogger()
		c.Storage.UpdateLogger()
		c.Risk.UpdateLogger()
		c.Pprof.UpdateLogger()
		c.Monitoring.UpdateLogger()
	*/
	// Any new package configs with a logger should be added here <>
}
