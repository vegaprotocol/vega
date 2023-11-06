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

package tendermint

import (
	"encoding/json"
	"fmt"
	"os"

	"code.vegaprotocol.io/vega/core/genesis"
	vgfs "code.vegaprotocol.io/vega/libs/fs"

	tmconfig "github.com/cometbft/cometbft/config"
	tmcrypto "github.com/cometbft/cometbft/crypto"
	tmjson "github.com/cometbft/cometbft/libs/json"
	tmos "github.com/cometbft/cometbft/libs/os"
	"github.com/cometbft/cometbft/privval"
	tmtypes "github.com/cometbft/cometbft/types"
)

type Config struct {
	config *tmconfig.Config
}

func NewConfig(home string) *Config {
	c := tmconfig.DefaultConfig()
	c.SetRoot(os.ExpandEnv(home))
	return &Config{
		config: c,
	}
}

func (c *Config) PublicValidatorKey() (tmcrypto.PubKey, error) {
	privValKeyFile := c.config.PrivValidatorKeyFile()
	if !tmos.FileExists(privValKeyFile) {
		return nil, fmt.Errorf("file \"%s\" not found", privValKeyFile)
	}
	// read private validator
	pv := privval.LoadFilePV(
		c.config.PrivValidatorKeyFile(),
		c.config.PrivValidatorStateFile(),
	)

	pubKey, err := pv.GetPubKey()
	if err != nil {
		return nil, fmt.Errorf("can't get tendermint public key: %w", err)
	}

	return pubKey, nil
}

func (c *Config) Genesis() (*tmtypes.GenesisDoc, *genesis.State, error) {
	path := c.config.GenesisFile()
	data, err := vgfs.ReadFile(path)
	if err != nil {
		return nil, nil, fmt.Errorf("couldn't read genesis file: %w", err)
	}

	doc := &tmtypes.GenesisDoc{}
	err = tmjson.Unmarshal(data, doc)
	if err != nil {
		return nil, nil, fmt.Errorf("couldn't unmarshal the genesis document: %w", err)
	}

	state := &genesis.State{}

	if len(doc.AppState) != 0 {
		if err := json.Unmarshal(doc.AppState, state); err != nil {
			return nil, nil, fmt.Errorf("couldn't unmarshal genesis state: %w", err)
		}
	}

	return doc, state, nil
}

func (c *Config) SaveGenesis(doc *tmtypes.GenesisDoc) error {
	path := c.config.GenesisFile()

	if err := doc.SaveAs(path); err != nil {
		return fmt.Errorf("couldn't save the genesis file: %w", err)
	}

	return nil
}

func AddAppStateToGenesis(doc *tmtypes.GenesisDoc, state *genesis.State) error {
	rawGenesisState, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("couldn't marshal the genesis state as JSON: %w", err)
	}

	doc.AppState = rawGenesisState

	return nil
}
