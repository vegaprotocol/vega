// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package commands

import (
	"context"
	"errors"
	"fmt"
	"runtime/debug"

	"code.vegaprotocol.io/vega/libs/memory"

	"code.vegaprotocol.io/vega/cmd/vega/commands/node"
	"code.vegaprotocol.io/vega/core/config"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/paths"

	"github.com/jessevdk/go-flags"
)

type StartCmd struct {
	config.Passphrase `long:"nodewallet-passphrase-file" description:"A file contain the passphrase to decrypt the node wallet"`
	config.VegaHomeFlag
	config.Config

	TendermintHome string `long:"tendermint-home" description:"Directory for tendermint config and data (default: $HOME/.cometbft)"`

	Network    string `long:"network" description:"The network to start this node with"`
	NetworkURL string `long:"network-url" description:"The URL to a genesis file to start this node with"`
}

var startCmd StartCmd

const namedLogger = "core"

func (cmd *StartCmd) Execute([]string) error {
	log := logging.NewLoggerFromConfig(
		logging.NewDefaultConfig())
	logCore := log.Named(namedLogger)

	defer func() {
		log.AtExit()
		logCore.AtExit()
	}()

	// we define this option to parse the cli args each time the config is
	// loaded. So that we can respect the cli flag precedence.
	parseFlagOpt := func(cfg *config.Config) error {
		_, err := flags.NewParser(cfg, flags.Default|flags.IgnoreUnknown).Parse()
		return err
	}

	vegaPaths := paths.New(cmd.VegaHome)

	if len(cmd.Network) > 0 && len(cmd.NetworkURL) > 0 {
		return errors.New("--network-url and --network cannot be set together")
	}

	confWatcher, err := config.NewWatcher(context.Background(), logCore, vegaPaths, config.Use(parseFlagOpt))
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

	// setup max memory usage
	memFactor, err := confWatcher.Get().GetMaxMemoryFactor()
	if err != nil {
		return err
	}

	// only set max memory if user didn't require 100%
	if memFactor != 1 {
		totalMem, err := memory.TotalMemory()
		if err != nil {
			return fmt.Errorf("failed to get total memory: %w", err)
		}
		debug.SetMemoryLimit(int64(float64(totalMem) * memFactor))
	}

	if len(startCmd.TendermintHome) <= 0 {
		startCmd.TendermintHome = "$HOME/.cometbft"
	}

	return (&node.Command{
		Log: logCore,
	}).Run(
		confWatcher,
		vegaPaths,
		pass,
		cmd.TendermintHome,
		cmd.NetworkURL,
		cmd.Network,
		log,
	)
}

func Start(ctx context.Context, parser *flags.Parser) error {
	startCmd = StartCmd{
		Config: config.NewDefaultConfig(),
	}
	cmd, err := parser.AddCommand("start", "Start a vega instance", "Runs a vega node", &startCmd)
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

func Node(ctx context.Context, parser *flags.Parser) error {
	startCmd = StartCmd{
		Config: config.NewDefaultConfig(),
	}
	cmd, err := parser.AddCommand("node", "deprecated, see vega start instead", "deprecated, use vega start instead", &startCmd)
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
