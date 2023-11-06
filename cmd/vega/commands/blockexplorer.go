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

	cmd "code.vegaprotocol.io/vega/cmd/blockexplorer/commands"

	"github.com/jessevdk/go-flags"
)

type blockExplorerCmd struct{}

func (opts *blockExplorerCmd) Execute(_ []string) error {
	os.Args = os.Args[1:]
	return cmd.Execute(context.Background())
}

func BlockExplorer(ctx context.Context, parser *flags.Parser) error {
	_, err := parser.AddCommand(
		"blockexplorer",
		"The vega block explorer backend",
		"The vega block explorer backend",
		&blockExplorerCmd{},
	)

	return err
}
