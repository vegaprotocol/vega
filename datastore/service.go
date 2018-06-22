package datastore

import "vega/datastore/inmemory"
import "vega/proto"

type StorageService interface {
	TradeStore() TradeStore
	OrderStore() OrderStore

	Init (chan<- msg.Order, chan<- msg.Trade)
}

type MemoryStorageService struct {
	memStore inmemory.MemStore
	tradeStore TradeStore
	orderStore OrderStore
	tradeChan chan<- msg.Trade
	orderChan chan<- msg.Order
}

func (m *MemoryStorageService) Init (orderChan chan<- msg.Order, tradeChan chan<- msg.Trade) {
	m.memStore = inmemory.NewMemStore()
	m.tradeStore = inmemory.NewTradeStore(&m.memStore)
	m.orderStore = inmemory.NewOrderStore(&m.memStore)
	m.tradeChan = tradeChan
	m.orderChan = orderChan
}

func (m *MemoryStorageService) TradeStore() TradeStore {
	return m.tradeStore
}

func (m *MemoryStorageService) OrderStore() OrderStore {
	return m.orderStore
}

