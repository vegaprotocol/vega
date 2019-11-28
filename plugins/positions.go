package plugins

import (
	"context"
	"sync"

	types "code.vegaprotocol.io/vega/proto"
)

//go:generate go run github.com/golang/mock/mockgen -destination mocks/pos_buffer_mock.go -package mocks code.vegaprotocol.io/vega/plugins PosBuffer
type PosBuffer interface {
	Subscribe() (<-chan map[string]map[string]types.Position, int)
	Unsubscribe(int)
}

// Positions - plugin taking settlement data to build positions API data
type Positions struct {
	mu   *sync.Mutex
	buf  PosBuffer
	ref  int
	ch   <-chan map[string]map[string]types.Position
	data map[string]map[string]types.Position
}

func NewPositions(buf PosBuffer) *Positions {
	return &Positions{
		mu:   &sync.Mutex{},
		data: map[string]map[string]types.Position{},
	}
}

func (p *Positions) Start(ctx context.Context) {
	p.mu.Lock()
	if p.ch == nil {
		// get the channel and the reference
		p.ch, p.ref = p.buf.Subscribe()
		// start consuming the data
		go p.consume(ctx)
	}
	p.mu.Unlock()
}

func (p *Positions) Stop() {
	p.mu.Lock()
	if p.ch != nil {
		// only unsubscribe if ch was set, otherwise we might end up unregistering ref 0, which
		// could (in theory at least) be used by another component
		p.buf.Unsubscribe(p.ref)
	}
	// we don't need to reassign ch here, because the channel is closed, the consume routine
	// will pick up on the fact that we don't have to consume data anylonger, and the ch/ref fields
	// will be unset there
	p.mu.Unlock()
}

// consume - keep reading the channel for as long as we need to
func (p *Positions) consume(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			// we're done consuming, let's unregister the channel
			p.buf.Unsubscribe(p.ref)
			// unset consume-related fields
			p.ref = 0
			p.ch = nil
			return
		case update, open := <-p.ch:
			if !open {
				// the channel was closed, so unset the field:
				p.ref = 0
				p.ch = nil
				return
			}
			p.mu.Lock()
			// @TODO update data intelligently, don't just reassign, this is just a placeholder
			p.data = update
			p.mu.Unlock()
		}
	}
}

func (p *Positions) updateData(update map[string]map[string]types.Position) {
	for mID, traderMap := range update {
		if _, ok := p.data[mID]; !ok {
			// this market is new to the plugin, so the update is actually
			// the initial data
			p.data[mID] = traderMap
			continue
		}
		// marketID is known, let's go over the traders
		for trader, data := range traderMap {
			current, ok := p.data[mID][trader]
			if !ok {
				// trader previously not known to the market, we can just add the data here
				p.data[mID][trader] = data
				continue
			}
			// update data
			// 1. Append to FIFO queue
			current.FifoQueue = append(current.FifoQueue, data.FifoQueue...)
			// @TODO calculations to get avergage entry price etc... are all found in the buffer/positions.go file
			// further work that needs to be done needs to match the python notebook
