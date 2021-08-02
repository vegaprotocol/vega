package main

import (
	"context"
	"os"

	"code.vegaprotocol.io/go-wallet/cmd"
	"github.com/jessevdk/go-flags"
)

type walletCmd struct {
	Help []bool `short:"h" long:"help" description:"Show this help message"`
}

func (opts *walletCmd) Execute(_ []string) error {
	os.Args = os.Args[1:]
	cmd.Execute()
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
