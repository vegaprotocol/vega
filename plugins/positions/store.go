package positions

import (
	"context"
	"sync"

	types "code.vegaprotocol.io/vega/proto"

	"github.com/pkg/errors"
)

var (
	ErrPositionNotFound = errors.New("closed position not found")
)

type Store struct {
	mu     sync.RWMutex
	closed map[string]map[string]types.Position
}

// NewPositionsStore - in memory store for closed positions
func NewPositionsStore(ctx context.Context) *Store {
	return &Store{
		closed: map[string]map[string]types.Position{},
	}
}

func (s *Store) Pop(marketID, traderID string) (pos *types.Position, err error) {
	s.mu.Lock()
	pos, err = s.get(marketID, traderID)
	if err != nil {
		s.mu.Unlock()
		return
	}
	s.remove(pos)
	s.mu.Unlock()
	return
}

// Get - get a position from store, offloaded to internal function that doesn't handle locks
// because we want to reuse that logic for the Pop call
func (s *Store) Get(marketID, traderID string) (*types.Position, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.get(marketID, traderID)
}

func (s *Store) Add(p types.Position) {
	s.mu.Lock()
	if _, ok := s.closed[p.MarketID]; !ok {
		s.closed[p.MarketID] = map[string]types.Position{}
	}
	s.closed[p.MarketID][p.PartyID] = p
	s.mu.Unlock()
}

// Remove - remove a position from store, offloads the logic to an internal remove func
// that is also used by the Pop func
func (s *Store) Remove(p types.Position) {
	s.mu.Lock()
	s.remove(&p)
	s.mu.Unlock()
}

func (s *Store) get(marketID, traderID string) (*types.Position, error) {
	if mm, ok := s.closed[marketID]; ok {
		if pos, ok := mm[traderID]; ok {
			return &pos, nil
		}
	}
	return nil, ErrPositionNotFound
}

func (s *Store) remove(p *types.Position) {
	if _, ok := s.closed[p.MarketID]; ok {
		delete(s.closed[p.MarketID], p.PartyID)
	}
}
