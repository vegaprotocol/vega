package cmd

import (
	"io"

	"github.com/spf13/cobra"
)

func NewCmdTransaction(w io.Writer, rf *RootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "transaction",
		Short: "Provides utilities for interacting with transactions",
		Long:  "Provides utilities for interacting with transactions",
	}

	cmd.AddCommand(NewCmdCheckTransaction(w, rf))
	cmd.AddCommand(NewCmdSendTransaction(w, rf))
	cmd.AddCommand(NewCmdSignTransaction(w, rf))
	return cmd
}
