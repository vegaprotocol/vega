package metadata

import (
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
	adapter Adapter
}

func (d *Database) Save(version int64, state *tmtypes.Snapshot) error {
	bufV := strconv.FormatInt(version, 10)
	bufS, err := proto.Marshal(state)
	if err != nil {
		return err
	}

	return d.adapter.Save([]byte(bufV), bufS)
}

func (d *Database) Load(version int64) (*tmtypes.Snapshot, error) {
	bufV := strconv.FormatInt(version, 10)
	state, err := d.adapter.Load([]byte(bufV))
	if err != nil {
		return nil, err
	}

	out := &tmtypes.Snapshot{}
	err = proto.Unmarshal(state, out)
	if err != nil {
		return nil, err
	}

	return out, nil
}

func (d *Database) Close() error {
	return d.adapter.Close()
}

func (d *Database) Clear() error {
	return d.adapter.Clear()
}

func NewDatabase(adapter Adapter) *Database {
	return &Database{
		adapter: adapter,
	}
}
