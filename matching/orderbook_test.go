package matching

import (
	"log"
	"testing"

	"vega/proto"

	"github.com/stretchr/testify/assert"
)

// test for order validation
func TestOrderBook_AddOrder2WithValidation(t *testing.T) {
	book := NewBook("testOrderBook", make(map[string]*OrderEntry), DefaultConfig())
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
	book := NewBook("testOrderBook", make(map[string]*OrderEntry), DefaultConfig())

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
			Party:     "D",
			Side:      msg.Side_Sell,
			Price:     102,
			Size:      100,
			Remaining: 100,
			Type:      msg.Order_GTC,
			Timestamp: 0,
		},
		{
			Market:    "testOrderBook",
			Party:     "E",
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
			Party:     "M",
			Side:      msg.Side_Buy,
			Price:     99,
			Size:      100,
			Remaining: 100,
			Type:      msg.Order_GTC,
			Timestamp: 0,
		},
		{
			Market:    "testOrderBook",
			Party:     "N",
			Side:      msg.Side_Buy,
			Price:     99,
			Size:      100,
			Remaining: 100,
			Type:      msg.Order_GTC,
			Timestamp: 0,
		},
		{
			Market:    "testOrderBook",
			Party:     "P",
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
			Party:     "C",
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
			Party:     "O",
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
			Party:     "C",
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
			Party:     "O",
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
			confirmationMsg, err := book.AddOrder(&order)
			// this should not return any errors
			assert.Equal(t, msg.OrderError_NONE, err)
			// this should not generate any trades
			assert.Equal(t, 0, len(confirmationMsg.Trades))
		}
	}

	// make sure order book contains all the orders
	assert.Equal(t, 11, len(book.orders))

	// launch aggressive orders from both sides to fully clear the order book
	type aggressiveOrderScenario struct {
		order msg.Order
		expectedNumberOfTrades int64
	}

	aggressiveOrders := []aggressiveOrderScenario{
		{
			order: msg.Order{
				Market:    "testOrderBook",
				Party:     "X",
				Side:      msg.Side_Buy,
				Price:     101,
				Size:      100,
				Remaining: 100,
				Type:      msg.Order_GTC,
				Timestamp: 3,
			},
			expectedNumberOfTrades: 1,
		},
		{
			order: msg.Order{
				Market:    "testOrderBook",
				Party:     "Y",
				Side:      msg.Side_Buy,
				Price:     101,
				Size:      100,
				Remaining: 100,
				Type:      msg.Order_GTC,
				Timestamp: 3,
			},
			expectedNumberOfTrades: 1,
		},
	}

	log.Println(aggressiveOrders)

	//for _, order := range aggressiveOrders {
	//	confirmationMsg, err := book.AddOrder(&order.order)
	//	// this should not return any errors
	//	assert.Equal(t, msg.OrderError_NONE, err)
	//	// this should not generate any trades
	//	//assert.Equal(t, order.expectedNumberOfTrades, len(confirmationMsg.Trades))
	//
	//	log.Println("confirmationMsg :", confirmationMsg)
	//}

}


//Remarks
// can you cross your own order ?? is there a counter party check?