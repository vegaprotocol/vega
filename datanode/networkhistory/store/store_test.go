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

package store_test

import (
	"archive/zip"
	"bytes"
	"context"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"code.vegaprotocol.io/vega/datanode/config/encoding"
	"code.vegaprotocol.io/vega/datanode/networkhistory/segment"
	"code.vegaprotocol.io/vega/datanode/networkhistory/store"
	"code.vegaprotocol.io/vega/logging"

	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const expectedSizeOnDiskWithNoGc = 20016

func TestRemoveWithNoEligibleSegments(t *testing.T) {
	chainID := uuid.NewV4().String()

	networkhistoryHome := t.TempDir()

	s, snapshotsDir := createStore(t, 20000, chainID, networkhistoryHome)
	addTestData(t, chainID, snapshotsDir, s)

	dirSize, err := dirSize(networkhistoryHome)
	require.NoError(t, err)

	assertRoughlyEqual(t, expectedSizeOnDiskWithNoGc, dirSize)

	postGcSegments, err := s.ListAllIndexEntriesOldestFirst()
	require.NoError(t, err)

	assert.Equal(t, 10, len(postGcSegments))
	assert.Equal(t, int64(1), postGcSegments[0].HeightFrom)
	assert.Equal(t, int64(10000), postGcSegments[9].HeightTo)
}

func TestPartialRemoveOfOldSegments(t *testing.T) {
	chainID := uuid.NewV4().String()

	networkhistoryHome := t.TempDir()

	s, snapshotsDir := createStore(t, 5000, chainID, networkhistoryHome)

	addTestData(t, chainID, snapshotsDir, s)

	dirSize, err := dirSize(networkhistoryHome)
	require.NoError(t, err)

	assertRoughlyEqual(t, 18463, dirSize)

	segments, err := s.ListAllIndexEntriesOldestFirst()
	require.NoError(t, err)

	assert.Equal(t, 6, len(segments))
	assert.Equal(t, int64(4001), segments[0].HeightFrom)
	assert.Equal(t, int64(10000), segments[5].HeightTo)
}

func TestRemoveAllOldSegments(t *testing.T) {
	chainID := uuid.NewV4().String()

	networkhistoryHome := t.TempDir()

	s, snapshotsDir := createStore(t, 0, chainID, networkhistoryHome)

	addTestData(t, chainID, snapshotsDir, s)

	dirSize, err := dirSize(networkhistoryHome)
	require.NoError(t, err)

	assertRoughlyEqual(t, 17504, dirSize)

	segments, err := s.ListAllIndexEntriesOldestFirst()
	require.NoError(t, err)

	assert.Equal(t, 1, len(segments))
	assert.Equal(t, int64(9001), segments[0].HeightFrom)
	assert.Equal(t, int64(10000), segments[0].HeightTo)
}

func addTestData(t *testing.T, chainID string, snapshotsDir string, s *store.Store) {
	t.Helper()
	for i := int64(0); i < 10; i++ {
		from := (i * 1000) + 1
		to := (i + 1) * 1000
		segment := segment.Unpublished{
			Base: segment.Base{
				HeightFrom:      from,
				HeightTo:        to,
				ChainID:         chainID,
				DatabaseVersion: 1,
			},
			Directory: snapshotsDir,
		}

		buf := new(bytes.Buffer)
		zipWriter := zip.NewWriter(buf)

		f, err := zipWriter.Create("dummy.txt")
		require.NoError(t, err)

		_, err = io.WriteString(f, "This is some dummy data.")
		require.NoError(t, err)

		err = zipWriter.Close()
		require.NoError(t, err)

		err = os.WriteFile(segment.UnpublishedSnapshotDataDirectory(), buf.Bytes(), fs.ModePerm)
		require.NoError(t, err)

		err = s.AddSnapshotData(context.Background(), segment)
		require.NoError(t, err)
	}
}

func assertRoughlyEqual(t *testing.T, expected, actual int64) {
	t.Helper()
	permissablePercentDiff := int64(5)
	lowerBound := expected - ((expected * permissablePercentDiff) / 100)
	upperBound := expected + ((expected * permissablePercentDiff) / 100)

	assert.Less(t, lowerBound, actual)
	assert.Greater(t, upperBound, actual)
}

func createStore(t *testing.T, historyRetentionBlockSpan int64, chainID string, networkhistoryHome string) (*store.Store, string) {
	t.Helper()
	log := logging.NewTestLogger()
	cfg := store.NewDefaultConfig()
	cfg.HistoryRetentionBlockSpan = historyRetentionBlockSpan
	cfg.GarbageCollectionInterval = encoding.Duration{Duration: 0}
	snapshotsDir := t.TempDir()

	s, err := store.New(context.Background(), log, chainID, cfg, networkhistoryHome, 33)
	require.NoError(t, err)
	return s, snapshotsDir
}

func dirSize(path string) (int64, error) {
	var size int64
	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil || (info != nil && info.IsDir()) {
			return nil //nolint:nilerr
		}

		size += info.Size()
		return nil
	})
	return size, err
}
