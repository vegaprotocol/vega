package pow_test

import (
	"testing"

	vgcrypto "code.vegaprotocol.io/vega/libs/crypto"
	nodetypes "code.vegaprotocol.io/vega/wallet/api/node/types"
	"code.vegaprotocol.io/vega/wallet/api/pow"
	"github.com/stretchr/testify/require"
)

func TestProofOfWorkGeneration(t *testing.T) {
	t.Run("Basic generation", testBasicGeneration)
	t.Run("Too many transactions without increasing difficulty", testTooManyTransactionsWithoutIncreasingDifficulty)
	t.Run("Too many transactions with increasing difficulty", testTooManyTransactionsWithIncreasingDifficulty)
	t.Run("Number of allowed past blocks changes", testNumberOfPastBlocksChanges)
	t.Run("Different chains", testDifferentChains)
}

func testBasicGeneration(t *testing.T) {
	p := pow.NewProofOfWork()

	pubkey := vgcrypto.RandomHash()
	res, err := p.Generate(pubkey, defaultLastBlock(t))
	require.NoError(t, err)
	require.NotEmpty(t, res.Tid)
}

func testTooManyTransactionsWithoutIncreasingDifficulty(t *testing.T) {
	p := pow.NewProofOfWork()

	pubkey1 := vgcrypto.RandomHash()
	pubkey2 := vgcrypto.RandomHash()

	lbh := defaultLastBlock(t)
	lbh.ProofOfWorkIncreasingDifficulty = false
	lbh.ProofOfWorkTxPerBlock = 5

	// when pubkey1 submits too many txn per block
	for i := 0; i < 5; i++ {
		_, err := p.Generate(pubkey1, lbh)
		require.NoError(t, err)
	}

	// then pubkey1 is blocked and pubkey2 isn't
	_, err := p.Generate(pubkey1, lbh)
	require.ErrorIs(t, err, pow.ErrTransactionsPerBlockLimitReached)

	_, err = p.Generate(pubkey2, lbh)
	require.NoError(t, err)

	// when we start a new block and our counters reset
	lbh.BlockHeight = 101

	// then pubkey1 can generate pow again
	_, err = p.Generate(pubkey1, lbh)
	require.NoError(t, err)
}

func testTooManyTransactionsWithIncreasingDifficulty(t *testing.T) {
	p := pow.NewProofOfWork()

	pubkey1 := vgcrypto.RandomHash()

	lbh := defaultLastBlock(t)
	lbh.ProofOfWorkIncreasingDifficulty = true
	lbh.ProofOfWorkTxPerBlock = 1

	// when pubkey1 submits 5 transactions with difficulty 1
	for i := 0; i < 5; i++ {
		_, err := p.Generate(pubkey1, lbh)
		require.NoError(t, err)
	}

	// then the next 10 should increase in difficulty by one
	for i := 2; i < 12; i++ {
		r, err := p.Generate(pubkey1, lbh)
		require.NoError(t, err)

		ok, d := vgcrypto.Verify(lbh.BlockHash, r.Tid, r.Nonce, lbh.ProofOfWorkHashFunction, 1)
		require.True(t, ok)
		require.GreaterOrEqual(t, uint(d), uint(i))
	}
}

func testNumberOfPastBlocksChanges(t *testing.T) {
	p := pow.NewProofOfWork()

	pubkey := vgcrypto.RandomHash()
	lbh := defaultLastBlock(t)

	// start with a buffer size of 5
	lbh.ProofOfWorkIncreasingDifficulty = false
	lbh.ProofOfWorkPastBlocks = 5
	lbh.ProofOfWorkTxPerBlock = 2

	// send stuff in
	for i := uint64(0); i < 10; i++ {
		lbh.BlockHeight = i + 1
		_, err := p.Generate(pubkey, lbh)
		require.NoError(t, err)
	}

	// reach limit on block 10
	_, err := p.Generate(pubkey, lbh)
	require.NoError(t, err)
	_, err = p.Generate(pubkey, lbh)
	require.ErrorIs(t, err, pow.ErrTransactionsPerBlockLimitReached)

	// check we're still blocked after resize
	lbh.ProofOfWorkPastBlocks = 2
	_, err = p.Generate(pubkey, lbh)
	require.ErrorIs(t, err, pow.ErrTransactionsPerBlockLimitReached)

	// now ask to create a pow for a block that is now too historic
	lbh.BlockHeight = 8
	_, err = p.Generate(pubkey, lbh)
	require.ErrorIs(t, err, pow.ErrBlockHeightTooHistoric)

	// now increase and it should be fine
	lbh.ProofOfWorkPastBlocks = 10
	_, err = p.Generate(pubkey, lbh)
	require.NoError(t, err)

	// check we are still blocked on the higher block
	lbh.BlockHeight = 10
	_, err = p.Generate(pubkey, lbh)
	require.ErrorIs(t, err, pow.ErrTransactionsPerBlockLimitReached)
}

func testDifferentChains(t *testing.T) {
	p := pow.NewProofOfWork()

	pubkey1 := vgcrypto.RandomHash()

	lbh := defaultLastBlock(t)
	lbh.ProofOfWorkIncreasingDifficulty = false
	lbh.ProofOfWorkTxPerBlock = 5

	// when pubkey1 submits too many txn per block
	for i := 0; i < 5; i++ {
		_, err := p.Generate(pubkey1, lbh)
		require.NoError(t, err)
	}

	// then pubkey1 is blocked and pubkey2 isn't
	_, err := p.Generate(pubkey1, lbh)
	require.ErrorIs(t, err, pow.ErrTransactionsPerBlockLimitReached)

	lbh.ChainID = "different-chain"
	// then pubkey1 is not blocked on a different chain
	_, err = p.Generate(pubkey1, lbh)
	require.NoError(t, err)
}

func defaultLastBlock(t *testing.T) *nodetypes.LastBlock {
	t.Helper()
	return &nodetypes.LastBlock{
		BlockHeight:                     10,
		BlockHash:                       vgcrypto.RandomHash(),
		ChainID:                         "default-chain-id",
		ProofOfWorkDifficulty:           1,
		ProofOfWorkPastBlocks:           100,
		ProofOfWorkTxPerBlock:           100,
		ProofOfWorkIncreasingDifficulty: true,
		ProofOfWorkHashFunction:         vgcrypto.Sha3,
	}
}
