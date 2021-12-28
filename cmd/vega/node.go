package main

import (
	"context"

	"code.vegaprotocol.io/shared/paths"
	"github.com/jessevdk/go-flags"

	"code.vegaprotocol.io/vega/cmd/vega/node"
	"code.vegaprotocol.io/vega/config"
	"code.vegaprotocol.io/vega/logging"
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

	pass, err := cmd.Passphrase.Get("node wallet", false)
	if err != nil {
		return err
	}

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
