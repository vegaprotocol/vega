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
