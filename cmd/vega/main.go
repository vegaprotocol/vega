package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"code.vegaprotocol.io/vega/cmd/vega/genesis"
	"code.vegaprotocol.io/vega/cmd/vega/nodewallet"
	"code.vegaprotocol.io/vega/config"
	"code.vegaprotocol.io/vega/logging"
	"github.com/jessevdk/go-flags"
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
		Faucet,
		Gateway,
		genesis.Genesis,
		Init,
		Node,
		nodewallet.NodeWallet,
		Verify,
		Version,
		Wallet,
		Watch,
		Tm,
	); err != nil {
		fmt.Printf("%+v\n", err)
		return err
	}

	if _, err := parser.Parse(); err != nil {
		return err
	}
	return nil
}

// waitSig will wait for a sigterm or sigint interrupt.
func waitSig(ctx context.Context, log *logging.Logger) {
	var gracefulStop = make(chan os.Signal, 1)
	signal.Notify(gracefulStop, syscall.SIGTERM)
	signal.Notify(gracefulStop, syscall.SIGINT)

	select {
	case sig := <-gracefulStop:
		log.Info("Caught signal", logging.String("name", fmt.Sprintf("%+v", sig)))
	case <-ctx.Done():
		// nothing to do
	}
}
