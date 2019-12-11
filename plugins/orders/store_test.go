package orders

import (
	"testing"

	types "code.vegaprotocol.io/vega/proto"

	"github.com/stretchr/testify/assert"
)

func TestOrderStore(t *testing.T) {
	t.Run("test add order", testAddOrder)
	t.Run("test order removed", testOrderRemoved)
}

func testAddOrder(t *testing.T) {
	orders := []types.Order{
		{
			Id:       "id1",
			MarketID: "mkt1",
			PartyID:  "pty1",
			Status:   types.Order_Active,
		},
		{
			Id:       "id3",
			MarketID: "mkt2",
			PartyID:  "pty1",
			Status:   types.Order_Active,
		},
		{
			Id:       "id3",
			MarketID: "mkt3",
			PartyID:  "pty1",
			Status:   types.Order_Active,
		},
		{
			Id:       "id4",
			MarketID: "mkt1",
			PartyID:  "pty2",
			Status:   types.Order_Active,
		},
		{
			Id:       "id5",
			MarketID: "mkt1",
			PartyID:  "pty2",
			Status:   types.Order_Active,
		},
		{
			Id:       "id6",
			MarketID: "mkt2",
			PartyID:  "pty3",
			Status:   types.Order_Active,
		},
	}

	store := newStore()
	store.SaveBatch(orders)

	// try to get the orders now
	o, err := store.GetByID("id5")
	assert.Nil(t, err)
	assert.NotNil(t, o)
	assert.Equal(t, "id5", o.Id)

	ords, err := store.GetByPartyID("pty1")
	assert.Nil(t, err)
	assert.NotNil(t, ords)
	assert.Len(t, ords, 3)

	ords, err = store.GetByPartyAndMarketID("pty2", "mkt1")
	assert.Nil(t, err)
	assert.NotNil(t, ords)
	assert.Len(t, ords, 2)
}

func testOrderRemoved(t *testing.T) {
	orders := []types.Order{
		{
			Id:       "id1",
			MarketID: "mkt1",
			PartyID:  "pty1",
			Status:   types.Order_Active,
		},
		{
			Id:       "id3",
			MarketID: "mkt2",
			PartyID:  "pty1",
			Status:   types.Order_Active,
		},
		{
			Id:       "id3",
			MarketID: "mkt3",
			PartyID:  "pty1",
			Status:   types.Order_Active,
		},
		{
			Id:       "id4",
			MarketID: "mkt1",
			PartyID:  "pty2",
			Status:   types.Order_Active,
		},
		{
			Id:       "id5",
			MarketID: "mkt1",
			PartyID:  "pty2",
			Status:   types.Order_Active,
		},
		{
			Id:       "id6",
			MarketID: "mkt2",
			PartyID:  "pty3",
			Status:   types.Order_Active,
		},
	}

	store := newStore()
	store.SaveBatch(orders)

	// expiring one of the order and sending it through a batch again
	store.SaveBatch([]types.Order{{
		Id:       "id3",
		MarketID: "mkt3",
		PartyID:  "pty1",
		Status:   types.Order_Expired,
	}})

	// try to get the orders now
	o, err := store.GetByID("id3")
	assert.NotNil(t, err)
	assert.Error(t, err, ErrNoOrderForID)
	assert.Nil(t, o)
}
