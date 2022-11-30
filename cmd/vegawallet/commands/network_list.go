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
	"code.vegaprotocol.io/vega/paths"
	"code.vegaprotocol.io/vega/wallet/api"
	networkStore "code.vegaprotocol.io/vega/wallet/network/store/v1"

	"github.com/spf13/cobra"
)

var (
	listNetworkLong = cli.LongDesc(`
		List all registered networks.
	`)

	listNetworkExample = cli.Examples(`
		# List networks
		{{.Software}} network list
	`)
)

type ListNetworksHandler func() (api.AdminListNetworksResult, error)

func NewCmdListNetworks(w io.Writer, rf *RootFlags) *cobra.Command {
	h := func() (api.AdminListNetworksResult, error) {
		vegaPaths := paths.New(rf.Home)

		networkStore, err := networkStore.InitialiseStore(vegaPaths)
		if err != nil {
			return api.AdminListNetworksResult{}, fmt.Errorf("couldn't initialise network store: %w", err)
		}

		listWallet := api.NewAdminListNetworks(networkStore)
		rawResult, errorDetails := listWallet.Handle(context.Background(), nil, jsonrpc.RequestMetadata{})
		if errorDetails != nil {
			return api.AdminListNetworksResult{}, errors.New(errorDetails.Data)
		}
		return rawResult.(api.AdminListNetworksResult), nil
	}

	return BuildCmdListNetworks(w, h, rf)
}

func BuildCmdListNetworks(w io.Writer, handler ListNetworksHandler, rf *RootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List all registered networks",
		Long:    listNetworkLong,
		Example: listNetworkExample,
		RunE: func(_ *cobra.Command, _ []string) error {
			resp, err := handler()
			if err != nil {
				return err
			}

			switch rf.Output {
			case flags.InteractiveOutput:
				PrintListNetworksResult(w, resp)
			case flags.JSONOutput:
				return printer.FprintJSON(w, resp)
			}

			return nil
		},
	}

	return cmd
}

func PrintListNetworksResult(w io.Writer, resp api.AdminListNetworksResult) {
	p := printer.NewInteractivePrinter(w)

	str := p.String()
	defer p.Print(str)

	if len(resp.Networks) == 0 {
		str.InfoText("No network registered").NextLine()
		return
	}

	for _, net := range resp.Networks {
		str.Text(fmt.Sprintf("- %s", net)).NextLine()
	}
}
