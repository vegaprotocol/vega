package matching

import (
	"fmt"
	"testing"

	types "code.vegaprotocol.io/vega/proto"
	"github.com/stretchr/testify/assert"
)

func TestPanicInLeaveAuction(t *testing.T) {
	market := "testMarket"
	book := getTestOrderBook(t, market)
	defer book.Finish()
	orders := []types.Order{
		{
			MarketID:    market,
			PartyID:     "A",
			Side:        types.Side_SIDE_BUY,
			Price:       100,
			Size:        5,
			Remaining:   5,
			TimeInForce: types.Order_TIF_GTC,
			Type:        types.Order_TYPE_LIMIT,
			Id:          "v0000000000000-0000001",
		},
		{
			MarketID:    market,
			PartyID:     "B",
			Side:        types.Side_SIDE_SELL,
			Price:       100,
			Size:        5,
			Remaining:   5,
			TimeInForce: types.Order_TIF_GTC,
			Type:        types.Order_TYPE_LIMIT,
			Id:          "v0000000000000-0000002",
		},
		{
			MarketID:    market,
			PartyID:     "C",
			Side:        types.Side_SIDE_BUY,
			Price:       150,
			Size:        2,
			Remaining:   2,
			TimeInForce: types.Order_TIF_GTC,
			Type:        types.Order_TYPE_LIMIT,
			Id:          "v0000000000000-0000003",
		},
		{
			MarketID:    market,
			PartyID:     "D",
			Side:        types.Side_SIDE_BUY,
			Price:       150,
			Size:        2,
			Remaining:   2,
			TimeInForce: types.Order_TIF_GTC,
			Type:        types.Order_TYPE_LIMIT,
			Id:          "v0000000000000-0000004",
		},
		{
			MarketID:    market,
			PartyID:     "E",
			Side:        types.Side_SIDE_SELL,
			Price:       150,
			Size:        2,
			Remaining:   2,
			TimeInForce: types.Order_TIF_GTC,
			Type:        types.Order_TYPE_LIMIT,
			Id:          "v0000000000000-0000005",
		},
		{
			MarketID:    market,
			PartyID:     "F",
			Side:        types.Side_SIDE_BUY,
			Price:       150,
			Size:        2,
			Remaining:   2,
			TimeInForce: types.Order_TIF_GTC,
			Type:        types.Order_TYPE_LIMIT,
			Id:          "v0000000000000-0000006",
		},
		{
			MarketID:    market,
			PartyID:     "G",
			Side:        types.Side_SIDE_SELL,
			Price:       150,
			Size:        2,
			Remaining:   2,
			TimeInForce: types.Order_TIF_GTC,
			Type:        types.Order_TYPE_LIMIT,
			Id:          "v0000000000000-0000007",
		},
		{
			MarketID:    market,
			PartyID:     "A",
			Side:        types.Side_SIDE_BUY,
			Price:       120,
			Size:        33,
			Remaining:   33,
			TimeInForce: types.Order_TIF_GTC,
			Type:        types.Order_TYPE_LIMIT,
			Id:          "v0000000000000-0000008",
		},
		// {
		// 	MarketID:    market,
		// 	PartyID:     "B",
		// 	Side:        types.Side_SIDE_SELL,
		// 	Size:        25,
		// 	Remaining:   25,
		// 	TimeInForce: types.Order_TIF_FOK,
		// 	Type:        types.Order_TYPE_MARKET,
		// 	Id:          "v0000000000000-0000008",
		// },
	}

	// enter auction, should return no error and no orders
	cnlorders, err := book.EnterAuction()
	assert.NoError(t, err)
	assert.Len(t, cnlorders, 0)

	for _, o := range orders {
		o := o
		fmt.Printf("YOLO\n")
		cnf, err := book.SubmitOrder(&o)
		assert.NoError(t, err)
		assert.Len(t, cnf.Trades, 0)
		assert.Len(t, cnf.PassiveOrdersAffected, 0)
	}

	cnf, porders, err := book.LeaveAuction()
	assert.NoError(t, err)
	_ = cnf
	_ = porders

}
