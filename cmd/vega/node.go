package main

import (
	"context"
	"fmt"
	"os"

	"code.vegaprotocol.io/vega/cmd/vega/node"
	"code.vegaprotocol.io/vega/config"
	"code.vegaprotocol.io/vega/logging"
	"github.com/jessevdk/go-flags"
)

type NodeCmd struct {
	config.Passphrase `long:"nodewallet-passphrase"`
	OldPath           string `short:"C" description:"[deprecated (use -r)] Path of the root directory in which the configuration will be located" env:"VEGA_CONFIG"`
	config.RootPathFlag

	config.Config
}

var nodeCmd NodeCmd

func (cmd *NodeCmd) Execute(args []string) error {
	log := logging.NewLoggerFromConfig(
		logging.NewDefaultConfig(),
	)
	defer log.AtExit()

	pass, err := cmd.Passphrase.Get("node wallet")
	if err != nil {
		return err
	}

	// we define this option to parse the cli args each time the config is
	// loaded. So that we can respect the cli flag presedence.
	parseFlagOpt := func(cfg *config.Config) error {
		_, err := flags.NewParser(cfg, flags.Default|flags.IgnoreUnknown).Parse()
		return err
	}

	var rootPath = cmd.RootPath
	if cmd.OldPath != "" {
		rootPath = cmd.OldPath
		fmt.Fprintf(os.Stderr, `
WARNING: Using -C is deprecated, please use -r
`)
	}
	cfgwatchr, err := config.NewFromFile(context.Background(), log, rootPath, rootPath, config.Use(parseFlagOpt))
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
		pass,
		args,
	)
}

func Node(ctx context.Context, parser *flags.Parser) error {
	rootPath := config.NewRootPathFlag()
	nodeCmd = NodeCmd{
		RootPathFlag: rootPath,
		Config:       config.NewDefaultConfig(rootPath.RootPath),
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
