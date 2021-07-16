package matching

import (
	"testing"

	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"

	"github.com/stretchr/testify/assert"
)

func TestBidAndAskPresentAfterAuction(t *testing.T) {
	market := "market"
	book := getTestOrderBook(t, market)
	defer book.Finish()

	// start an auction
	_ = book.EnterAuction()

	orders := []types.Order{
		{
			MarketID:    market,
			Party:       "party-1",
			Side:        types.SideBuy,
			Price:       num.NewUint(2000),
			Size:        5,
			Remaining:   5,
			TimeInForce: types.OrderTimeInForceGTC,
			Type:        types.OrderTypeLimit,
		},
		{
			MarketID:    market,
			Party:       "party-1",
			Side:        types.SideSell,
			Price:       num.NewUint(2000),
			Size:        5,
			Remaining:   5,
			TimeInForce: types.OrderTimeInForceGTC,
			Type:        types.OrderTypeLimit,
		},
		{
			MarketID:    market,
			Party:       "party-1",
			Side:        types.SideBuy,
			Price:       num.NewUint(1900),
			Size:        5,
			Remaining:   5,
			TimeInForce: types.OrderTimeInForceGTC,
			Type:        types.OrderTypeLimit,
		},
		{
			MarketID:    market,
			Party:       "party-1",
			Side:        types.SideSell,
			Price:       num.NewUint(1950),
			Size:        5,
			Remaining:   5,
			TimeInForce: types.OrderTimeInForceGTC,
			Type:        types.OrderTypeLimit,
		},
	}

	for _, order := range orders {
		_, err := book.SubmitOrder(&order)
		assert.NoError(t, err)
	}

	indicativePrice, indicativeVolume, indicativeSide := book.GetIndicativePriceAndVolume()
	assert.Equal(t, indicativePrice.Uint64(), uint64(1975))
	assert.Equal(t, int(indicativeVolume), 5)
	assert.Equal(t, indicativeSide, types.SideBuy)
	assert.True(t, book.BidAndAskPresentAfterAuction())
}

func TestBidAndAskPresentAfterAuctionInverse(t *testing.T) {
	market := "market"
	book := getTestOrderBook(t, market)
	defer book.Finish()

	// start an auction
	_ = book.EnterAuction()

	orders := []types.Order{
		{
			MarketID:    market,
			Party:       "party-1",
			Side:        types.SideBuy,
			Price:       num.NewUint(2000),
			Size:        5,
			Remaining:   5,
			TimeInForce: types.OrderTimeInForceGTC,
			Type:        types.OrderTypeLimit,
		},
		{
			MarketID:    market,
			Party:       "party-1",
			Side:        types.SideSell,
			Price:       num.NewUint(2050),
			Size:        5,
			Remaining:   5,
			TimeInForce: types.OrderTimeInForceGTC,
			Type:        types.OrderTypeLimit,
		},
		{
			MarketID:    market,
			Party:       "party-1",
			Side:        types.SideBuy,
			Price:       num.NewUint(1900),
			Size:        5,
			Remaining:   5,
			TimeInForce: types.OrderTimeInForceGTC,
			Type:        types.OrderTypeLimit,
		},
		{
			MarketID:    market,
			Party:       "party-1",
			Side:        types.SideSell,
			Price:       num.NewUint(1900),
			Size:        5,
			Remaining:   5,
			TimeInForce: types.OrderTimeInForceGTC,
			Type:        types.OrderTypeLimit,
		},
	}

	for _, order := range orders {
		_, err := book.SubmitOrder(&order)
		assert.NoError(t, err)
	}

	indicativePrice, indicativeVolume, indicativeSide := book.GetIndicativePriceAndVolume()
	assert.Equal(t, indicativePrice.Uint64(), uint64(1950))
	assert.Equal(t, int(indicativeVolume), 5)
	assert.Equal(t, indicativeSide, types.SideBuy)
	assert.True(t, book.BidAndAskPresentAfterAuction())
}
