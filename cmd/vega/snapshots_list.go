package main

import (
	"context"
	"fmt"

	"code.vegaprotocol.io/shared/paths"
	"github.com/jessevdk/go-flags"

	"code.vegaprotocol.io/vega/config"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/snapshot"
	"code.vegaprotocol.io/vega/vegatime"
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
	timeService := vegatime.New(conf.Time)
	ctx, _ := context.WithCancel(context.Background())

	snapshotEngine, err := snapshot.New(ctx, vegaPaths, conf.Snapshot, log, timeService)
	if err != nil {
		return err
	}
	found, err := snapshotEngine.List()
	if err != nil {
		return err
	}

	if len(found) > 0 {
		fmt.Printf("Snapshots available: %d", len(found))
		for _, snap := range found {
			fmt.Printf("\tVersion: %d, chunks: %s\n", snap.Meta.Version, snap.Meta.ChunkHashes)
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
