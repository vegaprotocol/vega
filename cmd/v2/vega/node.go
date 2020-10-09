package main

import (
	"context"

	"code.vegaprotocol.io/vega/cmd/v2/vega/node"
	"code.vegaprotocol.io/vega/logging"
	"github.com/jessevdk/go-flags"
)

type NodeCmd struct {
}

var nodeCmd NodeCmd

func (cmd *NodeCmd) Execute(args []string) error {
	log := logging.NewLoggerFromConfig(
		logging.NewDefaultConfig(),
	)
	defer log.AtExit()

	return (&node.NodeCommand{
		Log: log,
	}).Execute(args)
}

func Node(ctx context.Context, parser *flags.Parser) error {
	nodeCmd = NodeCmd{}
	_, err := parser.AddCommand("node", "Runs a vega node", "Runs a vega node as defined by the config files", &nodeCmd)
	return err
}
