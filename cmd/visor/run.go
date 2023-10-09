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
	"fmt"

	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/visor"
	"code.vegaprotocol.io/vega/visor/client"
	"github.com/spf13/cobra"
)

const homeFlagName = "home"

func init() {
	rootCmd.AddCommand(runCmd)

	runCmd.Flags().String(homeFlagName, "", "Path to visor home folder")
	runCmd.MarkFlagRequired(homeFlagName)
}

var runCmd = &cobra.Command{
	Use:          "run",
	Short:        "Runs visor",
	SilenceUsage: false,
	RunE: func(cmd *cobra.Command, args []string) error {
		homePath, err := cmd.Flags().GetString(homeFlagName)
		if err != nil {
			return err
		}

		log := logging.NewDevLogger()

		runner, err := visor.NewVisor(cmd.Context(), log, client.NewClientFactory(log), homePath)
		if err != nil {
			return fmt.Errorf("failed to create new runner: %w", err)
		}

		return runner.Run(cmd.Context())
	},
}
