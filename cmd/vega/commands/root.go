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
	"fmt"
	"os"

	"code.vegaprotocol.io/vega/cmd/vega/commands/faucet"
	"code.vegaprotocol.io/vega/cmd/vega/commands/genesis"
	"code.vegaprotocol.io/vega/cmd/vega/commands/nodewallet"
	"code.vegaprotocol.io/vega/cmd/vega/commands/paths"
	tools "code.vegaprotocol.io/vega/cmd/vegatools"
	"code.vegaprotocol.io/vega/core/config"

	"github.com/jessevdk/go-flags"
)

// Subcommand is the signature of a sub command that can be registered.
type Subcommand func(context.Context, *flags.Parser) error

// Register registers one or more subcommands.
func Register(ctx context.Context, parser *flags.Parser, cmds ...Subcommand) error {
	for _, fn := range cmds {
		if err := fn(ctx, parser); err != nil {
			return err
		}
	}
	return nil
}

func Main(ctx context.Context) error {
	// special case for the tendermint subcommand, so we bypass the command line
	if len(os.Args) >= 2 {
		switch os.Args[1] {
		case "tendermint", "tm", "cometbft":
			return (&cometbftCmd{}).Execute(nil)
		case "wallet":
			return (&walletCmd{}).Execute(nil)
		case "datanode":
			return (&datanodeCmd{}).Execute(nil)
		case "blockexplorer":
			return (&blockExplorerCmd{}).Execute(nil)
		}
	}

	parser := flags.NewParser(&config.Empty{}, flags.Default)

	if err := Register(ctx, parser,
		faucet.Faucet,
		genesis.Genesis,
		Init,
		nodewallet.NodeWallet,
		Verify,
		Version,
		Wallet,
		Datanode,
		tools.VegaTools,
		Watch,
		Tm,
		Tendermint,
		CometBFT,
		Query,
		Bridge,
		paths.Paths,
		UnsafeResetAll,
		AnnounceNode,
		RotateEthKey,
		ProposeProtocolUpgrade,
		Start,
		Node,
		BlockExplorer,
		Prune,
	); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		return err
	}

	if _, err := parser.Parse(); err != nil {
		return err
	}
	return nil
}
