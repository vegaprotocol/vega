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
	"fmt"
	"runtime/debug"

	"code.vegaprotocol.io/vega/cmd/data-node/commands/start"
	"code.vegaprotocol.io/vega/datanode/config"
	"code.vegaprotocol.io/vega/libs/memory"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/paths"
	"code.vegaprotocol.io/vega/version"

	"github.com/jessevdk/go-flags"
)

type StartCmd struct {
	config.VegaHomeFlag

	config.Config
}

var startCmd StartCmd

const namedLogger = "datanode"

func (cmd *StartCmd) Execute(args []string) error {
	ctx, cfunc := context.WithCancel(context.Background())
	defer cfunc()
	log := logging.NewLoggerFromConfig(
		logging.NewDefaultConfig()).Named(namedLogger)
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

	// setup max memory usage
	memFactor, err := configWatcher.Get().GetMaxMemoryFactor()
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

	return (&start.NodeCommand{
		Log:         log,
		Version:     version.Get(),
		VersionHash: version.GetCommitHash(),
	}).Run(
		ctx,
		configWatcher,
		vegaPaths,
		args,
	)
}

func Node(ctx context.Context, parser *flags.Parser) error {
	startCmd = StartCmd{
		Config: config.NewDefaultConfig(),
	}
	cmd, err := parser.AddCommand("node", "deprecated, see data-node start instead", "deprecated, see data-node start instead", &startCmd)
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

func Start(_ context.Context, parser *flags.Parser) error {
	startCmd = StartCmd{
		Config: config.NewDefaultConfig(),
	}
	cmd, err := parser.AddCommand("start", "Start a vega data node", "Start a vega data node as defined by the config files", &startCmd)
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
