package matching

import (
	"fmt"
	"testing"

	types "vega/proto"

	"github.com/stretchr/testify/assert"
	"time"
	"vega/internal/logging"
)

// launch aggressiveOrder orders from both sides to fully clear the order book
type aggressiveOrderScenario struct {
	aggressiveOrder               *types.Order
	expectedPassiveOrdersAffected []types.Order
	expectedTrades                []types.Trade
}

func getCurrentUtcTimestampNano() uint64 {
	return uint64(time.Now().UTC().UnixNano())
}

func TestOrderBook_RemoveExpiredOrders(t *testing.T) {
	market := "expiringOrderBookTest"
	party := "clay-davis"


	logger := logging.NewLoggerFromEnv("dev")
	defer logger.Sync()

	book := NewBook(market, ProRataModeConfig(logger))
	currentTimestamp := getCurrentUtcTimestampNano()
	someTimeLater := currentTimestamp + (1000*1000)

	order1 := &types.Order{
		Market:    market,
		Party:     party,
		Side:      types.Side_Sell,
		Price:     1,
		Size:      1,
		Remaining: 1,
		Type:      types.Order_GTT,
		Timestamp: currentTimestamp,
		ExpirationTimestamp: someTimeLater,
		Id:        "1",
	}
	_, err := book.AddOrder(order1)
	assert.Equal(t, err, types.OrderError_NONE)

	order2 := &types.Order{
		Market:    market,
		Party:     party,
		Side:      types.Side_Sell,
		Price:     3298,
		Size:      99,
		Remaining: 99,
		Type:      types.Order_GTT,
		Timestamp: currentTimestamp,
		ExpirationTimestamp: someTimeLater+1,
		Id:        "2",
	}
	_, err = book.AddOrder(order2)
	assert.Equal(t, err, types.OrderError_NONE)

	order3 := &types.Order{
		Market:    market,
		Party:     party,
		Side:      types.Side_Sell,
		Price:     771,
		Size:      19,
		Remaining: 19,
		Type:      types.Order_GTT,
		Timestamp: currentTimestamp,
		ExpirationTimestamp: someTimeLater,
		Id:        "3",
	}
	_, err = book.AddOrder(order3)
	assert.Equal(t, err, types.OrderError_NONE)

	order4 := &types.Order{
		Market:    market,
		Party:     party,
		Side:      types.Side_Sell,
		Price:     1000,
		Size:      7,
		Remaining: 7,
		Type:      types.Order_GTC,
		Timestamp: currentTimestamp,
		Id:        "4",
	}
	_, err = book.AddOrder(order4)
	assert.Equal(t, err, types.OrderError_NONE)

	order5 := &types.Order{
		Market:    market,
		Party:     party,
		Side:      types.Side_Sell,
		Price:     199,
		Size:      99999,
		Remaining: 99999,
		Type:      types.Order_GTT,
		Timestamp: currentTimestamp,
		ExpirationTimestamp: someTimeLater,
		Id:        "5",
	}
	_, err = book.AddOrder(order5)
	assert.Equal(t, err, types.OrderError_NONE)

	order6 := &types.Order{
		Market:    market,
		Party:     party,
		Side:      types.Side_Sell,
		Price:     100,
		Size:      100,
		Remaining: 100,
		Type:      types.Order_GTC,
		Timestamp: currentTimestamp,
		Id:        "6",
	}
	_, err = book.AddOrder(order6)
	assert.Equal(t, err, types.OrderError_NONE)

	order7 := &types.Order{
		Market:    market,
		Party:     party,
		Side:      types.Side_Sell,
		Price:     41,
		Size:      9999,
		Remaining: 9999,
		Type:      types.Order_GTT,
		Timestamp: currentTimestamp,
		ExpirationTimestamp: someTimeLater+9999,
		Id:        "7",
	}
	_, err = book.AddOrder(order7)
	assert.Equal(t, err, types.OrderError_NONE)

	order8 := &types.Order{
		Market:    market,
		Party:     party,
		Side:      types.Side_Sell,
		Price:     1,
		Size:      1,
		Remaining: 1,
		Type:      types.Order_GTT,
		Timestamp: currentTimestamp,
		ExpirationTimestamp: someTimeLater-9999,
		Id:        "8",
	}
	_, err = book.AddOrder(order8)
	assert.Equal(t, err, types.OrderError_NONE)
	
	order9 := &types.Order{
		Market:    market,
		Party:     party,
		Side:      types.Side_Sell,
		Price:     65,
		Size:      12,
		Remaining: 12,
		Type:      types.Order_GTC,
		Timestamp: currentTimestamp,
		Id:        "9",
	}
	_, err = book.AddOrder(order9)
	assert.Equal(t, err, types.OrderError_NONE)
	
	order10 := &types.Order{
		Market:    market,
		Party:     party,
		Side:      types.Side_Sell,
		Price:     1,
		Size:      1,
		Remaining: 1,
		Type:      types.Order_GTT,
		Timestamp: currentTimestamp,
		ExpirationTimestamp: someTimeLater-1,
		Id:        "10",
	}
	_, err = book.AddOrder(order10)
	assert.Equal(t, err, types.OrderError_NONE)

	expired := book.RemoveExpiredOrders(someTimeLater)
	assert.Len(t, expired, 5)
	assert.Equal(t, "1", expired[0].Id)
	assert.Equal(t, "3", expired[1].Id)
	assert.Equal(t, "5", expired[2].Id)
	assert.Equal(t, "8", expired[3].Id)
	assert.Equal(t, "10", expired[4].Id)
}

//test for order validation
func TestOrderBook_AddOrder2WithValidation(t *testing.T) {

	logger := logging.NewLoggerFromEnv("dev")
	defer logger.Sync()

	book := NewBook("testOrderBook", ProRataModeConfig(logger))
	book.latestTimestamp = 10

	invalidTimestampOrdertypes := &types.Order{
		Market:    "testOrderBook",
		Party:     "A",
		Side:      types.Side_Sell,
		Price:     100,
		Size:      100,
		Remaining: 100,
		Type:      types.Order_GTC,
		Timestamp: 0,
		Id:        "id-number-one",
	}
	_, err := book.AddOrder(invalidTimestampOrdertypes)
	assert.Equal(t, types.OrderError_ORDER_OUT_OF_SEQUENCE, err)

	book.latestTimestamp = 0
	invalidRemainginSizeOrdertypes := &types.Order{
		Market:    "testOrderBook",
		Party:     "A",
		Side:      types.Side_Sell,
		Price:     100,
		Size:      100,
		Remaining: 300,
		Type:      types.Order_GTC,
		Timestamp: 0,
		Id:        "id-number-one",
	}
	_, err = book.AddOrder(invalidRemainginSizeOrdertypes)
	assert.Equal(t, types.OrderError_INVALID_REMAINING_SIZE, err)
}

func TestOrderBook_RemoveOrder(t *testing.T) {
	logger := logging.NewLoggerFromEnv("dev")
	defer logger.Sync()
	book := NewBook("testOrderBook", ProRataModeConfig(logger))

	newOrder := &types.Order{
		Market:    "testOrderBook",
		Party:     "A",
		Side:      types.Side_Sell,
		Price:     101,
		Size:      100,
		Remaining: 100,
		Type:      types.Order_GTC,
		Timestamp: 0,
	}

	book.AddOrder(newOrder)

	err := book.RemoveOrder(newOrder)
	if err != nil {
		fmt.Println(err, "ORDER_NOT_FOUND")
	}

	book.PrintState("AFTER REMOVE ORDER")
}

func TestOrderBook_AddOrder(t *testing.T) {
	logger := logging.NewLoggerFromEnv("dev")
	defer logger.Sync()
	
	book := NewBook("testOrderBook", ProRataModeConfig(logger))

	const numberOfTimestamps = 3
	m := make(map[int64][]*types.Order, numberOfTimestamps)

	// sell and buy side orders at timestamp 0
	m[0] = []*types.Order{
		// Side Sell
		{
			Market:    "testOrderBook",
			Party:     "A",
			Side:      types.Side_Sell,
			Price:     101,
			Size:      100,
			Remaining: 100,
			Type:      types.Order_GTC,
			Timestamp: 0,
		},
		{
			Market:    "testOrderBook",
			Party:     "B",
			Side:      types.Side_Sell,
			Price:     101,
			Size:      100,
			Remaining: 100,
			Type:      types.Order_GTC,
			Timestamp: 0,
		},
		{
			Market:    "testOrderBook",
			Party:     "C",
			Side:      types.Side_Sell,
			Price:     102,
			Size:      100,
			Remaining: 100,
			Type:      types.Order_GTC,
			Timestamp: 0,
		},
		{
			Market:    "testOrderBook",
			Party:     "D",
			Side:      types.Side_Sell,
			Price:     103,
			Size:      100,
			Remaining: 100,
			Type:      types.Order_GTC,
			Timestamp: 0,
		},
		// Side Buy
		{
			Market:    "testOrderBook",
			Party:     "E",
			Side:      types.Side_Buy,
			Price:     99,
			Size:      100,
			Remaining: 100,
			Type:      types.Order_GTC,
			Timestamp: 0,
		},
		{
			Market:    "testOrderBook",
			Party:     "F",
			Side:      types.Side_Buy,
			Price:     99,
			Size:      100,
			Remaining: 100,
			Type:      types.Order_GTC,
			Timestamp: 0,
		},
		{
			Market:    "testOrderBook",
			Party:     "G",
			Side:      types.Side_Buy,
			Price:     98,
			Size:      100,
			Remaining: 100,
			Type:      types.Order_GTC,
			Timestamp: 0,
		},
	}

	// sell and buy orders at timestamp 1
	m[1] = []*types.Order{
		// Side Sell
		{
			Market:    "testOrderBook",
			Party:     "M",
			Side:      types.Side_Sell,
			Price:     101,
			Size:      100,
			Remaining: 100,
			Type:      types.Order_GTC,
			Timestamp: 1,
		},
		// Side Buy
		{
			Market:    "testOrderBook",
			Party:     "N",
			Side:      types.Side_Buy,
			Price:     99,
			Size:      100,
			Remaining: 100,
			Type:      types.Order_GTC,
			Timestamp: 1,
		},
	}

	// sell and buy orders at timestamp 2
	m[2] = []*types.Order{
		// Side Sell
		{
			Market:    "testOrderBook",
			Party:     "R",
			Side:      types.Side_Sell,
			Price:     101,
			Size:      100,
			Remaining: 100,
			Type:      types.Order_GTC,
			Timestamp: 2,
		},
		// Side Buy
		{
			Market:    "testOrderBook",
			Party:     "S",
			Side:      types.Side_Buy,
			Price:     99,
			Size:      100,
			Remaining: 100,
			Type:      types.Order_GTC,
			Timestamp: 2,
		},
	}

	timestamps := []int64{0, 1, 2}
	for _, timestamp := range timestamps {
		for index, _ := range m[timestamp] {
			fmt.Println("tests calling book.AddOrder: ", m[timestamp][index])
			confirmationtypes, err := book.AddOrder(m[timestamp][index])
			// this should not return any errors
			assert.Equal(t, types.OrderError_NONE, err)
			// this should not generate any trades
			assert.Equal(t, 0, len(confirmationtypes.Trades))
		}
	}

	scenario := []aggressiveOrderScenario{
		{
			// same price level, remaining on the passive
			aggressiveOrder: &types.Order{
				Market:    "testOrderBook",
				Party:     "X",
				Side:      types.Side_Buy,
				Price:     101,
				Size:      100,
				Remaining: 100,
				Type:      types.Order_GTC,
				Timestamp: 3,
			},
			expectedTrades: []types.Trade{
				{
					Market:    "testOrderBook",
					Price:     101,
					Size:      50,
					Buyer:     "X",
					Seller:    "A",
					Aggressor: types.Side_Buy,
				},
				{
					Market:    "testOrderBook",
					Price:     101,
					Size:      50,
					Buyer:     "X",
					Seller:    "B",
					Aggressor: types.Side_Buy,
				},
			},
			expectedPassiveOrdersAffected: []types.Order{
				{
					Market:    "testOrderBook",
					Party:     "A",
					Side:      types.Side_Sell,
					Price:     101,
					Size:      100,
					Remaining: 50,
					Type:      types.Order_GTC,
					Timestamp: 0,
				},
				{
					Market:    "testOrderBook",
					Party:     "B",
					Side:      types.Side_Sell,
					Price:     101,
					Size:      100,
					Remaining: 50,
					Type:      types.Order_GTC,
					Timestamp: 0,
				},
			},
		},
		{
			// lower price is available on the passive side, 2 orders removed, 1 passive remaining
			aggressiveOrder: &types.Order{
				Market:    "testOrderBook",
				Party:     "Y",
				Side:      types.Side_Buy,
				Price:     102,
				Size:      150,
				Remaining: 150,
				Type:      types.Order_GTC,
				Timestamp: 3,
			},
			expectedTrades: []types.Trade{
				{
					Market:    "testOrderBook",
					Price:     101,
					Size:      50,
					Buyer:     "Y",
					Seller:    "A",
					Aggressor: types.Side_Buy,
				},
				{
					Market:    "testOrderBook",
					Price:     101,
					Size:      50,
					Buyer:     "Y",
					Seller:    "B",
					Aggressor: types.Side_Buy,
				},
				{
					Market:    "testOrderBook",
					Price:     101,
					Size:      50,
					Buyer:     "Y",
					Seller:    "M",
					Aggressor: types.Side_Buy,
				},
			},
			expectedPassiveOrdersAffected: []types.Order{
				{
					Market:    "testOrderBook",
					Party:     "A",
					Side:      types.Side_Sell,
					Price:     101,
					Size:      100,
					Remaining: 0,
					Type:      types.Order_GTC,
					Timestamp: 0,
				},
				{
					Market:    "testOrderBook",
					Party:     "B",
					Side:      types.Side_Sell,
					Price:     101,
					Size:      100,
					Remaining: 0,
					Type:      types.Order_GTC,
					Timestamp: 0,
				},
				{
					Market:    "testOrderBook",
					Party:     "M",
					Side:      types.Side_Sell,
					Price:     101,
					Size:      100,
					Remaining: 50,
					Type:      types.Order_GTC,
					Timestamp: 1,
				},
			},
		},
		{
			// lower price is available on the passive side, 1 order removed, 1 passive remaining
			aggressiveOrder: &types.Order{
				Market:    "testOrderBook",
				Party:     "Z",
				Side:      types.Side_Buy,
				Price:     102,
				Size:      70,
				Remaining: 70,
				Type:      types.Order_GTC,
				Timestamp: 3,
			},
			expectedTrades: []types.Trade{
				{
					Market:    "testOrderBook",
					Price:     101,
					Size:      50,
					Buyer:     "Z",
					Seller:    "M",
					Aggressor: types.Side_Buy,
				},
				{
					Market:    "testOrderBook",
					Price:     101,
					Size:      20,
					Buyer:     "Z",
					Seller:    "R",
					Aggressor: types.Side_Buy,
				},
			},
			expectedPassiveOrdersAffected: []types.Order{
				{
					Market:    "testOrderBook",
					Party:     "M",
					Side:      types.Side_Sell,
					Price:     101,
					Size:      100,
					Remaining: 0,
					Type:      types.Order_GTC,
					Timestamp: 1,
				},
				{
					Market:    "testOrderBook",
					Party:     "R",
					Side:      types.Side_Sell,
					Price:     101,
					Size:      100,
					Remaining: 80,
					Type:      types.Order_GTC,
					Timestamp: 2,
				},
			},
		},
		{
			// price level jump, lower price is available on the passive side but its entirely consumed,
			// 1 order removed, 1 passive remaining at higher price level
			aggressiveOrder: &types.Order{
				Market:    "testOrderBook",
				Party:     "X",
				Side:      types.Side_Buy,
				Price:     102,
				Size:      100,
				Remaining: 100,
				Type:      types.Order_GTC,
				Timestamp: 3,
			},
			expectedTrades: []types.Trade{
				{
					Market:    "testOrderBook",
					Price:     101,
					Size:      80,
					Buyer:     "X",
					Seller:    "R",
					Aggressor: types.Side_Buy,
				},
				{
					Market:    "testOrderBook",
					Price:     102,
					Size:      20,
					Buyer:     "X",
					Seller:    "C",
					Aggressor: types.Side_Buy,
				},
			},
			expectedPassiveOrdersAffected: []types.Order{
				{
					Market:    "testOrderBook",
					Party:     "R",
					Side:      types.Side_Sell,
					Price:     101,
					Size:      100,
					Remaining: 0,
					Type:      types.Order_GTC,
					Timestamp: 2,
				},
				{
					Market:    "testOrderBook",
					Party:     "C",
					Side:      types.Side_Sell,
					Price:     102,
					Size:      100,
					Remaining: 80,
					Type:      types.Order_GTC,
					Timestamp: 0,
				},
			},
		},
		{
			// Sell is agressive, aggressive at lower price than on the book, pro rata at 99, aggressive is removed
			aggressiveOrder: &types.Order{
				Market:    "testOrderBook",
				Party:     "Y",
				Side:      types.Side_Sell,
				Price:     98,
				Size:      100,
				Remaining: 100,
				Type:      types.Order_GTC,
				Timestamp: 4,
			},
			expectedTrades: []types.Trade{
				{
					Market:    "testOrderBook",
					Price:     99,
					Size:      50,
					Buyer:     "E",
					Seller:    "Y",
					Aggressor: types.Side_Sell,
				},
				{
					Market:    "testOrderBook",
					Price:     99,
					Size:      50,
					Buyer:     "F",
					Seller:    "Y",
					Aggressor: types.Side_Sell,
				},
			},
			expectedPassiveOrdersAffected: []types.Order{
				{
					Market:    "testOrderBook",
					Party:     "E",
					Side:      types.Side_Buy,
					Price:     99,
					Size:      100,
					Remaining: 50,
					Type:      types.Order_GTC,
					Timestamp: 0,
				},
				{
					Market:    "testOrderBook",
					Party:     "F",
					Side:      types.Side_Buy,
					Price:     99,
					Size:      100,
					Remaining: 50,
					Type:      types.Order_GTC,
					Timestamp: 0,
				},
			},
		},
		{
			// Sell is agressive, aggressive at exact price, all orders at this price level should be hitted plus order should remain on the sell side of the book at 99 level
			aggressiveOrder: &types.Order{
				Market:    "testOrderBook",
				Party:     "Z",
				Side:      types.Side_Sell,
				Price:     99,
				Size:      350,
				Remaining: 350,
				Type:      types.Order_GTC,
				Timestamp: 4,
			},
			expectedTrades: []types.Trade{
				{
					Market:    "testOrderBook",
					Price:     99,
					Size:      50,
					Buyer:     "E",
					Seller:    "Z",
					Aggressor: types.Side_Sell,
				},
				{
					Market:    "testOrderBook",
					Price:     99,
					Size:      50,
					Buyer:     "F",
					Seller:    "Z",
					Aggressor: types.Side_Sell,
				},
				{
					Market:    "testOrderBook",
					Price:     99,
					Size:      100,
					Buyer:     "N",
					Seller:    "Z",
					Aggressor: types.Side_Sell,
				},
				{
					Market:    "testOrderBook",
					Price:     99,
					Size:      100,
					Buyer:     "S",
					Seller:    "Z",
					Aggressor: types.Side_Sell,
				},
			},
			expectedPassiveOrdersAffected: []types.Order{
				{
					Market:    "testOrderBook",
					Party:     "E",
					Side:      types.Side_Buy,
					Price:     99,
					Size:      100,
					Remaining: 0,
					Type:      types.Order_GTC,
					Timestamp: 0,
				},
				{
					Market:    "testOrderBook",
					Party:     "F",
					Side:      types.Side_Buy,
					Price:     99,
					Size:      100,
					Remaining: 0,
					Type:      types.Order_GTC,
					Timestamp: 0,
				},
				{
					Market:    "testOrderBook",
					Party:     "N",
					Side:      types.Side_Buy,
					Price:     99,
					Size:      100,
					Remaining: 0,
					Type:      types.Order_GTC,
					Timestamp: 1,
				},
				{
					Market:    "testOrderBook",
					Party:     "S",
					Side:      types.Side_Buy,
					Price:     99,
					Size:      100,
					Remaining: 0,
					Type:      types.Order_GTC,
					Timestamp: 2,
				},
			},
		},
		{ // aggressive nonpersistent buy order, hits two price levels and is not added to order book
			aggressiveOrder: &types.Order{
				Market:    "testOrderBook",
				Party:     "XX",
				Side:      types.Side_Buy,
				Price:     102,
				Size:      100,
				Remaining: 100,
				Type:      types.Order_FOK, // nonpersistent
				Timestamp: 4,
			},
			expectedTrades: []types.Trade{
				{
					Market:    "testOrderBook",
					Price:     99,
					Size:      50,
					Buyer:     "XX",
					Seller:    "Z",
					Aggressor: types.Side_Buy,
				},
				{
					Market:    "testOrderBook",
					Price:     102,
					Size:      50,
					Buyer:     "XX",
					Seller:    "C",
					Aggressor: types.Side_Buy,
				},
			},
			expectedPassiveOrdersAffected: []types.Order{
				{
					Market:    "testOrderBook",
					Party:     "Z",
					Side:      types.Side_Sell,
					Price:     99,
					Size:      350,
					Remaining: 0,
					Type:      types.Order_GTC,
					Timestamp: 4,
				},
				{
					Market:    "testOrderBook",
					Party:     "C",
					Side:      types.Side_Sell,
					Price:     102,
					Size:      100,
					Remaining: 30,
					Type:      types.Order_GTC,
					Timestamp: 0,
				},
			},
		},
		{ // aggressive nonpersistent buy order, hits one price levels and is not added to order book
			aggressiveOrder: &types.Order{
				Market:    "testOrderBook",
				Party:     "YY",
				Side:      types.Side_Buy,
				Price:     103,
				Size:      200,
				Remaining: 200,
				Type:      types.Order_ENE, // nonpersistent
				Timestamp: 5,
			},
			expectedTrades: []types.Trade{
				{
					Market:    "testOrderBook",
					Price:     102,
					Size:      30,
					Buyer:     "YY",
					Seller:    "C",
					Aggressor: types.Side_Buy,
				},
				{
					Market:    "testOrderBook",
					Price:     103,
					Size:      100,
					Buyer:     "YY",
					Seller:    "D",
					Aggressor: types.Side_Buy,
				},
			},
			expectedPassiveOrdersAffected: []types.Order{
				{
					Market:    "testOrderBook",
					Party:     "C",
					Side:      types.Side_Sell,
					Price:     102,
					Size:      100,
					Remaining: 0,
					Type:      types.Order_GTC,
					Timestamp: 0,
				},
				{
					Market:    "testOrderBook",
					Party:     "D",
					Side:      types.Side_Sell,
					Price:     103,
					Size:      100,
					Remaining: 0,
					Type:      types.Order_GTC,
					Timestamp: 0,
				},
			},
		},
		{ // aggressive nonpersistent buy order, hits two price levels and is not added to order book
			aggressiveOrder: &types.Order{
				Market:    "testOrderBook",
				Party:     "XX",
				Side:      types.Side_Sell,
				Price:     96,
				Size:      2000,
				Remaining: 2000,
				Type:      types.Order_FOK, // nonpersistent
				Timestamp: 5,
			},
			expectedTrades:                []types.Trade{},
			expectedPassiveOrdersAffected: []types.Order{},
		},
		{ // aggressive nonpersistent buy order, hits two price levels and is not added to order book
			aggressiveOrder: &types.Order{
				Market:    "testOrderBook",
				Party:     "XX",
				Side:      types.Side_Buy,
				Price:     102,
				Size:      2000,
				Remaining: 2000,
				Type:      types.Order_FOK, // nonpersistent
				Timestamp: 5,
			},
			expectedTrades:                []types.Trade{},
			expectedPassiveOrdersAffected: []types.Order{},
		},
		{ // aggressive nonpersistent buy order, at super low price hits one price levels and is not added to order book
			aggressiveOrder: &types.Order{
				Market:    "testOrderBook",
				Party:     "ZZ",
				Side:      types.Side_Sell,
				Price:     95,
				Size:      200,
				Remaining: 200,
				Type:      types.Order_ENE, // nonpersistent
				Timestamp: 5,
			},
			expectedTrades: []types.Trade{
				{
					Market:    "testOrderBook",
					Price:     98,
					Size:      100,
					Buyer:     "G",
					Seller:    "ZZ",
					Aggressor: types.Side_Sell,
				},
			},
			expectedPassiveOrdersAffected: []types.Order{
				{
					Market:    "testOrderBook",
					Party:     "G",
					Side:      types.Side_Buy,
					Price:     98,
					Size:      100,
					Remaining: 0,
					Type:      types.Order_GTC,
					Timestamp: 0,
				},
			},
		},
		{ // aggressive nonpersistent buy order, at super low price hits one price levels and is not added to order book
			aggressiveOrder: &types.Order{
				Market:              "testOrderBook",
				Party:               "ZZ",
				Side:                types.Side_Sell,
				Price:               95,
				Size:                200,
				Remaining:           200,
				Type:                types.Order_GTT, // nonpersistent
				Timestamp:           5,
				ExpirationDatetime:  "2006-01-02T15:04:05Z07:00",
				ExpirationTimestamp: 6,
			},
			expectedTrades:                []types.Trade{},
			expectedPassiveOrdersAffected: []types.Order{},
		},
		{ // aggressive nonpersistent buy order, at super low price hits one price levels and is not added to order book
			aggressiveOrder: &types.Order{
				Market:    "testOrderBook",
				Party:     "ZXY",
				Side:      types.Side_Buy,
				Price:     95,
				Size:      100,
				Remaining: 100,
				Type:      types.Order_FOK, // nonpersistent
				Timestamp: 6,
			},
			expectedTrades: []types.Trade{
				{
					Market:    "testOrderBook",
					Price:     95,
					Size:      100,
					Buyer:     "ZXY",
					Seller:    "ZZ",
					Aggressor: types.Side_Buy,
				},
			},
			expectedPassiveOrdersAffected: []types.Order{
				{
					Market:              "testOrderBook",
					Party:               "ZZ",
					Side:                types.Side_Sell,
					Price:               95,
					Size:                200,
					Remaining:           100,
					Type:                types.Order_GTT, // nonpersistent
					Timestamp:           5,
					ExpirationDatetime:  "2006-01-02T15:04:05Z07:00",
					ExpirationTimestamp: 7,
				},
			},
		},
		{ // aggressive nonpersistent buy order, hits two price levels and is not added to order book
			aggressiveOrder: &types.Order{
				Market:    "testOrderBook",
				Party:     "XX",
				Side:      types.Side_Buy,
				Price:     102,
				Size:      2000,
				Remaining: 2000,
				Type:      types.Order_FOK, // nonpersistent
				Timestamp: 6,
			},
			expectedTrades:                []types.Trade{},
			expectedPassiveOrdersAffected: []types.Order{},
		},
		// expect empty book after that as remaining order GTT has to expire
	}

	for i, s := range scenario {
		fmt.Println()
		fmt.Println()
		fmt.Printf("SCENARIO %d / %d ------------------------------------------------------------------", i+1, len(scenario))
		fmt.Println()
		fmt.Println("aggressor: ", s.aggressiveOrder)
		fmt.Println("expectedPassiveOrdersAffected: ", s.expectedPassiveOrdersAffected)
		fmt.Println("expectedTrades: ", s.expectedTrades)
		fmt.Println()

		confirmationtypes, err := book.AddOrder(s.aggressiveOrder)

		//this should not return any errors
		assert.Equal(t, types.OrderError_NONE, err)

		//this should not generate any trades
		assert.Equal(t, len(s.expectedTrades), len(confirmationtypes.Trades))

		fmt.Println("CONFIRMATION types:")
		fmt.Println("-> Aggresive:", confirmationtypes.Order)
		fmt.Println("-> Trades :", confirmationtypes.Trades)
		fmt.Println("-> PassiveOrdersAffected:", confirmationtypes.PassiveOrdersAffected)
		fmt.Printf("Scenario: %d / %d \n", i+1, len(scenario))

		// trades should match expected trades
		for i, trade := range confirmationtypes.Trades {
			expectTrade(t, &s.expectedTrades[i], trade)
		}

		// orders affected should match expected values
		for i, orderAffected := range confirmationtypes.PassiveOrdersAffected {
			expectOrder(t, &s.expectedPassiveOrdersAffected[i], orderAffected)
		}

		// call remove expired orders every scenario
		book.RemoveExpiredOrders(s.aggressiveOrder.Timestamp)
	}
}

func TestOrderBook_AddOrderInvalidMarket(t *testing.T) {

	logger := logging.NewLoggerFromEnv("dev")
	defer logger.Sync()

	book := NewBook("testOrderBook", ProRataModeConfig(logger))
	newOrder := &types.Order{
		Market:    "invalid",
		Party:     "A",
		Side:      types.Side_Sell,
		Price:     101,
		Size:      100,
		Remaining: 100,
		Type:      types.Order_GTC,
		Timestamp: 0,
		Id:        fmt.Sprintf("V%d-%d", 1, 1),
	}

	_, err := book.AddOrder(newOrder)
	if err != types.OrderError_NONE {
		fmt.Println(err)
	}

	assert.Equal(t, types.OrderError_INVALID_MARKET_ID, err)

}

func TestOrderBook_CancelSellOrder(t *testing.T) {
	logger := logging.NewLoggerFromEnv("dev")
	defer logger.Sync()

	logger.Debug("BEGIN CANCELLING VALID ORDER")

	// Arrange
	book := NewBook("testOrderBook", ProRataModeConfig(logger))
	newOrder := &types.Order{
		Market:    "testOrderBook",
		Party:     "A",
		Side:      types.Side_Sell,
		Price:     101,
		Size:      100,
		Remaining: 100,
		Type:      types.Order_GTC,
		Timestamp: 0,
		Id:        fmt.Sprintf("V%d-%d", 1, 1),
	}

	confirmation, err := book.AddOrder(newOrder)
	orderAdded := confirmation.Order

	// Act
	res, err := book.CancelOrder(orderAdded)
	if err != types.OrderError_NONE {
		fmt.Println(err)
	}

	// Assert
	assert.Equal(t, types.OrderError_NONE, err)
	assert.Equal(t, "V1-1", res.Order.Id)
	assert.Equal(t, types.Order_Cancelled, res.Order.Status)

	book.PrintState("AFTER CANCEL ORDER")
}

func TestOrderBook_CancelBuyOrder(t *testing.T) {
	logger := logging.NewLoggerFromEnv("dev")
	defer logger.Sync()

	logger.Debug("BEGIN CANCELLING VALID ORDER")

	// Arrange
	book := NewBook("testOrderBook", ProRataModeConfig(logger))
	newOrder := &types.Order{
		Market:    "testOrderBook",
		Party:     "A",
		Side:      types.Side_Buy,
		Price:     101,
		Size:      100,
		Remaining: 100,
		Type:      types.Order_GTC,
		Timestamp: 0,
		Id:        fmt.Sprintf("V%d-%d", 1, 1),
	}

	confirmation, err := book.AddOrder(newOrder)
	orderAdded := confirmation.Order

	// Act
	res, err := book.CancelOrder(orderAdded)
	if err != types.OrderError_NONE {
		fmt.Println(err)
	}

	// Assert
	assert.Equal(t, types.OrderError_NONE, err)
	assert.Equal(t, "V1-1", res.Order.Id)
	assert.Equal(t, types.Order_Cancelled, res.Order.Status)

	book.PrintState("AFTER CANCEL ORDER")
}

func TestOrderBook_CancelOrderMarketMismatch(t *testing.T) {
	logger := logging.NewLoggerFromEnv("dev")
	defer logger.Sync()

	logger.Debug("BEGIN CANCELLING MARKET MISMATCH ORDER")

	book := NewBook("testOrderBook", ProRataModeConfig(logger))
	newOrder := &types.Order{
		Market: "testOrderBook",
		Id:     "123456",
	}

	confirmation, err := book.AddOrder(newOrder)
	orderAdded := confirmation.Order

	orderAdded.Market = "invalid" // Bad market, malformed?

	_, err = book.CancelOrder(orderAdded)
	if err != types.OrderError_NONE {
		fmt.Println(err)
	}

	assert.Equal(t, types.OrderError_INVALID_MARKET_ID, err)
}

func TestOrderBook_CancelOrderInvalidID(t *testing.T) {
	logger := logging.NewLoggerFromEnv("dev")
	defer logger.Sync()

	logger.Debug("BEGIN CANCELLING INVALID ORDER")

	book := NewBook("testOrderBook", ProRataModeConfig(logger))
	newOrder := &types.Order{
		Market: "testOrderBook",
		Id:     "id",
	}

	confirmation, err := book.AddOrder(newOrder)
	orderAdded := confirmation.Order

	_, err = book.CancelOrder(orderAdded)
	if err != types.OrderError_NONE {
		logger.Debug(err.String())
	}

	assert.Equal(t, types.OrderError_INVALID_ORDER_ID, err)
}

func expectTrade(t *testing.T, expectedTrade, trade *types.Trade) {
	// run asserts for protocol trade data
	assert.Equal(t, expectedTrade.Price, trade.Price)
	assert.Equal(t, expectedTrade.Size, trade.Size)
	assert.Equal(t, expectedTrade.Buyer, trade.Buyer)
	assert.Equal(t, expectedTrade.Seller, trade.Seller)
	assert.Equal(t, expectedTrade.Aggressor, trade.Aggressor)
}

func expectOrder(t *testing.T, expectedOrder, order *types.Order) {
	// run asserts for order
	assert.Equal(t, expectedOrder.Market, order.Market)
	assert.Equal(t, expectedOrder.Party, order.Party)
	assert.Equal(t, expectedOrder.Side, order.Side)
	assert.Equal(t, expectedOrder.Price, order.Price)
	assert.Equal(t, expectedOrder.Size, order.Size)
	assert.Equal(t, expectedOrder.Remaining, order.Remaining)
	assert.Equal(t, expectedOrder.Type, order.Type)
	assert.Equal(t, expectedOrder.Timestamp, order.Timestamp)
}

func TestOrderBook_AmendOrder(t *testing.T) {
	logger := logging.NewLoggerFromEnv("dev")
	defer logger.Sync()
	
	book := NewBook("testOrderBook", ProRataModeConfig(logger))
	newOrder := &types.Order{
		Market:    "testOrderBook",
		Id:        "123456",
		Side:      types.Side_Buy,
		Price:     100,
		Size:      200,
		Remaining: 200,
		Type:      types.Order_GTC,
	}

	confirmation, err := book.AddOrder(newOrder)
	if err != types.OrderError_NONE {
		t.Log(err)
	}
	
	assert.Equal(t, types.OrderError_NONE, err)
	assert.NotNil(t, confirmation)
	assert.Equal(t, "123456", confirmation.Order.Id)
	assert.Equal(t, 0, len(confirmation.Trades))

	editedOrder := &types.Order{
		Market:    "testOrderBook",
		Id:        "123456",
		Side:      types.Side_Buy,
		Price:     100,
		Size:      200,
		Remaining: 200,
		Type:      types.Order_GTC,
	}

	err = book.AmendOrder(editedOrder)
	if err != types.OrderError_NONE {
		t.Log(err)
	}
	
	assert.Equal(t, types.OrderError_NONE, err)
}

func TestOrderBook_AmendOrderInvalidRemaining(t *testing.T) {
	logger := logging.NewLoggerFromEnv("dev")
	defer logger.Sync()

	book := NewBook("testOrderBook", ProRataModeConfig(logger))
	newOrder := &types.Order{
		Market:    "testOrderBook",
		Id:        "123456",
		Side:      types.Side_Buy,
		Price:     100,
		Size:      200,
		Remaining: 200,
		Type:      types.Order_GTC,
	}

	confirmation, err := book.AddOrder(newOrder)
	if err != types.OrderError_NONE {
		t.Log(err)
	}

	assert.Equal(t, types.OrderError_NONE, err)
	assert.NotNil(t, confirmation)
	assert.Equal(t, "123456", confirmation.Order.Id)
	assert.Equal(t, 0, len(confirmation.Trades))

	editedOrder := &types.Order{
		Market:    "testOrderBook",
		Id:        "123456",
		Side:      types.Side_Sell,
		Price:     100,
		Size:      100,
		Remaining: 200,
		Type:      types.Order_GTC,
	}
	err = book.AmendOrder(editedOrder)
	if err != types.OrderError_INVALID_REMAINING_SIZE {
		t.Log(err)
	}

	assert.Equal(t, types.OrderError_INVALID_REMAINING_SIZE, err)
}

func TestOrderBook_AmendOrderInvalidAmend(t *testing.T) {
	logger := logging.NewLoggerFromEnv("dev")
	defer logger.Sync()
	
	book := NewBook("testOrderBook", ProRataModeConfig(logger))
	newOrder := &types.Order{
		Market:    "testOrderBook",
		Id:        "123456",
		Side:      types.Side_Buy,
		Price:     100,
		Size:      200,
		Remaining: 200,
		Type:      types.Order_GTC,
	}

	confirmation, err := book.AddOrder(newOrder)
	if err != types.OrderError_NONE {
		fmt.Println(err)
	}

	fmt.Printf("confirmation : %+v", confirmation)

	editedOrder := &types.Order{
		Market:    "testOrderBook",
		Id:        "123456",
		Side:      types.Side_Sell,
		Price:     100,
		Size:      200,
		Remaining: 200,
		Type:      types.Order_GTC,
	}

	err = book.AmendOrder(editedOrder)
	if err != types.OrderError_ORDER_NOT_FOUND {
		fmt.Println(err)
	}

	assert.Equal(t, types.OrderError_ORDER_NOT_FOUND, err)
}

func TestOrderBook_AmendOrderInvalidAmend1(t *testing.T) {
	logger := logging.NewLoggerFromEnv("dev")
	defer logger.Sync()

	logger.Debug("BEGIN AMENDING ORDER")
	
	book := NewBook("testOrderBook", ProRataModeConfig(logger))
	newOrder := &types.Order{
		Market:    "testOrderBook",
		Id:        "123456",
		Side:      types.Side_Buy,
		Price:     100,
		Party:     "A",
		Size:      200,
		Remaining: 200,
		Type:      types.Order_GTC,
	}

	confirmation, err := book.AddOrder(newOrder)
	if err != types.OrderError_NONE {
		t.Log(err)
	}

	assert.Equal(t, types.OrderError_NONE, err)
	assert.NotNil(t, confirmation)
	assert.Equal(t, "123456", confirmation.Order.Id)
	assert.Equal(t, 0, len(confirmation.Trades))

	editedOrder := &types.Order{
		Market:    "testOrderBook",
		Id:        "123456",
		Side:      types.Side_Buy,
		Price:     100,
		Party:     "B",
		Size:      200,
		Remaining: 200,
		Type:      types.Order_GTC,
	}

	err = book.AmendOrder(editedOrder)
	if err != types.OrderError_ORDER_AMEND_FAILURE {
		t.Log(err)
	}

	assert.Equal(t, types.OrderError_ORDER_AMEND_FAILURE, err)
}

func TestOrderBook_AmendOrderInvalidAmendOutOfSequence(t *testing.T) {
	logger := logging.NewLoggerFromEnv("dev")
	defer logger.Sync()

	logger.Debug("BEGIN AMENDING OUT OF SEQUENCE ORDER")

	book := NewBook("testOrderBook", ProRataModeConfig(logger))
	newOrder := &types.Order{
		Market:    "testOrderBook",
		Id:        "123456",
		Side:      types.Side_Buy,
		Price:     100,
		Party:     "A",
		Size:      200,
		Remaining: 200,
		Type:      types.Order_GTC,
		Timestamp: 10,
	}

	confirmation, err := book.AddOrder(newOrder)
	if err != types.OrderError_NONE {
		fmt.Println(err)
	}

	assert.Equal(t, types.OrderError_NONE, err)
	assert.NotNil(t, confirmation)
	assert.Equal(t, "123456", confirmation.Order.Id)
	assert.Equal(t, 0, len(confirmation.Trades))

	editedOrder := &types.Order{
		Market:    "testOrderBook",
		Id:        "123456",
		Side:      types.Side_Buy,
		Price:     100,
		Party:     "A",
		Size:      200,
		Remaining: 200,
		Type:      types.Order_GTC,
		Timestamp: 5,
	}

	err = book.AmendOrder(editedOrder)
	if err != types.OrderError_ORDER_OUT_OF_SEQUENCE {
		t.Log(err)
	}

	assert.Equal(t, types.OrderError_ORDER_OUT_OF_SEQUENCE, err)
}

func TestOrderBook_AmendOrderInvalidAmendSize(t *testing.T) {
	logger := logging.NewLoggerFromEnv("dev")
	defer logger.Sync()

	logger.Debug("BEGIN AMEND ORDER INVALID SIZE")

	book := NewBook("testOrderBook", ProRataModeConfig(logger))
	newOrder := &types.Order{
		Market:    "testOrderBook",
		Id:        "123456",
		Side:      types.Side_Buy,
		Price:     100,
		Party:     "A",
		Size:      200,
		Remaining: 200,
		Type:      types.Order_GTC,
		Timestamp: 10,
	}

	confirmation, err := book.AddOrder(newOrder)
	if err != types.OrderError_NONE {
		t.Log(err)
	}

	assert.Equal(t, types.OrderError_NONE, err)
	assert.NotNil(t, confirmation)
	assert.Equal(t, "123456", confirmation.Order.Id)
	assert.Equal(t, 0, len(confirmation.Trades))

	editedOrder := &types.Order{
		Market:    "testOrderBook",
		Id:        "123456",
		Side:      types.Side_Buy,
		Price:     100,
		Party:     "B",
		Size:      300,
		Remaining: 300,
		Type:      types.Order_GTC,
		Timestamp: 10,
	}

	err = book.AmendOrder(editedOrder)
	if err != types.OrderError_ORDER_AMEND_FAILURE {
		t.Log(err)
	}

	assert.Equal(t, types.OrderError_ORDER_AMEND_FAILURE, err)
}

// ProRata mode OFF which is a default config for vega ME
func TestOrderBook_AddOrderProRataModeOff(t *testing.T) {
	logger := logging.NewLoggerFromEnv("dev")
	defer logger.Sync()

	logger.Debug("BEGIN PRO-RATA MODE OFF")

	book := NewBook("testOrderBook", NewConfig(logger))

	const numberOfTimestamps = 2
	m := make(map[int64][]*types.Order, numberOfTimestamps)

	// sell and buy side orders at timestamp 0
	m[0] = []*types.Order{
		// Side Sell
		{
			Market:    "testOrderBook",
			Party:     "A",
			Side:      types.Side_Sell,
			Price:     101,
			Size:      100,
			Remaining: 100,
			Type:      types.Order_GTC,
			Timestamp: 0,
		},
		{
			Market:    "testOrderBook",
			Party:     "B",
			Side:      types.Side_Sell,
			Price:     101,
			Size:      100,
			Remaining: 100,
			Type:      types.Order_GTC,
			Timestamp: 0,
		},
		// Side Buy
		{
			Market:    "testOrderBook",
			Party:     "C",
			Side:      types.Side_Buy,
			Price:     98,
			Size:      100,
			Remaining: 100,
			Type:      types.Order_GTC,
			Timestamp: 0,
		},
		{
			Market:    "testOrderBook",
			Party:     "D",
			Side:      types.Side_Buy,
			Price:     98,
			Size:      100,
			Remaining: 100,
			Type:      types.Order_GTC,
			Timestamp: 0,
		},
	}

	// sell and buy orders at timestamp 1
	m[1] = []*types.Order{
		// Side Sell
		{
			Market:    "testOrderBook",
			Party:     "E",
			Side:      types.Side_Sell,
			Price:     101,
			Size:      100,
			Remaining: 100,
			Type:      types.Order_GTC,
			Timestamp: 1,
		},
		// Side Buy
		{
			Market:    "testOrderBook",
			Party:     "F",
			Side:      types.Side_Buy,
			Price:     99,
			Size:      100,
			Remaining: 100,
			Type:      types.Order_GTC,
			Timestamp: 1,
		},
	}

	timestamps := []int64{0, 1}
	for _, timestamp := range timestamps {
		for index, _ := range m[timestamp] {
			fmt.Println("tests calling book.AddOrder: ", m[timestamp][index])
			confirmationtypes, err := book.AddOrder(m[timestamp][index])
			// this should not return any errors
			assert.Equal(t, types.OrderError_NONE, err)
			// this should not generate any trades
			assert.Equal(t, 0, len(confirmationtypes.Trades))
		}
	}

	scenario := []aggressiveOrderScenario{
		{
			// same price level, remaining on the passive
			aggressiveOrder: &types.Order{
				Market:    "testOrderBook",
				Party:     "M",
				Side:      types.Side_Buy,
				Price:     101,
				Size:      100,
				Remaining: 100,
				Type:      types.Order_GTC,
				Timestamp: 3,
			},
			expectedTrades: []types.Trade{
				{
					Market:    "testOrderBook",
					Price:     101,
					Size:      100,
					Buyer:     "M",
					Seller:    "A",
					Aggressor: types.Side_Buy,
				},
			},
			expectedPassiveOrdersAffected: []types.Order{
				{
					Market:    "testOrderBook",
					Party:     "A",
					Side:      types.Side_Sell,
					Price:     101,
					Size:      100,
					Remaining: 0,
					Type:      types.Order_GTC,
					Timestamp: 0,
				},
			},
		},
		{
			// same price level, remaining on the passive
			aggressiveOrder: &types.Order{
				Market:    "testOrderBook",
				Party:     "N",
				Side:      types.Side_Buy,
				Price:     102,
				Size:      200,
				Remaining: 200,
				Type:      types.Order_GTC,
				Timestamp: 4,
			},
			expectedTrades: []types.Trade{
				{
					Market:    "testOrderBook",
					Price:     101,
					Size:      100,
					Buyer:     "N",
					Seller:    "B",
					Aggressor: types.Side_Buy,
				},
				{
					Market:    "testOrderBook",
					Price:     101,
					Size:      100,
					Buyer:     "N",
					Seller:    "E",
					Aggressor: types.Side_Buy,
				},
			},
			expectedPassiveOrdersAffected: []types.Order{
				{
					Market:    "testOrderBook",
					Party:     "B",
					Side:      types.Side_Sell,
					Price:     101,
					Size:      100,
					Remaining: 0,
					Type:      types.Order_GTC,
					Timestamp: 0,
				},
				{
					Market:    "testOrderBook",
					Party:     "E",
					Side:      types.Side_Sell,
					Price:     101,
					Size:      100,
					Remaining: 0,
					Type:      types.Order_GTC,
					Timestamp: 1,
				},
			},
		},
		{
			// same price level, NO PRORATA
			aggressiveOrder: &types.Order{
				Market:    "testOrderBook",
				Party:     "O",
				Side:      types.Side_Sell,
				Price:     97,
				Size:      250,
				Remaining: 250,
				Type:      types.Order_GTC,
				Timestamp: 5,
			},
			expectedTrades: []types.Trade{
				{
					Market:    "testOrderBook",
					Price:     99,
					Size:      100,
					Buyer:     "F",
					Seller:    "O",
					Aggressor: types.Side_Sell,
				},
				{
					Market:    "testOrderBook",
					Price:     98,
					Size:      100,
					Buyer:     "C",
					Seller:    "O",
					Aggressor: types.Side_Sell,
				},
				{
					Market:    "testOrderBook",
					Price:     98,
					Size:      50,
					Buyer:     "D",
					Seller:    "O",
					Aggressor: types.Side_Sell,
				},
			},
			expectedPassiveOrdersAffected: []types.Order{
				{
					Market:    "testOrderBook",
					Party:     "F",
					Side:      types.Side_Buy,
					Price:     99,
					Size:      100,
					Remaining: 0,
					Type:      types.Order_GTC,
					Timestamp: 1,
				},
				{
					Market:    "testOrderBook",
					Party:     "C",
					Side:      types.Side_Buy,
					Price:     98,
					Size:      100,
					Remaining: 0,
					Type:      types.Order_GTC,
					Timestamp: 0,
				},
				{
					Market:    "testOrderBook",
					Party:     "D",
					Side:      types.Side_Buy,
					Price:     98,
					Size:      100,
					Remaining: 50,
					Type:      types.Order_GTC,
					Timestamp: 0,
				},
			},
		},
		{
			// same price level, NO PRORATA
			aggressiveOrder: &types.Order{
				Market:    "testOrderBook",
				Party:     "X",
				Side:      types.Side_Sell,
				Price:     98,
				Size:      50,
				Remaining: 50,
				Type:      types.Order_GTC,
				Timestamp: 6,
			},
			expectedTrades: []types.Trade{
				{
					Market:    "testOrderBook",
					Price:     98,
					Size:      50,
					Buyer:     "D",
					Seller:    "X",
					Aggressor: types.Side_Sell,
				},
			},
			expectedPassiveOrdersAffected: []types.Order{
				{
					Market:    "testOrderBook",
					Party:     "D",
					Side:      types.Side_Buy,
					Price:     98,
					Size:      100,
					Remaining: 0,
					Type:      types.Order_GTC,
					Timestamp: 0,
				},
			},
		},
	}

	for i, s := range scenario {
		fmt.Println()
		fmt.Println()
		fmt.Printf("SCENARIO %d / %d ------------------------------------------------------------------", i+1, len(scenario))
		fmt.Println()
		fmt.Println("aggressor: ", s.aggressiveOrder)
		fmt.Println("expectedPassiveOrdersAffected: ", s.expectedPassiveOrdersAffected)
		fmt.Println("expectedTrades: ", s.expectedTrades)
		fmt.Println()

		confirmationtypes, err := book.AddOrder(s.aggressiveOrder)

		//this should not return any errors
		assert.Equal(t, types.OrderError_NONE, err)

		//this should not generate any trades
		assert.Equal(t, len(s.expectedTrades), len(confirmationtypes.Trades))

		fmt.Println("CONFIRMATION types:")
		fmt.Println("-> Aggresive:", confirmationtypes.Order)
		fmt.Println("-> Trades :", confirmationtypes.Trades)
		fmt.Println("-> PassiveOrdersAffected:", confirmationtypes.PassiveOrdersAffected)
		fmt.Printf("Scenario: %d / %d \n", i+1, len(scenario))

		// trades should match expected trades
		for i, trade := range confirmationtypes.Trades {
			expectTrade(t, &s.expectedTrades[i], trade)
		}

		// orders affected should match expected values
		for i, orderAffected := range confirmationtypes.PassiveOrdersAffected {
			expectOrder(t, &s.expectedPassiveOrdersAffected[i], orderAffected)
		}

		// call remove expired orders every scenario
		book.RemoveExpiredOrders(s.aggressiveOrder.Timestamp)
	}
}