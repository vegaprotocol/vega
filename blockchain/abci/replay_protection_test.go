package abci_test

import (
	"fmt"
	"testing"

	"code.vegaprotocol.io/vega/blockchain/abci"
	"github.com/stretchr/testify/require"
)

func TestReplayProtector(t *testing.T) {
	t.Run("duplicated key on the same block", testOnDuplicatedKeyOnTheSameblock)
	t.Run("duplicated key on the different block", testOnDuplicatedKeyOnTheDifferentblock)
	t.Run("cache eviction", testCacheEviction)
}

func testOnDuplicatedKeyOnTheSameblock(t *testing.T) {
	rp := abci.NewReplayProtector(1)
	require.NoError(t, rp.Add("k1"))
	require.Error(t, rp.Add("k1"))
}

func testOnDuplicatedKeyOnTheDifferentblock(t *testing.T) {
	rp := abci.NewReplayProtector(2)
	rp.SetHeight(0)
	require.NoError(t, rp.Add("k1"))

	rp.SetHeight(1)
	require.Error(t, rp.Add("k1"))
}

func testCacheEviction(t *testing.T) {
	rp := abci.NewReplayProtector(2)
	rp.SetHeight(0)
	require.NoError(t, rp.Add("k1"))

	rp.SetHeight(1)
	require.Error(t, rp.Add("k1"))

	rp.SetHeight(2)
	require.NoError(t, rp.Add("k1"))
}

// newPopulatedRP will create a ReplayProtector with `nBlocks`
// block capacity and `nKeys` per block.
func newPopulatedRP(nBlocks, nKeys int) *abci.ReplayProtector {
	rp := abci.NewReplayProtector(uint(nBlocks))
	for i := 0; i < nBlocks; i++ {
		rp.SetHeight(uint64(i))

		for j := 0; j < nKeys; j++ {
			key := fmt.Sprintf("key-%d-%d", i, j)
			rp.Add(key)
		}
	}
	return rp
}
func benchmarkReplayProtector(size int, b *testing.B) {
	rp := newPopulatedRP(size, b.N)
	for i := 0; i < b.N; i++ {
		rp.Has("xxx")
	}
}

func BenchmarkReplayProtectorLookup10(b *testing.B)   { benchmarkReplayProtector(10, b) }
func BenchmarkReplayProtectorLookup50(b *testing.B)   { benchmarkReplayProtector(50, b) }
func BenchmarkReplayProtectorLookup100(b *testing.B)  { benchmarkReplayProtector(100, b) }
func BenchmarkReplayProtectorLookup500(b *testing.B)  { benchmarkReplayProtector(500, b) }
func BenchmarkReplayProtectorLookup1000(b *testing.B) { benchmarkReplayProtector(1000, b) }
