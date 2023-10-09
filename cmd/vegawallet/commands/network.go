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

	"github.com/spf13/cobra"
)

func NewCmdNetwork(w io.Writer, rf *RootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "network",
		Short: "Manage networks",
		Long:  "Manage networks",
	}

	cmd.AddCommand(NewCmdListNetworks(w, rf))
	cmd.AddCommand(NewCmdImportNetwork(w, rf))
	cmd.AddCommand(NewCmdDescribeNetwork(w, rf))
	cmd.AddCommand(NewCmdLocateNetworks(w, rf))
	cmd.AddCommand(NewCmdDeleteNetwork(w, rf))
	cmd.AddCommand(NewCmdRenameNetwork(w, rf))
	return cmd
}
