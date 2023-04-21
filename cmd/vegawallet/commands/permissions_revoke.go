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
	revokePermissionsLong = cli.LongDesc(`
	    Revoke the permissions of the specified hostname
	`)

	revokePermissionsExample = cli.Examples(`
		# Revoke the permissions for the specified hostname
		{{.Software}} network revoke --wallet WALLET --hostname HOSTNAME

		# Revoke the permissions for the specified hostname without 
        # asking for confirmation
		{{.Software}} network revoke --wallet WALLET --hostname HOSTNAME --force
	`)
)

type RevokePermissionsHandler func(api.AdminRevokePermissionsParams, string) error

func NewCmdRevokePermissions(w io.Writer, rf *RootFlags) *cobra.Command {
	h := func(params api.AdminRevokePermissionsParams, passphrase string) error {
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

		if _, errDetails := api.NewAdminRevokePermissions(walletStore).Handle(ctx, params); errDetails != nil {
			return errors.New(errDetails.Data)
		}
		return nil
	}

	return BuildCmdRevokePermissions(w, h, rf)
}

func BuildCmdRevokePermissions(w io.Writer, handler RevokePermissionsHandler, rf *RootFlags) *cobra.Command {
	f := &RevokePermissionsFlags{}
	cmd := &cobra.Command{
		Use:     "revoke",
		Short:   "Revoke the permissions for the specified hostname",
		Long:    revokePermissionsLong,
		Example: revokePermissionsExample,
		RunE: func(_ *cobra.Command, _ []string) error {
			req, pass, err := f.Validate()
			if err != nil {
				return err
			}

			if !f.Force && vgterm.HasTTY() {
				if !flags.AreYouSure() {
					return nil
				}
			}

			if err = handler(req, pass); err != nil {
				return err
			}

			if rf.Output == flags.InteractiveOutput {
				PrintRevokePermissionsResponse(w, req)
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&f.Wallet,
		"wallet", "w",
		"",
		"Name of the wallet",
	)
	cmd.Flags().StringVar(&f.Hostname,
		"hostname",
		"",
		"Hostname from which access is revoked",
	)
	cmd.Flags().StringVarP(&f.PassphraseFile,
		"passphrase-file", "p",
		"",
		"Path to the file containing the wallet's passphrase",
	)
	cmd.Flags().BoolVarP(&f.Force,
		"force", "f",
		false,
		"Do not ask for confirmation",
	)

	autoCompleteWallet(cmd, rf.Home, "wallet")

	return cmd
}

type RevokePermissionsFlags struct {
	Wallet         string
	Hostname       string
	Force          bool
	PassphraseFile string
}

func (f *RevokePermissionsFlags) Validate() (api.AdminRevokePermissionsParams, string, error) {
	if len(f.Wallet) == 0 {
		return api.AdminRevokePermissionsParams{}, "", flags.MustBeSpecifiedError("wallet")
	}

	if len(f.Hostname) == 0 {
		return api.AdminRevokePermissionsParams{}, "", flags.MustBeSpecifiedError("hostname")
	}

	passphrase, err := flags.GetPassphrase(f.PassphraseFile)
	if err != nil {
		return api.AdminRevokePermissionsParams{}, "", err
	}

	return api.AdminRevokePermissionsParams{
		Wallet:   f.Wallet,
		Hostname: f.Hostname,
	}, passphrase, nil
}

func PrintRevokePermissionsResponse(w io.Writer, req api.AdminRevokePermissionsParams) {
	p := printer.NewInteractivePrinter(w)
	str := p.String()
	defer p.Print(str)
	str.CheckMark().SuccessText("Permissions for hostname ").SuccessBold(req.Hostname).SuccessText(" has been revoked from wallet ").SuccessBold(req.Wallet).SuccessText(".").NextLine()
}
