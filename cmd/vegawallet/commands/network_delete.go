// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"

	"code.vegaprotocol.io/vega/cmd/vegawallet/commands/cli"
	"code.vegaprotocol.io/vega/cmd/vegawallet/commands/flags"
	"code.vegaprotocol.io/vega/cmd/vegawallet/commands/printer"
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

		_, errDetails := deleteNetwork.Handle(context.Background(), params)
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
