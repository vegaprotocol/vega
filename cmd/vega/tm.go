package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"code.vegaprotocol.io/vega/genesis"
	"github.com/jessevdk/go-flags"
	"github.com/spf13/cobra"
	tmcmd "github.com/tendermint/tendermint/cmd/tendermint/commands"
	"github.com/tendermint/tendermint/cmd/tendermint/commands/debug"
	cfg "github.com/tendermint/tendermint/config"
	"github.com/tendermint/tendermint/libs/cli"
	tmflags "github.com/tendermint/tendermint/libs/cli/flags"
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
	rootCmd := tmcmd.RootCmd
	rootCmd.AddCommand(
		tmcmd.GenValidatorCmd,
		tmcmd.InitFilesCmd,
		tmcmd.ProbeUpnpCmd,
		tmcmd.LightCmd,
		tmcmd.ReplayCmd,
		tmcmd.ReplayConsoleCmd,
		tmcmd.ResetAllCmd,
		tmcmd.ResetPrivValidatorCmd,
		tmcmd.ShowValidatorCmd,
		tmcmd.TestnetFilesCmd,
		tmcmd.ShowNodeIDCmd,
		tmcmd.GenNodeKeyCmd,
		tmcmd.VersionCmd,
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
	genesisFilesRootPath := fmt.Sprintf("https://raw.githubusercontent.com/vegaprotocol/networks/master/%s", networkSelect)

	doc, state, err := getGenesisFromRemote(genesisFilesRootPath)
	if err != nil {
		return nil, err
	}

	sig, err := getSignatureFromRemote(genesisFilesRootPath)
	if err != nil {
		return nil, err
	}

	validSignature, err := genesis.VerifyGenesisStateSignature(state, sig)
	if err != nil {
		return nil, fmt.Errorf("couldn't verify the genesis state signature: %s", err)
	}
	if !validSignature {
		return nil, fmt.Errorf("genesis state doesn't match the signature: %s", sig)
	}

	return doc, nil
}

func getGenesisFromRemote(genesisFilesRootPath string) (*tmtypes.GenesisDoc, *genesis.GenesisState, error) {
	genesisFilePath := fmt.Sprintf("%s/genesis.json", genesisFilesRootPath)
	resp, err := http.Get(genesisFilePath)
	if err != nil {
		return nil, nil, fmt.Errorf("couldn't load genesis file from %s: %w", genesisFilePath, err)
	}
	defer resp.Body.Close()
	jsonGenesis, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, err
	}

	doc, state, err := genesis.GenesisFromJSON(jsonGenesis)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid genesis file from %s: %w", genesisFilePath, err)
	}
	return doc, state, nil
}

func getSignatureFromRemote(genesisFilesRootPath string) (string, error) {
	signatureFilePath := fmt.Sprintf("%s/signature.txt", genesisFilesRootPath)
	sigResp, err := http.Get(signatureFilePath)
	if err != nil {
		return "", fmt.Errorf("couldn't load signature file from %s: %w", signatureFilePath, err)
	}
	defer sigResp.Body.Close()
	sig, err := ioutil.ReadAll(sigResp.Body)
	if err != nil {
		return "", err
	}
	return strings.Trim(string(sig), "\n"), nil
}

// this is taken from tendermint
func newRunNodeCmd(nodeProvider nm.Provider) *cobra.Command {
	logger := tmlog.NewTMLogger(tmlog.NewSyncWriter(os.Stdout))
	cobraCmd := &cobra.Command{
		Use:     "start",
		Aliases: []string{"node", "run"},
		Short:   "Run the tendermint node",
		RunE: func(cmd *cobra.Command, args []string) error {
			if cmd.Name() == tmcmd.VersionCmd.Name() {
				return nil
			}

			config, err := tmcmd.ParseConfig()
			if err != nil {
				return err
			}

			if config.LogFormat == cfg.LogFormatJSON {
				logger = tmlog.NewTMJSONLogger(tmlog.NewSyncWriter(os.Stdout))
			}

			logger, err = tmflags.ParseLogLevel(config.LogLevel, logger, cfg.DefaultLogLevel)
			if err != nil {
				return err
			}

			logger = logger.With("module", "main")

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

	tmcmd.AddNodeFlags(cobraCmd)
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
