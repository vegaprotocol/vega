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
	vgterm "code.vegaprotocol.io/vega/libs/term"
	"code.vegaprotocol.io/vega/wallet/api"
	"code.vegaprotocol.io/vega/wallet/wallets"

	"github.com/spf13/cobra"
)

var (
	purgePermissionsLong = cli.LongDesc(`
	    Purge all the permissions of the specified wallet
	`)

	purgePermissionsExample = cli.Examples(`
		# Purge all the permissions of the specified wallet
		{{.Software}} network purge --wallet WALLET

		# Purge all the permissions of the specified wallet without 
        # asking for confirmation
		{{.Software}} network purge --wallet WALLET --force
	`)
)

type PurgePermissionsHandler func(api.AdminPurgePermissionsParams) error

func NewCmdPurgePermissions(w io.Writer, rf *RootFlags) *cobra.Command {
	h := func(params api.AdminPurgePermissionsParams) error {
		s, err := wallets.InitialiseStore(rf.Home)
		if err != nil {
			return fmt.Errorf("couldn't initialise wallets store: %w", err)
		}

		purgePermissions := api.NewAdminPurgePermissions(s)
		_, errDetails := purgePermissions.Handle(context.Background(), params, jsonrpc.RequestMetadata{})
		if errDetails != nil {
			return errors.New(errDetails.Data)
		}
		return nil
	}

	return BuildCmdPurgePermissions(w, h, rf)
}

func BuildCmdPurgePermissions(w io.Writer, handler PurgePermissionsHandler, rf *RootFlags) *cobra.Command {
	f := &PurgePermissionsFlags{}
	cmd := &cobra.Command{
		Use:     "purge",
		Short:   "Purge the permissions for the specified hostname",
		Long:    purgePermissionsLong,
		Example: purgePermissionsExample,
		RunE: func(_ *cobra.Command, _ []string) error {
			req, err := f.Validate()
			if err != nil {
				return err
			}

			if !f.Force && vgterm.HasTTY() {
				if !flags.AreYouSure() {
					return nil
				}
			}

			if err = handler(req); err != nil {
				return err
			}

			if rf.Output == flags.InteractiveOutput {
				PrintPurgePermissionsResponse(w, f.Wallet)
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&f.Wallet,
		"wallet", "w",
		"",
		"Name of the wallet to purge",
	)
	cmd.Flags().BoolVarP(&f.Force,
		"force", "f",
		false,
		"Do not ask for confirmation",
	)
	cmd.Flags().StringVarP(&f.PassphraseFile,
		"passphrase-file", "p",
		"",
		"Path to the file containing the wallet's passphrase",
	)

	autoCompleteWallet(cmd, rf.Home, "wallet")

	return cmd
}

type PurgePermissionsFlags struct {
	Wallet         string
	PassphraseFile string
	Force          bool
}

func (f *PurgePermissionsFlags) Validate() (api.AdminPurgePermissionsParams, error) {
	if len(f.Wallet) == 0 {
		return api.AdminPurgePermissionsParams{}, flags.MustBeSpecifiedError("wallet")
	}

	passphrase, err := flags.GetPassphrase(f.PassphraseFile)
	if err != nil {
		return api.AdminPurgePermissionsParams{}, err
	}

	return api.AdminPurgePermissionsParams{
		Wallet:     f.Wallet,
		Passphrase: passphrase,
	}, nil
}

func PrintPurgePermissionsResponse(w io.Writer, wallet string) {
	p := printer.NewInteractivePrinter(w)
	p.Print(p.String().CheckMark().SuccessText("All permissions on wallet ").SuccessBold(wallet).SuccessText(" have been purged.").NextLine())
}
