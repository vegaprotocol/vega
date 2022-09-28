package commands

import (
	"context"
	"fmt"
	"os"

	"code.vegaprotocol.io/vega/cmd/vega/commands/faucet"
	"code.vegaprotocol.io/vega/cmd/vega/commands/genesis"
	"code.vegaprotocol.io/vega/cmd/vega/commands/nodewallet"
	"code.vegaprotocol.io/vega/cmd/vega/commands/paths"
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
		case "tendermint", "tm":
			return (&tmCmd{}).Execute(nil)
		case "wallet":
			return (&walletCmd{}).Execute(nil)
		case "datanode":
			return (&datanodeCmd{}).Execute(nil)
		case "tools":
			return (&toolsCmd{}).Execute(nil)
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
		BlockExplorer,
		Datanode,
		VegaTools,
		Watch,
		Tm,
		Tendermint,
		Query,
		Bridge,
		paths.Paths,
		UnsafeResetAll,
		AnnounceNode,
		ProposeProtocolUpgrade,
		Start,
		Node,
	); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		return err
	}

	if _, err := parser.Parse(); err != nil {
		return err
	}
	return nil
}
