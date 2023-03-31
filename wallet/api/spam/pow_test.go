package spam_test

import (
	"testing"

	vgcrypto "code.vegaprotocol.io/vega/libs/crypto"
	walletapi "code.vegaprotocol.io/vega/wallet/api"
	nodetypes "code.vegaprotocol.io/vega/wallet/api/node/types"
	"code.vegaprotocol.io/vega/wallet/api/spam"
	"github.com/stretchr/testify/require"
)

func TestProofOfWorkGeneration(t *testing.T) {
	t.Run("Basic generation", testBasicGeneration)
	t.Run("Too many transactions without increasing difficulty", testTooManyTransactionsWithoutIncreasingDifficulty)
	t.Run("Too many transactions with increasing difficulty", testTooManyTransactionsWithIncreasingDifficulty)
	t.Run("Number of allowed past blocks changes", testNumberOfPastBlocksChanges)
	t.Run("Different chains", testDifferentChains)
	t.Run("Statistics vs own count", testStatsVsOwnCount)
}

func testBasicGeneration(t *testing.T) {
	p := spam.NewHandler()

	pubkey := vgcrypto.RandomHash()
	res, err := p.GenerateProofOfWork(pubkey, defaultSpamStats(t))
	require.NoError(t, err)
	require.NotEmpty(t, res.Tid)
}

func testTooManyTransactionsWithoutIncreasingDifficulty(t *testing.T) {
	p := spam.NewHandler()

	pubkey1 := vgcrypto.RandomHash()
	pubkey2 := vgcrypto.RandomHash()

	st := defaultSpamStats(t)
	st.PoW.PowBlockStates[0].IncreasingDifficulty = false
	st.PoW.PowBlockStates[0].TxPerBlock = 5

	// when pubkey1 submits too many txn per block
	for i := 0; i < 5; i++ {
		_, err := p.GenerateProofOfWork(pubkey1, st)
		require.NoError(t, err)
	}

	// then pubkey1 is blocked and pubkey2 isn't
	_, err := p.GenerateProofOfWork(pubkey1, st)
	require.ErrorIs(t, err, walletapi.ErrTransactionsPerBlockLimitReached)

	_, err = p.GenerateProofOfWork(pubkey2, st)
	require.NoError(t, err)

	// when we start a new block and our counters reset
	st.PoW.PowBlockStates[0].BlockHeight = 101

	// then pubkey1 can generate pow again
	_, err = p.GenerateProofOfWork(pubkey1, st)
	require.NoError(t, err)
}

func testTooManyTransactionsWithIncreasingDifficulty(t *testing.T) {
	p := spam.NewHandler()

	pubkey1 := vgcrypto.RandomHash()

	st := defaultSpamStats(t)
	st.PoW.PowBlockStates[0].IncreasingDifficulty = true
	st.PoW.PowBlockStates[0].TxPerBlock = 1

	// when pubkey1 submits 5 transactions with difficulty 1
	for i := 0; i < 5; i++ {
		_, err := p.GenerateProofOfWork(pubkey1, st)
		require.NoError(t, err)
	}

	// then the next 10 should increase in difficulty by one
	bs := st.PoW.PowBlockStates[0]
	for i := 2; i < 12; i++ {
		r, err := p.GenerateProofOfWork(pubkey1, st)
		require.NoError(t, err)

		ok, d := vgcrypto.Verify(bs.BlockHash, r.Tid, r.Nonce, bs.HashFunction, 1)
		require.True(t, ok)
		require.GreaterOrEqual(t, uint(d), uint(i))
	}
}

func testNumberOfPastBlocksChanges(t *testing.T) {
	p := spam.NewHandler()

	pubkey := vgcrypto.RandomHash()
	st := defaultSpamStats(t)

	// start with a buffer size of 5
	st.PoW.PowBlockStates[0].IncreasingDifficulty = false
	st.PoW.PastBlocks = 5
	st.PoW.PowBlockStates[0].TxPerBlock = 2

	// send stuff in
	for i := uint64(0); i < 10; i++ {
		st.PoW.PowBlockStates[0].BlockHeight = i + 1
		_, err := p.GenerateProofOfWork(pubkey, st)
		require.NoError(t, err)
	}

	// reach limit on block 10
	_, err := p.GenerateProofOfWork(pubkey, st)
	require.NoError(t, err)
	_, err = p.GenerateProofOfWork(pubkey, st)
	require.ErrorIs(t, err, walletapi.ErrTransactionsPerBlockLimitReached)

	// check we're still blocked after resize
	st.PoW.PastBlocks = 2
	_, err = p.GenerateProofOfWork(pubkey, st)
	require.ErrorIs(t, err, walletapi.ErrTransactionsPerBlockLimitReached)

	// now ask to create a pow for a block that is now too historic
	st.PoW.PowBlockStates[0].BlockHeight = 8
	_, err = p.GenerateProofOfWork(pubkey, st)
	require.ErrorIs(t, err, walletapi.ErrBlockHeightTooHistoric)

	// now increase and it should be fine
	st.PoW.PastBlocks = 10
	_, err = p.GenerateProofOfWork(pubkey, st)
	require.NoError(t, err)

	// check we are still blocked on the higher block
	st.PoW.PowBlockStates[0].BlockHeight = 10
	_, err = p.GenerateProofOfWork(pubkey, st)
	require.ErrorIs(t, err, walletapi.ErrTransactionsPerBlockLimitReached)
}

func testDifferentChains(t *testing.T) {
	p := spam.NewHandler()

	pubkey1 := vgcrypto.RandomHash()

	st := defaultSpamStats(t)
	st.PoW.PowBlockStates[0].IncreasingDifficulty = false
	st.PoW.PowBlockStates[0].TxPerBlock = 5

	// when pubkey1 submits too many txn per block
	for i := 0; i < 5; i++ {
		_, err := p.GenerateProofOfWork(pubkey1, st)
		require.NoError(t, err)
	}

	// then pubkey1 is blocked and pubkey2 isn't
	_, err := p.GenerateProofOfWork(pubkey1, st)
	require.ErrorIs(t, err, walletapi.ErrTransactionsPerBlockLimitReached)

	st.ChainID = "different-chain"
	// then pubkey1 is not blocked on a different chain
	_, err = p.GenerateProofOfWork(pubkey1, st)
	require.NoError(t, err)
}

func testStatsVsOwnCount(t *testing.T) {
	p := spam.NewHandler()

	pubkey1 := vgcrypto.RandomHash()

	st := defaultSpamStats(t)
	st.PoW.PowBlockStates[0].IncreasingDifficulty = false
	st.PoW.PowBlockStates[0].TxPerBlock = 2
	st.PoW.PowBlockStates[0].TransactionsSeen = 2

	// we have a fresh state and stats tells us we're already at the limit
	_, err := p.GenerateProofOfWork(pubkey1, st)
	require.ErrorIs(t, err, walletapi.ErrTransactionsPerBlockLimitReached)

	// we now move onto the next block and generate some proof-of-work
	st.PoW.PowBlockStates[0].BlockHeight = 150
	st.PoW.PowBlockStates[0].TransactionsSeen = 0
	_, err = p.GenerateProofOfWork(pubkey1, st)
	require.NoError(t, err)
	_, err = p.GenerateProofOfWork(pubkey1, st)
	require.NoError(t, err)

	// next generation should fail because we've reached the limit, but the stats
	// haven't caught up and will tell us there is space, we ignore it
	st.PoW.PowBlockStates[0].TransactionsSeen = 0
	_, err = p.GenerateProofOfWork(pubkey1, st)
	require.ErrorIs(t, err, walletapi.ErrTransactionsPerBlockLimitReached)
}

func defaultSpamStats(t *testing.T) *nodetypes.SpamStatistics {
	t.Helper()
	return &nodetypes.SpamStatistics{
		ChainID:           "default-chain-id",
		LastBlockHeight:   10,
		Delegations:       &nodetypes.SpamStatistic{},
		Proposals:         &nodetypes.SpamStatistic{},
		Transfers:         &nodetypes.SpamStatistic{},
		NodeAnnouncements: &nodetypes.SpamStatistic{},
		IssuesSignatures:  &nodetypes.SpamStatistic{},
		Votes: &nodetypes.VoteSpamStatistics{
			Proposals: map[string]uint64{},
		},
		PoW: &nodetypes.PoWStatistics{
			PowBlockStates: []nodetypes.PoWBlockState{
				{
					BlockHash:            vgcrypto.RandomHash(),
					BlockHeight:          10,
					TxPerBlock:           100,
					IncreasingDifficulty: true,
					HashFunction:         vgcrypto.Sha3,
				},
			},
			PastBlocks: 100,
		},
	}
}
