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
