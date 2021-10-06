package faucet

import (
	"context"

	"github.com/jessevdk/go-flags"
)

type Cmd struct {
	Init faucetInit `command:"init" description:"Generates the faucet configuration"`
	Run  faucetRun  `command:"run" description:"Runs the faucet"`
}

// faucetCmd is a global variable that holds generic options for the faucet
// sub-commands.
var faucetCmd Cmd

func Faucet(ctx context.Context, parser *flags.Parser) error {
	faucetCmd = Cmd{
		Init: faucetInit{},
		Run: faucetRun{
			ctx: ctx,
		},
	}

	_, err := parser.AddCommand("faucet", "Allow deposit of builtin asset", "", &faucetCmd)
	return err
}
