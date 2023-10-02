package main

import (
	"flag"
	"fmt"
	"os"
)

type globalFlags struct {
	Home string
}

func (g *globalFlags) Register(fset *flag.FlagSet) {
	fset.StringVar(&g.Home, "home", "$HOME/.vegaone", "the root home for all vega configurations and state")
}

type Command interface {
	Execute() error
	Parse([]string) error
}

func main() {
	if len(os.Args) <= 1 {
		printUsage()
	}

	cmdName := os.Args[1]
	os.Args = os.Args[1:]
	var cmd Command
	switch cmdName {
	case "init":
		cmd = newInit()
	case "start":
		cmd = newStart()
	case "wallet":
		cmd = newWallet(os.Args)
	case "tendermint":
		cmd = newTendermint(os.Args)
	case "version":
		cmd = newVersion()
	case "help", "-h", "--help":
		printUsage()
	default:
		printUnknownCommand(cmdName)
	}

	if err := cmd.Parse(os.Args[1:]); err != nil {
		printError(err)
	}

	if err := cmd.Execute(); err != nil {
		printError(err)
	}
}

func printUnknownCommand(cmdName string) {
	fmt.Printf(`vegaone %s: unknown command
Run 'vegaone help' for usage.
`, cmdName)
	os.Exit(2)
}

func printUsage() {
	fmt.Printf("%v\n", help)
	os.Exit(0)
}

func printError(err error) {
	fmt.Printf("error: %v\n", err)
	os.Exit(1)
}

const help = `Vegaone is a tool for operating a Vega node.

Usage:

	vegaone <command> [arguments]

The commands are:

	init         initialise a new vega node
	start        start a vega node
	tendermint   manage the tendermint state
	version      show the protocol version
	wallet       use the vega wallet
`
