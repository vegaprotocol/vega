package ratelimit

import (
	"encoding/base64"
)

type Key []byte

func (k Key) String() string {
	return base64.StdEncoding.EncodeToString(k)
}

type Rates struct {
	block      int
	requests   int
	perNBlocks int
	entries    map[string][]int
	whiteList  map[string]struct{}
}

func New(requests, perNBlocks int) *Rates {
	return &Rates{
		block:      0,
		requests:   requests,
		perNBlocks: perNBlocks,
		entries:    map[string][]int{},
		whiteList:  map[string]struct{}{},
	}
}

// Count returns the number of requests recorded for a given key
// It returns -1 if the key has been not recorded or evicted
func (r *Rates) Count(key string) int {
	entry, ok := r.entries[key]
	if !ok {
		return -1
	}

	var count int
	for _, n := range entry {
		count += n
	}
	return count
}

func (r *Rates) NextBlock() {
	// compute the next block index
	r.block = (r.block + 1) % (r.perNBlocks)

	// reset the counters for that particular block index
	for _, c := range r.entries {
		c[r.block] = 0
	}

	// We clean up the entries after finishing the block round
	if r.block != 0 {
		return
	}

	for key := range r.entries {
		if r.Count(key) == 0 {
			delete(r.entries, key)
		}
	}
}

func (r *Rates) WhiteList(keys ...string) *Rates {
	for _, key := range keys {
		r.whiteList[key] = struct{}{}
	}
	return r
}

func (r *Rates) Allow(key string) bool {
	if _, ok := r.whiteList[key]; ok {
		return true
	}

	entry, ok := r.entries[key]
	if !ok {
		entry = make([]int, r.perNBlocks)
		r.entries[key] = entry
	}
	entry[r.block]++

	if r.Count(key) >= r.requests {
		return false
	}

	return true
}
