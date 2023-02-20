package main

import (
	"os"

	cmd "code.vegaprotocol.io/vega/cmd/vegawallet/commands"
)

type walletCommand struct {
	args []string
}

func newWallet(args []string) *walletCommand {
	return &walletCommand{args: args}
}

func (*walletCommand) Parse(args []string) error { return nil }

func (i *walletCommand) Execute() error {
	writer := &cmd.Writer{
		Out: os.Stdout,
		Err: os.Stderr,
	}
	cmd.Execute(writer)

	return nil
}
