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

package main

import (
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/visor"

	"github.com/spf13/cobra"
)

const withDataNodeFlagName = "with-data-node"

func init() {
	rootCmd.AddCommand(initCmd)

	initCmd.Flags().String(homeFlagName, "", "Path to visor home folder to be generated")
	initCmd.MarkFlagRequired(homeFlagName)

	initCmd.Flags().Bool(withDataNodeFlagName, false, "Determines whether or not data node config should be also generated")
}

var initCmd = &cobra.Command{
	Use:          "init",
	Short:        "Initiates home folder for visor",
	SilenceUsage: false,
	RunE: func(cmd *cobra.Command, args []string) error {
		homePath, err := cmd.Flags().GetString(homeFlagName)
		if err != nil {
			return err
		}

		withDataNode, err := cmd.Flags().GetBool(withDataNodeFlagName)
		if err != nil {
			return err
		}

		log := logging.NewDevLogger()

		return visor.Init(log, homePath, withDataNode)
	},
}
