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

	println("replay protection height set to", h)

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

	if tx.BlockHeight() > rp.height {
		return nil
	}

	// Calculate the distance
	tolerance := len(rp.txs)
	if rp.height-tx.BlockHeight() >= uint64(tolerance) {
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
