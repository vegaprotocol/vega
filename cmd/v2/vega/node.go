package main

import (
	"context"

	"code.vegaprotocol.io/vega/cmd/v2/vega/node"
	"code.vegaprotocol.io/vega/config"
	"code.vegaprotocol.io/vega/logging"
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

	cfgwatchr, err := config.NewFromFile(context.Background(), log, cmd.RootPath, cmd.RootPath)
	if err != nil {
		return err
	}

	// reload the config on update keeping user specified flags
	cfgwatchr.Use(func(cfg *config.Config) {
		cmd.Config = *cfg
		if _, err := flags.Parse(cmd); err != nil {
			log.Error("Couldn't parse config", logging.Error(err))
			return
		}
		*cfg = cmd.Config
	})

	cmd.Config = cfgwatchr.Get()
	if _, err := flags.Parse(cmd); err != nil {
		return err
	}

	return (&node.NodeCommand{
		Log: log,
	}).Execute(args)
}

func Node(ctx context.Context, parser *flags.Parser) error {
	rootPath := config.NewRootPathFlag()
	nodeCmd = NodeCmd{
		RootPathFlag: rootPath,
		Config:       config.NewDefaultConfig(rootPath.RootPath),
	}
	_, err := parser.AddCommand("node", "Runs a vega node", "Runs a vega node as defined by the config files", &nodeCmd)
	return err
}
