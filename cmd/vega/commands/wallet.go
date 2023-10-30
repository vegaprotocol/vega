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

package commands

import (
	"context"
	"os"

	cmd "code.vegaprotocol.io/vega/cmd/vegawallet/commands"
	"github.com/jessevdk/go-flags"
)

type walletCmd struct{}

func (opts *walletCmd) Execute(_ []string) error {
	os.Args = os.Args[1:]

	writer := &cmd.Writer{
		Out: os.Stdout,
		Err: os.Stderr,
	}
	cmd.Execute(writer)

	return nil
}

func Wallet(ctx context.Context, parser *flags.Parser) error {
	_, err := parser.AddCommand(
		"wallet",
		"Run vega wallet",
		"Run the vega wallet",
		&walletCmd{},
	)

	return err
}
