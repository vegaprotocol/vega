package snapshot

import (
	"errors"
	"strconv"

	"code.vegaprotocol.io/vega/libs/proto"

	tmtypes "github.com/tendermint/tendermint/abci/types"
	db "github.com/tendermint/tm-db"
)

var ErrUnknownSnapshotVersion = errors.New("unknown snapshot version")

type MDB interface {
	Save(version []byte, state []byte) error
	Load(version []byte) (state []byte, err error)
	Close() error
}

type MetaDB struct {
	db MDB
}

func NewMetaDB(db MDB) *MetaDB {
	return &MetaDB{
		db: db,
	}
}

func (m *MetaDB) Save(version int64, state *tmtypes.Snapshot) error {
	bufV := strconv.FormatInt(version, 10)
	bufS, err := proto.Marshal(state)
	if err != nil {
		return err
	}

	return m.db.Save([]byte(bufV), bufS)
}

func (m *MetaDB) Load(version int64) (*tmtypes.Snapshot, error) {
	bufV := strconv.FormatInt(version, 10)
	state, err := m.db.Load([]byte(bufV))
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

func (m *MetaDB) Close() error {
	return m.db.Close()
}

type MetaMemDB struct {
	store map[string][]byte
}

func NewMetaMemDB() MDB {
	return &MetaMemDB{
		store: map[string][]byte{},
	}
}

func (m *MetaMemDB) Save(version []byte, state []byte) error {
	m.store[string(version)] = state
	return nil
}

func (m *MetaMemDB) Load(version []byte) (state []byte, err error) {
	s, ok := m.store[string(version)]
	if !ok {
		return nil, ErrUnknownSnapshotVersion
	}
	return s, nil
}

func (m *MetaMemDB) Close() error {
	return nil
}

type MetaGoLevelDB struct {
	store *db.GoLevelDB
}

func NewMetaGoLevelDB(db *db.GoLevelDB) MDB {
	return &MetaGoLevelDB{
		store: db,
	}
}

func (m *MetaGoLevelDB) Save(version []byte, state []byte) error {
	return m.store.Set(version, state)
}

func (m *MetaGoLevelDB) Load(version []byte) (state []byte, err error) {
	return m.store.Get(version)
}

func (m *MetaGoLevelDB) Close() error {
	return m.store.Close()
}
