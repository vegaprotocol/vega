package genesis

import (
	"encoding/base64"
	"errors"
	"fmt"

	vgjson "code.vegaprotocol.io/shared/libs/json"
	"code.vegaprotocol.io/shared/paths"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/nodewallets"
	vgtm "code.vegaprotocol.io/vega/tendermint"
	"code.vegaprotocol.io/vega/validators"

	"github.com/jessevdk/go-flags"
	tmjson "github.com/tendermint/tendermint/libs/json"
	tmtypes "github.com/tendermint/tendermint/types"
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
	ShouldReplace bool   `long:"replace" description:"Replace the existing validators by the the generated validator in the genesis file"`
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
