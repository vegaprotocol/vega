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
	"errors"
	"fmt"

	"code.vegaprotocol.io/vega/core/nodewallets"
	vgtm "code.vegaprotocol.io/vega/core/tendermint"
	"code.vegaprotocol.io/vega/core/validators"
	vgjson "code.vegaprotocol.io/vega/libs/json"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/paths"

	tmjson "github.com/cometbft/cometbft/libs/json"
	tmtypes "github.com/cometbft/cometbft/types"
	"github.com/jessevdk/go-flags"
)

var ErrAppendAndReplaceAreMutuallyExclusive = errors.New("--append and --replace and mutually exclusive")

type newValidatorCmd struct {
	Config nodewallets.Config

	TmHome        string `description:"The home path of tendermint"                                                    long:"tm-home"    short:"t"`
	Country       string `description:"The country from which the validator operates"                                  long:"country"    required:"true"`
	InfoURL       string `description:"The URL from which people can get to know the validator"                        long:"info-url"   required:"true"`
	Name          string `description:"The name of the validator node"                                                 long:"name"       required:"true"`
	AvatarURL     string `description:"An URL to an avatar for the validator"                                          long:"avatar-url"`
	ShouldAppend  bool   `description:"Append the generated validator to the existing validators in the genesis file"  long:"append"`
	ShouldReplace bool   `description:"Replace the existing validators by the generated validator in the genesis file" long:"replace"`
}

func (opts *newValidatorCmd) Execute(_ []string) error {
	if err := opts.Validate(); err != nil {
		return err
	}

	log := logging.NewLoggerFromConfig(logging.NewDefaultConfig())
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

	validatorDataDoc := tmtypes.GenesisValidator{
		Address: pubKey.Address(),
		PubKey:  pubKey,
		Power:   10,
		Name:    opts.Name,
	}

	b64TmPubKey := base64.StdEncoding.EncodeToString(pubKey.Bytes())
	validatorDataState := validators.ValidatorData{
		ID:              walletID,
		VegaPubKey:      vegaKey.value,
		VegaPubKeyIndex: vegaKey.index,
		TmPubKey:        b64TmPubKey,
		EthereumAddress: ethAddress,
		Country:         opts.Country,
		InfoURL:         opts.InfoURL,
		Name:            opts.Name,
		AvatarURL:       opts.AvatarURL,
	}

	if !opts.ShouldAppend && !opts.ShouldReplace {
		marshalledGenesisDoc, err := tmjson.MarshalIndent(validatorDataDoc, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println("Info to add in genesis file under `validators` key")
		fmt.Println(string(marshalledGenesisDoc))

		fmt.Println("Info to add in genesis file under `app_state.validators` key")
		return vgjson.PrettyPrint(map[string]validators.ValidatorData{
			b64TmPubKey: validatorDataState,
		})
	}

	genesisDoc, genesisState, err := tmConfig.Genesis()
	if err != nil {
		return fmt.Errorf("couldn't get genesis file: %w", err)
	}

	if opts.ShouldAppend {
		if _, ok := genesisState.Validators[b64TmPubKey]; ok {
			return fmt.Errorf("validator with tendermint key \"%s\" is already registered under app_state.validators key", b64TmPubKey)
		}
		for _, validator := range genesisDoc.Validators {
			if validator.PubKey.Equals(pubKey) {
				return fmt.Errorf("validator with tendermint key \"%s\" is already registered under validators key", b64TmPubKey)
			}
		}
		genesisDoc.Validators = append(genesisDoc.Validators, validatorDataDoc)
		genesisState.Validators[b64TmPubKey] = validatorDataState
	} else if opts.ShouldReplace {
		genesisDoc.Validators = []tmtypes.GenesisValidator{validatorDataDoc}
		genesisState.Validators = map[string]validators.ValidatorData{
			b64TmPubKey: validatorDataState,
		}
	}

	if err = vgtm.AddAppStateToGenesis(genesisDoc, genesisState); err != nil {
		return fmt.Errorf("couldn't add app_state to genesis: %w", err)
	}

	if err := tmConfig.SaveGenesis(genesisDoc); err != nil {
		return fmt.Errorf("couldn't save genesis: %w", err)
	}

	prettifiedDoc, err := vgtm.Prettify(genesisDoc)
	if err != nil {
		return err
	}
	fmt.Println(prettifiedDoc)
	return err
}

func (opts *newValidatorCmd) Validate() error {
	if opts.ShouldAppend && opts.ShouldReplace {
		return ErrAppendAndReplaceAreMutuallyExclusive
	}
	return nil
}
