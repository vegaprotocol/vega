package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"

	"code.vegaprotocol.io/vega/cmd/vegawallet/commands/cli"
	"code.vegaprotocol.io/vega/cmd/vegawallet/commands/flags"
	"code.vegaprotocol.io/vega/cmd/vegawallet/commands/printer"
	"code.vegaprotocol.io/vega/libs/jsonrpc"
	"code.vegaprotocol.io/vega/wallet/api"
	"code.vegaprotocol.io/vega/wallet/wallets"

	"github.com/spf13/cobra"
)

var (
	renameWalletLong = cli.LongDesc(`
		Rename the wallet with the specified name.
	`)

	renameWalletExample = cli.Examples(`
		# Rename the specified wallet
		{{.Software}} rename --wallet WALLET --new-name NEW_WALLET_NAME
	`)
)

type RenameWalletHandler func(api.AdminRenameWalletParams) error

func NewCmdRenameWallet(w io.Writer, rf *RootFlags) *cobra.Command {
	h := func(params api.AdminRenameWalletParams) error {
		s, err := wallets.InitialiseStore(rf.Home)
		if err != nil {
			return fmt.Errorf("couldn't initialise wallets store: %w", err)
		}

		renameWallet := api.NewAdminRenameWallet(s)

		_, errDetails := renameWallet.Handle(context.Background(), params, jsonrpc.RequestMetadata{})
		if errDetails != nil {
			return errors.New(errDetails.Data)
		}
		return nil
	}

	return BuildCmdRenameWallet(w, h, rf)
}

func BuildCmdRenameWallet(w io.Writer, handler RenameWalletHandler, rf *RootFlags) *cobra.Command {
	f := &RenameWalletFlags{}

	cmd := &cobra.Command{
		Use:     "rename",
		Short:   "Rename the specified wallet",
		Long:    renameWalletLong,
		Example: renameWalletExample,
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
				PrintRenameWalletResponse(w, f)
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
	cmd.Flags().StringVar(&f.NewName,
		"new-name",
		"",
		"New name for the wallet",
	)

	autoCompleteWallet(cmd, rf.Home, "wallet")

	return cmd
}

type RenameWalletFlags struct {
	Wallet  string
	NewName string
}

func (f *RenameWalletFlags) Validate() (api.AdminRenameWalletParams, error) {
	if len(f.Wallet) == 0 {
		return api.AdminRenameWalletParams{}, flags.MustBeSpecifiedError("wallet")
	}

	if len(f.NewName) == 0 {
		return api.AdminRenameWalletParams{}, flags.MustBeSpecifiedError("new-name")
	}

	return api.AdminRenameWalletParams{
		Wallet:  f.Wallet,
		NewName: f.NewName,
	}, nil
}

func PrintRenameWalletResponse(w io.Writer, f *RenameWalletFlags) {
	p := printer.NewInteractivePrinter(w)

	str := p.String()
	defer p.Print(str)

	str.CheckMark().SuccessText("The wallet ").SuccessBold(f.Wallet).SuccessText(" has been renamed to ").SuccessBold(f.NewName).SuccessText(".").NextLine()
}
