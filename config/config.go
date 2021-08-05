//lint:file-ignore SA5008 duplicated struct tags are ok for config

package config

import (
	"io/ioutil"
	"path/filepath"

	"code.vegaprotocol.io/data-node/accounts"
	"code.vegaprotocol.io/data-node/api"
	"code.vegaprotocol.io/data-node/assets"
	"code.vegaprotocol.io/data-node/broker"
	"code.vegaprotocol.io/data-node/candles"
	"code.vegaprotocol.io/data-node/config/encoding"
	"code.vegaprotocol.io/data-node/fee"
	"code.vegaprotocol.io/data-node/gateway"
	"code.vegaprotocol.io/data-node/governance"
	"code.vegaprotocol.io/data-node/liquidity"
	"code.vegaprotocol.io/data-node/logging"
	"code.vegaprotocol.io/data-node/markets"
	"code.vegaprotocol.io/data-node/metrics"
	"code.vegaprotocol.io/data-node/netparams"
	"code.vegaprotocol.io/data-node/notary"
	"code.vegaprotocol.io/data-node/oracles"
	"code.vegaprotocol.io/data-node/orders"
	"code.vegaprotocol.io/data-node/parties"
	"code.vegaprotocol.io/data-node/pprof"
	"code.vegaprotocol.io/data-node/risk"
	"code.vegaprotocol.io/data-node/stats"
	"code.vegaprotocol.io/data-node/storage"
	"code.vegaprotocol.io/data-node/subscribers"
	"code.vegaprotocol.io/data-node/trades"
	"code.vegaprotocol.io/data-node/transfers"
	"code.vegaprotocol.io/data-node/vegatime"

	"github.com/zannen/toml"
)

// Config ties together all other application configuration types.
type Config struct {
	API               api.Config         `group:"API" namespace:"api"`
	Accounts          accounts.Config    `group:"Accounts" namespace:"accounts"`
	Candles           candles.Config     `group:"Candles" namespace:"candles"`
	Logging           logging.Config     `group:"Logging" namespace:"logging"`
	Markets           markets.Config     `group:"Markets" namespace:"markets"`
	Oracles           oracles.Config     `group:"Oracles" namespace:"oracles"`
	Orders            orders.Config      `group:"Orders" namespace:"orders"`
	Liquidity         liquidity.Config   `group:"Liquidity" namespace:"liquidity"`
	Parties           parties.Config     `group:"Parties" namespace:"parties"`
	Risk              risk.Config        `group:"Risk" namespace:"risk"`
	Storage           storage.Config     `group:"Storage" namespace:"storage"`
	Trades            trades.Config      `group:"Trades" namespace:"trades"`
	Time              vegatime.Config    `group:"Time" namespace:"time"`
	Gateway           gateway.Config     `group:"Gateway" namespace:"gateway"`
	Metrics           metrics.Config     `group:"Metrics" namespace:"metrics"`
	Transfers         transfers.Config   `group:"Transfers" namespace:"transfers"`
	Governance        governance.Config  `group:"Governance" namespace:"governance"`
	Assets            assets.Config      `group:"Assets" namespace:"assets"`
	Notary            notary.Config      `group:"Notary" namespace:"notary"`
	Subscribers       subscribers.Config `group:"Subscribers" namespace:"subscribers"`
	Stats             stats.Config       `group:"Stats" namespace:"stats"`
	Fee               fee.Config         `group:"Fee" namespace:"fee"`
	Broker            broker.Config      `group:"Broker" namespace:"broker"`
	NetworkParameters netparams.Config

	Pprof          pprof.Config  `group:"Pprof" namespace:"pprof"`
	GatewayEnabled encoding.Bool `long:"gateway-enabled" choice:"true" choice:"false" description:" "`
	StoresEnabled  encoding.Bool `long:"stores-enabled" choice:"true" choice:"false" description:" "`
	UlimitNOFile   uint64        `long:"ulimit-no-files" description:"Set the max number of open files (see: ulimit -n)" tomlcp:"Set the max number of open files (see: ulimit -n)"`
}

// NewDefaultConfig returns a set of default configs for all vega packages, as specified at the per package
// config level, if there is an error initialising any of the configs then this is returned.
func NewDefaultConfig(defaultStoreDirPath string) Config {
	return Config{
		Trades:            trades.NewDefaultConfig(),
		API:               api.NewDefaultConfig(),
		Accounts:          accounts.NewDefaultConfig(),
		Oracles:           oracles.NewDefaultConfig(),
		Orders:            orders.NewDefaultConfig(),
		Liquidity:         liquidity.NewDefaultConfig(),
		Time:              vegatime.NewDefaultConfig(),
		Markets:           markets.NewDefaultConfig(),
		Parties:           parties.NewDefaultConfig(),
		Candles:           candles.NewDefaultConfig(),
		Risk:              risk.NewDefaultConfig(),
		Storage:           storage.NewDefaultConfig(defaultStoreDirPath),
		Pprof:             pprof.NewDefaultConfig(),
		Logging:           logging.NewDefaultConfig(),
		Gateway:           gateway.NewDefaultConfig(),
		Metrics:           metrics.NewDefaultConfig(),
		Transfers:         transfers.NewDefaultConfig(),
		Governance:        governance.NewDefaultConfig(),
		Assets:            assets.NewDefaultConfig(),
		Notary:            notary.NewDefaultConfig(),
		Stats:             stats.NewDefaultConfig(),
		Subscribers:       subscribers.NewDefaultConfig(),
		Fee:               fee.NewDefaultConfig(),
		NetworkParameters: netparams.NewDefaultConfig(),
		Broker:            broker.NewDefaultConfig(),
		GatewayEnabled:    true,
		StoresEnabled:     true,
		UlimitNOFile:      8192,
	}
}

func Read(rootPath string) (*Config, error) {
	path := filepath.Join(rootPath, configFileName)
	buf, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	cfg := NewDefaultConfig(rootPath)
	if _, err := toml.Decode(string(buf), &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil

}
