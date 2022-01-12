package genesis

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"

	vgfs "code.vegaprotocol.io/shared/libs/fs"
	"code.vegaprotocol.io/shared/paths"
	"code.vegaprotocol.io/vega/genesis"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/nodewallets"
	"code.vegaprotocol.io/vega/validators"

	"github.com/jessevdk/go-flags"
	tmconfig "github.com/tendermint/tendermint/config"
	tmjson "github.com/tendermint/tendermint/libs/json"
	"github.com/tendermint/tendermint/types"
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

	// genesis file
	tmConfig := tmconfig.DefaultConfig()
	tmConfig.SetRoot(os.ExpandEnv(opts.TmHome))

	pubKey, err := loadTendermintPrivateValidatorKey(tmConfig)
	if err != nil {
		return err
	}
	b64TmPubKey := base64.StdEncoding.EncodeToString(pubKey.Bytes())
	genesisState := genesis.DefaultGenesisState()
	genesisState.Validators[base64.StdEncoding.EncodeToString(pubKey.Bytes())] = validators.ValidatorData{
		ID:              walletID,
		VegaPubKey:      vegaKey.value,
		VegaPubKeyIndex: vegaKey.index,
		EthereumAddress: ethAddress,
		TmPubKey:        b64TmPubKey,
	}

	rawGenesisState, err := json.Marshal(genesisState)
	if err != nil {
		return fmt.Errorf("couldn't marshal the genesis state as JSON: %w", err)
	}

	genesisFilePath := tmConfig.GenesisFile()
	data, err := vgfs.ReadFile(genesisFilePath)
	if err != nil {
		return err
	}

	genesisDoc := &types.GenesisDoc{}
	err = tmjson.Unmarshal(data, genesisDoc)
	if err != nil {
		return fmt.Errorf("couldn't unmarshal the genesis document: %w", err)
	}

	genesisDoc.AppState = rawGenesisState

	if !opts.DryRun {
		log.Infof("Saving genesis doc at %s", genesisFilePath)
		if err := genesisDoc.SaveAs(genesisFilePath); err != nil {
			return fmt.Errorf("couldn't save the genesis file: %w", err)
		}
	}

	marshalledGenesisDoc, err := tmjson.MarshalIndent(genesisDoc, "", "  ")
	if err != nil {
		return fmt.Errorf("couldn't marshal the genesis document as JSON: %w", err)
	}
	fmt.Println(string(marshalledGenesisDoc))
	return err
}
