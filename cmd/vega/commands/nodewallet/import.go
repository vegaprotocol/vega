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

package nodewallet

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"

	"code.vegaprotocol.io/vega/core/config"
	"code.vegaprotocol.io/vega/core/nodewallets"
	vgfmt "code.vegaprotocol.io/vega/libs/fmt"
	vgfs "code.vegaprotocol.io/vega/libs/fs"
	vgjson "code.vegaprotocol.io/vega/libs/json"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/paths"

	tmconfig "github.com/cometbft/cometbft/config"
	"github.com/jessevdk/go-flags"
)

var (
	ErrOneOfTendermintFlagIsRequired       = errors.New("one of --tendermint-home or --tendermint-pubkey flag is required")
	ErrTendermintFlagsAreMutuallyExclusive = errors.New("--tendermint-home and --tendermint-pubkey are mutually exclusive")
	ErrClefOptionMissing                   = errors.New("--clef-account and --clef-address must both be set to import a clef wallet")
)

type importCmd struct {
	config.OutputFlag

	Config nodewallets.Config

	WalletPassphrase config.Passphrase `long:"wallet-passphrase-file"`

	Chain      string              `choice:"vega"                                                                      choice:"ethereum"  choice:"tendermint" description:"The chain to be imported" long:"chain" required:"true" short:"c"`
	WalletPath config.PromptString `description:"The path to the wallet file to import"                                long:"wallet-path"`
	Force      bool                `description:"Should the command re-write an existing nodewallet file if it exists" long:"force"`

	// clef flags
	EthereumClefAddress string `description:"The URL to the clef instance that Vega will use."               long:"ethereum-clef-address"`
	EthereumClefAccount string `description:"The Ethereum account to be imported by Vega from Clef. In hex." long:"ethereum-clef-account"`

	// tendermint flags
	TendermintPubkey string `description:"The tendermint pubkey of the tendermint validator node"    long:"tendermint-pubkey"`
	TendermintHome   string `description:"The tendermint home from which to look the for the pubkey" long:"tendermint-home"`
}

func (opts *importCmd) Execute(_ []string) error {
	output, err := opts.GetOutput()
	if err != nil {
		return err
	}

	log := logging.NewLoggerFromConfig(logging.NewDefaultConfig())
	defer log.AtExit()

	if (opts.EthereumClefAccount != "") != (opts.EthereumClefAddress != "") {
		return ErrClefOptionMissing
	}

	registryPass, err := rootCmd.PassphraseFile.Get("node wallet", false)
	if err != nil {
		return err
	}

	vegaPaths := paths.New(rootCmd.VegaHome)

	_, conf, err := config.EnsureNodeConfig(vegaPaths)
	if err != nil {
		return err
	}

	opts.Config = conf.NodeWallet

	if _, err := flags.NewParser(opts, flags.Default|flags.IgnoreUnknown).Parse(); err != nil {
		return err
	}

	var walletPass, walletPath string
	if opts.Chain == vegaChain || (opts.Chain == ethereumChain && opts.EthereumClefAddress == "") {
		walletPass, err = opts.WalletPassphrase.Get("blockchain wallet", false)
		if err != nil {
			return err
		}
		walletPath, err = opts.WalletPath.Get("wallet path", "wallet-path")
		if err != nil {
			return err
		}
	}

	var data map[string]string
	switch opts.Chain {
	case ethereumChain:
		data, err = nodewallets.ImportEthereumWallet(
			vegaPaths,
			registryPass,
			walletPass,
			opts.EthereumClefAccount,
			opts.EthereumClefAddress,
			walletPath,
			opts.Force,
		)
		if err != nil {
			return fmt.Errorf("couldn't import Ethereum node wallet: %w", err)
		}
	case vegaChain:
		data, err = nodewallets.ImportVegaWallet(vegaPaths, registryPass, walletPass, walletPath, opts.Force)
		if err != nil {
			return fmt.Errorf("couldn't import Vega node wallet: %w", err)
		}
	case tendermintChain:
		if len(opts.TendermintHome) == 0 && len(opts.TendermintPubkey) == 0 {
			return ErrOneOfTendermintFlagIsRequired
		}
		if len(opts.TendermintHome) > 0 && len(opts.TendermintPubkey) > 0 {
			return ErrTendermintFlagsAreMutuallyExclusive
		}

		tendermintPubkey := opts.TendermintPubkey
		if len(opts.TendermintHome) > 0 {
			tendermintPubkey, err = getLocalTendermintPubkey(opts.TendermintHome)
			if err != nil {
				return fmt.Errorf("couldn't retrieve tendermint public key: %w", err)
			}
		}

		// validate the key is base64
		_, err := base64.StdEncoding.DecodeString(tendermintPubkey)
		if err != nil {
			return fmt.Errorf("tendermint pubkey must be base64 encoded: %w", err)
		}

		data, err = nodewallets.ImportTendermintPubkey(vegaPaths, registryPass, tendermintPubkey, opts.Force)
		if err != nil {
			return fmt.Errorf("couldn't import Tendermint pubkey: %w", err)
		}
	}

	if output.IsHuman() {
		fmt.Println(green("import successful:"))
		vgfmt.PrettyPrint(data)
	} else if output.IsJSON() {
		if err := vgjson.Print(data); err != nil {
			return err
		}
	}

	return nil
}

func getLocalTendermintPubkey(tendermintHome string) (string, error) {
	tmConfig := tmconfig.DefaultConfig()
	tmConfig.SetRoot(tendermintHome)
	genesisFilePath := tmConfig.PrivValidatorKeyFile()

	data, err := vgfs.ReadFile(genesisFilePath)
	if err != nil {
		return "", err
	}

	privValidatorKey := struct {
		PubKey struct {
			Value string `json:"value"`
		} `json:"pub_key"`
	}{}

	if err = json.Unmarshal(data, &privValidatorKey); err != nil {
		return "", fmt.Errorf("could not read priv_validator_key.json: %w", err)
	}

	return privValidatorKey.PubKey.Value, nil
}
