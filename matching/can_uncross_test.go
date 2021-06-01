package matching

import (
	"testing"

	"code.vegaprotocol.io/vega/types"

	"github.com/stretchr/testify/assert"
)

func TestBidAndAskPresentAfterAuction(t *testing.T) {
	market := "market"
	book := getTestOrderBook(t, market)
	defer book.Finish()

	// start an auction
	_, err := book.EnterAuction()
	assert.NoError(t, err)

	orders := []types.Order{
		{
			MarketId:    market,
			PartyId:     "party-1",
			Side:        types.Side_SIDE_BUY,
			Price:       2000,
			Size:        5,
			Remaining:   5,
			TimeInForce: types.Order_TIME_IN_FORCE_GTC,
			Type:        types.Order_TYPE_LIMIT,
		},
		{
			MarketId:    market,
			PartyId:     "party-1",
			Side:        types.Side_SIDE_SELL,
			Price:       2000,
			Size:        5,
			Remaining:   5,
			TimeInForce: types.Order_TIME_IN_FORCE_GTC,
			Type:        types.Order_TYPE_LIMIT,
		},
		{
			MarketId:    market,
			PartyId:     "party-1",
			Side:        types.Side_SIDE_BUY,
			Price:       1900,
			Size:        5,
			Remaining:   5,
			TimeInForce: types.Order_TIME_IN_FORCE_GTC,
			Type:        types.Order_TYPE_LIMIT,
		},
		{
			MarketId:    market,
			PartyId:     "party-1",
			Side:        types.Side_SIDE_SELL,
			Price:       1950,
			Size:        5,
			Remaining:   5,
			TimeInForce: types.Order_TIME_IN_FORCE_GTC,
			Type:        types.Order_TYPE_LIMIT,
		},
	}

	for _, order := range orders {
		_, err := book.SubmitOrder(&order)
		assert.NoError(t, err)
	}

	indicativePrice, indicativeVolume, indicativeSide := book.GetIndicativePriceAndVolume()
	assert.Equal(t, int(indicativePrice), 1975)
	assert.Equal(t, int(indicativeVolume), 5)
	assert.Equal(t, indicativeSide, types.Side_SIDE_BUY)
	assert.True(t, book.BidAndAskPresentAfterAuction())
}

func TestBidAndAskPresentAfterAuctionInverse(t *testing.T) {
	market := "market"
	book := getTestOrderBook(t, market)
	defer book.Finish()

	// start an auction
	_, err := book.EnterAuction()
	assert.NoError(t, err)

	orders := []types.Order{
		{
			MarketId:    market,
			PartyId:     "party-1",
			Side:        types.Side_SIDE_BUY,
			Price:       2000,
			Size:        5,
			Remaining:   5,
			TimeInForce: types.Order_TIME_IN_FORCE_GTC,
			Type:        types.Order_TYPE_LIMIT,
		},
		{
			MarketId:    market,
			PartyId:     "party-1",
			Side:        types.Side_SIDE_SELL,
			Price:       2050,
			Size:        5,
			Remaining:   5,
			TimeInForce: types.Order_TIME_IN_FORCE_GTC,
			Type:        types.Order_TYPE_LIMIT,
		},
		{
			MarketId:    market,
			PartyId:     "party-1",
			Side:        types.Side_SIDE_BUY,
			Price:       1900,
			Size:        5,
			Remaining:   5,
			TimeInForce: types.Order_TIME_IN_FORCE_GTC,
			Type:        types.Order_TYPE_LIMIT,
		},
		{
			MarketId:    market,
			PartyId:     "party-1",
			Side:        types.Side_SIDE_SELL,
			Price:       1900,
			Size:        5,
			Remaining:   5,
			TimeInForce: types.Order_TIME_IN_FORCE_GTC,
			Type:        types.Order_TYPE_LIMIT,
		},
	}

	for _, order := range orders {
		_, err := book.SubmitOrder(&order)
		assert.NoError(t, err)
	}

	indicativePrice, indicativeVolume, indicativeSide := book.GetIndicativePriceAndVolume()
	assert.Equal(t, int(indicativePrice), 1950)
	assert.Equal(t, int(indicativeVolume), 5)
	assert.Equal(t, indicativeSide, types.Side_SIDE_BUY)
	assert.True(t, book.BidAndAskPresentAfterAuction())
}
