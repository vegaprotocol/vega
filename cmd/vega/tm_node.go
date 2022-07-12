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

package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"code.vegaprotocol.io/vega/genesis"
	"github.com/spf13/cobra"
	tmcmd "github.com/tendermint/tendermint/cmd/tendermint/commands"
	cfg "github.com/tendermint/tendermint/config"
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
	networkSelect        string
	networkSelectFromURL string
)

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
	} else if len(networkSelectFromURL) > 0 {
		return genesisDocHTTPFromURL
	}

	return nm.DefaultGenesisDocProviderFunc(config)
}

func genesisDocHTTPFromURL() (*tmtypes.GenesisDoc, error) {
	genesisFilePath := networkSelectFromURL

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "GET", genesisFilePath, nil)
	if err != nil {
		return nil, fmt.Errorf("couldn't load genesis file from %s: %w", genesisFilePath, err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("couldn't load genesis file from %s: %w", genesisFilePath, err)
	}
	defer resp.Body.Close()
	jsonGenesis, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	doc, _, err := genesis.GenesisFromJSON(jsonGenesis)
	if err != nil {
		return nil, fmt.Errorf("invalid genesis file from %s: %w", genesisFilePath, err)
	}

	return doc, nil
}

func httpGenesisDocProvider() (*tmtypes.GenesisDoc, error) {
	genesisFilesRootPath := fmt.Sprintf("https://raw.githubusercontent.com/vegaprotocol/networks/master/%s", networkSelect)

	doc, _, err := getGenesisFromRemote(genesisFilesRootPath)

	return doc, err
}

func getGenesisFromRemote(genesisFilesRootPath string) (*tmtypes.GenesisDoc, *genesis.GenesisState, error) {
	genesisFilePath := fmt.Sprintf("%s/genesis.json", genesisFilesRootPath)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "GET", genesisFilePath, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("couldn't load genesis file from %s: %w", genesisFilePath, err)
	}
	resp, err := http.DefaultClient.Do(req)
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

// this is taken from tendermint.
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
		&networkSelectFromURL,
		"network-url",
		"",
		"The URL to a genesis file to start this node with")
	cobraCmd.Flags().StringVar(
		&networkSelect,
		"network",
		"",
		"The network to start this node with")

	tmcmd.AddNodeFlags(cobraCmd)
	return cobraCmd
}
