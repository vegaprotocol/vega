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

	TmHome        string `short:"t" long:"tm-home" description:"The home path of tendermint"`
	Country       string `long:"country" description:"The country from which the validator operates" required:"true"`
	InfoURL       string `long:"info-url" description:"The URL from which people can get to know the validator" required:"true"`
	Name          string `long:"name" description:"The name of the validator node" required:"true"`
	AvatarURL     string `long:"avatar-url" description:"An URL to an avatar for the validator"`
	ShouldAppend  bool   `long:"append" description:"Append the generated validator to the existing validators in the genesis file"`
	ShouldReplace bool   `long:"replace" description:"Replace the existing validators by the generated validator in the genesis file"`
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
