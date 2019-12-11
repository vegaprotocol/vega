package orders

import (
	"errors"
	"sync"

	types "code.vegaprotocol.io/vega/proto"
)

var (
	ErrNoOrderForID                 = errors.New("not matching order for id")
	ErrPartyNotFoundInStore         = errors.New("party not found in store")
	ErrPartyOrMarketNotFoundInStore = errors.New("party or market not found in store")
)

type partyMarket struct {
	partyID, marketID string
}

type orderStore struct {
	mu sync.RWMutex
	// partyid -> marketid -> orderid -> order
	store map[string]map[string]map[string]types.Order
	// orderid -> partyMarketRef
	idrefs map[string]partyMarket
}

func newStore() *orderStore {
	return &orderStore{
		store:  map[string]map[string]map[string]types.Order{},
		idrefs: map[string]partyMarket{},
	}
}

func (s *orderStore) SaveBatch(batch []types.Order) {
	s.mu.Lock()
	for _, v := range batch {
		party, ok := s.store[v.PartyID]
		if !ok {
			s.store[v.PartyID] = map[string]map[string]types.Order{}
			party = s.store[v.PartyID]
		}
		mkt, ok := party[v.MarketID]
		if !ok {
			party[v.MarketID] = map[string]types.Order{}
			mkt = party[v.MarketID]
		}

		if v.Status != types.Order_Active {
			delete(mkt, v.Id)
			delete(s.idrefs, v.Id)
		} else {
			if _, ok := s.idrefs[v.Id]; !ok {
				s.idrefs[v.Id] = partyMarket{partyID: v.PartyID, marketID: v.MarketID}
			}
			mkt[v.Id] = v
		}
	}
	s.mu.Unlock()
}

func (s *orderStore) GetByID(id string) (*types.Order, error) {
	s.mu.RLock()
	pm, ok := s.idrefs[id]
	if !ok {
		return nil, ErrNoOrderForID
	}
	o := s.store[pm.partyID][pm.marketID][id]
	s.mu.RUnlock()
	return &o, nil
}

func (s *orderStore) GetByPartyID(partyID string) ([]*types.Order, error) {
	s.mu.RLock()
	party, ok := s.store[partyID]
	if !ok {
		return nil, ErrPartyNotFoundInStore
	}

	var ln int
	for _, v := range party {
		ln += len(v)
	}

	orders := make([]*types.Order, 0, ln)
	for _, mkts := range party {
		for _, ord := range mkts {
			ord := ord
			orders = append(orders, &ord)
		}
	}
	s.mu.RUnlock()
	return orders, nil
}

func (s *orderStore) GetByPartyAndMarketID(partyID, marketID string) ([]*types.Order, error) {
	s.mu.RLock()
	mkt, ok := s.store[partyID][marketID]
	if !ok {
		return nil, ErrPartyOrMarketNotFoundInStore
	}

	orders := make([]*types.Order, 0, len(mkt))
	for _, ord := range mkt {
		ord := ord
		orders = append(orders, &ord)
	}

	s.mu.RUnlock()
	return orders, nil
}
