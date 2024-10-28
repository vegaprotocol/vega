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

	"code.vegaprotocol.io/vega/cmd/resign/emit_withdrawals"

	"github.com/jessevdk/go-flags"
)

type emitWithdrawalsCmd struct{}

func (opts *emitWithdrawalsCmd) Execute(_ []string) error {
	os.Args = os.Args[1:]
	emit_withdrawals.Main()

	return nil
}

func EmitWithdrawals(ctx context.Context, parser *flags.Parser) error {
	_, err := parser.AddCommand(
		"emit_withdrawals",
		"Emit unclaimed withdrawals on vega",
		"Emit unclaimed withdrawals on vega",
		&emitWithdrawalsCmd{},
	)

	return err
}
