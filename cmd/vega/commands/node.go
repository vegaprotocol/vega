// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package commands

import (
	"context"
	"errors"
	"fmt"
	"runtime/debug"
	"time"

	"code.vegaprotocol.io/vega/cmd/vega/commands/node"
	"code.vegaprotocol.io/vega/core/config"
	"code.vegaprotocol.io/vega/core/evtforward"
	"code.vegaprotocol.io/vega/libs/memory"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/paths"

	"github.com/jessevdk/go-flags"
)

type StartCmd struct {
	config.Passphrase `description:"A file contain the passphrase to decrypt the node wallet" long:"nodewallet-passphrase-file"`
	config.VegaHomeFlag
	config.Config

	TendermintHome string `description:"Directory for tendermint config and data (default: $HOME/.cometbft)" long:"tendermint-home"`

	Network    string `description:"The network to start this node with"               long:"network"`
	NetworkURL string `description:"The URL to a genesis file to start this node with" long:"network-url"`
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

	// this is to migrate all validators configuration at once
	// and set the event forwarder to the appropriate value
	migrateConfig := func(cnf *config.Config) {
		sevenDays := 24 * 7 * time.Hour
		if cnf.EvtForward.KeepHashesDurationForTestOnlyDoNotChange.Duration == sevenDays {
			cnf.EvtForward.KeepHashesDurationForTestOnlyDoNotChange.Duration = evtforward.DefaultKeepHashesDuration
		}
	}

	confWatcher, err := config.NewWatcher(context.Background(), logCore, vegaPaths, migrateConfig, config.Use(parseFlagOpt))
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
