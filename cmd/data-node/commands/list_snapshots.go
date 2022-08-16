package commands

import (
	"context"
	"fmt"
	"sort"

	"code.vegaprotocol.io/vega/datanode/snapshot"
	"code.vegaprotocol.io/vega/paths"

	"github.com/jessevdk/go-flags"

	"code.vegaprotocol.io/vega/core/config"
)

type ListSnapshotsCmd struct {
	config.VegaHomeFlag
}

func (cmd *ListSnapshotsCmd) Execute(_ []string) error {
	vegaPaths := paths.New(cmd.VegaHome)

	snapshots, err := snapshot.ListSnapshots(vegaPaths)
	if err != nil {
		return fmt.Errorf("failed to list snapshots:%w", err)
	}

	sort.Slice(snapshots, func(i, j int) bool {
		return snapshots[i].Height < snapshots[j].Height
	})

	fmt.Println("Available Snapshots:")
	for _, snapshot := range snapshots {
		fmt.Printf("\tHeight:%d \tChain ID:%s \n", snapshot.Height, snapshot.ChainId)
	}

	return nil
}

var listSnapshotsCmd ListSnapshotsCmd

func ListSnapshots(ctx context.Context, parser *flags.Parser) error {
	listSnapshotsCmd = ListSnapshotsCmd{}
	_, err := parser.AddCommand("snapshots", "Lists snapshots", "List the block-heights and chain ids of the available snapshots", &listSnapshotsCmd)
	return err
}
