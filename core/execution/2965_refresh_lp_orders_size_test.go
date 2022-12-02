// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.VEGA file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package execution_test

import (
	"context"
	"testing"
	"time"

	vegacontext "code.vegaprotocol.io/vega/libs/context"
	vgcrypto "code.vegaprotocol.io/vega/libs/crypto"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	vegapb "code.vegaprotocol.io/vega/protos/vega"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRefreshLiquidityProvisionOrdersSizes(t *testing.T) {
	now := time.Unix(10, 0)
	ctx := context.Background()
	ctx = vegacontext.WithTraceID(ctx, vgcrypto.RandomHash())

	mktCfg := getMarket(defaultPriceMonitorSettings, &types.AuctionDuration{
		Duration: 10000,
	})
	mktCfg.Fees = &types.Fees{
		Factors: &types.FeeFactors{
			LiquidityFee:      num.DecimalFromFloat(0.001),
			InfrastructureFee: num.DecimalFromFloat(0.0005),
			MakerFee:          num.DecimalFromFloat(0.00025),
		},
	}
	mktCfg.TradableInstrument.RiskModel = &types.TradableInstrumentLogNormalRiskModel{
		LogNormalRiskModel: &types.LogNormalRiskModel{
			RiskAversionParameter: num.DecimalFromFloat(0.001),
			Tau:                   num.DecimalFromFloat(0.00011407711613050422),
			Params: &types.LogNormalModelParams{
				Mu:    num.DecimalZero(),
				R:     num.DecimalFromFloat(0.016),
				Sigma: num.DecimalFromFloat(20),
			},
		},
	}

	tm := newTestMarket(t, now).Run(ctx, mktCfg)
	tm.StartOpeningAuction().
		WithAccountAndAmount("party-0", 1000000).
		WithAccountAndAmount("party-1", 1000000).
		WithAccountAndAmount("party-2", 10000000000).
		// provide stake as well but will cancel
		WithAccountAndAmount("party-2-bis", 10000000000).
		WithAccountAndAmount("party-3", 1000000).
		WithAccountAndAmount("party-4", 1000000)

	tm.market.OnSuppliedStakeToObligationFactorUpdate(num.DecimalFromFloat(1.0))
	tm.market.OnTick(ctx, now)

	orderParams := []struct {
		id        string
		size      uint64
		side      types.Side
		tif       types.OrderTimeInForce
		pegRef    types.PeggedReference
		pegOffset *num.Uint
	}{
		{"party-4", 1, types.SideBuy, types.OrderTimeInForceGTC, types.PeggedReferenceBestBid, num.NewUint(2000)},
		{"party-3", 1, types.SideSell, types.OrderTimeInForceGTC, types.PeggedReferenceBestAsk, num.NewUint(1000)},
	}
	partyA, partyB := orderParams[0], orderParams[1]

	tpl := OrderTemplate{
		Type: types.OrderTypeLimit,
	}
	orders := []*types.Order{
		// Limit Orders
		tpl.New(types.Order{
			Size:        20,
			Remaining:   20,
			Price:       num.Sum(num.NewUint(5500), partyA.pegOffset), // 3500
			Side:        types.SideBuy,
			Party:       "party-0",
			TimeInForce: types.OrderTimeInForceGFA,
		}),
		tpl.New(types.Order{
			Size:        20,
			Remaining:   20,
			Price:       num.UintZero().Sub(num.NewUint(5000), partyB.pegOffset), // 4000
			Side:        types.SideSell,
			Party:       "party-1",
			TimeInForce: types.OrderTimeInForceGFA,
		}),
		tpl.New(types.Order{
			Size:        10,
			Remaining:   10,
			Price:       num.NewUint(5500),
			Side:        types.SideBuy,
			Party:       "party-2",
			TimeInForce: types.OrderTimeInForceGFA,
		}),
		tpl.New(types.Order{
			Size:        100,
			Remaining:   100,
			Price:       num.NewUint(5000),
			Side:        types.SideSell,
			Party:       "party-2",
			TimeInForce: types.OrderTimeInForceGTC,
		}),
		tpl.New(types.Order{
			Size:        100,
			Remaining:   100,
			Price:       num.NewUint(3500),
			Side:        types.SideBuy,
			Party:       "party-0",
			TimeInForce: types.OrderTimeInForceGTC,
		}),
		tpl.New(types.Order{
			Size:        20,
			Remaining:   20,
			Price:       num.NewUint(8500),
			Side:        types.SideBuy,
			Party:       "party-0",
			TimeInForce: types.OrderTimeInForceGTC,
		}),

		// Pegged Orders
		tpl.New(types.Order{
			Party:       partyA.id,
			Side:        partyA.side,
			Size:        partyA.size,
			Remaining:   partyA.size,
			TimeInForce: partyA.tif,
			PeggedOrder: &types.PeggedOrder{
				Reference: partyA.pegRef,
				Offset:    partyA.pegOffset,
			},
		}),
		tpl.New(types.Order{
			Party:       partyB.id,
			Side:        partyB.side,
			Size:        partyB.size,
			Remaining:   partyB.size,
			TimeInForce: partyB.tif,
			PeggedOrder: &types.PeggedOrder{
				Reference: partyB.pegRef,
				Offset:    partyB.pegOffset,
			},
		}),
	}

	tm.WithSubmittedOrders(t, orders...)

	// Add a LPSubmission
	// this is a log of stake, enough to cover all
	// the required stake for the market
	lp := &types.LiquidityProvisionSubmission{
		MarketID:         tm.market.GetID(),
		CommitmentAmount: num.NewUint(3120580),
		Fee:              num.DecimalFromFloat(0.01),
		Sells: []*types.LiquidityOrder{
			newLiquidityOrder(types.PeggedReferenceBestAsk, 2, 10),
			newLiquidityOrder(types.PeggedReferenceBestAsk, 1, 13),
		},
		Buys: []*types.LiquidityOrder{
			newLiquidityOrder(types.PeggedReferenceBestBid, 1, 10),
			newLiquidityOrder(types.PeggedReferenceMid, 15, 13),
		},
	}

	// Leave the auction
	newT := now.Add(10001 * time.Second)
	tm.now = newT
	tm.market.OnTick(ctx, newT)

	require.NoError(t, tm.market.SubmitLiquidityProvision(ctx, lp, "party-2", vgcrypto.RandomHash()))
	assert.Equal(t, 1, tm.market.GetLPSCount())

	newT = newT.Add(10 * time.Second)
	tm.now = newT
	tm.market.OnTick(ctx, newT)

	newOrder := tpl.New(types.Order{
		MarketID:    tm.market.GetID(),
		Size:        20,
		Remaining:   20,
		Price:       num.NewUint(4235),
		Side:        types.SideSell,
		Party:       "party-0",
		TimeInForce: types.OrderTimeInForceGTC,
	})

	md := tm.market.GetMarketData()
	require.Equal(t, md.MarketTradingMode, types.MarketTradingModeContinuous, "not in continuous trading")
	tm.events = nil

	// assure that the order price is within the valid price range so it can trade as expected
	require.True(t, newOrder.Price.GT(md.PriceMonitoringBounds[0].MinValidPrice))
	require.True(t, newOrder.Price.LT(md.PriceMonitoringBounds[0].MaxValidPrice))

	cnf, err := tm.market.SubmitOrder(ctx, newOrder)
	assert.NoError(t, err)
	assert.True(t, len(cnf.Trades) > 0)
	// just trigger MTM bit
	tm.market.OnTick(ctx, newT)

	// now all our orders have been cancelled
	t.Run("ExpectedOrderStatus", func(t *testing.T) {
		// First collect all the orders events
		found := []*types.Order{}
		for _, e := range tm.events {
			switch evt := e.(type) {
			case *events.Order:
				if evt.Order().PartyId == "party-2" &&
					evt.Order().Size == 833 { // "V0000000000-0000000010" {
					found = append(found, mustOrderFromProto(evt.Order()))
				}
			}
		}

		// one event is sent, this is a rejected event from
		// the first order we try to place, the party does
		// not have enough funds
		expectedStatus := []struct {
			status    types.OrderStatus
			remaining int
		}{
			{
				// this is the first update indicating the order
				// was matched
				types.OrderStatusActive,
				813, // size - 20
			},
			{
				// this is the replacement order created
				// by engine.
				types.OrderStatusCancelled,
				813, // size
			},
			{
				// this is the cancellation
				types.OrderStatusActive,
				833, // cancelled
			},
			{
				// this is quite possibly a duplicate because we're forcing the check
				// for reference moves
				types.OrderStatusActive,
				833, // cancelled
			},
		}

		require.Len(t, found, len(expectedStatus))

		for i, expect := range expectedStatus {
			got := found[i].Status
			remaining := int(found[i].Remaining)
			assert.Equal(t, expect.status.String(), got.String())
			assert.Equal(t, expect.remaining, remaining)
		}
	})
}

func TestRefreshLiquidityProvisionOrdersSizesCrashOnSubmitOrder(t *testing.T) {
	now := time.Unix(10, 0)

	ctx := vegacontext.WithTraceID(context.Background(), vgcrypto.RandomHash())

	auctionEnd := now.Add(10001 * time.Second)
	mktCfg := getMarket(defaultPriceMonitorSettings, &types.AuctionDuration{
		Duration: 10000,
	})
	mktCfg.Fees = &types.Fees{
		Factors: &types.FeeFactors{
			LiquidityFee:      num.DecimalFromFloat(0.001),
			InfrastructureFee: num.DecimalFromFloat(0.0005),
			MakerFee:          num.DecimalFromFloat(0.00025),
		},
	}
	mktCfg.TradableInstrument.RiskModel = &types.TradableInstrumentLogNormalRiskModel{
		LogNormalRiskModel: &types.LogNormalRiskModel{
			RiskAversionParameter: num.DecimalFromFloat(0.001),
			Tau:                   num.DecimalFromFloat(0.00011407711613050422),
			Params: &types.LogNormalModelParams{
				Mu:    num.DecimalZero(),
				R:     num.DecimalFromFloat(0.016),
				Sigma: num.DecimalFromFloat(20),
			},
		},
	}

	lpparty := "lp-party-1"

	tm := newTestMarket(t, now).Run(ctx, mktCfg)
	tm.StartOpeningAuction().
		// the liquidity provider
		WithAccountAndAmount(lpparty, 155000)

	tm.market.OnSuppliedStakeToObligationFactorUpdate(num.DecimalFromFloat(1.0))
	tm.market.OnTick(ctx, now)

	// Add a LPSubmission
	// this is a log of stake, enough to cover all
	// the required stake for the market
	lpSubmission := &types.LiquidityProvisionSubmission{
		MarketID:         tm.market.GetID(),
		CommitmentAmount: num.NewUint(150000),
		Fee:              num.DecimalFromFloat(0.01),
		Reference:        "ref-lp-submission-1",
		Buys: []*types.LiquidityOrder{
			newLiquidityOrder(types.PeggedReferenceBestBid, 500, 2),
			newLiquidityOrder(types.PeggedReferenceMid, 500, 2),
		},
		Sells: []*types.LiquidityOrder{
			newLiquidityOrder(types.PeggedReferenceBestAsk, 500, 13),
			newLiquidityOrder(types.PeggedReferenceBestAsk, 500, 13),
		},
	}

	// submit our lp
	require.NoError(t,
		tm.market.SubmitLiquidityProvision(
			ctx, lpSubmission, lpparty, vgcrypto.RandomHash()),
	)

	// clear auction
	tm.EndOpeningAuction(t, auctionEnd, true)
}

func TestCommitmentIsDeployed(t *testing.T) {
	now := time.Unix(10, 0)
	ctx := vegacontext.WithTraceID(context.Background(), vgcrypto.RandomHash())

	auctionEnd := now.Add(10001 * time.Second)
	mktCfg := getMarket(defaultPriceMonitorSettings, &types.AuctionDuration{
		Duration: 10000,
	})
	mktCfg.Fees = &types.Fees{
		Factors: &types.FeeFactors{
			LiquidityFee:      num.DecimalFromFloat(0.001),
			InfrastructureFee: num.DecimalFromFloat(0.0005),
			MakerFee:          num.DecimalFromFloat(0.00025),
		},
	}
	mktCfg.TradableInstrument.RiskModel = &types.TradableInstrumentLogNormalRiskModel{
		LogNormalRiskModel: &types.LogNormalRiskModel{
			RiskAversionParameter: num.DecimalFromFloat(0.001),
			Tau:                   num.DecimalFromFloat(0.00011407711613050422),
			Params: &types.LogNormalModelParams{
				Mu:    num.DecimalZero(),
				R:     num.DecimalFromFloat(0.016),
				Sigma: num.DecimalFromFloat(20),
			},
		},
	}

	lpparty := "lp-party-1"

	tm := newTestMarket(t, now).Run(ctx, mktCfg)
	tm.StartOpeningAuction().
		// the liquidity provider
		WithAccountAndAmount(lpparty, 90000000)

	tm.market.OnSuppliedStakeToObligationFactorUpdate(num.DecimalFromFloat(1.0))
	tm.market.OnTick(ctx, now)

	// Add a LPSubmission
	// this is a log of stake, enough to cover all
	// the required stake for the market
	lpSubmission := &types.LiquidityProvisionSubmission{
		MarketID:         tm.market.GetID(),
		CommitmentAmount: num.NewUint(50000000),
		Fee:              num.DecimalFromFloat(0.01),
		Reference:        "ref-lp-submission-1",
		Buys: []*types.LiquidityOrder{
			newLiquidityOrder(types.PeggedReferenceBestBid, 50, 2),
			newLiquidityOrder(types.PeggedReferenceMid, 50, 7),
		},
		Sells: []*types.LiquidityOrder{
			newLiquidityOrder(types.PeggedReferenceBestAsk, 50, 13),
			newLiquidityOrder(types.PeggedReferenceBestAsk, 50, 5),
		},
	}

	// submit our lp
	require.NoError(t,
		tm.market.SubmitLiquidityProvision(
			ctx, lpSubmission, lpparty, vgcrypto.RandomHash()),
	)

	// clear auction
	tm.EndOpeningAuction(t, auctionEnd, true)
}

func (tm *testMarket) EndOpeningAuction(t *testing.T, auctionEnd time.Time, setMarkPrice bool) {
	t.Helper()
	var (
		party0 = "clearing-auction-party0"
		party1 = "clearing-auction-party1"
		party2 = "lpprov-party"
	)

	// parties used for clearing opening auction
	tm.WithAccountAndAmount(party0, 1000000).
		WithAccountAndAmount(party1, 1000000).
		WithAccountAndAmount(party2, 90000000000) // LP needs a lot of balance

	auctionOrders := []*types.Order{
		// Limit Orders
		{
			Type:        types.OrderTypeLimit,
			Size:        5,
			Remaining:   5,
			Price:       num.NewUint(1000),
			Side:        types.SideBuy,
			Party:       party0,
			TimeInForce: types.OrderTimeInForceGTC,
		},
		{
			Type:        types.OrderTypeLimit,
			Size:        5,
			Remaining:   5,
			Price:       num.NewUint(1000),
			Side:        types.SideSell,
			Party:       party1,
			TimeInForce: types.OrderTimeInForceGTC,
		},
		{
			Type:        types.OrderTypeLimit,
			Size:        1,
			Remaining:   1,
			Price:       num.NewUint(900),
			Side:        types.SideBuy,
			Party:       party0,
			TimeInForce: types.OrderTimeInForceGTC,
		},
		{
			Type:        types.OrderTypeLimit,
			Size:        1,
			Remaining:   1,
			Price:       num.NewUint(1100),
			Side:        types.SideSell,
			Party:       party1,
			TimeInForce: types.OrderTimeInForceGTC,
		},
	}
	ctx := vegacontext.WithTraceID(context.Background(), vgcrypto.RandomHash())
	// submit the auctions orders & LP
	tm.WithSubmittedOrders(t, auctionOrders...)
	// update the time to get out of auction
	if setMarkPrice {
		// now set the markprice
		mpOrders := []*types.Order{
			{
				Type:        types.OrderTypeLimit,
				Size:        1,
				Remaining:   1,
				Price:       num.NewUint(900),
				Side:        types.SideSell,
				Party:       party1,
				TimeInForce: types.OrderTimeInForceGTC,
			},
			{
				Type:        types.OrderTypeLimit,
				Size:        1,
				Remaining:   1,
				Price:       num.NewUint(2500),
				Side:        types.SideBuy,
				Party:       party0,
				TimeInForce: types.OrderTimeInForceGTC,
			},
		}
		// submit the auctions orders
		tm.WithSubmittedOrders(t, mpOrders...)
	}

	tm.now = auctionEnd
	tm.market.OnTick(ctx, auctionEnd)

	assert.Equal(t,
		tm.market.GetMarketData().MarketTradingMode,
		types.MarketTradingModeContinuous,
	)
}

func (tm *testMarket) EndOpeningAuction2(t *testing.T, auctionEnd time.Time, setMarkPrice bool) {
	t.Helper()
	var (
		party0 = "clearing-auction-party0"
		party1 = "clearing-auction-party1"
	)

	// parties used for clearing opening auction
	tm.WithAccountAndAmount(party0, 1000000).
		WithAccountAndAmount(party1, 1000000)

	auctionOrders := []*types.Order{
		// Limit Orders
		{
			Type:        types.OrderTypeLimit,
			Size:        5,
			Remaining:   5,
			Price:       num.NewUint(1000),
			Side:        types.SideBuy,
			Party:       party0,
			TimeInForce: types.OrderTimeInForceGTC,
		},
		{
			Type:        types.OrderTypeLimit,
			Size:        5,
			Remaining:   5,
			Price:       num.NewUint(1000),
			Side:        types.SideSell,
			Party:       party1,
			TimeInForce: types.OrderTimeInForceGTC,
		},
		{
			Type:        types.OrderTypeLimit,
			Size:        1,
			Remaining:   1,
			Price:       num.NewUint(900),
			Side:        types.SideBuy,
			Party:       party0,
			TimeInForce: types.OrderTimeInForceGTC,
		},
		{
			Type:        types.OrderTypeLimit,
			Size:        1,
			Remaining:   1,
			Price:       num.NewUint(1200),
			Side:        types.SideSell,
			Party:       party1,
			TimeInForce: types.OrderTimeInForceGTC,
		},
	}

	// submit the auctions orders
	tm.WithSubmittedOrders(t, auctionOrders...)

	// update the time to get out of auction
	tm.market.OnTick(context.Background(), auctionEnd)

	assert.Equal(t,
		tm.market.GetMarketData().MarketTradingMode,
		types.MarketTradingModeContinuous,
	)

	if setMarkPrice {
		// now set the markprice
		mpOrders := []*types.Order{
			{
				Type:        types.OrderTypeLimit,
				Size:        1,
				Remaining:   1,
				Price:       num.NewUint(900),
				Side:        types.SideSell,
				Party:       party1,
				TimeInForce: types.OrderTimeInForceGTC,
			},
			{
				Type:        types.OrderTypeLimit,
				Size:        1,
				Remaining:   1,
				Price:       num.NewUint(1200),
				Side:        types.SideBuy,
				Party:       party0,
				TimeInForce: types.OrderTimeInForceGTC,
			},
		}
		// submit the auctions orders
		tm.WithSubmittedOrders(t, mpOrders...)
	}
}

func mustOrderFromProto(o *vegapb.Order) *types.Order {
	order, _ := types.OrderFromProto(o)
	return order
}
