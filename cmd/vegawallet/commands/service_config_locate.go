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
	svcStoreV1 "code.vegaprotocol.io/vega/wallet/service/store/v1"

	"github.com/spf13/cobra"
)

var (
	locateServiceConfigLong = cli.LongDesc(`
		Locate the wallet service configuration file.
	`)

	locateServiceConfigExample = cli.Examples(`
		# Locate the wallet service configuration file
		{{.Software}} service config locate
	`)
)

type LocateServiceConfigsResponse struct {
	Path string `json:"path"`
}

type LocateServiceConfigsHandler func() (*LocateServiceConfigsResponse, error)

func NewCmdLocateServiceConfig(w io.Writer, rf *RootFlags) *cobra.Command {
	h := func() (*LocateServiceConfigsResponse, error) {
		vegaPaths := paths.New(rf.Home)

		svcConfig, err := svcStoreV1.InitialiseStore(vegaPaths)
		if err != nil {
			return nil, fmt.Errorf("couldn't initialise service store: %w", err)
		}

		return &LocateServiceConfigsResponse{
			Path: svcConfig.GetServiceConfigsPath(),
		}, nil
	}

	return BuildCmdLocateServiceConfigs(w, h, rf)
}

func BuildCmdLocateServiceConfigs(w io.Writer, handler LocateServiceConfigsHandler, rf *RootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "locate",
		Short:   " Locate the wallet service configuration file",
		Long:    locateServiceConfigLong,
		Example: locateServiceConfigExample,
		RunE: func(_ *cobra.Command, _ []string) error {
			resp, err := handler()
			if err != nil {
				return err
			}

			switch rf.Output {
			case flags.InteractiveOutput:
				PrintLocateServiceConfigsResponse(w, resp)
			case flags.JSONOutput:
				return printer.FprintJSON(w, resp)
			}

			return nil
		},
	}

	return cmd
}

func PrintLocateServiceConfigsResponse(w io.Writer, resp *LocateServiceConfigsResponse) {
	p := printer.NewInteractivePrinter(w)

	str := p.String()
	defer p.Print(str)

	str.Text("The service configuration file is located at: ").SuccessText(resp.Path).NextLine()
}
