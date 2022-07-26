package cmd

import (
	"io"

	"github.com/spf13/cobra"
)

func NewCmdPermissions(w io.Writer, rf *RootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "permissions",
		Short: "Manage permissions of a given wallet",
		Long:  "Manage permissions of a given wallet",
	}

	cmd.AddCommand(NewCmdListPermissions(w, rf))
	cmd.AddCommand(NewCmdDescribePermissions(w, rf))
	cmd.AddCommand(NewCmdPurgePermissions(w, rf))
	cmd.AddCommand(NewCmdRevokePermissions(w, rf))
	return cmd
}
