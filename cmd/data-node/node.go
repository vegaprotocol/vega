package main

import (
	"context"

	"code.vegaprotocol.io/data-node/cmd/data-node/node"
	"code.vegaprotocol.io/data-node/config"
	"code.vegaprotocol.io/data-node/logging"
	"github.com/jessevdk/go-flags"
)

type NodeCmd struct {
	config.RootPathFlag

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

	cfgwatchr, err := config.NewFromFile(context.Background(), log, cmd.RootPath, cmd.RootPath, config.Use(parseFlagOpt))
	if err != nil {
		return err
	}

	return (&node.NodeCommand{
		Log:         log,
		Version:     CLIVersion,
		VersionHash: CLIVersionHash,
	}).Run(
		cfgwatchr,
		cmd.RootPath,
		args,
	)
}

func Node(ctx context.Context, parser *flags.Parser) error {
	rootPath := config.NewRootPathFlag()
	nodeCmd = NodeCmd{
		RootPathFlag: rootPath,
		Config:       config.NewDefaultConfig(rootPath.RootPath),
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
