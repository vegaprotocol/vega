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
	"encoding/json"

	tmtypes "github.com/cometbft/cometbft/types"
)

func FromJSON(rawGenesisDoc []byte) (*tmtypes.GenesisDoc, *State, error) {
	genesisDoc, err := tmtypes.GenesisDocFromJSON(rawGenesisDoc)
	if err != nil {
		return nil, nil, err
	}

	genesisState := &State{}
	err = json.Unmarshal(genesisDoc.AppState, genesisState)
	if err != nil {
		return nil, nil, err
	}
	return genesisDoc, genesisState, nil
}
