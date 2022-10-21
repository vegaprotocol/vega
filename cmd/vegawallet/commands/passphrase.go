package cmd

import (
	"io"

	"github.com/spf13/cobra"
)

func NewCmdPassphrase(w io.Writer, rf *RootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "passphrase",
		Short: "Manage the wallet passphrase",
		Long:  "Manage the wallet passphrase",
	}

	cmd.AddCommand(NewCmdUpdatePassphrase(w, rf))
	return cmd
}
