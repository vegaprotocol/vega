package main

import (
	"context"
	"fmt"

	"github.com/jessevdk/go-flags"
)

type VersionCmd struct {
	version string
	hash    string
	Help    bool `short:"h" long:"help" description:"Show this help message"`
}

func (cmd *VersionCmd) Execute(_ []string) error {
	if cmd.Help {
		return &flags.Error{
			Type:    flags.ErrHelp,
			Message: "vega version subcommand help",
		}
	}
	fmt.Printf("Vega CLI %s (%s)\n", cmd.version, cmd.hash)
	return nil
}

var versionCmd VersionCmd

func Version(ctx context.Context, parser *flags.Parser) error {
	versionCmd = VersionCmd{
		version: CLIVersion,
		hash:    CLIVersionHash,
	}

	_, err := parser.AddCommand("version", "Show version info", "Show version info", &versionCmd)
	return err
}
