package cmd

import (
	"code.vegaprotocol.io/vega/vegatools/checkpoint"

	"github.com/spf13/cobra"
)

var (
	cp struct {
		inPath   string
		outPath  string
		format   string
		validate bool
		create   bool
		dummy    bool
	}

	checkpointCmd = &cobra.Command{
		Use:   "checkpoint",
		Short: "Make checkpoint human-readable, or generate checkpoint from human readable format",
		RunE:  parseCheckpoint,
	}
)

func init() {
	checkpointCmd.Flags().StringVarP(&cp.inPath, "file", "f", "", "input file to parse")
	checkpointCmd.Flags().StringVarP(&cp.outPath, "out", "o", "", "output file to write to [default is STDOUT]")
	checkpointCmd.Flags().BoolVarP(&cp.validate, "validate", "v", false, "validate contents of the checkpoint file")
	checkpointCmd.Flags().BoolVarP(&cp.create, "generate", "g", false, "input is human readable, generate checkpoint file")
	checkpointCmd.Flags().BoolVarP(&cp.dummy, "dummy", "d", false, "generate a dummy file [added for debugging, but could be useful]")
	_ = checkpointCmd.MarkFlagRequired("file")
}

func parseCheckpoint(*cobra.Command, []string) error {
	return checkpoint.Run(
		cp.inPath,
		cp.outPath,
		cp.create,
		cp.validate,
		cp.dummy,
	)
}
