//lint:file-ignore SA5008 duplicated struct tags are ok for config

package config

import (
	"fmt"
	"os"

	"code.vegaprotocol.io/data-node/candlesv2"
	"code.vegaprotocol.io/data-node/service"

	"code.vegaprotocol.io/data-node/accounts"
	"code.vegaprotocol.io/data-node/api"
	"code.vegaprotocol.io/data-node/assets"
	"code.vegaprotocol.io/data-node/broker"
	"code.vegaprotocol.io/data-node/candles"
	"code.vegaprotocol.io/data-node/checkpoint"
	"code.vegaprotocol.io/data-node/config/encoding"
	"code.vegaprotocol.io/data-node/delegations"
	"code.vegaprotocol.io/data-node/epochs"
	"code.vegaprotocol.io/data-node/fee"
	"code.vegaprotocol.io/data-node/gateway"
	"code.vegaprotocol.io/data-node/governance"
	"code.vegaprotocol.io/data-node/liquidity"
	"code.vegaprotocol.io/data-node/logging"
	"code.vegaprotocol.io/data-node/markets"
	"code.vegaprotocol.io/data-node/metrics"
	"code.vegaprotocol.io/data-node/netparams"
	"code.vegaprotocol.io/data-node/nodes"
	"code.vegaprotocol.io/data-node/notary"
	"code.vegaprotocol.io/data-node/oracles"
	"code.vegaprotocol.io/data-node/orders"
	"code.vegaprotocol.io/data-node/parties"
	"code.vegaprotocol.io/data-node/pprof"
	"code.vegaprotocol.io/data-node/risk"
	"code.vegaprotocol.io/data-node/sqlstore"
	"code.vegaprotocol.io/data-node/storage"
	"code.vegaprotocol.io/data-node/subscribers"
	"code.vegaprotocol.io/data-node/trades"
	"code.vegaprotocol.io/data-node/transfers"
	"code.vegaprotocol.io/data-node/vegatime"
	vgfs "code.vegaprotocol.io/shared/libs/fs"
	"code.vegaprotocol.io/shared/paths"
)

// Config ties together all other application configuration types.
type Config struct {
	API               api.Config         `group:"API" namespace:"api"`
	Accounts          accounts.Config    `group:"Accounts" namespace:"accounts"`
	Candles           candles.Config     `group:"Candles" namespace:"candles"`
	CandlesV2         candlesv2.Config   `group:"CandlesV2" namespace:"candlesv2"`
	Logging           logging.Config     `group:"Logging" namespace:"logging"`
	Markets           markets.Config     `group:"Markets" namespace:"markets"`
	Oracles           oracles.Config     `group:"Oracles" namespace:"oracles"`
	Orders            orders.Config      `group:"Orders" namespace:"orders"`
	Liquidity         liquidity.Config   `group:"Liquidity" namespace:"liquidity"`
	Parties           parties.Config     `group:"Parties" namespace:"parties"`
	Risk              risk.Config        `group:"Risk" namespace:"risk"`
	Storage           storage.Config     `group:"Storage" namespace:"storage"`
	SQLStore          sqlstore.Config    `group:"Sqlstore" namespace:"sqlstore"`
	Trades            trades.Config      `group:"Trades" namespace:"trades"`
	Time              vegatime.Config    `group:"Time" namespace:"time"`
	Gateway           gateway.Config     `group:"Gateway" namespace:"gateway"`
	Metrics           metrics.Config     `group:"Metrics" namespace:"metrics"`
	Transfers         transfers.Config   `group:"Transfers" namespace:"transfers"`
	Governance        governance.Config  `group:"Governance" namespace:"governance"`
	Assets            assets.Config      `group:"Assets" namespace:"assets"`
	Notary            notary.Config      `group:"Notary" namespace:"notary"`
	Subscribers       subscribers.Config `group:"Subscribers" namespace:"subscribers"`
	Fee               fee.Config         `group:"Fee" namespace:"fee"`
	Broker            broker.Config      `group:"Broker" namespace:"broker"`
	Nodes             nodes.Config       `group:"Nodes" namespace:"nodes"`
	Epochs            epochs.Config      `group:"Epochs" namespace:"epochs"`
	Delegations       delegations.Config `group:"Delegations" namespace:"delegations"`
	Checkpoint        checkpoint.Config  `group:"Checkpoint" namespace:"checkpoint"`
	NetworkParameters netparams.Config   `group:"NetworkParameters" namespace:"network_parameters"`
	Service           service.Config     `group:"Service" namespace:"service"`

	Pprof          pprof.Config  `group:"Pprof" namespace:"pprof"`
	GatewayEnabled encoding.Bool `long:"gateway-enabled" choice:"true" choice:"false" description:" "`
	UlimitNOFile   uint64        `long:"ulimit-no-files" description:"Set the max number of open files (see: ulimit -n)" tomlcp:"Set the max number of open files (see: ulimit -n)"`
}

// NewDefaultConfig returns a set of default configs for all vega packages, as specified at the per package
// config level, if there is an error initialising any of the configs then this is returned.
func NewDefaultConfig() Config {
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
		CandlesV2:         candlesv2.NewDefaultConfig(),
		Risk:              risk.NewDefaultConfig(),
		Storage:           storage.NewDefaultConfig(),
		SQLStore:          sqlstore.NewDefaultConfig(),
		Pprof:             pprof.NewDefaultConfig(),
		Logging:           logging.NewDefaultConfig(),
		Gateway:           gateway.NewDefaultConfig(),
		Metrics:           metrics.NewDefaultConfig(),
		Transfers:         transfers.NewDefaultConfig(),
		Governance:        governance.NewDefaultConfig(),
		Assets:            assets.NewDefaultConfig(),
		Notary:            notary.NewDefaultConfig(),
		Subscribers:       subscribers.NewDefaultConfig(),
		Fee:               fee.NewDefaultConfig(),
		NetworkParameters: netparams.NewDefaultConfig(),
		Broker:            broker.NewDefaultConfig(),
		Epochs:            epochs.NewDefaultConfig(),
		Nodes:             nodes.NewDefaultConfig(),
		Delegations:       delegations.NewDefaultConfig(),
		Service:           service.NewDefaultConfig(),
		GatewayEnabled:    true,
		UlimitNOFile:      8192,
	}
}

type Loader struct {
	configFilePath string
}

func InitialiseLoader(vegaPaths paths.Paths) (*Loader, error) {
	configFilePath, err := vegaPaths.CreateConfigPathFor(paths.DataNodeDefaultConfigFile)
	if err != nil {
		return nil, fmt.Errorf("couldn't get path for %s: %w", paths.NodeDefaultConfigFile, err)
	}

	return &Loader{
		configFilePath: configFilePath,
	}, nil
}

func (l *Loader) ConfigFilePath() string {
	return l.configFilePath
}

func (l *Loader) ConfigExists() (bool, error) {
	exists, err := vgfs.FileExists(l.configFilePath)
	if err != nil {
		return false, err
	}
	return exists, nil
}

func (l *Loader) Save(cfg *Config) error {
	if err := paths.WriteStructuredFile(l.configFilePath, cfg); err != nil {
		return fmt.Errorf("couldn't write configuration file at %s: %w", l.configFilePath, err)
	}
	return nil
}

func (l *Loader) Get() (*Config, error) {
	cfg := NewDefaultConfig()
	if err := paths.ReadStructuredFile(l.configFilePath, &cfg); err != nil {
		return nil, fmt.Errorf("couldn't read configuration file at %s: %w", l.configFilePath, err)
	}
	return &cfg, nil
}

func (l *Loader) Remove() {
	_ = os.RemoveAll(l.configFilePath)
}
