package spam

import (
	vgcrypto "code.vegaprotocol.io/vega/libs/crypto"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
	"code.vegaprotocol.io/vega/wallet/api"
	"code.vegaprotocol.io/vega/wallet/api/node/types"
	nodetypes "code.vegaprotocol.io/vega/wallet/api/node/types"
)

type txCounter struct {
	// slice of maps from pubKey->nTxnSent where size is the numbers of past
	// blocks a pow is valid for the transaction count for party p sent in
	// block b is store[b%size][p].
	store []map[string]uint32
	size  int64

	lastBlock int64 // the highest last block we've counted against.
}

// add increments the counter for the number of times the public key has sent in
// a transaction with pow against a particular height.
func (t *txCounter) add(pubKey string, state types.PoWBlockState) (uint32, error) {
	height := int64(state.BlockHeight)
	if height <= t.lastBlock-t.size {
		return 0, api.ErrBlockHeightTooHistoric
	}

	// our new height might be more than 1 bigger than the lastBlock we sent a transaction for,
	// so we need to scrub all those heights in between because we sent 0 transactions in those.
	for i := t.lastBlock + 1; i <= height; i++ {
		t.store[i%t.size] = nil
	}

	i := height % t.size
	if t.store[i] == nil {
		t.store[i] = map[string]uint32{}
	}

	// If our stored height is less than the current block state either we've
	// restarted the wallet, and we can now pick up the current amount, or some
	// external transaction were sent outside our view, so we take the biggest
	// value
	if t.store[i][pubKey] < uint32(state.TransactionsSeen) {
		t.store[i][pubKey] = uint32(state.TransactionsSeen)
	}
	t.store[i][pubKey]++

	if height > t.lastBlock {
		t.lastBlock = height
	}
	return t.store[i][pubKey], nil
}

func (t *txCounter) resize(n int64) {
	if n == t.size {
		return
	}

	if t.size == 0 {
		t.store = make([]map[string]uint32, n)
		t.size = n
		return
	}

	// make a new slice
	newStore := make([]map[string]uint32, n)

	// transfer maps from old slice to new
	nTransfer := n
	if t.size < nTransfer || t.lastBlock < nTransfer {
		nTransfer = t.size
	}

	for i := int64(0); i < nTransfer; i++ {
		offset := t.lastBlock - i
		newStore[offset%n] = t.store[offset%t.size]
	}
	t.size = n
	t.store = newStore
}

func (s *Handler) getCounterForChain(chainID string) *txCounter {
	if _, ok := s.counters[chainID]; !ok {
		s.counters[chainID] = &txCounter{}
	}
	return s.counters[chainID]
}

// GenerateProofOfWork Generate returns a proof-of-work with difficult that
// respects the history of transactions sent in against a particular block.
func (s *Handler) GenerateProofOfWork(pubKey string, st *nodetypes.SpamStatistics) (*commandspb.ProofOfWork, error) {
	s.mu.Lock()
	counter := s.getCounterForChain(st.ChainID)
	blockState := st.PoW.PowBlockStates[0]

	// If the network parameter for past blocks has changed we need to tell the
	// counter, so it can tell us if we're using a now historic block.
	counter.resize(int64(st.PoW.PastBlocks))
	nSent, err := counter.add(pubKey, blockState)
	s.mu.Unlock()
	if err != nil {
		return nil, err
	}

	nPerBlock := blockState.TxPerBlock

	// now work out the pow difficulty
	difficulty := blockState.Difficulty
	if uint64(nSent) > nPerBlock {
		if !blockState.IncreasingDifficulty {
			return nil, api.ErrTransactionsPerBlockLimitReached
		}
		// how many times have we hit the limit
		difficulty += uint64(nSent) / nPerBlock
	}

	tid := vgcrypto.RandomHash()
	powNonce, _, err := vgcrypto.PoW(blockState.BlockHash, tid, uint(difficulty), vgcrypto.Sha3)
	if err != nil {
		return nil, err
	}

	return &commandspb.ProofOfWork{
		Tid:   tid,
		Nonce: powNonce,
	}, nil
}
