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

package snapshot_test

import (
	"os"
	"path/filepath"
	"testing"

	"code.vegaprotocol.io/vega/datanode/networkhistory/snapshot"
	"code.vegaprotocol.io/vega/logging"

	"github.com/stretchr/testify/assert"
)

func TestGetHistorySnapshots(t *testing.T) {
	snapshotsDir := t.TempDir()
	service, err := snapshot.NewSnapshotService(logging.NewTestLogger(), snapshot.NewDefaultConfig(), nil, nil, snapshotsDir, nil, nil)
	if err != nil {
		panic(err)
	}

	os.MkdirAll(filepath.Join(snapshotsDir, "testnet-fde111-42-0-1000"), os.ModePerm)
	os.MkdirAll(filepath.Join(snapshotsDir, "testnet-fde111-42-1001-2000"), os.ModePerm)
	os.MkdirAll(filepath.Join(snapshotsDir, "testnet-fde111-42-3001-4000"), os.ModePerm)
	os.MkdirAll(filepath.Join(snapshotsDir, "testnet-fde111-42-4001-5000"), os.ModePerm)
	os.MkdirAll(filepath.Join(snapshotsDir, "testnet-fde111-42-5001-6000"), os.ModePerm)
	os.MkdirAll(filepath.Join(snapshotsDir, "testnet-fde111-42-6001-7000"), os.ModePerm)
	os.Create(filepath.Join(snapshotsDir, "testnet-fde111-8000.snapshotinprogress"))
	os.MkdirAll(filepath.Join(snapshotsDir, "testnet-fde111-42-7001-8000"), os.ModePerm)

	ss, err := service.GetUnpublishedSnapshots()
	assert.NoError(t, err)
	for i := range ss {
		assert.Equal(t, "testnet-fde111", ss[i].ChainID)
	}

	assert.Equal(t, 6, len(ss))
	assert.Equal(t, ss[0].HeightFrom, int64(0))
	assert.Equal(t, ss[0].HeightTo, int64(1000))
	assert.Equal(t, ss[5].HeightFrom, int64(6001))
	assert.Equal(t, ss[5].HeightTo, int64(7000))
}
