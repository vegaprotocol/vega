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

func NewCmdServiceConfig(w io.Writer, rf *RootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage the Vega wallet's service configuration",
		Long:  "Manage the Vega wallet's service configuration",
	}

	cmd.AddCommand(NewCmdLocateServiceConfig(w, rf))
	cmd.AddCommand(NewCmdDescribeServiceConfig(w, rf))
	cmd.AddCommand(NewCmdResetServiceConfig(w, rf))
	return cmd
}
