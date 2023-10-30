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
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"code.vegaprotocol.io/vega/core/genesis"
	vgtm "code.vegaprotocol.io/vega/core/tendermint"
	"code.vegaprotocol.io/vega/core/types"
	vgfs "code.vegaprotocol.io/vega/libs/fs"
	"code.vegaprotocol.io/vega/logging"
	"github.com/jessevdk/go-flags"
	tmjson "github.com/tendermint/tendermint/libs/json"
	tmtypes "github.com/tendermint/tendermint/types"
)

type loadCheckpointCmd struct {
	DryRun         bool   `description:"Display the genesis file without writing it" long:"dry-run"`
	TmHome         string `description:"The home path of tendermint"                 long:"tm-home"         short:"t"`
	GenesisFile    string `description:"A genesis file to be updated"                long:"genesis-file"    short:"g"`
	CheckpointPath string `description:"The path to the checkpoint file to load"     long:"checkpoint-path" required:"true"`
}

func (opts *loadCheckpointCmd) Execute(_ []string) error {
	log := logging.NewLoggerFromConfig(
		logging.NewDefaultConfig(),
	)
	defer log.AtExit()

	if _, err := flags.NewParser(opts, flags.Default|flags.IgnoreUnknown).Parse(); err != nil {
		return err
	}

	var (
		genesisDoc *tmtypes.GenesisDoc
		err        error
		tmConfig   *vgtm.Config
	)

	if len(opts.GenesisFile) > 0 {
		genesisDoc, _, err = readGenesisFile(opts.GenesisFile)
	} else {
		tmConfig = vgtm.NewConfig(opts.TmHome)
		genesisDoc, _, err = tmConfig.Genesis()
	}
	if err != nil {
		return fmt.Errorf("couldn't get genesis file: %w", err)
	}

	appState := genesis.State{}
	err = json.Unmarshal(genesisDoc.AppState, &appState)
	if err != nil {
		return err
	}

	f, err := os.Open(opts.CheckpointPath)
	if err != nil {
		return err
	}

	splits := strings.Split(filepath.Base(f.Name()), "-")
	if len(splits) != 3 {
		return fmt.Errorf("invalid checkpoint file name: `%v`", f.Name())
	}

	expectHash := strings.TrimSuffix(splits[2], ".cp")
	f.Close()

	buf, err := ioutil.ReadFile(opts.CheckpointPath)
	if err != nil {
		return err
	}

	cpt := &types.CheckpointState{}
	if err := cpt.SetState(buf); err != nil {
		return fmt.Errorf("invalid restore checkpoint command: %w", err)
	}

	if hex.EncodeToString(cpt.Hash) != expectHash {
		return fmt.Errorf("invalid hash, file name have hash `%s` but content hash is `%s`", expectHash, hex.EncodeToString(cpt.Hash))
	}

	appState.Checkpoint.CheckpointHash = expectHash
	appState.Checkpoint.CheckpointState = base64.StdEncoding.EncodeToString(buf)

	if err := vgtm.AddAppStateToGenesis(genesisDoc, &appState); err != nil {
		return fmt.Errorf("couldn't add app_state to genesis: %w", err)
	}

	if !opts.DryRun {
		if len(opts.GenesisFile) > 0 {
			if err := genesisDoc.SaveAs(opts.GenesisFile); err != nil {
				return fmt.Errorf("couldn't save the genesis file: %w", err)
			}
		} else {
			if err := tmConfig.SaveGenesis(genesisDoc); err != nil {
				return fmt.Errorf("couldn't save genesis: %w", err)
			}
		}
	}

	prettifiedDoc, err := vgtm.Prettify(genesisDoc)
	if err != nil {
		return err
	}
	fmt.Println(prettifiedDoc)
	return nil
}

func readGenesisFile(path string) (*tmtypes.GenesisDoc, *genesis.State, error) {
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
