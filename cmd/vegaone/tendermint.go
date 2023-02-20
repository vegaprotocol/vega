package main

import (
	"os"

	tmcmd "github.com/tendermint/tendermint/cmd/cometbft/commands"
	tmdebug "github.com/tendermint/tendermint/cmd/cometbft/commands/debug"
	tmcli "github.com/tendermint/tendermint/libs/cli"
)

type tendermintCommand struct {
	args []string
}

func newTendermint(args []string) *tendermintCommand {
	return &tendermintCommand{args: args}
}

func (*tendermintCommand) Parse(args []string) error { return nil }

func (i *tendermintCommand) Execute() error {
	os.Args = i.args
	rootCmd := tmcmd.RootCmd
	rootCmd.AddCommand(
		tmcmd.GenValidatorCmd,
		tmcmd.InitFilesCmd,
		tmcmd.ProbeUpnpCmd,
		tmcmd.LightCmd,
		tmcmd.ReplayCmd,
		tmcmd.ReplayConsoleCmd,
		tmcmd.ResetAllCmd,
		tmcmd.ResetPrivValidatorCmd,
		tmcmd.ResetStateCmd,
		tmcmd.ShowValidatorCmd,
		tmcmd.TestnetFilesCmd,
		tmcmd.ShowNodeIDCmd,
		tmcmd.GenNodeKeyCmd,
		tmcmd.VersionCmd,
		tmcmd.RollbackStateCmd,
		tmcmd.CompactGoLevelDBCmd,
		tmdebug.DebugCmd,
		tmcli.NewCompletionCmd(rootCmd, true),
	)

	baseCmd := tmcli.PrepareBaseCmd(rootCmd, "TM", os.ExpandEnv("$HOME/.vegaone/tendermint"))
	if err := baseCmd.Execute(); err != nil {
		return err
	}

	return nil

}
