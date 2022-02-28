package abci

import (
	"context"
	"errors"

	"code.vegaprotocol.io/vega/types"
)

var (
	ErrTxAlreadyInCache   = errors.New("reply protection: tx already in the cache")
	ErrTxStaled           = errors.New("reply protection: stale")
	ErrTxReferFutureBlock = errors.New("reply protection: tx refer future block")
)

// ReplayProtector implement a block distance and ring buffer cache
// based replay protection.
type ReplayProtector struct {
	height      uint64
	txs         []map[string]struct{}
	rss         *replaySnapshotState
	forwardTol  uint
	backwardTol uint
}

// NewReplayProtector returns a new ReplayProtector instance given a tolerance.
func NewReplayProtector(tolerance uint) *ReplayProtector {
	rp := &ReplayProtector{
		txs: make([]map[string]struct{}, 2*tolerance),
		rss: &replaySnapshotState{
			changed:    true,
			hash:       []byte{},
			serialised: []byte{},
		},
		// in the future there can be separate tolerances for back and future
		backwardTol: tolerance,
		forwardTol:  tolerance,
	}

	for i := range rp.txs {
		rp.txs[i] = map[string]struct{}{}
	}

	return rp
}

// GetReplacement if we've already replace it with a real one, replacing again gets itself.
func (r *ReplayProtector) GetReplacement() *ReplayProtector {
	return r
}

// SetHeight tells the ReplayProtector to clear the oldest cached Tx in the internal ring buffer.
func (rp *ReplayProtector) SetHeight(h uint64) {
	rp.height = h

	if l := uint64(len(rp.txs)); h >= l {
		rp.txs[h%l] = map[string]struct{}{}
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

	// If the tx is on a future block, we accept if it's not further than the len of the ring buffer
	// For posterity, the reason we care about future block for replay protection is to prevent someone from submitting a transaction
	// with far enough block that once it gets out of the ring buffer it become replayable.
	// For example, suppose the ring buffer size is 100 (meaning we keep transactions from the last 100 blocks), if someone signs a transaction
	// with block height 100000000000, if it doesn't get rejected, it will be added in deliverTx to the ring bugger to the index 100000000000%100.
	// Then within 100 blocks it can be replayed.
	// To avoid that we keep a ring buffer with `len(backTol + forwardTol)`` and allow transactions that are less than forwardTol in the future and
	// no less than backwardTol in the past.
	// so suppose a tx comes in when current block height is 200 and forward tol is 150 and backward tol is 150. We will accept a transaction
	// if its block height is between 51 and 349. Let say we accepted a transaction with block height = current block height + forward tol - 1,
	// the transaction will stay in the block for ring size (= `backwardTol + forwardTol`) blocks, meaning when it is evicted it is at least `backwardTol`
	//`blocks behind meaning it's guaranteed to get rejected if someone tries to replay it.
	if tx.BlockHeight() >= rp.height+uint64(rp.forwardTol) {
		return ErrTxReferFutureBlock
	}

	// Calculate the distance
	if rp.height > tx.BlockHeight() && rp.height >= tx.BlockHeight()+uint64(rp.backwardTol) {
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
	if tx.BlockHeight() >= rp.height+uint64(rp.forwardTol) {
		return ErrTxReferFutureBlock
	}

	// Calculate the distance
	if rp.height > tx.BlockHeight() && rp.height >= tx.BlockHeight()+uint64(rp.backwardTol) {
		return ErrTxStaled
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
		// if len(pl.Blocks) is zero, we should still assume a full-blown replay protector is required
		// the snapshot engine shouldn't store nil-state/nil-hashes and as such there will be no LoadState
		// call when the Noop protector was used as state provider
		rp.replacement = NewReplayProtector(uint(len(pl.Blocks))) // this tolerance may or may not be sufficient
		err = rp.replacement.restoreReplayState(ctx, pl.Blocks)
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
