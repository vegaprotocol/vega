package main

import (
	"context"
	"os"
	"path/filepath"

	"github.com/jessevdk/go-flags"
	tmcmd "github.com/tendermint/tendermint/cmd/tendermint/commands"
	tmdebug "github.com/tendermint/tendermint/cmd/tendermint/commands/debug"
	tmcfg "github.com/tendermint/tendermint/config"
	tmcli "github.com/tendermint/tendermint/libs/cli"
)

func Tm(_ context.Context, parser *flags.Parser) error {
	_, err := parser.AddCommand(
		"tendermint",
		"Tendermint utilities",
		"Run a tendermint node",
		&tmCmd{},
	)

	return err
}

type tmCmd struct{}

func (opts *tmCmd) Execute(_ []string) error {
	os.Args = os.Args[1:]
	rootCmd := tmcmd.RootCmd
	rootCmd.AddCommand(
		tmcmd.GenValidatorCmd,
		tmcmd.ReIndexEventCmd,
		// tmcmd.InitFilesCmd,
		tmcmd.ProbeUpnpCmd,
		tmcmd.LightCmd,
		tmcmd.ReplayCmd,
		tmcmd.ReplayConsoleCmd,
		tmcmd.ResetAllCmd,
		tmcmd.ResetPrivValidatorCmd,
		tmcmd.ShowValidatorCmd,
		tmcmd.TestnetFilesCmd,
		tmcmd.ShowNodeIDCmd,
		tmcmd.GenNodeKeyCmd,
		tmcmd.VersionCmd,
		tmcmd.InspectCmd,
		tmcmd.MakeKeyMigrateCommand(),
		tmdebug.DebugCmd,
		tmcli.NewCompletionCmd(rootCmd, true),
	)

	// rootCmd.AddCommand(NewRunNodeCmd())

	baseCmd := tmcli.PrepareBaseCmd(rootCmd, "TM", os.ExpandEnv(filepath.Join("$HOME", tmcfg.DefaultTendermintDir)))
	if err := baseCmd.Execute(); err != nil {
		return err
	}

	return nil
}
