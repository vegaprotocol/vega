package metadata

import (
	"fmt"
	"strconv"

	"code.vegaprotocol.io/vega/libs/proto"

	tmtypes "github.com/tendermint/tendermint/abci/types"
)

type Adapter interface {
	Save(version []byte, state []byte) error
	Load(version []byte) (state []byte, err error)
	Close() error
	Clear() error
}

type Database struct {
	Adapter
}

func (d *Database) Save(version int64, state *tmtypes.Snapshot) error {
	serializedVersion := strconv.FormatInt(version, 10)
	serializedState, err := proto.Marshal(state)
	if err != nil {
		return fmt.Errorf("could not serialize snaspshot state: %w", err)
	}

	return d.Adapter.Save([]byte(serializedVersion), serializedState)
}

func (d *Database) Load(version int64) (*tmtypes.Snapshot, error) {
	bufV := strconv.FormatInt(version, 10)
	state, err := d.Adapter.Load([]byte(bufV))
	if err != nil {
		return nil, err
	}

	snapshot := &tmtypes.Snapshot{}
	if err := proto.Unmarshal(state, snapshot); err != nil {
		return nil, fmt.Errorf("could not deserialize snapshot state: %w", err)
	}

	return snapshot, nil
}

func NewDatabase(adapter Adapter) *Database {
	return &Database{
		Adapter: adapter,
	}
}

func noMetadataForSnapshotVersion(version []byte) error {
	return fmt.Errorf("no metadata found for snapshot version %q", version)
}
