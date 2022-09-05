package cmd

import (
	"fmt"
	"io"

	"code.vegaprotocol.io/vega/cmd/vegawallet/commands/cli"
	"code.vegaprotocol.io/vega/cmd/vegawallet/commands/flags"
	"code.vegaprotocol.io/vega/cmd/vegawallet/commands/printer"
	"code.vegaprotocol.io/vega/wallet/wallet"
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

type DescribePermissionsHandler func(*wallet.DescribePermissionsRequest) (*wallet.DescribePermissionsResponse, error)

func NewCmdDescribePermissions(w io.Writer, rf *RootFlags) *cobra.Command {
	h := func(req *wallet.DescribePermissionsRequest) (*wallet.DescribePermissionsResponse, error) {
		s, err := wallets.InitialiseStore(rf.Home)
		if err != nil {
			return nil, fmt.Errorf("couldn't initialise wallets store: %w", err)
		}

		return wallet.DescribePermissions(s, req)
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
				PrintDescribePermissionsResponse(w, resp)
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

	autoCompleteWallet(cmd, rf.Home)

	return cmd
}

type DescribePermissionsFlags struct {
	Wallet         string
	Hostname       string
	PassphraseFile string
}

func (f *DescribePermissionsFlags) Validate() (*wallet.DescribePermissionsRequest, error) {
	req := &wallet.DescribePermissionsRequest{}

	if len(f.Wallet) == 0 {
		return nil, flags.MustBeSpecifiedError("wallet")
	}
	req.Wallet = f.Wallet

	if len(f.Hostname) == 0 {
		return nil, flags.MustBeSpecifiedError("hostname")
	}
	req.Hostname = f.Hostname

	passphrase, err := flags.GetPassphrase(f.PassphraseFile)
	if err != nil {
		return nil, err
	}
	req.Passphrase = passphrase

	return req, nil
}

func PrintDescribePermissionsResponse(w io.Writer, resp *wallet.DescribePermissionsResponse) {
	p := printer.NewInteractivePrinter(w)

	str := p.String()
	defer p.Print(str)

	str.Text("Public keys: ").NextLine()
	str.Text("  Access mode: ").WarningText(fmt.Sprintf("%v", resp.Permissions.PublicKeys.Access)).NextLine()
	if len(resp.Permissions.PublicKeys.RestrictedKeys) != 0 {
		str.Text("  Restricted keys: ")
		for _, k := range resp.Permissions.PublicKeys.RestrictedKeys {
			str.WarningText(fmt.Sprintf("    - %s", k)).NextLine()
		}
	}
}
