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
	service, err := snapshot.NewSnapshotService(logging.NewTestLogger(), snapshot.NewDefaultConfig(), nil, snapshotsDir, nil, nil)
	if err != nil {
		panic(err)
	}

	os.Create(filepath.Join(snapshotsDir, "testnet-fde111-42-0-1000.zip"))
	os.Create(filepath.Join(snapshotsDir, "testnet-fde111-42-1001-2000.zip"))
	os.Create(filepath.Join(snapshotsDir, "testnet-fde111-42-3001-4000.zip"))
	os.Create(filepath.Join(snapshotsDir, "testnet-fde111-42-4001-5000.zip"))
	os.Create(filepath.Join(snapshotsDir, "testnet-fde111-42-5001-6000.zip"))
	os.Create(filepath.Join(snapshotsDir, "testnet-fde111-42-6001-7000.zip"))
	os.Create(filepath.Join(snapshotsDir, "testnet-fde111-8000.snapshotinprogress"))
	os.Create(filepath.Join(snapshotsDir, "testnet-fde111-42-7001-8000.zip"))

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
