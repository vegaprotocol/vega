package internal

import (
	"code.vegaprotocol.io/vega/internal/api"
	"code.vegaprotocol.io/vega/internal/blockchain"
	"code.vegaprotocol.io/vega/internal/candles"
	"code.vegaprotocol.io/vega/internal/execution"
	"code.vegaprotocol.io/vega/internal/logging"
	"code.vegaprotocol.io/vega/internal/markets"
	"code.vegaprotocol.io/vega/internal/matching"
	"code.vegaprotocol.io/vega/internal/orders"
	"code.vegaprotocol.io/vega/internal/parties"
	"code.vegaprotocol.io/vega/internal/risk"
	"code.vegaprotocol.io/vega/internal/storage"
	"code.vegaprotocol.io/vega/internal/trades"
	"code.vegaprotocol.io/vega/internal/vegatime"

	"github.com/fsnotify/fsnotify"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

// Config ties together all other application configuration types.
type Config struct {
	log        *logging.Logger
	API        *api.Config
	Blockchain *blockchain.Config
	Candles    *candles.Config
	//Collatoral collatoral.config
	Execution *execution.Config
	//Fees *fees.config
	//Governanace *governance.config
	Logging  *logging.Config
	Markets  *markets.Config
	Matching *matching.Config
	//Monitoring monitoring.Config
	Orders  *orders.Config
	Parties *parties.Config
	Risk    *risk.Config
	//Settlement *settlement.Config
	Storage *storage.Config
	Trades  *trades.Config
	Time    *vegatime.Config
	// Any new package configs should be added here <> (see examples above)
}

// NewDefaultConfig returns a set of default configs for all vega packages, as specified at the per package
// config level, if there is an error initialising any of the configs then this is returned.
func NewDefaultConfig(logger *logging.Logger, defaultStoreDirPath string) (*Config, error) {
	if logger == nil {
		return nil, errors.New("logger instance is nil when calling NewConfig")
	}
	if defaultStoreDirPath == "" {
		return nil, errors.New("root storage directory is empty when calling NewConfig")
	}
	c := &Config{
		log: logger,
	}

	c.Trades = trades.NewDefaultConfig(c.log)
	c.Blockchain = blockchain.NewDefaultConfig(c.log)
	c.Execution = execution.NewDefaultConfig(c.log)
	c.Matching = matching.NewDefaultConfig(c.log)
	c.API = api.NewDefaultConfig(c.log)
	c.Orders = orders.NewDefaultConfig(c.log)
	c.Time = vegatime.NewDefaultConfig(c.log)
	c.Markets = markets.NewDefaultConfig(c.log)
	c.Parties = parties.NewDefaultConfig(c.log)
	c.Candles = candles.NewDefaultConfig(c.log)
	c.Storage = storage.NewDefaultConfig(c.log, defaultStoreDirPath)
	c.Risk = risk.NewDefaultConfig(c.log)
	c.Logging = logging.NewDefaultConfig()
	// Any new package configs should be added here <>

	return c, nil
}

// NewConfigFromFile attempts to load the full vega configuration tree from file at the path specified (config.toml)
// If a path of '.' is specified the current working directory will be searched.
func NewConfigFromFile(logger *logging.Logger, path string) (*Config, error) {

	// Read in the default configuration for VEGA (defined in each sub-package config).
	c, err := NewDefaultConfig(logger, path)
	if err != nil {
		return nil, err
	}

	// Sadly this step is manual, assign defaults to viper so when it merges config it can set initial values.
	viper.SetDefault("API", c.API)
	viper.SetDefault("Blockchain", c.Blockchain)
	viper.SetDefault("Candles", c.Candles)
	//viper.SetDefault("Collatoral", c.Collatoral)
	viper.SetDefault("Execution", c.Execution)
	//viper.SetDefault("Fees", c.Fees)
	//viper.SetDefault("Governance", c.Governance)
	viper.SetDefault("Logging", c.Logging)
	viper.SetDefault("Markets", c.Markets)
	viper.SetDefault("Matching", c.Matching)
	//viper.SetDefault("Monitoring", c.Monitoring)
	viper.SetDefault("Orders", c.Orders)
	viper.SetDefault("Parties", c.Parties)
	viper.SetDefault("Risk", c.Risk)
	//viper.SetDefault("Settlement", c.Settlement)
	viper.SetDefault("Storage", c.Storage)
	viper.SetDefault("Trades", c.Trades)
	viper.SetDefault("Time", c.Time)
	// Any new package configs should be added here <> (see examples above)

	// Read in the configs from toml file and attempt to unmarshal into config struct.
	viper.SetConfigName("config")
	if len(path) == 0 {
		return nil, errors.New("config from file requires a path")
	}
	viper.AddConfigPath(path)

	viper.AutomaticEnv()
	err = viper.ReadInConfig()
	if err != nil {
		return nil, err
	}
	err = viper.Unmarshal(&c)
	if err != nil {
		return nil, errors.Wrap(err, "unable to decode into struct")
	}

	// We need to call update logger on each config instance so that
	// the zap core is updated to the new logging level.
	c.updateLoggers()
	return c, nil
}

// ListenForChanges adds a file system watcher for the config file specified as a path in `ReadViperConfig`. This
// will update the configuration dynamically when a field changes and filter throughout the application.
func (c *Config) ListenForChanges() {
	viper.WatchConfig()
	viper.OnConfigChange(func(e fsnotify.Event) {
		err := viper.Unmarshal(&c)
		if err != nil {
			c.log.Warn("Failed to unmarshal vega config to struct on config change",
				logging.Error(errors.Wrap(err, "unable to decode into struct")))
		}
		c.log.Debug("Vega config toml file changed, updating package level loggers",
			logging.String("config-file", e.Name))

		// We need to call update logger on each config instance so that
		// the zap core is updated to the new logging level.
		// ==> If the file changes we should hot reload.
		c.updateLoggers()
	})
}

func (c *Config) updateLoggers() {
	// We need to call update logger on each config instance so that
	// the zap core is updated to the new logging level.
	c.Trades.UpdateLogger()
	c.Blockchain.UpdateLogger()
	c.Execution.UpdateLogger()
	c.Matching.UpdateLogger()
	c.API.UpdateLogger()
	c.Orders.UpdateLogger()
	c.Time.UpdateLogger()
	c.Markets.UpdateLogger()
	c.Parties.UpdateLogger()
	c.Candles.UpdateLogger()
	c.Storage.UpdateLogger()
	c.Risk.UpdateLogger()
	// Any new package configs with a logger should be added here <>
}
