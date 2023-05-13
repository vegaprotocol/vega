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
