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
	updatePassphraseLong = cli.LongDesc(`
		Update the passphrase of the specified wallet.
	`)

	updatePassphraseExample = cli.Examples(`
		# Update the passphrase of the specified wallet
		{{.Software}} passphrase update --wallet WALLET
	`)

	newWalletPassphraseOptions = flags.PassphraseOptions{
		Name: "new",
	}
)

type UpdatePassphraseHandler func(api.AdminUpdatePassphraseParams) error

func NewCmdUpdatePassphrase(w io.Writer, rf *RootFlags) *cobra.Command {
	h := func(params api.AdminUpdatePassphraseParams) error {
		walletStore, err := wallets.InitialiseStore(rf.Home)
		if err != nil {
			return fmt.Errorf("couldn't initialise wallets store: %w", err)
		}
		defer walletStore.Close()

		updatePassphrase := api.NewAdminUpdatePassphrase(walletStore)

		_, errDetails := updatePassphrase.Handle(context.Background(), params)
		if errDetails != nil {
			return errors.New(errDetails.Data)
		}
		return nil
	}

	return BuildCmdUpdatePassphrase(w, h, rf)
}

func BuildCmdUpdatePassphrase(w io.Writer, handler UpdatePassphraseHandler, rf *RootFlags) *cobra.Command {
	f := &UpdatePassphraseFlags{}

	cmd := &cobra.Command{
		Use:     "update",
		Short:   "Update the passphrase of the specified wallet",
		Long:    updatePassphraseLong,
		Example: updatePassphraseExample,
		RunE: func(_ *cobra.Command, _ []string) error {
			params, err := f.Validate()
			if err != nil {
				return err
			}

			if err := handler(params); err != nil {
				return err
			}

			switch rf.Output {
			case flags.InteractiveOutput:
				PrintUpdatePassphraseResponse(w)
			case flags.JSONOutput:
				return nil
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&f.Wallet,
		"wallet", "w",
		"",
		"Wallet to rename",
	)
	cmd.Flags().StringVarP(&f.PassphraseFile,
		"passphrase-file", "p",
		"",
		"Path to the file containing the current wallet's passphrase",
	)
	cmd.Flags().StringVar(&f.NewPassphraseFile,
		"new-passphrase-file",
		"",
		"Path to the file containing the new wallet's passphrase",
	)

	autoCompleteWallet(cmd, rf.Home, "wallet")

	return cmd
}

type UpdatePassphraseFlags struct {
	Wallet            string
	PassphraseFile    string
	NewPassphraseFile string
}

func (f *UpdatePassphraseFlags) Validate() (api.AdminUpdatePassphraseParams, error) {
	if len(f.Wallet) == 0 {
		return api.AdminUpdatePassphraseParams{}, flags.MustBeSpecifiedError("wallet")
	}

	passphrase, err := flags.GetPassphrase(f.PassphraseFile)
	if err != nil {
		return api.AdminUpdatePassphraseParams{}, err
	}

	newPassphrase, err := flags.GetConfirmedPassphraseWithContext(newWalletPassphraseOptions, f.NewPassphraseFile)
	if err != nil {
		return api.AdminUpdatePassphraseParams{}, err
	}

	return api.AdminUpdatePassphraseParams{
		Wallet:        f.Wallet,
		Passphrase:    passphrase,
		NewPassphrase: newPassphrase,
	}, nil
}

func PrintUpdatePassphraseResponse(w io.Writer) {
	p := printer.NewInteractivePrinter(w)

	str := p.String()
	defer p.Print(str)

	str.CheckMark().SuccessText("The wallet's passphrase has been updated.").NextLine()
}
