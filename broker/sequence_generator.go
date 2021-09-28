package broker

import (
	"sync"

	"code.vegaprotocol.io/vega/events"
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
// returns the more restrictive event object - once seq ID is set, it should be treated as RO
func (g *gen) setSequence(evts ...events.Event) []events.Event {
	ln := uint64(len(evts))
	if ln == 0 {
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
	// so current == 1, sending 3 events -> map == 4
	// sequences set are 1 + 0, 1 + 1, and 1 + 2 (so 1 through 3)
	g.blockSeq[hash] += ln
	g.mu.Unlock()
	// set sequence ID to the next sequence ID available
	ret := make([]events.Event, 0, len(evts))
	// create slice of ids
	for i, e := range evts {
		e.SetSequenceID(cur + uint64(i))
		ret = append(ret, e)
	}
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
