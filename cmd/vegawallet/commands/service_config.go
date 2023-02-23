package cmd

import (
	"io"

	"github.com/spf13/cobra"
)

func NewCmdServiceConfig(w io.Writer, rf *RootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage the Vega wallet's service configuration",
		Long:  "Manage the Vega wallet's service configuration",
	}

	cmd.AddCommand(NewCmdLocateServiceConfig(w, rf))
	cmd.AddCommand(NewCmdDescribeServiceConfig(w, rf))
	return cmd
}
