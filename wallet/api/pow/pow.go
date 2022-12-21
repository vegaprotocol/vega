package pow

import (
	"errors"
	"sync"

	vgcrypto "code.vegaprotocol.io/vega/libs/crypto"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
	nodetypes "code.vegaprotocol.io/vega/wallet/api/node/types"
)

var (
	ErrTransactionsPerBlockLimitReached = errors.New("cannot generate proof-of-work - transaction per block limit reached")
	ErrBlockHeightTooHistoric           = errors.New("cannot generate proof-of-work - block data is too historic")
)

type txCounter struct {
	// slice of maps from pubKey->nTxnSent where size is the numbers of past blocks a pow is valid for
	// the transcation count for party p sent in block b is store[b%size][p]
	store []map[string]uint32
	size  int64

	lastBlock int64 // the highest last block we've counted against
}

// add increments the counter for the number of times pubkey has sent in a transaction with pow against a particular height.
func (t *txCounter) add(pubKey string, height int64) (uint32, error) {
	if height <= t.lastBlock-t.size {
		return 0, ErrBlockHeightTooHistoric
	}

	// our new height might be more than 1 bigger than the lastBlock we sent a transaction for
	// so we need to scrub all those heights in between because we sent 0 transactions in those
	for i := t.lastBlock + 1; i <= height; i++ {
		t.store[i%t.size] = nil
	}

	i := height % t.size
	if t.store[i] == nil {
		t.store[i] = map[string]uint32{}
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
	if t.size < nTransfer {
		nTransfer = t.size
	}
	if t.lastBlock < nTransfer {
		nTransfer = t.size
	}

	for i := int64(0); i < nTransfer; i++ {
		offset := t.lastBlock - i
		newStore[offset%n] = t.store[offset%t.size]
	}
	t.size = n
	t.store = newStore
}

type ProofOfWork struct {
	// chainID to the counter for transactions sent.
	counters map[string]*txCounter
	mu       sync.Mutex
}

func NewProofOfWork() *ProofOfWork {
	return &ProofOfWork{
		counters: map[string]*txCounter{},
	}
}

func (p *ProofOfWork) getCounterForChain(chainID string) *txCounter {
	if _, ok := p.counters[chainID]; !ok {
		p.counters[chainID] = &txCounter{}
	}
	return p.counters[chainID]
}

// Generate returns a proof-of-work with difficult that respects the history of transactions sent in against a particular block.
func (p *ProofOfWork) Generate(pubKey string, lastBlock *nodetypes.LastBlock) (*commandspb.ProofOfWork, error) {
	p.mu.Lock()
	counter := p.getCounterForChain(lastBlock.ChainID)

	// if the network parameter for past blocks has changed we need to tell the counter so it
	// can tell us if we're using a now historic block.
	counter.resize(int64(lastBlock.ProofOfWorkPastBlocks))
	nSent, err := counter.add(pubKey, int64(lastBlock.BlockHeight))
	p.mu.Unlock()
	if err != nil {
		return nil, err
	}

	nPerBlock := lastBlock.ProofOfWorkTxPerBlock

	// now work out the pow difficulty
	difficulty := lastBlock.ProofOfWorkDifficulty
	if nSent > nPerBlock {
		if !lastBlock.ProofOfWorkIncreasingDifficulty {
			return nil, ErrTransactionsPerBlockLimitReached
		}
		// how many times have we hit the limit
		difficulty += nSent / nPerBlock
	}

	tid := vgcrypto.RandomHash()
	powNonce, _, err := vgcrypto.PoW(lastBlock.BlockHash, tid, uint(difficulty), vgcrypto.Sha3)
	if err != nil {
		return nil, err
	}

	return &commandspb.ProofOfWork{
		Tid:   tid,
		Nonce: powNonce,
	}, nil
}
