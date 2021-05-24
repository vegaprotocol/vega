package execution_test

import (
	"context"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/events"
	types "code.vegaprotocol.io/vega/proto"
	commandspb "code.vegaprotocol.io/vega/proto/commands/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRefreshLiquidityProvisionOrdersSizes(t *testing.T) {
	now := time.Unix(10, 0)
	closingAt := time.Unix(1000000000, 0)
	ctx := context.Background()

	mktCfg := getMarket(closingAt, defaultPriceMonitorSettings, &types.AuctionDuration{
		Duration: 10000,
	})
	mktCfg.Fees = &types.Fees{
		Factors: &types.FeeFactors{
			LiquidityFee:      "0.001",
			InfrastructureFee: "0.0005",
			MakerFee:          "0.00025",
		},
	}
	mktCfg.TradableInstrument.RiskModel = &types.TradableInstrument_LogNormalRiskModel{
		LogNormalRiskModel: &types.LogNormalRiskModel{
			RiskAversionParameter: 0.001,
			Tau:                   0.00011407711613050422,
			Params: &types.LogNormalModelParams{
				Mu:    0,
				R:     0.016,
				Sigma: 20,
			},
		},
	}

	tm := newTestMarket(t, now).Run(ctx, mktCfg)
	tm.StartOpeningAuction().
		WithAccountAndAmount("trader-0", 1000000).
		WithAccountAndAmount("trader-1", 1000000).
		WithAccountAndAmount("trader-2", 10000000000).
		// provide stake as well but will cancel
		WithAccountAndAmount("trader-2-bis", 10000000000).
		WithAccountAndAmount("trader-3", 1000000).
		WithAccountAndAmount("trader-4", 1000000)

	tm.market.OnSuppliedStakeToObligationFactorUpdate(1.0)
	tm.market.OnChainTimeUpdate(ctx, now)

	orderParams := []struct {
		id        string
		size      uint64
		side      types.Side
		tif       types.Order_TimeInForce
		pegRef    types.PeggedReference
		pegOffset int64
	}{
		{"trader-4", 1, types.Side_SIDE_BUY, types.Order_TIME_IN_FORCE_GTC, types.PeggedReference_PEGGED_REFERENCE_BEST_BID, -2000},
		{"trader-3", 1, types.Side_SIDE_SELL, types.Order_TIME_IN_FORCE_GTC, types.PeggedReference_PEGGED_REFERENCE_BEST_ASK, 1000},
	}
	traderA, traderB := orderParams[0], orderParams[1]

	tpl := OrderTemplate{
		Type: types.Order_TYPE_LIMIT,
	}
	var orders = []*types.Order{
		// Limit Orders
		tpl.New(types.Order{
			Size:        20,
			Remaining:   20,
			Price:       uint64(5500 + traderA.pegOffset), // 3500
			Side:        types.Side_SIDE_BUY,
			PartyId:     "trader-0",
			TimeInForce: types.Order_TIME_IN_FORCE_GFA,
		}),
		tpl.New(types.Order{
			Size:        20,
			Remaining:   20,
			Price:       uint64(5000 - traderB.pegOffset), // 4000
			Side:        types.Side_SIDE_SELL,
			PartyId:     "trader-1",
			TimeInForce: types.Order_TIME_IN_FORCE_GFA,
		}),
		tpl.New(types.Order{
			Size:        10,
			Remaining:   10,
			Price:       5500,
			Side:        types.Side_SIDE_BUY,
			PartyId:     "trader-2",
			TimeInForce: types.Order_TIME_IN_FORCE_GFA,
		}),
		tpl.New(types.Order{
			Size:        100,
			Remaining:   100,
			Price:       5000,
			Side:        types.Side_SIDE_SELL,
			PartyId:     "trader-2",
			TimeInForce: types.Order_TIME_IN_FORCE_GTC,
		}),
		tpl.New(types.Order{
			Size:        100,
			Remaining:   100,
			Price:       3500,
			Side:        types.Side_SIDE_BUY,
			PartyId:     "trader-0",
			TimeInForce: types.Order_TIME_IN_FORCE_GTC,
		}),
		tpl.New(types.Order{
			Size:        20,
			Remaining:   20,
			Price:       8500,
			Side:        types.Side_SIDE_BUY,
			PartyId:     "trader-0",
			TimeInForce: types.Order_TIME_IN_FORCE_GTC,
		}),

		// Pegged Orders
		tpl.New(types.Order{
			PartyId:     traderA.id,
			Side:        traderA.side,
			Size:        traderA.size,
			Remaining:   traderA.size,
			TimeInForce: traderA.tif,
			PeggedOrder: &types.PeggedOrder{
				Reference: traderA.pegRef,
				Offset:    traderA.pegOffset,
			},
		}),
		tpl.New(types.Order{
			PartyId:     traderB.id,
			Side:        traderB.side,
			Size:        traderB.size,
			Remaining:   traderB.size,
			TimeInForce: traderB.tif,
			PeggedOrder: &types.PeggedOrder{
				Reference: traderB.pegRef,
				Offset:    traderB.pegOffset,
			},
		}),
	}

	tm.WithSubmittedOrders(t, orders...)

	// Add a LPSubmission
	// this is a log of stake, enough to cover all
	// the required stake for the market
	lp := &commandspb.LiquidityProvisionSubmission{
		MarketId:         tm.market.GetID(),
		CommitmentAmount: 2000000,
		Fee:              "0.01",
		Sells: []*types.LiquidityOrder{
			{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_ASK, Proportion: 10, Offset: 2},
			{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_ASK, Proportion: 13, Offset: 1},
		},
		Buys: []*types.LiquidityOrder{
			{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_BID, Proportion: 10, Offset: -1},
			{Reference: types.PeggedReference_PEGGED_REFERENCE_MID, Proportion: 13, Offset: -15},
		},
	}

	// Leave the auction
	tm.market.OnChainTimeUpdate(ctx, now.Add(10001*time.Second))

	require.NoError(t, tm.market.SubmitLiquidityProvision(ctx, lp, "trader-2", "id-lp"))
	assert.Equal(t, 1, tm.market.GetLPSCount())

	tm.market.OnChainTimeUpdate(ctx, now.Add(10011*time.Second))

	newOrder := tpl.New(types.Order{
		MarketId:    tm.market.GetID(),
		Size:        20,
		Remaining:   20,
		Price:       4235,
		Side:        types.Side_SIDE_SELL,
		PartyId:     "trader-0",
		TimeInForce: types.Order_TIME_IN_FORCE_GTC,
	})

	tm.events = nil
	cnf, err := tm.market.SubmitOrder(ctx, newOrder)
	assert.NoError(t, err)
	assert.True(t, len(cnf.Trades) > 0)

	// now all our orders have been cancelled
	t.Run("ExpectedOrderStatus", func(t *testing.T) {
		// First collect all the orders events
		found := []*types.Order{}
		for _, e := range tm.events {
			switch evt := e.(type) {
			case *events.Order:
				if evt.Order().PartyId == "trader-2" {
					found = append(found, evt.Order())
				}
			}
		}

		// one event is sent, this is a rejected event from
		// the first order we try to place, the party does
		// not have enough funds
		expectedStatus := []struct {
			status    types.Order_Status
			remaining uint64
		}{
			{
				// this is the first update indicating the order
				// was matched
				types.Order_STATUS_ACTIVE,
				0x202, // size - 20
			},
			{
				// this is the replacement order created
				// by engine.
				types.Order_STATUS_CANCELLED,
				0x202, // size
			},
			{
				// this is the cancellation
				types.Order_STATUS_ACTIVE,
				0x216, // cancelled
			},
		}

		require.Len(t, found, len(expectedStatus))

		for i, expect := range expectedStatus {
			got := found[i].Status
			remaining := found[i].Remaining
			assert.Equal(t, expect.status.String(), got.String())
			assert.Equal(t, expect.remaining, remaining)
		}
	})
}

func TestRefreshLiquidityProvisionOrdersSizesCrashOnSubmitOrder(t *testing.T) {
	now := time.Unix(10, 0)
	closingAt := time.Unix(1000000000, 0)
	ctx := context.Background()

	auctionEnd := now.Add(10001 * time.Second)
	mktCfg := getMarket(closingAt, defaultPriceMonitorSettings, &types.AuctionDuration{
		Duration: 10000,
	})
	mktCfg.Fees = &types.Fees{
		Factors: &types.FeeFactors{
			LiquidityFee:      "0.001",
			InfrastructureFee: "0.0005",
			MakerFee:          "0.00025",
		},
	}
	mktCfg.TradableInstrument.RiskModel = &types.TradableInstrument_LogNormalRiskModel{
		LogNormalRiskModel: &types.LogNormalRiskModel{
			RiskAversionParameter: 0.001,
			Tau:                   0.00011407711613050422,
			Params: &types.LogNormalModelParams{
				Mu:    0,
				R:     0.016,
				Sigma: 20,
			},
		},
	}

	lpparty := "lp-party-1"

	tm := newTestMarket(t, now).Run(ctx, mktCfg)
	tm.StartOpeningAuction().
		// the liquidity provider
		WithAccountAndAmount(lpparty, 155000)

	tm.market.OnSuppliedStakeToObligationFactorUpdate(1.0)
	tm.market.OnChainTimeUpdate(ctx, now)

	// Add a LPSubmission
	// this is a log of stake, enough to cover all
	// the required stake for the market
	lpSubmission := &commandspb.LiquidityProvisionSubmission{
		MarketId:         tm.market.GetID(),
		CommitmentAmount: 150000,
		Fee:              "0.01",
		Reference:        "ref-lp-submission-1",
		Buys: []*types.LiquidityOrder{
			{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_BID, Proportion: 2, Offset: -500},
			{Reference: types.PeggedReference_PEGGED_REFERENCE_MID, Proportion: 2, Offset: -500},
		},
		Sells: []*types.LiquidityOrder{
			{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_ASK, Proportion: 13, Offset: 500},
			{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_ASK, Proportion: 13, Offset: 500},
		},
	}

	// submit our lp
	require.NoError(t,
		tm.market.SubmitLiquidityProvision(
			ctx, lpSubmission, lpparty, "liquidity-submission-1"),
	)

	// clear auction
	tm.EndOpeningAuction(t, auctionEnd, true)
}

func TestCommitmentIsDeployed(t *testing.T) {
	now := time.Unix(10, 0)
	closingAt := time.Unix(1000000000, 0)
	ctx := context.Background()

	auctionEnd := now.Add(10001 * time.Second)
	mktCfg := getMarket(closingAt, defaultPriceMonitorSettings, &types.AuctionDuration{
		Duration: 10000,
	})
	mktCfg.Fees = &types.Fees{
		Factors: &types.FeeFactors{
			LiquidityFee:      "0.001",
			InfrastructureFee: "0.0005",
			MakerFee:          "0.00025",
		},
	}
	mktCfg.TradableInstrument.RiskModel = &types.TradableInstrument_LogNormalRiskModel{
		LogNormalRiskModel: &types.LogNormalRiskModel{
			RiskAversionParameter: 0.001,
			Tau:                   0.00011407711613050422,
			Params: &types.LogNormalModelParams{
				Mu:    0,
				R:     0.016,
				Sigma: 20,
			},
		},
	}

	lpparty := "lp-party-1"

	tm := newTestMarket(t, now).Run(ctx, mktCfg)
	tm.StartOpeningAuction().
		// the liquidity provider
		WithAccountAndAmount(lpparty, 50000000)

	tm.market.OnSuppliedStakeToObligationFactorUpdate(1.0)
	tm.market.OnChainTimeUpdate(ctx, now)

	// Add a LPSubmission
	// this is a log of stake, enough to cover all
	// the required stake for the market
	lpSubmission := &commandspb.LiquidityProvisionSubmission{
		MarketId:         tm.market.GetID(),
		CommitmentAmount: 200,
		Fee:              "0.01",
		Reference:        "ref-lp-submission-1",
		Buys: []*types.LiquidityOrder{
			{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_BID, Proportion: 2, Offset: -50},
			{Reference: types.PeggedReference_PEGGED_REFERENCE_MID, Proportion: 7, Offset: -50},
		},
		Sells: []*types.LiquidityOrder{
			{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_ASK, Proportion: 13, Offset: 50},
			{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_ASK, Proportion: 5, Offset: 50},
		},
	}

	// submit our lp
	require.NoError(t,
		tm.market.SubmitLiquidityProvision(
			ctx, lpSubmission, lpparty, "liquidity-submission-1"),
	)

	// clear auction
	tm.EndOpeningAuction(t, auctionEnd, true)
}

func (tm *testMarket) EndOpeningAuction(t *testing.T, auctionEnd time.Time, setMarkPrice bool) {
	var (
		party0 = "clearing-auction-party0"
		party1 = "clearing-auction-party1"
	)

	// parties used for clearing opening auction
	tm.WithAccountAndAmount(party0, 1000000).
		WithAccountAndAmount(party1, 1000000)

	var auctionOrders = []*types.Order{
		// Limit Orders
		{
			Type:        types.Order_TYPE_LIMIT,
			Size:        5,
			Remaining:   5,
			Price:       1000,
			Side:        types.Side_SIDE_BUY,
			PartyId:     party0,
			TimeInForce: types.Order_TIME_IN_FORCE_GTC,
		},
		{
			Type:        types.Order_TYPE_LIMIT,
			Size:        5,
			Remaining:   5,
			Price:       1000,
			Side:        types.Side_SIDE_SELL,
			PartyId:     party1,
			TimeInForce: types.Order_TIME_IN_FORCE_GTC,
		},
		{
			Type:        types.Order_TYPE_LIMIT,
			Size:        1,
			Remaining:   1,
			Price:       900,
			Side:        types.Side_SIDE_BUY,
			PartyId:     party0,
			TimeInForce: types.Order_TIME_IN_FORCE_GTC,
		},
		{
			Type:        types.Order_TYPE_LIMIT,
			Size:        1,
			Remaining:   1,
			Price:       1100,
			Side:        types.Side_SIDE_SELL,
			PartyId:     party1,
			TimeInForce: types.Order_TIME_IN_FORCE_GTC,
		},
	}

	// submit the auctions orders
	tm.WithSubmittedOrders(t, auctionOrders...)

	// update the time to get out of auction
	tm.market.OnChainTimeUpdate(context.Background(), auctionEnd)

	assert.Equal(t,
		tm.market.GetMarketData().MarketTradingMode,
		types.Market_TRADING_MODE_CONTINUOUS,
	)

	if setMarkPrice {
		// now set the markprice
		mpOrders := []*types.Order{
			{
				Type:        types.Order_TYPE_LIMIT,
				Size:        1,
				Remaining:   1,
				Price:       900,
				Side:        types.Side_SIDE_SELL,
				PartyId:     party1,
				TimeInForce: types.Order_TIME_IN_FORCE_GTC,
			},
			{
				Type:        types.Order_TYPE_LIMIT,
				Size:        1,
				Remaining:   1,
				Price:       2500,
				Side:        types.Side_SIDE_BUY,
				PartyId:     party0,
				TimeInForce: types.Order_TIME_IN_FORCE_GTC,
			},
		}
		// submit the auctions orders
		tm.WithSubmittedOrders(t, mpOrders...)
	}

}

func (tm *testMarket) EndOpeningAuction2(t *testing.T, auctionEnd time.Time, setMarkPrice bool) {
	var (
		party0 = "clearing-auction-party0"
		party1 = "clearing-auction-party1"
	)

	// parties used for clearing opening auction
	tm.WithAccountAndAmount(party0, 1000000).
		WithAccountAndAmount(party1, 1000000)

	var auctionOrders = []*types.Order{
		// Limit Orders
		{
			Type:        types.Order_TYPE_LIMIT,
			Size:        5,
			Remaining:   5,
			Price:       1000,
			Side:        types.Side_SIDE_BUY,
			PartyId:     party0,
			TimeInForce: types.Order_TIME_IN_FORCE_GTC,
		},
		{
			Type:        types.Order_TYPE_LIMIT,
			Size:        5,
			Remaining:   5,
			Price:       1000,
			Side:        types.Side_SIDE_SELL,
			PartyId:     party1,
			TimeInForce: types.Order_TIME_IN_FORCE_GTC,
		},
		{
			Type:        types.Order_TYPE_LIMIT,
			Size:        1,
			Remaining:   1,
			Price:       900,
			Side:        types.Side_SIDE_BUY,
			PartyId:     party0,
			TimeInForce: types.Order_TIME_IN_FORCE_GTC,
		},
		{
			Type:        types.Order_TYPE_LIMIT,
			Size:        1,
			Remaining:   1,
			Price:       1200,
			Side:        types.Side_SIDE_SELL,
			PartyId:     party1,
			TimeInForce: types.Order_TIME_IN_FORCE_GTC,
		},
	}

	// submit the auctions orders
	tm.WithSubmittedOrders(t, auctionOrders...)

	// update the time to get out of auction
	tm.market.OnChainTimeUpdate(context.Background(), auctionEnd)

	assert.Equal(t,
		tm.market.GetMarketData().MarketTradingMode,
		types.Market_TRADING_MODE_CONTINUOUS,
	)

	if setMarkPrice {
		// now set the markprice
		mpOrders := []*types.Order{
			{
				Type:        types.Order_TYPE_LIMIT,
				Size:        1,
				Remaining:   1,
				Price:       900,
				Side:        types.Side_SIDE_SELL,
				PartyId:     party1,
				TimeInForce: types.Order_TIME_IN_FORCE_GTC,
			},
			{
				Type:        types.Order_TYPE_LIMIT,
				Size:        1,
				Remaining:   1,
				Price:       1200,
				Side:        types.Side_SIDE_BUY,
				PartyId:     party0,
				TimeInForce: types.Order_TIME_IN_FORCE_GTC,
			},
		}
		// submit the auctions orders
		tm.WithSubmittedOrders(t, mpOrders...)
	}

}
