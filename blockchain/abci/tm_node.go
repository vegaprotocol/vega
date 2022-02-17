package abci

import (
	"fmt"

	"code.vegaprotocol.io/vega/blockchain"
	"github.com/spf13/viper"
	abciclient "github.com/tendermint/tendermint/abci/client"
	"github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/config"
	tmlog "github.com/tendermint/tendermint/libs/log"
	"github.com/tendermint/tendermint/libs/service"
	nm "github.com/tendermint/tendermint/node"
	tmtypes "github.com/tendermint/tendermint/types"
)

type TmNode struct {
	node service.Service
}

func (s *TmNode) ReloadConf(cfg blockchain.Config) {
}

func (t *TmNode) GetClient() (*LocalClient, error) {
	return newLocalClient(t.node)
}

func (t *TmNode) Start() error {
	return t.node.Start()
}

func (t *TmNode) Stop() error {
	if err := t.node.Stop(); err != nil {
		return err
	}
	t.node.Wait()
	return nil
}

func NewTmNode(
	homeDir string,
	app types.Application,
	genesisDoc *tmtypes.GenesisDoc,
) (*TmNode, error) {
	defer func() { fmt.Printf("\n\n\n\nFUUUUUUUUUUUUUUUUUUUUUUUUUUU\n\n\n\n") }()
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

	acc := abciclient.NewLocalCreator(app)
	logger := tmlog.MustNewDefaultLogger(tmlog.LogFormatPlain, tmlog.LogLevelInfo, false)
	node, err := nm.New(config, logger, acc, genesisDoc)
	if err != nil {
		return nil, fmt.Errorf("creating tendermint node: %v", err)
	}

	return &TmNode{node}, nil
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
