package genesis

import (
	"encoding/base64"
	"fmt"

	"code.vegaprotocol.io/shared/paths"
	"code.vegaprotocol.io/vega/genesis"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/nodewallets"
	vgtm "code.vegaprotocol.io/vega/tendermint"
	"code.vegaprotocol.io/vega/validators"
	"github.com/jessevdk/go-flags"
)

type updateCmd struct {
	Config nodewallets.Config

	DryRun bool   `long:"dry-run" description:"Display the genesis file without writing it"`
	TmHome string `short:"t" long:"tm-home" description:"The home path of tendermint"`
}

func (opts *updateCmd) Execute(_ []string) error {
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
	genesisState := genesis.DefaultGenesisState()
	genesisState.Validators[b64TmPubKey] = validators.ValidatorData{
		ID:              walletID,
		VegaPubKey:      vegaKey.value,
		VegaPubKeyIndex: vegaKey.index,
		EthereumAddress: ethAddress,
		TmPubKey:        b64TmPubKey,
	}

	genesisDoc, _, err := tmConfig.Genesis()
	if err != nil {
		return fmt.Errorf("couldn't get genesis file: %w", err)
	}

	if err := vgtm.AddAppStateToGenesis(genesisDoc, &genesisState); err != nil {
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
