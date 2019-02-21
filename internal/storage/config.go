package storage

import (
	"fmt"
	"os"
	"path/filepath"

	"vega/internal/fsutil"
	"vega/internal/logging"

	"github.com/pkg/errors"
)

const (
	CandelStoreDataPath = "candlestore"
	OrderStoreDataPath  = "orderstore"
	TradeStoreDataPath  = "tradestore"

	// namedLogger is the identifier for package and should ideally match the package name
	// this is simply emitted as a hierarchical label e.g. 'api.grpc'.
	namedLogger = "storage"
)

// Config provides package level settings, configuration and logging.
type Config struct {
	log   *logging.Logger
	Level logging.Level

	OrderStoreDirPath  string `mapstructure:"order_store_path"`
	TradeStoreDirPath  string `mapstructure:"trade_store_path"`
	CandleStoreDirPath string `mapstructure:"candle_store_path"`
	//LogPartyStoreDebug    bool     `mapstructure:"party_store_debug"`
	//LogOrderStoreDebug    bool     `mapstructure:"order_store_debug"`
	//LogCandleStoreDebug   bool     `mapstructure:"candle_store_debug"`
	LogPositionStoreDebug bool `mapstructure:"position_store_debug"`
}

// NewConfig constructs a new Config instance with default parameters.
// This constructor is used by the vega application code.
func NewConfig(logger *logging.Logger) *Config {
	logger = logger.Named(namedLogger)
	rootpath := fsutil.DefaultRootDir()

	return &Config{
		log:                logger,
		Level:              logging.InfoLevel,
		OrderStoreDirPath:  filepath.Join(rootpath, OrderStoreDataPath),
		TradeStoreDirPath:  filepath.Join(rootpath, TradeStoreDataPath),
		CandleStoreDirPath: filepath.Join(rootpath, CandelStoreDataPath),
		//LogPartyStoreDebug:    true,
		//LogOrderStoreDebug:    true,
		//LogCandleStoreDebug:   false,
		LogPositionStoreDebug: false,
	}
}

// NewTestConfig constructs a new Config instance with test parameters.
// This constructor is exclusively used in unit tests/integration tests
func NewTestConfig() *Config {
	// Test logger can be configured here, default to console not file etc.
	logger := logging.NewLoggerFromEnv("dev") // todo(cdm): add test env or some other config e.g file
	logger.AddExitHandler()
	// Test configuration for badger stores
	return &Config{
		log:                   logger,
		OrderStoreDirPath:     "./tmp/orderstore-test",
		TradeStoreDirPath:     "./tmp/tradestore-test",
		CandleStoreDirPath:    "./tmp/candlestore-test",
		LogPositionStoreDebug: true,
	}
}

// FlushStores will remove/clear the badger key and value files (i.e. databases)
// from disk at the locations specified by the given storage.Config. This is
// currently used within unit and integration tests to clear between runs.
func FlushStores(c *Config) {
	err := os.RemoveAll(c.OrderStoreDirPath)
	if err != nil {
		c.log.Error("Failed to flush the order store",
			logging.String("path", c.OrderStoreDirPath),
			logging.Error(err))
	}
	if _, err := os.Stat(c.OrderStoreDirPath); os.IsNotExist(err) {
		err = os.MkdirAll(c.OrderStoreDirPath, os.ModePerm)
		if err != nil {
			c.log.Error("Failed to create the order store",
				logging.String("path", c.OrderStoreDirPath),
				logging.Error(err))
		}
	}
	err = os.RemoveAll(c.TradeStoreDirPath)
	if err != nil {
		c.log.Error("Failed to flush the trade store",
			logging.String("path", c.TradeStoreDirPath),
			logging.Error(err))
	}
	if _, err := os.Stat(c.TradeStoreDirPath); os.IsNotExist(err) {
		err = os.MkdirAll(c.TradeStoreDirPath, os.ModePerm)
		if err != nil {
			c.log.Error("Failed to create the trade store",
				logging.String("path", c.TradeStoreDirPath),
				logging.Error(err))
		}
	}
	err = os.RemoveAll(c.CandleStoreDirPath)
	if err != nil {
		c.log.Error("Failed to flush the candle store",
			logging.String("path", c.CandleStoreDirPath),
			logging.Error(err))
	}
	if _, err := os.Stat(c.CandleStoreDirPath); os.IsNotExist(err) {
		err = os.MkdirAll(c.CandleStoreDirPath, os.ModePerm)
		if err != nil {
			c.log.Error("Failed to create the candle store",
				logging.String("path", c.TradeStoreDirPath),
				logging.Error(err))
		}
	}
}

func InitStoreDirectory(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		err := os.MkdirAll(path, os.ModePerm)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("could not create directory path for badger data store: %s", path))
		}
	}
	return nil
}
