package tree

import (
	cometbftdb "github.com/cometbft/cometbft-db"
	tmtypes "github.com/tendermint/tendermint/abci/types"
)

type MetadataDatabase interface {
	Save(int64, *tmtypes.Snapshot) error
	Load(int64) (*tmtypes.Snapshot, error)
	Close() error
	Clear() error
	IsEmpty() bool
	FindVersionByBlockHeight(uint64) (int64, error)
	Delete(int64) error
	DeleteRange(fromVersion, toVersion int64) error
}

type SnapshotsDatabase interface {
	cometbftdb.DB
	Clear() error
}
