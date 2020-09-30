package abci

import "errors"

type ReplayProtector struct {
	height uint64
	txs    []map[string]struct{}
}

func NewReplayProtector(blocks uint) *ReplayProtector {
	rp := &ReplayProtector{
		txs: make([]map[string]struct{}, blocks),
	}

	for i := range rp.txs {
		rp.txs[i] = make(map[string]struct{})
	}

	return rp
}

// SetHeight tells the ReplayProtector to clear the oldest cached Tx in the internal ring buffer.
func (rp *ReplayProtector) SetHeight(h uint64) {
	rp.height = h

	l := uint64(len(rp.txs))
	if h < l {
		return
	}

	rp.txs[h%l] = make(map[string]struct{})
}

func (rp *ReplayProtector) Has(key string) bool {
	for i := range rp.txs {
		if _, ok := rp.txs[i][key]; ok {
			return true
		}
	}
	return false
}

// Add tries to add a key into the cache and returns an error if the given key already exists.
func (rp *ReplayProtector) Add(key string) error {
	if rp.Has(key) {
		return errors.New("key already in the cache")
	}

	target := rp.height % uint64(len(rp.txs))
	rp.txs[target][key] = struct{}{}
	return nil
}
