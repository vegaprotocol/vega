// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.VEGA file and at https://www.mariadb.com/bsl11.
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
	"errors"
	"fmt"
	"os"

	"code.vegaprotocol.io/vega/core/admin"
	"code.vegaprotocol.io/vega/core/api"
	"code.vegaprotocol.io/vega/core/assets"
	"code.vegaprotocol.io/vega/core/banking"
	"code.vegaprotocol.io/vega/core/blockchain"
	"code.vegaprotocol.io/vega/core/broker"
	"code.vegaprotocol.io/vega/core/checkpoint"
	"code.vegaprotocol.io/vega/core/client/eth"
	"code.vegaprotocol.io/vega/core/collateral"
	cfgencoding "code.vegaprotocol.io/vega/core/config/encoding"
	"code.vegaprotocol.io/vega/core/coreapi"
	"code.vegaprotocol.io/vega/core/delegation"
	"code.vegaprotocol.io/vega/core/epochtime"
	"code.vegaprotocol.io/vega/core/evtforward"
	"code.vegaprotocol.io/vega/core/execution"
	"code.vegaprotocol.io/vega/core/genesis"
	"code.vegaprotocol.io/vega/core/governance"
	"code.vegaprotocol.io/vega/core/limits"
	"code.vegaprotocol.io/vega/core/metrics"
	"code.vegaprotocol.io/vega/core/netparams"
	"code.vegaprotocol.io/vega/core/nodewallets"
	"code.vegaprotocol.io/vega/core/notary"
	"code.vegaprotocol.io/vega/core/oracles"
	"code.vegaprotocol.io/vega/core/pow"
	"code.vegaprotocol.io/vega/core/processor"
	"code.vegaprotocol.io/vega/core/protocolupgrade"
	"code.vegaprotocol.io/vega/core/rewards"
	"code.vegaprotocol.io/vega/core/snapshot"
	"code.vegaprotocol.io/vega/core/spam"
	"code.vegaprotocol.io/vega/core/staking"
	"code.vegaprotocol.io/vega/core/statevar"
	"code.vegaprotocol.io/vega/core/stats"
	"code.vegaprotocol.io/vega/core/validators"
	"code.vegaprotocol.io/vega/core/validators/erc20multisig"
	"code.vegaprotocol.io/vega/core/vegatime"
	vgfs "code.vegaprotocol.io/vega/libs/fs"
	"code.vegaprotocol.io/vega/libs/pprof"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/paths"
)

// Config ties together all other application configuration types.
type Config struct {
	Admin             admin.Config           `group:"Admin"             namespace:"admin"`
	API               api.Config             `group:"API"               namespace:"api"`
	Blockchain        blockchain.Config      `group:"Blockchain"        namespace:"blockchain"`
	Collateral        collateral.Config      `group:"Collateral"        namespace:"collateral"`
	CoreAPI           coreapi.Config         `group:"CoreAPI"           namespace:"coreapi"`
	Execution         execution.Config       `group:"Execution"         namespace:"execution"`
	Ethereum          eth.Config             `group:"Ethereum"          namespace:"ethereum"`
	Processor         processor.Config       `group:"Processor"         namespace:"processor"`
	Logging           logging.Config         `group:"Logging"           namespace:"logging"`
	Oracles           oracles.Config         `group:"Oracles"           namespace:"oracles"`
	Time              vegatime.Config        `group:"Time"              namespace:"time"`
	Epoch             epochtime.Config       `group:"Epoch"             namespace:"epochtime"`
	Metrics           metrics.Config         `group:"Metrics"           namespace:"metrics"`
	Governance        governance.Config      `group:"Governance"        namespace:"governance"`
	NodeWallet        nodewallets.Config     `group:"NodeWallet"        namespace:"nodewallet"`
	Assets            assets.Config          `group:"Assets"            namespace:"assets"`
	Notary            notary.Config          `group:"Notary"            namespace:"notary"`
	EvtForward        evtforward.Config      `group:"EvtForward"        namespace:"evtForward"`
	Genesis           genesis.Config         `group:"Genesis"           namespace:"genesis"`
	Validators        validators.Config      `group:"Validators"        namespace:"validators"`
	Banking           banking.Config         `group:"Banking"           namespace:"banking"`
	Stats             stats.Config           `group:"Stats"             namespace:"stats"`
	NetworkParameters netparams.Config       `group:"NetworkParameters" namespace:"netparams"`
	Limits            limits.Config          `group:"Limits"            namespace:"limits"`
	Checkpoint        checkpoint.Config      `group:"Checkpoint"        namespace:"checkpoint"`
	Staking           staking.Config         `group:"Staking"           namespace:"staking"`
	Broker            broker.Config          `group:"Broker"            namespace:"broker"`
	Rewards           rewards.Config         `group:"Rewards"           namespace:"rewards"`
	Delegation        delegation.Config      `group:"Delegation"        namespace:"delegation"`
	Spam              spam.Config            `group:"Spam"              namespace:"spam"`
	PoW               pow.Config             `group:"ProofOfWork"       namespace:"pow"`
	Snapshot          snapshot.Config        `group:"Snapshot"          namespace:"snapshot"`
	StateVar          statevar.Config        `group:"StateVar"          namespace:"statevar"`
	ERC20MultiSig     erc20multisig.Config   `group:"ERC20MultiSig"     namespace:"erc20multisig"`
	ProtocolUpgrade   protocolupgrade.Config `group:"ProtocolUpgrade"   namespace:"protocolupgrade"`
	Pprof             pprof.Config           `group:"Pprof"             namespace:"pprof"`

	NodeMode         cfgencoding.NodeMode `description:"The mode of the vega node [validator, full]"                            long:"mode"`
	MaxMemoryPercent uint8                `description:"The maximum amount of memory reserved for the vega node (default: 33%)" long:"max-memory-percent"`
}

// NewDefaultConfig returns a set of default configs for all vega packages, as specified at the per package
// config level, if there is an error initialising any of the configs then this is returned.
func NewDefaultConfig() Config {
	return Config{
		NodeMode:          cfgencoding.NodeModeValidator,
		MaxMemoryPercent:  33,
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
		Snapshot:          snapshot.NewDefaultConfig(),
		StateVar:          statevar.NewDefaultConfig(),
		ERC20MultiSig:     erc20multisig.NewDefaultConfig(),
		PoW:               pow.NewDefaultConfig(),
		ProtocolUpgrade:   protocolupgrade.NewDefaultConfig(),
	}
}

func (c Config) IsValidator() bool {
	return c.NodeMode == cfgencoding.NodeModeValidator
}

func (c *Config) SetDefaultMaxMemoryPercent() {
	// disable restriction if node is a validator
	if c.NodeMode == cfgencoding.NodeModeValidator {
		c.MaxMemoryPercent = 100
	}
}

func (c Config) GetMaxMemoryFactor() (float64, error) {
	if c.MaxMemoryPercent <= 0 || c.MaxMemoryPercent > 100 {
		return 0, errors.New("MaxMemoryPercent is out of range, expect > 0 and <= 100")
	}

	return float64(c.MaxMemoryPercent) / 100., nil
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
