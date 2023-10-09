// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

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
		entries:    map[string][]int{},
	}
}

// Count returns the number of requests recorded for a given key
// It returns -1 if the key has been not recorded or evicted.
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

func (r *Rates) Allow(key string) bool {
	entry, ok := r.entries[key]
	if !ok {
		entry = make([]int, r.perNBlocks)
		r.entries[key] = entry
	}
	entry[r.block]++

	return r.Count(key) < r.requests
}
