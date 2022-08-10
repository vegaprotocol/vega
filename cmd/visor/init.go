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
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/visor"
	"github.com/spf13/cobra"
)

const withDataNodeFlagName = "home"

func init() {
	rootCmd.AddCommand(initCmd)

	initCmd.Flags().String(homeFlagName, "", "Path to visor home folder to be generated")
	initCmd.MarkFlagRequired(homeFlagName)

	initCmd.Flags().Bool(withDataNodeFlagName, false, "Determines whether or not data node config should be also generated")
	initCmd.MarkFlagRequired(withDataNodeFlagName)
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
