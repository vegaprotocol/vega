package abci

import (
	"errors"
)

var (
	ErrTxAlreadyInCache   = errors.New("reply protection: tx already in the cache")
	ErrTxStaled           = errors.New("reply protection: staled")
	ErrTxReferFutureBlock = errors.New("reply protection: tx refer future block")
)

// ReplayProtector implement a block distance and ring buffer cache
// based replay protection.
type ReplayProtector struct {
	height uint64
	txs    []map[string]struct{}
	rss    *replaySnapshotState
}

// NewReplayProtector returns a new ReplayProtector instance given a tolerance.
func NewReplayProtector(tolerance uint) *ReplayProtector {
	rp := &ReplayProtector{
		txs: make([]map[string]struct{}, tolerance),
		rss: &replaySnapshotState{
			changed:    true,
			hash:       []byte{},
			serialised: []byte{},
		},
	}

	for i := range rp.txs {
		rp.txs[i] = make(map[string]struct{})
	}

	return rp
}

// SetHeight tells the ReplayProtector to clear the oldest cached Tx in the internal ring buffer.
func (rp *ReplayProtector) SetHeight(h uint64) {
	rp.height = h

	if l := uint64(len(rp.txs)); h >= l {
		rp.txs[h%l] = make(map[string]struct{})
		rp.rss.changed = true
	}
}

// Has checks if a given key is present in the cache.
func (rp *ReplayProtector) Has(key string) bool {
	for i := range rp.txs {
		if _, ok := rp.txs[i][key]; ok {
			return true
		}
	}
	return false
}

// Add tries to add a key into the cache, it returns false if the given key already exists.
func (rp *ReplayProtector) Add(key string) bool {
	if rp.Has(key) {
		return false
	}

	target := rp.height % uint64(len(rp.txs))
	rp.txs[target][key] = struct{}{}
	rp.rss.changed = true
	return true
}

// DeliverTx excercises both strategies (cache and tolerance) to determine if a Tx should be allowed or not.
func (rp *ReplayProtector) DeliverTx(tx Tx) error {
	// We perform 2 verifications:
	// First we make sure that the Tx is not on the ring buffer.
	key := string(tx.Hash())
	if !rp.Add(key) {
		return ErrTxAlreadyInCache
	}

	// Then we verify the block distance:

	// If the tx is on a future block, we accept.
	if tx.BlockHeight() > rp.height {
		return ErrTxReferFutureBlock
	}

	// Calculate the distance
	tolerance := len(rp.txs)
	if rp.height-tx.BlockHeight() >= uint64(tolerance) {
		return ErrTxStaled
	}

	return nil
}

// CheckTx excercises the strategies  tolerance to determine if a Tx should be allowed or not.
func (rp *ReplayProtector) CheckTx(tx Tx) error {
	// We perform 2 verifications:
	// First we make sure that the Tx is not on the ring buffer.
	if rp.Has(string(tx.Hash())) {
		return ErrTxAlreadyInCache
	}

	// Then we verify the block distance:

	// If the tx is on a future block, we accept.
	if tx.BlockHeight() > rp.height {
		return ErrTxReferFutureBlock
	}

	// Calculate the distance
	tolerance := len(rp.txs)
	if rp.height-tx.BlockHeight() >= uint64(tolerance) {
		return ErrTxStaled
	}

	return nil
}

type replayProtectorNoop struct{}

func (*replayProtectorNoop) SetHeight(uint64)   {}
func (*replayProtectorNoop) DeliverTx(Tx) error { return nil }
func (*replayProtectorNoop) CheckTx(Tx) error   { return nil }
