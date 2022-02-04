package main

import (
	"context"
	"os"

	"github.com/jessevdk/go-flags"
	tmcmd "github.com/tendermint/tendermint/cmd/tendermint/commands"
	tmdebug "github.com/tendermint/tendermint/cmd/tendermint/commands/debug"
	tmcfg "github.com/tendermint/tendermint/config"
	tmcli "github.com/tendermint/tendermint/libs/cli"
	tmlog "github.com/tendermint/tendermint/libs/log"
)

func Tm(_ context.Context, parser *flags.Parser) error {
	_, err := parser.AddCommand(
		"tm",
		"Run tendermint nodes",
		"Run a tendermint node",
		&tmCmd{},
	)

	return err
}

type tmCmd struct{}

func (opts *tmCmd) Execute(_ []string) error {
	os.Args = os.Args[1:]
	conf, err := tmcmd.ParseConfig(tmcfg.DefaultConfig())
	if err != nil {
		panic(err)
	}

	logger, err := tmlog.NewDefaultLogger(conf.LogFormat, conf.LogLevel)
	if err != nil {
		panic(err)
	}

	rcmd := tmcmd.RootCommand(conf, logger)
	rcmd.AddCommand(
		tmcmd.MakeGenValidatorCommand(),
		tmcmd.MakeReindexEventCommand(conf, logger),
		tmcmd.MakeInitFilesCommand(conf, logger),
		tmcmd.MakeLightCommand(conf, logger),
		tmcmd.MakeReplayCommand(conf, logger),
		tmcmd.MakeReplayConsoleCommand(conf, logger),
		tmcmd.MakeResetAllCommand(conf, logger),
		tmcmd.MakeResetPrivateValidatorCommand(conf, logger),
		tmcmd.MakeShowValidatorCommand(conf, logger),
		tmcmd.MakeTestnetFilesCommand(conf, logger),
		tmcmd.MakeShowNodeIDCommand(conf),
		tmcmd.GenNodeKeyCmd,
		tmcmd.VersionCmd,
		tmcmd.MakeInspectCommand(conf, logger),
		tmcmd.MakeRollbackStateCommand(conf),
		tmcmd.MakeKeyMigrateCommand(conf, logger),
		tmdebug.DebugCmd,
		tmcli.NewCompletionCmd(rcmd, true),
	)

	// Create & start node
	rcmd.AddCommand(NewRunNodeCmd(conf, logger))
	if err := rcmd.Execute(); err != nil {
		return err
	}

	return nil
}
