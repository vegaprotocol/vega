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
	listPermissionsLong = cli.LongDesc(`
		List all permitted hostnames for the specified wallet.
	`)

	listPermissionsExample = cli.Examples(`
		# List all permitted hostnames for the specified wallet
		{{.Software}} permissions list --wallet WALLET
	`)
)

type ListPermissionsHandler func(api.AdminListPermissionsParams) (api.AdminListPermissionsResult, error)

func NewCmdListPermissions(w io.Writer, rf *RootFlags) *cobra.Command {
	h := func(params api.AdminListPermissionsParams) (api.AdminListPermissionsResult, error) {
		s, err := wallets.InitialiseStore(rf.Home)
		if err != nil {
			return api.AdminListPermissionsResult{}, fmt.Errorf("couldn't initialise wallets store: %w", err)
		}

		listPermissions := api.NewAdminListPermissions(s)
		rawResult, errDetails := listPermissions.Handle(context.Background(), params)
		if errDetails != nil {
			return api.AdminListPermissionsResult{}, errors.New(errDetails.Data)
		}
		return rawResult.(api.AdminListPermissionsResult), nil
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

	autoCompleteWallet(cmd, rf.Home, "wallet")

	return cmd
}

type ListPermissionsFlags struct {
	Wallet         string
	PassphraseFile string
}

func (f *ListPermissionsFlags) Validate() (api.AdminListPermissionsParams, error) {
	if len(f.Wallet) == 0 {
		return api.AdminListPermissionsParams{}, flags.MustBeSpecifiedError("wallet")
	}

	passphrase, err := flags.GetPassphrase(f.PassphraseFile)
	if err != nil {
		return api.AdminListPermissionsParams{}, err
	}

	return api.AdminListPermissionsParams{
		Wallet:     f.Wallet,
		Passphrase: passphrase,
	}, nil
}

func PrintListPermissionsResponse(w io.Writer, resp api.AdminListPermissionsResult) {
	p := printer.NewInteractivePrinter(w)

	str := p.String()
	defer p.Print(str)

	if len(resp.Permissions) == 0 {
		str.InfoText("No permission has been given to any hostname").NextLine()
		return
	}

	for hostname, permissions := range resp.Permissions {
		str.Text(fmt.Sprintf("* %s", hostname)).NextLine()
		for scope, access := range permissions {
			str.Pad().Text(fmt.Sprintf("- %s: %s", scope, access)).NextLine()
		}
		str.NextLine()
	}
}
