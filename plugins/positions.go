package plugins

import (
	"context"
	"sync"

	types "code.vegaprotocol.io/vega/proto"

	"github.com/pkg/errors"
)

var (
	MarketNotFoundErr = errors.New("could not find market")
	PartyNotFoundErr  = errors.New("party not found")
)

//go:generate go run github.com/golang/mock/mockgen -destination mocks/pos_buffer_mock.go -package mocks code.vegaprotocol.io/vega/plugins PosBuffer
type PosBuffer interface {
	Subscribe() (<-chan []types.Position, int)
	Unsubscribe(int)
}

// Positions - plugin taking settlement data to build positions API data
type Positions struct {
	mu   *sync.RWMutex
	buf  PosBuffer
	ref  int
	ch   <-chan []types.Position
	data map[string]map[string]types.Position
}

func NewPositions(buf PosBuffer) *Positions {
	return &Positions{
		mu:   &sync.RWMutex{},
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
			p.updateData(update)
			p.mu.Unlock()
		}
	}
}

func (p *Positions) updateData(raw []types.Position) {
	// build the map from the updated data
	p.data = map[string]map[string]types.Position{}
	for _, pos := range raw {
		if _, ok := p.data[pos.MarketID]; !ok {
			p.data[pos.MarketID] = map[string]types.Position{}
		}
		// there can only be 1 position for a trader in a market
		if pos.OpenVolume == 0 {
			delete(p.data[pos.MarketID], pos.PartyID)
		} else {
			p.data[pos.MarketID][pos.PartyID] = pos
		}
	}
}

// GetPositionsByMarketAndParty - get the position of a single trader in a given market
func (p *Positions) GetPositionsByMarketAndParty(market, party string) (*types.Position, error) {
	p.mu.RLock()
	mp, ok := p.data[market]
	if !ok {
		p.mu.RUnlock()
		return nil, MarketNotFoundErr
	}
	pos, ok := mp[party]
	if !ok {
		p.mu.Unlock()
		return nil, PartyNotFoundErr
	}
	p.mu.RUnlock()
	return &pos, nil
}

// GetPositionsByParty - get all positions for a given trader
func (p *Positions) GetPositionsByParty(party string) ([]*types.Position, error) {
	p.mu.RLock()
	// at most, trader is active in all markets
	positions := make([]*types.Position, 0, len(p.data))
	for _, traders := range p.data {
		if pos, ok := traders[party]; ok {
			positions = append(positions, &pos)
		}
	}
	p.mu.RUnlock()
	if len(positions) == 0 {
		return nil, PartyNotFoundErr
	}
	return positions, nil
}

// GetPositionsByMarket - get all trader positions in a given market
func (p *Positions) GetPositionsByMarket(market string) ([]*types.Position, error) {
	p.mu.RLock()
	mp, ok := p.data[market]
	if !ok {
		p.mu.RUnlock()
		return nil, MarketNotFoundErr
	}
	s := make([]*types.Position, 0, len(mp))
	for _, tp := range mp {
		s = append(s, &tp)
	}
	p.mu.RUnlock()
	return s, nil
}
