package genesis

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"

	"code.vegaprotocol.io/vegawallet/wallets"
	"code.vegaprotocol.io/vega/config"
	"code.vegaprotocol.io/vega/genesis"
	"code.vegaprotocol.io/vega/logging"
)

type signCmd struct {
	config.VegaHomeFlag
	TmRoot           string            `short:"t" long:"tm-root" description:"The root path of tendermint"`
	WalletName       string            `long:"wallet-name" description:"The name of the wallet to use" required:"true"`
	WalletPassphrase config.Passphrase `long:"wallet-passphrase-file" description:"A file containing the passphrase for the wallet, if empty will prompt for input"`
}

func (opts *signCmd) Execute(_ []string) error {
	log := logging.NewLoggerFromConfig(
		logging.NewDefaultConfig(),
	)
	defer log.AtExit()

	pass, err := opts.WalletPassphrase.Get("wallet", false)
	if err != nil {
		return err
	}

	_, genesisState, err := genesis.GetLocalGenesisState(os.ExpandEnv(opts.TmRoot))
	if err != nil {
		return err
	}

	sps, err := genesis.GetSignedParameters(genesisState)
	if err != nil {
		return err
	}

	jsonSps, err := json.Marshal(sps)
	if err != nil {
		return err
	}

	store, err := wallets.InitialiseStore(opts.VegaHome)
	if err != nil {
		return err
	}

	handler := wallets.NewHandler(store)

	err = handler.LoginWallet(opts.WalletName, pass)
	if err != nil {
		return err
	}

	signature, err := handler.SignAny(opts.WalletName, jsonSps, genesis.PubKey)
	if err != nil {
		return err
	}

	fmt.Println("Here is the signature of the signed parameters:")
	fmt.Println(hex.EncodeToString(signature))

	return nil
}
