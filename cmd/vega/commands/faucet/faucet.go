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

package faucet

import (
	"context"

	"github.com/jessevdk/go-flags"
)

type Cmd struct {
	Init faucetInit `command:"init" description:"Generates the faucet configuration"`
	Run  faucetRun  `command:"run" description:"Runs the faucet"`
}

// faucetCmd is a global variable that holds generic options for the faucet
// sub-commands.
var faucetCmd Cmd

func Faucet(ctx context.Context, parser *flags.Parser) error {
	faucetCmd = Cmd{
		Init: faucetInit{},
		Run: faucetRun{
			ctx: ctx,
		},
	}

	_, err := parser.AddCommand("faucet", "Allow deposit of builtin asset", "", &faucetCmd)
	return err
}
