// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.VEGA file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package broker

import (
	"sync"

	"code.vegaprotocol.io/vega/core/events"
)

type gen struct {
	mu       sync.Mutex
	blockSeq map[string]uint64
	blocks   []string
}

func newGen() *gen {
	return &gen{
		blockSeq: map[string]uint64{},
		blocks:   make([]string, 0, 4),
	}
}

// setSequence adds sequence ID to the event objects, returns the arguments because
// the events might be passed by value (interface values)
// returns the more restrictive event object - once seq ID is set, it should be treated as RO.
func (g *gen) setSequence(evts ...events.Event) []events.Event {
	if len(evts) == 0 {
		return nil
	}
	hash := evts[0].TraceID()
	g.mu.Lock()
	cur, ok := g.blockSeq[hash]
	if !ok {
		g.blocks = append(g.blocks, hash)
		cur = 1
		g.blockSeq[hash] = cur
		// if we're adding a new hash, check if we're up to 3, and remove it if needed
		defer g.cleanID()
	}
	// defer call stack is LIFO, cleanID acquires a lock, so ensure we release it here first
	defer g.mu.Unlock()
	// set sequence ID to the next sequence ID available
	ret := make([]events.Event, 0, len(evts))
	// create slice of ids
	for _, e := range evts {
		e.SetSequenceID(cur)
		// so if cur == 1 (new block), and we send 3 events with composite count of 1 -> cur == 4 (the next seq id to be set)
		// if cur == 1, a composite count of 3 will send the event with seq ID 1, and set the next seq ID to be 1 + 3 == 4
		// unpacking such an event leaves seq. ID 1, 2, and 3 available.
		cur += e.CompositeCount()
		ret = append(ret, e)
	}
	// update the mapk
	g.blockSeq[hash] = cur
	return ret
}

func (g *gen) cleanID() {
	g.mu.Lock()
	if len(g.blocks) == 4 {
		delete(g.blockSeq, g.blocks[0])
		g.blocks = g.blocks[1:]
	}
	g.mu.Unlock()
}
