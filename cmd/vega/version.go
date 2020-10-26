package main

import (
	"context"
	"fmt"

	"github.com/jessevdk/go-flags"
)

type VersionCmd struct {
	version string
	hash    string
}

func (cmd *VersionCmd) Execute(_ []string) error {
	fmt.Printf("Vega CLI %s (%s)\n", cmd.version, cmd.hash)
	return nil
}

var versionCmd VersionCmd

func Version(ctx context.Context, parser *flags.Parser) error {
	versionCmd = VersionCmd{
		version: CLIVersion,
		hash:    CLIVersionHash,
	}

	_, err := parser.AddCommand("version", "", "", &versionCmd)
	return err
}
