package abci

import (
	"fmt"

	"code.vegaprotocol.io/vega/core/blockchain"
	"code.vegaprotocol.io/vega/logging"

	"github.com/spf13/viper"
	"github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/config"
	"github.com/tendermint/tendermint/libs/service"
	nm "github.com/tendermint/tendermint/node"
	"github.com/tendermint/tendermint/p2p"
	"github.com/tendermint/tendermint/privval"
	"github.com/tendermint/tendermint/proxy"
	tmtypes "github.com/tendermint/tendermint/types"
)

type TmNode struct {
	conf blockchain.Config
	node service.Service
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
	if err := config.ValidateBasic(); err != nil {
		return nil, fmt.Errorf("invalid tendermint configuration data: %v", err)
	}

	if genesisDoc == nil {
		genesisDoc, err = tmtypes.GenesisDocFromFile(config.GenesisFile())
		if err != nil {
			return nil, fmt.Errorf("loading tendermint genesis document: %v", err)
		}
	}

	genesisDocProvider := func() (*tmtypes.GenesisDoc, error) {
		return genesisDoc, nil
	}

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
	logger := newTmLogger(log)

	// create node
	node, err := nm.NewNode(
		config,
		pv,
		nodeKey,
		proxy.NewLocalClientCreator(app),
		genesisDocProvider,
		nm.DefaultDBProvider,
		nm.DefaultMetricsProvider(config.Instrumentation),
		logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create new Tendermint node: %w", err)
	}

	// acc := abciclient.NewLocalCreator(app)
	// node, err := nm.New(config, logger, acc, genesisDoc)
	// if err != nil {
	// 	return nil, fmt.Errorf("creating tendermint node: %v", err)
	// }

	return &TmNode{conf, node}, nil
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

func newTmLogger(log *logging.Logger) *TmLogger {
	// return tmlog.MustNewDefaultLogger(tmlog.LogFormatPlain, tmlog.LogLevelInfo, false)
	tmLogger := &TmLogger{log.ToSugared()}
	return tmLogger
}

func loadConfig(homeDir string) (*config.Config, error) {
	conf := config.DefaultConfig()
	if err := viper.Unmarshal(conf); err != nil {
		return nil, err
	}

	conf.SetRoot(homeDir)
	overwriteConfig(conf)
	if err := conf.ValidateBasic(); err != nil {
		return nil, fmt.Errorf("error in config file: %w", err)
	}

	return conf, nil
}

// we want to force validators to skip timeout on commit so they don't wait after consensus has been reached.
func overwriteConfig(config *config.Config) {
	config.Consensus.SkipTimeoutCommit = true
	config.Consensus.CreateEmptyBlocks = true
	// enforce using priority mempool
	config.Mempool.Version = "v1"
	// enforce compatibility
	config.P2P.MaxPacketMsgPayloadSize = 16384
}
