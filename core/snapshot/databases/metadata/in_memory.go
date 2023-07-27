package metadata

import (
	"sort"

	tmtypes "github.com/tendermint/tendermint/abci/types"
)

type InMemoryDatabase struct {
	store map[int64]*tmtypes.Snapshot
}

func (d *InMemoryDatabase) Save(version int64, state *tmtypes.Snapshot) error {
	d.store[version] = state
	return nil
}

func (d *InMemoryDatabase) Load(version int64) (*tmtypes.Snapshot, error) {
	s, ok := d.store[version]
	if !ok {
		return nil, noMetadataForSnapshotVersion(version)
	}
	return s, nil
}

func (d *InMemoryDatabase) Close() error {
	return nil
}

func (d *InMemoryDatabase) Clear() error {
	d.store = map[int64]*tmtypes.Snapshot{}
	return nil
}

func (d *InMemoryDatabase) IsEmpty() bool {
	return len(d.store) == 0
}

func (d *InMemoryDatabase) FindVersionByBlockHeight(blockHeight uint64) (int64, error) {
	for version, snapshot := range d.store {
		if snapshot.Height == blockHeight {
			return version, nil
		}
	}
	return -1, nil
}

func (d *InMemoryDatabase) Delete(version int64) error {
	delete(d.store, version)
	return nil
}

func (d *InMemoryDatabase) DeleteRange(fromVersion, toVersion int64) error {
	versions := make([]int, 0, len(d.store))
	for version := range d.store {
		versions = append(versions, int(version))
	}
	sort.Ints(versions)

	for _, version := range versions {
		if version >= int(fromVersion) && version < int(toVersion) {
			delete(d.store, int64(version))
		}
	}

	return nil
}

func NewInMemoryDatabase() *InMemoryDatabase {
	return &InMemoryDatabase{
		store: map[int64]*tmtypes.Snapshot{},
	}
}
