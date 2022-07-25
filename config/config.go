// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

//lint:file-ignore SA5008 duplicated struct tags are ok for config

package config

import (
	"fmt"
	"os"

	vgfs "code.vegaprotocol.io/shared/libs/fs"
	"code.vegaprotocol.io/shared/paths"
	"code.vegaprotocol.io/vega/admin"
	"code.vegaprotocol.io/vega/api"
	"code.vegaprotocol.io/vega/assets"
	"code.vegaprotocol.io/vega/banking"
	"code.vegaprotocol.io/vega/blockchain"
	"code.vegaprotocol.io/vega/broker"
	"code.vegaprotocol.io/vega/checkpoint"
	"code.vegaprotocol.io/vega/client/eth"
	"code.vegaprotocol.io/vega/collateral"
	cfgencoding "code.vegaprotocol.io/vega/config/encoding"
	"code.vegaprotocol.io/vega/coreapi"
	"code.vegaprotocol.io/vega/delegation"
	"code.vegaprotocol.io/vega/epochtime"
	"code.vegaprotocol.io/vega/evtforward"
	"code.vegaprotocol.io/vega/execution"
	"code.vegaprotocol.io/vega/genesis"
	"code.vegaprotocol.io/vega/governance"
	"code.vegaprotocol.io/vega/libs/pprof"
	"code.vegaprotocol.io/vega/limits"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/metrics"
	"code.vegaprotocol.io/vega/monitoring"
	"code.vegaprotocol.io/vega/netparams"
	"code.vegaprotocol.io/vega/nodewallets"
	"code.vegaprotocol.io/vega/notary"
	"code.vegaprotocol.io/vega/oracles"
	"code.vegaprotocol.io/vega/pow"
	"code.vegaprotocol.io/vega/processor"
	"code.vegaprotocol.io/vega/rewards"
	"code.vegaprotocol.io/vega/snapshot"
	"code.vegaprotocol.io/vega/spam"
	"code.vegaprotocol.io/vega/staking"
	"code.vegaprotocol.io/vega/statevar"
	"code.vegaprotocol.io/vega/stats"
	"code.vegaprotocol.io/vega/validators"
	"code.vegaprotocol.io/vega/validators/erc20multisig"
	"code.vegaprotocol.io/vega/vegatime"
)

// Config ties together all other application configuration types.
type Config struct {
	Admin             admin.Config         `group:"Admin" namespace:"admin"`
	API               api.Config           `group:"API" namespace:"api"`
	Blockchain        blockchain.Config    `group:"Blockchain" namespace:"blockchain"`
	Collateral        collateral.Config    `group:"Collateral" namespace:"collateral"`
	CoreAPI           coreapi.Config       `group:"CoreAPI" namespace:"coreapi"`
	Execution         execution.Config     `group:"Execution" namespace:"execution"`
	Ethereum          eth.Config           `group:"Ethereum" namespace:"ethereum"`
	Processor         processor.Config     `group:"Processor" namespace:"processor"`
	Logging           logging.Config       `group:"Logging" namespace:"logging"`
	Oracles           oracles.Config       `group:"Oracles" namespace:"oracles"`
	Time              vegatime.Config      `group:"Time" namespace:"time"`
	Epoch             epochtime.Config     `group:"Epoch" namespace:"epochtime"`
	Monitoring        monitoring.Config    `group:"Monitoring" namespace:"monitoring"`
	Metrics           metrics.Config       `group:"Metrics" namespace:"metrics"`
	Governance        governance.Config    `group:"Governance" namespace:"governance"`
	NodeWallet        nodewallets.Config   `group:"NodeWallet" namespace:"nodewallet"`
	Assets            assets.Config        `group:"Assets" namespace:"assets"`
	Notary            notary.Config        `group:"Notary" namespace:"notary"`
	EvtForward        evtforward.Config    `group:"EvtForward" namespace:"evtForward"`
	Genesis           genesis.Config       `group:"Genesis" namespace:"genesis"`
	Validators        validators.Config    `group:"Validators" namespace:"validators"`
	Banking           banking.Config       `group:"Banking" namespace:"banking"`
	Stats             stats.Config         `group:"Stats" namespace:"stats"`
	NetworkParameters netparams.Config     `group:"NetworkParameters" namespace:"netparams"`
	Limits            limits.Config        `group:"Limits" namespace:"limits"`
	Checkpoint        checkpoint.Config    `group:"Checkpoint" namespace:"checkpoint"`
	Staking           staking.Config       `group:"Staking" namespace:"staking"`
	Broker            broker.Config        `group:"Broker" namespace:"broker"`
	Rewards           rewards.Config       `group:"Rewards" namespace:"rewards"`
	Delegation        delegation.Config    `group:"Delegation" namespace:"delegation"`
	Spam              spam.Config          `group:"Spam" namespace:"spam"`
	PoW               pow.Config           `group:"ProofOfWork" namespace:"pow"`
	Snapshot          snapshot.Config      `group:"Snapshot" namespace:"snapshot"`
	StateVar          statevar.Config      `group:"StateVar" namespace:"statevar"`
	ERC20MultiSig     erc20multisig.Config `group:"ERC20MultiSig" namespace:"erc20multisig"`

	Pprof        pprof.Config         `group:"Pprof" namespace:"pprof"`
	NodeMode     cfgencoding.NodeMode `long:"mode" description:"The mode of the vega node [validator, full]"`
	UlimitNOFile uint64               `long:"ulimit-no-files" description:"Set the max number of open files (see: ulimit -n)" tomlcp:"Set the max number of open files (see: ulimit -n)"`
}

// NewDefaultConfig returns a set of default configs for all vega packages, as specified at the per package
// config level, if there is an error initialising any of the configs then this is returned.
func NewDefaultConfig() Config {
	return Config{
		NodeMode:          cfgencoding.NodeModeValidator,
		Admin:             admin.NewDefaultConfig(),
		API:               api.NewDefaultConfig(),
		CoreAPI:           coreapi.NewDefaultConfig(),
		Blockchain:        blockchain.NewDefaultConfig(),
		Execution:         execution.NewDefaultConfig(),
		Ethereum:          eth.NewDefaultConfig(),
		Processor:         processor.NewDefaultConfig(),
		Oracles:           oracles.NewDefaultConfig(),
		Time:              vegatime.NewDefaultConfig(),
		Epoch:             epochtime.NewDefaultConfig(),
		Pprof:             pprof.NewDefaultConfig(),
		Monitoring:        monitoring.NewDefaultConfig(),
		Logging:           logging.NewDefaultConfig(),
		Collateral:        collateral.NewDefaultConfig(),
		Metrics:           metrics.NewDefaultConfig(),
		Governance:        governance.NewDefaultConfig(),
		NodeWallet:        nodewallets.NewDefaultConfig(),
		Assets:            assets.NewDefaultConfig(),
		Notary:            notary.NewDefaultConfig(),
		EvtForward:        evtforward.NewDefaultConfig(),
		Genesis:           genesis.NewDefaultConfig(),
		Validators:        validators.NewDefaultConfig(),
		Banking:           banking.NewDefaultConfig(),
		Stats:             stats.NewDefaultConfig(),
		NetworkParameters: netparams.NewDefaultConfig(),
		Limits:            limits.NewDefaultConfig(),
		Checkpoint:        checkpoint.NewDefaultConfig(),
		Staking:           staking.NewDefaultConfig(),
		Broker:            broker.NewDefaultConfig(),
		UlimitNOFile:      8192,
		Snapshot:          snapshot.NewDefaultConfig(),
		StateVar:          statevar.NewDefaultConfig(),
		ERC20MultiSig:     erc20multisig.NewDefaultConfig(),
		PoW:               pow.NewDefaultConfig(),
	}
}

func (c Config) IsValidator() bool {
	return c.NodeMode == cfgencoding.NodeModeValidator
}

func (c Config) HaveEthClient() bool {
	if c.Blockchain.ChainProvider == blockchain.ProviderNullChain {
		return false
	}
	return c.IsValidator()
}

type Loader struct {
	configFilePath string
}

func InitialiseLoader(vegaPaths paths.Paths) (*Loader, error) {
	configFilePath, err := vegaPaths.CreateConfigPathFor(paths.NodeDefaultConfigFile)
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
