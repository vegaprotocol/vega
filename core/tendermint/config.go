// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.VEGA file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

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
