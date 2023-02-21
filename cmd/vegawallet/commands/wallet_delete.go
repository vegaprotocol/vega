package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"

	"code.vegaprotocol.io/vega/cmd/vegawallet/commands/cli"
	"code.vegaprotocol.io/vega/cmd/vegawallet/commands/flags"
	"code.vegaprotocol.io/vega/cmd/vegawallet/commands/printer"
	vgterm "code.vegaprotocol.io/vega/libs/term"
	"code.vegaprotocol.io/vega/wallet/api"
	"code.vegaprotocol.io/vega/wallet/wallets"

	"github.com/spf13/cobra"
)

var (
	ErrForceFlagIsRequiredWithoutTTY = errors.New("--force is required without TTY")

	deleteWalletLong = cli.LongDesc(`
		Delete the specified wallet and its keys.

		Be sure to have its recovery phrase, otherwise you won't be able to restore it. If you
		lost it, you should transfer your funds, assets, orders, and anything else attached to
		this wallet to another wallet.

		The deletion removes the file in which the wallet and its keys are stored, meaning you
		can reuse the wallet name, without causing any conflict.
	`)

	deleteWalletExample = cli.Examples(`
		# Delete the specified wallet
		{{.Software}} delete --wallet WALLET

		# Delete the specified wallet without asking for confirmation
		{{.Software}} delete --wallet WALLET --force
	`)
)

type RemoveWalletHandler func(api.AdminRemoveWalletParams) error

func NewCmdDeleteWallet(w io.Writer, rf *RootFlags) *cobra.Command {
	h := func(params api.AdminRemoveWalletParams) error {
		walletStore, err := wallets.InitialiseStore(rf.Home)
		if err != nil {
			return fmt.Errorf("couldn't initialise wallets store: %w", err)
		}
		defer walletStore.Close()

		deleteWallet := api.NewAdminRemoveWallet(walletStore)

		_, errDetails := deleteWallet.Handle(context.Background(), params)
		if errDetails != nil {
			return errors.New(errDetails.Data)
		}
		return nil
	}

	return BuildCmdDeleteWallet(w, h, rf)
}

func BuildCmdDeleteWallet(w io.Writer, handler RemoveWalletHandler, rf *RootFlags) *cobra.Command {
	f := &DeleteWalletFlags{}

	cmd := &cobra.Command{
		Use:     "delete",
		Short:   "Delete the specified wallet and its keys",
		Long:    deleteWalletLong,
		Example: deleteWalletExample,
		RunE: func(_ *cobra.Command, _ []string) error {
			params, err := f.Validate()
			if err != nil {
				return err
			}

			if !f.Force && vgterm.HasTTY() {
				if !flags.AreYouSure() {
					return nil
				}
			}

			if err := handler(params); err != nil {
				return err
			}

			switch rf.Output {
			case flags.InteractiveOutput:
				PrintDeleteWalletResponse(w, f.Wallet)
			case flags.JSONOutput:
				return nil
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&f.Wallet,
		"wallet", "w",
		"",
		"Wallet to delete",
	)
	cmd.Flags().BoolVarP(&f.Force,
		"force", "f",
		false,
		"Do not ask for confirmation",
	)

	autoCompleteWallet(cmd, rf.Home, "wallet")

	return cmd
}

type DeleteWalletFlags struct {
	Wallet string
	Force  bool
}

func (f *DeleteWalletFlags) Validate() (api.AdminRemoveWalletParams, error) {
	if len(f.Wallet) == 0 {
		return api.AdminRemoveWalletParams{}, flags.MustBeSpecifiedError("wallet")
	}

	if !f.Force && vgterm.HasNoTTY() {
		return api.AdminRemoveWalletParams{}, ErrForceFlagIsRequiredWithoutTTY
	}

	return api.AdminRemoveWalletParams{
		Wallet: f.Wallet,
	}, nil
}

func PrintDeleteWalletResponse(w io.Writer, walletName string) {
	p := printer.NewInteractivePrinter(w)

	str := p.String()
	defer p.Print(str)

	str.CheckMark().SuccessText("Wallet ").SuccessBold(walletName).SuccessText(" deleted").NextLine()
}
