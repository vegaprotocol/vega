package cmd

import (
	"io"

	"code.vegaprotocol.io/vega/cmd/vegawallet/commands/cli"
	"github.com/spf13/cobra"
)

var sessionLong = cli.LongDesc(`
		Manage the session tokens.

		These tokens are generated when a third-party application initiates a connections
        with a wallet.

		To avoid going through the connection flow everytime the wallet software is restarted,
        the software keeps track of the previous session tokens and restore them when the wallet
        associated to them is unlocked.
	`)

func NewCmdSession(w io.Writer, rf *RootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "session",
		Short: "Manage the session tokens",
		Long:  sessionLong,
	}

	cmd.AddCommand(NewCmdListSessions(w, rf))

	return cmd
}
