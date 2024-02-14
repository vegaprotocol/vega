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

package faucet

import (
	"fmt"

	"code.vegaprotocol.io/vega/core/config"
	"code.vegaprotocol.io/vega/core/faucet"
	vgjson "code.vegaprotocol.io/vega/libs/json"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/paths"
)

type faucetInit struct {
	config.VegaHomeFlag
	config.PassphraseFlag
	config.OutputFlag

	Force         bool `description:"Erase existing configuration at specified path"                long:"force"           short:"f"`
	UpdateInPlace bool `description:"Update the Vega node configuration with the faucet public key" long:"update-in-place"`
}

func (opts *faucetInit) Execute(_ []string) error {
	logDefaultConfig := logging.NewDefaultConfig()
	log := logging.NewLoggerFromConfig(logDefaultConfig)
	defer log.AtExit()

	output, err := opts.OutputFlag.GetOutput()
	if err != nil {
		return err
	}

	pass, err := opts.PassphraseFile.Get("faucet wallet", true)
	if err != nil {
		return err
	}

	vegaPaths := paths.New(opts.VegaHome)

	initResult, err := faucet.Initialise(vegaPaths, pass, opts.Force)
	if err != nil {
		return fmt.Errorf("couldn't initialise faucet: %w", err)
	}

	var nodeCfgFilePath string
	if opts.UpdateInPlace {
		nodeCfgLoader, nodeCfg, err := config.EnsureNodeConfig(vegaPaths)
		if err != nil {
			return err
		}

		// add the faucet public key to the allowlist
		nodeCfg.EvtForward.BlockchainQueueAllowlist = append(
			nodeCfg.EvtForward.BlockchainQueueAllowlist, initResult.Wallet.PublicKey)

		nodeCfg.SecondaryEvtForward.BlockchainQueueAllowlist = append(
			nodeCfg.SecondaryEvtForward.BlockchainQueueAllowlist, initResult.Wallet.PublicKey)

		if err := nodeCfgLoader.Save(nodeCfg); err != nil {
			return fmt.Errorf("couldn't update node configuration: %w", err)
		}

		nodeCfgFilePath = nodeCfgLoader.ConfigFilePath()
	}

	result := struct {
		PublicKey            string `json:"publicKey"`
		NodeConfigFilePath   string `json:"nodeConfigFilePath,omitempty"`
		FaucetConfigFilePath string `json:"faucetConfigFilePath"`
		FaucetWalletFilePath string `json:"faucetWalletFilePath"`
	}{
		NodeConfigFilePath:   nodeCfgFilePath,
		FaucetConfigFilePath: initResult.ConfigFilePath,
		FaucetWalletFilePath: initResult.Wallet.FilePath,
		PublicKey:            initResult.Wallet.PublicKey,
	}

	if output.IsHuman() {
		log.Info("faucet initialised successfully", logging.String("public-key", initResult.Wallet.PublicKey))
		err := vgjson.PrettyPrint(result)
		if err != nil {
			return fmt.Errorf("couldn't pretty print result: %w", err)
		}
	} else if output.IsJSON() {
		return vgjson.Print(result)
	}

	return nil
}
