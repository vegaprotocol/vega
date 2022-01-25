package main

import (
	"context"
	"fmt"

	"code.vegaprotocol.io/shared/paths"
	"github.com/jessevdk/go-flags"

	"code.vegaprotocol.io/vega/cmd/vega/snapshots"
	"code.vegaprotocol.io/vega/config"
	"code.vegaprotocol.io/vega/logging"
)

type SnapshotListCmd struct {
	config.Config
	config.VegaHomeFlag
}

var snapshotListCmd SnapshotListCmd

func (cmd *SnapshotListCmd) Execute(args []string) error {
	log := logging.NewLoggerFromConfig(
		logging.NewDefaultConfig(),
	)
	defer log.AtExit()

	parseFlagOpt := func(cfg *config.Config) error {
		_, err := flags.NewParser(cfg, flags.Default|flags.IgnoreUnknown).Parse()
		return err
	}

	vegaPaths := paths.New(cmd.VegaHome)

	confWatcher, err := config.NewWatcher(context.Background(), log, vegaPaths, config.Use(parseFlagOpt))
	if err != nil {
		return err
	}

	conf := confWatcher.Get()

	log = logging.NewLoggerFromConfig(conf.Logging)

	dbPath := vegaPaths.StatePathFor(paths.SnapshotStateHome)
	found, err := snapshots.AvailableSnapshotsHeights(dbPath)
	if err != nil {
		log.Error("Faile to get snapshots heights", logging.Error(err))
		return err
	}

	if len(found) > 0 {
		fmt.Printf("Snapshots available: %d", len(found))
		for height, snap := range found {
			fmt.Printf("\tHeight %d, version: %d, hash: %d\n", height, snap.Version, snap.Hash)
		}
	}
	return nil
}

func SnapshotList(ctx context.Context, parser *flags.Parser) error {
	snapshotListCmd = SnapshotListCmd{
		Config: config.NewDefaultConfig(),
	}
	cmd, err := parser.AddCommand("snapshots", "Lists snapshots", "List the block-heights of the snapshots saved to disk", &snapshotListCmd)
	if err != nil {
		return err
	}

	for _, parent := range cmd.Groups() {
		for _, grp := range parent.Groups() {
			grp.ShortDescription = parent.ShortDescription + "::" + grp.ShortDescription
		}
	}
	return nil
}
