package config

import (
	"io/ioutil"
	"path/filepath"

	"code.vegaprotocol.io/vega/accounts"
	"code.vegaprotocol.io/vega/api"
	"code.vegaprotocol.io/vega/assets"
	"code.vegaprotocol.io/vega/banking"
	"code.vegaprotocol.io/vega/blockchain"
	"code.vegaprotocol.io/vega/candles"
	"code.vegaprotocol.io/vega/collateral"
	"code.vegaprotocol.io/vega/config/encoding"
	"code.vegaprotocol.io/vega/evtforward"
	"code.vegaprotocol.io/vega/execution"
	"code.vegaprotocol.io/vega/gateway"
	"code.vegaprotocol.io/vega/genesis"
	"code.vegaprotocol.io/vega/governance"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/markets"
	"code.vegaprotocol.io/vega/matching"
	"code.vegaprotocol.io/vega/metrics"
	"code.vegaprotocol.io/vega/monitoring"
	"code.vegaprotocol.io/vega/netparams"
	"code.vegaprotocol.io/vega/nodewallet"
	"code.vegaprotocol.io/vega/notary"
	"code.vegaprotocol.io/vega/orders"
	"code.vegaprotocol.io/vega/parties"
	"code.vegaprotocol.io/vega/positions"
	"code.vegaprotocol.io/vega/pprof"
	"code.vegaprotocol.io/vega/processor"
	"code.vegaprotocol.io/vega/risk"
	"code.vegaprotocol.io/vega/settlement"
	"code.vegaprotocol.io/vega/stats"
	"code.vegaprotocol.io/vega/storage"
	"code.vegaprotocol.io/vega/subscribers"
	"code.vegaprotocol.io/vega/trades"
	"code.vegaprotocol.io/vega/transfers"
	"code.vegaprotocol.io/vega/validators"
	"code.vegaprotocol.io/vega/vegatime"

	"github.com/zannen/toml"
)

// Config ties together all other application configuration types.
type Config struct {
	API         api.Config         `group:"API" namespace:"api"`
	Accounts    accounts.Config    `group:"Accounts" namespace:"accounts"`
	Blockchain  blockchain.Config  `group:"Blockchain" namespace:"blockchain"`
	Candles     candles.Config     `group:"Candles" namespace:"candles"`
	Collateral  collateral.Config  `group:"Collateral" namespace:"collateral"`
	Execution   execution.Config   `group:"Execution" namespace:"execution"`
	Processor   processor.Config   `group:"Processor" namespace:"processor"`
	Logging     logging.Config     `group:"logging" namespace:"logging"`
	Matching    matching.Config    `group:"Matching" namespace:"matching"`
	Markets     markets.Config     `group:"Markets" namespace:"markets"`
	Orders      orders.Config      `group:"Orders" namespace:"orders"`
	Parties     parties.Config     `group:"Parties" namespace:"parties"`
	Position    positions.Config   `group:"Position" namespace:"position"`
	Risk        risk.Config        `group:"Risk" namespace:"risk"`
	Settlement  settlement.Config  `group:"Settlement" namespace:"settlement"`
	Storage     storage.Config     `group:"Storage" namespace:"storage"`
	Trades      trades.Config      `group:"Trades" namespace:"trades"`
	Time        vegatime.Config    `group:"Time" namespace:"time"`
	Monitoring  monitoring.Config  `group:"Monitoring" namespace:"monitoring"`
	Gateway     gateway.Config     `group:"Gateway" namespace:"gateway"`
	Metrics     metrics.Config     `group:"Metrics" namespace:"metrics"`
	Transfers   transfers.Config   `group:"Transfers" namespace:"transfers"`
	Governance  governance.Config  `group:"Governance" namespace:"governance"`
	NodeWallet  nodewallet.Config  `group:"NodeWallet" namespace:"nodewallet"`
	Assets      assets.Config      `group:"Assets" namespace:"assets"`
	Notary      notary.Config      `group:"Notary" namespace:"notary"`
	EvtForward  evtforward.Config  `group:"EvtForward" namespace:"evtForward"`
	Subscribers subscribers.Config `group:"Subscribers" namespace:"subscribers"`
	Genesis     genesis.Config     `group:"Genesis" namespace:"genesis"`
	Validators  validators.Config  `group:"Validators" namespace:"validators"`
	Banking     banking.Config     `group:"Banking" namespace:"banking"`
	Stats       stats.Config       `group:"Stats" namespace:"stats"`
	NetworkParameters netparams.Config

	Pprof          pprof.Config `group:"Pprof" namespace:"pprof"`
	GatewayEnabled encoding.Bool
	StoresEnabled  encoding.Bool
	UlimitNOFile   uint64 `tomlcp:"Set the max number of open files (see: ulimit -n)"`
}

// NewDefaultConfig returns a set of default configs for all vega packages, as specified at the per package
// config level, if there is an error initialising any of the configs then this is returned.
func NewDefaultConfig(defaultStoreDirPath string) Config {
	return Config{
		Trades:            trades.NewDefaultConfig(),
		Blockchain:        blockchain.NewDefaultConfig(),
		Execution:         execution.NewDefaultConfig(defaultStoreDirPath),
		Processor:         processor.NewDefaultConfig(),
		API:               api.NewDefaultConfig(),
		Accounts:          accounts.NewDefaultConfig(),
		Orders:            orders.NewDefaultConfig(),
		Time:              vegatime.NewDefaultConfig(),
		Markets:           markets.NewDefaultConfig(),
		Matching:          matching.NewDefaultConfig(),
		Parties:           parties.NewDefaultConfig(),
		Candles:           candles.NewDefaultConfig(),
		Risk:              risk.NewDefaultConfig(),
		Storage:           storage.NewDefaultConfig(defaultStoreDirPath),
		Pprof:             pprof.NewDefaultConfig(),
		Monitoring:        monitoring.NewDefaultConfig(),
		Logging:           logging.NewDefaultConfig(),
		Gateway:           gateway.NewDefaultConfig(),
		Position:          positions.NewDefaultConfig(),
		Settlement:        settlement.NewDefaultConfig(),
		Collateral:        collateral.NewDefaultConfig(),
		Metrics:           metrics.NewDefaultConfig(),
		Transfers:         transfers.NewDefaultConfig(),
		Governance:        governance.NewDefaultConfig(),
		NodeWallet:        nodewallet.NewDefaultConfig(defaultStoreDirPath),
		Assets:            assets.NewDefaultConfig(defaultStoreDirPath),
		Notary:            notary.NewDefaultConfig(),
		EvtForward:        evtforward.NewDefaultConfig(),
		Genesis:           genesis.NewDefaultConfig(),
		Validators:        validators.NewDefaultConfig(),
		Banking:           banking.NewDefaultConfig(),
		Stats:             stats.NewDefaultConfig(),
		Subscribers:       subscribers.NewDefaultConfig(),
		NetworkParameters: netparams.NewDefaultConfig(),
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
