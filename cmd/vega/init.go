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
	"errors"
	"fmt"
	"os"
	"time"

	"code.vegaprotocol.io/vega/core/config"
	"code.vegaprotocol.io/vega/core/config/encoding"
	"code.vegaprotocol.io/vega/core/nodewallets/registry"
	vgjson "code.vegaprotocol.io/vega/libs/json"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/paths"

	"github.com/jessevdk/go-flags"
	tmcfg "github.com/tendermint/tendermint/config"
	tmos "github.com/tendermint/tendermint/libs/os"
	tmrand "github.com/tendermint/tendermint/libs/rand"
	"github.com/tendermint/tendermint/p2p"
	"github.com/tendermint/tendermint/privval"
	"github.com/tendermint/tendermint/types"
)

type InitCmd struct {
	config.VegaHomeFlag
	config.OutputFlag
	config.Passphrase `long:"nodewallet-passphrase-file"`

	Force bool `short:"f" long:"force" description:"Erase exiting vega configuration at the specified path"`

	NoTendermint   bool   `long:"no-tendermint" description:"Disable tendermint configuration generation"`
	TendermintHome string `long:"tendermint-home" required:"true" description:"Directory for tendermint config and data" default:"$HOME/.tendermint"`
	TendermintKey  string `long:"tendermint-key" description:"Key type to generate privval file with" choice:"ed25519" choice:"secp256k1" default:"ed25519"`
}

var initCmd InitCmd

func (opts *InitCmd) Usage() string {
	return "<full | validator>"
}

func (opts *InitCmd) Execute(args []string) error {
	logger := logging.NewLoggerFromConfig(logging.NewDefaultConfig())
	defer logger.AtExit()

	if len(args) != 1 {
		return errors.New("require exactly 1 parameter mode, expected modes [validator, full, seed]")
	}

	mode, err := encoding.NodeModeFromString(args[0])
	if err != nil {
		return err
	}

	output, err := opts.GetOutput()
	if err != nil {
		return err
	}

	vegaPaths := paths.New(opts.VegaHome)

	// a nodewallet will be required only for a validator node
	var nwRegistry *registry.Loader
	if mode == encoding.NodeModeValidator {
		pass, err := opts.Get("node wallet", true)
		if err != nil {
			return err
		}

		nwRegistry, err = registry.NewLoader(vegaPaths, pass)
		if err != nil {
			return err
		}
	}

	cfgLoader, err := config.InitialiseLoader(vegaPaths)
	if err != nil {
		return fmt.Errorf("couldn't initialise configuration loader: %w", err)
	}

	configExists, err := cfgLoader.ConfigExists()
	if err != nil {
		return fmt.Errorf("couldn't verify configuration presence: %w", err)
	}

	if configExists && !opts.Force {
		return fmt.Errorf("configuration already exists at `%s` please remove it first or re-run using -f", cfgLoader.ConfigFilePath())
	}

	if configExists && opts.Force {
		if output.IsHuman() {
			logger.Info("removing existing configuration", logging.String("path", cfgLoader.ConfigFilePath()))
		}
		cfgLoader.Remove()
	}

	cfg := config.NewDefaultConfig()
	cfg.NodeMode = mode
	cfg.SetDefaultMaxMemoryPercent()

	if err := cfgLoader.Save(&cfg); err != nil {
		return fmt.Errorf("couldn't save configuration file: %w", err)
	}

	if output.IsHuman() {
		logger.Info("configuration generated successfully",
			logging.String("path", cfgLoader.ConfigFilePath()))
	}

	if !initCmd.NoTendermint {
		tmCfg := tmcfg.DefaultConfig()
		tmCfg.SetRoot(os.ExpandEnv(initCmd.TendermintHome))
		tmcfg.EnsureRoot(tmCfg.RootDir)
		if err := initTendermintConfiguration(output, logger, tmCfg); err != nil {
			return fmt.Errorf("couldn't initialise tendermint %w", err)
		}
	}

	if output.IsJSON() {
		if mode == encoding.NodeModeValidator {
			return vgjson.Print(struct {
				ConfigFilePath           string `json:"configFilePath"`
				NodeWalletConfigFilePath string `json:"nodeWalletConfigFilePath"`
			}{
				ConfigFilePath:           cfgLoader.ConfigFilePath(),
				NodeWalletConfigFilePath: nwRegistry.RegistryFilePath(),
			})
		}
		return vgjson.Print(struct {
			ConfigFilePath string `json:"configFilePath"`
		}{
			ConfigFilePath: cfgLoader.ConfigFilePath(),
		})
	}

	return nil
}

func initTendermintConfiguration(output config.Output, logger *logging.Logger, config *tmcfg.Config) error {
	// private validator
	privValKeyFile := config.PrivValidatorKeyFile()
	privValStateFile := config.PrivValidatorStateFile()
	var pv *privval.FilePV
	if tmos.FileExists(privValKeyFile) {
		pv = privval.LoadFilePV(privValKeyFile, privValStateFile)
		if output.IsHuman() {
			logger.Info("Found private validator",
				logging.String("keyFile", privValKeyFile),
				logging.String("stateFile", privValStateFile),
			)
		}
	} else {
		pv = privval.GenFilePV(privValKeyFile, privValStateFile)
		pv.Save()
		if output.IsHuman() {
			logger.Info("Generated private validator",
				logging.String("keyFile", privValKeyFile),
				logging.String("stateFile", privValStateFile),
			)
		}
	}

	nodeKeyFile := config.NodeKeyFile()
	if tmos.FileExists(nodeKeyFile) {
		if output.IsHuman() {
			logger.Info("Found node key", logging.String("path", nodeKeyFile))
		}
	} else {
		if _, err := p2p.LoadOrGenNodeKey(nodeKeyFile); err != nil {
			return err
		}
		if output.IsHuman() {
			logger.Info("Generated node key", logging.String("path", nodeKeyFile))
		}
	}

	// genesis file
	genFile := config.GenesisFile()
	if tmos.FileExists(genFile) {
		if output.IsHuman() {
			logger.Info("Found genesis file", logging.String("path", genFile))
		}
	} else {
		genDoc := types.GenesisDoc{
			ChainID:         fmt.Sprintf("test-chain-%v", tmrand.Str(6)),
			GenesisTime:     time.Now().Round(0).UTC(),
			ConsensusParams: types.DefaultConsensusParams(),
		}
		pubKey, err := pv.GetPubKey()
		if err != nil {
			return fmt.Errorf("can't get pubkey: %w", err)
		}
		genDoc.Validators = []types.GenesisValidator{{
			Address: pubKey.Address(),
			PubKey:  pubKey,
			Power:   10,
		}}

		if err := genDoc.SaveAs(genFile); err != nil {
			return err
		}
		if output.IsHuman() {
			logger.Info("Generated genesis file", logging.String("path", genFile))
		}
	}

	return nil
}

func Init(ctx context.Context, parser *flags.Parser) error {
	initCmd = InitCmd{}

	var (
		short = "Initializes a vega node"
		long  = "Generate the minimal configuration required for a vega node to start. You must specify 'full' or 'validator'"
	)
	_, err := parser.AddCommand("init", short, long, &initCmd)
	return err
}
