package genesis

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	vgfs "code.vegaprotocol.io/shared/libs/fs"
	"code.vegaprotocol.io/vega/genesis"
	"code.vegaprotocol.io/vega/logging"
	vgtm "code.vegaprotocol.io/vega/tendermint"
	"code.vegaprotocol.io/vega/types"
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

	appState := genesis.GenesisState{}
	err = json.Unmarshal(genesisDoc.AppState, &appState)
	if err != nil {
		return err
	}

	f, err := os.Open(opts.CheckpointPath)
	if err != nil {
		return err
	}

	splits := strings.Split(f.Name(), "-")
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

func readGenesisFile(path string) (*tmtypes.GenesisDoc, *genesis.GenesisState, error) {
	data, err := vgfs.ReadFile(path)
	if err != nil {
		return nil, nil, fmt.Errorf("couldn't read genesis file: %w", err)
	}

	doc := &tmtypes.GenesisDoc{}
	err = tmjson.Unmarshal(data, doc)
	if err != nil {
		return nil, nil, fmt.Errorf("couldn't unmarshal the genesis document: %w", err)
	}

	state := &genesis.GenesisState{}

	if len(doc.AppState) != 0 {
		if err := json.Unmarshal(doc.AppState, state); err != nil {
			return nil, nil, fmt.Errorf("couldn't unmarshal genesis state: %w", err)
		}
	}

	return doc, state, nil
}
