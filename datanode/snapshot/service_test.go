package snapshot_test

import (
	"testing"

	"code.vegaprotocol.io/vega/datanode/snapshot"
	"github.com/stretchr/testify/assert"
)

func TestInfoFromFilename(t *testing.T) {
	info, err := snapshot.InfoFromFileName("testnet-fde111-25000.datanode-snapshot")
	assert.NoError(t, err)
	assert.Equal(t, int64(25000), info.Height)
	assert.Equal(t, "testnet-fde111", info.ChainId)

	_, err = snapshot.InfoFromFileName("testnet-fde111-25000.datanode2-snapshot")
	assert.Error(t, err)
}
