// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package main

import (
	"fmt"

	"code.vegaprotocol.io/vega/visor"
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
	Short:        "Runs visor.",
	SilenceUsage: false,
	RunE: func(cmd *cobra.Command, args []string) error {
		homePath, err := cmd.Flags().GetString(homeFlagName)
		if err != nil {
			return err
		}

		runner, err := visor.NewVisor(cmd.Context(), homePath)
		if err != nil {
			return fmt.Errorf("failed to create new runner: %w", err)
		}

		return runner.Run(cmd.Context())
	},
}
