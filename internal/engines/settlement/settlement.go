package settlement

import (
	"sync"

	"code.vegaprotocol.io/vega/internal/engines/position"
)

type pos struct {
	size  int64
	price uint64
}

type Engine struct {
	*Config
	mu    *sync.Mutex
	buys  map[string][]*pos
	sells map[string][]*pos
}

func New(conf *Config) *Engine {
	return &Engine{
		Config: conf,
		mu:     &sync.Mutex{},
		buys:   map[string]*pos{},
		sells:  map[string]*pos{},
	}
}

// Update - takes market positions, keeps track of things
func (e *Engine) Update(positions []position.MarketPosition) {
	e.mu.Lock()
	for _, p := range positions {
		party := p.Party()
		if size > 0 {
			ps, ok := e.buys[party]
			if !ok {
				ps = &pos{}
				e.buys[party] = ps
			}
			// price and size are running totals
			ps.size = p.Size()
			ps.price = p.Price()
		} else {
			ps, ok := e.sells[party]
			if !ok {
				ps = &pos{}
				e.sells[party] = ps
			}
			// price and size are running totals
			ps.size = p.Size()
			ps.price = p.Price()
		}
	}
	e.mu.Unlock()
}

func (e *Engine) Settle() {
	e.mu.Lock()
	// first all the buys
	e.settleBuy()
	e.settleSell()
	e.mu.Unlock()
}

func (e *Engine) settleBuy() {
	// mu is locked here
	for party, bpos := range e.buys {
		e.log.Debugf("Settling buys for %s", party)
		netPrice := bpos.price / bpos.size
		e.log.Debugf("Settling %d buys at average price: %d", bpos.Size, netPrice)
	}
}

func (e *Engine) settleSell() {
	for party, spos := range e.sells {
		e.log.Debugf("Settling sales for %s", party)
		netPrice := spos.price / (-spos.size)
		e.log.Debugf("Settling %d sales at average price: %d", spos.size, netPrice)
	}
}
