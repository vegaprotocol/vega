package store_test

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"code.vegaprotocol.io/vega/datanode/networkhistory/snapshot"
	"code.vegaprotocol.io/vega/datanode/networkhistory/store"
	"code.vegaprotocol.io/vega/logging"

	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const expectedSizeOnDiskWithNoGc = 56000

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

	assertRoughlyEqual(t, 41000, dirSize)

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

	assertRoughlyEqual(t, 22000, dirSize)

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
		css := snapshot.NewCurrentSnapshot(chainID, to)
		hss := snapshot.NewHistorySnapshot(chainID, from, to)

		err := os.WriteFile(filepath.Join(snapshotsDir, css.CompressedFileName()), []byte(fmt.Sprintf("%d", to)), fs.ModePerm)
		require.NoError(t, err)
		err = os.WriteFile(filepath.Join(snapshotsDir, hss.CompressedFileName()), []byte(fmt.Sprintf("%d to %d", from, to)), fs.ModePerm)
		require.NoError(t, err)
		err = s.AddSnapshotData(context.Background(), hss, css, snapshotsDir)
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
	snapshotsDir := t.TempDir()

	s, err := store.New(context.Background(), log, chainID, cfg, networkhistoryHome, false, 33)
	require.NoError(t, err)
	return s, snapshotsDir
}

func dirSize(path string) (int64, error) {
	var size int64
	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	return size, err
}
