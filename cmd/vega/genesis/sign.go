package genesis

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"

	"code.vegaprotocol.io/go-wallet/wallet"
	storev1 "code.vegaprotocol.io/go-wallet/wallet/store/v1"
	"code.vegaprotocol.io/vega/config"
	"code.vegaprotocol.io/vega/genesis"
	"code.vegaprotocol.io/vega/logging"
)

type signCmd struct {
	TmRoot             string            `short:"t" long:"tm-root" description:"The root path of tendermint"`
	WalletRoot         string            `long:"wallet-path" description:"The root path to the wallets"`
	WalletName         string            `long:"wallet-name" description:"The name of the wallet to use" required:"true"`
	WalletPubKey       string            `long:"wallet-pubkey" description:"The public key of the wallet used for the signing" required:"true"`
	WalletPasswordFile config.Passphrase `long:"wallet-password" description:"A file containing the passphrase for the wallet, if empty will prompt for input"`
}

func (opts *signCmd) Execute(_ []string) error {
	log := logging.NewLoggerFromConfig(
		logging.NewDefaultConfig(),
	)
	defer log.AtExit()

	pass, err := opts.WalletPasswordFile.Get("wallet")
	if err != nil {
		return err
	}

	sps, err := genesis.GetSignedParameters(os.ExpandEnv(opts.TmRoot))
	if err != nil {
		return err
	}

	jsonSps, err := json.Marshal(sps)
	if err != nil {
		return err
	}

	store, err := storev1.NewStore(os.ExpandEnv(opts.WalletRoot))
	if err != nil {
		return err
	}

	err = store.Initialise()
	if err != nil {
		return err
	}

	handler := wallet.NewHandler(store)

	err = handler.LoginWallet(opts.WalletName, pass)
	if err != nil {
		return err
	}

	signature, err := handler.SignAny(opts.WalletName, jsonSps, opts.WalletPubKey)
	if err != nil {
		return err
	}

	fmt.Println("Here is the signature of the signed parameters:")
	fmt.Println(hex.EncodeToString(signature))

	return nil
}
