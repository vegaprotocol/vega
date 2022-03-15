package main

import (
	"context"

	"code.vegaprotocol.io/shared/paths"
	"code.vegaprotocol.io/vega/cmd/vega/node"
	"code.vegaprotocol.io/vega/cmd/vega/node2"
	"code.vegaprotocol.io/vega/config"
	"code.vegaprotocol.io/vega/logging"

	"github.com/jessevdk/go-flags"
)

type NodeCmd struct {
	config.Passphrase `long:"nodewallet-passphrase-file"`
	config.VegaHomeFlag

	config.Config
}

var nodeCmd NodeCmd

func (cmd *NodeCmd) Execute(args []string) error {
	log := logging.NewLoggerFromConfig(
		logging.NewDefaultConfig(),
	)
	defer log.AtExit()

	// we define this option to parse the cli args each time the config is
	// loaded. So that we can respect the cli flag precedence.
	parseFlagOpt := func(cfg *config.Config) error {
		_, err := flags.NewParser(cfg, flags.Default|flags.IgnoreUnknown).Parse()
		return err
	}

	vegaPaths := paths.New(cmd.VegaHome)

	confWatcher, err := config.NewWatcher(context.Background(), log, vegaPaths, config.Use(parseFlagOpt))
	if err != nil {
		return err
	}

	// only try to get the passphrase if the node is started
	// as a validator
	var pass string
	if confWatcher.Get().IsValidator() {
		pass, err = cmd.Get("node wallet", false)
		if err != nil {
			return err
		}
	}

	return (&node.NodeCommand{
		Log:         log,
		Version:     CLIVersion,
		VersionHash: CLIVersionHash,
	}).Run(
		confWatcher,
		vegaPaths,
		pass,
		args,
	)
}

func Node(ctx context.Context, parser *flags.Parser) error {
	nodeCmd = NodeCmd{
		Config: config.NewDefaultConfig(),
	}
	cmd, err := parser.AddCommand("node", "Runs a vega node", "Runs a vega node as defined by the config files", &nodeCmd)
	if err != nil {
		return err
	}

	// Print nested groups under parent's name using `::` as the separator.
	for _, parent := range cmd.Groups() {
		for _, grp := range parent.Groups() {
			grp.ShortDescription = parent.ShortDescription + "::" + grp.ShortDescription
		}
	}
	return nil
}

type StartCmd struct {
	config.Passphrase `long:"nodewallet-passphrase-file"`
	config.VegaHomeFlag

	config.Config
}

var startCmd StartCmd

func (cmd *StartCmd) Execute(args []string) error {
	log := logging.NewLoggerFromConfig(
		logging.NewDefaultConfig(),
	)
	defer log.AtExit()

	// we define this option to parse the cli args each time the config is
	// loaded. So that we can respect the cli flag precedence.
	parseFlagOpt := func(cfg *config.Config) error {
		_, err := flags.NewParser(cfg, flags.Default|flags.IgnoreUnknown).Parse()
		return err
	}

	vegaPaths := paths.New(cmd.VegaHome)

	confWatcher, err := config.NewWatcher(context.Background(), log, vegaPaths, config.Use(parseFlagOpt))
	if err != nil {
		return err
	}

	// only try to get the passphrase if the node is started
	// as a validator
	var pass string
	if confWatcher.Get().IsValidator() {
		pass, err = cmd.Get("node wallet", false)
		if err != nil {
			return err
		}
	}

	return (&node2.NodeCommand{
		Log:         log,
		Version:     CLIVersion,
		VersionHash: CLIVersionHash,
	}).Run(
		confWatcher,
		vegaPaths,
		pass,
		args,
	)
}

func Start(ctx context.Context, parser *flags.Parser) error {
	startCmd = StartCmd{
		Config: config.NewDefaultConfig(),
	}
	cmd, err := parser.AddCommand("start", "Start a vega run", "Runs a vega node as defined by the config files", &startCmd)
	if err != nil {
		return err
	}

	// Print nested groups under parent's name using `::` as the separator.
	for _, parent := range cmd.Groups() {
		for _, grp := range parent.Groups() {
			grp.ShortDescription = parent.ShortDescription + "::" + grp.ShortDescription
		}
	}
	return nil
}
