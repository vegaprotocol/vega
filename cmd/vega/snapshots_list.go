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

package main

import (
	"context"
	"fmt"

	"code.vegaprotocol.io/shared/paths"
	"code.vegaprotocol.io/vega/core/config"
	"code.vegaprotocol.io/vega/core/snapshot"
	"code.vegaprotocol.io/vega/logging"

	"github.com/jessevdk/go-flags"
)

type SnapshotListCmd struct {
	DBPath string `long:"db-path" description:"Path to database"`
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

	found, invalidVersions, err := snapshot.AvailableSnapshotsHeights(dbPath)
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

	if len(invalidVersions) > 0 {
		fmt.Println("Invalid versions:", len(invalidVersions))
		for _, snap := range found {
			fmt.Printf("\tVersion: %d, hash: %x\n", snap.Version, snap.Hash)
		}
	}

	return nil
}

func SnapshotList(ctx context.Context, parser *flags.Parser) error {
	snapshotListCmd = SnapshotListCmd{}
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
