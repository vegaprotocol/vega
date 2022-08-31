package cmd

import (
	"github.com/spf13/cobra"

	"code.vegaprotocol.io/vega/vegatools/snapshotdb"
)

var (
	snapshotDBOpts struct {
		databasePath   string
		vegaHome       string
		outputPath     string
		showing        string
		heightToOutput uint64
	}

	snapshotDBCmd = &cobra.Command{
		Use:   "snapshotdb",
		Short: "Displays information about the snapshot database",
		RunE:  runSnapshotDBCmd,
	}
)

func init() {
	snapshotDBCmd.Flags().StringVarP(&snapshotDBOpts.databasePath, "db-path", "d", "", "path to the goleveldb database folder")
	snapshotDBCmd.Flags().StringVarP(&snapshotDBOpts.vegaHome, "home", "V", "", "path to the vega home folder")
	snapshotDBCmd.Flags().StringVarP(&snapshotDBOpts.outputPath, "out", "o", "", "file to write JSON to")
	snapshotDBCmd.Flags().StringVarP(&snapshotDBOpts.showing, "show", "s", "", "what to show. Allowed values: 'list', 'json', 'versions'")
	snapshotDBCmd.Flags().Uint64VarP(&snapshotDBOpts.heightToOutput, "block-height", "r", 0, "block-height of the snapshot to dump")
	_ = snapshotDBCmd.MarkPersistentFlagDirname("home")
}

func runSnapshotDBCmd(*cobra.Command, []string) error {
	return snapshotdb.ShowSnapshotData(
		snapshotDBOpts.databasePath,
		snapshotDBOpts.vegaHome,
		snapshotdb.ShowingFromString(snapshotDBOpts.showing),
		snapshotDBOpts.outputPath,
		snapshotDBOpts.heightToOutput,
	)
}
