package abci_test

import (
	"bytes"
	"context"
	"fmt"
	"testing"

	snapshot "code.vegaprotocol.io/protos/vega/snapshot/v1"
	"code.vegaprotocol.io/vega/blockchain/abci"
	"code.vegaprotocol.io/vega/txn"
	"code.vegaprotocol.io/vega/types"
	"github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/require"
)

func TestReplayProtector(t *testing.T) {
	t.Run("duplicated key on the same block", testOnDuplicatedKeyOnTheSameblock)
	t.Run("duplicated key on the different block", testOnDuplicatedKeyOnTheDifferentblock)
	t.Run("cache eviction", testCacheEviction)
}

func testOnDuplicatedKeyOnTheSameblock(t *testing.T) {
	rp := abci.NewReplayProtector(1, 0)
	require.True(t, rp.Add("k1"))
	require.False(t, rp.Add("k1"))
}

func testOnDuplicatedKeyOnTheDifferentblock(t *testing.T) {
	rp := abci.NewReplayProtector(2, 0)
	rp.SetHeight(0)
	require.True(t, rp.Add("k1"))

	rp.SetHeight(1)
	require.False(t, rp.Add("k1"))
}

func testCacheEviction(t *testing.T) {
	rp := abci.NewReplayProtector(2, 0)
	rp.SetHeight(0)
	require.True(t, rp.Add("k1"))

	rp.SetHeight(1)
	require.False(t, rp.Add("k1"))

	rp.SetHeight(1000)
	require.True(t, rp.Add("k1"))
}

func TestSnapshot(t *testing.T) {
	rp := abci.NewReplayProtector(2, 0)
	snap1, err := rp.Snapshot()
	require.Nil(t, err)
	require.Equal(t, 1, len(snap1))

	rp.SetHeight(0)
	snap2, err := rp.Snapshot()
	require.Nil(t, err)
	require.Equal(t, 1, len(snap2))
	require.True(t, bytes.Equal(snap1["all"], snap2["all"]))

	require.True(t, rp.Add("k1"))
	snap3, err := rp.Snapshot()
	require.Nil(t, err)
	require.False(t, bytes.Equal(snap3["all"], snap2["all"]))

	rp.SetHeight(1)
	require.False(t, rp.Add("k1"))
	snap4, err := rp.Snapshot()
	require.Nil(t, err)
	require.True(t, bytes.Equal(snap4["all"], snap3["all"]))
}

func TestSnapshotRoundTrip(t *testing.T) {
	rp := abci.NewReplayProtector(2, 0)
	rp.SetHeight(0)
	require.True(t, rp.Add("k11"))
	require.True(t, rp.Add("k12"))
	require.True(t, rp.Add("k13"))
	require.True(t, rp.Add("k14"))
	rp.SetHeight(1)
	require.True(t, rp.Add("k21"))
	require.True(t, rp.Add("k22"))
	require.True(t, rp.Add("k23"))
	require.True(t, rp.Add("k24"))
	state1, _, err := rp.GetState("all")
	require.Nil(t, err)
	hash1, err := rp.GetHash("all")
	require.Nil(t, err)
	var pl snapshot.Payload
	proto.Unmarshal(state1, &pl)
	payload := types.PayloadFromProto(&pl)
	_, err = rp.LoadState(context.Background(), payload)
	require.Nil(t, err)
	state2, _, err := rp.GetState("all")
	require.Nil(t, err)
	require.True(t, bytes.Equal(state1, state2))
	hash2, err := rp.GetHash("all")
	require.Nil(t, err)
	require.True(t, bytes.Equal(hash1, hash2))
}

// newPopulatedRP will create a ReplayProtector with `nBlocks`
// block capacity and `nKeys` per block.
func newPopulatedRP(nBlocks, nKeys int) *abci.ReplayProtector {
	rp := abci.NewReplayProtector(uint64(nBlocks), 0)
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

func TestCheckTx(t *testing.T) {
	rp := abci.NewReplayProtector(150, 150)
	rp.SetHeight(100)
	// current block 100, forward tolerance 150, tx block 250 => reject
	require.Error(t, abci.ErrTxReferFutureBlock, rp.CheckTx(FakeTx{block: 250, hash: "shouldberejected"}))
	// current block 100, forward tolerance 150, tx block 249 => accept
	require.NoError(t, rp.CheckTx(FakeTx{block: 249, hash: "shouldbeaccepted"}))

	rp.SetHeight(200)
	// current block 200, backward tolerance 150, tx block 50 => reject
	require.Error(t, abci.ErrTxStaled, rp.CheckTx(FakeTx{block: 50, hash: "shouldberejected"}))
	// current block 200, backward tolerance 150, tx block 51 => accept
	require.NoError(t, rp.CheckTx(FakeTx{block: 51, hash: "shouldbeaccepted"}))
}

func TestDeliverTx(t *testing.T) {
	rp := abci.NewReplayProtector(150, 150)
	rp.SetHeight(100)
	// current block 100, forward tolerance 150, tx block 250 => reject
	require.Error(t, abci.ErrTxReferFutureBlock, rp.DeliverTx(FakeTx{block: 250, hash: "shouldberejected"}))
	// current block 100, forward tolerance 150, tx block 249 => accept
	require.NoError(t, rp.DeliverTx(FakeTx{block: 249, hash: "shouldbeaccepted"}))

	rp.SetHeight(200)
	// current block 200, backward tolerance 150, tx block 50 => reject
	require.Error(t, abci.ErrTxStaled, rp.DeliverTx(FakeTx{block: 50, hash: "shouldberejected2"}))
	// current block 200, backward tolerance 150, tx block 51 => accept
	require.NoError(t, rp.DeliverTx(FakeTx{block: 51, hash: "shouldbeaccepted2"}))
}

func TestReplayAttempts(t *testing.T) {
	rp := abci.NewReplayProtector(150, 150)
	rp.SetHeight(200)

	// add a transaction with height 349
	require.NoError(t, rp.DeliverTx(FakeTx{block: 349, hash: "h1"}))

	// now lets try to replay it at any height until 600
	for i := 201; i < 600; i++ {
		// until block 500 the transaction would still be in the ring buffer.
		// after block 500 it's not there anymore but is also too old
		require.NotNil(t, rp.DeliverTx(FakeTx{block: 250, hash: "h1"}))
	}
}

func BenchmarkReplayProtectorLookup10(b *testing.B)   { benchmarkReplayProtector(b, 10) }
func BenchmarkReplayProtectorLookup50(b *testing.B)   { benchmarkReplayProtector(b, 50) }
func BenchmarkReplayProtectorLookup100(b *testing.B)  { benchmarkReplayProtector(b, 100) }
func BenchmarkReplayProtectorLookup500(b *testing.B)  { benchmarkReplayProtector(b, 500) }
func BenchmarkReplayProtectorLookup1000(b *testing.B) { benchmarkReplayProtector(b, 1000) }

type FakeTx struct {
	block uint64
	hash  string
}

func (fk FakeTx) Command() txn.Command        { return txn.Command(0) }
func (fk FakeTx) Unmarshal(interface{}) error { return nil }
func (fk FakeTx) PubKey() []byte              { return nil }
func (fk FakeTx) PubKeyHex() string           { return "" }
func (fk FakeTx) Party() string               { return "" }
func (fk FakeTx) Hash() []byte                { return []byte(fk.hash) }
func (fk FakeTx) Signature() []byte           { return []byte{} }
func (fk FakeTx) Validate() error             { return nil }
func (fk FakeTx) BlockHeight() uint64         { return fk.block }
func (fk FakeTx) GetCmd() interface{}         { return nil }
