package main

import (
	"context"
	"fmt"

	"code.vegaprotocol.io/shared/paths"
	"code.vegaprotocol.io/vega/config"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/snapshot"

	"github.com/jessevdk/go-flags"
)

type Config struct {
	DBPath string `long:"db-path" description:"Path to database"`
}

type SnapshotListCmd struct {
	Config
	config.VegaHomeFlag
}

var snapshotListCmd SnapshotListCmd

func (cmd *SnapshotListCmd) Execute(args []string) error {
	log := logging.NewLoggerFromConfig(
		logging.NewDefaultConfig(),
	)
	defer log.AtExit()

	if _, err := flags.NewParser(cmd, flags.Default|flags.IgnoreUnknown).Parse(); err != nil {
		return err
	}

	var dbPath string

	if cmd.DBPath == "" {
		vegaPaths := paths.New(cmd.VegaHome)
		dbPath = vegaPaths.StatePathFor(paths.SnapshotStateHome)
	} else {
		dbPath = paths.StatePath(cmd.DBPath).String()
	}

	found, err := snapshot.AvailableSnapshotsHeights(dbPath)
	if err != nil {
		return err
	}

	if len(found) > 0 {
		fmt.Println("Snapshots available:", len(found))
		for _, snap := range found {
			fmt.Printf("\tHeight %d, version: %d, size %d, hash: %x\n", snap.Height, snap.Version, snap.Size, snap.Hash)
		}
	} else {
		fmt.Println("No snapshots available")
	}

	return nil
}

func SnapshotList(ctx context.Context, parser *flags.Parser) error {
	snapshotListCmd = SnapshotListCmd{
		Config: Config{},
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
