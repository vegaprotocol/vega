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
	describePermissionsLong = cli.LongDesc(`
	    Describe the permissions associated to a given hostname.
	`)

	describePermissionsExample = cli.Examples(`
		# Describe the permissions
		{{.Software}} permissions describe --wallet WALLET --hostname HOSTNAME
	`)
)

type DescribePermissionsHandler func(api.AdminDescribePermissionsParams, string) (api.AdminDescribePermissionsResult, error)

func NewCmdDescribePermissions(w io.Writer, rf *RootFlags) *cobra.Command {
	h := func(params api.AdminDescribePermissionsParams, passphrase string) (api.AdminDescribePermissionsResult, error) {
		ctx := context.Background()

		walletStore, err := wallets.InitialiseStore(rf.Home, false)
		if err != nil {
			return api.AdminDescribePermissionsResult{}, fmt.Errorf("couldn't initialise wallets store: %w", err)
		}
		defer walletStore.Close()

		if _, errDetails := api.NewAdminUnlockWallet(walletStore).Handle(ctx, api.AdminUnlockWalletParams{
			Wallet:     params.Wallet,
			Passphrase: passphrase,
		}); errDetails != nil {
			return api.AdminDescribePermissionsResult{}, errors.New(errDetails.Data)
		}

		rawResult, errDetails := api.NewAdminDescribePermissions(walletStore).Handle(ctx, params)
		if errDetails != nil {
			return api.AdminDescribePermissionsResult{}, errors.New(errDetails.Data)
		}
		return rawResult.(api.AdminDescribePermissionsResult), nil
	}

	return BuildCmdDescribePermissions(w, h, rf)
}

func BuildCmdDescribePermissions(w io.Writer, handler DescribePermissionsHandler, rf *RootFlags) *cobra.Command {
	f := &DescribePermissionsFlags{}
	cmd := &cobra.Command{
		Use:     "describe",
		Short:   "Describe the permissions associated to the specified hostname",
		Long:    describePermissionsLong,
		Example: describePermissionsExample,
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
				PrintDescribePermissionsResult(w, resp)
			case flags.JSONOutput:
				return printer.FprintJSON(w, resp)
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
		"Hostname to describe",
	)
	cmd.Flags().StringVarP(&f.PassphraseFile,
		"passphrase-file", "p",
		"",
		"Path to the file containing the wallet's passphrase",
	)

	autoCompleteWallet(cmd, rf.Home, "wallet")

	return cmd
}

type DescribePermissionsFlags struct {
	Wallet         string
	Hostname       string
	PassphraseFile string
}

func (f *DescribePermissionsFlags) Validate() (api.AdminDescribePermissionsParams, string, error) {
	if len(f.Wallet) == 0 {
		return api.AdminDescribePermissionsParams{}, "", flags.MustBeSpecifiedError("wallet")
	}

	if len(f.Hostname) == 0 {
		return api.AdminDescribePermissionsParams{}, "", flags.MustBeSpecifiedError("hostname")
	}

	passphrase, err := flags.GetPassphrase(f.PassphraseFile)
	if err != nil {
		return api.AdminDescribePermissionsParams{}, "", err
	}

	return api.AdminDescribePermissionsParams{
		Wallet:   f.Wallet,
		Hostname: f.Hostname,
	}, passphrase, nil
}

func PrintDescribePermissionsResult(w io.Writer, resp api.AdminDescribePermissionsResult) {
	p := printer.NewInteractivePrinter(w)

	str := p.String()
	defer p.Print(str)

	str.Text("Public keys: ").NextLine()
	str.Text("  Access mode: ").WarningText(fmt.Sprintf("%v", resp.Permissions.PublicKeys.Access)).NextLine()
	if len(resp.Permissions.PublicKeys.AllowedKeys) != 0 {
		str.Text("  Allowed keys: ").NextLine()
		for _, k := range resp.Permissions.PublicKeys.AllowedKeys {
			str.ListItem().Text("- ").WarningText(k).NextLine()
		}
	}
}
