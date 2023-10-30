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
	vgterm "code.vegaprotocol.io/vega/libs/term"
	"code.vegaprotocol.io/vega/paths"
	"code.vegaprotocol.io/vega/wallet/service"
	svcStoreV1 "code.vegaprotocol.io/vega/wallet/service/store/v1"

	"github.com/spf13/cobra"
)

var (
	resetServiceConfigLong = cli.LongDesc(`
	    Reset the service configuration to its defaults.
	`)

	resetServiceConfigExample = cli.Examples(`
		# Reset the service configuration
		{{.Software}} service config reset
	`)
)

type ResetServiceConfigHandler func() (*service.Config, error)

func NewCmdResetServiceConfig(w io.Writer, rf *RootFlags) *cobra.Command {
	h := func() (*service.Config, error) {
		vegaPaths := paths.New(rf.Home)

		svcStore, err := svcStoreV1.InitialiseStore(vegaPaths)
		if err != nil {
			return nil, fmt.Errorf("couldn't initialise service store: %w", err)
		}

		defaultCfg := service.DefaultConfig()

		if err := svcStore.SaveConfig(defaultCfg); err != nil {
			return nil, fmt.Errorf("could not save the default service configuration: %w", err)
		}

		return defaultCfg, nil
	}

	return BuildCmdResetServiceConfig(w, h, rf)
}

func BuildCmdResetServiceConfig(w io.Writer, handler ResetServiceConfigHandler, rf *RootFlags) *cobra.Command {
	f := &ResetServiceConfigFlags{}

	cmd := &cobra.Command{
		Use:     "reset",
		Short:   "Reset the service configuration to its defaults",
		Long:    resetServiceConfigLong,
		Example: resetServiceConfigExample,
		RunE: func(_ *cobra.Command, _ []string) error {
			if !f.Force && vgterm.HasTTY() {
				if !flags.AreYouSure() {
					return nil
				}
			}

			cfg, err := handler()
			if err != nil {
				return err
			}

			switch rf.Output {
			case flags.InteractiveOutput:
				PrintResetServiceConfigResponse(w, cfg)
			case flags.JSONOutput:
				return printer.FprintJSON(w, cfg)
			}

			return nil
		},
	}

	cmd.Flags().BoolVarP(&f.Force,
		"force", "f",
		false,
		"Do not ask for confirmation",
	)

	return cmd
}

type ResetServiceConfigFlags struct {
	Force bool
}

func (f *ResetServiceConfigFlags) Validate() error {
	if !f.Force && vgterm.HasNoTTY() {
		return ErrForceFlagIsRequiredWithoutTTY
	}

	return nil
}

func PrintResetServiceConfigResponse(w io.Writer, cfg *service.Config) {
	p := printer.NewInteractivePrinter(w)

	str := p.String()
	defer p.Print(str)

	str.CheckMark().SuccessText("The service configuration has been reset.").NextSection()
	str.Text("Service URL: ").WarningText(cfg.Server.String()).NextSection()
	str.Text("Log level: ").WarningText(cfg.LogLevel.String()).NextSection()
	str.Text("API V1").NextLine()
	str.Pad().Text("Maximum token duration: ").WarningText(cfg.APIV1.MaximumTokenDuration.String()).NextSection()
	str.Text("API V2").NextLine()
	str.Pad().Text("Nodes:").NextLine()
	str.Pad().Pad().Text("Maximum retry per request: ").WarningText(fmt.Sprintf("%d", cfg.APIV2.Nodes.MaximumRetryPerRequest)).NextLine()
	str.Pad().Pad().Text("Maximum request duration: ").WarningText(cfg.APIV2.Nodes.MaximumRequestDuration.String()).NextLine()
}
