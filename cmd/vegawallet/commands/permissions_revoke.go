package cmd

import (
	"fmt"
	"io"

	vgterm "code.vegaprotocol.io/vega/libs/term"
	"code.vegaprotocol.io/vega/cmd/vegawallet/commands/cli"
	"code.vegaprotocol.io/vega/cmd/vegawallet/commands/flags"
	"code.vegaprotocol.io/vega/cmd/vegawallet/commands/printer"
	"code.vegaprotocol.io/vega/wallet/wallet"
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

type RevokePermissionsHandler func(*wallet.RevokePermissionsRequest) error

func NewCmdRevokePermissions(w io.Writer, rf *RootFlags) *cobra.Command {
	h := func(req *wallet.RevokePermissionsRequest) error {
		s, err := wallets.InitialiseStore(rf.Home)
		if err != nil {
			return fmt.Errorf("couldn't initialise wallets store: %w", err)
		}

		return wallet.RevokePermissions(s, req)
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

	autoCompleteWallet(cmd, rf.Home)

	return cmd
}

type RevokePermissionsFlags struct {
	Wallet         string
	Hostname       string
	Force          bool
	PassphraseFile string
}

func (f *RevokePermissionsFlags) Validate() (*wallet.RevokePermissionsRequest, error) {
	req := &wallet.RevokePermissionsRequest{}

	if len(f.Wallet) == 0 {
		return nil, flags.FlagMustBeSpecifiedError("wallet")
	}
	req.Wallet = f.Wallet

	if len(f.Hostname) == 0 {
		return nil, flags.FlagMustBeSpecifiedError("hostname")
	}
	req.Hostname = f.Hostname

	passphrase, err := flags.GetPassphrase(f.PassphraseFile)
	if err != nil {
		return nil, err
	}
	req.Passphrase = passphrase

	return req, nil
}

func PrintRevokePermissionsResponse(w io.Writer, req *wallet.RevokePermissionsRequest) {
	p := printer.NewInteractivePrinter(w)
	str := p.String()
	defer p.Print(str)
	str.CheckMark().SuccessText("Permissions for hostname ").SuccessBold(req.Hostname).SuccessText(" has been revoked from wallet ").SuccessBold(req.Wallet).NextLine()
}
