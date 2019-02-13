package internal

import (
	"vega/api"

	"vega/internal/blockchain"
	"vega/internal/candles"
	"vega/internal/execution"
	"vega/internal/logging"
	"vega/internal/markets"
	"vega/internal/matching"
	"vega/internal/orders"
	"vega/internal/parties"
	"vega/internal/risk"
	"vega/internal/storage"
	"vega/internal/trades"
	"vega/internal/vegatime"

	"github.com/pkg/errors"
	"github.com/spf13/viper"
	"github.com/fsnotify/fsnotify"
	"log"
)

// Config ties together all other application configuration types.
type Config struct {
	log        logging.Logger
	API        *api.Config
	Blockchain *blockchain.Config
	Candles    *candles.Config
	//Collatoral collatoral.config         // As packages continue to be
	Execution *execution.Config            // developed we add their config
	//Fees fees.config                     // options here see examples
	//Governanace governance.config
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
}

// NewConfig creates a top level configuration structure, this references all internal/sub-package configs and can
// load the configurations from file, etc. Typically a root logger will be passed in at this point and fed to the
// other sub-configs as required via DI.
func NewConfig(logger logging.Logger) (*Config, error) {
	if logger == nil {
		return nil, errors.New("logger instance is nil when calling NewConfig.")
	}
	return &Config{
		log: logger,
	}, nil
}

// ReadViperConfig attempts to load the full vega configuration tree from file at the path specified (config.toml)
// If a path of '.' is specified the current working directory will be searched.
func (c *Config) ReadViperConfig(path string) error {
	viper.SetConfigName("config")
	if len(path) == 0 {
		return errors.New("config from file requires a path")
	}
	viper.AddConfigPath(path)
	viper.AutomaticEnv()
	err := viper.ReadInConfig()
	if err != nil {
		return errors.Wrap(err, "error reading config from file")
	}
	err = viper.Unmarshal(&c)
	if err != nil {
		log.Fatalf("unable to decode into struct, %v", err)
	}
	c.PrintDebugging()
	return nil
}

// ListenForChanges adds a file system watcher for the config file specified as a path in `ReadViperConfig`. This
// will update the configuration dynamically when a field changes and filter throughout the application.
func (c *Config) ListenForChanges() {
	viper.WatchConfig()
	viper.OnConfigChange(func(e fsnotify.Event) {
		c.log.Infof("Config file changed:", e.Name)
	})
}


func (c *Config) DefaultConfig() []byte {
    return nil
}

func (c *Config) PrintDebugging() {

	c.log.Infof("blockchain server port: %d", c.Blockchain.ServerPort)

	c.log.Infof("blockchain debug time: %v", c.Blockchain.LogTimeDebug)

	c.log.Infof("blockchain level: %d", c.Blockchain.Level)
}