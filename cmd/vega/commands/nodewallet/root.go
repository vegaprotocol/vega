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

package nodewallet

import (
	"context"

	"code.vegaprotocol.io/vega/core/admin"
	"code.vegaprotocol.io/vega/core/config"
	"code.vegaprotocol.io/vega/core/nodewallets"

	"github.com/fatih/color"
	"github.com/jessevdk/go-flags"
)

var (
	yellow = color.New(color.FgYellow).SprintFunc()
	green  = color.New(color.FgGreen).SprintFunc()
)

type RootCmd struct {
	// Global options
	config.VegaHomeFlag
	config.PassphraseFlag

	// Subcommands
	Show     showCmd     `command:"show"     description:"List the wallets registers into the nodewallet"`
	Generate generateCmd `command:"generate" description:"Generate and register a wallet into the nodewallet"`
	Import   importCmd   `command:"import"   description:"Import the configuration of a wallet required by the vega node"`
	Verify   verifyCmd   `command:"verify"   description:"Verify the configuration imported in the nodewallet"`
	Reload   reloadCmd   `command:"reload"   description:"Reload node wallet of a running node instance"`
}

var rootCmd RootCmd

func NodeWallet(ctx context.Context, parser *flags.Parser) error {
	rootCmd = RootCmd{
		Generate: generateCmd{
			Config: nodewallets.NewDefaultConfig(),
		},
		Import: importCmd{
			Config: nodewallets.NewDefaultConfig(),
		},
		Verify: verifyCmd{
			Config: nodewallets.NewDefaultConfig(),
		},
		Reload: reloadCmd{
			Config: admin.NewDefaultConfig(),
		},
	}

	var (
		short = "Manages the node wallet"
		long  = `The nodewallet is a wallet owned by the vega node, it contains all
	the information to login to other wallets from external blockchain that
	vega will need to run properly (e.g and ethereum wallet, which allow vega
	to sign transaction to be verified on the ethereum blockchain) available
	wallet: eth, vega`
	)

	_, err := parser.AddCommand("nodewallet", short, long, &rootCmd)
	return err
}
