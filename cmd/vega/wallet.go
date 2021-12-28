package main

import (
	"context"
	"os"

	"code.vegaprotocol.io/vegawallet/cmd"

	"github.com/jessevdk/go-flags"
)

type walletCmd struct{}

func (opts *walletCmd) Execute(_ []string) error {
	os.Args = os.Args[1:]

	writer := &cmd.Writer{
		Out: os.Stdout,
		Err: os.Stderr,
	}
	cmd.Execute(writer)

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
