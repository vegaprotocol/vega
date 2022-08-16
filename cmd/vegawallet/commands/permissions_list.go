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
	listPermissionsLong = cli.LongDesc(`
		List all permitted hostnames for the specified wallet.
	`)

	listPermissionsExample = cli.Examples(`
		# List all permitted hostnames for the specified wallet
		{{.Software}} permissions list --wallet WALLET
	`)
)

type ListPermissionsHandler func(*wallet.ListPermissionsRequest) (*wallet.ListPermissionsResponse, error)

func NewCmdListPermissions(w io.Writer, rf *RootFlags) *cobra.Command {
	h := func(req *wallet.ListPermissionsRequest) (*wallet.ListPermissionsResponse, error) {
		s, err := wallets.InitialiseStore(rf.Home)
		if err != nil {
			return nil, fmt.Errorf("couldn't initialise wallets store: %w", err)
		}

		return wallet.ListPermissions(s, req)
	}

	return BuildCmdListPermissions(w, h, rf)
}

func BuildCmdListPermissions(w io.Writer, handler ListPermissionsHandler, rf *RootFlags) *cobra.Command {
	f := &ListPermissionsFlags{}

	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List all permitted hostnames for the specified wallet",
		Long:    listPermissionsLong,
		Example: listPermissionsExample,
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
				PrintListPermissionsResponse(w, resp)
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
	cmd.Flags().StringVarP(&f.PassphraseFile,
		"passphrase-file", "p",
		"",
		"Path to the file containing the wallet's passphrase",
	)

	autoCompleteWallet(cmd, rf.Home)

	return cmd
}

type ListPermissionsFlags struct {
	Wallet         string
	PassphraseFile string
}

func (f *ListPermissionsFlags) Validate() (*wallet.ListPermissionsRequest, error) {
	req := &wallet.ListPermissionsRequest{}

	if len(f.Wallet) == 0 {
		return nil, flags.FlagMustBeSpecifiedError("wallet")
	}
	req.Wallet = f.Wallet

	passphrase, err := flags.GetPassphrase(f.PassphraseFile)
	if err != nil {
		return nil, err
	}
	req.Passphrase = passphrase

	return req, nil
}

func PrintListPermissionsResponse(w io.Writer, resp *wallet.ListPermissionsResponse) {
	p := printer.NewInteractivePrinter(w)

	str := p.String()
	defer p.Print(str)

	if len(resp.Hostnames) == 0 {
		str.InfoText("No permission has been given to any hostname").NextLine()
		return
	}

	for _, hostname := range resp.Hostnames {
		str.Text(fmt.Sprintf("- %s", hostname)).NextLine()
	}
}
