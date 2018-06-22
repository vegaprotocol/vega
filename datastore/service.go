package datastore

import "vega/datastore/inmemory"
import (
	"vega/proto"
	"fmt"
)

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

func (m *MemoryStorageService) listenForOrders() {
	for orderMsg := range m.orderChan {
		// todo switch on order status (not yet part of proto)

		o := &Order{}
		o = o.fromProtoMessage(orderMsg)
		
		m.orderStore.Put(o)

		fmt.Println("Added order of size %s, price %s", o.Size, o.Price)
	}
}

func (m *MemoryStorageService) listenForTrades() {
	for tradeMsg := range m.tradeChan {

		t := &Trade{}
		t = t.fromProtoMessage(tradeMsg)

		m.tradeStore.Put(t)

		fmt.Println("Added trade of size %s, price %s", t.Size, t.Price)

	}

}


