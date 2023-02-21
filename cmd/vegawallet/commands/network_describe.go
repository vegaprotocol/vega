package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"

	"code.vegaprotocol.io/vega/cmd/vegawallet/commands/cli"
	"code.vegaprotocol.io/vega/cmd/vegawallet/commands/flags"
	"code.vegaprotocol.io/vega/cmd/vegawallet/commands/printer"
	"code.vegaprotocol.io/vega/paths"
	"code.vegaprotocol.io/vega/wallet/api"
	networkStoreV1 "code.vegaprotocol.io/vega/wallet/network/store/v1"

	"github.com/spf13/cobra"
)

var (
	describeNetworkLong = cli.LongDesc(`
	    Describe all known information about the specified network.
	`)

	describeNetworkExample = cli.Examples(`
		# Describe a network
		{{.Software}} network describe --network NETWORK
	`)
)

type DescribeNetworkHandler func(api.AdminDescribeNetworkParams) (api.AdminDescribeNetworkResult, error)

func NewCmdDescribeNetwork(w io.Writer, rf *RootFlags) *cobra.Command {
	h := func(params api.AdminDescribeNetworkParams) (api.AdminDescribeNetworkResult, error) {
		vegaPaths := paths.New(rf.Home)

		networkStore, err := networkStoreV1.InitialiseStore(vegaPaths)
		if err != nil {
			return api.AdminDescribeNetworkResult{}, fmt.Errorf("couldn't initialise network store: %w", err)
		}

		describeNetwork := api.NewAdminDescribeNetwork(networkStore)
		rawResult, errorDetails := describeNetwork.Handle(context.Background(), params)
		if errorDetails != nil {
			return api.AdminDescribeNetworkResult{}, errors.New(errorDetails.Data)
		}
		return rawResult.(api.AdminDescribeNetworkResult), nil
	}

	return BuildCmdDescribeNetwork(w, h, rf)
}

type DescribeNetworkFlags struct {
	Network string
}

func (f *DescribeNetworkFlags) Validate() (api.AdminDescribeNetworkParams, error) {
	req := api.AdminDescribeNetworkParams{}

	if len(f.Network) == 0 {
		return api.AdminDescribeNetworkParams{}, flags.MustBeSpecifiedError("network")
	}
	req.Name = f.Network

	return req, nil
}

func BuildCmdDescribeNetwork(w io.Writer, handler DescribeNetworkHandler, rf *RootFlags) *cobra.Command {
	f := &DescribeNetworkFlags{}
	cmd := &cobra.Command{
		Use:     "describe",
		Short:   "Describe the specified network",
		Long:    describeNetworkLong,
		Example: describeNetworkExample,
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
				PrintDescribeNetworkResponse(w, resp)
			case flags.JSONOutput:
				return printer.FprintJSON(w, resp)
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&f.Network,
		"network", "n",
		"",
		"Network to describe",
	)

	autoCompleteNetwork(cmd, rf.Home)

	return cmd
}

func PrintDescribeNetworkResponse(w io.Writer, resp api.AdminDescribeNetworkResult) {
	p := printer.NewInteractivePrinter(w)

	str := p.String()
	defer p.Print(str)

	str.NextLine().Text("Network").NextLine()
	str.Text("  Name: ").WarningText(resp.Name).NextLine()
	if len(resp.Metadata) != 0 {
		str.Text("  Metadata: ").NextLine()
		padding := 0
		for _, m := range resp.Metadata {
			keyLen := len(m.Key)
			if keyLen > padding {
				padding = keyLen
			}
		}

		for _, m := range resp.Metadata {
			str.WarningText(fmt.Sprintf("%-*s", padding, m.Key)).Text(" | ").WarningText(m.Value).NextLine()
		}
	}
	str.NextSection()

	str.NextLine().Text("Applications").NextLine()
	str.ListItem().Text("- Console: ")
	PrintDescribeNetworkWithValueNotSet(str, resp.Apps.Console)
	str.ListItem().Text("- Token dApp: ").WarningText(resp.Apps.TokenDApp)
	PrintDescribeNetworkWithValueNotSet(str, resp.Apps.TokenDApp)
	str.ListItem().Text("- Explorer: ").WarningText(resp.Apps.Explorer)
	PrintDescribeNetworkWithValueNotSet(str, resp.Apps.Explorer)
	str.NextSection()

	str.Text("API.GRPC").NextLine()
	str.Text("  Retries: ").WarningText(fmt.Sprint(resp.API.GRPCConfig.Retries)).NextLine()
	str.Text("  Hosts:")
	PrintDescribeNetworkWithValuesNotSet(str, resp.API.GRPCConfig.Hosts)
	str.NextLine()

	str.Text("API.REST").NextLine()
	str.Text("  Hosts:")
	PrintDescribeNetworkWithValuesNotSet(str, resp.API.RESTConfig.Hosts)
	str.NextLine()

	str.Text("API.GraphQL").NextLine()
	str.Text("  Hosts:")
	PrintDescribeNetworkWithValuesNotSet(str, resp.API.GraphQLConfig.Hosts)
	str.NextLine()
}

func PrintDescribeNetworkWithValueNotSet(str *printer.FormattedString, value string) {
	if value == "" {
		str.DangerText("<not set>")
	} else {
		str.WarningText(value)
	}
	str.NextLine()
}

func PrintDescribeNetworkWithValuesNotSet(str *printer.FormattedString, hosts []string) {
	if len(hosts) == 0 {
		str.DangerText(" <not set>").NextLine()
		return
	}

	str.NextLine()
	for _, h := range hosts {
		str.ListItem().Text("- ").WarningText(h).NextLine()
	}
}
