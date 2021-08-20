//lint:file-ignore SA5008 duplicated struct tags are ok for config

package config

import (
	"io/ioutil"
	"path/filepath"

	"code.vegaprotocol.io/vega/assets"
	"code.vegaprotocol.io/vega/banking"
	"code.vegaprotocol.io/vega/blockchain"
	"code.vegaprotocol.io/vega/broker"
	"code.vegaprotocol.io/vega/checkpoint"
	"code.vegaprotocol.io/vega/collateral"
	"code.vegaprotocol.io/vega/delegation"
	"code.vegaprotocol.io/vega/epochtime"
	"code.vegaprotocol.io/vega/evtforward"
	"code.vegaprotocol.io/vega/execution"
	"code.vegaprotocol.io/vega/genesis"
	"code.vegaprotocol.io/vega/governance"
	"code.vegaprotocol.io/vega/libs/pprof"
	"code.vegaprotocol.io/vega/limits"
	"code.vegaprotocol.io/vega/liquidity"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/matching"
	"code.vegaprotocol.io/vega/metrics"
	"code.vegaprotocol.io/vega/monitoring"
	"code.vegaprotocol.io/vega/netparams"
	"code.vegaprotocol.io/vega/nodewallet"
	"code.vegaprotocol.io/vega/notary"
	"code.vegaprotocol.io/vega/oracles"
	"code.vegaprotocol.io/vega/positions"
	"code.vegaprotocol.io/vega/processor"
	"code.vegaprotocol.io/vega/rewards"
	"code.vegaprotocol.io/vega/risk"
	"code.vegaprotocol.io/vega/settlement"
	"code.vegaprotocol.io/vega/staking"
	"code.vegaprotocol.io/vega/stats"
	"code.vegaprotocol.io/vega/subscribers"
	"code.vegaprotocol.io/vega/validators"
	"code.vegaprotocol.io/vega/vegatime"

	"github.com/zannen/toml"
)

// Config ties together all other application configuration types.
type Config struct {
	Blockchain        blockchain.Config  `group:"Blockchain" namespace:"blockchain"`
	Collateral        collateral.Config  `group:"Collateral" namespace:"collateral"`
	Execution         execution.Config   `group:"Execution" namespace:"execution"`
	Processor         processor.Config   `group:"Processor" namespace:"processor"`
	Logging           logging.Config     `group:"Logging" namespace:"logging"`
	Matching          matching.Config    `group:"Matching" namespace:"matching"`
	Oracles           oracles.Config     `group:"Oracles" namespace:"oracles"`
	Liquidity         liquidity.Config   `group:"Liquidity" namespace:"liquidity"`
	Position          positions.Config   `group:"Position" namespace:"position"`
	Risk              risk.Config        `group:"Risk" namespace:"risk"`
	Settlement        settlement.Config  `group:"Settlement" namespace:"settlement"`
	Time              vegatime.Config    `group:"Time" namespace:"time"`
	Epoch             epochtime.Config   `group:"Epoch" namespace:"epochtime"`
	Monitoring        monitoring.Config  `group:"Monitoring" namespace:"monitoring"`
	Metrics           metrics.Config     `group:"Metrics" namespace:"metrics"`
	Governance        governance.Config  `group:"Governance" namespace:"governance"`
	NodeWallet        nodewallet.Config  `group:"NodeWallet" namespace:"nodewallet"`
	Assets            assets.Config      `group:"Assets" namespace:"assets"`
	Notary            notary.Config      `group:"Notary" namespace:"notary"`
	EvtForward        evtforward.Config  `group:"EvtForward" namespace:"evtForward"`
	Subscribers       subscribers.Config `group:"Subscribers" namespace:"subscribers"`
	Genesis           genesis.Config     `group:"Genesis" namespace:"genesis"`
	Validators        validators.Config  `group:"Validators" namespace:"validators"`
	Banking           banking.Config     `group:"Banking" namespace:"banking"`
	Stats             stats.Config       `group:"Stats" namespace:"stats"`
	NetworkParameters netparams.Config   `group:"NetworkParameters" namespace:"netparams"`
	Limits            limits.Config      `group:"Limits" namespace:"limits"`
	Checkpoint        checkpoint.Config  `group:"Checkpoint" namespace:"checkpoint"`
	Staking           staking.Config     `group:"Staking" namespace:"staking"`
	Broker            broker.Config      `group:"Broker" namespace:"broker"`
	Rewards           rewards.Config     `group:"Rewards" namespace:"rewards"`
	Delegation        delegation.Config  `group:"Delegation" namespace:"delegation"`

	Pprof        pprof.Config `group:"Pprof" namespace:"pprof"`
	UlimitNOFile uint64       `long:"ulimit-no-files" description:"Set the max number of open files (see: ulimit -n)" tomlcp:"Set the max number of open files (see: ulimit -n)"`
}

// NewDefaultConfig returns a set of default configs for all vega packages, as specified at the per package
// config level, if there is an error initialising any of the configs then this is returned.
func NewDefaultConfig(defaultStoreDirPath string) Config {
	return Config{
		Blockchain:        blockchain.NewDefaultConfig(),
		Execution:         execution.NewDefaultConfig(defaultStoreDirPath),
		Processor:         processor.NewDefaultConfig(defaultStoreDirPath),
		Oracles:           oracles.NewDefaultConfig(),
		Liquidity:         liquidity.NewDefaultConfig(),
		Time:              vegatime.NewDefaultConfig(),
		Epoch:             epochtime.NewDefaultConfig(),
		Matching:          matching.NewDefaultConfig(),
		Risk:              risk.NewDefaultConfig(),
		Pprof:             pprof.NewDefaultConfig(),
		Monitoring:        monitoring.NewDefaultConfig(),
		Logging:           logging.NewDefaultConfig(),
		Position:          positions.NewDefaultConfig(),
		Settlement:        settlement.NewDefaultConfig(),
		Collateral:        collateral.NewDefaultConfig(),
		Metrics:           metrics.NewDefaultConfig(),
		Governance:        governance.NewDefaultConfig(),
		NodeWallet:        nodewallet.NewDefaultConfig(),
		Assets:            assets.NewDefaultConfig(),
		Notary:            notary.NewDefaultConfig(),
		EvtForward:        evtforward.NewDefaultConfig(),
		Genesis:           genesis.NewDefaultConfig(),
		Validators:        validators.NewDefaultConfig(),
		Banking:           banking.NewDefaultConfig(),
		Stats:             stats.NewDefaultConfig(),
		Subscribers:       subscribers.NewDefaultConfig(),
		NetworkParameters: netparams.NewDefaultConfig(),
		Limits:            limits.NewDefaultConfig(),
		Checkpoint:        checkpoint.NewDefaultConfig(),
		Staking:           staking.NewDefaultConfig(),
		Broker:            broker.NewDefaultConfig(),
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
