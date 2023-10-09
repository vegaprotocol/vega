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
	describeWalletLong = cli.LongDesc(`
		Get wallet information such as wallet ID, key derivation version and type.
	`)

	describeWalletExample = cli.Examples(`
		# Get the wallet information
		{{.Software}} describe --wallet WALLET
	`)
)

type DescribeWalletHandler func(api.AdminDescribeWalletParams, string) (api.AdminDescribeWalletResult, error)

func NewCmdDescribeWallet(w io.Writer, rf *RootFlags) *cobra.Command {
	h := func(params api.AdminDescribeWalletParams, passphrase string) (api.AdminDescribeWalletResult, error) {
		ctx := context.Background()

		walletStore, err := wallets.InitialiseStore(rf.Home, false)
		if err != nil {
			return api.AdminDescribeWalletResult{}, fmt.Errorf("couldn't initialise wallets store: %w", err)
		}
		defer walletStore.Close()

		if _, errDetails := api.NewAdminUnlockWallet(walletStore).Handle(ctx, api.AdminUnlockWalletParams{
			Wallet:     params.Wallet,
			Passphrase: passphrase,
		}); errDetails != nil {
			return api.AdminDescribeWalletResult{}, errors.New(errDetails.Data)
		}

		rawResult, errorDetails := api.NewAdminDescribeWallet(walletStore).Handle(ctx, params)
		if errorDetails != nil {
			return api.AdminDescribeWalletResult{}, errors.New(errorDetails.Data)
		}
		return rawResult.(api.AdminDescribeWalletResult), nil
	}
	return BuildCmdDescribeWallet(w, h, rf)
}

func BuildCmdDescribeWallet(w io.Writer, handler DescribeWalletHandler, rf *RootFlags) *cobra.Command {
	f := &DescribeWalletFlags{}

	cmd := &cobra.Command{
		Use:     "describe",
		Short:   "Describe the specified wallet",
		Long:    describeWalletLong,
		Example: describeWalletExample,
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
				PrintDescribeWalletResponse(w, resp)
			case flags.JSONOutput:
				return printer.FprintJSON(w, resp)
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&f.Wallet,
		"wallet", "w",
		"",
		"Name of the wallet to use",
	)
	cmd.Flags().StringVarP(&f.PassphraseFile,
		"passphrase-file", "p",
		"",
		"Path to the file containing the wallet's passphrase",
	)

	autoCompleteWallet(cmd, rf.Home, "wallet")

	return cmd
}

type DescribeWalletFlags struct {
	Wallet         string
	PassphraseFile string
}

func (f *DescribeWalletFlags) Validate() (api.AdminDescribeWalletParams, string, error) {
	req := api.AdminDescribeWalletParams{}

	if len(f.Wallet) == 0 {
		return api.AdminDescribeWalletParams{}, "", flags.MustBeSpecifiedError("wallet")
	}
	req.Wallet = f.Wallet

	passphrase, err := flags.GetPassphrase(f.PassphraseFile)
	if err != nil {
		return api.AdminDescribeWalletParams{}, "", err
	}

	return req, passphrase, nil
}

func PrintDescribeWalletResponse(w io.Writer, resp api.AdminDescribeWalletResult) {
	p := printer.NewInteractivePrinter(w)

	str := p.String()
	defer p.Print(str)

	str.Text("Type:").NextLine().WarningText(resp.Type).NextLine()
	str.Text("Key derivation version:").NextLine().WarningText(fmt.Sprintf("%d", resp.KeyDerivationVersion)).NextLine()
	str.Text("ID:").NextLine().WarningText(resp.ID).NextLine()
}
