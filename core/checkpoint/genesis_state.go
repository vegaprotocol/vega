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

package checkpoint

import "encoding/json"

type GenesisState struct {
	CheckpointHash  string `json:"load_hash"`
	CheckpointState string `json:"state"`
}

func DefaultGenesisState() GenesisState {
	return GenesisState{} // default no hash
}

func LoadGenesisState(data []byte) (*GenesisState, error) {
	cp := &struct {
		Checkpoint *GenesisState `json:"checkpoint"`
	}{}
	if err := json.Unmarshal(data, &cp); err != nil {
		return nil, err
	}
	return cp.Checkpoint, nil
}
