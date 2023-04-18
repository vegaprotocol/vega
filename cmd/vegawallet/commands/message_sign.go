package cmd

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"

	"code.vegaprotocol.io/vega/cmd/vegawallet/commands/cli"
	"code.vegaprotocol.io/vega/cmd/vegawallet/commands/flags"
	"code.vegaprotocol.io/vega/cmd/vegawallet/commands/printer"
	"code.vegaprotocol.io/vega/wallet/api"
	"code.vegaprotocol.io/vega/wallet/wallets"

	"github.com/spf13/cobra"
)

var (
	signMessageLong = cli.LongDesc(`
		Sign any message using a Vega wallet key.
	`)

	signMessageExample = cli.Examples(`
		# Sign a message
		{{.Software}} message sign --message MESSAGE --wallet WALLET --pubkey PUBKEY
	`)
)

type SignMessageHandler func(api.AdminSignMessageParams, string) (api.AdminSignMessageResult, error)

func NewCmdSignMessage(w io.Writer, rf *RootFlags) *cobra.Command {
	h := func(params api.AdminSignMessageParams, passphrase string) (api.AdminSignMessageResult, error) {
		ctx := context.Background()

		walletStore, err := wallets.InitialiseStore(rf.Home, false)
		if err != nil {
			return api.AdminSignMessageResult{}, fmt.Errorf("couldn't initialise wallets store: %w", err)
		}
		defer walletStore.Close()

		if _, errDetails := api.NewAdminUnlockWallet(walletStore).Handle(ctx, api.AdminUnlockWalletParams{
			Wallet:     params.Wallet,
			Passphrase: passphrase,
		}); errDetails != nil {
			return api.AdminSignMessageResult{}, errors.New(errDetails.Data)
		}

		rawResult, errorDetails := api.NewAdminSignMessage(walletStore).Handle(ctx, params)
		if errorDetails != nil {
			return api.AdminSignMessageResult{}, errors.New(errorDetails.Data)
		}
		return rawResult.(api.AdminSignMessageResult), nil
	}
	return BuildCmdSignMessage(w, h, rf)
}

func BuildCmdSignMessage(w io.Writer, handler SignMessageHandler, rf *RootFlags) *cobra.Command {
	f := &SignMessageFlags{}

	cmd := &cobra.Command{
		Use:     "sign",
		Short:   "Sign a message using a Vega wallet key",
		Long:    signMessageLong,
		Example: signMessageExample,
		RunE: func(_ *cobra.Command, _ []string) error {
			req, pass, err := f.Validate()
			if err != nil {
				return err
			}

			resp, err := handler(req, pass)
			if err != nil {
				return err
			}

			switch rf.Output {
			case flags.InteractiveOutput:
				PrintSignMessageResponse(w, resp)
			case flags.JSONOutput:
				return printer.FprintJSON(w, struct {
					Signature string `json:"signature"`
				}{
					Signature: resp.Base64Signature,
				})
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
		"Public key to use to the sign the message (hex-encoded)",
	)
	cmd.Flags().StringVarP(&f.Message,
		"message", "m",
		"",
		"Message to be verified (base64-encoded)",
	)
	cmd.Flags().StringVarP(&f.PassphraseFile,
		"passphrase-file", "p",
		"",
		"Path to the file containing the wallet's passphrase",
	)

	autoCompleteWallet(cmd, rf.Home, "wallet")

	return cmd
}

type SignMessageFlags struct {
	Wallet         string
	PubKey         string
	Message        string
	PassphraseFile string
}

func (f *SignMessageFlags) Validate() (api.AdminSignMessageParams, string, error) {
	req := api.AdminSignMessageParams{}

	if len(f.Wallet) == 0 {
		return api.AdminSignMessageParams{}, "", flags.MustBeSpecifiedError("wallet")
	}
	req.Wallet = f.Wallet

	if len(f.PubKey) == 0 {
		return api.AdminSignMessageParams{}, "", flags.MustBeSpecifiedError("pubkey")
	}
	req.PubKey = f.PubKey

	if len(f.Message) == 0 {
		return api.AdminSignMessageParams{}, "", flags.MustBeSpecifiedError("message")
	}
	_, err := base64.StdEncoding.DecodeString(f.Message)
	if err != nil {
		return api.AdminSignMessageParams{}, "", flags.MustBase64EncodedError("message")
	}
	req.EncodedMessage = f.Message

	passphrase, err := flags.GetPassphrase(f.PassphraseFile)
	if err != nil {
		return api.AdminSignMessageParams{}, "", err
	}

	return req, passphrase, nil
}

func PrintSignMessageResponse(w io.Writer, req api.AdminSignMessageResult) {
	p := printer.NewInteractivePrinter(w)

	str := p.String()
	defer p.Print(str)

	str.CheckMark().SuccessText("Message signature successful").NextSection()
	str.Text("Signature (base64-encoded):").NextLine().WarningText(req.Base64Signature).NextSection()

	str.BlueArrow().InfoText("Sign a message").NextLine()
	str.Text("To verify a message, see the following command:").NextSection()
	str.Code(fmt.Sprintf("%s verify --help", os.Args[0])).NextLine()
}
