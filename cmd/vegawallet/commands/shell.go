package cmd

import (
	"io"

	"github.com/spf13/cobra"
)

func NewCmdShell(w io.Writer, rf *RootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "shell",
		Short: "Manage the shell integration of the software",
		Long:  "Manage the shell integration of the software",
	}

	cmd.AddCommand(NewCmdShellCompletion(w))
	return cmd
}
