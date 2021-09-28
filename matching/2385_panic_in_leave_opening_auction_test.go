package matching

import (
	"testing"
	"time"

	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"
	"github.com/stretchr/testify/assert"
)

func TestPanicInLeaveAuction(t *testing.T) {
	market := "testMarket"
	book := getTestOrderBook(t, market)
	defer book.Finish()
	orders := []types.Order{
		{
			MarketID:    market,
			Party:       "A",
			Side:        types.SideBuy,
			Price:       num.NewUint(100),
			Size:        5,
			Remaining:   5,
			TimeInForce: types.OrderTimeInForceGTC,
			Type:        types.OrderTypeLimit,
			ID:          "v0000000000000-0000001",
		},
		{
			MarketID:    market,
			Party:       "B",
			Side:        types.SideSell,
			Price:       num.NewUint(100),
			Size:        5,
			Remaining:   5,
			TimeInForce: types.OrderTimeInForceGTC,
			Type:        types.OrderTypeLimit,
			ID:          "v0000000000000-0000002",
		},
		{
			MarketID:    market,
			Party:       "C",
			Side:        types.SideBuy,
			Price:       num.NewUint(150),
			Size:        2,
			Remaining:   2,
			TimeInForce: types.OrderTimeInForceGTC,
			Type:        types.OrderTypeLimit,
			ID:          "v0000000000000-0000003",
		},
		{
			MarketID:    market,
			Party:       "D",
			Side:        types.SideBuy,
			Price:       num.NewUint(150),
			Size:        2,
			Remaining:   2,
			TimeInForce: types.OrderTimeInForceGTC,
			Type:        types.OrderTypeLimit,
			ID:          "v0000000000000-0000004",
		},
		{
			MarketID:    market,
			Party:       "E",
			Side:        types.SideSell,
			Price:       num.NewUint(150),
			Size:        2,
			Remaining:   2,
			TimeInForce: types.OrderTimeInForceGTC,
			Type:        types.OrderTypeLimit,
			ID:          "v0000000000000-0000005",
		},
		{
			MarketID:    market,
			Party:       "F",
			Side:        types.SideBuy,
			Price:       num.NewUint(150),
			Size:        2,
			Remaining:   2,
			TimeInForce: types.OrderTimeInForceGTC,
			Type:        types.OrderTypeLimit,
			ID:          "v0000000000000-0000006",
		},
		{
			MarketID:    market,
			Party:       "G",
			Side:        types.SideSell,
			Price:       num.NewUint(150),
			Size:        2,
			Remaining:   2,
			TimeInForce: types.OrderTimeInForceGTC,
			Type:        types.OrderTypeLimit,
			ID:          "v0000000000000-0000007",
		},
		{
			MarketID:    market,
			Party:       "A",
			Side:        types.SideBuy,
			Price:       num.NewUint(120),
			Size:        33,
			Remaining:   33,
			TimeInForce: types.OrderTimeInForceGTC,
			Type:        types.OrderTypeLimit,
			ID:          "v0000000000000-0000008",
		},
	}

	// enter auction, should return no error and no orders
	cnlorders := book.EnterAuction()
	assert.Len(t, cnlorders, 0)

	for _, o := range orders {
		o := o
		cnf, err := book.SubmitOrder(&o)
		assert.NoError(t, err)
		assert.Len(t, cnf.Trades, 0)
		assert.Len(t, cnf.PassiveOrdersAffected, 0)
	}

	cnf, porders, err := book.LeaveAuction(time.Now())
	assert.NoError(t, err)
	_ = cnf
	_ = porders

}
