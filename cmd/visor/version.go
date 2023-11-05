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

	vgjson "code.vegaprotocol.io/vega/libs/json"
	"code.vegaprotocol.io/vega/version"

	"github.com/spf13/cobra"
)

const (
	outputFlagName     = "output"
	outputFlagValJSON  = "json"
	outputFlagValHuman = "human"
)

func init() {
	rootCmd.AddCommand(versionCmd)
	versionCmd.Flags().String(outputFlagName, outputFlagValHuman, "Specify the output format: json,human")
}

var versionCmd = &cobra.Command{
	Use:          "version",
	Short:        "Returns a Vega Visor version",
	SilenceUsage: false,
	RunE: func(cmd *cobra.Command, args []string) error {
		output, err := cmd.Flags().GetString(outputFlagName)
		if err != nil {
			return err
		}

		switch output {
		case outputFlagValHuman:
			fmt.Printf("Vega Visor CLI %s (%s)\n", version.Get(), version.GetCommitHash())
			return nil
		case outputFlagValJSON:
			return vgjson.Print(struct {
				Version string `json:"version"`
				Hash    string `json:"hash"`
			}{
				Version: version.Get(),
				Hash:    version.GetCommitHash(),
			})
		default:
			return fmt.Errorf("%s flag must be either %q or %q", outputFlagName, outputFlagValHuman, outputFlagValJSON)
		}
	},
}
