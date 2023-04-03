// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

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
	Show     showCmd     `command:"show" description:"List the wallets registers into the nodewallet"`
	Generate generateCmd `command:"generate" description:"Generate and register a wallet into the nodewallet"`
	Import   importCmd   `command:"import" description:"Import the configuration of a wallet required by the vega node"`
	Verify   verifyCmd   `command:"verify" description:"Verify the configuration imported in the nodewallet"`
	Reload   reloadCmd   `command:"reload" description:"Reload node wallet of a running node instance"`
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
