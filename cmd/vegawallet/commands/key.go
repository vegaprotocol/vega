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
	"fmt"
	"io"

	"code.vegaprotocol.io/vega/cmd/vegawallet/commands/printer"
	"code.vegaprotocol.io/vega/wallet/wallet"

	"github.com/spf13/cobra"
)

func NewCmdKey(w io.Writer, rf *RootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "key",
		Short: "Manage Vega wallets' keys",
		Long:  "Manage Vega wallets' keys",
	}

	cmd.AddCommand(NewCmdAnnotateKey(w, rf))
	cmd.AddCommand(NewCmdGenerateKey(w, rf))
	cmd.AddCommand(NewCmdIsolateKey(w, rf))
	cmd.AddCommand(NewCmdListKeys(w, rf))
	cmd.AddCommand(NewCmdDescribeKey(w, rf))
	cmd.AddCommand(NewCmdTaintKey(w, rf))
	cmd.AddCommand(NewCmdUntaintKey(w, rf))
	cmd.AddCommand(NewCmdRotateKey(w, rf))
	return cmd
}

func printMeta(str *printer.FormattedString, meta []wallet.Metadata) {
	padding := 0
	for _, m := range meta {
		keyLen := len(m.Key)
		if keyLen > padding {
			padding = keyLen
		}
	}

	for _, m := range meta {
		str.WarningText(fmt.Sprintf("%-*s", padding, m.Key)).Text(" | ").WarningText(m.Value).NextLine()
	}
}
