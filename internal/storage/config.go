package storage

import (
	"os"
	"vega/internal/logging"
	"github.com/pkg/errors"
	"fmt"
)

// Config provides package level settings, configuration and logging.
type Config struct {
	log   logging.Logger
	level logging.Level

	OrderStoreDirPath     string
	TradeStoreDirPath     string
	CandleStoreDirPath    string
	LogPartyStoreDebug    bool
	LogOrderStoreDebug    bool
	LogCandleStoreDebug   bool
	LogPositionStoreDebug bool
}

// NewConfig constructs a new Config instance with default parameters.
// This constructor is used by the vega application code.
func NewConfig(logger logging.Logger) *Config {
	level := logging.DebugLevel
	logger = logger.Named("storage")
	return &Config{
		log:                   logger,
		level:                 level,
		OrderStoreDirPath:     "../../data/orderstore",
		TradeStoreDirPath:     "../../data/tradestore",
		CandleStoreDirPath:    "../../data/candlestore",
		LogPartyStoreDebug:    true,
		LogOrderStoreDebug:    true,
		LogCandleStoreDebug:   false,
		LogPositionStoreDebug: false,
	}
}

// NewTestConfig constructs a new Config instance with test parameters.
// This constructor is exclusively used in unit tests/integration tests
func NewTestConfig() *Config {
	// Test logger can be configured here, default to console not file etc.
	logger := logging.NewLogger()
	logger.InitConsoleLogger(logging.DebugLevel)
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
		c.log.Errorf("error flushing order store: %s", err)
	}
	if _, err := os.Stat(c.OrderStoreDirPath); os.IsNotExist(err) {
		err = os.MkdirAll(c.OrderStoreDirPath, os.ModePerm)
		if err != nil {
			c.log.Errorf("error creating order store: %s", err)
		}
	}
	err = os.RemoveAll(c.TradeStoreDirPath)
	if err != nil {
		c.log.Errorf("error flushing trade store: %s", err)
	}
	if _, err := os.Stat(c.TradeStoreDirPath); os.IsNotExist(err) {
		err = os.MkdirAll(c.TradeStoreDirPath, os.ModePerm)
		if err != nil {
			c.log.Errorf("error creating trade store: %s", err)
		}
	}
	err = os.RemoveAll(c.CandleStoreDirPath)
	if err != nil {
		c.log.Errorf("error flushing candle store: %s", err)
	}
	if _, err := os.Stat(c.CandleStoreDirPath); os.IsNotExist(err) {
		err = os.MkdirAll(c.CandleStoreDirPath, os.ModePerm)
		if err != nil {
			c.log.Errorf("error creating candle store: %s", err)
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