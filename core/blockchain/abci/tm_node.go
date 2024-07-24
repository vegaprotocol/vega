// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package abci

import (
	"context"
	"fmt"

	"code.vegaprotocol.io/vega/core/blockchain"
	"code.vegaprotocol.io/vega/logging"

	"github.com/cometbft/cometbft/abci/types"
	"github.com/cometbft/cometbft/config"
	bftconfig "github.com/cometbft/cometbft/config"
	bftlog "github.com/cometbft/cometbft/libs/log"
	nm "github.com/cometbft/cometbft/node"
	"github.com/cometbft/cometbft/p2p"
	"github.com/cometbft/cometbft/privval"
	"github.com/cometbft/cometbft/proxy"
	tmtypes "github.com/cometbft/cometbft/types"
	"github.com/spf13/viper"
)

// Service defines a service that can be started, stopped, and reset.
type Service interface {
	// Start the service.
	// If it's already started or stopped, will return an error.
	// If OnStart() returns an error, it's returned by Start()
	Start() error
	OnStart() error

	// Stop the service.
	// If it's already stopped, will return an error.
	// OnStop must never error.
	Stop() error
	OnStop()

	// Reset the service.
	// Panics by default - must be overwritten to enable reset.
	Reset() error
	OnReset() error

	// Return true if the service is running
	IsRunning() bool

	// Quit returns a channel, which is closed once service is stopped.
	Quit() <-chan struct{}

	// String representation of the service
	String() string

	// SetLogger sets a logger.
	SetLogger(l bftlog.Logger)
}

type TmNode struct {
	conf        blockchain.Config
	node        Service
	MempoolSize int64
}

const namedLogger = "tendermint"

func NewTmNode(
	conf blockchain.Config,
	log *logging.Logger,
	homeDir string,
	app types.Application,
	genesisDoc *tmtypes.GenesisDoc,
) (*TmNode, error) {
	log = log.Named(namedLogger)
	log.SetLevel(conf.Tendermint.Level.Get())

	config, err := loadConfig(homeDir)
	if err != nil {
		return nil, err
	}

	viper.SetConfigFile(fmt.Sprintf("%s/%s", homeDir, "config/config.toml"))
	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("reading tendermint config: %v", err)
	}
	if err := viper.Unmarshal(config); err != nil {
		return nil, fmt.Errorf("decoding tendermint config: %v", err)
	}

	overwriteConfig(config)

	if err := config.ValidateBasic(); err != nil {
		return nil, fmt.Errorf("invalid tendermint configuration data: %v", err)
	}

	genesisDocProvider := nm.DefaultGenesisDocProviderFunc(config)

	// read private validator
	pv := privval.LoadFilePV(
		config.PrivValidatorKeyFile(),
		config.PrivValidatorStateFile(),
	)

	// read node key
	nodeKey, err := p2p.LoadNodeKey(config.NodeKeyFile())
	if err != nil {
		return nil, fmt.Errorf("failed to load node's key: %w", err)
	}

	// create logger
	logger := &TmLogger{log.ToSugared()}

	// create node
	node, err := nm.NewNode(
		context.Background(),
		config,
		pv,
		nodeKey,
		proxy.NewLocalClientCreator(app),
		genesisDocProvider,
		bftconfig.DefaultDBProvider,
		nm.DefaultMetricsProvider(config.Instrumentation),
		logger,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create new Tendermint node: %w", err)
	}

	// acc := abciclient.NewLocalCreator(app)
	// node, err := nm.New(config, logger, acc, genesisDoc)
	// if err != nil {
	// 	return nil, fmt.Errorf("creating tendermint node: %v", err)
	// }

	return &TmNode{conf, node, config.Mempool.MaxTxsBytes}, nil
}

func (*TmNode) ReloadConf(cfg blockchain.Config) {
}

func (t *TmNode) GetClient() (*LocalClient, error) {
	return newLocalClient(t.node)
}

func (t *TmNode) Start() error {
	return t.node.Start()
}

func (t *TmNode) Stop() error {
	if t.node != nil && t.node.IsRunning() {
		if err := t.node.Stop(); err != nil {
			return err
		}
	}
	<-t.node.Quit()
	return nil
}

func loadConfig(homeDir string) (*config.Config, error) {
	conf := config.DefaultConfig()
	if err := viper.Unmarshal(conf); err != nil {
		return nil, err
	}

	conf.SetRoot(homeDir)
	if err := conf.ValidateBasic(); err != nil {
		return nil, fmt.Errorf("error in config file: %w", err)
	}

	return conf, nil
}

// we want to force validators to skip timeout on commit so they don't wait after consensus has been reached.
func overwriteConfig(config *config.Config) {
	config.Consensus.TimeoutCommit = 0
	config.Consensus.CreateEmptyBlocks = true
	// ensure rechecking tx is enabled
	config.Mempool.Recheck = true
	// enforce compatibility
	config.P2P.MaxPacketMsgPayloadSize = 16384
}
