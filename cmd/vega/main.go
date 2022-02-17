package main

import (
	"context"
	"fmt"
	"os"

	"github.com/jessevdk/go-flags"

	"code.vegaprotocol.io/vega/cmd/vega/faucet"
	"code.vegaprotocol.io/vega/cmd/vega/genesis"
	"code.vegaprotocol.io/vega/cmd/vega/nodewallet"
	"code.vegaprotocol.io/vega/cmd/vega/paths"
	"code.vegaprotocol.io/vega/config"
)

var (
	// CLIVersionHash specifies the git commit used to build the application. See VERSION_HASH in Makefile for details.
	CLIVersionHash = ""

	// CLIVersion specifies the version used to build the application. See VERSION in Makefile for details.
	CLIVersion = ""
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

func main() {
	ctx := context.Background()
	if err := Main(ctx); err != nil {
		os.Exit(-1)
	}
}

func Main(ctx context.Context) error {
	// special case for the tendermint subcommand, so we bypass the command line
	if len(os.Args) >= 2 {
		switch os.Args[1] {
		case "tm":
			return (&tmCmd{}).Execute(nil)
		case "wallet":
			return (&walletCmd{}).Execute(nil)
		}
	}

	parser := flags.NewParser(&config.Empty{}, flags.Default)

	if err := Register(ctx, parser,
		faucet.Faucet,
		genesis.Genesis,
		Init,
		Node,
		nodewallet.NodeWallet,
		Verify,
		Version,
		Wallet,
		Watch,
		Tm,
		Checkpoint,
		Query,
		Bridge,
		paths.Paths,
		UnsafeResetAll,
		SnapshotList,
		AnnounceNode,
	); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		return err
	}

	if _, err := parser.Parse(); err != nil {
		return err
	}
	return nil
}
