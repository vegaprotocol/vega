package main

import (
	"context"
	"os"
	"path/filepath"

	"github.com/jessevdk/go-flags"

	cmd "github.com/tendermint/tendermint/cmd/tendermint/commands"
	"github.com/tendermint/tendermint/cmd/tendermint/commands/debug"
	cfg "github.com/tendermint/tendermint/config"
	"github.com/tendermint/tendermint/libs/cli"
	nm "github.com/tendermint/tendermint/node"
)

type tmCmd struct {
	Help []bool `short:"h" long:"help" description:"Show this help message"`
}

func (opts *tmCmd) Execute(_ []string) error {

	os.Args = os.Args[1:]
	rootCmd := cmd.RootCmd
	rootCmd.AddCommand(
		cmd.GenValidatorCmd,
		cmd.InitFilesCmd,
		cmd.ProbeUpnpCmd,
		cmd.LightCmd,
		cmd.ReplayCmd,
		cmd.ReplayConsoleCmd,
		cmd.ResetAllCmd,
		cmd.ResetPrivValidatorCmd,
		cmd.ShowValidatorCmd,
		cmd.TestnetFilesCmd,
		cmd.ShowNodeIDCmd,
		cmd.GenNodeKeyCmd,
		cmd.VersionCmd,
		debug.DebugCmd,
		cli.NewCompletionCmd(rootCmd, true),
	)

	nodeFunc := nm.DefaultNewNode

	// Create & start node
	rootCmd.AddCommand(cmd.NewRunNodeCmd(nodeFunc))

	cmd := cli.PrepareBaseCmd(rootCmd, "TM", os.ExpandEnv(filepath.Join("$HOME", cfg.DefaultTendermintDir)))
	if err := cmd.Execute(); err != nil {
		panic(err)
	}

	return nil
}

func Tm(ctx context.Context, parser *flags.Parser) error {
	_, err := parser.AddCommand(
		"tm",
		"Run tendermint nodes",
		"Run a tendermint node",
		&tmCmd{},
	)

	return err
}
