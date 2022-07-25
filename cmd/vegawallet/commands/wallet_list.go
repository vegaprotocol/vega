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
	listWalletsLong = cli.LongDesc(`
		List all registered wallets.
	`)

	listWalletsExample = cli.Examples(`
		# List all registered wallets
		{{.Software}} list
	`)
)

type ListWalletsHandler func() (*wallet.ListWalletsResponse, error)

func NewCmdListWallets(w io.Writer, rf *RootFlags) *cobra.Command {
	h := func() (*wallet.ListWalletsResponse, error) {
		s, err := wallets.InitialiseStore(rf.Home)
		if err != nil {
			return nil, fmt.Errorf("couldn't initialise wallets store: %w", err)
		}

		return wallet.ListWallets(s)
	}

	return BuildCmdListWallets(w, h, rf)
}

func BuildCmdListWallets(w io.Writer, handler ListWalletsHandler, rf *RootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List all registered wallets",
		Long:    listWalletsLong,
		Example: listWalletsExample,
		RunE: func(_ *cobra.Command, _ []string) error {
			resp, err := handler()
			if err != nil {
				return err
			}

			switch rf.Output {
			case flags.InteractiveOutput:
				PrintListWalletsResponse(w, resp)
			case flags.JSONOutput:
				return printer.FprintJSON(w, resp)
			}

			return nil
		},
	}

	return cmd
}

func PrintListWalletsResponse(w io.Writer, resp *wallet.ListWalletsResponse) {
	p := printer.NewInteractivePrinter(w)

	str := p.String()
	defer p.Print(str)

	if len(resp.Wallets) == 0 {
		str.InfoText("No wallet registered").NextLine()
		return
	}

	for _, w := range resp.Wallets {
		str.Text(fmt.Sprintf("- %s", w)).NextLine()
	}
}
