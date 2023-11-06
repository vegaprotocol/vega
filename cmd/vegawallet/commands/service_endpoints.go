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
	"os"
	"text/template"

	"code.vegaprotocol.io/vega/cmd/vegawallet/commands/cli"
	"code.vegaprotocol.io/vega/cmd/vegawallet/commands/flags"
	"code.vegaprotocol.io/vega/cmd/vegawallet/commands/printer"
	"code.vegaprotocol.io/vega/paths"
	svcStoreV1 "code.vegaprotocol.io/vega/wallet/service/store/v1"

	"github.com/spf13/cobra"
)

const startupT = ` # Authentication
 - login:                   POST   {{.WalletServiceLocalAddress}}/api/v1/auth/token
 - logout:                  DELETE {{.WalletServiceLocalAddress}}/api/v1/auth/token

 # Network management
 - network:                 GET    {{.WalletServiceLocalAddress}}/api/v1/network
 - network chainid:         GET    {{.WalletServiceLocalAddress}}/api/v1/network/chainid

 # Wallet management
 - create a wallet:         POST   {{.WalletServiceLocalAddress}}/api/v1/wallets
 - import a wallet:         POST   {{.WalletServiceLocalAddress}}/api/v1/wallets/import

 # Key pair management
 - generate a key pair:     POST   {{.WalletServiceLocalAddress}}/api/v1/keys
 - list keys:               GET    {{.WalletServiceLocalAddress}}/api/v1/keys
 - describe a key pair:     GET    {{.WalletServiceLocalAddress}}/api/v1/keys/:keyid
 - taint a key pair:        PUT    {{.WalletServiceLocalAddress}}/api/v1/keys/:keyid/taint
 - annotate a key pair:     PUT    {{.WalletServiceLocalAddress}}/api/v1/keys/:keyid/metadata

 # Commands
 - sign a command:          POST   {{.WalletServiceLocalAddress}}/api/v1/command
 - sign a command (sync):   POST   {{.WalletServiceLocalAddress}}/api/v1/command/sync
 - sign a command (commit): POST   {{.WalletServiceLocalAddress}}/api/v1/command/commit
 - sign data:               POST   {{.WalletServiceLocalAddress}}/api/v1/sign
 - verify data:             POST   {{.WalletServiceLocalAddress}}/api/v1/verify

 # Information
 - get service status:      GET    {{.WalletServiceLocalAddress}}/api/v1/status
 - get the version:         GET    {{.WalletServiceLocalAddress}}/api/v1/version
`

var (
	listEndpointsLong = cli.LongDesc(`
		List the Vega wallet service HTTP endpoints
	`)

	listEndpointsExample = cli.Examples(`
		# List service endpoints
		{{.Software}} endpoints --network NETWORK
	`)
)

type ListEndpointsHandler func(io.Writer, *RootFlags, *ListEndpointsFlags) error

func NewCmdListEndpoints(w io.Writer, rf *RootFlags) *cobra.Command {
	return BuildCmdListEndpoints(w, ListEndpoints, rf)
}

func BuildCmdListEndpoints(w io.Writer, handler ListEndpointsHandler, rf *RootFlags) *cobra.Command {
	f := &ListEndpointsFlags{}

	cmd := &cobra.Command{
		Use:     "endpoints",
		Short:   "List endpoints",
		Long:    listEndpointsLong,
		Example: listEndpointsExample,
		RunE: func(_ *cobra.Command, _ []string) error {
			if err := f.Validate(); err != nil {
				return err
			}

			if err := handler(w, rf, f); err != nil {
				return err
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&f.Network,
		"network", "n",
		"",
		"Network configuration to use",
	)

	return cmd
}

type ListEndpointsFlags struct {
	Network string
}

func (f *ListEndpointsFlags) Validate() error {
	if len(f.Network) == 0 {
		return flags.MustBeSpecifiedError("network")
	}

	return nil
}

func ListEndpoints(w io.Writer, rf *RootFlags, f *ListEndpointsFlags) error {
	p := printer.NewInteractivePrinter(w)

	vegaPaths := paths.New(rf.Home)
	svcStore, err := svcStoreV1.InitialiseStore(vegaPaths)
	if err != nil {
		return fmt.Errorf("couldn't initialise the service store: %w", err)
	}

	cfg, err := svcStore.GetConfig()
	if err != nil {
		return fmt.Errorf("couldn't retrieve the service configuration: %w", err)
	}

	str := p.String()
	defer p.Print(str)

	str.BlueArrow().InfoText("Available endpoints").NextLine()
	printServiceEndpoints(cfg.Server.String())
	str.NextLine()

	return nil
}

func printServiceEndpoints(serviceHost string) {
	params := struct {
		WalletServiceLocalAddress string
	}{
		WalletServiceLocalAddress: serviceHost,
	}

	tmpl, err := template.New("wallet-cmdline").Parse(startupT)
	if err != nil {
		panic(err)
	}
	err = tmpl.Execute(os.Stdout, params)
	if err != nil {
		panic(err)
	}
}
