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
	"code.vegaprotocol.io/vega/wallet/wallets"

	"github.com/spf13/cobra"
)

var (
	locateWalletsLong = cli.LongDesc(`
		Locate the folder in which all the wallet files are stored.
	`)

	locateWalletsExample = cli.Examples(`
		# Locate wallet files
		{{.Software}} locate
	`)
)

type LocateWalletsResponse struct {
	Path string `json:"path"`
}

type LocateWalletsHandler func() (*LocateWalletsResponse, error)

func NewCmdLocateWallets(w io.Writer, rf *RootFlags) *cobra.Command {
	h := func() (*LocateWalletsResponse, error) {
		vegaPaths := paths.New(rf.Home)

		walletStore, err := wallets.InitialiseStoreFromPaths(vegaPaths, false)
		if err != nil {
			return nil, fmt.Errorf("couldn't initialise networks store: %w", err)
		}

		return &LocateWalletsResponse{
			Path: walletStore.GetWalletsPath(),
		}, nil
	}

	return BuildCmdLocateWallets(w, h, rf)
}

func BuildCmdLocateWallets(w io.Writer, handler LocateWalletsHandler, rf *RootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "locate",
		Short:   "Locate the folder containing the wallet files",
		Long:    locateWalletsLong,
		Example: locateWalletsExample,
		RunE: func(_ *cobra.Command, _ []string) error {
			resp, err := handler()
			if err != nil {
				return err
			}

			switch rf.Output {
			case flags.InteractiveOutput:
				PrintLocateWalletsResponse(w, resp)
			case flags.JSONOutput:
				return printer.FprintJSON(w, resp)
			}

			return nil
		},
	}

	return cmd
}

func PrintLocateWalletsResponse(w io.Writer, resp *LocateWalletsResponse) {
	p := printer.NewInteractivePrinter(w)

	str := p.String()
	defer p.Print(str)

	str.Text("Wallet files are located at: ").SuccessText(resp.Path).NextLine()
}
