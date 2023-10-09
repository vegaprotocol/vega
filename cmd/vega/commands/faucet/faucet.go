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

package faucet

import (
	"context"

	"github.com/jessevdk/go-flags"
)

type Cmd struct {
	Init faucetInit `command:"init" description:"Generates the faucet configuration"`
	Run  faucetRun  `command:"run"  description:"Runs the faucet"`
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
