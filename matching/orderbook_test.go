package matching

import (
	"log"
	"testing"

	"vega/proto"

	"github.com/stretchr/testify/assert"
)

// test for order validation
func TestOrderBook_AddOrder2WithValidation(t *testing.T) {
	book := NewBook("testOrderBook", DefaultConfig())
	book.latestTimestamp = 10

	invalidTimestampOrderMsg := &msg.Order{
		Market:    "testOrderBook",
		Party:     "A",
		Side:      msg.Side_Sell,
		Price:     100,
		Size:      100,
		Remaining: 100,
		Type:      msg.Order_GTC,
		Timestamp: 0,
		Id:        "id-number-one",
	}
	_, err := book.AddOrder(invalidTimestampOrderMsg)
	assert.Equal(t, msg.OrderError_ORDER_OUT_OF_SEQUENCE, err)

	book.latestTimestamp = 0
	invalidRemainginSizeOrderMsg := &msg.Order{
		Market:    "testOrderBook",
		Party:     "A",
		Side:      msg.Side_Sell,
		Price:     100,
		Size:      100,
		Remaining: 300,
		Type:      msg.Order_GTC,
		Timestamp: 0,
		Id:        "id-number-one",
	}
	_, err = book.AddOrder(invalidRemainginSizeOrderMsg)
	assert.Equal(t, msg.OrderError_INVALID_REMAINING_SIZE, err)

	invalidIdOrderMsg := &msg.Order{
		Market:    "testOrderBook",
		Party:     "A",
		Side:      msg.Side_Sell,
		Price:     100,
		Size:      100,
		Remaining: 100,
		Type:      msg.Order_GTC,
		Timestamp: 0,
		Id:        "foobar",
	}
	_, err = book.AddOrder(invalidIdOrderMsg)
	assert.Equal(t, msg.OrderError_NON_EMPTY_NEW_ORDER_ID, err)
}

func TestOrderBook_AddOrder(t *testing.T) {
	book := NewBook("testOrderBook", DefaultConfig())

	const numberOfTimestamps = 3
	m := make(map[int64][]msg.Order, numberOfTimestamps)

	// sell and buy side orders at timestamp 0
	m[0] = []msg.Order{
		// Side Sell
		{
			Market:    "testOrderBook",
			Party:     "A",
			Side:      msg.Side_Sell,
			Price:     101,
			Size:      100,
			Remaining: 100,
			Type:      msg.Order_GTC,
			Timestamp: 0,
		},
		{
			Market:    "testOrderBook",
			Party:     "B",
			Side:      msg.Side_Sell,
			Price:     101,
			Size:      100,
			Remaining: 100,
			Type:      msg.Order_GTC,
			Timestamp: 0,
		},
		{
			Market:    "testOrderBook",
			Party:     "C",
			Side:      msg.Side_Sell,
			Price:     102,
			Size:      100,
			Remaining: 100,
			Type:      msg.Order_GTC,
			Timestamp: 0,
		},
		{
			Market:    "testOrderBook",
			Party:     "D",
			Side:      msg.Side_Sell,
			Price:     103,
			Size:      100,
			Remaining: 100,
			Type:      msg.Order_GTC,
			Timestamp: 0,
		},
		// Side Buy
		{
			Market:    "testOrderBook",
			Party:     "E",
			Side:      msg.Side_Buy,
			Price:     99,
			Size:      100,
			Remaining: 100,
			Type:      msg.Order_GTC,
			Timestamp: 0,
		},
		{
			Market:    "testOrderBook",
			Party:     "F",
			Side:      msg.Side_Buy,
			Price:     99,
			Size:      100,
			Remaining: 100,
			Type:      msg.Order_GTC,
			Timestamp: 0,
		},
		{
			Market:    "testOrderBook",
			Party:     "G",
			Side:      msg.Side_Buy,
			Price:     98,
			Size:      100,
			Remaining: 100,
			Type:      msg.Order_GTC,
			Timestamp: 0,
		},
	}

	// sell and buy orders at timestamp 1
	m[1] = []msg.Order{
		// Side Sell
		{
			Market:    "testOrderBook",
			Party:     "M",
			Side:      msg.Side_Sell,
			Price:     101,
			Size:      100,
			Remaining: 100,
			Type:      msg.Order_GTC,
			Timestamp: 1,
		},
		// Side Buy
		{
			Market:    "testOrderBook",
			Party:     "N",
			Side:      msg.Side_Buy,
			Price:     99,
			Size:      100,
			Remaining: 100,
			Type:      msg.Order_GTC,
			Timestamp: 1,
		},
	}

	// sell and buy orders at timestamp 2
	m[2] = []msg.Order{
		// Side Sell
		{
			Market:    "testOrderBook",
			Party:     "R",
			Side:      msg.Side_Sell,
			Price:     101,
			Size:      100,
			Remaining: 100,
			Type:      msg.Order_GTC,
			Timestamp: 2,
		},
		// Side Buy
		{
			Market:    "testOrderBook",
			Party:     "S",
			Side:      msg.Side_Buy,
			Price:     99,
			Size:      100,
			Remaining: 100,
			Type:      msg.Order_GTC,
			Timestamp: 2,
		},
	}

	for _, v := range m {
		for _, order := range v {
			log.Println("tests calling book.AddOrder: ", order)
			confirmationMsg, err := book.AddOrder(&order)
			// this should not return any errors
			assert.Equal(t, msg.OrderError_NONE, err)
			// this should not generate any trades
			assert.Equal(t, 0, len(confirmationMsg.Trades))
		}
	}

	// launch aggressiveOrder orders from both sides to fully clear the order book
	type aggressiveOrderScenario struct {
		aggressiveOrder msg.Order
		expectedTrades  []msg.Trade
	}

	scenario := []aggressiveOrderScenario{
		{
			// same price level, remaining on the passive
			aggressiveOrder: msg.Order{
				Market:    "testOrderBook",
				Party:     "X",
				Side:      msg.Side_Buy,
				Price:     101,
				Size:      100,
				Remaining: 100,
				Type:      msg.Order_GTC,
				Timestamp: 3,
			},
			expectedTrades: []msg.Trade{
				{
					Market:    "testOrderBook",
					Price:     101,
					Size:      50,
					Buyer:     "X",
					Seller:    "A",
					Aggressor: msg.Side_Buy,
				},
				{
					Market:    "testOrderBook",
					Price:     101,
					Size:      50,
					Buyer:     "X",
					Seller:    "B",
					Aggressor: msg.Side_Buy,
				},
			},
		},
		{
			// lower price is available on the passive side, 2 orders removed, 1 passive remaining
			aggressiveOrder: msg.Order{
				Market:    "testOrderBook",
				Party:     "Y",
				Side:      msg.Side_Buy,
				Price:     102,
				Size:      150,
				Remaining: 150,
				Type:      msg.Order_GTC,
				Timestamp: 3,
			},
			expectedTrades: []msg.Trade{
				{
					Market:    "testOrderBook",
					Price:     101,
					Size:      50,
					Buyer:     "Y",
					Seller:    "A",
					Aggressor: msg.Side_Buy,
				},
				{
					Market:    "testOrderBook",
					Price:     101,
					Size:      50,
					Buyer:     "Y",
					Seller:    "B",
					Aggressor: msg.Side_Buy,
				},
				{
					Market:    "testOrderBook",
					Price:     101,
					Size:      50,
					Buyer:     "Y",
					Seller:    "M",
					Aggressor: msg.Side_Buy,
				},
			},
		},
		{
			// lower price is available on the passive side, 1 order removed, 1 passive remaining
			aggressiveOrder: msg.Order{
				Market:    "testOrderBook",
				Party:     "Z",
				Side:      msg.Side_Buy,
				Price:     102,
				Size:      70,
				Remaining: 70,
				Type:      msg.Order_GTC,
				Timestamp: 3,
			},
			expectedTrades: []msg.Trade{
				{
					Market:    "testOrderBook",
					Price:     101,
					Size:      50,
					Buyer:     "Z",
					Seller:    "M",
					Aggressor: msg.Side_Buy,
				},
				{
					Market:    "testOrderBook",
					Price:     101,
					Size:      20,
					Buyer:     "Z",
					Seller:    "R",
					Aggressor: msg.Side_Buy,
				},
			},
		},
		{
			// price level jump, lower price is available on the passive side but its entirely consumed,
			// 1 order removed, 1 passive remaining at higher price level
			aggressiveOrder: msg.Order{
				Market:    "testOrderBook",
				Party:     "X",
				Side:      msg.Side_Buy,
				Price:     102,
				Size:      100,
				Remaining: 100,
				Type:      msg.Order_GTC,
				Timestamp: 3,
			},
			expectedTrades: []msg.Trade{
				{
					Market:    "testOrderBook",
					Price:     101,
					Size:      80,
					Buyer:     "X",
					Seller:    "R",
					Aggressor: msg.Side_Buy,
				},
				{
					Market:    "testOrderBook",
					Price:     102,
					Size:      20,
					Buyer:     "X",
					Seller:    "C",
					Aggressor: msg.Side_Buy,
				},
			},
		},
		{
			// Sell is agressive, aggressive at lower price than on the book, pro rata at 99, aggressive is removed
			aggressiveOrder: msg.Order{
				Market:    "testOrderBook",
				Party:     "Y",
				Side:      msg.Side_Sell,
				Price:     98,
				Size:      100,
				Remaining: 100,
				Type:      msg.Order_GTC,
				Timestamp: 4,
			},
			expectedTrades: []msg.Trade{
				{
					Market:    "testOrderBook",
					Price:     99,
					Size:      50,
					Buyer:     "E",
					Seller:    "Y",
					Aggressor: msg.Side_Sell,
				},
				{
					Market:    "testOrderBook",
					Price:     99,
					Size:      50,
					Buyer:     "F",
					Seller:    "Y",
					Aggressor: msg.Side_Sell,
				},
			},
		},
		{
			// Sell is agressive, aggressive at exact price, all orders at this price level should be hitted plus order should remain on the sell side of the book at 99 level
			aggressiveOrder: msg.Order{
				Market:    "testOrderBook",
				Party:     "Z",
				Side:      msg.Side_Sell,
				Price:     99,
				Size:      350,
				Remaining: 350,
				Type:      msg.Order_GTC,
				Timestamp: 4,
			},
			expectedTrades: []msg.Trade{
				{
					Market:    "testOrderBook",
					Price:     99,
					Size:      50,
					Buyer:     "E",
					Seller:    "Z",
					Aggressor: msg.Side_Sell,
				},
				{
					Market:    "testOrderBook",
					Price:     99,
					Size:      50,
					Buyer:     "F",
					Seller:    "Z",
					Aggressor: msg.Side_Sell,
				},
				{
					Market:    "testOrderBook",
					Price:     99,
					Size:      100,
					Buyer:     "N",
					Seller:    "Z",
					Aggressor: msg.Side_Sell,
				},
				{
					Market:    "testOrderBook",
					Price:     99,
					Size:      100,
					Buyer:     "S",
					Seller:    "Z",
					Aggressor: msg.Side_Sell,
				},
			},
		},
		{ // aggressive nonpersistent buy order, hits two price levels and is not added to order book
			aggressiveOrder: msg.Order{
				Market:    "testOrderBook",
				Party:     "XX",
				Side:      msg.Side_Buy,
				Price:     102,
				Size:      200,
				Remaining: 200,
				Type:      msg.Order_FOK, // nonpersistent
				Timestamp: 4,
			},
			expectedTrades: []msg.Trade{
				{
					Market:    "testOrderBook",
					Price:     99,
					Size:      50,
					Buyer:     "XX",
					Seller:    "Z",
					Aggressor: msg.Side_Buy,
				},
				{
					Market:    "testOrderBook",
					Price:     102,
					Size:      80,
					Buyer:     "XX",
					Seller:    "C",
					Aggressor: msg.Side_Buy,
				},
			},
		},
		{ // aggressive nonpersistent buy order, hits one price levels and is not added to order book
			aggressiveOrder: msg.Order{
				Market:    "testOrderBook",
				Party:     "YY",
				Side:      msg.Side_Buy,
				Price:     103,
				Size:      200,
				Remaining: 200,
				Type:      msg.Order_ENE, // nonpersistent
				Timestamp: 5,
			},
			expectedTrades: []msg.Trade{
				{
					Market:    "testOrderBook",
					Price:     103,
					Size:      100,
					Buyer:     "YY",
					Seller:    "D",
					Aggressor: msg.Side_Buy,
				},
			},
		},
		{ // aggressive nonpersistent buy order, at super low price hits one price levels and is not added to order book
			aggressiveOrder: msg.Order{
				Market:    "testOrderBook",
				Party:     "ZZ",
				Side:      msg.Side_Sell,
				Price:     95,
				Size:      200,
				Remaining: 200,
				Type:      msg.Order_ENE, // nonpersistent
				Timestamp: 5,
			},
			expectedTrades: []msg.Trade{
				{
					Market:    "testOrderBook",
					Price:     98,
					Size:      100,
					Buyer:     "G",
					Seller:    "ZZ",
					Aggressor: msg.Side_Sell,
				},
			},
		},
	}

	//log.Println(scenario)

	for i, s := range scenario {
		log.Println()
		log.Println()
		log.Printf("SCENARIO %d / %d ------------------------------------------------------------------", i+1, len(scenario))
		log.Println()
		log.Println("aggressor: ", s.aggressiveOrder)
		log.Println("expectedTrades: ", s.expectedTrades)
		log.Println()

		//printStateOfBook(book)

		confirmationMsg, err := book.AddOrder(&s.aggressiveOrder)
		//this should not return any errors
		assert.Equal(t, msg.OrderError_NONE, err)
		//this should not generate any trades
		assert.Equal(t, len(s.expectedTrades), len(confirmationMsg.Trades))

		log.Println("confirmationMsg :", confirmationMsg)

		// trades should match expected trades
		// assert.
		for i, trade := range confirmationMsg.Trades {
			expectTrade(t, &s.expectedTrades[i], trade)
		}
	}

}

//Remarks
// can you cross your own order ?? is there a counter party check?