package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"code.vegaprotocol.io/vega/logging"
	"github.com/jessevdk/go-flags"
)

// Subcommand is the signature of a sub command that can be registered.
type Subcommand func(*flags.Parser) error

// Register registers one or more subcommands.
func Register(parser *flags.Parser, cmds ...Subcommand) error {
	for _, fn := range cmds {
		if err := fn(parser); err != nil {
			return err
		}
	}
	return nil
}

func main() {
	if err := Main(os.Args[1:]...); err != nil {
		os.Exit(-1)
	}
}

func Main(args ...string) error {
	parser := flags.NewParser(&Empty{}, flags.Default)

	Register(parser,
		Faucet,
		Gateway,
		Wallet,
		Watch,
	)

	if _, err := parser.ParseArgs(args); err != nil {
		switch t := err.(type) {
		case *flags.Error:
			if t.Type != flags.ErrHelp {
				parser.WriteHelp(os.Stdout)
			}
		}
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
