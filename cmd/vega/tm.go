package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"

	"github.com/jessevdk/go-flags"
	"github.com/spf13/cobra"

	cmd "github.com/tendermint/tendermint/cmd/tendermint/commands"
	"github.com/tendermint/tendermint/cmd/tendermint/commands/debug"
	cfg "github.com/tendermint/tendermint/config"
	"github.com/tendermint/tendermint/libs/cli"
	tmlog "github.com/tendermint/tendermint/libs/log"
	tmos "github.com/tendermint/tendermint/libs/os"
	nm "github.com/tendermint/tendermint/node"
	"github.com/tendermint/tendermint/p2p"
	"github.com/tendermint/tendermint/privval"
	"github.com/tendermint/tendermint/proxy"
	tmtypes "github.com/tendermint/tendermint/types"
)

var (
	networkSelect string
)

type tmCmd struct{}

func (opts *tmCmd) Execute(_ []string) error {

	os.Args = os.Args[1:]
	rootCmd := cmd.RootCmd
	rootCmd.AddCommand(
		cmd.GenValidatorCmd,
		cmd.InitFilesCmd,
		cmd.ProbeUpnpCmd,
		cmd.LightCmd,
		cmd.ReplayCmd,
		cmd.ReplayConsoleCmd,
		cmd.ResetAllCmd,
		cmd.ResetPrivValidatorCmd,
		cmd.ShowValidatorCmd,
		cmd.TestnetFilesCmd,
		cmd.ShowNodeIDCmd,
		cmd.GenNodeKeyCmd,
		cmd.VersionCmd,
		debug.DebugCmd,
		cli.NewCompletionCmd(rootCmd, true),
	)

	nodeFunc := defaultNewNode
	rootCmd.AddCommand(newRunNodeCmd(nodeFunc))

	c := cli.PrepareBaseCmd(rootCmd, "TM", os.ExpandEnv(filepath.Join("$HOME", cfg.DefaultTendermintDir)))
	if err := c.Execute(); err != nil {
		panic(err)
	}

	return nil
}

func defaultNewNode(config *cfg.Config, logger tmlog.Logger) (*nm.Node, error) {
	nodeKey, err := p2p.LoadOrGenNodeKey(config.NodeKeyFile())
	if err != nil {
		return nil, fmt.Errorf("failed to load or gen node key %s: %w", config.NodeKeyFile(), err)
	}

	return nm.NewNode(config,
		privval.LoadOrGenFilePV(config.PrivValidatorKeyFile(), config.PrivValidatorStateFile()),
		nodeKey,
		proxy.DefaultClientCreator(config.ProxyApp, config.ABCI, config.DBDir()),
		selectGenesisDocProviderFunc(config),
		nm.DefaultDBProvider,
		nm.DefaultMetricsProvider(config.Instrumentation),
		logger,
	)
}

func selectGenesisDocProviderFunc(config *cfg.Config) nm.GenesisDocProvider {
	if len(networkSelect) > 0 {
		return httpGenesisDocProvider
	}

	return nm.DefaultGenesisDocProviderFunc(config)
}

func httpGenesisDocProvider() (*tmtypes.GenesisDoc, error) {
	resp, err := http.Get(fmt.Sprintf("https://raw.githubusercontent.com/vegaprotocol/networks/master/%s/genesis.json", networkSelect))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	jsonGenesis, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return tmtypes.GenesisDocFromJSON(jsonGenesis)
}

// this is taken from tendermint
func newRunNodeCmd(nodeProvider nm.Provider) *cobra.Command {
	logger := tmlog.NewTMLogger(tmlog.NewSyncWriter(os.Stdout))
	cobraCmd := &cobra.Command{
		Use:     "start",
		Aliases: []string{"node", "run"},
		Short:   "Run the tendermint node",
		RunE: func(_ *cobra.Command, args []string) error {
			config, err := cmd.ParseConfig()
			if err != nil {
				return err
			}

			n, err := nodeProvider(config, logger)
			if err != nil {
				return fmt.Errorf("failed to create node: %w", err)
			}

			if err := n.Start(); err != nil {
				return fmt.Errorf("failed to start node: %w", err)
			}

			logger.Info("Started node", "nodeInfo", n.Switch().NodeInfo())

			// Stop upon receiving SIGTERM or CTRL-C.
			tmos.TrapSignal(logger, func() {
				if n.IsRunning() {
					if err := n.Stop(); err != nil {
						logger.Error("unable to stop the node", "error", err)
					}
				}
			})

			// Run forever.
			select {}
		},
	}

	cobraCmd.Flags().StringVar(
		&networkSelect,
		"network",
		"",
		"The network to start this node with")

	cmd.AddNodeFlags(cobraCmd)
	return cobraCmd
}

func Tm(ctx context.Context, parser *flags.Parser) error {
	_, err := parser.AddCommand(
		"tm",
		"Run tendermint nodes",
		"Run a tendermint node",
		&tmCmd{},
	)

	return err
}
