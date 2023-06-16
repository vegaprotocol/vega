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

package genesis

import (
	"context"

	"code.vegaprotocol.io/vega/core/config"
	"github.com/jessevdk/go-flags"
)

type Cmd struct {
	// Global options
	config.VegaHomeFlag
	config.PassphraseFlag

	// Subcommands
	Generate       generateCmd       `command:"generate"        description:"Generates the genesis file"`
	Update         updateCmd         `command:"update"          description:"Update the genesis file with the app_state, useful if the genesis generation is not done using \"vega genesis generate\""`
	LoadCheckpoint loadCheckpointCmd `command:"load_checkpoint" description:"Load the given checkpoint file in the genesis file"`
}

var genesisCmd Cmd

func Genesis(ctx context.Context, parser *flags.Parser) error {
	genesisCmd = Cmd{
		Generate: generateCmd{
			TmHome: "$HOME/.cometbft",
		},
		Update: updateCmd{
			TmHome: "$HOME/.cometbft",
		},
		LoadCheckpoint: loadCheckpointCmd{
			TmHome: "$HOME/.cometbft",
		},
	}

	desc := "Manage the genesis file"
	cmd, err := parser.AddCommand("genesis", desc, desc, &genesisCmd)
	if err != nil {
		return err
	}
	return initNewCmd(ctx, cmd)
}
