package tendermint

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	vgfs "code.vegaprotocol.io/shared/libs/fs"
	"code.vegaprotocol.io/vega/genesis"
	tmconfig "github.com/tendermint/tendermint/config"
	tmcrypto "github.com/tendermint/tendermint/crypto"
	tmjson "github.com/tendermint/tendermint/libs/json"
	tmos "github.com/tendermint/tendermint/libs/os"
	"github.com/tendermint/tendermint/privval"
	tmtypes "github.com/tendermint/tendermint/types"
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
	privValKeyFile := c.config.PrivValidator.KeyFile()
	privValStateFile := c.config.PrivValidator.StateFile()
	if !tmos.FileExists(privValKeyFile) {
		return nil, fmt.Errorf("file \"%s\" not found", privValKeyFile)
	}

	pv, err := privval.LoadFilePV(privValKeyFile, privValStateFile)
	if err != nil {
		return nil, err
	}

	pubKey, err := pv.GetPubKey(context.Background())
	if err != nil {
		return nil, fmt.Errorf("can't get tendermint public key: %w", err)
	}

	return pubKey, nil
}

func (c *Config) Genesis() (*tmtypes.GenesisDoc, *genesis.GenesisState, error) {
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

	state := &genesis.GenesisState{}

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

func AddAppStateToGenesis(doc *tmtypes.GenesisDoc, state *genesis.GenesisState) error {
	rawGenesisState, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("couldn't marshal the genesis state as JSON: %w", err)
	}

	doc.AppState = rawGenesisState

	return nil
}
