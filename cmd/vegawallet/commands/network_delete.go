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
	"code.vegaprotocol.io/vega/paths"
	"code.vegaprotocol.io/vega/wallet/api"
	networkStore "code.vegaprotocol.io/vega/wallet/network/store/v1"

	"github.com/spf13/cobra"
)

var (
	deleteNetworkLong = cli.LongDesc(`
	    Delete the specified network
	`)

	deleteNetworkExample = cli.Examples(`
		# Delete the specified network
		{{.Software}} network delete --network NETWORK

		# Delete the specified network without asking for confirmation
		{{.Software}} delete --wallet WALLET --force
	`)
)

type RemoveNetworkHandler func(api.AdminRemoveNetworkParams) error

func NewCmdDeleteNetwork(w io.Writer, rf *RootFlags) *cobra.Command {
	h := func(params api.AdminRemoveNetworkParams) error {
		vegaPaths := paths.New(rf.Home)

		s, err := networkStore.InitialiseStore(vegaPaths)
		if err != nil {
			return fmt.Errorf("couldn't initialise network store: %w", err)
		}

		deleteNetwork := api.NewAdminRemoveNetwork(s)

		_, errDetails := deleteNetwork.Handle(context.Background(), params, jsonrpc.RequestMetadata{})
		if errDetails != nil {
			return errors.New(errDetails.Data)
		}
		return nil
	}

	return BuildCmdDeleteNetwork(w, h, rf)
}

func BuildCmdDeleteNetwork(w io.Writer, handler RemoveNetworkHandler, rf *RootFlags) *cobra.Command {
	f := &DeleteNetworkFlags{}
	cmd := &cobra.Command{
		Use:     "delete",
		Short:   "Delete the specified network",
		Long:    deleteNetworkLong,
		Example: deleteNetworkExample,
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
				PrintDeleteNetworkResponse(w, f.Network)
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&f.Network,
		"network", "n",
		"",
		"Network to delete",
	)
	cmd.Flags().BoolVarP(&f.Force,
		"force", "f",
		false,
		"Do not ask for confirmation",
	)

	autoCompleteNetwork(cmd, rf.Home)

	return cmd
}

type DeleteNetworkFlags struct {
	Network string
	Force   bool
}

func (f *DeleteNetworkFlags) Validate() (api.AdminRemoveNetworkParams, error) {
	req := api.AdminRemoveNetworkParams{}

	if len(f.Network) == 0 {
		return api.AdminRemoveNetworkParams{}, flags.MustBeSpecifiedError("network")
	}
	req.Name = f.Network

	return req, nil
}

func PrintDeleteNetworkResponse(w io.Writer, networkName string) {
	p := printer.NewInteractivePrinter(w)

	str := p.String()
	defer p.Print(str)

	str.CheckMark().SuccessText("Network ").SuccessBold(networkName).SuccessText(" deleted").NextLine()
}
