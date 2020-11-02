package matching_test

import (
	"encoding/hex"
	"fmt"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/matching"
	types "code.vegaprotocol.io/vega/proto"
	"code.vegaprotocol.io/vega/vegatime"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	tob.OrderBook = matching.NewOrderBook(tob.log, matching.NewDefaultConfig(), market, markPrice, false)

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

func TestOrderBook_CancelBulk(t *testing.T) {
	t.Run("Cancel all order for a party", cancelAllOrderForAParty)
	t.Run("Get all order for a party", getAllOrderForAParty)
	t.Run("Party with no order cancel nothing", partyWithNoOrderCancelNothing)
}

func TestHash(t *testing.T) {
	market := "testMarket"
	book := getTestOrderBook(t, market)
	defer book.Finish()

	orders := []*types.Order{
		{
			Id:          "1111111111111111111111",
			Status:      types.Order_STATUS_ACTIVE,
			Type:        types.Order_TYPE_LIMIT,
			MarketID:    market,
			PartyID:     "A",
			Side:        types.Side_SIDE_BUY,
			Price:       10,
			Size:        1,
			Remaining:   1,
			TimeInForce: types.Order_TIF_GTC,
		},
		{
			Id:          "2222222222222222222222",
			Status:      types.Order_STATUS_ACTIVE,
			Type:        types.Order_TYPE_LIMIT,
			MarketID:    market,
			PartyID:     "A",
			Side:        types.Side_SIDE_BUY,
			Price:       30,
			Size:        5,
			Remaining:   5,
			TimeInForce: types.Order_TIF_GTC,
		},
		{
			Id:          "3333333333333333333333",
			Status:      types.Order_STATUS_ACTIVE,
			Type:        types.Order_TYPE_LIMIT,
			MarketID:    market,
			PartyID:     "B",
			Side:        types.Side_SIDE_SELL,
			Price:       200,
			Size:        1,
			Remaining:   1,
			TimeInForce: types.Order_TIF_GTC,
		},
		{
			Id:          "4444444444444444444444",
			Status:      types.Order_STATUS_ACTIVE,
			Type:        types.Order_TYPE_LIMIT,
			MarketID:    market,
			PartyID:     "A",
			Side:        types.Side_SIDE_SELL,
			Price:       400,
			Size:        10,
			Remaining:   10,
			TimeInForce: types.Order_TIF_GTC,
		},
	}

	for _, o := range orders {
		_, err := book.SubmitOrder(o)
		assert.NoError(t, err)
	}

	hash := book.Hash()
	require.Equal(t,
		"ff7190c637406cf6907a73f8eab20c0465e3ee0f471ca95f739aa5f4e4192a8e",
		hex.EncodeToString(hash),
		"It should match against the known hash",
	)
	// compute the hash 100 times for determinism verification
	for i := 0; i < 100; i++ {
		got := book.Hash()
		require.Equal(t, hash, got)
	}
}

func cancelAllOrderForAParty(t *testing.T) {
	market := "testMarket"
	book := getTestOrderBook(t, market)
	defer book.Finish()
	orders := []*types.Order{
		{
			Id:          "1111111111111111111111",
			Status:      types.Order_STATUS_ACTIVE,
			Type:        types.Order_TYPE_LIMIT,
			MarketID:    market,
			PartyID:     "A",
			Side:        types.Side_SIDE_BUY,
			Price:       100,
			Size:        1,
			Remaining:   1,
			TimeInForce: types.Order_TIF_GTC,
		},
		{
			Id:          "2222222222222222222222",
			Status:      types.Order_STATUS_ACTIVE,
			Type:        types.Order_TYPE_LIMIT,
			MarketID:    market,
			PartyID:     "A",
			Side:        types.Side_SIDE_BUY,
			Price:       300,
			Size:        5,
			Remaining:   5,
			TimeInForce: types.Order_TIF_GTC,
		},
		{
			Id:          "3333333333333333333333",
			Status:      types.Order_STATUS_ACTIVE,
			Type:        types.Order_TYPE_LIMIT,
			MarketID:    market,
			PartyID:     "B",
			Side:        types.Side_SIDE_BUY,
			Price:       200,
			Size:        1,
			Remaining:   1,
			TimeInForce: types.Order_TIF_GTC,
		},
		{
			Id:          "4444444444444444444444",
			Status:      types.Order_STATUS_ACTIVE,
			Type:        types.Order_TYPE_LIMIT,
			MarketID:    market,
			PartyID:     "A",
			Side:        types.Side_SIDE_BUY,
			Price:       300,
			Size:        10,
			Remaining:   10,
			TimeInForce: types.Order_TIF_GTC,
		},
	}
	for _, o := range orders {
		confirm, err := book.SubmitOrder(o)
		assert.NoError(t, err)
		assert.Equal(t, 0, len(confirm.Trades))
	}
	confs, err := book.CancelAllOrders("A")
	assert.NoError(t, err)
	assert.Len(t, confs, 3)
	expectedIDs := map[string]struct{}{
		"1111111111111111111111": {},
		"2222222222222222222222": {},
		"4444444444444444444444": {},
	}
	for _, conf := range confs {
		if _, ok := expectedIDs[conf.Order.Id]; ok {
			delete(expectedIDs, conf.Order.Id)
		} else {
			t.Fatalf("unexpected order has been cancelled %v", conf.Order)
		}
	}
}

func getAllOrderForAParty(t *testing.T) {
	market := "testMarket"
	book := getTestOrderBook(t, market)
	defer book.Finish()
	orders := []*types.Order{
		{
			Id:          "1111111111111111111111",
			Status:      types.Order_STATUS_ACTIVE,
			Type:        types.Order_TYPE_LIMIT,
			MarketID:    market,
			PartyID:     "A",
			Side:        types.Side_SIDE_BUY,
			Price:       100,
			Size:        1,
			Remaining:   1,
			TimeInForce: types.Order_TIF_GTC,
		},
		{
			Id:          "2222222222222222222222",
			Status:      types.Order_STATUS_ACTIVE,
			Type:        types.Order_TYPE_LIMIT,
			MarketID:    market,
			PartyID:     "A",
			Side:        types.Side_SIDE_BUY,
			Price:       300,
			Size:        5,
			Remaining:   5,
			TimeInForce: types.Order_TIF_GTC,
		},
		{
			Id:          "3333333333333333333333",
			Status:      types.Order_STATUS_ACTIVE,
			Type:        types.Order_TYPE_LIMIT,
			MarketID:    market,
			PartyID:     "B",
			Side:        types.Side_SIDE_BUY,
			Price:       200,
			Size:        1,
			Remaining:   1,
			TimeInForce: types.Order_TIF_GTC,
		},
		{
			Id:          "4444444444444444444444",
			Status:      types.Order_STATUS_ACTIVE,
			Type:        types.Order_TYPE_LIMIT,
			MarketID:    market,
			PartyID:     "A",
			Side:        types.Side_SIDE_BUY,
			Price:       300,
			Size:        10,
			Remaining:   10,
			TimeInForce: types.Order_TIF_GTC,
		},
	}
	for _, o := range orders {
		confirm, err := book.SubmitOrder(o)
		assert.NoError(t, err)
		assert.Equal(t, 0, len(confirm.Trades))
	}
	ordersLs := book.GetOrdersPerParty("A")
	assert.Len(t, ordersLs, 3)
	expectedIDs := map[string]struct{}{
		"1111111111111111111111": {},
		"2222222222222222222222": {},
		"4444444444444444444444": {},
	}
	for _, o := range ordersLs {
		if _, ok := expectedIDs[o.Id]; ok {
			delete(expectedIDs, o.Id)
		} else {
			t.Fatalf("unexpected order has been cancelled %v", o)
		}
	}
}

func partyWithNoOrderCancelNothing(t *testing.T) {
	market := "testMarket"
	book := getTestOrderBook(t, market)
	defer book.Finish()
	orders := []*types.Order{
		{
			Id:          "1111111111111111111111",
			Status:      types.Order_STATUS_ACTIVE,
			Type:        types.Order_TYPE_LIMIT,
			MarketID:    market,
			PartyID:     "A",
			Side:        types.Side_SIDE_BUY,
			Price:       100,
			Size:        1,
			Remaining:   1,
			TimeInForce: types.Order_TIF_GTC,
		},
		{
			Id:          "2222222222222222222222",
			Status:      types.Order_STATUS_ACTIVE,
			Type:        types.Order_TYPE_LIMIT,
			MarketID:    market,
			PartyID:     "A",
			Side:        types.Side_SIDE_BUY,
			Price:       300,
			Size:        5,
			Remaining:   5,
			TimeInForce: types.Order_TIF_GTC,
		},
		{
			Id:          "3333333333333333333333",
			Status:      types.Order_STATUS_ACTIVE,
			Type:        types.Order_TYPE_LIMIT,
			MarketID:    market,
			PartyID:     "B",
			Side:        types.Side_SIDE_BUY,
			Price:       200,
			Size:        1,
			Remaining:   1,
			TimeInForce: types.Order_TIF_GTC,
		},
		{
			Id:          "4444444444444444444444",
			Status:      types.Order_STATUS_ACTIVE,
			Type:        types.Order_TYPE_LIMIT,
			MarketID:    market,
			PartyID:     "A",
			Side:        types.Side_SIDE_BUY,
			Price:       300,
			Size:        10,
			Remaining:   10,
			TimeInForce: types.Order_TIF_GTC,
		},
	}
	for _, o := range orders {
		confirm, err := book.SubmitOrder(o)
		assert.NoError(t, err)
		assert.Equal(t, 0, len(confirm.Trades))
	}
	ordersLs := book.GetOrdersPerParty("X")
	assert.Len(t, ordersLs, 0)
}

func testBestBidPriceAndVolume(t *testing.T) {
	market := "testMarket"
	book := getTestOrderBook(t, market)
	defer book.Finish()
	// 3 orders of size 1, 3 different prices
	orders := []*types.Order{
		{
			Status:      types.Order_STATUS_ACTIVE,
			Type:        types.Order_TYPE_LIMIT,
			MarketID:    market,
			PartyID:     "A",
			Side:        types.Side_SIDE_BUY,
			Price:       100,
			Size:        1,
			Remaining:   1,
			TimeInForce: types.Order_TIF_GTC,
		},
		{
			Status:      types.Order_STATUS_ACTIVE,
			Type:        types.Order_TYPE_LIMIT,
			MarketID:    market,
			PartyID:     "B",
			Side:        types.Side_SIDE_BUY,
			Price:       300,
			Size:        5,
			Remaining:   5,
			TimeInForce: types.Order_TIF_GTC,
		},
		{
			Status:      types.Order_STATUS_ACTIVE,
			Type:        types.Order_TYPE_LIMIT,
			MarketID:    market,
			PartyID:     "B",
			Side:        types.Side_SIDE_BUY,
			Price:       200,
			Size:        1,
			Remaining:   1,
			TimeInForce: types.Order_TIF_GTC,
		},
		{
			Status:      types.Order_STATUS_ACTIVE,
			Type:        types.Order_TYPE_LIMIT,
			MarketID:    market,
			PartyID:     "d",
			Side:        types.Side_SIDE_BUY,
			Price:       300,
			Size:        10,
			Remaining:   10,
			TimeInForce: types.Order_TIF_GTC,
		},
	}
	for _, o := range orders {
		trades, getErr := book.GetTrades(o)
		assert.NoError(t, getErr)
		confirm, err := book.SubmitOrder(o)
		assert.NoError(t, err)
		assert.Equal(t, 0, len(confirm.Trades))
		assert.Equal(t, len(confirm.Trades), len(trades))
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
			Type:        types.Order_TYPE_LIMIT,
			MarketID:    market,
			PartyID:     "A",
			Side:        types.Side_SIDE_SELL,
			Price:       100,
			Size:        1,
			Remaining:   1,
			TimeInForce: types.Order_TIF_GTC,
		},
		{
			Status:      types.Order_STATUS_ACTIVE,
			Type:        types.Order_TYPE_LIMIT,
			MarketID:    market,
			PartyID:     "B",
			Side:        types.Side_SIDE_SELL,
			Price:       10,
			Size:        5,
			Remaining:   5,
			TimeInForce: types.Order_TIF_GTC,
		},
		{
			Status:      types.Order_STATUS_ACTIVE,
			Type:        types.Order_TYPE_LIMIT,
			MarketID:    market,
			PartyID:     "B",
			Side:        types.Side_SIDE_SELL,
			Price:       200,
			Size:        1,
			Remaining:   1,
			TimeInForce: types.Order_TIF_GTC,
		},
		{
			Status:      types.Order_STATUS_ACTIVE,
			Type:        types.Order_TYPE_LIMIT,
			MarketID:    market,
			PartyID:     "d",
			Side:        types.Side_SIDE_SELL,
			Price:       10,
			Size:        10,
			Remaining:   10,
			TimeInForce: types.Order_TIF_GTC,
		},
	}
	for _, o := range orders {
		trades, getErr := book.GetTrades(o)
		assert.NoError(t, getErr)
		confirm, err := book.SubmitOrder(o)
		assert.NoError(t, err)
		assert.Equal(t, 0, len(confirm.Trades))
		assert.Equal(t, len(trades), len(confirm.Trades))
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
			Type:        types.Order_TYPE_LIMIT,
			MarketID:    market,
			PartyID:     "A",
			Side:        types.Side_SIDE_BUY,
			Price:       100,
			Size:        1,
			Remaining:   1,
			TimeInForce: types.Order_TIF_GTC,
			CreatedAt:   0,
		},
		{
			Status:      types.Order_STATUS_ACTIVE,
			Type:        types.Order_TYPE_LIMIT,
			MarketID:    market,
			PartyID:     "B",
			Side:        types.Side_SIDE_BUY,
			Price:       110,
			Size:        1,
			Remaining:   1,
			TimeInForce: types.Order_TIF_GTC,
			CreatedAt:   0,
		},
	}
	for _, o := range orders {
		trades, getErr := book.GetTrades(o)
		assert.NoError(t, getErr)
		confirm, err := book.SubmitOrder(o)
		assert.NoError(t, err)
		assert.Equal(t, 0, len(confirm.Trades))
		assert.Equal(t, len(trades), len(confirm.Trades))
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
			Type:        types.Order_TYPE_LIMIT,
			MarketID:    market,
			PartyID:     "A",
			Side:        types.Side_SIDE_SELL,
			Price:       100,
			Size:        1,
			Remaining:   1,
			TimeInForce: types.Order_TIF_GTC,
			CreatedAt:   0,
		},
		{
			Status:      types.Order_STATUS_ACTIVE,
			Type:        types.Order_TYPE_LIMIT,
			MarketID:    market,
			PartyID:     "B",
			Side:        types.Side_SIDE_SELL,
			Price:       110,
			Size:        1,
			Remaining:   1,
			TimeInForce: types.Order_TIF_GTC,
			CreatedAt:   0,
		},
	}
	for _, o := range orders {
		trades, getErr := book.GetTrades(o)
		assert.NoError(t, getErr)
		confirm, err := book.SubmitOrder(o)
		assert.NoError(t, err)
		assert.Equal(t, 0, len(confirm.Trades))
		assert.Equal(t, len(trades), len(confirm.Trades))
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
			Type:        types.Order_TYPE_LIMIT,
			MarketID:    market,
			PartyID:     "A",
			Side:        types.Side_SIDE_BUY,
			Price:       100,
			Size:        1,
			Remaining:   1,
			TimeInForce: types.Order_TIF_GTC,
			CreatedAt:   0,
		},
		{
			Status:      types.Order_STATUS_ACTIVE,
			Type:        types.Order_TYPE_LIMIT,
			MarketID:    market,
			PartyID:     "B",
			Side:        types.Side_SIDE_BUY,
			Price:       110,
			Size:        1,
			Remaining:   1,
			TimeInForce: types.Order_TIF_GTC,
			CreatedAt:   0,
		},
		{
			Status:      types.Order_STATUS_ACTIVE,
			Type:        types.Order_TYPE_LIMIT,
			MarketID:    market,
			PartyID:     "C",
			Side:        types.Side_SIDE_BUY,
			Price:       120,
			Size:        1,
			Remaining:   1,
			TimeInForce: types.Order_TIF_GTC,
			CreatedAt:   0,
		},
	}
	for _, o := range orders {
		trades, getErr := book.GetTrades(o)
		assert.NoError(t, getErr)
		confirm, err := book.SubmitOrder(o)
		assert.NoError(t, err)
		assert.Equal(t, 0, len(confirm.Trades))
		assert.Equal(t, len(trades), len(confirm.Trades))
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
			Type:        types.Order_TYPE_LIMIT,
			MarketID:    market,
			PartyID:     "A",
			Side:        types.Side_SIDE_SELL,
			Price:       100,
			Size:        1,
			Remaining:   1,
			TimeInForce: types.Order_TIF_GTC,
			CreatedAt:   0,
		},
		{
			Status:      types.Order_STATUS_ACTIVE,
			Type:        types.Order_TYPE_LIMIT,
			MarketID:    market,
			PartyID:     "B",
			Side:        types.Side_SIDE_SELL,
			Price:       110,
			Size:        1,
			Remaining:   1,
			TimeInForce: types.Order_TIF_GTC,
			CreatedAt:   0,
		},
		{
			Status:      types.Order_STATUS_ACTIVE,
			Type:        types.Order_TYPE_LIMIT,
			MarketID:    market,
			PartyID:     "C",
			Side:        types.Side_SIDE_SELL,
			Price:       120,
			Size:        1,
			Remaining:   1,
			TimeInForce: types.Order_TIF_GTC,
			CreatedAt:   0,
		},
	}
	for _, o := range orders {
		trades, getErr := book.GetTrades(o)
		assert.NoError(t, getErr)
		confirm, err := book.SubmitOrder(o)
		assert.NoError(t, err)
		assert.Equal(t, 0, len(confirm.Trades))
		assert.Equal(t, len(trades), len(confirm.Trades))
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
		Type:        types.Order_TYPE_LIMIT,
		MarketID:    market,
		PartyID:     party,
		Side:        types.Side_SIDE_SELL,
		Price:       1,
		Size:        100,
		Remaining:   100,
		TimeInForce: types.Order_TIF_GTC,
		CreatedAt:   currentTimestamp,
		Id:          "v0000000000000-0000001",
	}
	order2 := types.Order{
		Status:      types.Order_STATUS_ACTIVE,
		Type:        types.Order_TYPE_LIMIT,
		MarketID:    market,
		PartyID:     party,
		Side:        types.Side_SIDE_SELL,
		Price:       1,
		Size:        100,
		Remaining:   1, // use a wrong remaining here to get the order from the book
		TimeInForce: types.Order_TIF_GTC,
		CreatedAt:   currentTimestamp,
		Id:          "v0000000000000-0000001",
	}

	trades, getErr := book.GetTrades(&order1)
	assert.NoError(t, getErr)
	confirm, err := book.SubmitOrder(&order1)
	assert.Equal(t, err, nil)
	assert.Equal(t, len(trades), len(confirm.Trades))

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
		Type:        types.Order_TYPE_LIMIT,
		MarketID:    market,
		PartyID:     party,
		Side:        types.Side_SIDE_SELL,
		Price:       1,
		Size:        1,
		Remaining:   1,
		TimeInForce: types.Order_TIF_GTT,
		CreatedAt:   currentTimestamp,
		ExpiresAt:   someTimeLater,
		Id:          "1",
	}
	trades, getErr := book.GetTrades(order1)
	assert.NoError(t, getErr)
	confirm, err := book.SubmitOrder(order1)
	assert.Equal(t, err, nil)
	assert.Equal(t, len(trades), len(confirm.Trades))

	order2 := &types.Order{
		Status:      types.Order_STATUS_ACTIVE,
		Type:        types.Order_TYPE_LIMIT,
		MarketID:    market,
		PartyID:     party,
		Side:        types.Side_SIDE_SELL,
		Price:       3298,
		Size:        99,
		Remaining:   99,
		TimeInForce: types.Order_TIF_GTT,
		CreatedAt:   currentTimestamp,
		ExpiresAt:   someTimeLater + 1,
		Id:          "2",
	}
	trades, getErr = book.GetTrades(order2)
	assert.NoError(t, getErr)
	confirm, err = book.SubmitOrder(order2)
	assert.Equal(t, err, nil)
	assert.Equal(t, len(trades), len(confirm.Trades))

	order3 := &types.Order{
		Status:      types.Order_STATUS_ACTIVE,
		Type:        types.Order_TYPE_LIMIT,
		MarketID:    market,
		PartyID:     party,
		Side:        types.Side_SIDE_SELL,
		Price:       771,
		Size:        19,
		Remaining:   19,
		TimeInForce: types.Order_TIF_GTT,
		CreatedAt:   currentTimestamp,
		ExpiresAt:   someTimeLater,
		Id:          "3",
	}
	trades, getErr = book.GetTrades(order3)
	assert.NoError(t, getErr)
	confirm, err = book.SubmitOrder(order3)
	assert.Equal(t, err, nil)
	assert.Equal(t, len(trades), len(confirm.Trades))

	order4 := &types.Order{
		Status:      types.Order_STATUS_ACTIVE,
		Type:        types.Order_TYPE_LIMIT,
		MarketID:    market,
		PartyID:     party,
		Side:        types.Side_SIDE_SELL,
		Price:       1000,
		Size:        7,
		Remaining:   7,
		TimeInForce: types.Order_TIF_GTC,
		CreatedAt:   currentTimestamp,
		Id:          "4",
	}
	trades, getErr = book.GetTrades(order4)
	assert.NoError(t, getErr)
	confirm, err = book.SubmitOrder(order4)
	assert.Equal(t, err, nil)
	assert.Equal(t, len(trades), len(confirm.Trades))

	order5 := &types.Order{
		Status:      types.Order_STATUS_ACTIVE,
		Type:        types.Order_TYPE_LIMIT,
		MarketID:    market,
		PartyID:     party,
		Side:        types.Side_SIDE_SELL,
		Price:       199,
		Size:        99999,
		Remaining:   99999,
		TimeInForce: types.Order_TIF_GTT,
		CreatedAt:   currentTimestamp,
		ExpiresAt:   someTimeLater,
		Id:          "5",
	}

	trades, getErr = book.GetTrades(order5)
	assert.NoError(t, getErr)
	confirm, err = book.SubmitOrder(order5)
	assert.Equal(t, err, nil)
	assert.Equal(t, len(trades), len(confirm.Trades))

	order6 := &types.Order{
		Status:      types.Order_STATUS_ACTIVE,
		Type:        types.Order_TYPE_LIMIT,
		MarketID:    market,
		PartyID:     party,
		Side:        types.Side_SIDE_SELL,
		Price:       100,
		Size:        100,
		Remaining:   100,
		TimeInForce: types.Order_TIF_GTC,
		CreatedAt:   currentTimestamp,
		Id:          "6",
	}
	trades, getErr = book.GetTrades(order6)
	assert.NoError(t, getErr)
	confirm, err = book.SubmitOrder(order6)
	assert.Equal(t, err, nil)
	assert.Equal(t, len(trades), len(confirm.Trades))

	order7 := &types.Order{
		Status:      types.Order_STATUS_ACTIVE,
		Type:        types.Order_TYPE_LIMIT,
		MarketID:    market,
		PartyID:     party,
		Side:        types.Side_SIDE_SELL,
		Price:       41,
		Size:        9999,
		Remaining:   9999,
		TimeInForce: types.Order_TIF_GTT,
		CreatedAt:   currentTimestamp,
		ExpiresAt:   someTimeLater + 9999,
		Id:          "7",
	}
	trades, getErr = book.GetTrades(order7)
	assert.NoError(t, getErr)
	confirm, err = book.SubmitOrder(order7)
	assert.Equal(t, err, nil)
	assert.Equal(t, len(trades), len(confirm.Trades))

	order8 := &types.Order{
		Status:      types.Order_STATUS_ACTIVE,
		Type:        types.Order_TYPE_LIMIT,
		MarketID:    market,
		PartyID:     party,
		Side:        types.Side_SIDE_SELL,
		Price:       1,
		Size:        1,
		Remaining:   1,
		TimeInForce: types.Order_TIF_GTT,
		CreatedAt:   currentTimestamp,
		ExpiresAt:   someTimeLater - 9999,
		Id:          "8",
	}
	trades, getErr = book.GetTrades(order8)
	assert.NoError(t, getErr)
	confirm, err = book.SubmitOrder(order8)
	assert.Equal(t, err, nil)
	assert.Equal(t, len(trades), len(confirm.Trades))

	order9 := &types.Order{
		Status:      types.Order_STATUS_ACTIVE,
		Type:        types.Order_TYPE_LIMIT,
		MarketID:    market,
		PartyID:     party,
		Side:        types.Side_SIDE_SELL,
		Price:       65,
		Size:        12,
		Remaining:   12,
		TimeInForce: types.Order_TIF_GTC,
		CreatedAt:   currentTimestamp,
		Id:          "9",
	}
	trades, getErr = book.GetTrades(order9)
	assert.NoError(t, getErr)
	confirm, err = book.SubmitOrder(order9)
	assert.Equal(t, err, nil)
	assert.Equal(t, len(trades), len(confirm.Trades))

	order10 := &types.Order{
		Status:      types.Order_STATUS_ACTIVE,
		Type:        types.Order_TYPE_LIMIT,
		MarketID:    market,
		PartyID:     party,
		Side:        types.Side_SIDE_SELL,
		Price:       1,
		Size:        1,
		Remaining:   1,
		TimeInForce: types.Order_TIF_GTT,
		CreatedAt:   currentTimestamp,
		ExpiresAt:   someTimeLater - 1,
		Id:          "10",
	}
	trades, getErr = book.GetTrades(order10)
	assert.NoError(t, getErr)
	confirm, err = book.SubmitOrder(order10)
	assert.Equal(t, err, nil)
	assert.Equal(t, len(trades), len(confirm.Trades))

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
		Type:      types.Order_TYPE_LIMIT,
		Id:        "timestamporderID",
		MarketID:  market,
		PartyID:   "A",
		CreatedAt: 10,
		Side:      types.Side_SIDE_BUY,
		Size:      1,
		Remaining: 1,
	}
	trades, getErr := book.GetTrades(&timeStampOrder)
	assert.NoError(t, getErr)
	confirm, err := book.SubmitOrder(&timeStampOrder)
	assert.NoError(t, err)
	assert.Equal(t, len(trades), len(confirm.Trades))
	// cancel order again, just so we set the timestamp as expected
	book.CancelOrder(&timeStampOrder)

	invalidRemainingSizeOrdertypes := &types.Order{
		Status:      types.Order_STATUS_ACTIVE,
		Type:        types.Order_TYPE_LIMIT,
		MarketID:    market,
		PartyID:     "A",
		Side:        types.Side_SIDE_SELL,
		Price:       100,
		Size:        100,
		Remaining:   300,
		TimeInForce: types.Order_TIF_GTC,
		CreatedAt:   10,
		Id:          "id-number-one",
	}
	trades, getErr = book.GetTrades(invalidRemainingSizeOrdertypes)
	_, err = book.SubmitOrder(invalidRemainingSizeOrdertypes)
	assert.Equal(t, err, getErr)
	assert.Equal(t, types.OrderError_ORDER_ERROR_INVALID_REMAINING_SIZE, err)
	assert.Equal(t, 0, len(trades))
}

func TestOrderBook_DeleteOrder(t *testing.T) {
	market := "testOrderbook"
	book := getTestOrderBook(t, market)
	defer book.Finish()

	newOrder := &types.Order{
		Status:      types.Order_STATUS_ACTIVE,
		Type:        types.Order_TYPE_LIMIT,
		MarketID:    market,
		PartyID:     "A",
		Side:        types.Side_SIDE_SELL,
		Price:       101,
		Size:        100,
		Remaining:   100,
		TimeInForce: types.Order_TIF_GTC,
		CreatedAt:   0,
	}

	trades, err := book.GetTrades(newOrder)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(trades))
	book.SubmitOrder(newOrder)

	if _, err := book.DeleteOrder(newOrder); err != nil {
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
		Type:        types.Order_TYPE_LIMIT,
		MarketID:    "invalid",
		PartyID:     "A",
		Side:        types.Side_SIDE_SELL,
		Price:       101,
		Size:        100,
		Remaining:   100,
		TimeInForce: types.Order_TIF_GTC,
		CreatedAt:   0,
		Id:          fmt.Sprintf("V%010d-%010d", 1, 1),
	}

	trades, getErr := book.GetTrades(newOrder)
	assert.Error(t, getErr)
	assert.Equal(t, 0, len(trades))
	_, err := book.SubmitOrder(newOrder)
	if err != nil {
		fmt.Println(err)
	}

	assert.Equal(t, types.OrderError_ORDER_ERROR_INVALID_MARKET_ID, err)
	assert.Equal(t, getErr, err)

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
		Type:        types.Order_TYPE_LIMIT,
		MarketID:    market,
		PartyID:     "A",
		Side:        types.Side_SIDE_SELL,
		Price:       101,
		Size:        100,
		Remaining:   100,
		TimeInForce: types.Order_TIF_GTC,
		CreatedAt:   0,
		Id:          id,
	}

	trades, getErr := book.GetTrades(newOrder)
	assert.NoError(t, getErr)
	confirmation, err := book.SubmitOrder(newOrder)
	assert.NoError(t, err)
	orderAdded := confirmation.Order
	assert.Equal(t, len(trades), len(confirmation.Trades))

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
		Type:        types.Order_TYPE_LIMIT,
		MarketID:    market,
		PartyID:     "A",
		Side:        types.Side_SIDE_BUY,
		Price:       101,
		Size:        100,
		Remaining:   100,
		TimeInForce: types.Order_TIF_GTC,
		CreatedAt:   0,
		Id:          id,
	}

	trades, getErr := book.GetTrades(newOrder)
	assert.NoError(t, getErr)
	confirmation, err := book.SubmitOrder(newOrder)
	assert.NoError(t, err)
	assert.Equal(t, len(trades), len(confirmation.Trades))
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
		Type:        types.Order_TYPE_LIMIT,
		MarketID:    market,
		PartyID:     "A",
		Side:        types.Side_SIDE_BUY,
		Price:       101,
		Size:        100,
		Remaining:   100,
		TimeInForce: types.Order_TIF_GTC,
		CreatedAt:   0,
		Id:          id,
	}

	trades, getErr := book.GetTrades(newOrder)
	assert.NoError(t, getErr)
	confirmation, err := book.SubmitOrder(newOrder)
	assert.NotNil(t, confirmation, "submit order should succeed")
	assert.NoError(t, err, "submit order should succeed")
	orderAdded := confirmation.Order
	assert.NotNil(t, orderAdded, "submitted order is expected to be valid")
	assert.Equal(t, len(trades), len(confirmation.Trades))

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
		Type:      types.Order_TYPE_LIMIT,
		MarketID:  market,
		Id:        fmt.Sprintf("V%010d-%010d", 1, 1),
		PartyID:   "A",
		Size:      100,
		Remaining: 100,
	}

	trades, getErr := book.GetTrades(newOrder)
	assert.NoError(t, getErr)
	confirmation, err := book.SubmitOrder(newOrder)
	assert.NoError(t, err)
	orderAdded := confirmation.Order
	assert.Equal(t, len(trades), len(confirmation.Trades))

	orderAdded.MarketID = "invalid" // Bad market, malformed?

	_, err = book.CancelOrder(orderAdded)
	if err != nil {
		fmt.Println(err)
	}

	assert.Equal(t, types.OrderError_ORDER_ERROR_INVALID_MARKET_ID, err)
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
		Type:      types.Order_TYPE_LIMIT,
		MarketID:  market,
		Id:        "id",
		PartyID:   "A",
		Size:      100,
		Remaining: 100,
	}

	trades, getErr := book.GetTrades(newOrder)
	assert.NoError(t, getErr)
	confirmation, err := book.SubmitOrder(newOrder)
	assert.NoError(t, err)
	orderAdded := confirmation.Order
	assert.Equal(t, len(trades), len(confirmation.Trades))

	_, err = book.CancelOrder(orderAdded)
	if err != nil {
		logger.Debug("error cancelling order", logging.Error(err))
	}

	assert.Equal(t, types.OrderError_ORDER_ERROR_INVALID_ORDER_ID, err)
}

func expectTrade(t *testing.T, expectedTrade, trade *types.Trade) {
	// run asserts for protocol trade data
	assert.Equal(t, expectedTrade.Type, trade.Type, "invalid trade type")
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
		Type:        types.Order_TYPE_LIMIT,
		MarketID:    market,
		Id:          "123456",
		PartyID:     "A",
		Side:        types.Side_SIDE_BUY,
		Price:       100,
		Size:        200,
		Remaining:   200,
		TimeInForce: types.Order_TIF_GTC,
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
		Type:        types.Order_TYPE_LIMIT,
		MarketID:    market,
		Id:          "123456",
		PartyID:     "A",
		Side:        types.Side_SIDE_BUY,
		Price:       100,
		Size:        200,
		Remaining:   200,
		TimeInForce: types.Order_TIF_GTC,
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
		Type:        types.Order_TYPE_LIMIT,
		MarketID:    market,
		Id:          "123456",
		PartyID:     "A",
		Side:        types.Side_SIDE_BUY,
		Price:       100,
		Size:        200,
		Remaining:   200,
		TimeInForce: types.Order_TIF_GTC,
	}

	trades, getErr := book.GetTrades(newOrder)
	assert.NoError(t, getErr)
	confirmation, err := book.SubmitOrder(newOrder)

	assert.Equal(t, nil, err)
	assert.NotNil(t, confirmation)
	assert.Equal(t, "123456", confirmation.Order.Id)
	assert.Equal(t, 0, len(confirmation.Trades))
	assert.Equal(t, len(trades), len(confirmation.Trades))

	editedOrder := &types.Order{
		Status:      types.Order_STATUS_ACTIVE,
		Type:        types.Order_TYPE_LIMIT,
		MarketID:    market,
		PartyID:     "A",
		Id:          "123456",
		Side:        types.Side_SIDE_SELL,
		Price:       100,
		Size:        100,
		Remaining:   200,
		TimeInForce: types.Order_TIF_GTC,
	}
	err = book.AmendOrder(newOrder, editedOrder)
	if err != types.OrderError_ORDER_ERROR_INVALID_REMAINING_SIZE {
		t.Log(err)
	}

	assert.Equal(t, types.OrderError_ORDER_ERROR_INVALID_REMAINING_SIZE, err)
}

func TestOrderBook_AmendOrderInvalidAmend(t *testing.T) {
	market := "testOrderbook"
	book := getTestOrderBook(t, market)
	defer book.Finish()

	newOrder := &types.Order{
		Status:      types.Order_STATUS_ACTIVE,
		Type:        types.Order_TYPE_LIMIT,
		MarketID:    market,
		Id:          "123456",
		Side:        types.Side_SIDE_BUY,
		Price:       100,
		Size:        200,
		Remaining:   200,
		TimeInForce: types.Order_TIF_GTC,
	}

	trades, getErr := book.GetTrades(newOrder)
	confirmation, err := book.SubmitOrder(newOrder)
	if err != nil {
		fmt.Println(err)
	}
	assert.Equal(t, err, getErr)
	assert.Equal(t, 0, len(trades))

	fmt.Printf("confirmation : %+v", confirmation)

	editedOrder := &types.Order{
		Status:      types.Order_STATUS_ACTIVE,
		Type:        types.Order_TYPE_LIMIT,
		MarketID:    market,
		Id:          "123456",
		PartyID:     "A",
		Side:        types.Side_SIDE_SELL,
		Price:       100,
		Size:        200,
		Remaining:   200,
		TimeInForce: types.Order_TIF_GTC,
	}

	err = book.AmendOrder(newOrder, editedOrder)
	if err != types.OrderError_ORDER_ERROR_NOT_FOUND {
		fmt.Println(err)
	}

	assert.Equal(t, types.OrderError_ORDER_ERROR_NOT_FOUND, err)
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
		Type:        types.Order_TYPE_LIMIT,
		MarketID:    market,
		Id:          "123456",
		Side:        types.Side_SIDE_BUY,
		Price:       100,
		PartyID:     "A",
		Size:        200,
		Remaining:   200,
		TimeInForce: types.Order_TIF_GTC,
	}

	trades, getErr := book.GetTrades(newOrder)
	assert.NoError(t, getErr)
	confirmation, err := book.SubmitOrder(newOrder)

	assert.Equal(t, nil, err)
	assert.NotNil(t, confirmation)
	assert.Equal(t, "123456", confirmation.Order.Id)
	assert.Equal(t, 0, len(confirmation.Trades))
	assert.Equal(t, len(trades), len(confirmation.Trades))

	editedOrder := &types.Order{
		Status:      types.Order_STATUS_ACTIVE,
		Type:        types.Order_TYPE_LIMIT,
		MarketID:    market,
		Id:          "123456",
		Side:        types.Side_SIDE_BUY,
		Price:       100,
		PartyID:     "B",
		Size:        200,
		Remaining:   200,
		TimeInForce: types.Order_TIF_GTC,
	}

	err = book.AmendOrder(newOrder, editedOrder)
	if err != types.OrderError_ORDER_ERROR_AMEND_FAILURE {
		t.Log(err)
	}

	assert.Equal(t, types.OrderError_ORDER_ERROR_AMEND_FAILURE, err)
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
		Type:        types.Order_TYPE_LIMIT,
		MarketID:    market,
		Id:          "123456",
		Side:        types.Side_SIDE_BUY,
		Price:       100,
		PartyID:     "A",
		Size:        200,
		Remaining:   200,
		TimeInForce: types.Order_TIF_GTC,
		CreatedAt:   10,
	}

	trades, getErr := book.GetTrades(newOrder)
	assert.NoError(t, getErr)
	confirmation, err := book.SubmitOrder(newOrder)

	assert.Equal(t, nil, err)
	assert.NotNil(t, confirmation)
	assert.Equal(t, "123456", confirmation.Order.Id)
	assert.Equal(t, 0, len(confirmation.Trades))
	assert.Equal(t, len(trades), len(confirmation.Trades))

	editedOrder := &types.Order{
		Status:      types.Order_STATUS_ACTIVE,
		Type:        types.Order_TYPE_LIMIT,
		MarketID:    market,
		Id:          "123456",
		Side:        types.Side_SIDE_BUY,
		Price:       100,
		PartyID:     "A",
		Size:        200,
		Remaining:   200,
		TimeInForce: types.Order_TIF_GTC,
		CreatedAt:   5,
	}

	err = book.AmendOrder(newOrder, editedOrder)
	if err != types.OrderError_ORDER_ERROR_OUT_OF_SEQUENCE {
		t.Log(err)
	}

	assert.Equal(t, types.OrderError_ORDER_ERROR_OUT_OF_SEQUENCE, err)
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
		Type:        types.Order_TYPE_LIMIT,
		MarketID:    market,
		Id:          "123456",
		Side:        types.Side_SIDE_BUY,
		Price:       100,
		PartyID:     "A",
		Size:        200,
		Remaining:   200,
		TimeInForce: types.Order_TIF_GTC,
		CreatedAt:   10,
	}

	trades, getErr := book.GetTrades(newOrder)
	assert.NoError(t, getErr)
	confirmation, err := book.SubmitOrder(newOrder)

	assert.Equal(t, nil, err)
	assert.NotNil(t, confirmation)
	assert.Equal(t, "123456", confirmation.Order.Id)
	assert.Equal(t, 0, len(confirmation.Trades))
	assert.Equal(t, len(trades), len(confirmation.Trades))

	editedOrder := &types.Order{
		Status:      types.Order_STATUS_ACTIVE,
		Type:        types.Order_TYPE_LIMIT,
		MarketID:    market,
		Id:          "123456",
		Side:        types.Side_SIDE_BUY,
		Price:       100,
		PartyID:     "B",
		Size:        300,
		Remaining:   300,
		TimeInForce: types.Order_TIF_GTC,
		CreatedAt:   10,
	}

	err = book.AmendOrder(newOrder, editedOrder)
	if err != types.OrderError_ORDER_ERROR_AMEND_FAILURE {
		t.Log(err)
	}

	assert.Equal(t, types.OrderError_ORDER_ERROR_AMEND_FAILURE, err)
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
			Type:        types.Order_TYPE_LIMIT,
			MarketID:    market,
			PartyID:     "A",
			Side:        types.Side_SIDE_SELL,
			Price:       101,
			Size:        100,
			Remaining:   100,
			TimeInForce: types.Order_TIF_GTC,
			CreatedAt:   0,
		},
		{
			Status:      types.Order_STATUS_ACTIVE,
			Type:        types.Order_TYPE_LIMIT,
			MarketID:    market,
			PartyID:     "B",
			Side:        types.Side_SIDE_SELL,
			Price:       101,
			Size:        100,
			Remaining:   100,
			TimeInForce: types.Order_TIF_GTC,
			CreatedAt:   0,
		},
		// Side Buy
		{
			Status:      types.Order_STATUS_ACTIVE,
			Type:        types.Order_TYPE_LIMIT,
			MarketID:    market,
			PartyID:     "C",
			Side:        types.Side_SIDE_BUY,
			Price:       98,
			Size:        100,
			Remaining:   100,
			TimeInForce: types.Order_TIF_GTC,
			CreatedAt:   0,
		},
		{
			Status:      types.Order_STATUS_ACTIVE,
			Type:        types.Order_TYPE_LIMIT,
			MarketID:    market,
			PartyID:     "D",
			Side:        types.Side_SIDE_BUY,
			Price:       98,
			Size:        100,
			Remaining:   100,
			TimeInForce: types.Order_TIF_GTC,
			CreatedAt:   0,
		},
	}

	// sell and buy orders at timestamp 1
	m[1] = []*types.Order{
		// Side Sell
		{
			Status:      types.Order_STATUS_ACTIVE,
			Type:        types.Order_TYPE_LIMIT,
			MarketID:    market,
			PartyID:     "E",
			Side:        types.Side_SIDE_SELL,
			Price:       101,
			Size:        100,
			Remaining:   100,
			TimeInForce: types.Order_TIF_GTC,
			CreatedAt:   1,
		},
		// Side Buy
		{
			Status:      types.Order_STATUS_ACTIVE,
			Type:        types.Order_TYPE_LIMIT,
			MarketID:    market,
			PartyID:     "F",
			Side:        types.Side_SIDE_BUY,
			Price:       99,
			Size:        100,
			Remaining:   100,
			TimeInForce: types.Order_TIF_GTC,
			CreatedAt:   1,
		},
	}

	timestamps := []int64{0, 1}
	for _, timestamp := range timestamps {
		for _, o := range m[timestamp] {
			trades, getErr := book.GetTrades(o)
			assert.NoError(t, getErr)
			confirmation, err := book.SubmitOrder(o)
			// this should not return any errors
			assert.Equal(t, nil, err)
			// this should not generate any trades
			assert.Equal(t, 0, len(confirmation.Trades))
			assert.Equal(t, len(trades), len(confirmation.Trades))
		}
	}

	scenario := []aggressiveOrderScenario{
		{
			// same price level, remaining on the passive
			aggressiveOrder: &types.Order{
				Status:      types.Order_STATUS_ACTIVE,
				Type:        types.Order_TYPE_LIMIT,
				MarketID:    market,
				PartyID:     "M",
				Side:        types.Side_SIDE_BUY,
				Price:       101,
				Size:        100,
				Remaining:   100,
				TimeInForce: types.Order_TIF_GTC,
				CreatedAt:   3,
			},
			expectedTrades: []types.Trade{
				{
					Type:      types.Trade_TYPE_DEFAULT,
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
					Type:        types.Order_TYPE_LIMIT,
					MarketID:    market,
					PartyID:     "A",
					Side:        types.Side_SIDE_SELL,
					Price:       101,
					Size:        100,
					Remaining:   0,
					TimeInForce: types.Order_TIF_GTC,
					CreatedAt:   0,
				},
			},
		},
		{
			// same price level, remaining on the passive
			aggressiveOrder: &types.Order{
				Status:      types.Order_STATUS_ACTIVE,
				Type:        types.Order_TYPE_LIMIT,
				MarketID:    market,
				PartyID:     "N",
				Side:        types.Side_SIDE_BUY,
				Price:       102,
				Size:        200,
				Remaining:   200,
				TimeInForce: types.Order_TIF_GTC,
				CreatedAt:   4,
			},
			expectedTrades: []types.Trade{
				{
					Type:      types.Trade_TYPE_DEFAULT,
					MarketID:  market,
					Price:     101,
					Size:      100,
					Buyer:     "N",
					Seller:    "B",
					Aggressor: types.Side_SIDE_BUY,
				},
				{
					Type:      types.Trade_TYPE_DEFAULT,
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
					Type:        types.Order_TYPE_LIMIT,
					MarketID:    market,
					PartyID:     "B",
					Side:        types.Side_SIDE_SELL,
					Price:       101,
					Size:        100,
					Remaining:   0,
					TimeInForce: types.Order_TIF_GTC,
					CreatedAt:   0,
				},
				{
					Status:      types.Order_STATUS_ACTIVE,
					Type:        types.Order_TYPE_LIMIT,
					MarketID:    market,
					PartyID:     "E",
					Side:        types.Side_SIDE_SELL,
					Price:       101,
					Size:        100,
					Remaining:   0,
					TimeInForce: types.Order_TIF_GTC,
					CreatedAt:   1,
				},
			},
		},
		{
			// same price level, NO PRORATA
			aggressiveOrder: &types.Order{
				Status:      types.Order_STATUS_ACTIVE,
				Type:        types.Order_TYPE_LIMIT,
				MarketID:    market,
				PartyID:     "O",
				Side:        types.Side_SIDE_SELL,
				Price:       97,
				Size:        250,
				Remaining:   250,
				TimeInForce: types.Order_TIF_GTC,
				CreatedAt:   5,
			},
			expectedTrades: []types.Trade{
				{
					Type:      types.Trade_TYPE_DEFAULT,
					MarketID:  market,
					Price:     99,
					Size:      100,
					Buyer:     "F",
					Seller:    "O",
					Aggressor: types.Side_SIDE_SELL,
				},
				{
					Type:      types.Trade_TYPE_DEFAULT,
					MarketID:  market,
					Price:     98,
					Size:      100,
					Buyer:     "C",
					Seller:    "O",
					Aggressor: types.Side_SIDE_SELL,
				},
				{
					Type:      types.Trade_TYPE_DEFAULT,
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
					Type:        types.Order_TYPE_LIMIT,
					MarketID:    market,
					PartyID:     "F",
					Side:        types.Side_SIDE_BUY,
					Price:       99,
					Size:        100,
					Remaining:   0,
					TimeInForce: types.Order_TIF_GTC,
					CreatedAt:   1,
				},
				{
					Status:      types.Order_STATUS_ACTIVE,
					Type:        types.Order_TYPE_LIMIT,
					MarketID:    market,
					PartyID:     "C",
					Side:        types.Side_SIDE_BUY,
					Price:       98,
					Size:        100,
					Remaining:   0,
					TimeInForce: types.Order_TIF_GTC,
					CreatedAt:   0,
				},
				{
					Status:      types.Order_STATUS_ACTIVE,
					Type:        types.Order_TYPE_LIMIT,
					MarketID:    market,
					PartyID:     "D",
					Side:        types.Side_SIDE_BUY,
					Price:       98,
					Size:        100,
					Remaining:   50,
					TimeInForce: types.Order_TIF_GTC,
					CreatedAt:   0,
				},
			},
		},
		{
			// same price level, NO PRORATA
			aggressiveOrder: &types.Order{
				Status:      types.Order_STATUS_ACTIVE,
				Type:        types.Order_TYPE_LIMIT,
				MarketID:    market,
				PartyID:     "X",
				Side:        types.Side_SIDE_SELL,
				Price:       98,
				Size:        50,
				Remaining:   50,
				TimeInForce: types.Order_TIF_GTC,
				CreatedAt:   6,
			},
			expectedTrades: []types.Trade{
				{
					Type:      types.Trade_TYPE_DEFAULT,
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
					Type:        types.Order_TYPE_LIMIT,
					MarketID:    market,
					PartyID:     "D",
					Side:        types.Side_SIDE_BUY,
					Price:       98,
					Size:        100,
					Remaining:   0,
					TimeInForce: types.Order_TIF_GTC,
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

		trades, getErr := book.GetTrades(s.aggressiveOrder)
		assert.NoError(t, getErr)
		confirmationtypes, err := book.SubmitOrder(s.aggressiveOrder)

		//this should not return any errors
		assert.Equal(t, nil, err)

		//this should not generate any trades
		assert.Equal(t, len(s.expectedTrades), len(confirmationtypes.Trades))
		assert.Equal(t, len(confirmationtypes.Trades), len(trades))

		fmt.Println("CONFIRMATION types:")
		fmt.Println("-> Aggressive:", confirmationtypes.Order)
		fmt.Println("-> Trades :", confirmationtypes.Trades)
		fmt.Println("-> PassiveOrdersAffected:", confirmationtypes.PassiveOrdersAffected)
		fmt.Printf("Scenario: %d / %d \n", i+1, len(scenario))

		// assert.Equal(t, len(s.expectedTrades), len(confirmationtypes.Trades))
		// trades should match expected trades
		for i, exp := range s.expectedTrades {
			expectTrade(t, &exp, confirmationtypes.Trades[i])
			expectTrade(t, &exp, trades[i])
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
		Type:        types.Order_TYPE_LIMIT,
		MarketID:    market,
		Id:          "100000",
		Side:        types.Side_SIDE_SELL,
		Price:       100,
		PartyID:     "A",
		Size:        100,
		Remaining:   100,
		TimeInForce: types.Order_TIF_GTC,
		CreatedAt:   10,
	}

	trades, getErr := book.GetTrades(newOrder)
	assert.NoError(t, getErr)
	confirmation, err := book.SubmitOrder(newOrder)

	assert.Equal(t, nil, err)
	assert.NotNil(t, confirmation)
	assert.Equal(t, "100000", confirmation.Order.Id)
	assert.Equal(t, 0, len(confirmation.Trades))
	assert.Equal(t, len(trades), len(confirmation.Trades))

	iocOrderId := "1000000000000000000000" //Must be 22 characters
	iocOrder := &types.Order{
		Status:      types.Order_STATUS_ACTIVE,
		Type:        types.Order_TYPE_LIMIT,
		MarketID:    market,
		Id:          iocOrderId,
		Side:        types.Side_SIDE_BUY,
		Price:       100,
		PartyID:     "B",
		Size:        20,
		Remaining:   20,
		TimeInForce: types.Order_TIF_IOC,
		CreatedAt:   10,
	}
	trades, getErr = book.GetTrades(iocOrder)
	assert.NoError(t, getErr)
	confirmation, err = book.SubmitOrder(iocOrder)

	assert.Equal(t, nil, err)
	assert.NotNil(t, confirmation)
	assert.Equal(t, iocOrderId, confirmation.Order.Id)
	assert.Equal(t, 1, len(confirmation.Trades))
	assert.Equal(t, len(trades), len(confirmation.Trades))

	// Check to see if the order still exists (it should not)
	nonorder, err := book.GetOrderByID(iocOrderId)
	assert.Equal(t, matching.ErrOrderDoesNotExist, err)
	assert.Nil(t, nonorder)
}

func makeOrder(t *testing.T, orderbook *tstOB, market string, id string, side types.Side, price uint64, partyid string, size uint64) {
	order := getOrder(t, orderbook, market, id, side, price, partyid, size)
	_, err := orderbook.SubmitOrder(order)
	assert.Equal(t, err, nil)
}

func getOrder(t *testing.T, orderbook *tstOB, market string, id string, side types.Side, price uint64, partyid string, size uint64) *types.Order {
	order := &types.Order{
		Status:      types.Order_STATUS_ACTIVE,
		Type:        types.Order_TYPE_LIMIT,
		MarketID:    market,
		Id:          id,
		Side:        side,
		Price:       price,
		PartyID:     partyid,
		Size:        size,
		Remaining:   size,
		TimeInForce: types.Order_TIF_GTC,
		CreatedAt:   10,
	}
	return order
}

/*****************************************************************************/
/*                             GFN/GFA TESTING                               */
/*****************************************************************************/

func TestOrderBook_GFNMarketNoExpiry(t *testing.T) {
	market := "testOrderbook"
	book := getTestOrderBook(t, market)
	defer book.Finish()

	logger := logging.NewTestLogger()
	defer logger.Sync()

	// Enter a GFN market order with no expiration time
	buyOrder := getOrder(t, book, market, "BuyOrder01", types.Side_SIDE_BUY, 100, "party01", 10)
	buyOrder.TimeInForce = types.Order_TIF_GFN
	buyOrder.Type = types.Order_TYPE_MARKET
	buyOrder.ExpiresAt = 0
	buyOrderConf, err := book.SubmitOrder(buyOrder)
	assert.NoError(t, err)
	assert.NotNil(t, buyOrderConf)

	// Enter a GFN market order with no expiration time
	sellOrder := getOrder(t, book, market, "SellOrder01", types.Side_SIDE_SELL, 100, "party01", 10)
	sellOrder.TimeInForce = types.Order_TIF_GFN
	sellOrder.Type = types.Order_TYPE_MARKET
	sellOrder.ExpiresAt = 0
	sellOrderConf, err := book.SubmitOrder(sellOrder)
	assert.NoError(t, err)
	assert.NotNil(t, sellOrderConf)
}

func TestOrderBook_GFNMarketWithExpiry(t *testing.T) {
	market := "testOrderbook"
	book := getTestOrderBook(t, market)
	defer book.Finish()

	logger := logging.NewTestLogger()
	defer logger.Sync()

	// Enter a GFN market order with an expiration time (which is invalid)
	buyOrder := getOrder(t, book, market, "BuyOrder01", types.Side_SIDE_BUY, 100, "party01", 10)
	buyOrder.TimeInForce = types.Order_TIF_GFN
	buyOrder.Type = types.Order_TYPE_MARKET
	buyOrder.ExpiresAt = 100
	buyOrderConf, err := book.SubmitOrder(buyOrder)
	assert.Error(t, err, types.ErrInvalidExpirationDatetime)
	assert.Nil(t, buyOrderConf)

	// Enter a GFN market order with an expiration time (which is invalid)
	sellOrder := getOrder(t, book, market, "SellOrder01", types.Side_SIDE_SELL, 100, "party01", 10)
	sellOrder.TimeInForce = types.Order_TIF_GFN
	sellOrder.Type = types.Order_TYPE_MARKET
	sellOrder.ExpiresAt = 100
	sellOrderConf, err := book.SubmitOrder(sellOrder)
	assert.Error(t, err, types.ErrInvalidExpirationDatetime)
	assert.Nil(t, sellOrderConf)
}

func TestOrderBook_GFNLimitInstantMatch(t *testing.T) {
	market := "testOrderbook"
	book := getTestOrderBook(t, market)
	defer book.Finish()

	logger := logging.NewTestLogger()
	defer logger.Sync()

	// Normal limit buy order to match against
	buyOrder := getOrder(t, book, market, "BuyOrder01", types.Side_SIDE_BUY, 100, "party01", 10)
	buyOrderConf, err := book.SubmitOrder(buyOrder)
	assert.NoError(t, err)
	assert.NotNil(t, buyOrderConf)

	// Enter a GFN market order with an expiration time (which is invalid)
	sellOrder := getOrder(t, book, market, "SellOrder01", types.Side_SIDE_SELL, 100, "party02", 10)
	sellOrder.TimeInForce = types.Order_TIF_GFN
	sellOrder.Type = types.Order_TYPE_LIMIT
	sellOrderConf, err := book.SubmitOrder(sellOrder)
	assert.NoError(t, err)
	assert.NotNil(t, sellOrderConf)
}

/*****************************************************************************/
/*                             AUCTION TESTING                               */
/*****************************************************************************/
func TestOrderBook_AuctionGFNAreRejected(t *testing.T) {
	market := "testOrderbook"
	book := getTestOrderBook(t, market)
	defer book.Finish()

	logger := logging.NewTestLogger()
	defer logger.Sync()

	// Switch to auction mode
	book.EnterAuction()
	assert.True(t, book.InAuction())

	// Try to add an order of type GFN which should be rejected
	order := getOrder(t, book, market, "BuyOrder01", types.Side_SIDE_BUY, 100, "party01", 10)
	order.TimeInForce = types.Order_TIF_GFN
	orderConf, err := book.SubmitOrder(order)
	assert.Equal(t, err, types.OrderError_ORDER_ERROR_INVALID_TIME_IN_FORCE)
	assert.Nil(t, orderConf)
}

func TestOrderBook_ContinuousGFAAreRejected(t *testing.T) {
	market := "testOrderbook"
	book := getTestOrderBook(t, market)
	defer book.Finish()

	logger := logging.NewTestLogger()
	defer logger.Sync()

	// We start in continuous mode
	assert.False(t, book.InAuction())

	// Try to add an order of type GFA which should be rejected
	order := getOrder(t, book, market, "BuyOrder01", types.Side_SIDE_BUY, 100, "party01", 10)
	order.TimeInForce = types.Order_TIF_GFA
	orderConf, err := book.SubmitOrder(order)
	assert.Equal(t, err, types.OrderError_ORDER_ERROR_INVALID_TIME_IN_FORCE)
	assert.Nil(t, orderConf)
}

func TestOrderBook_GFNOrdersCancelledInAuction(t *testing.T) {
	market := "testOrderbook"
	book := getTestOrderBook(t, market)
	defer book.Finish()

	logger := logging.NewTestLogger()
	defer logger.Sync()

	// We start in continuous mode
	assert.False(t, book.InAuction())

	// Add a GFN order
	order := getOrder(t, book, market, "BuyOrder01", types.Side_SIDE_BUY, 100, "party01", 10)
	order.TimeInForce = types.Order_TIF_GFN
	orderConf, err := book.SubmitOrder(order)
	assert.NoError(t, err)
	assert.NotNil(t, orderConf)

	// Switch to auction and makes sure the order is cancelled
	orders, parked, err := book.EnterAuction()
	assert.NoError(t, err)
	assert.Equal(t, len(orders), 1)
	assert.Equal(t, len(parked), 0)
	assert.Equal(t, book.GetTotalNumberOfOrders(), int64(1))
}

func TestOrderBook_GFAOrdersCancelledInContinuous(t *testing.T) {
	market := "testOrderbook"
	book := getTestOrderBook(t, market)
	defer book.Finish()

	logger := logging.NewTestLogger()
	defer logger.Sync()

	// Flip straight to auction mode
	_, _, err := book.EnterAuction()
	assert.NoError(t, err)
	assert.True(t, book.InAuction())

	// Add a GFA order
	order := getOrder(t, book, market, "BuyOrder01", types.Side_SIDE_BUY, 100, "party01", 10)
	order.TimeInForce = types.Order_TIF_GFA
	orderConf, err := book.SubmitOrder(order)
	assert.NoError(t, err)
	assert.NotNil(t, orderConf)

	// Switch to continuous mode and makes sure the order is cancelled
	uncrossedOrders, cancels, err := book.LeaveAuction(time.Now())
	assert.NoError(t, err)
	assert.Equal(t, len(uncrossedOrders), 0)
	assert.Equal(t, book.GetTotalNumberOfOrders(), int64(1))
	assert.Equal(t, len(cancels), 1)
}

func TestOrderBook_IndicativePriceAndVolumeState(t *testing.T) {
	market := "testOrderbook"
	book := getTestOrderBook(t, market)
	defer book.Finish()

	logger := logging.NewTestLogger()
	defer logger.Sync()

	// We start in continuous trading mode
	assert.False(t, book.InAuction())
	assert.Equal(t, book.GetTotalNumberOfOrders(), int64(0))

	// Get indicative auction price and volume which should be zero as we are out of auction
	price, volume, side := book.GetIndicativePriceAndVolume()
	assert.Equal(t, price, uint64(0))
	assert.Equal(t, volume, uint64(0))
	assert.Equal(t, side, types.Side_SIDE_UNSPECIFIED)
	price = book.GetIndicativePrice()
	assert.Equal(t, price, uint64(0))

	// Switch to auction mode
	book.EnterAuction()
	assert.True(t, book.InAuction())
	assert.Equal(t, book.GetTotalNumberOfOrders(), int64(0))

	// Get indicative auction price and volume
	price, volume, side = book.GetIndicativePriceAndVolume()
	assert.Equal(t, price, uint64(0))
	assert.Equal(t, volume, uint64(0))
	assert.Equal(t, side, types.Side_SIDE_UNSPECIFIED)
	price = book.GetIndicativePrice()
	assert.Equal(t, price, uint64(0))

	// Leave auction and uncross the book
	uncrossedOrders, cancels, err := book.LeaveAuction(time.Now())
	assert.Nil(t, err)
	assert.False(t, book.InAuction())
	assert.Equal(t, book.GetTotalNumberOfOrders(), int64(0))
	assert.Equal(t, len(uncrossedOrders), 0)
	assert.Equal(t, len(cancels), 0)
}

func TestOrderBook_IndicativePriceAndVolumeEmpty(t *testing.T) {
	market := "testOrderbook"
	book := getTestOrderBook(t, market)
	defer book.Finish()

	logger := logging.NewTestLogger()
	defer logger.Sync()

	// Switch to auction mode
	book.EnterAuction()
	assert.True(t, book.InAuction())

	// No trades!

	// Get indicative auction price and volume
	price, volume, side := book.GetIndicativePriceAndVolume()
	assert.Equal(t, price, uint64(0))
	assert.Equal(t, volume, uint64(0))
	assert.Equal(t, side, types.Side_SIDE_UNSPECIFIED)
	price = book.GetIndicativePrice()
	assert.Equal(t, price, uint64(0))

	// Leave auction and uncross the book
	uncrossedOrders, cancels, err := book.LeaveAuction(time.Now())
	assert.False(t, book.InAuction())
	assert.Nil(t, err)
	assert.Equal(t, len(uncrossedOrders), 0)
	assert.Equal(t, len(cancels), 0)
}

func TestOrderBook_IndicativePriceAndVolumeOnlyBuySide(t *testing.T) {
	market := "testOrderbook"
	book := getTestOrderBook(t, market)
	defer book.Finish()

	logger := logging.NewTestLogger()
	defer logger.Sync()

	// Switch to auction mode
	book.EnterAuction()

	// Trades on just one side of the book
	makeOrder(t, book, market, "BuyOrder01", types.Side_SIDE_BUY, 100, "party01", 10)
	makeOrder(t, book, market, "BuyOrder02", types.Side_SIDE_BUY, 99, "party01", 10)
	makeOrder(t, book, market, "BuyOrder03", types.Side_SIDE_BUY, 98, "party01", 10)

	// Get indicative auction price and volume
	price, volume, side := book.GetIndicativePriceAndVolume()
	assert.Equal(t, price, uint64(0))
	assert.Equal(t, volume, uint64(0))
	assert.Equal(t, side, types.Side_SIDE_UNSPECIFIED)
	price = book.GetIndicativePrice()
	assert.Equal(t, price, uint64(0))

	// Leave auction and uncross the book
	uncrossedOrders, cancels, err := book.LeaveAuction(time.Now())
	assert.Nil(t, err)
	assert.Equal(t, len(uncrossedOrders), 0)
	assert.Equal(t, len(cancels), 0)

	// All of the orders should remain on the book
	assert.Equal(t, book.GetTotalNumberOfOrders(), int64(3))
}

func TestOrderBook_IndicativePriceAndVolumeOnlySellSide(t *testing.T) {
	market := "testOrderbook"
	book := getTestOrderBook(t, market)
	defer book.Finish()

	logger := logging.NewTestLogger()
	defer logger.Sync()

	// Switch to auction mode
	book.EnterAuction()

	// Trades on just one side of the book
	makeOrder(t, book, market, "SellOrder01", types.Side_SIDE_SELL, 100, "party01", 10)
	makeOrder(t, book, market, "SellOrder02", types.Side_SIDE_SELL, 99, "party01", 10)
	makeOrder(t, book, market, "SellOrder03", types.Side_SIDE_SELL, 98, "party01", 10)

	// Get indicative auction price and volume
	price, volume, side := book.GetIndicativePriceAndVolume()
	assert.Equal(t, price, uint64(0))
	assert.Equal(t, volume, uint64(0))
	assert.Equal(t, side, types.Side_SIDE_UNSPECIFIED)
	price = book.GetIndicativePrice()
	assert.Equal(t, price, uint64(0))

	// Leave auction and uncross the book
	uncrossedOrders, cancels, err := book.LeaveAuction(time.Now())
	assert.Nil(t, err)
	assert.Equal(t, len(uncrossedOrders), 0)
	assert.Equal(t, len(cancels), 0)

	// All of the orders should remain on the book
	assert.Equal(t, book.GetTotalNumberOfOrders(), int64(3))
}

func TestOrderBook_IndicativePriceAndVolume1(t *testing.T) {
	market := "testOrderbook"
	book := getTestOrderBook(t, market)
	defer book.Finish()

	logger := logging.NewTestLogger()
	defer logger.Sync()

	// Switch to auction mode
	book.EnterAuction()

	// Populate buy side
	makeOrder(t, book, market, "BuyOrder01", types.Side_SIDE_BUY, 101, "party01", 20)
	makeOrder(t, book, market, "BuyOrder02", types.Side_SIDE_BUY, 100, "party01", 10)
	makeOrder(t, book, market, "BuyOrder03", types.Side_SIDE_BUY, 99, "party01", 20)
	makeOrder(t, book, market, "BuyOrder04", types.Side_SIDE_BUY, 98, "party01", 10)

	// Populate sell side
	makeOrder(t, book, market, "SellOrder01", types.Side_SIDE_SELL, 100, "party02", 10)
	makeOrder(t, book, market, "SellOrder02", types.Side_SIDE_SELL, 101, "party02", 15)
	makeOrder(t, book, market, "SellOrder03", types.Side_SIDE_SELL, 102, "party02", 5)
	makeOrder(t, book, market, "SellOrder04", types.Side_SIDE_SELL, 103, "party02", 10)

	// Get indicative auction price and volume
	price, volume, side := book.GetIndicativePriceAndVolume()
	assert.Equal(t, price, uint64(101))
	assert.Equal(t, volume, uint64(20))
	assert.Equal(t, side, types.Side_SIDE_BUY)
	price = book.GetIndicativePrice()
	assert.Equal(t, price, uint64(101))

	// Leave auction and uncross the book
	uncrossedOrders, cancels, err := book.LeaveAuction(time.Now())
	assert.Nil(t, err)
	assert.Equal(t, len(uncrossedOrders), 1)
	assert.Equal(t, len(cancels), 0)
}

func TestOrderBook_IndicativePriceAndVolume2(t *testing.T) {
	market := "testOrderbook"
	book := getTestOrderBook(t, market)
	defer book.Finish()

	logger := logging.NewTestLogger()
	defer logger.Sync()

	// Switch to auction mode
	book.EnterAuction()

	// Populate buy side
	makeOrder(t, book, market, "BuyOrder01", types.Side_SIDE_BUY, 101, "party01", 30)
	makeOrder(t, book, market, "BuyOrder02", types.Side_SIDE_BUY, 100, "party01", 10)
	makeOrder(t, book, market, "BuyOrder03", types.Side_SIDE_BUY, 99, "party01", 20)
	makeOrder(t, book, market, "BuyOrder04", types.Side_SIDE_BUY, 98, "party01", 10)
	makeOrder(t, book, market, "BuyOrder05", types.Side_SIDE_BUY, 97, "party01", 5)

	// Populate sell side
	makeOrder(t, book, market, "SellOrder01", types.Side_SIDE_SELL, 100, "party02", 30)
	makeOrder(t, book, market, "SellOrder02", types.Side_SIDE_SELL, 101, "party02", 15)
	makeOrder(t, book, market, "SellOrder03", types.Side_SIDE_SELL, 102, "party02", 5)
	makeOrder(t, book, market, "SellOrder04", types.Side_SIDE_SELL, 103, "party02", 10)

	// Get indicative auction price and volume
	price, volume, side := book.GetIndicativePriceAndVolume()
	assert.Equal(t, price, uint64(101))
	assert.Equal(t, volume, uint64(30))
	assert.Equal(t, side, types.Side_SIDE_BUY)
	price = book.GetIndicativePrice()
	assert.Equal(t, price, uint64(101))

	// Leave auction and uncross the book
	uncrossedOrders, cancels, err := book.LeaveAuction(time.Now())
	assert.Nil(t, err)
	assert.Equal(t, len(uncrossedOrders), 1)
	assert.Equal(t, len(cancels), 0)
}

func TestOrderBook_IndicativePriceAndVolume3(t *testing.T) {
	market := "testOrderbook"
	book := getTestOrderBook(t, market)
	defer book.Finish()

	logger := logging.NewTestLogger()
	defer logger.Sync()

	// Switch to auction mode
	book.EnterAuction()

	// Populate buy side
	makeOrder(t, book, market, "BuyOrder01", types.Side_SIDE_BUY, 104, "party01", 10)
	makeOrder(t, book, market, "BuyOrder02", types.Side_SIDE_BUY, 103, "party01", 20)
	makeOrder(t, book, market, "BuyOrder03", types.Side_SIDE_BUY, 102, "party01", 15)

	// Populate sell side
	makeOrder(t, book, market, "SellOrder01", types.Side_SIDE_SELL, 98, "party02", 10)
	makeOrder(t, book, market, "SellOrder02", types.Side_SIDE_SELL, 97, "party02", 20)
	makeOrder(t, book, market, "SellOrder03", types.Side_SIDE_SELL, 96, "party02", 15)

	// Get indicative auction price and volume
	price, volume, side := book.GetIndicativePriceAndVolume()
	assert.Equal(t, int(price), 102)
	assert.Equal(t, int(volume), 45)
	assert.Equal(t, side, types.Side_SIDE_BUY)
	price = book.GetIndicativePrice()
	assert.Equal(t, int(price), 102)

	// Leave auction and uncross the book
	uncrossedOrders, cancels, err := book.LeaveAuction(time.Now())
	assert.Nil(t, err)
	assert.Equal(t, len(uncrossedOrders), 3)
	assert.Equal(t, len(cancels), 0)
}

func TestOrderBook_IndicativePriceAndVolume4(t *testing.T) {
	market := "testOrderbook"
	book := getTestOrderBook(t, market)
	defer book.Finish()

	logger := logging.NewTestLogger()
	defer logger.Sync()

	// Switch to auction mode
	book.EnterAuction()

	// Populate buy side
	makeOrder(t, book, market, "BuyOrder01", types.Side_SIDE_BUY, 99, "party01", 10)
	makeOrder(t, book, market, "BuyOrder02", types.Side_SIDE_BUY, 98, "party01", 25)
	makeOrder(t, book, market, "BuyOrder03", types.Side_SIDE_BUY, 97, "party01", 5)

	// Populate sell side
	makeOrder(t, book, market, "SellOrder01", types.Side_SIDE_SELL, 102, "party02", 30)
	makeOrder(t, book, market, "SellOrder02", types.Side_SIDE_SELL, 101, "party02", 15)
	makeOrder(t, book, market, "SellOrder03", types.Side_SIDE_SELL, 100, "party02", 5)

	// Get indicative auction price and volume
	price, volume, side := book.GetIndicativePriceAndVolume()
	assert.Equal(t, price, uint64(0))
	assert.Equal(t, volume, uint64(0))
	assert.Equal(t, side, types.Side_SIDE_UNSPECIFIED)
	price = book.GetIndicativePrice()
	assert.Equal(t, price, uint64(0))

	// Leave auction and uncross the book
	uncrossedOrders, cancels, err := book.LeaveAuction(time.Now())
	assert.Nil(t, err)
	assert.Equal(t, len(uncrossedOrders), 0)
	assert.Equal(t, len(cancels), 0)
}

func TestOrderBook_IndicativePriceAndVolume5(t *testing.T) {
	market := "testOrderbook"
	book := getTestOrderBook(t, market)
	defer book.Finish()

	logger := logging.NewTestLogger()
	defer logger.Sync()

	// Switch to auction mode
	book.EnterAuction()

	// Populate buy side
	makeOrder(t, book, market, "BuyOrder01", types.Side_SIDE_BUY, 103, "party01", 10)
	makeOrder(t, book, market, "BuyOrder02", types.Side_SIDE_BUY, 102, "party01", 9)
	makeOrder(t, book, market, "BuyOrder03", types.Side_SIDE_BUY, 101, "party01", 8)
	makeOrder(t, book, market, "BuyOrder04", types.Side_SIDE_BUY, 100, "party01", 7)
	makeOrder(t, book, market, "BuyOrder05", types.Side_SIDE_BUY, 99, "party01", 6)
	makeOrder(t, book, market, "BuyOrder06", types.Side_SIDE_BUY, 98, "party01", 5)
	makeOrder(t, book, market, "BuyOrder07", types.Side_SIDE_BUY, 97, "party01", 4)
	makeOrder(t, book, market, "BuyOrder08", types.Side_SIDE_BUY, 96, "party01", 3)
	makeOrder(t, book, market, "BuyOrder09", types.Side_SIDE_BUY, 95, "party01", 2)
	makeOrder(t, book, market, "BuyOrder10", types.Side_SIDE_BUY, 94, "party01", 1)

	// Populate sell side
	makeOrder(t, book, market, "SellOrder01", types.Side_SIDE_SELL, 105, "party02", 1)
	makeOrder(t, book, market, "SellOrder02", types.Side_SIDE_SELL, 104, "party02", 2)
	makeOrder(t, book, market, "SellOrder03", types.Side_SIDE_SELL, 103, "party02", 3)
	makeOrder(t, book, market, "SellOrder04", types.Side_SIDE_SELL, 102, "party02", 4)
	makeOrder(t, book, market, "SellOrder05", types.Side_SIDE_SELL, 101, "party02", 5)
	makeOrder(t, book, market, "SellOrder06", types.Side_SIDE_SELL, 100, "party02", 6)
	makeOrder(t, book, market, "SellOrder07", types.Side_SIDE_SELL, 99, "party02", 7)
	makeOrder(t, book, market, "SellOrder08", types.Side_SIDE_SELL, 98, "party02", 8)
	makeOrder(t, book, market, "SellOrder09", types.Side_SIDE_SELL, 97, "party02", 9)
	makeOrder(t, book, market, "SellOrder10", types.Side_SIDE_SELL, 96, "party02", 10)

	// Get indicative auction price and volume
	price, volume, side := book.GetIndicativePriceAndVolume()
	assert.Equal(t, price, uint64(100))
	assert.Equal(t, volume, uint64(34))
	assert.Equal(t, side, types.Side_SIDE_BUY)
	price = book.GetIndicativePrice()
	assert.Equal(t, price, uint64(100))

	// Leave auction and uncross the book
	uncrossedOrders, cancels, err := book.LeaveAuction(time.Now())
	assert.Nil(t, err)
	assert.Equal(t, len(uncrossedOrders), 4)
	assert.Equal(t, len(cancels), 0)
}

// Set up an auction so that the sell side is processed when we uncross
func TestOrderBook_IndicativePriceAndVolume6(t *testing.T) {
	market := "testOrderbook"
	book := getTestOrderBook(t, market)
	defer book.Finish()

	logger := logging.NewTestLogger()
	defer logger.Sync()

	// Switch to auction mode
	book.EnterAuction()

	// Populate buy side
	makeOrder(t, book, market, "BuyOrder01", types.Side_SIDE_BUY, 103, "party01", 10)
	makeOrder(t, book, market, "BuyOrder02", types.Side_SIDE_BUY, 102, "party01", 9)
	makeOrder(t, book, market, "BuyOrder03", types.Side_SIDE_BUY, 101, "party01", 8)
	makeOrder(t, book, market, "BuyOrder04", types.Side_SIDE_BUY, 100, "party01", 7)

	// Populate sell side
	makeOrder(t, book, market, "SellOrder01", types.Side_SIDE_SELL, 99, "party02", 1)
	makeOrder(t, book, market, "SellOrder02", types.Side_SIDE_SELL, 98, "party02", 2)
	makeOrder(t, book, market, "SellOrder03", types.Side_SIDE_SELL, 97, "party02", 3)
	makeOrder(t, book, market, "SellOrder04", types.Side_SIDE_SELL, 96, "party02", 4)

	// Get indicative auction price and volume
	price, volume, side := book.GetIndicativePriceAndVolume()
	assert.Equal(t, int(price), 102)
	assert.Equal(t, int(volume), 10)
	assert.Equal(t, side, types.Side_SIDE_SELL)
	price = book.GetIndicativePrice()
	assert.Equal(t, int(price), 102)

	// Leave auction and uncross the book
	uncrossedOrders, cancels, err := book.LeaveAuction(time.Now())
	assert.Nil(t, err)
	assert.Equal(t, len(uncrossedOrders), 4)
	assert.Equal(t, len(cancels), 0)
}

// Check that multiple orders per price level work
func TestOrderBook_IndicativePriceAndVolume7(t *testing.T) {
	market := "testOrderbook"
	book := getTestOrderBook(t, market)
	defer book.Finish()

	logger := logging.NewTestLogger()
	defer logger.Sync()

	// Switch to auction mode
	book.EnterAuction()

	// Populate buy side
	makeOrder(t, book, market, "BuyOrder01", types.Side_SIDE_BUY, 103, "party01", 10)
	makeOrder(t, book, market, "BuyOrder02", types.Side_SIDE_BUY, 103, "party01", 1)
	makeOrder(t, book, market, "BuyOrder03", types.Side_SIDE_BUY, 102, "party01", 9)
	makeOrder(t, book, market, "BuyOrder04", types.Side_SIDE_BUY, 102, "party01", 1)
	makeOrder(t, book, market, "BuyOrder05", types.Side_SIDE_BUY, 101, "party01", 8)
	makeOrder(t, book, market, "BuyOrder06", types.Side_SIDE_BUY, 101, "party01", 1)
	makeOrder(t, book, market, "BuyOrder07", types.Side_SIDE_BUY, 100, "party01", 7)
	makeOrder(t, book, market, "BuyOrder08", types.Side_SIDE_BUY, 100, "party01", 1)

	// Populate sell side
	makeOrder(t, book, market, "SellOrder01", types.Side_SIDE_SELL, 99, "party02", 10)
	makeOrder(t, book, market, "SellOrder02", types.Side_SIDE_SELL, 98, "party02", 10)
	makeOrder(t, book, market, "SellOrder03", types.Side_SIDE_SELL, 97, "party02", 10)
	makeOrder(t, book, market, "SellOrder04", types.Side_SIDE_SELL, 96, "party02", 7)

	// Get indicative auction price and volume
	price, volume, side := book.GetIndicativePriceAndVolume()
	assert.Equal(t, price, uint64(100))
	assert.Equal(t, volume, uint64(37))
	assert.Equal(t, side, types.Side_SIDE_SELL)
	price = book.GetIndicativePrice()
	assert.Equal(t, price, uint64(100))

	// Leave auction and uncross the book
	uncrossedOrders, cancels, err := book.LeaveAuction(time.Now())
	assert.Nil(t, err)
	assert.Equal(t, len(uncrossedOrders), 4)
	assert.Equal(t, len(cancels), 0)
}

func TestOrderBook_IndicativePriceAndVolume8(t *testing.T) {
	market := "testOrderbook"
	book := getTestOrderBook(t, market)
	defer book.Finish()

	logger := logging.NewTestLogger()
	defer logger.Sync()

	// Switch to auction mode
	book.EnterAuction()

	// Populate buy side
	makeOrder(t, book, market, "BuyOrder01", types.Side_SIDE_BUY, 103, "party01", 10)
	makeOrder(t, book, market, "BuyOrder02", types.Side_SIDE_BUY, 103, "party01", 1)
	makeOrder(t, book, market, "BuyOrder03", types.Side_SIDE_BUY, 102, "party01", 9)
	makeOrder(t, book, market, "BuyOrder04", types.Side_SIDE_BUY, 102, "party01", 1)
	makeOrder(t, book, market, "BuyOrder05", types.Side_SIDE_BUY, 101, "party01", 8)
	makeOrder(t, book, market, "BuyOrder06", types.Side_SIDE_BUY, 101, "party01", 1)
	makeOrder(t, book, market, "BuyOrder07", types.Side_SIDE_BUY, 100, "party01", 7)
	makeOrder(t, book, market, "BuyOrder08", types.Side_SIDE_BUY, 100, "party01", 1)

	// Populate sell side
	makeOrder(t, book, market, "SellOrder01", types.Side_SIDE_SELL, 99, "party02", 10)
	makeOrder(t, book, market, "SellOrder02", types.Side_SIDE_SELL, 98, "party02", 10)
	makeOrder(t, book, market, "SellOrder03", types.Side_SIDE_SELL, 97, "party02", 10)
	makeOrder(t, book, market, "SellOrder04", types.Side_SIDE_SELL, 96, "party02", 9)

	// Get indicative auction price and volume
	price, volume, side := book.GetIndicativePriceAndVolume()
	assert.Equal(t, int(price), 100)
	assert.Equal(t, int(volume), 38)
	assert.Equal(t, side, types.Side_SIDE_BUY)
	price = book.GetIndicativePrice()
	assert.Equal(t, int(price), 100)

	// Leave auction and uncross the book
	uncrossedOrders, cancels, err := book.LeaveAuction(time.Now())
	assert.Nil(t, err)
	assert.Equal(t, len(uncrossedOrders), 8)
	assert.Equal(t, len(cancels), 0)
}

func TestOrderBook_UncrossTest1(t *testing.T) {
	market := "testOrderbook"
	book := getTestOrderBook(t, market)
	defer book.Finish()

	logger := logging.NewTestLogger()
	defer logger.Sync()

	// Switch to auction mode
	book.EnterAuction()

	bo1 := getOrder(t, book, market, "BuyOrder01", types.Side_SIDE_BUY, 100, "party01", 5)
	bo1.TimeInForce = types.Order_TIF_GFA
	book.SubmitOrder(bo1)

	so1 := getOrder(t, book, market, "SellOrder01", types.Side_SIDE_SELL, 100, "party02", 5)
	so1.TimeInForce = types.Order_TIF_GFA
	book.SubmitOrder(so1)

	bo2 := getOrder(t, book, market, "BuyOrder02", types.Side_SIDE_BUY, 100, "party01", 5)
	bo2.TimeInForce = types.Order_TIF_GFA
	book.SubmitOrder(bo2)

	so2 := getOrder(t, book, market, "SellOrder02", types.Side_SIDE_SELL, 101, "party02", 5)
	so2.TimeInForce = types.Order_TIF_GFA
	book.SubmitOrder(so2)

	// Get indicative auction price and volume
	price, volume, side := book.GetIndicativePriceAndVolume()
	assert.Equal(t, price, uint64(100))
	assert.Equal(t, volume, uint64(5))
	assert.Equal(t, side, types.Side_SIDE_SELL)
	price = book.GetIndicativePrice()
	assert.Equal(t, price, uint64(100))

	// Leave auction and uncross the book
	uncrossedOrders, cancels, err := book.LeaveAuction(time.Now())
	assert.Nil(t, err)
	assert.Equal(t, len(uncrossedOrders), 1)
	assert.Equal(t, len(cancels), 2)
}

// this is a test for issue 2060 to ensure we process FOK orders properly
func TestOrderBook_NetworkOrderSuccess(t *testing.T) {
	market := "testOrderbook"
	book := getTestOrderBook(t, market)
	defer book.Finish()

	orders := []*types.Order{
		{
			Status:      types.Order_STATUS_ACTIVE,
			Type:        types.Order_TYPE_LIMIT,
			MarketID:    market,
			Id:          "123456",
			Side:        types.Side_SIDE_BUY,
			Price:       100,
			PartyID:     "A",
			Size:        100,
			Remaining:   100,
			TimeInForce: types.Order_TIF_GTC,
			CreatedAt:   10,
		},
		{
			Status:      types.Order_STATUS_ACTIVE,
			Type:        types.Order_TYPE_LIMIT,
			MarketID:    market,
			Id:          "234561",
			Side:        types.Side_SIDE_BUY,
			Price:       1,
			PartyID:     "B",
			Size:        100,
			Remaining:   100,
			TimeInForce: types.Order_TIF_GTC,
			CreatedAt:   11,
		},
	}

	// now we add the trades to the book
	for _, o := range orders {
		cnfm, err := book.SubmitOrder(o)
		assert.NoError(t, err)
		assert.Len(t, cnfm.Trades, 0)
	}

	// no price for network order
	// we want to consume the whole book
	netorder := &types.Order{
		Status:      types.Order_STATUS_ACTIVE,
		Type:        types.Order_TYPE_NETWORK,
		MarketID:    market,
		Id:          "345612",
		Side:        types.Side_SIDE_SELL,
		PartyID:     "C",
		Size:        200,
		Remaining:   200,
		TimeInForce: types.Order_TIF_FOK,
		CreatedAt:   12,
	}

	cnfm, err := book.SubmitOrder(netorder)
	assert.NoError(t, err)
	assert.Equal(t, types.Order_STATUS_FILLED, netorder.Status)
	assert.Equal(t, 50, int(netorder.Price))
	assert.Equal(t, 0, int(netorder.Remaining))
	_ = cnfm
}

func TestOrderBook_GetTradesInLineWithSubmitOrderDuringAuction(t *testing.T) {
	market := "testOrderbook"
	book := getTestOrderBook(t, market)

	orders, parked, err := book.EnterAuction()

	assert.Equal(t, 0, len(orders))
	assert.Equal(t, 0, len(parked))
	assert.Nil(t, err)
	order1Id := "1000000000000000000000" //Must be 22 characters
	order2Id := "1000000000000000000001" //Must be 22 characters

	order1 := &types.Order{
		Status:      types.Order_STATUS_ACTIVE,
		Type:        types.Order_TYPE_LIMIT,
		MarketID:    market,
		Id:          order1Id,
		Side:        types.Side_SIDE_SELL,
		Price:       100,
		PartyID:     "A",
		Size:        100,
		Remaining:   100,
		TimeInForce: types.Order_TIF_GTC,
		CreatedAt:   10,
	}

	trades, getErr := book.GetTrades(order1)
	assert.NoError(t, getErr)
	confirmation, err := book.SubmitOrder(order1)

	assert.Equal(t, nil, err)
	assert.NotNil(t, confirmation)
	assert.Equal(t, order1Id, confirmation.Order.Id)
	assert.Equal(t, 0, len(confirmation.Trades))
	assert.Equal(t, len(trades), len(confirmation.Trades))

	order2 := &types.Order{
		Status:      types.Order_STATUS_ACTIVE,
		Type:        types.Order_TYPE_LIMIT,
		MarketID:    market,
		Id:          order2Id,
		Side:        types.Side_SIDE_BUY,
		Price:       100,
		PartyID:     "B",
		Size:        20,
		Remaining:   20,
		TimeInForce: types.Order_TIF_GTC,
		CreatedAt:   10,
	}
	trades, getErr = book.GetTrades(order2)
	assert.NoError(t, getErr)
	confirmation, err = book.SubmitOrder(order2)

	assert.Equal(t, nil, err)
	assert.NotNil(t, confirmation)
	assert.Equal(t, order2Id, confirmation.Order.Id)
	assert.Equal(t, 0, len(confirmation.Trades))
	assert.Equal(t, len(trades), len(confirmation.Trades))

	// Confirm both orders still on the book
	order, err := book.GetOrderByID(order1Id)
	assert.NotNil(t, order)
	assert.Nil(t, err)
	order, err = book.GetOrderByID(order2Id)
	assert.NotNil(t, order)
	assert.Nil(t, err)
}

func TestOrderBook_AuctionUncrossWashTrades(t *testing.T) {
	market := "testOrderbook"
	book := getTestOrderBook(t, market)
	defer book.Finish()

	logger := logging.NewTestLogger()
	defer logger.Sync()

	// Switch to auction mode
	book.EnterAuction()

	bo1 := getOrder(t, book, market, "BuyOrder01", types.Side_SIDE_BUY, 100, "party01", 5)
	bo1.TimeInForce = types.Order_TIF_GFA
	book.SubmitOrder(bo1)

	so1 := getOrder(t, book, market, "SellOrder01", types.Side_SIDE_SELL, 100, "party01", 5)
	so1.TimeInForce = types.Order_TIF_GFA
	book.SubmitOrder(so1)

	// Get indicative auction price and volume
	price, volume, side := book.GetIndicativePriceAndVolume()
	assert.Equal(t, price, uint64(100))
	assert.Equal(t, volume, uint64(5))
	assert.Equal(t, side, types.Side_SIDE_BUY)

	// Leave auction and uncross the book
	uncrossedOrders, cancels, err := book.LeaveAuction(time.Now())
	assert.Nil(t, err)
	assert.Equal(t, len(uncrossedOrders), 1)
	assert.Equal(t, len(cancels), 0)
}
