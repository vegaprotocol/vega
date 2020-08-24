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
}

func New(requests, perNBlocks int) *Rates {
	return &Rates{
		block:      0,
		requests:   requests,
		perNBlocks: perNBlocks,
		entries:    make(map[string][]int),
	}
}

func (r *Rates) NextBlock() {
	// compute the next block index
	r.block = (r.block + 1) % (r.perNBlocks)

	// reset the counters for that particular block index
	for _, c := range r.entries {
		c[r.block] = 0
	}

	// TODO(gus): Clean up entries (delete from entries map) with 0 requests
}

func (r *Rates) Allow(key string) bool {
	entries, ok := r.entries[key]
	if !ok {
		entries = make([]int, r.perNBlocks)
		r.entries[key] = entries
	}
	entries[r.block]++

	var count int
	for _, n := range entries {
		count += n
	}

	if count >= r.requests {
		return false
	}

	return true
}
