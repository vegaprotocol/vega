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
	"code.vegaprotocol.io/vega/wallet/service"
	svcStoreV1 "code.vegaprotocol.io/vega/wallet/service/store/v1"

	"github.com/spf13/cobra"
)

var (
	describeServiceConfigLong = cli.LongDesc(`
	    Describe the service configuration.
	`)

	describeServiceConfigExample = cli.Examples(`
		# Describe the service configuration
		{{.Software}} service config describe
	`)
)

type DescribeServiceConfigHandler func() (*service.Config, error)

func NewCmdDescribeServiceConfig(w io.Writer, rf *RootFlags) *cobra.Command {
	h := func() (*service.Config, error) {
		vegaPaths := paths.New(rf.Home)

		svcStore, err := svcStoreV1.InitialiseStore(vegaPaths)
		if err != nil {
			return nil, fmt.Errorf("couldn't initialise service store: %w", err)
		}

		cfg, err := svcStore.GetConfig()
		if err != nil {
			return nil, fmt.Errorf("could not retrieve the service configuration: %w", err)
		}

		return cfg, nil
	}

	return BuildCmdDescribeServiceConfig(w, h, rf)
}

func BuildCmdDescribeServiceConfig(w io.Writer, handler DescribeServiceConfigHandler, rf *RootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "describe",
		Short:   "Describe the service configuration",
		Long:    describeServiceConfigLong,
		Example: describeServiceConfigExample,
		RunE: func(_ *cobra.Command, _ []string) error {
			cfg, err := handler()
			if err != nil {
				return err
			}

			switch rf.Output {
			case flags.InteractiveOutput:
				PrintDescribeServiceConfigResponse(w, cfg)
			case flags.JSONOutput:
				return printer.FprintJSON(w, cfg)
			}

			return nil
		},
	}

	return cmd
}

func PrintDescribeServiceConfigResponse(w io.Writer, cfg *service.Config) {
	p := printer.NewInteractivePrinter(w)

	str := p.String()
	defer p.Print(str)

	str.NextLine()
	str.Text("Service URL: ").WarningText(cfg.Server.String()).NextSection()
	str.Text("Log level: ").WarningText(cfg.LogLevel.String()).NextSection()
	str.Text("API V1").NextLine()
	str.Pad().Text("Maximum token duration: ").WarningText(cfg.APIV1.MaximumTokenDuration.String()).NextSection()
	str.Text("API V2").NextLine()
	str.Pad().Text("Nodes:").NextLine()
	str.Pad().Pad().Text("Maximum retry per request: ").WarningText(fmt.Sprintf("%d", cfg.APIV2.Nodes.MaximumRetryPerRequest)).NextLine()
	str.Pad().Pad().Text("Maximum request duration: ").WarningText(cfg.APIV2.Nodes.MaximumRequestDuration.String()).NextLine()
}
