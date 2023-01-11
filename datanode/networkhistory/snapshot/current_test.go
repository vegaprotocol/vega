package snapshot

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSnapshotFromFilename(t *testing.T) {
	snapshot, _ := fromCurrentStateSnapshotFileName("testnet-fde111-25000-currentstatesnapshot.tar.gz")
	assert.NotNil(t, snapshot)
	assert.Equal(t, int64(25000), snapshot.Height)
	assert.Equal(t, "testnet-fde111", snapshot.ChainID)

	snapshot, _ = fromCurrentStateSnapshotFileName("testnet-fde111-25000-datanode2-currentstatesnapshot.tar.gz")
	assert.Nil(t, snapshot)
}

func TestSnapshotExists(t *testing.T) {
	snapshotsDir := t.TempDir()

	os.Create(filepath.Join(snapshotsDir, "testnet-fde111-1000-currentstatesnapshot.tar.gz"))

	snapshot := NewCurrentSnapshot("testnet-fde111", 1000)

	exists, err := snapshotExists(snapshotsDir, snapshot)
	assert.NoError(t, err)
	assert.True(t, exists)

	snapshot = NewCurrentSnapshot("testnet-fde111", 2000)

	exists, err = snapshotExists(snapshotsDir, snapshot)
	assert.NoError(t, err)
	assert.False(t, exists)

	snapshotsDir = t.TempDir()

	os.Create(filepath.Join(snapshotsDir, "testnet-fde111-1000-currentstatesnapshot.tar.gz"))
	os.Create(filepath.Join(snapshotsDir, "testnet-fde111-1000.snapshotinprogress"))
	snapshot = NewCurrentSnapshot("testnet-fde111", 1000)
	assert.NoError(t, err)

	exists, err = snapshotExists(snapshotsDir, snapshot)
	assert.NoError(t, err)
	assert.False(t, exists)
}

func TestGetCurrentStateSnapshots(t *testing.T) {
	snapshotsDir := t.TempDir()

	os.Create(filepath.Join(snapshotsDir, "testnet-fde111-1000-currentstatesnapshot.tar.gz"))
	os.Create(filepath.Join(snapshotsDir, "testnet-fde111-1000.snapshotinprogress"))
	os.Create(filepath.Join(snapshotsDir, "testnet-fde111-2000-currentstatesnapshot.tar.gz"))
	os.Create(filepath.Join(snapshotsDir, "testnet-fde111-3000-currentstatesnapshot.tar.gz"))
	os.Create(filepath.Join(snapshotsDir, "testnet-fde111-4000-currentstatesnapshot.tar.gz"))
	os.Create(filepath.Join(snapshotsDir, "testnet-fde111-5000-currentstatesnapshot.tar.gz"))

	chainID, csSnapshot, err := GetCurrentStateSnapshots(snapshotsDir)
	assert.NoError(t, err)
	assert.Equal(t, "testnet-fde111", chainID)
	assert.Equal(t, 4, len(csSnapshot))
	assert.Equal(t, csSnapshot[2000].Height, int64(2000))
	assert.Equal(t, csSnapshot[3000].Height, int64(3000))
	assert.Equal(t, csSnapshot[4000].Height, int64(4000))
	assert.Equal(t, csSnapshot[5000].Height, int64(5000))
}
