package matching_test

import (
	"fmt"
	"testing"

	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/matching"
	types "code.vegaprotocol.io/vega/proto"
	"code.vegaprotocol.io/vega/vegatime"

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
	expectedAggressiveOrderStatus types.Order_Status
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

func getTestOrderBook(t *testing.T, market string) *tstOB {
	tob := tstOB{
		log: logging.NewTestLogger(),
	}
	tob.OrderBook = matching.NewOrderBook(tob.log, matching.NewDefaultConfig(), market, markPrice)
	tob.OrderBook.LogRemovedOrdersDebug = true
	return &tob
}

func TestOrderBook_GetClosePNL(t *testing.T) {
	t.Run("Get Buy-side close PNL values", getClosePNLBuy)
	t.Run("Get Sell-side close PNL values", getClosePNLSell)
	t.Run("Get Incomplete close-out-pnl (check error) - Buy", getClosePNLIncompleteBuy)
	t.Run("Get Incomplete close-out-pnl (check error) - Sell", getClosePNLIncompleteSell)
	t.Run("Get Best bid price and volume", testBestBidPriceAndVolume)
	t.Run("Get Best offer price and volume", testBestOfferPriceAndVolume)
}

func testBestBidPriceAndVolume(t *testing.T) {
	market := "testMarket"
	book := getTestOrderBook(t, market)
	defer book.Finish()
	// 3 orders of size 1, 3 different prices
	orders := []*types.Order{
		{
			Status:      types.Order_STATUS_ACTIVE,
			Type:        types.Order_LIMIT,
			MarketID:    market,
			PartyID:     "A",
			Side:        types.Side_SIDE_BUY,
			Price:       100,
			Size:        1,
			Remaining:   1,
			TimeInForce: types.Order_GTC,
		},
		{
			Status:      types.Order_STATUS_ACTIVE,
			Type:        types.Order_LIMIT,
			MarketID:    market,
			PartyID:     "B",
			Side:        types.Side_SIDE_BUY,
			Price:       300,
			Size:        5,
			Remaining:   5,
			TimeInForce: types.Order_GTC,
		},
		{
			Status:      types.Order_STATUS_ACTIVE,
			Type:        types.Order_LIMIT,
			MarketID:    market,
			PartyID:     "B",
			Side:        types.Side_SIDE_BUY,
			Price:       200,
			Size:        1,
			Remaining:   1,
			TimeInForce: types.Order_GTC,
		},
		{
			Status:      types.Order_STATUS_ACTIVE,
			Type:        types.Order_LIMIT,
			MarketID:    market,
			PartyID:     "d",
			Side:        types.Side_SIDE_BUY,
			Price:       300,
			Size:        10,
			Remaining:   10,
			TimeInForce: types.Order_GTC,
		},
	}
	for _, o := range orders {
		confirm, err := book.SubmitOrder(o)
		assert.NoError(t, err)
		assert.Equal(t, 0, len(confirm.Trades))
	}

	price, volume := book.BestBidPriceAndVolume()
	assert.Equal(t, uint64(300), price)
	assert.Equal(t, uint64(15), volume)
}

func testBestOfferPriceAndVolume(t *testing.T) {
	market := "testMarket"
	book := getTestOrderBook(t, market)
	defer book.Finish()
	// 3 orders of size 1, 3 different prices
	orders := []*types.Order{
		{
			Status:      types.Order_STATUS_ACTIVE,
			Type:        types.Order_LIMIT,
			MarketID:    market,
			PartyID:     "A",
			Side:        types.Side_SIDE_SELL,
			Price:       100,
			Size:        1,
			Remaining:   1,
			TimeInForce: types.Order_GTC,
		},
		{
			Status:      types.Order_STATUS_ACTIVE,
			Type:        types.Order_LIMIT,
			MarketID:    market,
			PartyID:     "B",
			Side:        types.Side_SIDE_SELL,
			Price:       10,
			Size:        5,
			Remaining:   5,
			TimeInForce: types.Order_GTC,
		},
		{
			Status:      types.Order_STATUS_ACTIVE,
			Type:        types.Order_LIMIT,
			MarketID:    market,
			PartyID:     "B",
			Side:        types.Side_SIDE_SELL,
			Price:       200,
			Size:        1,
			Remaining:   1,
			TimeInForce: types.Order_GTC,
		},
		{
			Status:      types.Order_STATUS_ACTIVE,
			Type:        types.Order_LIMIT,
			MarketID:    market,
			PartyID:     "d",
			Side:        types.Side_SIDE_SELL,
			Price:       10,
			Size:        10,
			Remaining:   10,
			TimeInForce: types.Order_GTC,
		},
	}
	for _, o := range orders {
		confirm, err := book.SubmitOrder(o)
		assert.NoError(t, err)
		assert.Equal(t, 0, len(confirm.Trades))
	}

	price, volume := book.BestOfferPriceAndVolume()
	assert.Equal(t, uint64(10), price)
	assert.Equal(t, uint64(15), volume)
}

func getClosePNLIncompleteBuy(t *testing.T) {
	market := "testMarket"
	book := getTestOrderBook(t, market)
	defer book.Finish()
	// 3 orders of size 1, 3 different prices
	orders := []*types.Order{
		{
			Status:      types.Order_STATUS_ACTIVE,
			Type:        types.Order_LIMIT,
			MarketID:    market,
			PartyID:     "A",
			Side:        types.Side_SIDE_BUY,
			Price:       100,
			Size:        1,
			Remaining:   1,
			TimeInForce: types.Order_GTC,
			CreatedAt:   0,
		},
		{
			Status:      types.Order_STATUS_ACTIVE,
			Type:        types.Order_LIMIT,
			MarketID:    market,
			PartyID:     "B",
			Side:        types.Side_SIDE_BUY,
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
		1: 110,
	}
	// this calculates the actual volume
	for vol, exp := range callExp {
		price, err := book.GetCloseoutPrice(vol, types.Side_SIDE_BUY)
		assert.Equal(t, exp, price)
		assert.NoError(t, err)
	}
	price, err := book.GetCloseoutPrice(3, types.Side_SIDE_BUY)
	assert.Equal(t, callExp[2], price)
	assert.Equal(t, matching.ErrNotEnoughOrders, err)
}

func getClosePNLIncompleteSell(t *testing.T) {
	market := "testMarket"
	book := getTestOrderBook(t, market)
	defer book.Finish()
	// 3 orders of size 1, 3 different prices
	orders := []*types.Order{
		{
			Status:      types.Order_STATUS_ACTIVE,
			Type:        types.Order_LIMIT,
			MarketID:    market,
			PartyID:     "A",
			Side:        types.Side_SIDE_SELL,
			Price:       100,
			Size:        1,
			Remaining:   1,
			TimeInForce: types.Order_GTC,
			CreatedAt:   0,
		},
		{
			Status:      types.Order_STATUS_ACTIVE,
			Type:        types.Order_LIMIT,
			MarketID:    market,
			PartyID:     "B",
			Side:        types.Side_SIDE_SELL,
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
		price, err := book.GetCloseoutPrice(vol, types.Side_SIDE_SELL)
		assert.Equal(t, exp, price)
		assert.NoError(t, err)
	}
	price, err := book.GetCloseoutPrice(3, types.Side_SIDE_SELL)
	assert.Equal(t, callExp[2], price)
	assert.Equal(t, matching.ErrNotEnoughOrders, err)
}

func getClosePNLBuy(t *testing.T) {
	market := "testMarket"
	book := getTestOrderBook(t, market)
	defer book.Finish()
	// 3 orders of size 1, 3 different prices
	orders := []*types.Order{
		{
			Status:      types.Order_STATUS_ACTIVE,
			Type:        types.Order_LIMIT,
			MarketID:    market,
			PartyID:     "A",
			Side:        types.Side_SIDE_BUY,
			Price:       100,
			Size:        1,
			Remaining:   1,
			TimeInForce: types.Order_GTC,
			CreatedAt:   0,
		},
		{
			Status:      types.Order_STATUS_ACTIVE,
			Type:        types.Order_LIMIT,
			MarketID:    market,
			PartyID:     "B",
			Side:        types.Side_SIDE_BUY,
			Price:       110,
			Size:        1,
			Remaining:   1,
			TimeInForce: types.Order_GTC,
			CreatedAt:   0,
		},
		{
			Status:      types.Order_STATUS_ACTIVE,
			Type:        types.Order_LIMIT,
			MarketID:    market,
			PartyID:     "C",
			Side:        types.Side_SIDE_BUY,
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
		2: 230 / 2,
		1: 120,
	}
	// this calculates the actual volume
	for vol, exp := range callExp {
		price, err := book.GetCloseoutPrice(vol, types.Side_SIDE_BUY)
		assert.Equal(t, int(exp), int(price))
		assert.NoError(t, err)
	}
}

func getClosePNLSell(t *testing.T) {
	market := "testMarket"
	book := getTestOrderBook(t, market)
	defer book.Finish()
	// 3 orders of size 1, 3 different prices
	orders := []*types.Order{
		{
			Status:      types.Order_STATUS_ACTIVE,
			Type:        types.Order_LIMIT,
			MarketID:    market,
			PartyID:     "A",
			Side:        types.Side_SIDE_SELL,
			Price:       100,
			Size:        1,
			Remaining:   1,
			TimeInForce: types.Order_GTC,
			CreatedAt:   0,
		},
		{
			Status:      types.Order_STATUS_ACTIVE,
			Type:        types.Order_LIMIT,
			MarketID:    market,
			PartyID:     "B",
			Side:        types.Side_SIDE_SELL,
			Price:       110,
			Size:        1,
			Remaining:   1,
			TimeInForce: types.Order_GTC,
			CreatedAt:   0,
		},
		{
			Status:      types.Order_STATUS_ACTIVE,
			Type:        types.Order_LIMIT,
			MarketID:    market,
			PartyID:     "C",
			Side:        types.Side_SIDE_SELL,
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
		price, err := book.GetCloseoutPrice(vol, types.Side_SIDE_SELL)
		assert.NoError(t, err)
		assert.Equal(t, exp, price)
	}
}

func TestOrderBook_CancelReturnsTheOrderFromTheBook(t *testing.T) {
	market := "cancel-returns-order"
	party := "p1"

	book := getTestOrderBook(t, market)
	defer book.Finish()
	currentTimestamp := getCurrentUtcTimestampNano()

	order1 := types.Order{
		Status:      types.Order_STATUS_ACTIVE,
		Type:        types.Order_LIMIT,
		MarketID:    market,
		PartyID:     party,
		Side:        types.Side_SIDE_SELL,
		Price:       1,
		Size:        100,
		Remaining:   100,
		TimeInForce: types.Order_GTC,
		CreatedAt:   currentTimestamp,
		Id:          "v0000000000000-0000001",
	}
	order2 := types.Order{
		Status:      types.Order_STATUS_ACTIVE,
		Type:        types.Order_LIMIT,
		MarketID:    market,
		PartyID:     party,
		Side:        types.Side_SIDE_SELL,
		Price:       1,
		Size:        100,
		Remaining:   1, // use a wrong remaining here to get the order from the book
		TimeInForce: types.Order_GTC,
		CreatedAt:   currentTimestamp,
		Id:          "v0000000000000-0000001",
	}

	_, err := book.SubmitOrder(&order1)
	assert.Equal(t, err, nil)

	o, err := book.CancelOrder(&order2)
	assert.Equal(t, err, nil)
	assert.Equal(t, o.Order.Remaining, order1.Remaining)
}

func TestOrderBook_RemoveExpiredOrders(t *testing.T) {
	market := "expiringOrderBookTest"
	party := "clay-davis"

	book := getTestOrderBook(t, market)
	defer book.Finish()
	currentTimestamp := getCurrentUtcTimestampNano()
	someTimeLater := currentTimestamp + (1000 * 1000)

	order1 := &types.Order{
		Status:      types.Order_STATUS_ACTIVE,
		Type:        types.Order_LIMIT,
		MarketID:    market,
		PartyID:     party,
		Side:        types.Side_SIDE_SELL,
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
		Status:      types.Order_STATUS_ACTIVE,
		Type:        types.Order_LIMIT,
		MarketID:    market,
		PartyID:     party,
		Side:        types.Side_SIDE_SELL,
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
		Status:      types.Order_STATUS_ACTIVE,
		Type:        types.Order_LIMIT,
		MarketID:    market,
		PartyID:     party,
		Side:        types.Side_SIDE_SELL,
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
		Status:      types.Order_STATUS_ACTIVE,
		Type:        types.Order_LIMIT,
		MarketID:    market,
		PartyID:     party,
		Side:        types.Side_SIDE_SELL,
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
		Status:      types.Order_STATUS_ACTIVE,
		Type:        types.Order_LIMIT,
		MarketID:    market,
		PartyID:     party,
		Side:        types.Side_SIDE_SELL,
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
		Status:      types.Order_STATUS_ACTIVE,
		Type:        types.Order_LIMIT,
		MarketID:    market,
		PartyID:     party,
		Side:        types.Side_SIDE_SELL,
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
		Status:      types.Order_STATUS_ACTIVE,
		Type:        types.Order_LIMIT,
		MarketID:    market,
		PartyID:     party,
		Side:        types.Side_SIDE_SELL,
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
		Status:      types.Order_STATUS_ACTIVE,
		Type:        types.Order_LIMIT,
		MarketID:    market,
		PartyID:     party,
		Side:        types.Side_SIDE_SELL,
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
		Status:      types.Order_STATUS_ACTIVE,
		Type:        types.Order_LIMIT,
		MarketID:    market,
		PartyID:     party,
		Side:        types.Side_SIDE_SELL,
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
		Status:      types.Order_STATUS_ACTIVE,
		Type:        types.Order_LIMIT,
		MarketID:    market,
		PartyID:     party,
		Side:        types.Side_SIDE_SELL,
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
	assert.Equal(t, "1", expired[2].Id)
	assert.Equal(t, "3", expired[3].Id)
	assert.Equal(t, "5", expired[4].Id)
}

//test for order validation
func TestOrderBook_SubmitOrder2WithValidation(t *testing.T) {
	market := "testOrderbook"
	book := getTestOrderBook(t, market)
	defer book.Finish()
	timeStampOrder := types.Order{
		Status:    types.Order_STATUS_ACTIVE,
		Type:      types.Order_LIMIT,
		Id:        "timestamporderID",
		MarketID:  market,
		PartyID:   "A",
		CreatedAt: 10,
		Side:      types.Side_SIDE_BUY,
		Size:      1,
		Remaining: 1,
	}
	_, err := book.SubmitOrder(&timeStampOrder)
	assert.NoError(t, err)
	// cancel order again, just so we set the timestamp as expected
	book.CancelOrder(&timeStampOrder)

	invalidRemainingSizeOrdertypes := &types.Order{
		Status:      types.Order_STATUS_ACTIVE,
		Type:        types.Order_LIMIT,
		MarketID:    market,
		PartyID:     "A",
		Side:        types.Side_SIDE_SELL,
		Price:       100,
		Size:        100,
		Remaining:   300,
		TimeInForce: types.Order_GTC,
		CreatedAt:   10,
		Id:          "id-number-one",
	}
	_, err = book.SubmitOrder(invalidRemainingSizeOrdertypes)
	assert.Equal(t, types.OrderError_INVALID_REMAINING_SIZE, err)
}

func TestOrderBook_DeleteOrder(t *testing.T) {
	market := "testOrderbook"
	book := getTestOrderBook(t, market)
	defer book.Finish()

	newOrder := &types.Order{
		Status:      types.Order_STATUS_ACTIVE,
		Type:        types.Order_LIMIT,
		MarketID:    market,
		PartyID:     "A",
		Side:        types.Side_SIDE_SELL,
		Price:       101,
		Size:        100,
		Remaining:   100,
		TimeInForce: types.Order_GTC,
		CreatedAt:   0,
	}

	book.SubmitOrder(newOrder)

	_, err := book.DeleteOrder(newOrder)
	if err != nil {
		fmt.Println(err, "ORDER_NOT_FOUND")
	}

	book.PrintState("AFTER REMOVE ORDER")
}

func TestOrderBook_SubmitOrderInvalidMarket(t *testing.T) {
	market := "testOrderbook"
	book := getTestOrderBook(t, market)
	defer book.Finish()

	newOrder := &types.Order{
		Status:      types.Order_STATUS_ACTIVE,
		Type:        types.Order_LIMIT,
		MarketID:    "invalid",
		PartyID:     "A",
		Side:        types.Side_SIDE_SELL,
		Price:       101,
		Size:        100,
		Remaining:   100,
		TimeInForce: types.Order_GTC,
		CreatedAt:   0,
		Id:          fmt.Sprintf("V%010d-%010d", 1, 1),
	}

	_, err := book.SubmitOrder(newOrder)
	if err != nil {
		fmt.Println(err)
	}

	assert.Equal(t, types.OrderError_INVALID_MARKET_ID, err)

}

func TestOrderBook_CancelSellOrder(t *testing.T) {
	market := "testOrderbook"
	book := getTestOrderBook(t, market)
	defer book.Finish()
	logger := logging.NewTestLogger()
	defer logger.Sync()

	logger.Debug("BEGIN CANCELLING VALID ORDER")

	// Arrange
	id := fmt.Sprintf("V%010d-%010d", 1, 1)
	newOrder := &types.Order{
		Status:      types.Order_STATUS_ACTIVE,
		Type:        types.Order_LIMIT,
		MarketID:    market,
		PartyID:     "A",
		Side:        types.Side_SIDE_SELL,
		Price:       101,
		Size:        100,
		Remaining:   100,
		TimeInForce: types.Order_GTC,
		CreatedAt:   0,
		Id:          id,
	}

	confirmation, err := book.SubmitOrder(newOrder)
	assert.NoError(t, err)
	orderAdded := confirmation.Order

	// Act
	res, err := book.CancelOrder(orderAdded)
	if err != nil {
		fmt.Println(err)
	}

	// Assert
	assert.Nil(t, err)
	assert.Equal(t, id, res.Order.Id)
	assert.Equal(t, types.Order_STATUS_CANCELLED, res.Order.Status)

	book.PrintState("AFTER CANCEL ORDER")
}

func TestOrderBook_CancelBuyOrder(t *testing.T) {
	market := "testOrderbook"
	book := getTestOrderBook(t, market)
	defer book.Finish()

	logger := logging.NewTestLogger()
	defer logger.Sync()
	logger.Debug("BEGIN CANCELLING VALID ORDER")

	// Arrange
	id := fmt.Sprintf("V%010d-%010d", 1, 1)
	newOrder := &types.Order{
		Status:      types.Order_STATUS_ACTIVE,
		Type:        types.Order_LIMIT,
		MarketID:    market,
		PartyID:     "A",
		Side:        types.Side_SIDE_BUY,
		Price:       101,
		Size:        100,
		Remaining:   100,
		TimeInForce: types.Order_GTC,
		CreatedAt:   0,
		Id:          id,
	}

	confirmation, err := book.SubmitOrder(newOrder)
	assert.NoError(t, err)
	orderAdded := confirmation.Order

	// Act
	res, err := book.CancelOrder(orderAdded)
	if err != nil {
		fmt.Println(err)
	}

	// Assert
	assert.Nil(t, err)
	assert.Equal(t, id, res.Order.Id)
	assert.Equal(t, types.Order_STATUS_CANCELLED, res.Order.Status)

	book.PrintState("AFTER CANCEL ORDER")
}

func TestOrderBook_CancelOrderByID(t *testing.T) {
	market := "testOrderbook"
	book := getTestOrderBook(t, market)
	defer book.Finish()

	logger := logging.NewTestLogger()
	defer logger.Sync()
	logger.Debug("BEGIN CANCELLING VALID ORDER BY ID")

	id := fmt.Sprintf("V%010d-%010d", 1, 1)
	newOrder := &types.Order{
		Status:      types.Order_STATUS_ACTIVE,
		Type:        types.Order_LIMIT,
		MarketID:    market,
		PartyID:     "A",
		Side:        types.Side_SIDE_BUY,
		Price:       101,
		Size:        100,
		Remaining:   100,
		TimeInForce: types.Order_GTC,
		CreatedAt:   0,
		Id:          id,
	}

	confirmation, err := book.SubmitOrder(newOrder)
	assert.NotNil(t, confirmation, "submit order should succeed")
	assert.NoError(t, err, "submit order should succeed")
	orderAdded := confirmation.Order
	assert.NotNil(t, orderAdded, "submitted order is expected to be valid")

	orderFound, err := book.GetOrderByID(orderAdded.Id)
	assert.NotNil(t, orderFound, "order lookup should work for the order just submitted")
	assert.NoError(t, err, "order lookup should not fail")

	res, err := book.CancelOrder(orderFound)
	assert.NotNil(t, res, "cancelling should work for a valid order that was just found")
	assert.NoError(t, err, "order cancel should not fail")

	orderFound, err = book.GetOrderByID(orderAdded.Id)
	assert.Error(t, err, "order lookup for an already cancelled order should fail")
	assert.Nil(t, orderFound, "order lookup for an already cancelled order should not be possible")

	book.PrintState("AFTER CANCEL ORDER BY ID")
}

func TestOrderBook_CancelOrderMarketMismatch(t *testing.T) {
	logger := logging.NewTestLogger()
	defer logger.Sync()
	logger.Debug("BEGIN CANCELLING MARKET MISMATCH ORDER")

	market := "testOrderbook"
	book := getTestOrderBook(t, market)
	defer book.Finish()
	newOrder := &types.Order{
		Status:    types.Order_STATUS_ACTIVE,
		Type:      types.Order_LIMIT,
		MarketID:  market,
		Id:        fmt.Sprintf("V%010d-%010d", 1, 1),
		PartyID:   "A",
		Size:      100,
		Remaining: 100,
	}

	confirmation, err := book.SubmitOrder(newOrder)
	assert.NoError(t, err)
	orderAdded := confirmation.Order

	orderAdded.MarketID = "invalid" // Bad market, malformed?

	_, err = book.CancelOrder(orderAdded)
	if err != nil {
		fmt.Println(err)
	}

	assert.Equal(t, types.OrderError_INVALID_MARKET_ID, err)
}

func TestOrderBook_CancelOrderInvalidID(t *testing.T) {
	logger := logging.NewTestLogger()
	defer logger.Sync()
	logger.Debug("BEGIN CANCELLING INVALID ORDER")

	market := "testOrderbook"
	book := getTestOrderBook(t, market)
	defer book.Finish()
	newOrder := &types.Order{
		Status:    types.Order_STATUS_ACTIVE,
		Type:      types.Order_LIMIT,
		MarketID:  market,
		Id:        "id",
		PartyID:   "A",
		Size:      100,
		Remaining: 100,
	}

	confirmation, err := book.SubmitOrder(newOrder)
	assert.NoError(t, err)
	orderAdded := confirmation.Order

	_, err = book.CancelOrder(orderAdded)
	if err != nil {
		logger.Debug("error cancelling order", logging.Error(err))
	}

	assert.Equal(t, types.OrderError_INVALID_ORDER_ID, err)
}

func expectTrade(t *testing.T, expectedTrade, trade *types.Trade) {
	// run asserts for protocol trade data
	assert.Equal(t, int(expectedTrade.Price), int(trade.Price), "invalid trade price")
	assert.Equal(t, int(expectedTrade.Size), int(trade.Size), "invalid trade size")
	assert.Equal(t, expectedTrade.Buyer, trade.Buyer, "invalid trade buyer")
	assert.Equal(t, expectedTrade.Seller, trade.Seller, "invalide trade sellet")
	assert.Equal(t, expectedTrade.Aggressor, trade.Aggressor, "invalid trade aggressor")
}

func expectOrder(t *testing.T, expectedOrder, order *types.Order) {
	// run asserts for order
	assert.Equal(t, expectedOrder.MarketID, order.MarketID, "invalid order market id")
	assert.Equal(t, expectedOrder.PartyID, order.PartyID, "invalid order party id")
	assert.Equal(t, expectedOrder.Side, order.Side, "invalid order side")
	assert.Equal(t, int(expectedOrder.Price), int(order.Price), "invalid order price")
	assert.Equal(t, int(expectedOrder.Size), int(order.Size), "invalid order size")
	assert.Equal(t, int(expectedOrder.Remaining), int(order.Remaining), "invalid order remaining")
	assert.Equal(t, expectedOrder.TimeInForce, order.TimeInForce, "invalid order tif")
	assert.Equal(t, expectedOrder.CreatedAt, order.CreatedAt, "invalid order created at")
}

func TestOrderBook_AmendOrder(t *testing.T) {
	market := "testOrderbook"
	book := getTestOrderBook(t, market)
	defer book.Finish()

	newOrder := &types.Order{
		Status:      types.Order_STATUS_ACTIVE,
		Type:        types.Order_LIMIT,
		MarketID:    market,
		Id:          "123456",
		PartyID:     "A",
		Side:        types.Side_SIDE_BUY,
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
		Status:      types.Order_STATUS_ACTIVE,
		Type:        types.Order_LIMIT,
		MarketID:    market,
		Id:          "123456",
		PartyID:     "A",
		Side:        types.Side_SIDE_BUY,
		Price:       100,
		Size:        200,
		Remaining:   200,
		TimeInForce: types.Order_GTC,
	}

	err = book.AmendOrder(newOrder, editedOrder)
	if err != nil {
		t.Log(err)
	}

	assert.Nil(t, err)
}

func TestOrderBook_AmendOrderInvalidRemaining(t *testing.T) {
	market := "testOrderbook"
	book := getTestOrderBook(t, market)
	defer book.Finish()

	newOrder := &types.Order{
		Status:      types.Order_STATUS_ACTIVE,
		Type:        types.Order_LIMIT,
		MarketID:    market,
		Id:          "123456",
		PartyID:     "A",
		Side:        types.Side_SIDE_BUY,
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
		Status:      types.Order_STATUS_ACTIVE,
		Type:        types.Order_LIMIT,
		MarketID:    market,
		PartyID:     "A",
		Id:          "123456",
		Side:        types.Side_SIDE_SELL,
		Price:       100,
		Size:        100,
		Remaining:   200,
		TimeInForce: types.Order_GTC,
	}
	err = book.AmendOrder(newOrder, editedOrder)
	if err != types.OrderError_INVALID_REMAINING_SIZE {
		t.Log(err)
	}

	assert.Equal(t, types.OrderError_INVALID_REMAINING_SIZE, err)
}

func TestOrderBook_AmendOrderInvalidAmend(t *testing.T) {
	market := "testOrderbook"
	book := getTestOrderBook(t, market)
	defer book.Finish()

	newOrder := &types.Order{
		Status:      types.Order_STATUS_ACTIVE,
		Type:        types.Order_LIMIT,
		MarketID:    market,
		Id:          "123456",
		Side:        types.Side_SIDE_BUY,
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
		Status:      types.Order_STATUS_ACTIVE,
		Type:        types.Order_LIMIT,
		MarketID:    market,
		Id:          "123456",
		PartyID:     "A",
		Side:        types.Side_SIDE_SELL,
		Price:       100,
		Size:        200,
		Remaining:   200,
		TimeInForce: types.Order_GTC,
	}

	err = book.AmendOrder(newOrder, editedOrder)
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
	book := getTestOrderBook(t, market)
	defer book.Finish()
	newOrder := &types.Order{
		Status:      types.Order_STATUS_ACTIVE,
		Type:        types.Order_LIMIT,
		MarketID:    market,
		Id:          "123456",
		Side:        types.Side_SIDE_BUY,
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
		Status:      types.Order_STATUS_ACTIVE,
		Type:        types.Order_LIMIT,
		MarketID:    market,
		Id:          "123456",
		Side:        types.Side_SIDE_BUY,
		Price:       100,
		PartyID:     "B",
		Size:        200,
		Remaining:   200,
		TimeInForce: types.Order_GTC,
	}

	err = book.AmendOrder(newOrder, editedOrder)
	if err != types.OrderError_ORDER_AMEND_FAILURE {
		t.Log(err)
	}

	assert.Equal(t, types.OrderError_ORDER_AMEND_FAILURE, err)
}

func TestOrderBook_AmendOrderInvalidAmendOutOfSequence(t *testing.T) {
	market := "testOrderbook"
	book := getTestOrderBook(t, market)
	defer book.Finish()

	logger := logging.NewTestLogger()
	defer logger.Sync()
	logger.Debug("BEGIN AMENDING OUT OF SEQUENCE ORDER")

	newOrder := &types.Order{
		Status:      types.Order_STATUS_ACTIVE,
		Type:        types.Order_LIMIT,
		MarketID:    market,
		Id:          "123456",
		Side:        types.Side_SIDE_BUY,
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
		Status:      types.Order_STATUS_ACTIVE,
		Type:        types.Order_LIMIT,
		MarketID:    market,
		Id:          "123456",
		Side:        types.Side_SIDE_BUY,
		Price:       100,
		PartyID:     "A",
		Size:        200,
		Remaining:   200,
		TimeInForce: types.Order_GTC,
		CreatedAt:   5,
	}

	err = book.AmendOrder(newOrder, editedOrder)
	if err != types.OrderError_ORDER_OUT_OF_SEQUENCE {
		t.Log(err)
	}

	assert.Equal(t, types.OrderError_ORDER_OUT_OF_SEQUENCE, err)
}

func TestOrderBook_AmendOrderInvalidAmendSize(t *testing.T) {
	market := "testOrderbook"
	book := getTestOrderBook(t, market)
	defer book.Finish()

	logger := logging.NewTestLogger()
	defer logger.Sync()
	logger.Debug("BEGIN AMEND ORDER INVALID SIZE")

	newOrder := &types.Order{
		Status:      types.Order_STATUS_ACTIVE,
		Type:        types.Order_LIMIT,
		MarketID:    market,
		Id:          "123456",
		Side:        types.Side_SIDE_BUY,
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
		Status:      types.Order_STATUS_ACTIVE,
		Type:        types.Order_LIMIT,
		MarketID:    market,
		Id:          "123456",
		Side:        types.Side_SIDE_BUY,
		Price:       100,
		PartyID:     "B",
		Size:        300,
		Remaining:   300,
		TimeInForce: types.Order_GTC,
		CreatedAt:   10,
	}

	err = book.AmendOrder(newOrder, editedOrder)
	if err != types.OrderError_ORDER_AMEND_FAILURE {
		t.Log(err)
	}

	assert.Equal(t, types.OrderError_ORDER_AMEND_FAILURE, err)
}

// ProRata mode OFF which is a default config for vega ME
func TestOrderBook_SubmitOrderProRataModeOff(t *testing.T) {
	market := "testOrderbook"
	book := getTestOrderBook(t, market)
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
			Status:      types.Order_STATUS_ACTIVE,
			Type:        types.Order_LIMIT,
			MarketID:    market,
			PartyID:     "A",
			Side:        types.Side_SIDE_SELL,
			Price:       101,
			Size:        100,
			Remaining:   100,
			TimeInForce: types.Order_GTC,
			CreatedAt:   0,
		},
		{
			Status:      types.Order_STATUS_ACTIVE,
			Type:        types.Order_LIMIT,
			MarketID:    market,
			PartyID:     "B",
			Side:        types.Side_SIDE_SELL,
			Price:       101,
			Size:        100,
			Remaining:   100,
			TimeInForce: types.Order_GTC,
			CreatedAt:   0,
		},
		// Side Buy
		{
			Status:      types.Order_STATUS_ACTIVE,
			Type:        types.Order_LIMIT,
			MarketID:    market,
			PartyID:     "C",
			Side:        types.Side_SIDE_BUY,
			Price:       98,
			Size:        100,
			Remaining:   100,
			TimeInForce: types.Order_GTC,
			CreatedAt:   0,
		},
		{
			Status:      types.Order_STATUS_ACTIVE,
			Type:        types.Order_LIMIT,
			MarketID:    market,
			PartyID:     "D",
			Side:        types.Side_SIDE_BUY,
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
			Status:      types.Order_STATUS_ACTIVE,
			Type:        types.Order_LIMIT,
			MarketID:    market,
			PartyID:     "E",
			Side:        types.Side_SIDE_SELL,
			Price:       101,
			Size:        100,
			Remaining:   100,
			TimeInForce: types.Order_GTC,
			CreatedAt:   1,
		},
		// Side Buy
		{
			Status:      types.Order_STATUS_ACTIVE,
			Type:        types.Order_LIMIT,
			MarketID:    market,
			PartyID:     "F",
			Side:        types.Side_SIDE_BUY,
			Price:       99,
			Size:        100,
			Remaining:   100,
			TimeInForce: types.Order_GTC,
			CreatedAt:   1,
		},
	}

	timestamps := []int64{0, 1}
	for _, timestamp := range timestamps {
		for index := range m[timestamp] {
			confirmation, err := book.SubmitOrder(m[timestamp][index])
			// this should not return any errors
			assert.Equal(t, nil, err)
			// this should not generate any trades
			assert.Equal(t, 0, len(confirmation.Trades))
		}
	}

	scenario := []aggressiveOrderScenario{
		{
			// same price level, remaining on the passive
			aggressiveOrder: &types.Order{
				Status:      types.Order_STATUS_ACTIVE,
				Type:        types.Order_LIMIT,
				MarketID:    market,
				PartyID:     "M",
				Side:        types.Side_SIDE_BUY,
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
					Aggressor: types.Side_SIDE_BUY,
				},
			},
			expectedPassiveOrdersAffected: []types.Order{
				{
					Status:      types.Order_STATUS_ACTIVE,
					Type:        types.Order_LIMIT,
					MarketID:    market,
					PartyID:     "A",
					Side:        types.Side_SIDE_SELL,
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
				Status:      types.Order_STATUS_ACTIVE,
				Type:        types.Order_LIMIT,
				MarketID:    market,
				PartyID:     "N",
				Side:        types.Side_SIDE_BUY,
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
					Aggressor: types.Side_SIDE_BUY,
				},
				{
					MarketID:  market,
					Price:     101,
					Size:      100,
					Buyer:     "N",
					Seller:    "E",
					Aggressor: types.Side_SIDE_BUY,
				},
			},
			expectedPassiveOrdersAffected: []types.Order{
				{
					Status:      types.Order_STATUS_ACTIVE,
					Type:        types.Order_LIMIT,
					MarketID:    market,
					PartyID:     "B",
					Side:        types.Side_SIDE_SELL,
					Price:       101,
					Size:        100,
					Remaining:   0,
					TimeInForce: types.Order_GTC,
					CreatedAt:   0,
				},
				{
					Status:      types.Order_STATUS_ACTIVE,
					Type:        types.Order_LIMIT,
					MarketID:    market,
					PartyID:     "E",
					Side:        types.Side_SIDE_SELL,
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
				Status:      types.Order_STATUS_ACTIVE,
				Type:        types.Order_LIMIT,
				MarketID:    market,
				PartyID:     "O",
				Side:        types.Side_SIDE_SELL,
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
					Aggressor: types.Side_SIDE_SELL,
				},
				{
					MarketID:  market,
					Price:     98,
					Size:      100,
					Buyer:     "C",
					Seller:    "O",
					Aggressor: types.Side_SIDE_SELL,
				},
				{
					MarketID:  market,
					Price:     98,
					Size:      50,
					Buyer:     "D",
					Seller:    "O",
					Aggressor: types.Side_SIDE_SELL,
				},
			},
			expectedPassiveOrdersAffected: []types.Order{
				{
					Status:      types.Order_STATUS_ACTIVE,
					Type:        types.Order_LIMIT,
					MarketID:    market,
					PartyID:     "F",
					Side:        types.Side_SIDE_BUY,
					Price:       99,
					Size:        100,
					Remaining:   0,
					TimeInForce: types.Order_GTC,
					CreatedAt:   1,
				},
				{
					Status:      types.Order_STATUS_ACTIVE,
					Type:        types.Order_LIMIT,
					MarketID:    market,
					PartyID:     "C",
					Side:        types.Side_SIDE_BUY,
					Price:       98,
					Size:        100,
					Remaining:   0,
					TimeInForce: types.Order_GTC,
					CreatedAt:   0,
				},
				{
					Status:      types.Order_STATUS_ACTIVE,
					Type:        types.Order_LIMIT,
					MarketID:    market,
					PartyID:     "D",
					Side:        types.Side_SIDE_BUY,
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
				Status:      types.Order_STATUS_ACTIVE,
				Type:        types.Order_LIMIT,
				MarketID:    market,
				PartyID:     "X",
				Side:        types.Side_SIDE_SELL,
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
					Aggressor: types.Side_SIDE_SELL,
				},
			},
			expectedPassiveOrdersAffected: []types.Order{
				{
					Status:      types.Order_STATUS_ACTIVE,
					Type:        types.Order_LIMIT,
					MarketID:    market,
					PartyID:     "D",
					Side:        types.Side_SIDE_BUY,
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
		fmt.Println("-> Aggressive:", confirmationtypes.Order)
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

// Validate that an IOC order that is not fully filled
// is not added to the order book.
func TestOrderBook_PartialFillIOCOrder(t *testing.T) {
	market := "testOrderbook"
	book := getTestOrderBook(t, market)
	defer book.Finish()

	logger := logging.NewTestLogger()
	defer logger.Sync()
	logger.Debug("BEGIN PARTIAL FILL IOC ORDER")

	newOrder := &types.Order{
		Status:      types.Order_STATUS_ACTIVE,
		Type:        types.Order_LIMIT,
		MarketID:    market,
		Id:          "100000",
		Side:        types.Side_SIDE_SELL,
		Price:       100,
		PartyID:     "A",
		Size:        100,
		Remaining:   100,
		TimeInForce: types.Order_GTC,
		CreatedAt:   10,
	}

	confirmation, err := book.SubmitOrder(newOrder)
	if err != nil {
		t.Log(err)
	}

	assert.Equal(t, nil, err)
	assert.NotNil(t, confirmation)
	assert.Equal(t, "100000", confirmation.Order.Id)
	assert.Equal(t, 0, len(confirmation.Trades))

	iocOrder := &types.Order{
		Status:      types.Order_STATUS_ACTIVE,
		Type:        types.Order_LIMIT,
		MarketID:    market,
		Id:          "100001",
		Side:        types.Side_SIDE_BUY,
		Price:       100,
		PartyID:     "B",
		Size:        20,
		Remaining:   20,
		TimeInForce: types.Order_IOC,
		CreatedAt:   10,
	}
	confirmation, err = book.SubmitOrder(iocOrder)
	if err != nil {
		t.Log(err)
	}

	assert.Equal(t, nil, err)
	assert.NotNil(t, confirmation)
	assert.Equal(t, "100001", confirmation.Order.Id)
	assert.Equal(t, 1, len(confirmation.Trades))

	// Check to see if the order still exists (it should not)
	nonorder, err := book.GetOrderByID("100001")
	assert.Equal(t, types.OrderError_INVALID_ORDER_ID, err)
	assert.Equal(t, (*types.Order)(nil), nonorder)
}
