package abci

import (
	"context"
	"errors"
	"sync"

	"code.vegaprotocol.io/vega/types"
)

var (
	ErrTxAlreadyInCache   = errors.New("reply protection: tx already in the cache")
	ErrTxStaled           = errors.New("reply protection: stale")
	ErrTxReferFutureBlock = errors.New("reply protection: tx refer future block")
)

// ReplayProtector implement a block distance and transactions cache
// based replay protection.
type ReplayProtector struct {
	height      uint64
	txs         map[string]uint64
	rss         *replaySnapshotState
	forwardTol  uint64
	backwardTol uint64
	mu          sync.RWMutex
}

// NewReplayProtector returns a new ReplayProtector instance given a tolerance.
func NewReplayProtector(backTolerance uint64, forwardTolerance uint64) *ReplayProtector {
	rp := &ReplayProtector{
		txs: map[string]uint64{},
		rss: &replaySnapshotState{
			changed:    true,
			hash:       []byte{},
			serialised: []byte{},
		},
		// in the future there can be separate tolerances for back and future
		backwardTol: backTolerance,
		forwardTol:  forwardTolerance,
	}

	return rp
}

// GetReplacement if we've already replace it with a real one, replacing again gets itself.
func (r *ReplayProtector) GetReplacement() *ReplayProtector {
	return r
}

// SetHeight tells the ReplayProtector to clear the old transactions that are no longer in scope.
// As an optimisation this is done only every 1000 blocks.
func (rp *ReplayProtector) SetHeight(h uint64) {
	rp.mu.Lock()
	defer rp.mu.Unlock()
	rp.height = h

	if h%1000 == 0 {
		for k, v := range rp.txs {
			if (h - v) > rp.backwardTol {
				delete(rp.txs, k)
			}
		}
		rp.rss.changed = true
	}
}

// Has checks if a given key is present in the cache.
func (rp *ReplayProtector) Has(key string) bool {
	rp.mu.RLock()
	defer rp.mu.RUnlock()
	_, ok := rp.txs[key]
	return ok
}

// Add tries to add a key into the cache, it returns false if the given key already exists.
func (rp *ReplayProtector) Add(key string) bool {
	rp.mu.Lock()
	defer rp.mu.Unlock()
	if _, ok := rp.txs[key]; ok {
		return false
	}
	rp.txs[key] = rp.height
	rp.rss.changed = true
	return true
}

// DeliverTx excercises both strategies (cache and tolerance) to determine if a Tx should be allowed or not.
func (rp *ReplayProtector) DeliverTx(tx Tx) error {
	// If the tx is on a future block, we accept if it's not further than the forward tolerance.
	// For posterity, the reason we care about future block for replay protection is to prevent someone from submitting a transaction
	// with far enough block that once it gets out of the cache it become replayable.
	// For example, suppose we keep a history of 100 blocks, if someone signs a transaction
	// with block height 100000000000, if it doesn't get rejected, it will be added in deliverTx to the cache and within 100 blocks it can be replayed.
	// To avoid that we keep a a history of (backTol + forwardTol)`` blocks and allow transactions that are less than forwardTol in the future and
	// no less than backwardTol in the past.
	// so suppose a tx comes in when current block height is 200 and forward tol is 150 and backward tol is 150. We will accept a transaction
	// if its block height is between 51 and 349. Let say we accepted a transaction with block height = current block height + forward tol - 1,
	// the transaction will stay in the cache for (= `backwardTol + forwardTol`) blocks, meaning when it is evicted it is at least `backwardTol`
	//`blocks behind meaning it's guaranteed to get rejected if someone tries to replay it.
	if tx.BlockHeight() >= rp.height+rp.forwardTol {
		return ErrTxReferFutureBlock
	}

	// Calculate the distance
	if rp.height > tx.BlockHeight() && rp.height >= tx.BlockHeight()+rp.backwardTol {
		return ErrTxStaled
	}

	// make sure that the Tx is not on the cache
	key := string(tx.Hash())
	if !rp.Add(key) {
		return ErrTxAlreadyInCache
	}

	return nil
}

// CheckTx excercises the strategies  tolerance to determine if a Tx should be allowed or not.
func (rp *ReplayProtector) CheckTx(tx Tx) error {
	// Then we verify the block distance:
	if tx.BlockHeight() >= rp.height+rp.forwardTol {
		return ErrTxReferFutureBlock
	}

	// Calculate the distance
	if rp.height > tx.BlockHeight() && rp.height >= tx.BlockHeight()+rp.backwardTol {
		return ErrTxStaled
	}

	// We perform 2 verifications:
	// First we make sure that the Tx is not on the cache.
	if rp.Has(string(tx.Hash())) {
		return ErrTxAlreadyInCache
	}

	return nil
}

type replayProtectorNoop struct {
	replacement *ReplayProtector
}

func (rp *replayProtectorNoop) Namespace() types.SnapshotNamespace {
	return types.ReplayProtectionSnapshot
}

func (rp *replayProtectorNoop) Keys() []string {
	return hashKeys
}

func (rp *replayProtectorNoop) Stopped() bool {
	return false
}

func (rp *replayProtectorNoop) GetHash(_ string) ([]byte, error) {
	return nil, nil
}

func (rp *replayProtectorNoop) GetState(_ string) ([]byte, []types.StateProvider, error) {
	return nil, nil, nil
}

func (rp *replayProtectorNoop) LoadState(ctx context.Context, p *types.Payload) ([]types.StateProvider, error) {
	if rp.Namespace() != p.Data.Namespace() {
		return nil, types.ErrInvalidSnapshotNamespace
	}
	var err error
	// see what we're reloading
	switch pl := p.Data.(type) {
	case *types.PayloadReplayProtection:
		// create new replay protector that will replace the noop one
		// if pl.BackTol + pl.ForwardTol) is zero, we should still assume a full-blown replay protector is required
		// the snapshot engine shouldn't store nil-state/nil-hashes and as such there will be no LoadState
		// call when the Noop protector was used as state provider
		rp.replacement = NewReplayProtector(pl.BackTol, pl.ForwardTol) // this tolerance may or may not be sufficient
		err = rp.replacement.restoreReplayState(ctx, pl)
	default:
		err = types.ErrUnknownSnapshotType
	}
	if err != nil {
		return nil, err
	}
	return []types.StateProvider{rp.replacement}, err
}

func (rp *replayProtectorNoop) GetReplacement() *ReplayProtector {
	return rp.replacement
}

func (*replayProtectorNoop) SetHeight(uint64)   {}
func (*replayProtectorNoop) DeliverTx(Tx) error { return nil }
func (*replayProtectorNoop) CheckTx(Tx) error   { return nil }
