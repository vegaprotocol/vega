package storage

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"time"

	"code.vegaprotocol.io/vega/config/encoding"
	"code.vegaprotocol.io/vega/logging"

	"github.com/pkg/errors"
)

const (
	// AccountsDataPath is the default path for the account store files
	AccountsDataPath = "accountstore"
	// CandlesDataPath is the default path for the candle store files
	CandlesDataPath = "candlestore"
	// MarketsDataPath is the default path for the market store files
	MarketsDataPath = "marketstore"
	// OrdersDataPath is the default path for the order store files
	OrdersDataPath = "orderstore"
	// TradesDataPath is the default path for the trade store files
	TradesDataPath = "tradestore"

	// namedLogger is the identifier for package and should ideally match the package name
	// this is simply emitted as a hierarchical label e.g. 'api.grpc'.
	namedLogger = "storage"

	defaultStorageAccessTimeout = 5 * time.Second
)

// Config provides package level settings, configuration and logging.
type Config struct {
	Accounts ConfigOptions
	Candles  ConfigOptions
	Markets  ConfigOptions
	Orders   ConfigOptions
	Trades   ConfigOptions
	//Parties   ConfigOptions  // Further badger store or hybrid store options
	//Depth     ConfigOptions  // will go here in the future (examples shown)
	//Risk      ConfigOptions
	//Positions ConfigOptions

	Level encoding.LogLevel `long:"log-level"`

	Timeout encoding.Duration `long:"timeout"`

	AccountsDirPath string `long:"accounts-dir-path" description:" "`
	CandlesDirPath  string `long:"candles-dir-path" description:" "`
	MarketsDirPath  string `long:"markets-dir-path" description:" "`
	OrdersDirPath   string `long:"orders-dir-path" description:" "`
	TradesDirPath   string `long:"trades-dir-path" description:" "`

	LogPositionStoreDebug bool `long:"log-position-store-debug"`
}

// NewDefaultConfig constructs a new Config instance with default parameters.
// This constructor is used by the vega application code. Logger is a
// pointer to a logging instance and defaultStoreDirPath is the root directory
// where all storage directories are to be read from and written to.
func NewDefaultConfig(defaultStoreDirPath string) Config {
	return Config{
		Accounts:              DefaultStoreOptions(),
		Candles:               DefaultStoreOptions(),
		Markets:               DefaultMarketStoreOptions(),
		Orders:                DefaultStoreOptions(),
		Trades:                DefaultStoreOptions(),
		Level:                 encoding.LogLevel{Level: logging.WarnLevel},
		AccountsDirPath:       filepath.Join(defaultStoreDirPath, AccountsDataPath),
		OrdersDirPath:         filepath.Join(defaultStoreDirPath, OrdersDataPath),
		TradesDirPath:         filepath.Join(defaultStoreDirPath, TradesDataPath),
		CandlesDirPath:        filepath.Join(defaultStoreDirPath, CandlesDataPath),
		MarketsDirPath:        filepath.Join(defaultStoreDirPath, MarketsDataPath),
		LogPositionStoreDebug: false,
		Timeout:               encoding.Duration{Duration: defaultStorageAccessTimeout},
	}
}

// FlushStores will remove/clear the badger key and value files (i.e. databases)
// from disk at the locations specified by the given storage.Config. This is
// currently used within unit and integration tests to clear between runs.
func FlushStores(log *logging.Logger, c Config) {
	paths := map[string]string{
		"account": c.AccountsDirPath,
		"order":   c.OrdersDirPath,
		"trade":   c.TradesDirPath,
		"candle":  c.CandlesDirPath,
		"market":  c.MarketsDirPath,
	}
	for name, path := range paths {
		if err := os.RemoveAll(path); err != nil {
			log.Error(
				fmt.Sprintf("Failed to flush the %s path", name),
				logging.String("path", path),
				logging.Error(err),
			)
		}
		if _, err := os.Stat(path); os.IsNotExist(err) {
			if err = os.MkdirAll(path, os.ModePerm); err != nil {
				log.Error(
					fmt.Sprintf("Failed to create the %s store", name),
					logging.String("path", path),
					logging.Error(err),
				)
			}
		}
	}
}

// InitStoreDirectory create a directory if it does not already exists on the filesystem
func InitStoreDirectory(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := os.MkdirAll(path, os.ModePerm); err != nil {
			return errors.Wrap(err, fmt.Sprintf("could not create directory path for badger data store: %s", path))
		}
	}
	return nil
}

// NewTestConfig constructs a new Config instance with test parameters.
// This constructor is exclusively used in unit tests/integration tests
func NewTestConfig() (Config, error) {
	// Test configuration for badger stores
	cfg := Config{
		Accounts:              DefaultStoreOptions(),
		Candles:               DefaultStoreOptions(),
		Markets:               DefaultStoreOptions(),
		Orders:                DefaultStoreOptions(),
		Trades:                DefaultStoreOptions(),
		AccountsDirPath:       fmt.Sprintf("/tmp/vegatests/accountstore-%v", randSeq(5)),
		OrdersDirPath:         fmt.Sprintf("/tmp/vegatests/orderstore-%v", randSeq(5)),
		TradesDirPath:         fmt.Sprintf("/tmp/vegatests/tradestore-%v", randSeq(5)),
		CandlesDirPath:        fmt.Sprintf("/tmp/vegatests/candlestore-%v", randSeq(5)),
		MarketsDirPath:        fmt.Sprintf("/tmp/vegatests/marketstore-%v", randSeq(5)),
		LogPositionStoreDebug: true,
		Timeout:               encoding.Duration{Duration: defaultStorageAccessTimeout},
	}

	if err := ensureDir(cfg.CandlesDirPath); err != nil {
		return Config{}, err
	}
	if err := ensureDir(cfg.OrdersDirPath); err != nil {
		return Config{}, err
	}
	if err := ensureDir(cfg.TradesDirPath); err != nil {
		return Config{}, err
	}

	if err := ensureDir(cfg.MarketsDirPath); err != nil {
		return Config{}, err
	}

	return cfg, nil
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
