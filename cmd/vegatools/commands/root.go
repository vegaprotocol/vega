package cmd

import (
	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands.
var rootCmd = &cobra.Command{
	Use:   "vegatools",
	Short: "A collection of tools to speak with a vega node",
}

// Execute is the main function of `cmd` package.
// Usually called by the `main.main()`.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.AddCommand(streamCmd)
	rootCmd.AddCommand(snapshotDBCmd)
	rootCmd.AddCommand(checkpointCmd)
}
