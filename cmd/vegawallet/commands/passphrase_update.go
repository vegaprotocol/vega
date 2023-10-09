// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

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

type UpdatePassphraseHandler func(api.AdminUpdatePassphraseParams, string) error

func NewCmdUpdatePassphrase(w io.Writer, rf *RootFlags) *cobra.Command {
	h := func(params api.AdminUpdatePassphraseParams, passphrase string) error {
		ctx := context.Background()

		walletStore, err := wallets.InitialiseStore(rf.Home, false)
		if err != nil {
			return fmt.Errorf("couldn't initialise wallets store: %w", err)
		}
		defer walletStore.Close()

		if _, errDetails := api.NewAdminUnlockWallet(walletStore).Handle(ctx, api.AdminUnlockWalletParams{
			Wallet:     params.Wallet,
			Passphrase: passphrase,
		}); errDetails != nil {
			return errors.New(errDetails.Data)
		}

		if _, errDetails := api.NewAdminUpdatePassphrase(walletStore).Handle(ctx, params); errDetails != nil {
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
			params, pass, err := f.Validate()
			if err != nil {
				return err
			}

			if err := handler(params, pass); err != nil {
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

func (f *UpdatePassphraseFlags) Validate() (api.AdminUpdatePassphraseParams, string, error) {
	if len(f.Wallet) == 0 {
		return api.AdminUpdatePassphraseParams{}, "", flags.MustBeSpecifiedError("wallet")
	}

	passphrase, err := flags.GetPassphrase(f.PassphraseFile)
	if err != nil {
		return api.AdminUpdatePassphraseParams{}, "", err
	}

	newPassphrase, err := flags.GetConfirmedPassphraseWithContext(newWalletPassphraseOptions, f.NewPassphraseFile)
	if err != nil {
		return api.AdminUpdatePassphraseParams{}, "", err
	}

	return api.AdminUpdatePassphraseParams{
		Wallet:        f.Wallet,
		NewPassphrase: newPassphrase,
	}, passphrase, nil
}

func PrintUpdatePassphraseResponse(w io.Writer) {
	p := printer.NewInteractivePrinter(w)

	str := p.String()
	defer p.Print(str)

	str.CheckMark().SuccessText("The wallet's passphrase has been updated.").NextLine()
}
