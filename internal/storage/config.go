package storage

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"time"

	"code.vegaprotocol.io/vega/internal/logging"

	"github.com/pkg/errors"
)

const (
	CandleStoreDataPath = "candlestore"
	OrderStoreDataPath  = "orderstore"
	TradeStoreDataPath  = "tradestore"

	// namedLogger is the identifier for package and should ideally match the package name
	// this is simply emitted as a hierarchical label e.g. 'api.grpc'.
	namedLogger = "storage"

	defaultStorageAccessTimeout = 5
)

// Config provides package level settings, configuration and logging.
type Config struct {
	log   *logging.Logger
	Level logging.Level

	OrderStoreDirPath  string
	TradeStoreDirPath  string
	CandleStoreDirPath string
	//LogPartyStoreDebug    bool
	//LogOrderStoreDebug    bool
	//LogCandleStoreDebug   bool
	LogPositionStoreDebug bool
	Timeout               time.Duration
}

// NewConfig constructs a new Config instance with default parameters.
// This constructor is used by the vega application code. Logger is a
// pointer to a logging instance and defaultStoreDirPath is the root directory
// where all storage directories are to be read from and written to.
func NewDefaultConfig(logger *logging.Logger, defaultStoreDirPath string) *Config {
	logger = logger.Named(namedLogger)

	return &Config{
		log:                logger,
		Level:              logging.InfoLevel,
		OrderStoreDirPath:  filepath.Join(defaultStoreDirPath, OrderStoreDataPath),
		TradeStoreDirPath:  filepath.Join(defaultStoreDirPath, TradeStoreDataPath),
		CandleStoreDirPath: filepath.Join(defaultStoreDirPath, CandleStoreDataPath),
		//LogPartyStoreDebug:    true,
		//LogOrderStoreDebug:    true,
		//LogCandleStoreDebug:   false,
		LogPositionStoreDebug: false,
		Timeout:               defaultStorageAccessTimeout,
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

// GetLogger returns a pointer to the current underlying logger for this package.
func (c *Config) GetLogger() *logging.Logger {
	return c.log
}

// UpdateLogger will set any new values on the underlying logging core. Useful when configs are
// hot reloaded at run time. Currently we only check and refresh the logging level.
func (c *Config) UpdateLogger() {
	c.log.SetLevel(c.Level)
}

// NewTestConfig constructs a new Config instance with test parameters.
// This constructor is exclusively used in unit tests/integration tests
func NewTestConfig() (*Config, error) {
	// Test logger can be configured here, default to console not file etc.
	logger := logging.NewLoggerFromEnv("dev")
	// Test configuration for badger stores
	cfg := Config{
		log:                   logger,
		OrderStoreDirPath:     fmt.Sprintf("/tmp/vegatests/orderstore-%v", randSeq(5)),
		TradeStoreDirPath:     fmt.Sprintf("/tmp/vegatests/tradestore-%v", randSeq(5)),
		CandleStoreDirPath:    fmt.Sprintf("/tmp/vegatests/candlestore-%v", randSeq(5)),
		LogPositionStoreDebug: true,
		Timeout:               defaultStorageAccessTimeout,
	}

	if err := ensureDir(cfg.CandleStoreDirPath); err != nil {
		return nil, err
	}
	if err := ensureDir(cfg.OrderStoreDirPath); err != nil {
		return nil, err
	}
	if err := ensureDir(cfg.TradeStoreDirPath); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func ensureDir(path string) error {
	const (
		dirPerms = 0700
	)

	_, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return os.MkdirAll(path, dirPerms)
		}
		return err
	}
	return nil
}

var chars = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

func randSeq(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = chars[rand.Intn(len(chars))]
	}
	return string(b)
}
