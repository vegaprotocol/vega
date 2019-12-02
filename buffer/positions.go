package buffer

import (
	"sync"

	"code.vegaprotocol.io/vega/events"
	types "code.vegaprotocol.io/vega/proto"
)

type positionConfig func(*Position)

type Position struct {
	market string
	chBuf  int
	mu     *sync.Mutex
	buf    map[string]types.MarketPosition
	out    map[int]chan map[string]types.MarketPosition
	free   []int
}

func SetChannelBuffer(bufSize int) positionConfig {
	return func(p *Position) {
		p.chBuf = bufSize
	}
}

func New__(marketID string, opts ...positionConfig) *Position {
	p := &Position{
		market: marketID,
		chBuf:  1,
		mu:     &sync.Mutex{},
		buf:    map[string]types.MarketPosition{},
		out:    map[int]chan map[string]types.MarketPosition{},
	}
	for _, o := range opts {
		o(p)
	}
	return p
}

func (p *Position) Add(evt events.MarketPosition) {
	party := evt.Party()
	var pos *types.MarketPosition
	p.mu.Lock()
	if ps, ok := p.buf[party]; ok {
		pos = &ps
	} else {
		ps.MarketID = p.market
		pos = &ps
	}
	if pos.RealisedVolume == 0 {
		pos.RealisedVolume = evt.Size()
		pos.AverageEntryPrice = evt.Price()
		pos.RealisedPNL = evt.Size()
	} else {
		delta := pos.RealisedVolume - evt.Size()
		// (current vol * avg price + delta * price)/(current vol + delta)
		avgPrice := ((pos.RealisedVolume * int64(pos.AverageEntryPrice)) + (abs(delta) * int64(evt.Price()))) / evt.Size()
		pos.RealisedVolume = evt.Size()
		pos.AverageEntryPrice = uint64(avgPrice)
		// add delta
		pos.RealisedPNL += delta
	}
	pos.UnrealisedPNL = evt.Buy() - evt.Sell()
	p.buf[party] = *pos
	p.mu.Unlock()
}

func (p *Position) Flush() {
	p.mu.Lock()
	pm := make(map[string]types.MarketPosition, len(p.buf))
	for k, v := range p.buf {
		pm[k] = v
	}
	// propagate map to all registered whatevers
	for _, ch := range p.out {
		ch <- pm
	}
	p.mu.Unlock()
}

// Register - get a channel to listen to all position updates on Flush
func (p *Position) Register() (<-chan map[string]types.MarketPosition, int) {
	p.mu.Lock()
	ch := make(chan map[string]types.MarketPosition, p.chBuf)
	k := p.getKey()
	p.out[k] = ch
	p.mu.Unlock()
	return ch, k
}

func (p *Position) Unregister(k int) {
	p.mu.Lock()
	// close channel, mark key as available
	if ch, ok := p.out[k]; ok {
		close(ch)
		p.free = append(p.free, k)
	}
	// either way, delete channel from map (no need to check if it exists)
	delete(p.out, k)
	p.mu.Unlock()
}

func (p *Position) getKey() int {
	if len(p.free) > 0 {
		k := p.free[0]
		// remove first element
		p.free = p.free[1:]
		return k
	}
	// the length == the next free ID (0, 1, 2, 3, 4, ...)
	return len(p.out)
}

func abs(v int64) int64 {
	if v < 0 {
		return -v
	}
	return v
}
