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
	require.True(t, rp.Add("k1"))
	require.False(t, rp.Add("k1"))
}

func testOnDuplicatedKeyOnTheDifferentblock(t *testing.T) {
	rp := abci.NewReplayProtector(2)
	rp.SetHeight(0)
	require.True(t, rp.Add("k1"))

	rp.SetHeight(1)
	require.False(t, rp.Add("k1"))
}

func testCacheEviction(t *testing.T) {
	rp := abci.NewReplayProtector(2)
	rp.SetHeight(0)
	require.True(t, rp.Add("k1"))

	rp.SetHeight(1)
	require.False(t, rp.Add("k1"))

	rp.SetHeight(2)
	require.True(t, rp.Add("k1"))
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
func benchmarkReplayProtector(b *testing.B, size int) {
	b.Helper()
	rp := newPopulatedRP(size, b.N)
	for i := 0; i < b.N; i++ {
		rp.Has("xxx")
	}
}

func BenchmarkReplayProtectorLookup10(b *testing.B)   { benchmarkReplayProtector(b, 10) }
func BenchmarkReplayProtectorLookup50(b *testing.B)   { benchmarkReplayProtector(b, 50) }
func BenchmarkReplayProtectorLookup100(b *testing.B)  { benchmarkReplayProtector(b, 100) }
func BenchmarkReplayProtectorLookup500(b *testing.B)  { benchmarkReplayProtector(b, 500) }
func BenchmarkReplayProtectorLookup1000(b *testing.B) { benchmarkReplayProtector(b, 1000) }
