package genesis

import (
	"encoding/base64"
	"fmt"

	vgrand "code.vegaprotocol.io/shared/libs/rand"
	"code.vegaprotocol.io/shared/paths"
	"code.vegaprotocol.io/vega/genesis"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/nodewallets"
	vgtm "code.vegaprotocol.io/vega/tendermint"
	"code.vegaprotocol.io/vega/validators"

	"github.com/jessevdk/go-flags"
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

	tmConfig := vgtm.NewConfig(opts.TmHome)

	pubKey, err := tmConfig.PublicValidatorKey()
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

	genesisDoc := &tmtypes.GenesisDoc{
		ChainID:         fmt.Sprintf("test-chain-%v", vgrand.RandomStr(6)),
		GenesisTime:     tmtime.Now(),
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
