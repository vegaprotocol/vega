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
	"fmt"
	"io"

	"code.vegaprotocol.io/vega/cmd/vegawallet/commands/cli"
	"code.vegaprotocol.io/vega/cmd/vegawallet/commands/flags"
	"code.vegaprotocol.io/vega/cmd/vegawallet/commands/printer"
	"code.vegaprotocol.io/vega/paths"
	netstore "code.vegaprotocol.io/vega/wallet/network/store/v1"

	"github.com/spf13/cobra"
)

var (
	locateNetworkLong = cli.LongDesc(`
		Locate the folder in which all the network configuration files are stored.
	`)

	locateNetworkExample = cli.Examples(`
		# Locate network configuration files
		{{.Software}} network locate
	`)
)

type LocateNetworksResponse struct {
	Path string `json:"path"`
}

type LocateNetworksHandler func() (*LocateNetworksResponse, error)

func NewCmdLocateNetworks(w io.Writer, rf *RootFlags) *cobra.Command {
	h := func() (*LocateNetworksResponse, error) {
		vegaPaths := paths.New(rf.Home)

		netStore, err := netstore.InitialiseStore(vegaPaths)
		if err != nil {
			return nil, fmt.Errorf("couldn't initialise networks store: %w", err)
		}

		return &LocateNetworksResponse{
			Path: netStore.GetNetworksPath(),
		}, nil
	}

	return BuildCmdLocateNetworks(w, h, rf)
}

func BuildCmdLocateNetworks(w io.Writer, handler LocateNetworksHandler, rf *RootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "locate",
		Short:   "Locate the folder of network configuration files",
		Long:    locateNetworkLong,
		Example: locateNetworkExample,
		RunE: func(_ *cobra.Command, _ []string) error {
			resp, err := handler()
			if err != nil {
				return err
			}

			switch rf.Output {
			case flags.InteractiveOutput:
				PrintLocateNetworksResponse(w, resp)
			case flags.JSONOutput:
				return printer.FprintJSON(w, resp)
			}

			return nil
		},
	}

	return cmd
}

func PrintLocateNetworksResponse(w io.Writer, resp *LocateNetworksResponse) {
	p := printer.NewInteractivePrinter(w)

	str := p.String()
	defer p.Print(str)

	str.Text("Network configuration files are located at: ").SuccessText(resp.Path).NextLine()
}
