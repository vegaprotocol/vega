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

package tree

import (
	cometbftdb "github.com/cometbft/cometbft-db"
	tmtypes "github.com/cometbft/cometbft/abci/types"
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
