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
	DryRun         bool   `long:"dry-run" description:"Display the genesis file without writing it"`
	TmHome         string `short:"t" long:"tm-home" description:"The home path of tendermint"`
	GenesisFile    string `short:"g" long:"genesis-file" description:"A genesis file to be updated"`
	CheckpointPath string `long:"checkpoint-path" required:"true" description:"The path to the checkpoint file to load"`
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
