package genesis

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"

	vgrand "code.vegaprotocol.io/shared/libs/rand"
	"code.vegaprotocol.io/shared/paths"
	"code.vegaprotocol.io/vega/genesis"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/nodewallets"
	"code.vegaprotocol.io/vega/validators"

	"github.com/jessevdk/go-flags"
	tmconfig "github.com/tendermint/tendermint/config"
	tmcrypto "github.com/tendermint/tendermint/crypto"
	tmjson "github.com/tendermint/tendermint/libs/json"
	tmos "github.com/tendermint/tendermint/libs/os"
	"github.com/tendermint/tendermint/privval"
	tmtypes "github.com/tendermint/tendermint/types"
	tmtime "github.com/tendermint/tendermint/types/time"
)

type generateCmd struct {
	Config nodewallets.Config

	DryRun bool   `long:"dry-run" description:"Display the genesis file without writing it"`
	TmHome string `short:"t" long:"tm-home" description:"The home path of tendermint"`
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
		return fmt.Errorf("couldn't marshal genesis state as JSON: %w", err)
	}

	genesisDoc := tmtypes.GenesisDoc{
		ChainID:         fmt.Sprintf("test-chain-%v", vgrand.RandomStr(6)),
		GenesisTime:     tmtime.Now(),
		ConsensusParams: tmtypes.DefaultConsensusParams(),
		AppState:        rawGenesisState,
		Validators: []tmtypes.GenesisValidator{
			{
				Address: pubKey.Address(),
				PubKey:  pubKey,
				Power:   10,
			},
		},
	}

	if !opts.DryRun {
		genesisFilePath := tmConfig.GenesisFile()
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

func loadTendermintPrivateValidatorKey(tmConfig *tmconfig.Config) (tmcrypto.PubKey, error) {
	privValKeyFile := tmConfig.PrivValidatorKeyFile()
	privValStateFile := tmConfig.PrivValidatorStateFile()
	if !tmos.FileExists(privValKeyFile) {
		return nil, fmt.Errorf("file \"%s\" not found", privValKeyFile)
	}

	pv := privval.LoadFilePV(privValKeyFile, privValStateFile)

	pubKey, err := pv.GetPubKey()
	if err != nil {
		return nil, fmt.Errorf("can't get pubkey: %w", err)
	}

	return pubKey, nil
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
