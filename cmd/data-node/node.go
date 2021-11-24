package main

import (
	"context"

	"code.vegaprotocol.io/data-node/cmd/data-node/node"
	"code.vegaprotocol.io/data-node/config"
	"code.vegaprotocol.io/data-node/logging"
	"code.vegaprotocol.io/shared/paths"
	"github.com/jessevdk/go-flags"
)

type NodeCmd struct {
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

	configWatcher, err := config.NewWatcher(context.Background(), log, vegaPaths, config.Use(parseFlagOpt))
	if err != nil {
		return err
	}

	return (&node.NodeCommand{
		Log:         log,
		Version:     CLIVersion,
		VersionHash: CLIVersionHash,
	}).Run(
		configWatcher,
		vegaPaths,
		args,
	)
}

func Node(ctx context.Context, parser *flags.Parser) error {
	nodeCmd = NodeCmd{
		Config: config.NewDefaultConfig(),
	}
	cmd, err := parser.AddCommand("node", "Runs a vega data node", "Runs a vega data node as defined by the config files", &nodeCmd)
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
