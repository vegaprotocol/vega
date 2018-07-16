package datastore

import (
	"testing"
	"github.com/stretchr/testify/assert"
	"vega/proto"
)

func TestMemoryStoreProvider_Init(t *testing.T) {
	memStoreProvider := MemoryStoreProvider{}
	memStoreProvider.Init([]string{ "market" })

	err := memStoreProvider.OrderStore().Post(Order {
		Order: msg.Order{ Id: "order-id", Market: "market", Price: 73921},
	})
	assert.Nil(t, err)

	err = memStoreProvider.TradeStore().Post(Trade {
		Trade: msg.Trade{ Id: "trade-id", Market: "market", Price: 23489},
		OrderId: "order-id",
	})
	assert.Nil(t, err)

	order, err := memStoreProvider.OrderStore().GetByMarketAndId("market", "order-id")
	assert.Nil(t, err)
	assert.Equal(t, uint64(73921), order.Price)

	trade, err := memStoreProvider.TradeStore().GetByMarketAndId("market", "trade-id")
	assert.Nil(t, err)
	assert.Equal(t, uint64(23489), trade.Price)
}