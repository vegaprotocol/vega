package matching_test

import (
	"fmt"
	"testing"

	"code.vegaprotocol.io/vega/internal/logging"
	"code.vegaprotocol.io/vega/internal/matching"
	"code.vegaprotocol.io/vega/internal/vegatime"
	types "code.vegaprotocol.io/vega/proto"

	"github.com/stretchr/testify/assert"
)

const (
	markPrice = 10
)

// launch aggressiveOrder orders from both sides to fully clear the order book
type aggressiveOrderScenario struct {
	aggressiveOrder               *types.Order
	expectedPassiveOrdersAffected []types.Order
	expectedTrades                []types.Trade
}

type tstOB struct {
	*matching.OrderBook
	log *logging.Logger
}

func (t *tstOB) Finish() {
	t.log.Sync()
}

func getCurrentUtcTimestampNano() int64 {
	return vegatime.Now().UnixNano()
}

func getTestOrderBook(t *testing.T, market string, proRata bool) *tstOB {
	tob := tstOB{
		log: logging.NewTestLogger(),
	}
	tob.OrderBook = matching.NewOrderBook(tob.log, matching.NewDefaultConfig(), market, markPrice, proRata)
	return &tob
}

func TestOrderBook_GetClosePNL(t *testing.T) {
	t.Run("Get Buy-side close PNL values", getClosePNLBuy)
	t.Run("Get Sell-side close PNL values", getClosePNLSell)
	t.Run("Get Incomplete close-out-pnl (check error) - Buy", getClosePNLIncompleteBuy)
	t.Run("Get Incomplete close-out-pnl (check error) - Sell", getClosePNLIncompleteSell)
}

func getClosePNLIncompleteBuy(t *testing.T) {
	market := "testMarket"
	book := getTestOrderBook(t, market, true)
	defer book.Finish()
	// 3 orders of size 1, 3 different prices
	orders := []*types.Order{
		{
			MarketID:    market,
			PartyID:     "A",
			Side:        types.Side_Buy,
			Price:       100,
			Size:        1,
			Remaining:   1,
			TimeInForce: types.Order_GTC,
			CreatedAt:   0,
		},
		{
			MarketID:    market,
			PartyID:     "B",
			Side:        types.Side_Buy,
			Price:       110,
			Size:        1,
			Remaining:   1,
			TimeInForce: types.Order_GTC,
			CreatedAt:   0,
		},
	}
	for _, o := range orders {
		confirm, err := book.SubmitOrder(o)
		assert.NoError(t, err)
		assert.Equal(t, 0, len(confirm.Trades))
	}
	// volume + expected price
	callExp := map[uint64]uint64{
		2: 210 / 2,
		1: 100,
	}
	// this calculates the actual volume
	for vol, exp := range callExp {
		price, err := book.GetCloseoutPrice(vol, types.Side_Buy)
		assert.Equal(t, exp, price)
		assert.NoError(t, err)
	}
	price, err := book.GetCloseoutPrice(3, types.Side_Buy)
	assert.Equal(t, callExp[2], price)
	assert.Equal(t, matching.ErrNotEnoughOrders, err)
}

func getClosePNLIncompleteSell(t *testing.T) {
	market := "testMarket"
	book := getTestOrderBook(t, market, true)
	defer book.Finish()
	// 3 orders of size 1, 3 different prices
	orders := []*types.Order{
		{
			MarketID:    market,
			PartyID:     "A",
			Side:        types.Side_Sell,
			Price:       100,
			Size:        1,
			Remaining:   1,
			TimeInForce: types.Order_GTC,
			CreatedAt:   0,
		},
		{
			MarketID:    market,
			PartyID:     "B",
			Side:        types.Side_Sell,
			Price:       110,
			Size:        1,
			Remaining:   1,
			TimeInForce: types.Order_GTC,
			CreatedAt:   0,
		},
	}
	for _, o := range orders {
		confirm, err := book.SubmitOrder(o)
		assert.NoError(t, err)
		assert.Equal(t, 0, len(confirm.Trades))
	}
	// volume + expected price
	callExp := map[uint64]uint64{
		2: 210 / 2,
		1: 100,
	}
	// this calculates the actual volume
	for vol, exp := range callExp {
		price, err := book.GetCloseoutPrice(vol, types.Side_Sell)
		assert.Equal(t, exp, price)
		assert.NoError(t, err)
	}
	price, err := book.GetCloseoutPrice(3, types.Side_Sell)
	assert.Equal(t, callExp[2], price)
	assert.Equal(t, matching.ErrNotEnoughOrders, err)
}

func getClosePNLBuy(t *testing.T) {
	market := "testMarket"
	book := getTestOrderBook(t, market, true)
	defer book.Finish()
	// 3 orders of size 1, 3 different prices
	orders := []*types.Order{
		{
			MarketID:    market,
			PartyID:     "A",
			Side:        types.Side_Buy,
			Price:       100,
			Size:        1,
			Remaining:   1,
			TimeInForce: types.Order_GTC,
			CreatedAt:   0,
		},
		{
			MarketID:    market,
			PartyID:     "B",
			Side:        types.Side_Buy,
			Price:       110,
			Size:        1,
			Remaining:   1,
			TimeInForce: types.Order_GTC,
			CreatedAt:   0,
		},
		{
			MarketID:    market,
			PartyID:     "C",
			Side:        types.Side_Buy,
			Price:       120,
			Size:        1,
			Remaining:   1,
			TimeInForce: types.Order_GTC,
			CreatedAt:   0,
		},
	}
	for _, o := range orders {
		confirm, err := book.SubmitOrder(o)
		assert.NoError(t, err)
		assert.Equal(t, 0, len(confirm.Trades))
	}
	// volume + expected price
	callExp := map[uint64]uint64{
		3: 330 / 3,
		2: 210 / 2,
		1: 100,
	}
	// this calculates the actual volume
	for vol, exp := range callExp {
		price, err := book.GetCloseoutPrice(vol, types.Side_Buy)
		assert.Equal(t, exp, price)
		assert.NoError(t, err)
	}
}

func getClosePNLSell(t *testing.T) {
	market := "testMarket"
	book := getTestOrderBook(t, market, true)
	defer book.Finish()
	// 3 orders of size 1, 3 different prices
	orders := []*types.Order{
		{
			MarketID:    market,
			PartyID:     "A",
			Side:        types.Side_Sell,
			Price:       100,
			Size:        1,
			Remaining:   1,
			TimeInForce: types.Order_GTC,
			CreatedAt:   0,
		},
		{
			MarketID:    market,
			PartyID:     "B",
			Side:        types.Side_Sell,
			Price:       110,
			Size:        1,
			Remaining:   1,
			TimeInForce: types.Order_GTC,
			CreatedAt:   0,
		},
		{
			MarketID:    market,
			PartyID:     "C",
			Side:        types.Side_Sell,
			Price:       120,
			Size:        1,
			Remaining:   1,
			TimeInForce: types.Order_GTC,
			CreatedAt:   0,
		},
	}
	for _, o := range orders {
		confirm, err := book.SubmitOrder(o)
		assert.NoError(t, err)
		assert.Equal(t, 0, len(confirm.Trades))
	}
	// volume + expected price
	callExp := map[uint64]uint64{
		3: 330 / 3,
		2: 210 / 2,
		1: 100,
	}
	// this calculates the actual volume
	for vol, exp := range callExp {
		price, err := book.GetCloseoutPrice(vol, types.Side_Sell)
		assert.NoError(t, err)
		assert.Equal(t, exp, price)
	}
}

func TestOrderBook_RemoveExpiredOrders(t *testing.T) {
	market := "expiringOrderBookTest"
	party := "clay-davis"

	book := getTestOrderBook(t, market, true)
	defer book.Finish()
	currentTimestamp := getCurrentUtcTimestampNano()
	someTimeLater := currentTimestamp + (1000 * 1000)

	order1 := &types.Order{
		MarketID:    market,
		PartyID:     party,
		Side:        types.Side_Sell,
		Price:       1,
		Size:        1,
		Remaining:   1,
		TimeInForce: types.Order_GTT,
		CreatedAt:   currentTimestamp,
		ExpiresAt:   someTimeLater,
		Id:          "1",
	}
	_, err := book.SubmitOrder(order1)
	assert.Equal(t, err, nil)

	order2 := &types.Order{
		MarketID:    market,
		PartyID:     party,
		Side:        types.Side_Sell,
		Price:       3298,
		Size:        99,
		Remaining:   99,
		TimeInForce: types.Order_GTT,
		CreatedAt:   currentTimestamp,
		ExpiresAt:   someTimeLater + 1,
		Id:          "2",
	}
	_, err = book.SubmitOrder(order2)
	assert.Equal(t, err, nil)

	order3 := &types.Order{
		MarketID:    market,
		PartyID:     party,
		Side:        types.Side_Sell,
		Price:       771,
		Size:        19,
		Remaining:   19,
		TimeInForce: types.Order_GTT,
		CreatedAt:   currentTimestamp,
		ExpiresAt:   someTimeLater,
		Id:          "3",
	}
	_, err = book.SubmitOrder(order3)
	assert.Equal(t, err, nil)

	order4 := &types.Order{
		MarketID:    market,
		PartyID:     party,
		Side:        types.Side_Sell,
		Price:       1000,
		Size:        7,
		Remaining:   7,
		TimeInForce: types.Order_GTC,
		CreatedAt:   currentTimestamp,
		Id:          "4",
	}
	_, err = book.SubmitOrder(order4)
	assert.Equal(t, err, nil)

	order5 := &types.Order{
		MarketID:    market,
		PartyID:     party,
		Side:        types.Side_Sell,
		Price:       199,
		Size:        99999,
		Remaining:   99999,
		TimeInForce: types.Order_GTT,
		CreatedAt:   currentTimestamp,
		ExpiresAt:   someTimeLater,
		Id:          "5",
	}
	_, err = book.SubmitOrder(order5)
	assert.Equal(t, err, nil)

	order6 := &types.Order{
		MarketID:    market,
		PartyID:     party,
		Side:        types.Side_Sell,
		Price:       100,
		Size:        100,
		Remaining:   100,
		TimeInForce: types.Order_GTC,
		CreatedAt:   currentTimestamp,
		Id:          "6",
	}
	_, err = book.SubmitOrder(order6)
	assert.Equal(t, err, nil)

	order7 := &types.Order{
		MarketID:    market,
		PartyID:     party,
		Side:        types.Side_Sell,
		Price:       41,
		Size:        9999,
		Remaining:   9999,
		TimeInForce: types.Order_GTT,
		CreatedAt:   currentTimestamp,
		ExpiresAt:   someTimeLater + 9999,
		Id:          "7",
	}
	_, err = book.SubmitOrder(order7)
	assert.Equal(t, err, nil)

	order8 := &types.Order{
		MarketID:    market,
		PartyID:     party,
		Side:        types.Side_Sell,
		Price:       1,
		Size:        1,
		Remaining:   1,
		TimeInForce: types.Order_GTT,
		CreatedAt:   currentTimestamp,
		ExpiresAt:   someTimeLater - 9999,
		Id:          "8",
	}
	_, err = book.SubmitOrder(order8)
	assert.Equal(t, err, nil)

	order9 := &types.Order{
		MarketID:    market,
		PartyID:     party,
		Side:        types.Side_Sell,
		Price:       65,
		Size:        12,
		Remaining:   12,
		TimeInForce: types.Order_GTC,
		CreatedAt:   currentTimestamp,
		Id:          "9",
	}
	_, err = book.SubmitOrder(order9)
	assert.Equal(t, err, nil)

	order10 := &types.Order{
		MarketID:    market,
		PartyID:     party,
		Side:        types.Side_Sell,
		Price:       1,
		Size:        1,
		Remaining:   1,
		TimeInForce: types.Order_GTT,
		CreatedAt:   currentTimestamp,
		ExpiresAt:   someTimeLater - 1,
		Id:          "10",
	}
	_, err = book.SubmitOrder(order10)
	assert.Equal(t, err, nil)

	expired := book.RemoveExpiredOrders(someTimeLater)
	assert.Len(t, expired, 5)
	assert.Equal(t, "8", expired[0].Id)
	assert.Equal(t, "10", expired[1].Id)
	assert.Equal(t, "5", expired[2].Id)
	assert.Equal(t, "3", expired[3].Id)
	assert.Equal(t, "1", expired[4].Id)
}

//test for order validation
func TestOrderBook_SubmitOrder2WithValidation(t *testing.T) {
	market := "testOrderbook"
	book := getTestOrderBook(t, market, true)
	defer book.Finish()
	timeStampOrder := types.Order{
		Id:        "timestamporderID",
		MarketID:  market,
		CreatedAt: 10,
		Side:      types.Side_Buy,
		Size:      1,
	}
	_, err := book.SubmitOrder(&timeStampOrder)
	assert.NoError(t, err)
	// cancel order again, just so we set the timestamp as expected
	book.CancelOrder(&timeStampOrder)

	invalidTimestampOrdertypes := &types.Order{
		MarketID:    market,
		PartyID:     "A",
		Side:        types.Side_Sell,
		Price:       100,
		Size:        100,
		Remaining:   100,
		TimeInForce: types.Order_GTC,
		CreatedAt:   0,
		Id:          "id-number-one",
	}
	_, err = book.SubmitOrder(invalidTimestampOrdertypes)
	assert.Equal(t, types.OrderError_ORDER_OUT_OF_SEQUENCE, err)

	invalidRemainginSizeOrdertypes := &types.Order{
		MarketID:    market,
		PartyID:     "A",
		Side:        types.Side_Sell,
		Price:       100,
		Size:        100,
		Remaining:   300,
		TimeInForce: types.Order_GTC,
		CreatedAt:   10,
		Id:          "id-number-one",
	}
	_, err = book.SubmitOrder(invalidRemainginSizeOrdertypes)
	assert.Equal(t, types.OrderError_INVALID_REMAINING_SIZE, err)
}

func TestOrderBook_DeleteOrder(t *testing.T) {
	market := "testOrderbook"
	book := getTestOrderBook(t, market, true)
	defer book.Finish()

	newOrder := &types.Order{
		MarketID:    market,
		PartyID:     "A",
		Side:        types.Side_Sell,
		Price:       101,
		Size:        100,
		Remaining:   100,
		TimeInForce: types.Order_GTC,
		CreatedAt:   0,
	}

	book.SubmitOrder(newOrder)

	err := book.DeleteOrder(newOrder)
	if err != nil {
		fmt.Println(err, "ORDER_NOT_FOUND")
	}

	book.PrintState("AFTER REMOVE ORDER")
}

func TestOrderBook_SubmitOrder(t *testing.T) {
	market := "testOrderbook"
	book := getTestOrderBook(t, market, true)
	defer book.Finish()

	const numberOfTimestamps = 3
	m := make(map[int64][]*types.Order, numberOfTimestamps)

	// sell and buy side orders at timestamp 0
	m[0] = []*types.Order{
		// Side Sell
		{
			MarketID:    market,
			PartyID:     "A",
			Side:        types.Side_Sell,
			Price:       101,
			Size:        100,
			Remaining:   100,
			TimeInForce: types.Order_GTC,
			CreatedAt:   0,
		},
		{
			MarketID:    market,
			PartyID:     "B",
			Side:        types.Side_Sell,
			Price:       101,
			Size:        100,
			Remaining:   100,
			TimeInForce: types.Order_GTC,
			CreatedAt:   0,
		},
		{
			MarketID:    market,
			PartyID:     "C",
			Side:        types.Side_Sell,
			Price:       102,
			Size:        100,
			Remaining:   100,
			TimeInForce: types.Order_GTC,
			CreatedAt:   0,
		},
		{
			MarketID:    market,
			PartyID:     "D",
			Side:        types.Side_Sell,
			Price:       103,
			Size:        100,
			Remaining:   100,
			TimeInForce: types.Order_GTC,
			CreatedAt:   0,
		},
		// Side Buy
		{
			MarketID:    market,
			PartyID:     "E",
			Side:        types.Side_Buy,
			Price:       99,
			Size:        100,
			Remaining:   100,
			TimeInForce: types.Order_GTC,
			CreatedAt:   0,
		},
		{
			MarketID:    market,
			PartyID:     "F",
			Side:        types.Side_Buy,
			Price:       99,
			Size:        100,
			Remaining:   100,
			TimeInForce: types.Order_GTC,
			CreatedAt:   0,
		},
		{
			MarketID:    market,
			PartyID:     "G",
			Side:        types.Side_Buy,
			Price:       98,
			Size:        100,
			Remaining:   100,
			TimeInForce: types.Order_GTC,
			CreatedAt:   0,
		},
	}

	// sell and buy orders at timestamp 1
	m[1] = []*types.Order{
		// Side Sell
		{
			MarketID:    market,
			PartyID:     "M",
			Side:        types.Side_Sell,
			Price:       101,
			Size:        100,
			Remaining:   100,
			TimeInForce: types.Order_GTC,
			CreatedAt:   1,
		},
		// Side Buy
		{
			MarketID:    market,
			PartyID:     "N",
			Side:        types.Side_Buy,
			Price:       99,
			Size:        100,
			Remaining:   100,
			TimeInForce: types.Order_GTC,
			CreatedAt:   1,
		},
	}

	// sell and buy orders at timestamp 2
	m[2] = []*types.Order{
		// Side Sell
		{
			MarketID:    market,
			PartyID:     "R",
			Side:        types.Side_Sell,
			Price:       101,
			Size:        100,
			Remaining:   100,
			TimeInForce: types.Order_GTC,
			CreatedAt:   2,
		},
		// Side Buy
		{
			MarketID:    market,
			PartyID:     "S",
			Side:        types.Side_Buy,
			Price:       99,
			Size:        100,
			Remaining:   100,
			TimeInForce: types.Order_GTC,
			CreatedAt:   2,
		},
	}

	timestamps := []int64{0, 1, 2}
	for _, timestamp := range timestamps {
		for index, _ := range m[timestamp] {
			fmt.Println("tests calling book.SubmitOrder: ", m[timestamp][index])
			confirmationtypes, err := book.SubmitOrder(m[timestamp][index])
			// this should not return any errors
			assert.Equal(t, nil, err)
			// this should not generate any trades
			assert.Equal(t, 0, len(confirmationtypes.Trades))
		}
	}

	scenario := []aggressiveOrderScenario{
		{
			// same price level, remaining on the passive
			aggressiveOrder: &types.Order{
				MarketID:    market,
				PartyID:     "X",
				Side:        types.Side_Buy,
				Price:       101,
				Size:        100,
				Remaining:   100,
				TimeInForce: types.Order_GTC,
				CreatedAt:   3,
			},
			expectedTrades: []types.Trade{
				{
					MarketID:  market,
					Price:     101,
					Size:      50,
					Buyer:     "X",
					Seller:    "A",
					Aggressor: types.Side_Buy,
				},
				{
					MarketID:  market,
					Price:     101,
					Size:      50,
					Buyer:     "X",
					Seller:    "B",
					Aggressor: types.Side_Buy,
				},
			},
			expectedPassiveOrdersAffected: []types.Order{
				{
					MarketID:    market,
					PartyID:     "A",
					Side:        types.Side_Sell,
					Price:       101,
					Size:        100,
					Remaining:   50,
					TimeInForce: types.Order_GTC,
					CreatedAt:   0,
				},
				{
					MarketID:    market,
					PartyID:     "B",
					Side:        types.Side_Sell,
					Price:       101,
					Size:        100,
					Remaining:   50,
					TimeInForce: types.Order_GTC,
					CreatedAt:   0,
				},
			},
		},
		{
			// lower price is available on the passive side, 2 orders removed, 1 passive remaining
			aggressiveOrder: &types.Order{
				MarketID:    market,
				PartyID:     "Y",
				Side:        types.Side_Buy,
				Price:       102,
				Size:        150,
				Remaining:   150,
				TimeInForce: types.Order_GTC,
				CreatedAt:   3,
			},
			expectedTrades: []types.Trade{
				{
					MarketID:  market,
					Price:     101,
					Size:      50,
					Buyer:     "Y",
					Seller:    "A",
					Aggressor: types.Side_Buy,
				},
				{
					MarketID:  market,
					Price:     101,
					Size:      50,
					Buyer:     "Y",
					Seller:    "B",
					Aggressor: types.Side_Buy,
				},
				{
					MarketID:  market,
					Price:     101,
					Size:      50,
					Buyer:     "Y",
					Seller:    "M",
					Aggressor: types.Side_Buy,
				},
			},
			expectedPassiveOrdersAffected: []types.Order{
				{
					MarketID:    market,
					PartyID:     "A",
					Side:        types.Side_Sell,
					Price:       101,
					Size:        100,
					Remaining:   0,
					TimeInForce: types.Order_GTC,
					CreatedAt:   0,
				},
				{
					MarketID:    market,
					PartyID:     "B",
					Side:        types.Side_Sell,
					Price:       101,
					Size:        100,
					Remaining:   0,
					TimeInForce: types.Order_GTC,
					CreatedAt:   0,
				},
				{
					MarketID:    market,
					PartyID:     "M",
					Side:        types.Side_Sell,
					Price:       101,
					Size:        100,
					Remaining:   50,
					TimeInForce: types.Order_GTC,
					CreatedAt:   1,
				},
			},
		},
		{
			// lower price is available on the passive side, 1 order removed, 1 passive remaining
			aggressiveOrder: &types.Order{
				MarketID:    market,
				PartyID:     "Z",
				Side:        types.Side_Buy,
				Price:       102,
				Size:        70,
				Remaining:   70,
				TimeInForce: types.Order_GTC,
				CreatedAt:   3,
			},
			expectedTrades: []types.Trade{
				{
					MarketID:  market,
					Price:     101,
					Size:      50,
					Buyer:     "Z",
					Seller:    "M",
					Aggressor: types.Side_Buy,
				},
				{
					MarketID:  market,
					Price:     101,
					Size:      20,
					Buyer:     "Z",
					Seller:    "R",
					Aggressor: types.Side_Buy,
				},
			},
			expectedPassiveOrdersAffected: []types.Order{
				{
					MarketID:    market,
					PartyID:     "M",
					Side:        types.Side_Sell,
					Price:       101,
					Size:        100,
					Remaining:   0,
					TimeInForce: types.Order_GTC,
					CreatedAt:   1,
				},
				{
					MarketID:    market,
					PartyID:     "R",
					Side:        types.Side_Sell,
					Price:       101,
					Size:        100,
					Remaining:   80,
					TimeInForce: types.Order_GTC,
					CreatedAt:   2,
				},
			},
		},
		{
			// price level jump, lower price is available on the passive side but its entirely consumed,
			// 1 order removed, 1 passive remaining at higher price level
			aggressiveOrder: &types.Order{
				MarketID:    market,
				PartyID:     "X",
				Side:        types.Side_Buy,
				Price:       102,
				Size:        100,
				Remaining:   100,
				TimeInForce: types.Order_GTC,
				CreatedAt:   3,
			},
			expectedTrades: []types.Trade{
				{
					MarketID:  market,
					Price:     101,
					Size:      80,
					Buyer:     "X",
					Seller:    "R",
					Aggressor: types.Side_Buy,
				},
				{
					MarketID:  market,
					Price:     102,
					Size:      20,
					Buyer:     "X",
					Seller:    "C",
					Aggressor: types.Side_Buy,
				},
			},
			expectedPassiveOrdersAffected: []types.Order{
				{
					MarketID:    market,
					PartyID:     "R",
					Side:        types.Side_Sell,
					Price:       101,
					Size:        100,
					Remaining:   0,
					TimeInForce: types.Order_GTC,
					CreatedAt:   2,
				},
				{
					MarketID:    market,
					PartyID:     "C",
					Side:        types.Side_Sell,
					Price:       102,
					Size:        100,
					Remaining:   80,
					TimeInForce: types.Order_GTC,
					CreatedAt:   0,
				},
			},
		},
		{
			// Sell is agressive, aggressive at lower price than on the book, pro rata at 99, aggressive is removed
			aggressiveOrder: &types.Order{
				MarketID:    market,
				PartyID:     "Y",
				Side:        types.Side_Sell,
				Price:       98,
				Size:        100,
				Remaining:   100,
				TimeInForce: types.Order_GTC,
				CreatedAt:   4,
			},
			expectedTrades: []types.Trade{
				{
					MarketID:  market,
					Price:     99,
					Size:      50,
					Buyer:     "E",
					Seller:    "Y",
					Aggressor: types.Side_Sell,
				},
				{
					MarketID:  market,
					Price:     99,
					Size:      50,
					Buyer:     "F",
					Seller:    "Y",
					Aggressor: types.Side_Sell,
				},
			},
			expectedPassiveOrdersAffected: []types.Order{
				{
					MarketID:    market,
					PartyID:     "E",
					Side:        types.Side_Buy,
					Price:       99,
					Size:        100,
					Remaining:   50,
					TimeInForce: types.Order_GTC,
					CreatedAt:   0,
				},
				{
					MarketID:    market,
					PartyID:     "F",
					Side:        types.Side_Buy,
					Price:       99,
					Size:        100,
					Remaining:   50,
					TimeInForce: types.Order_GTC,
					CreatedAt:   0,
				},
			},
		},
		{
			// Sell is agressive, aggressive at exact price, all orders at this price level should be hitted plus order should remain on the sell side of the book at 99 level
			aggressiveOrder: &types.Order{
				MarketID:    market,
				PartyID:     "Z",
				Side:        types.Side_Sell,
				Price:       99,
				Size:        350,
				Remaining:   350,
				TimeInForce: types.Order_GTC,
				CreatedAt:   4,
			},
			expectedTrades: []types.Trade{
				{
					MarketID:  market,
					Price:     99,
					Size:      50,
					Buyer:     "E",
					Seller:    "Z",
					Aggressor: types.Side_Sell,
				},
				{
					MarketID:  market,
					Price:     99,
					Size:      50,
					Buyer:     "F",
					Seller:    "Z",
					Aggressor: types.Side_Sell,
				},
				{
					MarketID:  market,
					Price:     99,
					Size:      100,
					Buyer:     "N",
					Seller:    "Z",
					Aggressor: types.Side_Sell,
				},
				{
					MarketID:  market,
					Price:     99,
					Size:      100,
					Buyer:     "S",
					Seller:    "Z",
					Aggressor: types.Side_Sell,
				},
			},
			expectedPassiveOrdersAffected: []types.Order{
				{
					MarketID:    market,
					PartyID:     "E",
					Side:        types.Side_Buy,
					Price:       99,
					Size:        100,
					Remaining:   0,
					TimeInForce: types.Order_GTC,
					CreatedAt:   0,
				},
				{
					MarketID:    market,
					PartyID:     "F",
					Side:        types.Side_Buy,
					Price:       99,
					Size:        100,
					Remaining:   0,
					TimeInForce: types.Order_GTC,
					CreatedAt:   0,
				},
				{
					MarketID:    market,
					PartyID:     "N",
					Side:        types.Side_Buy,
					Price:       99,
					Size:        100,
					Remaining:   0,
					TimeInForce: types.Order_GTC,
					CreatedAt:   1,
				},
				{
					MarketID:    market,
					PartyID:     "S",
					Side:        types.Side_Buy,
					Price:       99,
					Size:        100,
					Remaining:   0,
					TimeInForce: types.Order_GTC,
					CreatedAt:   2,
				},
			},
		},
		{ // aggressive nonpersistent buy order, hits two price levels and is not added to order book
			aggressiveOrder: &types.Order{
				MarketID:    market,
				PartyID:     "XX",
				Side:        types.Side_Buy,
				Price:       102,
				Size:        100,
				Remaining:   100,
				TimeInForce: types.Order_FOK, // nonpersistent
				CreatedAt:   4,
			},
			expectedTrades: []types.Trade{
				{
					MarketID:  market,
					Price:     99,
					Size:      50,
					Buyer:     "XX",
					Seller:    "Z",
					Aggressor: types.Side_Buy,
				},
				{
					MarketID:  market,
					Price:     102,
					Size:      50,
					Buyer:     "XX",
					Seller:    "C",
					Aggressor: types.Side_Buy,
				},
			},
			expectedPassiveOrdersAffected: []types.Order{
				{
					MarketID:    market,
					PartyID:     "Z",
					Side:        types.Side_Sell,
					Price:       99,
					Size:        350,
					Remaining:   0,
					TimeInForce: types.Order_GTC,
					CreatedAt:   4,
				},
				{
					MarketID:    market,
					PartyID:     "C",
					Side:        types.Side_Sell,
					Price:       102,
					Size:        100,
					Remaining:   30,
					TimeInForce: types.Order_GTC,
					CreatedAt:   0,
				},
			},
		},
		{ // aggressive nonpersistent buy order, hits one price levels and is not added to order book
			aggressiveOrder: &types.Order{
				MarketID:    market,
				PartyID:     "YY",
				Side:        types.Side_Buy,
				Price:       103,
				Size:        200,
				Remaining:   200,
				TimeInForce: types.Order_IOC, // nonpersistent
				CreatedAt:   5,
			},
			expectedTrades: []types.Trade{
				{
					MarketID:  market,
					Price:     102,
					Size:      30,
					Buyer:     "YY",
					Seller:    "C",
					Aggressor: types.Side_Buy,
				},
				{
					MarketID:  market,
					Price:     103,
					Size:      100,
					Buyer:     "YY",
					Seller:    "D",
					Aggressor: types.Side_Buy,
				},
			},
			expectedPassiveOrdersAffected: []types.Order{
				{
					MarketID:    market,
					PartyID:     "C",
					Side:        types.Side_Sell,
					Price:       102,
					Size:        100,
					Remaining:   0,
					TimeInForce: types.Order_GTC,
					CreatedAt:   0,
				},
				{
					MarketID:    market,
					PartyID:     "D",
					Side:        types.Side_Sell,
					Price:       103,
					Size:        100,
					Remaining:   0,
					TimeInForce: types.Order_GTC,
					CreatedAt:   0,
				},
			},
		},
		{ // aggressive nonpersistent buy order, hits two price levels and is not added to order book
			aggressiveOrder: &types.Order{
				MarketID:    market,
				PartyID:     "XX",
				Side:        types.Side_Sell,
				Price:       96,
				Size:        2000,
				Remaining:   2000,
				TimeInForce: types.Order_FOK, // nonpersistent
				CreatedAt:   5,
			},
			expectedTrades:                []types.Trade{},
			expectedPassiveOrdersAffected: []types.Order{},
		},
		{ // aggressive nonpersistent buy order, hits two price levels and is not added to order book
			aggressiveOrder: &types.Order{
				MarketID:    market,
				PartyID:     "XX",
				Side:        types.Side_Buy,
				Price:       102,
				Size:        2000,
				Remaining:   2000,
				TimeInForce: types.Order_FOK, // nonpersistent
				CreatedAt:   5,
			},
			expectedTrades:                []types.Trade{},
			expectedPassiveOrdersAffected: []types.Order{},
		},
		{ // aggressive nonpersistent buy order, at super low price hits one price levels and is not added to order book
			aggressiveOrder: &types.Order{
				MarketID:    market,
				PartyID:     "ZZ",
				Side:        types.Side_Sell,
				Price:       95,
				Size:        200,
				Remaining:   200,
				TimeInForce: types.Order_IOC, // nonpersistent
				CreatedAt:   5,
			},
			expectedTrades: []types.Trade{
				{
					MarketID:  market,
					Price:     98,
					Size:      100,
					Buyer:     "G",
					Seller:    "ZZ",
					Aggressor: types.Side_Sell,
				},
			},
			expectedPassiveOrdersAffected: []types.Order{
				{
					MarketID:    market,
					PartyID:     "G",
					Side:        types.Side_Buy,
					Price:       98,
					Size:        100,
					Remaining:   0,
					TimeInForce: types.Order_GTC,
					CreatedAt:   0,
				},
			},
		},
		{ // aggressive nonpersistent buy order, at super low price hits one price levels and is not added to order book
			aggressiveOrder: &types.Order{
				MarketID:    market,
				PartyID:     "ZZ",
				Side:        types.Side_Sell,
				Price:       95,
				Size:        200,
				Remaining:   200,
				TimeInForce: types.Order_GTT, // nonpersistent
				CreatedAt:   5,
				ExpiresAt:   6,
			},
			expectedTrades:                []types.Trade{},
			expectedPassiveOrdersAffected: []types.Order{},
		},
		{ // aggressive nonpersistent buy order, at super low price hits one price levels and is not added to order book
			aggressiveOrder: &types.Order{
				MarketID:    market,
				PartyID:     "ZXY",
				Side:        types.Side_Buy,
				Price:       95,
				Size:        100,
				Remaining:   100,
				TimeInForce: types.Order_FOK, // nonpersistent
				CreatedAt:   6,
			},
			expectedTrades: []types.Trade{
				{
					MarketID:  market,
					Price:     95,
					Size:      100,
					Buyer:     "ZXY",
					Seller:    "ZZ",
					Aggressor: types.Side_Buy,
				},
			},
			expectedPassiveOrdersAffected: []types.Order{
				{
					MarketID:    market,
					PartyID:     "ZZ",
					Side:        types.Side_Sell,
					Price:       95,
					Size:        200,
					Remaining:   100,
					TimeInForce: types.Order_GTT, // nonpersistent
					CreatedAt:   5,
					ExpiresAt:   7,
				},
			},
		},
		{ // aggressive nonpersistent buy order, hits two price levels and is not added to order book
			aggressiveOrder: &types.Order{
				MarketID:    market,
				PartyID:     "XX",
				Side:        types.Side_Buy,
				Price:       102,
				Size:        2000,
				Remaining:   2000,
				TimeInForce: types.Order_FOK, // nonpersistent
				CreatedAt:   6,
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

		confirmationtypes, err := book.SubmitOrder(s.aggressiveOrder)

		//this should not return any errors
		assert.Equal(t, nil, err)

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
		book.RemoveExpiredOrders(s.aggressiveOrder.CreatedAt)
	}
}

func TestOrderBook_SubmitOrderInvalidMarket(t *testing.T) {
	market := "testOrderbook"
	book := getTestOrderBook(t, market, true)
	defer book.Finish()

	newOrder := &types.Order{
		MarketID:    "invalid",
		PartyID:     "A",
		Side:        types.Side_Sell,
		Price:       101,
		Size:        100,
		Remaining:   100,
		TimeInForce: types.Order_GTC,
		CreatedAt:   0,
		Id:          fmt.Sprintf("V%d-%d", 1, 1),
	}

	_, err := book.SubmitOrder(newOrder)
	if err != nil {
		fmt.Println(err)
	}

	assert.Equal(t, types.OrderError_INVALID_MARKET_ID, err)

}

func TestOrderBook_CancelSellOrder(t *testing.T) {
	market := "testOrderbook"
	book := getTestOrderBook(t, market, true)
	defer book.Finish()
	logger := logging.NewTestLogger()
	defer logger.Sync()

	logger.Debug("BEGIN CANCELLING VALID ORDER")

	// Arrange
	newOrder := &types.Order{
		MarketID:    market,
		PartyID:     "A",
		Side:        types.Side_Sell,
		Price:       101,
		Size:        100,
		Remaining:   100,
		TimeInForce: types.Order_GTC,
		CreatedAt:   0,
		Id:          fmt.Sprintf("V%d-%d", 1, 1),
	}

	confirmation, err := book.SubmitOrder(newOrder)
	orderAdded := confirmation.Order

	// Act
	res, err := book.CancelOrder(orderAdded)
	if err != nil {
		fmt.Println(err)
	}

	// Assert
	assert.Nil(t, err)
	assert.Equal(t, "V1-1", res.Order.Id)
	assert.Equal(t, types.Order_Cancelled, res.Order.Status)

	book.PrintState("AFTER CANCEL ORDER")
}

func TestOrderBook_CancelBuyOrder(t *testing.T) {
	market := "testOrderbook"
	book := getTestOrderBook(t, market, true)
	defer book.Finish()

	logger := logging.NewTestLogger()
	defer logger.Sync()
	logger.Debug("BEGIN CANCELLING VALID ORDER")

	// Arrange
	newOrder := &types.Order{
		MarketID:    market,
		PartyID:     "A",
		Side:        types.Side_Buy,
		Price:       101,
		Size:        100,
		Remaining:   100,
		TimeInForce: types.Order_GTC,
		CreatedAt:   0,
		Id:          fmt.Sprintf("V%d-%d", 1, 1),
	}

	confirmation, err := book.SubmitOrder(newOrder)
	orderAdded := confirmation.Order

	// Act
	res, err := book.CancelOrder(orderAdded)
	if err != nil {
		fmt.Println(err)
	}

	// Assert
	assert.Nil(t, err)
	assert.Equal(t, "V1-1", res.Order.Id)
	assert.Equal(t, types.Order_Cancelled, res.Order.Status)

	book.PrintState("AFTER CANCEL ORDER")
}

func TestOrderBook_CancelOrderMarketMismatch(t *testing.T) {
	logger := logging.NewTestLogger()
	defer logger.Sync()
	logger.Debug("BEGIN CANCELLING MARKET MISMATCH ORDER")

	market := "testOrderbook"
	book := getTestOrderBook(t, market, true)
	defer book.Finish()
	newOrder := &types.Order{
		MarketID: market,
		Id:       "123456",
	}

	confirmation, err := book.SubmitOrder(newOrder)
	orderAdded := confirmation.Order

	orderAdded.MarketID = "invalid" // Bad market, malformed?

	_, err = book.CancelOrder(orderAdded)
	if err != nil {
		fmt.Println(err)
	}

	assert.Equal(t, types.OrderError_ORDER_REMOVAL_FAILURE, err)
}

func TestOrderBook_CancelOrderInvalidID(t *testing.T) {
	logger := logging.NewTestLogger()
	defer logger.Sync()
	logger.Debug("BEGIN CANCELLING INVALID ORDER")

	market := "testOrderbook"
	book := getTestOrderBook(t, market, true)
	defer book.Finish()
	newOrder := &types.Order{
		MarketID: market,
		Id:       "id",
	}

	confirmation, err := book.SubmitOrder(newOrder)
	orderAdded := confirmation.Order

	_, err = book.CancelOrder(orderAdded)
	if err != nil {
		logger.Debug("error cancelling order", logging.Error(err))
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
	assert.Equal(t, expectedOrder.MarketID, order.MarketID)
	assert.Equal(t, expectedOrder.PartyID, order.PartyID)
	assert.Equal(t, expectedOrder.Side, order.Side)
	assert.Equal(t, expectedOrder.Price, order.Price)
	assert.Equal(t, expectedOrder.Size, order.Size)
	assert.Equal(t, expectedOrder.Remaining, order.Remaining)
	assert.Equal(t, expectedOrder.TimeInForce, order.TimeInForce)
	assert.Equal(t, expectedOrder.CreatedAt, order.CreatedAt)
}

func TestOrderBook_AmendOrder(t *testing.T) {
	market := "testOrderbook"
	book := getTestOrderBook(t, market, true)
	defer book.Finish()

	newOrder := &types.Order{
		MarketID:    market,
		Id:          "123456",
		Side:        types.Side_Buy,
		Price:       100,
		Size:        200,
		Remaining:   200,
		TimeInForce: types.Order_GTC,
	}

	confirmation, err := book.SubmitOrder(newOrder)
	if err != nil {
		t.Log(err)
	}

	assert.Equal(t, nil, err)
	assert.NotNil(t, confirmation)
	assert.Equal(t, "123456", confirmation.Order.Id)
	assert.Equal(t, 0, len(confirmation.Trades))

	editedOrder := &types.Order{
		MarketID:    market,
		Id:          "123456",
		Side:        types.Side_Buy,
		Price:       100,
		Size:        200,
		Remaining:   200,
		TimeInForce: types.Order_GTC,
	}

	err = book.AmendOrder(editedOrder)
	if err != nil {
		t.Log(err)
	}

	assert.Nil(t, err)
}

func TestOrderBook_AmendOrderInvalidRemaining(t *testing.T) {
	market := "testOrderbook"
	book := getTestOrderBook(t, market, true)
	defer book.Finish()

	newOrder := &types.Order{
		MarketID:    market,
		Id:          "123456",
		Side:        types.Side_Buy,
		Price:       100,
		Size:        200,
		Remaining:   200,
		TimeInForce: types.Order_GTC,
	}

	confirmation, err := book.SubmitOrder(newOrder)
	if err != nil {
		t.Log(err)
	}

	assert.Equal(t, nil, err)
	assert.NotNil(t, confirmation)
	assert.Equal(t, "123456", confirmation.Order.Id)
	assert.Equal(t, 0, len(confirmation.Trades))

	editedOrder := &types.Order{
		MarketID:    market,
		Id:          "123456",
		Side:        types.Side_Sell,
		Price:       100,
		Size:        100,
		Remaining:   200,
		TimeInForce: types.Order_GTC,
	}
	err = book.AmendOrder(editedOrder)
	if err != types.OrderError_INVALID_REMAINING_SIZE {
		t.Log(err)
	}

	assert.Equal(t, types.OrderError_INVALID_REMAINING_SIZE, err)
}

func TestOrderBook_AmendOrderInvalidAmend(t *testing.T) {
	market := "testOrderbook"
	book := getTestOrderBook(t, market, true)
	defer book.Finish()

	newOrder := &types.Order{
		MarketID:    market,
		Id:          "123456",
		Side:        types.Side_Buy,
		Price:       100,
		Size:        200,
		Remaining:   200,
		TimeInForce: types.Order_GTC,
	}

	confirmation, err := book.SubmitOrder(newOrder)
	if err != nil {
		fmt.Println(err)
	}

	fmt.Printf("confirmation : %+v", confirmation)

	editedOrder := &types.Order{
		MarketID:    market,
		Id:          "123456",
		Side:        types.Side_Sell,
		Price:       100,
		Size:        200,
		Remaining:   200,
		TimeInForce: types.Order_GTC,
	}

	err = book.AmendOrder(editedOrder)
	if err != types.OrderError_ORDER_NOT_FOUND {
		fmt.Println(err)
	}

	assert.Equal(t, types.OrderError_ORDER_NOT_FOUND, err)
}

func TestOrderBook_AmendOrderInvalidAmend1(t *testing.T) {
	logger := logging.NewTestLogger()
	defer logger.Sync()
	logger.Debug("BEGIN AMENDING ORDER")

	market := "testOrderbook"
	book := getTestOrderBook(t, market, true)
	defer book.Finish()
	newOrder := &types.Order{
		MarketID:    market,
		Id:          "123456",
		Side:        types.Side_Buy,
		Price:       100,
		PartyID:     "A",
		Size:        200,
		Remaining:   200,
		TimeInForce: types.Order_GTC,
	}

	confirmation, err := book.SubmitOrder(newOrder)
	if err != nil {
		t.Log(err)
	}

	assert.Equal(t, nil, err)
	assert.NotNil(t, confirmation)
	assert.Equal(t, "123456", confirmation.Order.Id)
	assert.Equal(t, 0, len(confirmation.Trades))

	editedOrder := &types.Order{
		MarketID:    market,
		Id:          "123456",
		Side:        types.Side_Buy,
		Price:       100,
		PartyID:     "B",
		Size:        200,
		Remaining:   200,
		TimeInForce: types.Order_GTC,
	}

	err = book.AmendOrder(editedOrder)
	if err != types.OrderError_ORDER_AMEND_FAILURE {
		t.Log(err)
	}

	assert.Equal(t, types.OrderError_ORDER_AMEND_FAILURE, err)
}

func TestOrderBook_AmendOrderInvalidAmendOutOfSequence(t *testing.T) {
	market := "testOrderbook"
	book := getTestOrderBook(t, market, true)
	defer book.Finish()

	logger := logging.NewTestLogger()
	defer logger.Sync()
	logger.Debug("BEGIN AMENDING OUT OF SEQUENCE ORDER")

	newOrder := &types.Order{
		MarketID:    market,
		Id:          "123456",
		Side:        types.Side_Buy,
		Price:       100,
		PartyID:     "A",
		Size:        200,
		Remaining:   200,
		TimeInForce: types.Order_GTC,
		CreatedAt:   10,
	}

	confirmation, err := book.SubmitOrder(newOrder)
	if err != nil {
		fmt.Println(err)
	}

	assert.Equal(t, nil, err)
	assert.NotNil(t, confirmation)
	assert.Equal(t, "123456", confirmation.Order.Id)
	assert.Equal(t, 0, len(confirmation.Trades))

	editedOrder := &types.Order{
		MarketID:    market,
		Id:          "123456",
		Side:        types.Side_Buy,
		Price:       100,
		PartyID:     "A",
		Size:        200,
		Remaining:   200,
		TimeInForce: types.Order_GTC,
		CreatedAt:   5,
	}

	err = book.AmendOrder(editedOrder)
	if err != types.OrderError_ORDER_OUT_OF_SEQUENCE {
		t.Log(err)
	}

	assert.Equal(t, types.OrderError_ORDER_OUT_OF_SEQUENCE, err)
}

func TestOrderBook_AmendOrderInvalidAmendSize(t *testing.T) {
	market := "testOrderbook"
	book := getTestOrderBook(t, market, true)
	defer book.Finish()

	logger := logging.NewTestLogger()
	defer logger.Sync()
	logger.Debug("BEGIN AMEND ORDER INVALID SIZE")

	newOrder := &types.Order{
		MarketID:    market,
		Id:          "123456",
		Side:        types.Side_Buy,
		Price:       100,
		PartyID:     "A",
		Size:        200,
		Remaining:   200,
		TimeInForce: types.Order_GTC,
		CreatedAt:   10,
	}

	confirmation, err := book.SubmitOrder(newOrder)
	if err != nil {
		t.Log(err)
	}

	assert.Equal(t, nil, err)
	assert.NotNil(t, confirmation)
	assert.Equal(t, "123456", confirmation.Order.Id)
	assert.Equal(t, 0, len(confirmation.Trades))

	editedOrder := &types.Order{
		MarketID:    market,
		Id:          "123456",
		Side:        types.Side_Buy,
		Price:       100,
		PartyID:     "B",
		Size:        300,
		Remaining:   300,
		TimeInForce: types.Order_GTC,
		CreatedAt:   10,
	}

	err = book.AmendOrder(editedOrder)
	if err != types.OrderError_ORDER_AMEND_FAILURE {
		t.Log(err)
	}

	assert.Equal(t, types.OrderError_ORDER_AMEND_FAILURE, err)
}

// ProRata mode OFF which is a default config for vega ME
func TestOrderBook_SubmitOrderProRataModeOff(t *testing.T) {
	market := "testOrderbook"
	book := getTestOrderBook(t, market, false)
	defer book.Finish()

	// logger := logging.NewTestLogger()
	// defer logger.Sync()
	// logger.Debug("BEGIN PRO-RATA MODE OFF")

	const numberOfTimestamps = 2
	m := make(map[int64][]*types.Order, numberOfTimestamps)

	// sell and buy side orders at timestamp 0
	m[0] = []*types.Order{
		// Side Sell
		{
			MarketID:    market,
			PartyID:     "A",
			Side:        types.Side_Sell,
			Price:       101,
			Size:        100,
			Remaining:   100,
			TimeInForce: types.Order_GTC,
			CreatedAt:   0,
		},
		{
			MarketID:    market,
			PartyID:     "B",
			Side:        types.Side_Sell,
			Price:       101,
			Size:        100,
			Remaining:   100,
			TimeInForce: types.Order_GTC,
			CreatedAt:   0,
		},
		// Side Buy
		{
			MarketID:    market,
			PartyID:     "C",
			Side:        types.Side_Buy,
			Price:       98,
			Size:        100,
			Remaining:   100,
			TimeInForce: types.Order_GTC,
			CreatedAt:   0,
		},
		{
			MarketID:    market,
			PartyID:     "D",
			Side:        types.Side_Buy,
			Price:       98,
			Size:        100,
			Remaining:   100,
			TimeInForce: types.Order_GTC,
			CreatedAt:   0,
		},
	}

	// sell and buy orders at timestamp 1
	m[1] = []*types.Order{
		// Side Sell
		{
			MarketID:    market,
			PartyID:     "E",
			Side:        types.Side_Sell,
			Price:       101,
			Size:        100,
			Remaining:   100,
			TimeInForce: types.Order_GTC,
			CreatedAt:   1,
		},
		// Side Buy
		{
			MarketID:    market,
			PartyID:     "F",
			Side:        types.Side_Buy,
			Price:       99,
			Size:        100,
			Remaining:   100,
			TimeInForce: types.Order_GTC,
			CreatedAt:   1,
		},
	}

	timestamps := []int64{0, 1}
	for _, timestamp := range timestamps {
		for index, _ := range m[timestamp] {
			fmt.Println("tests calling book.SubmitOrder: ", m[timestamp][index])
			confirmationtypes, err := book.SubmitOrder(m[timestamp][index])
			// this should not return any errors
			assert.Equal(t, nil, err)
			// this should not generate any trades
			assert.Equal(t, 0, len(confirmationtypes.Trades))
		}
	}

	scenario := []aggressiveOrderScenario{
		{
			// same price level, remaining on the passive
			aggressiveOrder: &types.Order{
				MarketID:    market,
				PartyID:     "M",
				Side:        types.Side_Buy,
				Price:       101,
				Size:        100,
				Remaining:   100,
				TimeInForce: types.Order_GTC,
				CreatedAt:   3,
			},
			expectedTrades: []types.Trade{
				{
					MarketID:  market,
					Price:     101,
					Size:      100,
					Buyer:     "M",
					Seller:    "A",
					Aggressor: types.Side_Buy,
				},
			},
			expectedPassiveOrdersAffected: []types.Order{
				{
					MarketID:    market,
					PartyID:     "A",
					Side:        types.Side_Sell,
					Price:       101,
					Size:        100,
					Remaining:   0,
					TimeInForce: types.Order_GTC,
					CreatedAt:   0,
				},
			},
		},
		{
			// same price level, remaining on the passive
			aggressiveOrder: &types.Order{
				MarketID:    market,
				PartyID:     "N",
				Side:        types.Side_Buy,
				Price:       102,
				Size:        200,
				Remaining:   200,
				TimeInForce: types.Order_GTC,
				CreatedAt:   4,
			},
			expectedTrades: []types.Trade{
				{
					MarketID:  market,
					Price:     101,
					Size:      100,
					Buyer:     "N",
					Seller:    "B",
					Aggressor: types.Side_Buy,
				},
				{
					MarketID:  market,
					Price:     101,
					Size:      100,
					Buyer:     "N",
					Seller:    "E",
					Aggressor: types.Side_Buy,
				},
			},
			expectedPassiveOrdersAffected: []types.Order{
				{
					MarketID:    market,
					PartyID:     "B",
					Side:        types.Side_Sell,
					Price:       101,
					Size:        100,
					Remaining:   0,
					TimeInForce: types.Order_GTC,
					CreatedAt:   0,
				},
				{
					MarketID:    market,
					PartyID:     "E",
					Side:        types.Side_Sell,
					Price:       101,
					Size:        100,
					Remaining:   0,
					TimeInForce: types.Order_GTC,
					CreatedAt:   1,
				},
			},
		},
		{
			// same price level, NO PRORATA
			aggressiveOrder: &types.Order{
				MarketID:    market,
				PartyID:     "O",
				Side:        types.Side_Sell,
				Price:       97,
				Size:        250,
				Remaining:   250,
				TimeInForce: types.Order_GTC,
				CreatedAt:   5,
			},
			expectedTrades: []types.Trade{
				{
					MarketID:  market,
					Price:     99,
					Size:      100,
					Buyer:     "F",
					Seller:    "O",
					Aggressor: types.Side_Sell,
				},
				{
					MarketID:  market,
					Price:     98,
					Size:      100,
					Buyer:     "C",
					Seller:    "O",
					Aggressor: types.Side_Sell,
				},
				{
					MarketID:  market,
					Price:     98,
					Size:      50,
					Buyer:     "D",
					Seller:    "O",
					Aggressor: types.Side_Sell,
				},
			},
			expectedPassiveOrdersAffected: []types.Order{
				{
					MarketID:    market,
					PartyID:     "F",
					Side:        types.Side_Buy,
					Price:       99,
					Size:        100,
					Remaining:   0,
					TimeInForce: types.Order_GTC,
					CreatedAt:   1,
				},
				{
					MarketID:    market,
					PartyID:     "C",
					Side:        types.Side_Buy,
					Price:       98,
					Size:        100,
					Remaining:   0,
					TimeInForce: types.Order_GTC,
					CreatedAt:   0,
				},
				{
					MarketID:    market,
					PartyID:     "D",
					Side:        types.Side_Buy,
					Price:       98,
					Size:        100,
					Remaining:   50,
					TimeInForce: types.Order_GTC,
					CreatedAt:   0,
				},
			},
		},
		{
			// same price level, NO PRORATA
			aggressiveOrder: &types.Order{
				MarketID:    market,
				PartyID:     "X",
				Side:        types.Side_Sell,
				Price:       98,
				Size:        50,
				Remaining:   50,
				TimeInForce: types.Order_GTC,
				CreatedAt:   6,
			},
			expectedTrades: []types.Trade{
				{
					MarketID:  market,
					Price:     98,
					Size:      50,
					Buyer:     "D",
					Seller:    "X",
					Aggressor: types.Side_Sell,
				},
			},
			expectedPassiveOrdersAffected: []types.Order{
				{
					MarketID:    market,
					PartyID:     "D",
					Side:        types.Side_Buy,
					Price:       98,
					Size:        100,
					Remaining:   0,
					TimeInForce: types.Order_GTC,
					CreatedAt:   0,
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

		confirmationtypes, err := book.SubmitOrder(s.aggressiveOrder)

		//this should not return any errors
		assert.Equal(t, nil, err)

		//this should not generate any trades
		assert.Equal(t, len(s.expectedTrades), len(confirmationtypes.Trades))

		fmt.Println("CONFIRMATION types:")
		fmt.Println("-> Aggresive:", confirmationtypes.Order)
		fmt.Println("-> Trades :", confirmationtypes.Trades)
		fmt.Println("-> PassiveOrdersAffected:", confirmationtypes.PassiveOrdersAffected)
		fmt.Printf("Scenario: %d / %d \n", i+1, len(scenario))

		// assert.Equal(t, len(s.expectedTrades), len(confirmationtypes.Trades))
		// trades should match expected trades
		for i, exp := range s.expectedTrades {
			expectTrade(t, &exp, confirmationtypes.Trades[i])
		}
		// for i, trade := range confirmationtypes.Trades {
		// expectTrade(t, &s.expectedTrades[i], trade)
		// }

		// orders affected should match expected values
		for i, exp := range s.expectedPassiveOrdersAffected {
			expectOrder(t, &exp, confirmationtypes.PassiveOrdersAffected[i])
		}
		// for i, orderAffected := range confirmationtypes.PassiveOrdersAffected {
		// 	expectOrder(t, &s.expectedPassiveOrdersAffected[i], orderAffected)
		// }

		// call remove expired orders every scenario
		book.RemoveExpiredOrders(s.aggressiveOrder.CreatedAt)
	}
}
