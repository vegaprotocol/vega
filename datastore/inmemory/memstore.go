package inmemory

import (
	"vega/datastore"
)

type MemStore struct {
	orders map[string]*order
	trades map[string]*trade
}

func NewMemStore() MemStore {
	return MemStore{
		orders: map[string]*order{},
		trades: map[string]*trade{},
	}
}

func NewTradeStore(ms *MemStore) datastore.TradeStore {
	return &tradeStore{store: ms}
}

func NewOrderStore(ms *MemStore) datastore.OrderStore {
	return &orderStore{store: ms}
}


