package cmd

import (
	"context"
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
	createWalletLong = cli.LongDesc(`
		Create a wallet and generate the first Ed25519 key pair.

		You will be asked to create a passphrase. The passphrase is used to protect
		the file in which the keys are stored. This doesn't affect the key generation
		process in any way.
	`)

	createWalletExample = cli.Examples(`
		# Creating a wallet
		{{.Software}} create --wallet WALLET
	`)
)

type CreateWalletHandler func(api.CreateWalletParams) (api.CreateWalletResult, error)

func NewCmdCreateWallet(w io.Writer, rf *RootFlags) *cobra.Command {
	h := func(params api.CreateWalletParams) (api.CreateWalletResult, error) {
		s, err := wallets.InitialiseStore(rf.Home)
		if err != nil {
			return api.CreateWalletResult{}, fmt.Errorf("couldn't initialise wallets store: %w", err)
		}

		createWallet := api.NewCreateWallet(s)

		rawResult, errDetails := createWallet.Handle(context.Background(), params)
		if errDetails != nil {
			return api.CreateWalletResult{}, errors.New(errDetails.Data)
		}
		return rawResult.(api.CreateWalletResult), nil
	}

	return BuildCmdCreateWallet(w, h, rf)
}

func BuildCmdCreateWallet(w io.Writer, handler CreateWalletHandler, rf *RootFlags) *cobra.Command {
	f := &CreateWalletFlags{}

	cmd := &cobra.Command{
		Use:     "create",
		Short:   "Create a wallet",
		Long:    createWalletLong,
		Example: createWalletExample,
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
				PrintCreateWalletResponse(w, resp)
			case flags.JSONOutput:
				return printer.FprintJSON(w, resp)
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&f.Wallet,
		"wallet", "w",
		"",
		"The wallet where the key is generated in",
	)
	cmd.Flags().StringVarP(&f.PassphraseFile,
		"passphrase-file", "p",
		"",
		"Path to the file containing the wallet's passphrase",
	)

	return cmd
}

type CreateWalletFlags struct {
	Wallet         string
	PassphraseFile string
}

func (f *CreateWalletFlags) Validate() (api.CreateWalletParams, error) {
	req := api.CreateWalletParams{}

	if len(f.Wallet) == 0 {
		return api.CreateWalletParams{}, flags.FlagMustBeSpecifiedError("wallet")
	}
	req.Wallet = f.Wallet

	passphrase, err := flags.GetConfirmedPassphrase(f.PassphraseFile)
	if err != nil {
		return api.CreateWalletParams{}, err
	}
	req.Passphrase = passphrase

	return req, nil
}

func PrintCreateWalletResponse(w io.Writer, resp api.CreateWalletResult) {
	p := printer.NewInteractivePrinter(w)

	str := p.String()
	defer p.Print(str)

	str.CheckMark().Text("Wallet ").Bold(resp.Wallet.Name).Text(" has been created.").NextLine()
	str.CheckMark().Text("Its first key pair has been generated").NextLine()
	str.CheckMark().SuccessText("Creating wallet succeeded").NextSection()

	str.Text("Wallet recovery phrase:").NextLine()
	str.WarningText(resp.Wallet.RecoveryPhrase).NextLine()
	str.Text("Wallet version:").NextLine()
	str.WarningText(fmt.Sprintf("%d", resp.Wallet.Version)).NextLine()
	str.Text("First public key:").NextLine()
	str.WarningText(resp.Key.PublicKey).NextLine()
	str.NextSection()

	str.RedArrow().DangerText("Important").NextLine()
	str.Text("Write down the ").Bold("recovery phrase").Text(" and the ").Bold("wallet's version").Text(", and store it somewhere safe and secure, now.").NextLine()
	str.DangerText("The recovery phrase will not be displayed ever again, nor will you be able to retrieve it!").NextSection()

	str.BlueArrow().InfoText("Run the service").NextLine()
	str.Text("Now, you can run the service. See the following command:").NextSection()
	str.Code(fmt.Sprintf("%s service run --help", os.Args[0])).NextLine()
}
