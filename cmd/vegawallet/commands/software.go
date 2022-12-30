package cmd

import (
	"io"

	"github.com/spf13/cobra"
)

func NewCmdSoftware(w io.Writer, rf *RootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "software",
		Short: "Retrieve the technical details of the software",
		Long:  "Retrieve the technical details of the software",
	}

	cmd.AddCommand(NewCmdSoftwareVersion(w, rf))
	cmd.AddCommand(NewCmdSoftwareCompatibility(w, rf))
	return cmd
}
