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
	networkStore "code.vegaprotocol.io/vega/wallet/network/store/v1"

	"github.com/spf13/cobra"
)

var (
	renameNetworkLong = cli.LongDesc(`
	    Rename the network with the specified name.
	`)

	renameNetworkExample = cli.Examples(`
		# Rename the specified network
		{{.Software}} network rename --network NETWORK --new-name NEW_NETWORK_NAME
	`)
)

type RenameNetworkHandler func(api.AdminRenameNetworkParams) error

func NewCmdRenameNetwork(w io.Writer, rf *RootFlags) *cobra.Command {
	h := func(params api.AdminRenameNetworkParams) error {
		vegaPaths := paths.New(rf.Home)

		s, err := networkStore.InitialiseStore(vegaPaths)
		if err != nil {
			return fmt.Errorf("couldn't initialise network store: %w", err)
		}

		renameNetwork := api.NewAdminRenameNetwork(s)

		_, errDetails := renameNetwork.Handle(context.Background(), params)
		if errDetails != nil {
			return errors.New(errDetails.Data)
		}
		return nil
	}

	return BuildCmdRenameNetwork(w, h, rf)
}

func BuildCmdRenameNetwork(w io.Writer, handler RenameNetworkHandler, rf *RootFlags) *cobra.Command {
	f := &RenameNetworkFlags{}
	cmd := &cobra.Command{
		Use:     "rename",
		Short:   "Rename the specified network",
		Long:    renameNetworkLong,
		Example: renameNetworkExample,
		RunE: func(_ *cobra.Command, _ []string) error {
			req, err := f.Validate()
			if err != nil {
				return err
			}

			if err = handler(req); err != nil {
				return err
			}

			switch rf.Output {
			case flags.InteractiveOutput:
				PrintRenameNetworkResponse(w, f)
			case flags.JSONOutput:
				return nil
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&f.Network,
		"network", "n",
		"",
		"Network to rename",
	)
	cmd.Flags().StringVar(&f.NewName,
		"new-name",
		"",
		"New name for the network",
	)

	autoCompleteNetwork(cmd, rf.Home)

	return cmd
}

type RenameNetworkFlags struct {
	Network string
	NewName string
}

func (f *RenameNetworkFlags) Validate() (api.AdminRenameNetworkParams, error) {
	if len(f.Network) == 0 {
		return api.AdminRenameNetworkParams{}, flags.MustBeSpecifiedError("network")
	}

	if len(f.NewName) == 0 {
		return api.AdminRenameNetworkParams{}, flags.MustBeSpecifiedError("new-name")
	}

	return api.AdminRenameNetworkParams{
		Network: f.Network,
		NewName: f.NewName,
	}, nil
}

func PrintRenameNetworkResponse(w io.Writer, f *RenameNetworkFlags) {
	p := printer.NewInteractivePrinter(w)

	str := p.String()
	defer p.Print(str)

	str.CheckMark().SuccessText("Network ").SuccessBold(f.Network).SuccessText(" has been renamed to ").SuccessBold(f.NewName).SuccessText(".").NextLine()
}
