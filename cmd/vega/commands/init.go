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

package commands

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
	vgrand "code.vegaprotocol.io/vega/libs/rand"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/paths"

	tmcfg "github.com/cometbft/cometbft/config"
	"github.com/cometbft/cometbft/p2p"
	"github.com/cometbft/cometbft/privval"
	"github.com/cometbft/cometbft/types"
	"github.com/jessevdk/go-flags"
)

type InitCmd struct {
	config.VegaHomeFlag
	config.OutputFlag
	config.Passphrase `long:"nodewallet-passphrase-file"`

	Force bool `description:"Erase existing vega configuration at the specified path" long:"force" short:"f"`

	NoTendermint   bool   `description:"Disable tendermint configuration generation" long:"no-tendermint"`
	TendermintHome string `default:"$HOME/.cometbft"                                 description:"Directory for tendermint config and data" long:"tendermint-home" required:"true"`
	TendermintKey  string `choice:"ed25519"                                          choice:"secp256k1"                                     default:"ed25519"      description:"Key type to generate privval file with" long:"tendermint-key"`
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

func FileExists(filePath string) bool {
	_, err := os.Stat(filePath)
	return !os.IsNotExist(err)
}

func initTendermintConfiguration(output config.Output, logger *logging.Logger, config *tmcfg.Config) error {
	// private validator
	privValKeyFile := config.PrivValidatorKeyFile()
	privValStateFile := config.PrivValidatorStateFile()
	var pv *privval.FilePV
	if FileExists(privValKeyFile) {
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
	if FileExists(nodeKeyFile) {
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
	if FileExists(genFile) {
		if output.IsHuman() {
			logger.Info("Found genesis file", logging.String("path", genFile))
		}
	} else {
		genDoc := types.GenesisDoc{
			ChainID:         fmt.Sprintf("test-chain-%v", vgrand.RandomStr(6)),
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
