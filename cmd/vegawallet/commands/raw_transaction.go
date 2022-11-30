package cmd

import (
	"io"

	"github.com/spf13/cobra"
)

func NewCmdRawTransaction(w io.Writer, rf *RootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "raw_transaction",
		Short: "Provides utilities for interacting with raw transactions",
		Long:  "Provides utilities for interacting with raw transactions",
	}

	cmd.AddCommand(NewCmdRawTransactionSend(w, rf))
	return cmd
}
