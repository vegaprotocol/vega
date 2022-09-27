package snapshot

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHistoryDirName(t *testing.T) {
	history := NewHistorySnapshot("testnet-fde111", 2001, 3000)
	assert.Equal(t, "testnet-fde111-2001-3000-historysnapshot", history.UncompressedDataDir())
}

func TestGetHistorySnapshots(t *testing.T) {
	snapshotsDir := t.TempDir()

	os.Create(filepath.Join(snapshotsDir, "testnet-fde111-0-1000-historysnapshot.tar.gz"))
	os.Create(filepath.Join(snapshotsDir, "testnet-fde111-1001-2000-historysnapshot.tar.gz"))
	os.Create(filepath.Join(snapshotsDir, "testnet-fde111-3001-4000-historysnapshot.tar.gz"))
	os.Create(filepath.Join(snapshotsDir, "testnet-fde111-4001-5000-historysnapshot.tar.gz"))
	os.Create(filepath.Join(snapshotsDir, "testnet-fde111-5001-6000-historysnapshot.tar.gz"))
	os.Create(filepath.Join(snapshotsDir, "testnet-fde111-6001-7000-historysnapshot.tar.gz"))
	os.Create(filepath.Join(snapshotsDir, "testnet-fde111-8000.snapshotinprogress"))
	os.Create(filepath.Join(snapshotsDir, "testnet-fde111-7001-8000-historysnapshot.tar.gz"))

	chainID, histSnapshots, err := GetHistorySnapshots(snapshotsDir)
	assert.NoError(t, err)
	assert.Equal(t, "testnet-fde111", chainID)
	assert.Equal(t, 6, len(histSnapshots))
	assert.Equal(t, histSnapshots[0].HeightFrom, int64(0))
	assert.Equal(t, histSnapshots[0].HeightTo, int64(1000))
	assert.Equal(t, histSnapshots[5].HeightFrom, int64(6001))
	assert.Equal(t, histSnapshots[5].HeightTo, int64(7000))
}
