// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

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
