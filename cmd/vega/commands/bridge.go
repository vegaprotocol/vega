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

	"code.vegaprotocol.io/vega/cmd/vega/commands/bridge"

	"github.com/jessevdk/go-flags"
)

type BridgeCmd struct {
	ERC20 *bridge.ERC20Cmd `command:"erc20" description:"Validator utilities to manage the erc20 bridge"`
}

var bridgeCmd BridgeCmd

func Bridge(ctx context.Context, parser *flags.Parser) error {
	bridgeCmd = BridgeCmd{
		ERC20: bridge.ERC20(),
	}

	_, err := parser.AddCommand("bridge", "Utilities to control / manage vega bridges", "", &bridgeCmd)
	return err
}
