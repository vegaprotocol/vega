package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"

	"code.vegaprotocol.io/vega/cmd/vegawallet/commands/cli"
	"code.vegaprotocol.io/vega/cmd/vegawallet/commands/flags"
	"code.vegaprotocol.io/vega/cmd/vegawallet/commands/printer"
	"code.vegaprotocol.io/vega/wallet/api"
	"code.vegaprotocol.io/vega/wallet/wallets"

	"github.com/spf13/cobra"
)

var (
	isolateKeyLong = cli.LongDesc(`
		Extract the specified key pair into an isolated wallet.

		An isolated wallet is a wallet that contains a single key pair and that
		has been stripped from its cryptographic node.

		Removing the cryptographic node from the wallet minimizes the impact of a
		stolen wallet, as it makes it impossible to retrieve or generate keys out
		of it.

		This creates a wallet that is only able to sign and verify transactions.

		This adds an extra layer of security.
	`)

	isolateKeyExample = cli.Examples(`
		# Isolate a key pair
		{{.Software}} key isolate --wallet WALLET --pubkey PUBKEY
	`)

	isolatedWalletPassphraseOptions = flags.PassphraseOptions{
		Name:        "isolated wallet",
		Description: "When isolating the wallet, you can choose a brand-new passphrase, or reuse the original one.",
	}
)

type IsolateKeyHandler func(api.AdminIsolateKeyParams) (isolateKeyResult, error)

type isolateKeyResult struct {
	Wallet   string `json:"wallet"`
	FilePath string `json:"filePath"`
}

func NewCmdIsolateKey(w io.Writer, rf *RootFlags) *cobra.Command {
	h := func(params api.AdminIsolateKeyParams) (isolateKeyResult, error) {
		walletStore, err := wallets.InitialiseStore(rf.Home)
		if err != nil {
			return isolateKeyResult{}, fmt.Errorf("could not initialise wallets store: %w", err)
		}
		defer walletStore.Close()

		isolateKey := api.NewAdminIsolateKey(walletStore)
		rawResult, errDetails := isolateKey.Handle(context.Background(), params)
		if errDetails != nil {
			return isolateKeyResult{}, errors.New(errDetails.Data)
		}
		result := rawResult.(api.AdminIsolateKeyResult)
		return isolateKeyResult{
			Wallet:   result.Wallet,
			FilePath: walletStore.GetWalletPath(result.Wallet),
		}, nil
	}

	return BuildCmdIsolateKey(w, h, rf)
}

func BuildCmdIsolateKey(w io.Writer, handler IsolateKeyHandler, rf *RootFlags) *cobra.Command {
	f := &IsolateKeyFlags{}

	cmd := &cobra.Command{
		Use:     "isolate",
		Short:   "Extract the specified key pair into an isolated wallet",
		Long:    isolateKeyLong,
		Example: isolateKeyExample,
		RunE: func(_ *cobra.Command, _ []string) error {
			req, err := f.Validate()
			if err != nil {
				return err
			}

			resp, err := handler(req)
			if err != nil {
				return err
			}

			switch rf.Output {
			case flags.InteractiveOutput:
				PrintIsolateKeyResponse(w, resp)
			case flags.JSONOutput:
				return printer.FprintJSON(w, resp)
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&f.Wallet,
		"wallet", "w",
		"",
		"Wallet holding the public key",
	)
	cmd.Flags().StringVarP(&f.PubKey,
		"pubkey", "k",
		"",
		"Public key to isolate (hex-encoded)",
	)
	cmd.Flags().StringVarP(&f.PassphraseFile,
		"passphrase-file", "p",
		"",
		"Path to the file containing the wallet's passphrase",
	)
	cmd.Flags().StringVar(&f.IsolatedWalletPassphraseFile,
		"isolated-wallet-passphrase-file",
		"",
		"Path to the file containing the new isolated wallet's passphrase",
	)

	autoCompleteWallet(cmd, rf.Home, "wallet")

	return cmd
}

type IsolateKeyFlags struct {
	Wallet                       string
	PubKey                       string
	PassphraseFile               string
	IsolatedWalletPassphraseFile string
}

func (f *IsolateKeyFlags) Validate() (api.AdminIsolateKeyParams, error) {
	if len(f.Wallet) == 0 {
		return api.AdminIsolateKeyParams{}, flags.MustBeSpecifiedError("wallet")
	}

	if len(f.PubKey) == 0 {
		return api.AdminIsolateKeyParams{}, flags.MustBeSpecifiedError("pubkey")
	}

	passphrase, err := flags.GetPassphrase(f.PassphraseFile)
	if err != nil {
		return api.AdminIsolateKeyParams{}, err
	}

	newPassphrase, err := flags.GetConfirmedPassphraseWithContext(isolatedWalletPassphraseOptions, f.IsolatedWalletPassphraseFile)
	if err != nil {
		return api.AdminIsolateKeyParams{}, err
	}

	return api.AdminIsolateKeyParams{
		Wallet:                   f.Wallet,
		PublicKey:                f.PubKey,
		Passphrase:               passphrase,
		IsolatedWalletPassphrase: newPassphrase,
	}, nil
}

func PrintIsolateKeyResponse(w io.Writer, resp isolateKeyResult) {
	p := printer.NewInteractivePrinter(w)

	str := p.String()
	defer p.Print(str)
	str.CheckMark().Text("Key pair has been isolated in wallet ").Bold(resp.Wallet).Text(" at: ").SuccessText(resp.FilePath).NextLine()
	str.CheckMark().SuccessText("Key isolation succeeded").NextLine()
}
