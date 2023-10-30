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

package genesis

import (
	"encoding/base64"
	"fmt"
	"time"

	"code.vegaprotocol.io/vega/core/genesis"
	"code.vegaprotocol.io/vega/core/nodewallets"
	vgtm "code.vegaprotocol.io/vega/core/tendermint"
	"code.vegaprotocol.io/vega/core/validators"
	vgrand "code.vegaprotocol.io/vega/libs/rand"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/paths"

	"github.com/jessevdk/go-flags"
	tmtypes "github.com/tendermint/tendermint/types"
)

type generateCmd struct {
	Config nodewallets.Config

	DryRun bool   `description:"Display the genesis file without writing it" long:"dry-run"`
	TmHome string `description:"The home path of tendermint"                 long:"tm-home" short:"t"`
}

func (opts *generateCmd) Execute(_ []string) error {
	log := logging.NewLoggerFromConfig(
		logging.NewDefaultConfig(),
	)
	defer log.AtExit()

	pass, err := genesisCmd.PassphraseFile.Get("node wallet", false)
	if err != nil {
		return err
	}

	vegaPaths := paths.New(genesisCmd.VegaHome)

	if _, err := flags.NewParser(opts, flags.Default|flags.IgnoreUnknown).Parse(); err != nil {
		return err
	}

	vegaKey, ethAddress, walletID, err := loadNodeWalletPubKey(opts.Config, vegaPaths, pass)
	if err != nil {
		return err
	}

	tmConfig := vgtm.NewConfig(opts.TmHome)

	pubKey, err := tmConfig.PublicValidatorKey()
	if err != nil {
		return err
	}

	b64TmPubKey := base64.StdEncoding.EncodeToString(pubKey.Bytes())
	genesisState := genesis.DefaultState()
	genesisState.Validators[base64.StdEncoding.EncodeToString(pubKey.Bytes())] = validators.ValidatorData{
		ID:              walletID,
		VegaPubKey:      vegaKey.value,
		VegaPubKeyIndex: vegaKey.index,
		EthereumAddress: ethAddress,
		TmPubKey:        b64TmPubKey,
	}

	genesisDoc := &tmtypes.GenesisDoc{
		ChainID:         fmt.Sprintf("test-chain-%v", vgrand.RandomStr(6)),
		GenesisTime:     time.Now().Round(0).UTC(),
		ConsensusParams: tmtypes.DefaultConsensusParams(),
		Validators: []tmtypes.GenesisValidator{
			{
				Address: pubKey.Address(),
				PubKey:  pubKey,
				Power:   10,
			},
		},
	}

	if err = vgtm.AddAppStateToGenesis(genesisDoc, &genesisState); err != nil {
		return fmt.Errorf("couldn't add app_state to genesis: %w", err)
	}

	if !opts.DryRun {
		if err := tmConfig.SaveGenesis(genesisDoc); err != nil {
			return fmt.Errorf("couldn't save genesis: %w", err)
		}
	}

	prettifiedDoc, err := vgtm.Prettify(genesisDoc)
	if err != nil {
		return err
	}
	fmt.Println(prettifiedDoc)
	return nil
}

type vegaPubKey struct {
	index uint32
	value string
}

func loadNodeWalletPubKey(config nodewallets.Config, vegaPaths paths.Paths, registryPass string) (vegaKey *vegaPubKey, ethAddr, walletID string, err error) {
	nw, err := nodewallets.GetNodeWallets(config, vegaPaths, registryPass)
	if err != nil {
		return nil, "", "", fmt.Errorf("couldn't get node wallets: %w", err)
	}

	if err := nw.Verify(); err != nil {
		return nil, "", "", err
	}

	vegaPubKey := &vegaPubKey{
		index: nw.Vega.Index(),
		value: nw.Vega.PubKey().Hex(),
	}

	return vegaPubKey, nw.Ethereum.PubKey().Hex(), nw.Vega.ID().Hex(), nil
}
