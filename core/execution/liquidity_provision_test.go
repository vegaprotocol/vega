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
	"fmt"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/core/execution"
	"code.vegaprotocol.io/vega/core/idgeneration"

	vegacontext "code.vegaprotocol.io/vega/libs/context"
	vgcrypto "code.vegaprotocol.io/vega/libs/crypto"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	proto "code.vegaprotocol.io/vega/protos/vega"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newLiquidityOrder(reference types.PeggedReference, offset uint64, proportion uint32) *types.LiquidityOrder {
	return &types.LiquidityOrder{
		Reference:  reference,
		Proportion: proportion,
		Offset:     num.NewUint(offset),
	}
}

func TestSubmit(t *testing.T) {
	pMonitorSettings := &types.PriceMonitoringSettings{
		Parameters: &types.PriceMonitoringParameters{
			Triggers: []*types.PriceMonitoringTrigger{},
		},
	}
	now := time.Unix(10, 0)
	block := time.Second
	ctx := vegacontext.WithTraceID(context.Background(), vgcrypto.RandomHash())

	t.Run("check that we reject LP submission If fee is incorrect", func(t *testing.T) {
		tm := getTestMarket(t, now, nil, nil)

		// Create a new party account with very little funding
		addAccountWithAmount(tm, "party-A", 100000000)
		tm.broker.EXPECT().Send(gomock.Any()).AnyTimes()

		// Start the opening auction
		tm.mas.StartOpeningAuction(tm.now, &types.AuctionDuration{Duration: 10})
		tm.mas.AuctionStarted(ctx, tm.now)
		tm.market.EnterAuction(ctx)

		buys := []*types.LiquidityOrder{
			newLiquidityOrder(types.PeggedReferenceBestBid, 10, 50),
			newLiquidityOrder(types.PeggedReferenceBestBid, 20, 50),
		}
		sells := []*types.LiquidityOrder{
			newLiquidityOrder(types.PeggedReferenceBestAsk, 10, 50),
			newLiquidityOrder(types.PeggedReferenceBestAsk, 20, 50),
		}

		// Submitting a zero or smaller fee should cause a reject
		lps := &types.LiquidityProvisionSubmission{
			Fee:              num.DecimalFromFloat(-0.50),
			MarketID:         tm.market.GetID(),
			CommitmentAmount: num.NewUint(1000),
			Buys:             buys,
			Sells:            sells,
		}

		err := tm.market.SubmitLiquidityProvision(ctx, lps, "party-A", vgcrypto.RandomHash())
		require.Error(t, err)
		assert.Equal(t, 0, tm.market.GetLPSCount())

		// Submitting a fee greater than 1.0 should cause a reject
		lps = &types.LiquidityProvisionSubmission{
			Fee:              num.DecimalFromFloat(1.01),
			MarketID:         tm.market.GetID(),
			CommitmentAmount: num.NewUint(1000),
			Buys:             buys,
			Sells:            sells,
		}

		err = tm.market.SubmitLiquidityProvision(ctx, lps, "party-A", vgcrypto.RandomHash())
		require.Error(t, err)
		assert.Equal(t, 0, tm.market.GetLPSCount())
	})

	t.Run("check that we reject LP submission if side is missing", func(t *testing.T) {
		tm := getTestMarket(t, now, nil, nil)
		ctx := context.Background()

		// Create a new party account with very little funding
		addAccountWithAmount(tm, "party-A", 100000000)
		tm.broker.EXPECT().Send(gomock.Any()).AnyTimes()

		// Start the opening auction
		tm.mas.StartOpeningAuction(tm.now, &types.AuctionDuration{Duration: 10})
		tm.mas.AuctionStarted(ctx, tm.now)
		tm.market.EnterAuction(ctx)

		buys := []*types.LiquidityOrder{
			newLiquidityOrder(types.PeggedReferenceBestBid, 10, 50),
			newLiquidityOrder(types.PeggedReferenceBestBid, 20, 50),
		}
		sells := []*types.LiquidityOrder{
			newLiquidityOrder(types.PeggedReferenceBestAsk, 10, 50),
			newLiquidityOrder(types.PeggedReferenceBestAsk, 20, 50),
		}

		// Submitting a shape with no buys should cause a reject
		lps := &types.LiquidityProvisionSubmission{
			Fee:              num.DecimalFromFloat(0.01),
			MarketID:         tm.market.GetID(),
			CommitmentAmount: num.NewUint(1000),
			Sells:            sells,
		}

		err := tm.market.SubmitLiquidityProvision(ctx, lps, "party-A", vgcrypto.RandomHash())
		require.Error(t, err)
		assert.Equal(t, 0, tm.market.GetLPSCount())

		// Submitting a shape with no sells should cause a reject
		lps = &types.LiquidityProvisionSubmission{
			Fee:              num.DecimalFromFloat(0.01),
			MarketID:         tm.market.GetID(),
			CommitmentAmount: num.NewUint(1000),
			Buys:             buys,
		}

		err = tm.market.SubmitLiquidityProvision(ctx, lps, "party-A", vgcrypto.RandomHash())
		require.Error(t, err)
		assert.Equal(t, 0, tm.market.GetLPSCount())
	})

	// We have a limit to the number of orders in each shape of a liquidity provision submission
	// to prevent a user spaming the system. Place an LPSubmission order with too many
	// orders in to make it reject it.
	t.Run("check for too many shape levels", func(t *testing.T) {
		tm := getTestMarket(t, now, nil, nil)

		// Create a new party account with very little funding
		addAccountWithAmount(tm, "party-A", 10000000)
		tm.broker.EXPECT().Send(gomock.Any()).AnyTimes()

		// Start the opening auction
		tm.mas.StartOpeningAuction(tm.now, &types.AuctionDuration{Duration: 10})
		tm.mas.AuctionStarted(ctx, tm.now)
		tm.market.EnterAuction(ctx)

		// Create a buy side that has too many items
		buys := make([]*types.LiquidityOrder, 200)
		for i := 0; i < 200; i++ {
			buys[i] = newLiquidityOrder(types.PeggedReferenceBestBid, 10+1, 1)
		}

		sells := []*types.LiquidityOrder{
			newLiquidityOrder(types.PeggedReferenceBestAsk, 10, 50),
			newLiquidityOrder(types.PeggedReferenceBestAsk, 20, 50),
		}

		// Submitting a correct entry
		lps := &types.LiquidityProvisionSubmission{
			Fee:              num.DecimalFromFloat(0.01),
			MarketID:         tm.market.GetID(),
			CommitmentAmount: num.NewUint(1000),
			Buys:             buys,
			Sells:            sells,
		}

		err := tm.market.SubmitLiquidityProvision(ctx, lps, "party-A", vgcrypto.RandomHash())
		require.EqualError(t, err, "SIDE_BUY shape size exceed max (5)")
		assert.Equal(t, 0, tm.market.GetLPSCount())
	})

	t.Run("test liquidity provision fee validation", func(t *testing.T) {
		// auctionEnd := now.Add(10001 * time.Second)
		pMonitorSettings := &types.PriceMonitoringSettings{
			Parameters: &types.PriceMonitoringParameters{
				Triggers: []*types.PriceMonitoringTrigger{},
			},
		}
		mktCfg := getMarket(pMonitorSettings, &types.AuctionDuration{
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
			WithAccountAndAmount(lpparty, 50000000000000)

		tm.market.OnSuppliedStakeToObligationFactorUpdate(num.DecimalFromFloat(1.0))
		tm.market.OnTick(ctx, tm.now)

		// Add a LPSubmission
		// this is a log of stake, enough to cover all
		// the required stake for the market
		lpSubmission := &types.LiquidityProvisionSubmission{
			MarketID:         tm.market.GetID(),
			CommitmentAmount: num.NewUint(70000),
			Fee:              num.DecimalFromFloat(-0.1),
			Reference:        "ref-lp-submission-1",
			Buys: []*types.LiquidityOrder{
				newLiquidityOrder(types.PeggedReferenceBestBid, 5, 2),
				newLiquidityOrder(types.PeggedReferenceMid, 5, 2),
			},
			Sells: []*types.LiquidityOrder{
				newLiquidityOrder(types.PeggedReferenceBestAsk, 5, 13),
				newLiquidityOrder(types.PeggedReferenceBestAsk, 5, 13),
			},
		}

		// submit our lp
		require.EqualError(t,
			tm.market.SubmitLiquidityProvision(
				ctx, lpSubmission, lpparty, vgcrypto.RandomHash()),
			"invalid liquidity provision fee",
		)

		lpSubmission.Fee = num.DecimalFromFloat(10)

		// submit our lp
		require.EqualError(t,
			tm.market.SubmitLiquidityProvision(
				ctx, lpSubmission, lpparty, vgcrypto.RandomHash()),
			"invalid liquidity provision fee",
		)

		lpSubmission.Fee = num.DecimalZero()

		// submit our lp
		require.NoError(t,
			tm.market.SubmitLiquidityProvision(
				ctx, lpSubmission, lpparty, vgcrypto.RandomHash()),
		)
	})

	// When a liquidity provider submits an order and runs out of margin from both their general
	// and margin account, the system should take the required amount from the bond account.
	t.Run("check that bond account used to fund short fall in initial margin", func(t *testing.T) {
		tm := getTestMarket(t, now, nil, nil)

		// Create a new party account with very little funding
		addAccountWithAmount(tm, "party-A", 5000)
		addAccountWithAmount(tm, "party-B", 10000000)
		addAccountWithAmount(tm, "party-C", 10000000)
		tm.broker.EXPECT().Send(gomock.Any()).AnyTimes()

		tm.mas.StartOpeningAuction(tm.now, &types.AuctionDuration{Duration: 10})
		tm.mas.AuctionStarted(ctx, tm.now)
		tm.market.EnterAuction(ctx)

		// Create some normal orders to set the reference prices
		o1 := getMarketOrder(tm, tm.now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "Order01", types.SideBuy, "party-B", 10, 10)
		o1conf, err := tm.market.SubmitOrder(ctx, o1)
		require.NotNil(t, o1conf)
		require.NoError(t, err)

		o2 := getMarketOrder(tm, tm.now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "Order02", types.SideSell, "party-C", 2, 10)
		o2conf, err := tm.market.SubmitOrder(ctx, o2)
		require.NotNil(t, o2conf)
		require.NoError(t, err)

		o3 := getMarketOrder(tm, tm.now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "Order03", types.SideSell, "party-C", 1, 20)
		o3conf, err := tm.market.SubmitOrder(ctx, o3)
		require.NotNil(t, o3conf)
		require.NoError(t, err)

		buys := []*types.LiquidityOrder{
			newLiquidityOrder(types.PeggedReferenceBestBid, 1, 50),
			newLiquidityOrder(types.PeggedReferenceBestBid, 2, 50),
		}
		sells := []*types.LiquidityOrder{
			newLiquidityOrder(types.PeggedReferenceBestAsk, 1, 50),
			newLiquidityOrder(types.PeggedReferenceBestAsk, 2, 50),
		}

		// Submitting a correct entry
		lps := &types.LiquidityProvisionSubmission{
			Fee:              num.DecimalFromFloat(0.01),
			MarketID:         tm.market.GetID(),
			CommitmentAmount: num.NewUint(1000),
			Buys:             buys,
			Sells:            sells,
		}

		err = tm.market.SubmitLiquidityProvision(ctx, lps, "party-A", vgcrypto.RandomHash())
		require.NoError(t, err)

		// Check we have the right amount of bond balance
		assert.Equal(t, num.NewUint(1000), tm.market.GetBondAccountBalance(ctx, "party-A", tm.market.GetID(), tm.asset))

		// Leave auction
		tm.now = tm.now.Add(20 * block)
		tm.market.LeaveAuctionWithIDGen(ctx, tm.now, newTestIDGenerator())

		// Check we have an accepted LP submission
		assert.Equal(t, 1, tm.market.GetLPSCount())

		// Check we have the right number of live orders
		assert.Equal(t, int64(6), tm.market.GetOrdersOnBookCount())

		// Check that the bond balance has been reduced
		assert.True(t, tm.market.GetBondAccountBalance(ctx, "party-A", tm.market.GetID(), tm.asset).LT(num.NewUint(1000)))
	})

	// When a liquidity provider has a position that requires more margin after a MTM settlement,
	// they should use the assets in the bond account after the general and margin account are empty.
	t.Run("check that bond account used to fund short fall in maintenance margin", func(t *testing.T) {
		tm := getTestMarket(t, now, nil, nil)

		// Create a new party account with very little funding
		addAccountWithAmount(tm, "party-A", 7000)
		addAccountWithAmount(tm, "party-B", 10000000)
		addAccountWithAmount(tm, "party-C", 10000000)
		tm.broker.EXPECT().Send(gomock.Any()).AnyTimes()

		tm.mas.StartOpeningAuction(tm.now, &types.AuctionDuration{Duration: 10})
		tm.mas.AuctionStarted(ctx, tm.now)
		tm.market.EnterAuction(ctx)

		// Create some normal orders to set the reference prices
		o1 := getMarketOrder(tm, tm.now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "Order01", types.SideBuy, "party-B", 10, 10)
		o1conf, err := tm.market.SubmitOrder(ctx, o1)
		require.NotNil(t, o1conf)
		require.NoError(t, err)

		o2 := getMarketOrder(tm, tm.now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "Order02", types.SideSell, "party-C", 2, 10)
		o2conf, err := tm.market.SubmitOrder(ctx, o2)
		require.NotNil(t, o2conf)
		require.NoError(t, err)

		o3 := getMarketOrder(tm, tm.now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "Order03", types.SideSell, "party-C", 1, 20)
		o3conf, err := tm.market.SubmitOrder(ctx, o3)
		require.NotNil(t, o3conf)
		require.NoError(t, err)

		o31 := getMarketOrder(tm, tm.now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "Order031", types.SideSell, "party-C", 1, 30)
		o31conf, err := tm.market.SubmitOrder(ctx, o31)
		require.NotNil(t, o31conf)
		require.NoError(t, err)

		buys := []*types.LiquidityOrder{
			newLiquidityOrder(types.PeggedReferenceBestBid, 1, 50),
			newLiquidityOrder(types.PeggedReferenceMid, 6, 50),
		}
		sells := []*types.LiquidityOrder{
			newLiquidityOrder(types.PeggedReferenceBestAsk, 1, 50),
			newLiquidityOrder(types.PeggedReferenceMid, 6, 50),
		}

		// Submitting a correct entry
		lps := &types.LiquidityProvisionSubmission{
			Fee:              num.DecimalFromFloat(0.01),
			MarketID:         tm.market.GetID(),
			CommitmentAmount: num.NewUint(1000),
			Buys:             buys,
			Sells:            sells,
		}

		err = tm.market.SubmitLiquidityProvision(ctx, lps, "party-A", vgcrypto.RandomHash())
		require.NoError(t, err)

		// Check we have the right amount of bond balance
		assert.Equal(t, num.NewUint(1000), tm.market.GetBondAccountBalance(ctx, "party-A", tm.market.GetID(), tm.asset))

		// Leave auction
		tm.now = tm.now.Add(40 * block)
		tm.market.OnTick(ctx, tm.now)
		tm.market.LeaveAuctionWithIDGen(ctx, tm.now, newTestIDGenerator())

		// Check we have an accepted LP submission
		assert.Equal(t, 1, tm.market.GetLPSCount())

		// Check we have the right number of live orders
		assert.Equal(t, int64(7), tm.market.GetOrdersOnBookCount())

		// Check that the bond balance is untouched
		assert.True(t, tm.market.GetBondAccountBalance(ctx, "party-A", tm.market.GetID(), tm.asset).EQ(num.NewUint(1000)))

		tm.events = nil
		// Now move the mark price to force MTM settlement
		o4 := getMarketOrder(tm, tm.now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "Order04", types.SideBuy, "party-B", 1, 20)
		o4conf, err := tm.market.SubmitOrder(ctx, o4)
		// tm.now = tm.now.Add(time.Second)
		tm.market.OnTick(ctx, tm.now)
		require.NotNil(t, o4conf)
		require.NoError(t, err)

		t.Run("expect bond slashing transfer", func(t *testing.T) {
			// First collect all the orders events
			found := []*proto.LedgerMovement{}
			for _, e := range tm.events {
				switch evt := e.(type) {
				case *events.LedgerMovements:
					for _, v := range evt.LedgerMovements() {
						for _, t := range v.Entries {
							if t.Type == types.TransferTypeBondSlashing {
								found = append(found, v)
							}
						}
					}
				}
			}

			// @TODO figure out why this doesn't happen anymore
			assert.Len(t, found, 0)
		})
	})

	t.Run("check we can submit LP during price auction", func(t *testing.T) {
		hdec := num.DecimalFromFloat(60)
		pMonitorSettings := &types.PriceMonitoringSettings{
			Parameters: &types.PriceMonitoringParameters{
				Triggers: []*types.PriceMonitoringTrigger{
					{
						Horizon:          60,
						HorizonDec:       hdec,
						Probability:      num.DecimalFromFloat(0.15),
						AuctionExtension: 60,
					},
				},
			},
		}

		tm := getTestMarket(t, now, pMonitorSettings, nil)
		tm.market.OnMarketAuctionMinimumDurationUpdate(ctx, 10*time.Second)

		// Create a new party account with very little funding
		addAccountWithAmount(tm, "party-A", 70000000)
		addAccountWithAmount(tm, "party-B", 10000000)
		addAccountWithAmount(tm, "party-C", 10000000)
		tm.broker.EXPECT().Send(gomock.Any()).AnyTimes()

		tm.mas.StartOpeningAuction(tm.now, &types.AuctionDuration{Duration: 10})
		tm.mas.AuctionStarted(ctx, tm.now)
		tm.market.EnterAuction(ctx)

		// ensure LP is set
		addSimpleLP(t, tm, 5000000)
		// Create some normal orders to set the reference prices
		o1 := getMarketOrder(tm, tm.now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "Order01", types.SideBuy, "party-B", 10, 1000)
		o1conf, err := tm.market.SubmitOrder(ctx, o1)
		require.NotNil(t, o1conf)
		require.NoError(t, err)

		o2 := getMarketOrder(tm, tm.now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "Order02", types.SideSell, "party-C", 2, 1000)
		o2conf, err := tm.market.SubmitOrder(ctx, o2)
		require.NotNil(t, o2conf)
		require.NoError(t, err)

		o3 := getMarketOrder(tm, tm.now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "Order03", types.SideSell, "party-C", 1, 2000)
		o3conf, err := tm.market.SubmitOrder(ctx, o3)
		require.NotNil(t, o3conf)
		require.NoError(t, err)

		o4 := getMarketOrder(tm, tm.now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "Order04", types.SideSell, "party-C", 10, 3000)
		o4conf, err := tm.market.SubmitOrder(ctx, o4)
		require.NotNil(t, o4conf)
		require.NoError(t, err)

		assert.Equal(t, types.AuctionTriggerOpening, tm.market.GetMarketData().Trigger)
		// Leave the auction so we can uncross the book
		tm.now = tm.now.Add(block * 11)
		tm.market.OnTick(ctx, tm.now)
		// ensure we left auction
		assert.Equal(t, types.AuctionTriggerUnspecified, tm.market.GetMarketData().Trigger)

		// Move the price enough that we go into a price auction
		o5 := getMarketOrder(tm, tm.now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "Order05", types.SideBuy, "party-B", 3, 3000)
		o5conf, err := tm.market.SubmitOrder(ctx, o5)
		require.NotNil(t, o5conf)
		require.NoError(t, err)

		// Check we are in price auction
		assert.Equal(t, types.AuctionTriggerPrice, tm.market.GetMarketData().Trigger)

		// Now try to submit a LP submission
		buys := []*types.LiquidityOrder{
			newLiquidityOrder(types.PeggedReferenceBestBid, 1, 50),
			newLiquidityOrder(types.PeggedReferenceMid, 2, 50),
		}
		sells := []*types.LiquidityOrder{
			newLiquidityOrder(types.PeggedReferenceBestAsk, 1, 50),
			newLiquidityOrder(types.PeggedReferenceMid, 2, 50),
		}

		// Submitting a correct entry
		lps := &types.LiquidityProvisionSubmission{
			Fee:              num.DecimalFromFloat(0.01),
			MarketID:         tm.market.GetID(),
			CommitmentAmount: num.NewUint(1000),
			Buys:             buys,
			Sells:            sells,
		}

		err = tm.market.SubmitLiquidityProvision(ctx, lps, "party-A", vgcrypto.RandomHash())
		require.NoError(t, err)
		require.Equal(t, types.LiquidityProvisionStatusPending.String(), tm.market.GetLPSState("party-A").String())
		// Only 3 pegged orders as one fails due to price monitoring
		assert.Equal(t, 0, tm.market.GetPeggedOrderCount())
	})

	t.Run("check that existing pegged orders count towards commitment", func(t *testing.T) {
		tm := getTestMarket(t, now, nil, nil)

		// Create a new party account with very little funding
		addAccountWithAmount(tm, "party-A", 7000)
		addAccountWithAmount(tm, "party-B", 10000000)
		addAccountWithAmount(tm, "party-C", 10000000)
		tm.broker.EXPECT().Send(gomock.Any()).AnyTimes()

		tm.mas.StartOpeningAuction(tm.now, &types.AuctionDuration{Duration: 10})
		tm.mas.AuctionStarted(ctx, tm.now)
		tm.market.EnterAuction(ctx)

		// Create some normal orders to set the reference prices
		o1 := getMarketOrder(tm, tm.now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "Order01", types.SideBuy, "party-B", 10, 10)
		o1conf, err := tm.market.SubmitOrder(ctx, o1)
		require.NotNil(t, o1conf)
		require.NoError(t, err)

		o2 := getMarketOrder(tm, tm.now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "Order02", types.SideSell, "party-C", 2, 10)
		o2conf, err := tm.market.SubmitOrder(ctx, o2)
		require.NotNil(t, o2conf)
		require.NoError(t, err)

		o3 := getMarketOrder(tm, tm.now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "Order03", types.SideSell, "party-C", 1, 20)
		o3conf, err := tm.market.SubmitOrder(ctx, o3)
		require.NotNil(t, o3conf)
		require.NoError(t, err)

		// Add a manual pegged order which should be included in commitment calculations
		pegged := getMarketOrder(tm, tm.now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "Peggy", types.SideBuy, "party-A", 1, 0)
		pegged.PeggedOrder = &types.PeggedOrder{Reference: types.PeggedReferenceBestBid, Offset: num.NewUint(2)}
		peggedconf, err := tm.market.SubmitOrder(ctx, pegged)
		require.NotNil(t, peggedconf)
		require.NoError(t, err)

		buys := []*types.LiquidityOrder{
			newLiquidityOrder(types.PeggedReferenceBestBid, 1, 50),
			newLiquidityOrder(types.PeggedReferenceMid, 6, 50),
		}
		sells := []*types.LiquidityOrder{
			newLiquidityOrder(types.PeggedReferenceBestAsk, 1, 50),
			newLiquidityOrder(types.PeggedReferenceMid, 6, 50),
		}

		// Submitting a correct entry
		lps := &types.LiquidityProvisionSubmission{
			Fee:              num.DecimalFromFloat(0.01),
			MarketID:         tm.market.GetID(),
			CommitmentAmount: num.NewUint(1000),
			Buys:             buys,
			Sells:            sells,
		}

		err = tm.market.SubmitLiquidityProvision(ctx, lps, "party-A", vgcrypto.RandomHash())
		require.NoError(t, err)
		require.Equal(t, types.LiquidityProvisionStatusPending.String(), tm.market.GetLPSState("party-A").String())
		assert.Equal(t, 1, tm.market.GetPeggedOrderCount())
		assert.Equal(t, 1, tm.market.GetParkedOrderCount())

		// Leave the auction so we can uncross the book
		tm.now = tm.now.Add(20 * block)
		tm.market.LeaveAuctionWithIDGen(ctx, tm.now, newTestIDGenerator())
		tm.market.OnTick(ctx, tm.now)
		assert.Equal(t, 1, tm.market.GetPeggedOrderCount())
		assert.Equal(t, 0, tm.market.GetParkedOrderCount())

		// TODO Check that the liquidity provision has taken into account the pegged order we already had
	})

	// When a price monitoring auction is started, make sure we cancel all the pegged orders and
	// that no fees are charged to the liquidity providers.
	t.Run("check that no penality when going into price auction", func(t *testing.T) {
		pMonitorSettings := &types.PriceMonitoringSettings{
			Parameters: &types.PriceMonitoringParameters{
				Triggers: []*types.PriceMonitoringTrigger{
					{
						Horizon:          60,
						HorizonDec:       num.DecimalFromFloat(60),
						Probability:      num.DecimalFromFloat(0.95),
						AuctionExtension: 60,
					},
				},
			},
		}

		tm := getTestMarket(t, now, pMonitorSettings, &types.AuctionDuration{Duration: 10})
		tm.market.OnMarketAuctionMinimumDurationUpdate(ctx, time.Second*10)

		// Create a new party account with very little funding
		addAccountWithAmount(tm, "party-A", 700000)
		addAccountWithAmount(tm, "party-B", 10000000)
		addAccountWithAmount(tm, "party-C", 10000000)
		tm.broker.EXPECT().Send(gomock.Any()).AnyTimes()

		// Create some normal orders to set the reference prices
		o1 := getMarketOrder(tm, tm.now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "Order01", types.SideBuy, "party-B", 10, 1000)
		o1conf, err := tm.market.SubmitOrder(ctx, o1)
		require.NotNil(t, o1conf)
		require.NoError(t, err)

		o2 := getMarketOrder(tm, tm.now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "Order02", types.SideSell, "party-C", 2, 1000)
		o2conf, err := tm.market.SubmitOrder(ctx, o2)
		require.NotNil(t, o2conf)
		require.NoError(t, err)

		o3 := getMarketOrder(tm, tm.now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "Order03", types.SideSell, "party-C", 1, 2000)
		o3conf, err := tm.market.SubmitOrder(ctx, o3)
		require.NotNil(t, o3conf)
		require.NoError(t, err)

		o4 := getMarketOrder(tm, tm.now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "Order04", types.SideSell, "party-C", 10, 3000)
		o4conf, err := tm.market.SubmitOrder(ctx, o4)
		require.NotNil(t, o4conf)
		require.NoError(t, err)

		// Submit a LP submission
		buys := []*types.LiquidityOrder{newLiquidityOrder(types.PeggedReferenceBestBid, 500, 50)}
		sells := []*types.LiquidityOrder{newLiquidityOrder(types.PeggedReferenceBestAsk, 500, 50)}

		lps := &types.LiquidityProvisionSubmission{
			Fee:              num.DecimalFromFloat(0.01),
			MarketID:         tm.market.GetID(),
			CommitmentAmount: num.NewUint(5000),
			Buys:             buys,
			Sells:            sells,
		}

		err = tm.market.SubmitLiquidityProvision(ctx, lps, "party-A", vgcrypto.RandomHash())
		require.NoError(t, err)
		require.Equal(t, types.LiquidityProvisionStatusPending.String(), tm.market.GetLPSState("party-A").String())
		// Leave the auction so we can uncross the book
		tm.now = tm.now.Add(20 * block)
		tm.market.OnTick(ctx, tm.now)

		// Save the total amount of assets we have in general+margin+bond
		totalFunds := tm.market.GetTotalAccountBalance(ctx, "party-A", tm.market.GetID(), tm.asset)

		// Move the price enough that we go into a price auction
		tm.now = tm.now.Add(20 * time.Second)
		tm.market.OnTick(ctx, tm.now)
		// amount
		mktDat := tm.market.GetMarketData()
		fmt.Printf("Target: %s\nSupplied: %s\n\n", mktDat.TargetStake, mktDat.SuppliedStake)
		fmt.Printf("bounds: %d\n%#v\n", len(mktDat.PriceMonitoringBounds), mktDat)
		for _, pb := range mktDat.PriceMonitoringBounds {
			fmt.Printf("Horizon -> %s - %s \n", pb.MinValidPrice.String(), pb.MaxValidPrice.String())
		}

		o5 := getMarketOrder(tm, tm.now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "Order05", types.SideBuy, "party-B", 50, 3000)
		o5conf, err := tm.market.SubmitOrder(ctx, o5)
		require.NotNil(t, o5conf)
		require.NoError(t, err)

		// Check we are in price auction
		assert.Equal(t, types.AuctionTriggerPrice, tm.market.GetMarketData().Trigger, tm.market.GetMarketData().Trigger.String())

		// All pegged orders must be removed
		// TODO assert.Equal(t, 0, tm.market.GetPeggedOrderCount())
		// TODO assert.Equal(t, 0, tm.market.GetParkedOrderCount())

		// Check we have not lost any assets
		assert.Equal(t, totalFunds, tm.market.GetTotalAccountBalance(ctx, "party-A", tm.market.GetID(), tm.asset))
	})

	t.Run("check that LP cannot get closed out when deploying order for the first time", func(t *testing.T) {
		auctionEnd := now.Add(10001 * time.Second)

		mktCfg := getMarket(pMonitorSettings, &types.AuctionDuration{
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

		var (
			lpparty = "lp-party-1"
			party0  = "party-0"
			party1  = "party-1"
		)
		tm := newTestMarket(t, now).Run(ctx, mktCfg)
		tm.StartOpeningAuction().
			// the liquidity provider
			WithAccountAndAmount(lpparty, 200000).
			WithAccountAndAmount(party1, 1000000000).
			WithAccountAndAmount(party0, 1000000000)

		tm.market.OnSuppliedStakeToObligationFactorUpdate(num.DecimalFromFloat(.3))
		tm.now = now
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
				newLiquidityOrder(types.PeggedReferenceBestBid, 5, 2),
				newLiquidityOrder(types.PeggedReferenceMid, 5, 2),
			},
			Sells: []*types.LiquidityOrder{
				newLiquidityOrder(types.PeggedReferenceBestAsk, 5, 13),
				newLiquidityOrder(types.PeggedReferenceBestAsk, 5, 13),
			},
		}

		// submit our lp
		lpID := vgcrypto.RandomHash()
		tm.events = nil
		require.NoError(t,
			tm.market.SubmitLiquidityProvision(
				ctx, lpSubmission, lpparty, lpID),
		)

		tm.events = nil
		tm.EndOpeningAuction(t, auctionEnd, false)

		// make sure LP order is deployed
		t.Run("expect commitment statuses", func(t *testing.T) {
			// First collect all the orders events
			found := map[string]*proto.LiquidityProvision{}
			for _, e := range tm.events {
				switch evt := e.(type) {
				case *events.LiquidityProvision:
					lp := evt.LiquidityProvision()
					found[lp.Id] = lp
				}
			}

			expectedStatus := map[string]types.LiquidityProvisionStatus{
				lpID: types.LiquidityProvisionStatusCancelled,
			}

			require.Len(t, found, len(expectedStatus))

			for k, v := range expectedStatus {
				assert.Equal(t, v.String(), found[k].Status.String())
			}
		})
	})

	t.Run("test closed out LP party cont issue 3086", func(t *testing.T) {
		auctionEnd := now.Add(10001 * time.Second)
		mktCfg := getMarket(pMonitorSettings, &types.AuctionDuration{
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

		var (
			ruser1 = "ruser_1"
			ruser2 = "ruser_2"
			ruser3 = "ruser3"
		)
		tm := newTestMarket(t, now).Run(ctx, mktCfg)
		tm.StartOpeningAuction().
			// the liquidity provider
			WithAccountAndAmount(ruser1, 500000).
			WithAccountAndAmount(ruser2, 74490).
			WithAccountAndAmount(ruser3, 10000000)

		tm.market.OnSuppliedStakeToObligationFactorUpdate(num.DecimalFromFloat(.2))
		tm.now = now
		tm.market.OnTick(ctx, now)

		// Add a LPSubmission
		// this is a log of stake, enough to cover all
		// the required stake for the market
		lpSubmission := &types.LiquidityProvisionSubmission{
			MarketID:         tm.market.GetID(),
			CommitmentAmount: num.NewUint(50490),
			Fee:              num.DecimalFromFloat(0.01),
			Reference:        "ref-lp-submission-1",
			Buys: []*types.LiquidityOrder{
				newLiquidityOrder(types.PeggedReferenceBestBid, 5, 2),
			},
			Sells: []*types.LiquidityOrder{
				newLiquidityOrder(types.PeggedReferenceBestAsk, 5, 2),
			},
		}

		// submit our lp
		tm.events = nil
		lpID := vgcrypto.RandomHash()
		require.NoError(t,
			tm.market.SubmitLiquidityProvision(
				ctx, lpSubmission, ruser2, lpID),
		)

		tm.events = nil
		tm.EndOpeningAuction(t, auctionEnd, false)

		// make sure LP order is deployed
		t.Run("new LP order is active", func(t *testing.T) {
			// First collect all the orders events
			found := map[string]*proto.LiquidityProvision{}
			for _, e := range tm.events {
				switch evt := e.(type) {
				case *events.LiquidityProvision:
					lp := evt.LiquidityProvision()
					found[lp.Id] = lp
				}
			}

			expectedStatus := map[string]types.LiquidityProvisionStatus{
				lpID: types.LiquidityProvisionStatusActive,
			}

			require.Len(t, found, len(expectedStatus))

			for k, v := range expectedStatus {
				assert.Equal(t, v.String(), found[k].Status.String())
			}
		})

		// now add a couple of orders
		// now set the markprice
		mpOrders := []*types.Order{
			{
				Type:        types.OrderTypeLimit,
				Size:        3,
				Remaining:   3,
				Price:       num.NewUint(5000),
				Side:        types.SideSell,
				Party:       ruser3,
				TimeInForce: types.OrderTimeInForceGTC,
			},
			{
				Type:        types.OrderTypeLimit,
				Size:        2,
				Remaining:   2,
				Price:       num.NewUint(4500),
				Side:        types.SideSell,
				Party:       ruser2,
				TimeInForce: types.OrderTimeInForceGTC,
			},
			{
				Type:        types.OrderTypeLimit,
				Size:        4,
				Remaining:   4,
				Price:       num.NewUint(4500),
				Side:        types.SideBuy,
				Party:       ruser1,
				TimeInForce: types.OrderTimeInForceGTC,
			},
		}

		tm.events = nil
		mktD := tm.market.GetMarketData()
		fmt.Printf("TS: %s\nSS: %s\n", mktD.TargetStake, mktD.SuppliedStake)
		// submit the auctions orders
		tm.WithSubmittedOrders(t, mpOrders...)

		/*
			// check accounts
			t.Run("margin account is updated", func(t *testing.T) {
				_, err := tm.collateralEngine.GetPartyMarginAccount(
					tm.market.GetID(), ruser2, tm.asset)
				require.Error(t, err)
				assert.Contains(t, err.Error(), "account does not exist:")
			})

			t.Run("bond account", func(t *testing.T) {
				acc, err := tm.collateralEngine.GetPartyBondAccount(
					tm.market.GetID(), ruser2, tm.asset)
				assert.NoError(t, err)
				assert.True(t, acc.Balance.IsZero())
			})

			t.Run("general account", func(t *testing.T) {
				acc, err := tm.collateralEngine.GetPartyGeneralAccount(
					ruser2, tm.asset)
				assert.NoError(t, err)
				assert.True(t, acc.Balance.IsZero())
			})

			// deposit funds again for this party
			tm.WithAccountAndAmount(ruser2, 90000000)

			t.Run("general account", func(t *testing.T) {
				acc, err := tm.collateralEngine.GetPartyGeneralAccount(
					ruser2, tm.asset)
				assert.NoError(t, err)
				assert.Equal(t, num.NewUint(90000000), acc.Balance)
			})

			lpSubmission2 := &types.LiquidityProvisionSubmission{
				MarketID:         tm.market.GetID(),
				CommitmentAmount: num.NewUint(200000),
				Fee:              num.DecimalFromFloat(0.01),
				Reference:        "ref-lp-submission-2",
				Buys: []*types.LiquidityOrder{
					newLiquidityOrder(types.PeggedReferenceMid, 10, 10),
					newLiquidityOrder(types.PeggedReferenceMid, 15, 13),
				},
				Sells: []*types.LiquidityOrder{
					newLiquidityOrder(types.PeggedReferenceBestAsk, 20, 10),
					newLiquidityOrder(types.PeggedReferenceBestAsk, 10, 13),
				},
			}

			tm.events = nil
			lpID2 := vgcrypto.RandomHash()
			require.NoError(t,
				tm.market.SubmitLiquidityProvision(
					ctx, lpSubmission2, ruser2, lpID2),
			)

			// make sure LP order is deployed
			t.Run("new LP order is active", func(t *testing.T) {
				// First collect all the orders events
				found := map[string]*proto.LiquidityProvision{}
				for _, e := range tm.events {
					switch evt := e.(type) {
					case *events.LiquidityProvision:
						lp := evt.LiquidityProvision()
						found[lp.Id] = lp
					}
				}

				expectedStatus := map[string]types.LiquidityProvisionStatus{
					lpID2: types.LiquidityProvisionStatusPending,
				}

				require.Len(t, found, len(expectedStatus))

				for k, v := range expectedStatus {
					assert.Equal(t, v.String(), found[k].Status.String())
				}
			})*/
	})

	t.Run("test liquidity order generated sizes", func(t *testing.T) {
		auctionEnd := now.Add(10001 * time.Second)
		mktCfg := getMarketWithDP(pMonitorSettings, &types.AuctionDuration{
			Duration: 10000,
		}, 0)
		mktCfg.Fees = &types.Fees{
			Factors: &types.FeeFactors{
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
					Sigma: num.DecimalFromFloat(2),
				},
			},
		}

		lpparty := "lp-party-1"
		oth1 := "party1"
		oth2 := "party2"

		tm := newTestMarket(t, now).Run(ctx, mktCfg)
		tm.StartOpeningAuction().
			WithAccountAndAmount(lpparty, 100000000000000).
			WithAccountAndAmount(oth1, 500000000000).
			WithAccountAndAmount(oth2, 500000000000)

		tm.market.OnSuppliedStakeToObligationFactorUpdate(num.DecimalFromFloat(0.7))
		tm.now = now
		tm.market.OnTick(ctx, now)

		// Add a LPSubmission
		// this is a log of stake, enough to cover all
		// the required stake for the market
		lpSubmission := &types.LiquidityProvisionSubmission{
			MarketID:         tm.market.GetID(),
			CommitmentAmount: num.NewUint(10000000),
			Fee:              num.DecimalFromFloat(0.5),
			Reference:        "ref-lp-submission-1",
			Buys: []*types.LiquidityOrder{
				newLiquidityOrder(types.PeggedReferenceBestBid, 201, 99),
				newLiquidityOrder(types.PeggedReferenceBestBid, 200, 1),
			},
			Sells: []*types.LiquidityOrder{
				newLiquidityOrder(types.PeggedReferenceBestAsk, 100, 1),
				newLiquidityOrder(types.PeggedReferenceBestAsk, 101, 2),
				newLiquidityOrder(types.PeggedReferenceBestAsk, 102, 98),
			},
		}

		// submit our lp
		tm.events = nil
		require.NoError(t,
			tm.market.SubmitLiquidityProvision(
				ctx, lpSubmission, lpparty, vgcrypto.RandomHash()),
		)

		t.Run("lp submission is pending", func(t *testing.T) {
			// First collect all the orders events
			var found *proto.LiquidityProvision
			for _, e := range tm.events {
				switch evt := e.(type) {
				case *events.LiquidityProvision:
					found = evt.LiquidityProvision()
				}
			}
			require.NotNil(t, found)
			// no update to the liquidity fee
			assert.Equal(t, found.Status.String(), types.LiquidityProvisionStatusPending.String())
		})

		// then submit some orders, some for the lp party,
		// end some for the other parrties

		lpOrders := []*types.Order{
			// Limit Orders
			{
				Type:        types.OrderTypeLimit,
				Size:        10,
				Remaining:   10,
				Price:       num.NewUint(120000),
				Side:        types.SideBuy,
				Party:       lpparty,
				TimeInForce: types.OrderTimeInForceGTC,
			},
			{
				Type:        types.OrderTypeLimit,
				Size:        10,
				Remaining:   10,
				Price:       num.NewUint(123000),
				Side:        types.SideSell,
				Party:       lpparty,
				TimeInForce: types.OrderTimeInForceGTC,
			},
		}

		// submit the auctions orders
		tm.WithSubmittedOrders(t, lpOrders...)

		// set the mark price and end auction
		auctionOrders := []*types.Order{
			{
				Type:        types.OrderTypeLimit,
				Size:        1,
				Remaining:   1,
				Price:       num.NewUint(121500),
				Side:        types.SideBuy,
				Party:       oth1,
				TimeInForce: types.OrderTimeInForceGTC,
			},
			{
				Type:        types.OrderTypeLimit,
				Size:        1,
				Remaining:   1,
				Price:       num.NewUint(121500),
				Side:        types.SideSell,
				Party:       oth2,
				TimeInForce: types.OrderTimeInForceGTC,
			},
		}

		// submit the auctions orders
		tm.events = nil
		tm.WithSubmittedOrders(t, auctionOrders...)
		// update the time to get out of auction
		ctx := vegacontext.WithTraceID(context.Background(), vgcrypto.RandomHash())
		tm.now = auctionEnd
		tm.market.OnTick(ctx, auctionEnd)

		t.Run("verify LP orders sizes", func(t *testing.T) {
			// First collect all the orders events
			found := map[string]*proto.Order{}
			for _, e := range tm.events {
				switch evt := e.(type) {
				case *events.Order:
					if ord := evt.Order(); ord.PartyId == lpparty {
						found[ord.Id] = ord
					}
				}
			}

			expectedQnts := []struct {
				size  uint64
				found bool
			}{
				{104, false},
				{2, false},
				{2, false},
				{3, false},
				{113, false},
			}

			for _, v := range found {
				for i, expectedQnt := range expectedQnts {
					if v.Size == expectedQnt.size && expectedQnt.found == false {
						expectedQnts[i].found = true
					}
				}
			}

			allExpectedQntsFound := true
			for _, exp := range expectedQnts {
				allExpectedQntsFound = allExpectedQntsFound && exp.found
			}
			assert.True(t, allExpectedQntsFound, "missing expected order quantities")
		})

		newOrders := []*types.Order{
			{
				Type:        types.OrderTypeLimit,
				Size:        1000,
				Remaining:   1000,
				Price:       num.NewUint(121100),
				Side:        types.SideBuy,
				Party:       oth1,
				TimeInForce: types.OrderTimeInForceGTC,
			},
			{
				Type:        types.OrderTypeLimit,
				Size:        1000,
				Remaining:   1000,
				Price:       num.NewUint(122200),
				Side:        types.SideSell,
				Party:       oth2,
				TimeInForce: types.OrderTimeInForceGTC,
			},
		}

		// submit the auctions orders
		tm.events = nil
		tm.WithSubmittedOrders(t, newOrders...)
	})

	t.Run("check that rejected market stops liquidity provision", func(t *testing.T) {
		mktCfg := getMarket(pMonitorSettings, &types.AuctionDuration{
			Duration: 10000,
		})
		mktCfg.Fees = &types.Fees{
			Factors: &types.FeeFactors{
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
					Sigma: num.DecimalFromFloat(2),
				},
			},
		}

		lpparty := "lp-party-1"

		tm := newTestMarket(t, now).Run(ctx, mktCfg)
		tm.WithAccountAndAmount(lpparty, 100000000000000)
		tm.market.OnSuppliedStakeToObligationFactorUpdate(num.DecimalFromFloat(0.7))
		tm.now = now
		tm.market.OnTick(ctx, now)

		// Add a LPSubmission
		// this is a log of stake, enough to cover all
		// the required stake for the market
		lpSubmission := &types.LiquidityProvisionSubmission{
			MarketID:         tm.market.GetID(),
			CommitmentAmount: num.NewUint(10000000),
			Fee:              num.DecimalFromFloat(0.5),
			Reference:        "ref-lp-submission-1",
			Buys: []*types.LiquidityOrder{
				newLiquidityOrder(types.PeggedReferenceBestBid, 200, 1),
			},
			Sells: []*types.LiquidityOrder{
				newLiquidityOrder(types.PeggedReferenceBestAsk, 102, 98),
			},
		}

		// submit our lp
		tm.events = nil
		require.NoError(t,
			tm.market.SubmitLiquidityProvision(
				ctx, lpSubmission, lpparty, vgcrypto.RandomHash()),
		)

		t.Run("lp submission is pending", func(t *testing.T) {
			// First collect all the orders events
			var found *proto.LiquidityProvision
			for _, e := range tm.events {
				switch evt := e.(type) {
				case *events.LiquidityProvision:
					found = evt.LiquidityProvision()
				}
			}
			require.NotNil(t, found)
			// no update to the liquidity fee
			assert.Equal(t, found.Status.String(), types.LiquidityProvisionStatusPending.String())
		})

		tm.events = nil
		require.NoError(
			t,
			tm.market.Reject(context.Background()),
		)

		t.Run("lp submission is stopped", func(t *testing.T) {
			// First collect all the orders events
			var found *proto.LiquidityProvision
			for _, e := range tm.events {
				switch evt := e.(type) {
				case *events.LiquidityProvision:
					found = evt.LiquidityProvision()
				}
			}
			require.NotNil(t, found)
			// no update to the liquidity fee
			assert.Equal(t, found.Status.String(), types.LiquidityProvisionStatusStopped.String())
		})
	})

	t.Run("test park order panic order not found in book", func(t *testing.T) {
		auctionEnd := now.Add(10001 * time.Second)
		mktCfg := getMarket(defaultPriceMonitorSettings, &types.AuctionDuration{
			Duration: 10000,
		})
		mktCfg.Fees = &types.Fees{
			Factors: &types.FeeFactors{
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
					Sigma: num.DecimalFromFloat(10),
				},
			},
		}

		lpparty := "lp-party-1"

		tm := newTestMarket(t, now).Run(ctx, mktCfg)
		tm.StartOpeningAuction().
			WithAccountAndAmount(lpparty, 100000000000000)

		tm.market.OnMarketAuctionMinimumDurationUpdate(ctx, 1*time.Second)
		tm.market.OnSuppliedStakeToObligationFactorUpdate(num.DecimalFromFloat(0.2))
		tm.now = now
		tm.market.OnTick(ctx, now)

		// Add a LPSubmission
		// this is a log of stake, enough to cover all
		// the required stake for the market
		lpSubmission := &types.LiquidityProvisionSubmission{
			MarketID:         tm.market.GetID(),
			CommitmentAmount: num.NewUint(10000000),
			Fee:              num.DecimalFromFloat(0.5),
			Reference:        "ref-lp-submission-1",
			Buys: []*types.LiquidityOrder{
				newLiquidityOrder(types.PeggedReferenceBestBid, 201, 99),
				newLiquidityOrder(types.PeggedReferenceBestBid, 200, 1),
			},
			Sells: []*types.LiquidityOrder{
				newLiquidityOrder(types.PeggedReferenceBestAsk, 100, 1),
				newLiquidityOrder(types.PeggedReferenceBestAsk, 101, 2),
				newLiquidityOrder(types.PeggedReferenceBestAsk, 102, 98),
			},
		}

		// submit our lp
		tm.events = nil
		require.NoError(t,
			tm.market.SubmitLiquidityProvision(
				ctx, lpSubmission, lpparty, vgcrypto.RandomHash()),
		)

		t.Run("lp submission is pending", func(t *testing.T) {
			// First collect all the orders events
			var found *proto.LiquidityProvision
			for _, e := range tm.events {
				switch evt := e.(type) {
				case *events.LiquidityProvision:
					found = evt.LiquidityProvision()
				}
			}
			require.NotNil(t, found)
			// no update to the liquidity fee
			assert.Equal(t, found.Status.String(), types.LiquidityProvisionStatusPending.String())
		})

		// we end the auction

		tm.EndOpeningAuction(t, auctionEnd, false)

		// this are opening auction party
		// we'll cancel their orders
		var (
			party0 = "clearing-auction-party0"
			party1 = "clearing-auction-party1"

			pegged              = "pegged-order-party"
			peggedInitialAmount = num.NewUint(1000000)
			peggedExpiry        = auctionEnd.Add(10 * time.Minute)

			party4 = "party4"
		)

		tm.WithAccountAndAmount(pegged, peggedInitialAmount.Uint64()).
			WithAccountAndAmount(party4, 10000000000)

		// then cancel all remaining orders in the book.
		confs, err := tm.market.CancelAllOrders(context.Background(), party0)
		assert.NoError(t, err)
		assert.Len(t, confs, 1)
		confs, err = tm.market.CancelAllOrders(context.Background(), party1)
		assert.NoError(t, err)
		assert.Len(t, confs, 1)

		// now we place a pegged order, and ensure it's being
		// parked straight away with another party
		peggedO := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTT, "Order01", types.SideSell, pegged, 10, 0)
		peggedO.ExpiresAt = peggedExpiry.UnixNano()
		peggedO.PeggedOrder = &types.PeggedOrder{
			Reference: types.PeggedReferenceBestAsk,
			Offset:    num.NewUint(10),
		}
		peggedOConf, err := tm.market.SubmitOrder(ctx, peggedO)
		assert.NoError(t, err)
		assert.NotNil(t, peggedOConf)

		t.Run("pegged order is PARKED", func(t *testing.T) {
			// First collect all the orders events
			found := &proto.Order{}
			for _, e := range tm.events {
				switch evt := e.(type) {
				case *events.Order:
					found = evt.Order()
				}
			}
			// no update to the liquidity fee
			assert.Equal(t, found.Status.String(), types.OrderStatusParked.String())
		})

		// assert the general account is equal to the initial amount
		t.Run("no general account monies where taken", func(t *testing.T) {
			acc, err := tm.collateralEngine.GetPartyGeneralAccount(pegged, tm.asset)
			assert.NoError(t, err)
			assert.True(t, peggedInitialAmount.EQ(acc.Balance))
		})

		t.Run("withdraw all funds from the general account", func(t *testing.T) {
			acc, err := tm.collateralEngine.GetPartyGeneralAccount(pegged, tm.asset)
			assert.NoError(t, err)
			assert.NoError(t,
				tm.collateralEngine.DecrementBalance(context.Background(), acc.ID, peggedInitialAmount.Clone()),
			)

			// then ensure balance is 0
			acc, err = tm.collateralEngine.GetPartyGeneralAccount(pegged, tm.asset)
			assert.NoError(t, err)
			assert.True(t, acc.Balance.IsZero())
		})

		tm.events = nil
		t.Run("party place a new order which should unpark the pegged order", func(t *testing.T) {
			o := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "Order02", types.SideSell, party4, 10, 2400)
			conf, err := tm.market.SubmitOrder(ctx, o)
			assert.NoError(t, err)
			assert.NotNil(t, conf)
			o2 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "Order02", types.SideBuy, party4, 10, 800)
			conf2, err := tm.market.SubmitOrder(ctx, o2)
			assert.NoError(t, err)
			assert.NotNil(t, conf2)
			tm.now = auctionEnd.Add(10 * time.Second)
			tm.market.OnTick(ctx, auctionEnd.Add(10*time.Second))
		})

		t.Run("pegged order is ACCEPTED", func(t *testing.T) {
			tm.market.OnTick(ctx, tm.now.Add(block))
			tm.now = tm.now.Add(block)
			// First collect all the orders events
			found := &proto.Order{}
			for _, e := range tm.events {
				switch evt := e.(type) {
				case *events.Order:
					order := evt.Order()
					if order.PartyId == pegged {
						found = evt.Order()
					}
				}
			}
			// no update to the liquidity fee
			// assert.Equal(t, found.Status.String(), types.OrderStatusRejected.String())
			assert.Equal(t, found.Status.String(), types.OrderStatusUnspecified.String())
		})

		// now move the time to expire the pegged
		timeExpires := peggedExpiry.Add(1 * time.Hour)
		tm.now = timeExpires
		tm.events = nil
		tm.market.OnTick(ctx, timeExpires)
		t.Run("No orders except pegged order expired", func(t *testing.T) {
			// First collect all the orders events
			orders := []*types.Order{}
			for _, e := range tm.events {
				switch evt := e.(type) {
				case *events.Order:
					if evt.Order().Status == types.OrderStatusExpired {
						orders = append(orders, mustOrderFromProto(evt.Order()))
					}
				}
			}

			require.Len(t, orders, 1)
		})
	})

	t.Run("test lots of pegged and non pegged orders", func(t *testing.T) {
		auctionEnd := now.Add(10001 * time.Second)
		mktCfg := getMarket(defaultPriceMonitorSettings, &types.AuctionDuration{
			Duration: 10000,
		})
		mktCfg.Fees = &types.Fees{
			Factors: &types.FeeFactors{
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
					Sigma: num.DecimalFromFloat(2),
				},
			},
		}

		lpparty := "lp-party-1"

		tm := newTestMarket(t, now).Run(ctx, mktCfg)
		tm.StartOpeningAuction().
			WithAccountAndAmount(lpparty, 100000000000000)

		tm.market.OnMarketAuctionMinimumDurationUpdate(ctx, 1*time.Second)
		tm.market.OnSuppliedStakeToObligationFactorUpdate(num.DecimalFromFloat(0.7))
		tm.now = now
		tm.market.OnTick(ctx, now)

		// Add a LPSubmission
		// this is a log of stake, enough to cover all
		// the required stake for the market
		lpSubmission := &types.LiquidityProvisionSubmission{
			MarketID:         tm.market.GetID(),
			CommitmentAmount: num.NewUint(10000000),
			Fee:              num.DecimalFromFloat(0.5),
			Reference:        "ref-lp-submission-1",
			Buys: []*types.LiquidityOrder{
				newLiquidityOrder(types.PeggedReferenceBestBid, 201, 99),
				newLiquidityOrder(types.PeggedReferenceBestBid, 200, 1),
			},
			Sells: []*types.LiquidityOrder{
				newLiquidityOrder(types.PeggedReferenceBestAsk, 100, 1),
				newLiquidityOrder(types.PeggedReferenceBestAsk, 101, 2),
				newLiquidityOrder(types.PeggedReferenceBestAsk, 102, 98),
			},
		}

		// submit our lp
		tm.events = nil
		require.NoError(t,
			tm.market.SubmitLiquidityProvision(
				ctx, lpSubmission, lpparty, vgcrypto.RandomHash()),
		)

		t.Run("lp submission is pending", func(t *testing.T) {
			// First collect all the orders events
			var found *proto.LiquidityProvision
			for _, e := range tm.events {
				switch evt := e.(type) {
				case *events.LiquidityProvision:
					found = evt.LiquidityProvision()
				}
			}
			require.NotNil(t, found)
			// no update to the liquidity fee
			assert.Equal(t, found.Status.String(), types.LiquidityProvisionStatusPending.String())
		})

		// we end the auction

		tm.EndOpeningAuction(t, auctionEnd, false)

		// this are opening auction party
		// we'll cancel their orders
		var (
			party0 = "clearing-auction-party0"
			party1 = "clearing-auction-party1"
			party2 = "party2"
		)

		tm.WithAccountAndAmount(party2, 100000000000000)

		t.Run("lp submission is pending", func(t *testing.T) {
			// then cancel all remaining orders in the book.
			confs, err := tm.market.CancelAllOrders(context.Background(), party0)
			assert.NoError(t, err)
			assert.Len(t, confs, 1)
			confs, err = tm.market.CancelAllOrders(context.Background(), party1)
			assert.NoError(t, err)
			assert.Len(t, confs, 1)
		})

		curt := auctionEnd.Add(block)
		tm.now = curt
		tm.market.OnTick(ctx, curt)

		t.Run("party submit volume in both side of the book", func(t *testing.T) {
			for i := 0; i < 50; i++ {
				t.Run("buy side", func(t *testing.T) {
					peggedO := getMarketOrder(tm, curt, types.OrderTypeLimit, types.OrderTimeInForceGTC,
						fmt.Sprintf("order-pegged-buy-%v", i), types.SideBuy, party2, 1, 0)
					peggedO.PeggedOrder = &types.PeggedOrder{
						Reference: types.PeggedReferenceBestBid,
						Offset:    num.NewUint(20),
					}
					peggedOConf, err := tm.market.SubmitOrder(ctx, peggedO)
					assert.NoError(t, err)
					assert.NotNil(t, peggedOConf)
					o := getMarketOrder(tm, curt, types.OrderTypeLimit, types.OrderTimeInForceGTC,
						fmt.Sprintf("order-buy-%v", i), types.SideBuy, party2, 1, uint64(1250+(i*10)))
					conf, err := tm.market.SubmitOrder(ctx, o)
					assert.NoError(t, err)
					assert.NotNil(t, conf)
				})

				t.Run("sell side", func(t *testing.T) {
					peggedO := getMarketOrder(tm, curt, types.OrderTypeLimit, types.OrderTimeInForceGTC,
						fmt.Sprintf("order-pegged-sell-%v", i), types.SideSell, party2, 1, 0)
					peggedO.PeggedOrder = &types.PeggedOrder{
						Reference: types.PeggedReferenceBestAsk,
						Offset:    num.NewUint(10),
					}
					peggedOConf, err := tm.market.SubmitOrder(ctx, peggedO)
					assert.NoError(t, err)
					assert.NotNil(t, peggedOConf)
					o := getMarketOrder(tm, curt, types.OrderTypeLimit, types.OrderTimeInForceGTC,
						fmt.Sprintf("order-sell-%v", i), types.SideSell, party2, 1, uint64(950+(i*10)))
					conf, err := tm.market.SubmitOrder(ctx, o)
					assert.NoError(t, err)
					assert.NotNil(t, conf)
				})

				tm.now = curt
				tm.market.OnTick(ctx, curt)
				curt = curt.Add(block)
			}
		})

		t.Run("party submit 10 buy", func(t *testing.T) {
			for i := 0; i < 10; i++ {
				t.Run("submit buy", func(t *testing.T) {
					o := getMarketOrder(tm, curt, types.OrderTypeLimit, types.OrderTimeInForceGTC,
						fmt.Sprintf("order-buy-%v", i), types.SideBuy, party2, 1, uint64(550+(i*10)))
					conf, err := tm.market.SubmitOrder(ctx, o)
					assert.NoError(t, err)
					assert.NotNil(t, conf)
				})
				tm.now = curt
				tm.market.OnTick(ctx, curt)
				curt = curt.Add(block)
			}
		})

		t.Run("party submit 20 sell", func(t *testing.T) {
			for i := 0; i < 20; i++ {
				t.Run("submit buy", func(t *testing.T) {
					o := getMarketOrder(tm, curt, types.OrderTypeLimit, types.OrderTimeInForceGTC,
						fmt.Sprintf("order-buy-%v", i), types.SideSell, party2, 1, uint64(450+(i*10)))
					conf, err := tm.market.SubmitOrder(ctx, o)
					assert.NoError(t, err)
					assert.NotNil(t, conf)
				})
				tm.now = curt
				tm.market.OnTick(ctx, curt)
				curt = curt.Add(block)
			}
		})
	})

	t.Run("check that Market Value Proxy is updated with trades", func(t *testing.T) {
		auctionEnd := now.Add(10001 * time.Second)
		mktCfg := getMarket(pMonitorSettings, &types.AuctionDuration{
			Duration: 10000,
		})
		mktCfg.Fees = &types.Fees{
			Factors: &types.FeeFactors{
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
					Sigma: num.DecimalFromFloat(2),
				},
			},
		}

		lpparty := "lp-party-1"

		tm := newTestMarket(t, now).Run(ctx, mktCfg)
		tm.StartOpeningAuction().
			WithAccountAndAmount(lpparty, 100000000000000)

		tm.market.OnMarketValueWindowLengthUpdate(2 * time.Second)
		tm.market.OnSuppliedStakeToObligationFactorUpdate(num.DecimalFromFloat(0.7))
		tm.now = now
		tm.market.OnTick(ctx, now)

		// Add a LPSubmission
		// this is a log of stake, enough to cover all
		// the required stake for the market
		lpSubmission := &types.LiquidityProvisionSubmission{
			MarketID:         tm.market.GetID(),
			CommitmentAmount: num.NewUint(10000),
			Fee:              num.DecimalFromFloat(0.5),
			Reference:        "ref-lp-submission-1",
			Buys: []*types.LiquidityOrder{
				newLiquidityOrder(types.PeggedReferenceBestBid, 201, 99),
				newLiquidityOrder(types.PeggedReferenceBestBid, 200, 1),
			},
			Sells: []*types.LiquidityOrder{
				newLiquidityOrder(types.PeggedReferenceBestAsk, 100, 1),
				newLiquidityOrder(types.PeggedReferenceBestAsk, 101, 2),
				newLiquidityOrder(types.PeggedReferenceBestAsk, 102, 98),
			},
		}

		// submit our lp
		tm.events = nil
		require.NoError(t,
			tm.market.SubmitLiquidityProvision(
				ctx, lpSubmission, lpparty, vgcrypto.RandomHash()),
		)

		t.Run("lp submission is pending", func(t *testing.T) {
			// First collect all the orders events
			var found *proto.LiquidityProvision
			for _, e := range tm.events {
				switch evt := e.(type) {
				case *events.LiquidityProvision:
					found = evt.LiquidityProvision()
				}
			}
			require.NotNil(t, found)
			// no update to the liquidity fee
			assert.Equal(t, found.Status.String(), types.LiquidityProvisionStatusPending.String())
		})

		// we end the auction
		// This will also generate trades which are included in the MVP.
		tm.EndOpeningAuction(t, auctionEnd, false)

		// at the end of the opening auction, we check the mvp
		md := tm.market.GetMarketData()
		assert.Equal(t, "10000", md.MarketValueProxy)

		// place bunches of order which will match so we increase the trade value for fee purpose.
		var (
			richParty1 = "rich-party-1"
			richParty2 = "rich-party-2"
		)

		tm.WithAccountAndAmount(richParty1, 100000000000000).
			WithAccountAndAmount(richParty2, 100000000000000)

		orders := []*types.Order{
			{
				Type:        types.OrderTypeLimit,
				Size:        1000,
				Remaining:   1000,
				Price:       num.NewUint(1111),
				Side:        types.SideBuy,
				Party:       richParty1,
				TimeInForce: types.OrderTimeInForceGTC,
			},
			{
				Type:        types.OrderTypeLimit,
				Size:        1000,
				Remaining:   1000,
				Price:       num.NewUint(1111),
				Side:        types.SideSell,
				Party:       richParty2,
				TimeInForce: types.OrderTimeInForceGTC,
			},
		}

		// now we have place our trade just after the end of the auction
		// period, and the wwindow is of 2 seconds
		tm.WithSubmittedOrders(t, orders...)

		// we increase the time for 1 second
		// the active_window_length =
		// 1 second so factor = t_market_value_window_length / active_window_length = 2.
		tm.now = auctionEnd.Add(block)
		tm.market.OnTick(ctx, auctionEnd.Add(block))
		md = tm.market.GetMarketData()
		assert.Equal(t, "2221978", md.MarketValueProxy)

		// we increase the time for another second
		tm.now = tm.now.Add(block)
		tm.market.OnTick(ctx, tm.now)
		md = tm.market.GetMarketData()
		assert.Equal(t, "1110989", md.MarketValueProxy)

		// now we increase the time for another second, which makes us slide
		// out of the window, and reset the tradeValue + window
		// so the mvp is again the total stake submitted in the market
		tm.now = tm.now.Add(block)
		tm.market.OnTick(ctx, tm.now)
		md = tm.market.GetMarketData()
		assert.Equal(t, "10000", md.MarketValueProxy)
	})

	t.Run("check that fees are not paid for undeployed LPs", func(t *testing.T) {
		auctionEnd := now.Add(10001 * time.Second)
		mktCfg := getMarket(defaultPriceMonitorSettings, &types.AuctionDuration{
			Duration: 10000,
		})
		mktCfg.Fees = &types.Fees{
			Factors: &types.FeeFactors{
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
					Sigma: num.DecimalFromFloat(2),
				},
			},
		}

		lpparty := "lp-party-1"

		tm := newTestMarket(t, now).Run(ctx, mktCfg)
		tm.StartOpeningAuction().
			WithAccountAndAmount(lpparty, 100000000000000)

		tm.market.OnMarketValueWindowLengthUpdate(2 * time.Second)
		tm.market.OnSuppliedStakeToObligationFactorUpdate(num.DecimalFromFloat(0.7))
		tm.now = now
		tm.market.OnTick(ctx, now)

		// Add a LPSubmission
		// this is a log of stake, enough to cover all
		// the required stake for the market
		lpSubmission := &types.LiquidityProvisionSubmission{
			MarketID:         tm.market.GetID(),
			CommitmentAmount: num.NewUint(10000),
			Fee:              num.DecimalFromFloat(0.5),
			Reference:        "ref-lp-submission-1",
			Buys: []*types.LiquidityOrder{
				newLiquidityOrder(types.PeggedReferenceBestBid, 201, 99),
				newLiquidityOrder(types.PeggedReferenceBestBid, 1500, 1),
			},
			Sells: []*types.LiquidityOrder{
				newLiquidityOrder(types.PeggedReferenceBestAsk, 100, 1),
				newLiquidityOrder(types.PeggedReferenceBestAsk, 101, 2),
				newLiquidityOrder(types.PeggedReferenceBestAsk, 102, 98),
			},
		}

		// submit our lp
		tm.events = nil
		require.NoError(t,
			tm.market.SubmitLiquidityProvision(
				ctx, lpSubmission, lpparty, vgcrypto.RandomHash()),
		)
		tm.now = tm.now.Add(block)
		tm.market.OnTick(ctx, tm.now)

		t.Run("lp submission is pending", func(t *testing.T) {
			// First collect all the orders events
			var found *proto.LiquidityProvision
			for _, e := range tm.events {
				switch evt := e.(type) {
				case *events.LiquidityProvision:
					found = evt.LiquidityProvision()
				}
			}
			require.NotNil(t, found)
			// no update to the liquidity fee
			assert.Equal(t, found.Status.String(), types.LiquidityProvisionStatusPending.String())
		})

		// we end the auction
		// This will also generate trades which are included in the MVP.
		tm.EndOpeningAuction(t, auctionEnd, false)

		// at the end of the opening auction, we check the mvp
		md := tm.market.GetMarketData()
		assert.Equal(t, "10000", md.MarketValueProxy)

		// place bunches of order which will match so we increase the trade value for fee purpose.
		var (
			richParty1 = "rich-party-1"
			richParty2 = "rich-party-2"
		)

		tm.WithAccountAndAmount(richParty1, 100000000000000).
			WithAccountAndAmount(richParty2, 100000000000000)

		orders := []*types.Order{
			{
				Type:        types.OrderTypeLimit,
				Size:        1000,
				Remaining:   1000,
				Price:       num.NewUint(1111),
				Side:        types.SideBuy,
				Party:       richParty1,
				TimeInForce: types.OrderTimeInForceGTC,
			},
			{
				Type:        types.OrderTypeLimit,
				Size:        1000,
				Remaining:   1000,
				Price:       num.NewUint(1111),
				Side:        types.SideSell,
				Party:       richParty2,
				TimeInForce: types.OrderTimeInForceGTC,
			},
		}

		// now we have place our trade just after the end of the auction
		// period, and the wwindow is of 2 seconds
		tm.events = nil
		tm.WithSubmittedOrders(t, orders...)

		tm.events = nil

		tm.now = auctionEnd.Add(10 * time.Second)
		tm.market.OnTick(ctx, auctionEnd.Add(10*time.Second))

		for _, e := range tm.events {
			switch evt := e.(type) {
			case *events.LedgerMovements:
				for _, v := range evt.LedgerMovements() {
					// ensure no transfer is a LIQUIDITY_FEE_DISTRIBUTE
					assert.NotEqual(t, types.TransferTypeLiquidityFeeDistribute, v.Entries[0].Type)
				}
			}
		}
	})

	t.Run("test LP provider submit limit order which expires LPO order are redeployed", func(t *testing.T) {
		auctionEnd := now.Add(10001 * time.Second)
		mktCfg := getMarket(pMonitorSettings, &types.AuctionDuration{
			Duration: 10000,
		})
		mktCfg.Fees = &types.Fees{
			Factors: &types.FeeFactors{
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
					Sigma: num.DecimalFromFloat(5),
				},
			},
		}

		lpparty := "lp-party-1"

		tm := newTestMarket(t, now).Run(ctx, mktCfg)
		tm.StartOpeningAuction().
			WithAccountAndAmount(lpparty, 100000000000000)

		tm.market.OnMarketValueWindowLengthUpdate(2 * time.Second)
		tm.market.OnSuppliedStakeToObligationFactorUpdate(num.DecimalFromFloat(0.7))
		tm.now = now
		tm.market.OnTick(ctx, now)

		lpSubmission := &types.LiquidityProvisionSubmission{
			MarketID:         tm.market.GetID(),
			CommitmentAmount: num.NewUint(10000),
			Fee:              num.DecimalFromFloat(0.5),
			Reference:        "ref-lp-submission-1",
			Buys: []*types.LiquidityOrder{
				newLiquidityOrder(types.PeggedReferenceBestBid, 10, 100),
			},
			Sells: []*types.LiquidityOrder{
				newLiquidityOrder(types.PeggedReferenceBestAsk, 10, 100),
			},
		}

		// submit our lp
		tm.events = nil
		require.NoError(t,
			tm.market.SubmitLiquidityProvision(
				ctx, lpSubmission, lpparty, vgcrypto.RandomHash()),
		)

		// we end the auction
		// This will also generate trades which are included in the MVP.
		tm.EndOpeningAuction(t, auctionEnd, false)

		t.Run("lp submission is active", func(t *testing.T) {
			// First collect all the orders events
			var found *proto.LiquidityProvision
			var ord *proto.Order
			for _, e := range tm.events {
				switch evt := e.(type) {
				case *events.LiquidityProvision:
					found = evt.LiquidityProvision()
				case *events.Order:
					ord = evt.Order()
				}
			}
			require.NotNil(t, found)
			// no update to the liquidity fee
			assert.Equal(t, found.Status.String(), types.LiquidityProvisionStatusActive.String())
			// no update to the liquidity fee
			assert.Equal(t, 15, int(ord.Size))
		})

		// then we'll submit an order which would expire
		// we submit the order at the price of the LP shape generated order
		expiringOrder := getMarketOrder(tm, auctionEnd, types.OrderTypeLimit, types.OrderTimeInForceGTT, "GTT-1", types.SideBuy, lpparty, 19, 890)
		expiringOrder.ExpiresAt = auctionEnd.Add(11 * time.Second).UnixNano()

		tm.events = nil
		_, err := tm.market.SubmitOrder(ctx, expiringOrder)
		assert.NoError(t, err)
		tm.now = tm.now.Add(block)
		tm.market.OnTick(ctx, tm.now)

		// now we ensure we have 2 order on the buy side.
		// one lp of size 6, on normal limit of size 500
		t.Run("lp order size decrease", func(t *testing.T) {
			// First collect all the orders events
			found := map[string]*proto.Order{}
			for _, e := range tm.events {
				switch evt := e.(type) {
				case *events.Order:
					found[evt.Order().Id] = evt.Order()
				}
			}

			assert.Len(t, found, 3)

			expected := []struct {
				size   uint64
				status types.LiquidityProvisionStatus
				found  bool
			}{
				{
					size:   19,
					status: types.LiquidityProvisionStatusCancelled,
					found:  false,
				},
				{
					size:   15,
					status: types.LiquidityProvisionStatusActive,
					found:  false,
				},
				{
					size:   19,
					status: types.LiquidityProvisionStatusActive,
					found:  false,
				},
			}

			// no ensure that the orders in the map matches the size we have

			matched := 0
			for _, v := range found {
				for i, exp := range expected {
					if v.Size == exp.size && v.Status.String() == exp.status.String() && !exp.found {
						expected[i].found = true
						matched++
					}
				}
			}

			assert.Equal(t, len(expected), matched, "matched quantites and statues do not match those expected")
		})

		// now the limit order expires, and the LP order size should increase again
		tm.events = nil
		tm.now = auctionEnd.Add(12 * time.Second)
		tm.market.OnTick(ctx, auctionEnd.Add(12*time.Second)) // this is 1 second after order expiry

		// now we ensure we have 2 order on the buy side.
		// one lp of size 6, on normal limit of size 500
		t.Run("lp order size increase again after expiry", func(t *testing.T) {
			// First collect all the orders events
			found := map[string]*proto.Order{}
			for _, e := range tm.events {
				switch evt := e.(type) {
				case *events.Order:
					found[evt.Order().Id] = evt.Order()
				}
			}

			assert.Len(t, found, 3)

			expected := []struct {
				size   uint64
				status types.OrderStatus
				found  bool
			}{
				{
					size:   19,
					status: types.OrderStatusActive,
					found:  false,
				},
				{
					size:   19,
					status: types.OrderStatusExpired,
					found:  false,
				},
				{
					size:   15,
					status: types.OrderStatusActive,
					found:  false,
				},
			}

			// no ensure that the orders in the map matches the size we have

			matched := 0
			for _, v := range found {
				for i, exp := range expected {
					if v.Size == exp.size && v.Status.String() == exp.status.String() && !exp.found {
						expected[i].found = true
						matched++
					}
				}
			}

			assert.Equal(t, len(expected), matched, "matched quantites and statues do not match those expected")
		})
	})
}

func TestAmend(t *testing.T) {
	now := time.Unix(10, 0)
	ctx := vegacontext.WithTraceID(context.Background(), vgcrypto.RandomHash())

	t.Run("check that fee is selected properly after changes", func(t *testing.T) {
		auctionEnd := now.Add(10001 * time.Second)
		mktCfg := getMarket(defaultPriceMonitorSettings, &types.AuctionDuration{
			Duration: 10000,
		})
		mktCfg.Fees = &types.Fees{
			Factors: &types.FeeFactors{
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
		lpparty2 := "lp-party-2"

		tm := newTestMarket(t, now).Run(ctx, mktCfg)
		tm.StartOpeningAuction().
			WithAccountAndAmount(lpparty, 500000000000).
			WithAccountAndAmount(lpparty2, 500000000000)

		tm.market.OnSuppliedStakeToObligationFactorUpdate(num.DecimalFromFloat(1.0))
		tm.now = now
		tm.market.OnTick(ctx, now)

		// Add a LPSubmission
		// this is a log of stake, enough to cover all
		// the required stake for the market
		lpSubmission := &types.LiquidityProvisionSubmission{
			MarketID:         tm.market.GetID(),
			CommitmentAmount: num.NewUint(70000),
			Fee:              num.DecimalFromFloat(0.5),
			Reference:        "ref-lp-submission-1",
			Buys: []*types.LiquidityOrder{
				newLiquidityOrder(types.PeggedReferenceBestBid, 5, 2),
				newLiquidityOrder(types.PeggedReferenceMid, 5, 2),
			},
			Sells: []*types.LiquidityOrder{
				newLiquidityOrder(types.PeggedReferenceBestAsk, 5, 13),
				newLiquidityOrder(types.PeggedReferenceBestAsk, 5, 13),
			},
		}

		// submit our lp
		tm.events = nil
		require.NoError(t,
			tm.market.SubmitLiquidityProvision(
				ctx, lpSubmission, lpparty, vgcrypto.RandomHash()),
		)

		t.Run("current liquidity fee is 0.5", func(t *testing.T) {
			// First collect all the orders events
			found := proto.Market{}
			for _, e := range tm.events {
				switch evt := e.(type) {
				case *events.MarketUpdated:
					found = evt.Market()
				}
			}

			assert.Equal(t, found.Fees.Factors.LiquidityFee, "0.5")
		})

		tm.EndOpeningAuction(t, auctionEnd, false)

		// now we submit a second LP, with a lower fee,
		// but we still need the first LP to cover liquidity
		// so its fee is selected
		lpSubmission2 := &types.LiquidityProvisionSubmission{
			MarketID:         tm.market.GetID(),
			CommitmentAmount: num.NewUint(20000),
			Fee:              num.DecimalFromFloat(0.1),
			Reference:        "ref-lp-submission-1",
			Buys: []*types.LiquidityOrder{
				newLiquidityOrder(types.PeggedReferenceBestBid, 5, 2),
				newLiquidityOrder(types.PeggedReferenceMid, 5, 2),
			},
			Sells: []*types.LiquidityOrder{
				newLiquidityOrder(types.PeggedReferenceBestAsk, 5, 13),
				newLiquidityOrder(types.PeggedReferenceBestAsk, 5, 13),
			},
		}

		// submit our lp
		tm.events = nil
		require.NoError(t,
			tm.market.SubmitLiquidityProvision(
				ctx, lpSubmission2, lpparty2, vgcrypto.RandomHash()),
		)

		t.Run("current liquidity fee is still 0.5", func(t *testing.T) {
			// First collect all the orders events
			var found *proto.Market
			for _, e := range tm.events {
				switch evt := e.(type) {
				case *events.MarketUpdated:
					mkt := evt.Market()
					found = &mkt
				}
			}

			// no update to the liquidity fee
			assert.Nil(t, found)
		})

		// now submit again the commitment, but we coverall othe target stake
		// so our fee should be selected
		tm.events = nil

		lpa := &types.LiquidityProvisionAmendment{
			Fee:              lpSubmission2.Fee,
			MarketID:         lpSubmission2.MarketID,
			CommitmentAmount: num.NewUint(60000),
			Buys:             lpSubmission2.Buys,
			Sells:            lpSubmission2.Sells,
		}

		require.NoError(t,
			tm.market.AmendLiquidityProvision(
				ctx, lpa, lpparty2, vgcrypto.RandomHash()),
		)

		t.Run("current liquidity fee is again 0.1", func(t *testing.T) {
			// First collect all the orders events
			found := proto.Market{}
			for _, e := range tm.events {
				switch evt := e.(type) {
				case *events.MarketUpdated:
					found = evt.Market()
				}
			}
			// no update to the liquidity fee
			assert.Equal(t, found.Fees.Factors.LiquidityFee, "0.1")
		})
	})

	// Liquidity fee must be updated when new LP submissions are added or existing ones
	// removed.
	t.Run("check that LP fee is correct after changes", func(t *testing.T) {
		now := time.Unix(10, 0)
		tm := getTestMarket(t, now, nil, nil)
		ctx := context.Background()

		// Create a new party account with very little funding
		addAccountWithAmount(tm, "party-A", 7000)
		tm.broker.EXPECT().Send(gomock.Any()).AnyTimes()

		tm.mas.StartOpeningAuction(now, &types.AuctionDuration{Duration: 10})
		tm.mas.AuctionStarted(ctx, now)
		tm.market.EnterAuction(ctx)

		// We shouldn't have a liquidity fee yet
		// TODO	assert.Equal(t, 0.0, tm.market.GetLiquidityFee())

		buys := []*types.LiquidityOrder{newLiquidityOrder(types.PeggedReferenceBestBid, 1, 50)}
		sells := []*types.LiquidityOrder{newLiquidityOrder(types.PeggedReferenceBestAsk, 1, 50)}

		// Submitting a correct entry
		lps := &types.LiquidityProvisionSubmission{
			Fee:              num.DecimalFromFloat(0.01),
			MarketID:         tm.market.GetID(),
			CommitmentAmount: num.NewUint(1000),
			Buys:             buys,
			Sells:            sells,
		}

		err := tm.market.SubmitLiquidityProvision(ctx, lps, "party-A", vgcrypto.RandomHash())
		require.NoError(t, err)

		// Check the fee is correct
		// TODO	assert.Equal(t, 0.01, tm.market.GetLiquidityFee())

		// Update the fee
		lpa := &types.LiquidityProvisionAmendment{
			Fee: num.DecimalFromFloat(0.5),
		}
		err = tm.market.AmendLiquidityProvision(ctx, lpa, "party-A", vgcrypto.RandomHash())
		require.NoError(t, err)

		// Check the fee is correct
		// TODO	assert.Equal(t, 0.5, tm.market.GetLiquidityFee())
	})

	t.Run("check that LP commitment reduction is prevented correctly", func(t *testing.T) {
		now := time.Unix(10, 0)
		tm := getTestMarket(t, now, nil, nil)
		ctx := context.Background()

		// Create a new party account with very little funding
		addAccountWithAmount(tm, "party-A", 10000000000)
		addAccountWithAmount(tm, "party-B", 10000000)
		addAccountWithAmount(tm, "party-C", 10000000)
		tm.broker.EXPECT().Send(gomock.Any()).AnyTimes()

		// Start the opening auction
		tm.mas.StartOpeningAuction(now, &types.AuctionDuration{Duration: 10})
		tm.mas.AuctionStarted(ctx, now)
		tm.market.EnterAuction(ctx)

		// Create some normal orders to set the reference prices
		o1 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "Order01", types.SideBuy, "party-B", 10, 10)
		o1conf, err := tm.market.SubmitOrder(ctx, o1)
		require.NotNil(t, o1conf)
		require.NoError(t, err)

		o2 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "Order02", types.SideSell, "party-C", 2, 10)
		o2conf, err := tm.market.SubmitOrder(ctx, o2)
		require.NotNil(t, o2conf)
		require.NoError(t, err)

		o3 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "Order03", types.SideSell, "party-C", 1, 20)
		o3conf, err := tm.market.SubmitOrder(ctx, o3)
		require.NotNil(t, o3conf)
		require.NoError(t, err)

		o4 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "Order04", types.SideBuy, "party-C", 1, 9)
		o4conf, err := tm.market.SubmitOrder(ctx, o4)
		require.NotNil(t, o4conf)
		require.NoError(t, err)

		// Leave auction
		tm.market.LeaveAuctionWithIDGen(ctx, now.Add(time.Second*20), newTestIDGenerator())
		// mark price is set at 10, orders on book

		buys := []*types.LiquidityOrder{
			newLiquidityOrder(types.PeggedReferenceBestBid, 10, 50),
			newLiquidityOrder(types.PeggedReferenceBestBid, 20, 50),
		}
		sells := []*types.LiquidityOrder{
			newLiquidityOrder(types.PeggedReferenceBestAsk, 10, 50),
			newLiquidityOrder(types.PeggedReferenceBestAsk, 20, 50),
		}

		// Submitting a correct entry
		lps := &types.LiquidityProvisionSubmission{
			Fee:              num.DecimalFromFloat(0.01),
			MarketID:         tm.market.GetID(),
			CommitmentAmount: num.NewUint(1000),
			Buys:             buys,
			Sells:            sells,
		}

		err = tm.market.SubmitLiquidityProvision(ctx, lps, "party-A", vgcrypto.RandomHash())
		require.NoError(t, err)
		assert.Equal(t, 1, tm.market.GetLPSCount())

		// Try to reduce our commitment to below the minimum level
		lpa := &types.LiquidityProvisionAmendment{
			Fee:              num.DecimalFromFloat(0.01),
			MarketID:         tm.market.GetID(),
			CommitmentAmount: num.NewUint(1),
			Buys:             buys,
			Sells:            sells,
		}

		err = tm.market.AmendLiquidityProvision(ctx, lpa, "party-A", vgcrypto.RandomHash())
		require.Error(t, err)
		assert.Equal(t, 1, tm.market.GetLPSCount())
	})

	t.Run("check that changing LP during auction works", func(t *testing.T) {
		tm := getTestMarket(t, now, nil, nil)

		// Create a new party account with very little funding
		addAccountWithAmount(tm, "party-A", 7000)
		addAccountWithAmount(tm, "party-B", 10000000)
		addAccountWithAmount(tm, "party-C", 10000000)
		tm.broker.EXPECT().Send(gomock.Any()).AnyTimes()
		tm.market.OnSuppliedStakeToObligationFactorUpdate(num.DecimalFromFloat(0.2))

		tm.mas.StartOpeningAuction(now, &types.AuctionDuration{Duration: 10})
		tm.mas.AuctionStarted(ctx, now)
		tm.market.EnterAuction(ctx)

		// Create some normal orders to set the reference prices
		o1 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "Order01", types.SideBuy, "party-B", 10, 10)
		o1conf, err := tm.market.SubmitOrder(ctx, o1)
		require.NotNil(t, o1conf)
		require.NoError(t, err)

		o2 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "Order02", types.SideSell, "party-C", 2, 10)
		o2conf, err := tm.market.SubmitOrder(ctx, o2)
		require.NotNil(t, o2conf)
		require.NoError(t, err)

		o3 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "Order03", types.SideSell, "party-C", 1, 20)
		o3conf, err := tm.market.SubmitOrder(ctx, o3)
		require.NotNil(t, o3conf)
		require.NoError(t, err)

		o31 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "Order031", types.SideSell, "party-C", 1, 30)
		o31conf, err := tm.market.SubmitOrder(ctx, o31)
		require.NotNil(t, o31conf)
		require.NoError(t, err)

		buys := []*types.LiquidityOrder{
			newLiquidityOrder(types.PeggedReferenceBestBid, 1, 50),
			newLiquidityOrder(types.PeggedReferenceBestBid, 2, 50),
		}
		sells := []*types.LiquidityOrder{
			newLiquidityOrder(types.PeggedReferenceBestAsk, 1, 50),
			newLiquidityOrder(types.PeggedReferenceMid, 2, 50),
		}

		// Submitting a correct entry
		lps := &types.LiquidityProvisionSubmission{
			Fee:              num.DecimalFromFloat(0.01),
			MarketID:         tm.market.GetID(),
			CommitmentAmount: num.NewUint(1000),
			Buys:             buys,
			Sells:            sells,
		}

		err = tm.market.SubmitLiquidityProvision(ctx, lps, "party-A", vgcrypto.RandomHash())
		require.NoError(t, err)
		require.Equal(t, types.LiquidityProvisionStatusPending.String(), tm.market.GetLPSState("party-A").String())
		assert.Equal(t, 0, tm.market.GetPeggedOrderCount())

		// Check we have the right amount of bond balance
		assert.Equal(t, num.NewUint(1000), tm.market.GetBondAccountBalance(ctx, "party-A", tm.market.GetID(), tm.asset))

		// Amend the commitment

		lpa := &types.LiquidityProvisionAmendment{
			Fee:              lps.Fee,
			MarketID:         lps.MarketID,
			CommitmentAmount: num.NewUint(2000),
			Buys:             lps.Buys,
			Sells:            lps.Sells,
		}
		err = tm.market.AmendLiquidityProvision(ctx, lpa, "party-A", vgcrypto.RandomHash())
		require.NoError(t, err)

		// Check we have the right amount of bond balance
		assert.Equal(t, num.NewUint(2000), tm.market.GetBondAccountBalance(ctx, "party-A", tm.market.GetID(), tm.asset))

		// Amend the commitment
		lpa.CommitmentAmount = num.NewUint(500)
		err = tm.market.AmendLiquidityProvision(ctx, lpa, "party-A", vgcrypto.RandomHash())
		require.NoError(t, err)

		// Check we have the right amount of bond balance
		assert.Equal(t, num.NewUint(500), tm.market.GetBondAccountBalance(ctx, "party-A", tm.market.GetID(), tm.asset))

		// Change the shape of the lp submission
		buys = []*types.LiquidityOrder{newLiquidityOrder(types.PeggedReferenceBestBid, 1, 50)}
		sells = []*types.LiquidityOrder{newLiquidityOrder(types.PeggedReferenceBestAsk, 1, 50)}
		lpa.Buys = buys
		lpa.Sells = sells
		err = tm.market.AmendLiquidityProvision(ctx, lpa, "party-A", vgcrypto.RandomHash())
		require.NoError(t, err)
		assert.Equal(t, 0, tm.market.GetPeggedOrderCount())
		assert.Equal(t, 0, tm.market.GetParkedOrderCount())
	})

	// Check that we are unable to directly cancel or amend a pegged order that was
	// created by the LP system.
	t.Run("check that it is not possible to cancel or amend LP order", func(t *testing.T) {
		now := time.Unix(10, 0)
		tm := getTestMarket(t, now, nil, nil)
		ctx := context.Background()

		// Create a new party account with very little funding
		addAccountWithAmount(tm, "party-A", 10000000)
		addAccountWithAmount(tm, "party-B", 10000000)
		addAccountWithAmount(tm, "party-C", 10000000)
		tm.broker.EXPECT().Send(gomock.Any()).AnyTimes()

		tm.mas.StartOpeningAuction(now, &types.AuctionDuration{Duration: 10})
		tm.mas.AuctionStarted(ctx, now)
		tm.market.EnterAuction(ctx)

		// Create some normal orders to set the reference prices
		o1 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "Order01", types.SideBuy, "party-B", 10, 10)
		o1conf, err := tm.market.SubmitOrder(ctx, o1)
		require.NotNil(t, o1conf)
		require.NoError(t, err)

		o2 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "Order02", types.SideSell, "party-C", 2, 10)
		o2conf, err := tm.market.SubmitOrder(ctx, o2)
		require.NotNil(t, o2conf)
		require.NoError(t, err)

		o3 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "Order03", types.SideSell, "party-C", 1, 20)
		o3conf, err := tm.market.SubmitOrder(ctx, o3)
		require.NotNil(t, o3conf)
		require.NoError(t, err)

		buys := []*types.LiquidityOrder{
			newLiquidityOrder(types.PeggedReferenceBestBid, 1, 50),
			newLiquidityOrder(types.PeggedReferenceBestBid, 2, 50),
		}
		sells := []*types.LiquidityOrder{
			newLiquidityOrder(types.PeggedReferenceBestAsk, 1, 50),
			newLiquidityOrder(types.PeggedReferenceBestAsk, 2, 50),
		}

		// Submitting a correct entry
		lps := &types.LiquidityProvisionSubmission{
			Fee:              num.DecimalFromFloat(0.01),
			MarketID:         tm.market.GetID(),
			CommitmentAmount: num.NewUint(1000),
			Buys:             buys,
			Sells:            sells,
		}

		err = tm.market.SubmitLiquidityProvision(ctx, lps, "party-A", vgcrypto.RandomHash())
		require.NoError(t, err)

		// Leave auction
		tm.market.LeaveAuctionWithIDGen(ctx, now.Add(time.Second*20), newTestIDGenerator())

		// Check we have an accepted LP submission
		assert.Equal(t, 1, tm.market.GetLPSCount())

		// Check we have the right number of live orders
		assert.Equal(t, int64(6), tm.market.GetOrdersOnBookCount())

		// FIXME(): REDO THIS TEST
		// Attempt to cancel one of the pegged orders and it is rejected
		// orders := tm.market.GetPeggedOrders("party-A")
		// assert.GreaterOrEqual(t, len(orders), 0)

		// cancelConf, err := tm.market.CancelOrder(ctx, "party-A", orders[0].Id)
		// require.Nil(t, cancelConf)
		// require.Error(t, err)
		// assert.Equal(t, types.OrderError_ORDER_ERROR_EDIT_NOT_ALLOWED, err)

		// // Attempt to amend one of the pegged orders
		// amend := &commandspb.OrderAmendment{OrderId: orders[0].Id,
		// 	MarketId:  orders[0].MarketId,
		// 	SizeDelta: +5}
		// amendConf, err := tm.market.AmendOrder(ctx, amend, orders[0].PartyId)
		// require.Error(t, err)
		// require.Nil(t, amendConf)
		// assert.Equal(t, types.OrderError_ORDER_ERROR_EDIT_NOT_ALLOWED, err)
	})

	// If we submit a valid LP submission but then try ot alter it to something non valid
	// the amendment should be rejected and the original submission is still valid.
	t.Run("check that failed amend does not break existing LP", func(t *testing.T) {
		tm := getTestMarket(t, now, nil, nil)

		// Create a new party account with very little funding
		addAccountWithAmount(tm, "party-A", 7000)
		addAccountWithAmount(tm, "party-B", 10000000)
		addAccountWithAmount(tm, "party-C", 10000000)
		tm.broker.EXPECT().Send(gomock.Any()).AnyTimes()

		tm.mas.StartOpeningAuction(now, &types.AuctionDuration{Duration: 10})
		tm.mas.AuctionStarted(ctx, now)
		tm.market.EnterAuction(ctx)

		buys := []*types.LiquidityOrder{
			newLiquidityOrder(types.PeggedReferenceBestBid, 1, 50),
			newLiquidityOrder(types.PeggedReferenceMid, 6, 50),
		}
		sells := []*types.LiquidityOrder{
			newLiquidityOrder(types.PeggedReferenceBestAsk, 1, 50),
			newLiquidityOrder(types.PeggedReferenceMid, 6, 50),
		}

		// Submitting a correct entry
		lps := &types.LiquidityProvisionSubmission{
			Fee:              num.DecimalFromFloat(0.01),
			MarketID:         tm.market.GetID(),
			CommitmentAmount: num.NewUint(1000),
			Buys:             buys,
			Sells:            sells,
		}

		err := tm.market.SubmitLiquidityProvision(ctx, lps, "party-A", vgcrypto.RandomHash())
		require.NoError(t, err)
		require.Equal(t, types.LiquidityProvisionStatusPending.String(), tm.market.GetLPSState("party-A").String())
		assert.Equal(t, 0, tm.market.GetPeggedOrderCount())
		assert.Equal(t, num.NewUint(1000), tm.market.GetBondAccountBalance(ctx, "party-A", tm.market.GetID(), tm.asset))

		// Now attempt to amend the LP submission with empty fee
		lpa := &types.LiquidityProvisionAmendment{
			MarketID:         lps.MarketID,
			CommitmentAmount: lps.CommitmentAmount,
			Buys:             lps.Buys,
			Sells:            lps.Sells,
		}

		err = tm.market.AmendLiquidityProvision(ctx, lpa, "party-A", vgcrypto.RandomHash())
		require.NoError(t, err)

		// Now attempt to amend the LP submission with empty fee and commitment amount
		lpa = &types.LiquidityProvisionAmendment{
			MarketID: lps.MarketID,
			Buys:     lps.Buys,
			Sells:    lps.Sells,
		}

		err = tm.market.AmendLiquidityProvision(ctx, lpa, "party-A", vgcrypto.RandomHash())
		require.NoError(t, err)

		// Now attempt to amend the LP submission with empty buys
		lpa = &types.LiquidityProvisionAmendment{
			Fee:              lps.Fee,
			MarketID:         lps.MarketID,
			CommitmentAmount: lps.CommitmentAmount,
			Buys:             nil,
			Sells:            lps.Sells,
		}

		err = tm.market.AmendLiquidityProvision(ctx, lpa, "party-A", vgcrypto.RandomHash())
		require.NoError(t, err)

		// Now attempt to amend the LP submission with no changes with nil buys and nil sells
		lpa = &types.LiquidityProvisionAmendment{
			Fee:              num.DecimalZero(),
			MarketID:         lps.MarketID,
			CommitmentAmount: num.UintZero(),
			Buys:             nil,
			Sells:            nil,
		}

		err = tm.market.AmendLiquidityProvision(ctx, lpa, "party-A", vgcrypto.RandomHash())
		require.EqualError(t, err, "empty liquidity provision amendment content")

		// Now attempt to amend the LP submission with no changes with sells and buys empty lists
		lpa = &types.LiquidityProvisionAmendment{
			Fee:              num.DecimalZero(),
			MarketID:         lps.MarketID,
			CommitmentAmount: num.UintZero(),
			Buys:             []*types.LiquidityOrder{},
			Sells:            []*types.LiquidityOrder{},
		}

		err = tm.market.AmendLiquidityProvision(ctx, lpa, "party-A", vgcrypto.RandomHash())
		require.EqualError(t, err, "empty liquidity provision amendment content")

		// Check that the original LP submission is still working fine
		require.Equal(t, types.LiquidityProvisionStatusPending.String(), tm.market.GetLPSState("party-A").String())
	})

	// Reference must be updated when LP submissions are amended.
	t.Run("check reference is correct after changes", func(t *testing.T) {
		tm := getTestMarket(t, now, nil, nil)

		// Create a new party account with very little funding
		addAccountWithAmount(tm, "party-A", 7000)
		tm.broker.EXPECT().Send(gomock.Any()).AnyTimes()

		tm.mas.StartOpeningAuction(now, &types.AuctionDuration{Duration: 10})
		tm.mas.AuctionStarted(ctx, now)
		tm.market.EnterAuction(ctx)

		buys := []*types.LiquidityOrder{newLiquidityOrder(types.PeggedReferenceBestBid, 1, 50)}
		sells := []*types.LiquidityOrder{newLiquidityOrder(types.PeggedReferenceBestAsk, 1, 50)}

		// Submitting a correct entry
		lps := &types.LiquidityProvisionSubmission{
			Fee:              num.DecimalFromFloat(0.01),
			MarketID:         tm.market.GetID(),
			CommitmentAmount: num.NewUint(1000),
			Reference:        "ref-lp-1",
			Buys:             buys,
			Sells:            sells,
		}

		err := tm.market.SubmitLiquidityProvision(ctx, lps, "party-A", vgcrypto.RandomHash())
		require.NoError(t, err)

		// Update the fee
		lpa := &types.LiquidityProvisionAmendment{
			Fee: num.DecimalFromFloat(0.2),
		}
		err = tm.market.AmendLiquidityProvision(ctx, lpa, "party-A", vgcrypto.RandomHash())
		require.NoError(t, err)

		// Update the fee again with a new reference
		lpa = &types.LiquidityProvisionAmendment{
			Fee:       num.DecimalFromFloat(0.5),
			Reference: "ref-lp-2",
		}
		err = tm.market.AmendLiquidityProvision(ctx, lpa, "party-A", vgcrypto.RandomHash())
		require.NoError(t, err)

		t.Run("expect LP references", func(t *testing.T) {
			// First collect all the lp events
			found := map[string]*proto.LiquidityProvision{}
			for _, e := range tm.events {
				switch evt := e.(type) {
				case *events.LiquidityProvision:
					lp := evt.LiquidityProvision()
					found[lp.Fee] = lp
				}
			}
			expectedStatus := map[string]string{"0.01": "ref-lp-1", "0.2": "ref-lp-1", "0.5": "ref-lp-2"}
			require.Len(t, found, len(expectedStatus))

			for k, v := range expectedStatus {
				assert.Equal(t, v, found[k].Reference)
			}
		})
	})

	t.Run("should reject LP amendment if no current LP", func(t *testing.T) {
		now := time.Unix(10, 0)
		tm := getTestMarket(t, now, nil, nil)
		ctx := context.Background()

		// Create a new party account with very little funding
		addAccountWithAmount(tm, "party-A", 7000)
		tm.broker.EXPECT().Send(gomock.Any()).AnyTimes()

		tm.mas.StartOpeningAuction(now, &types.AuctionDuration{Duration: 10})
		tm.mas.AuctionStarted(ctx, now)
		tm.market.EnterAuction(ctx)

		// Try to update the fee
		lpa := &types.LiquidityProvisionAmendment{
			Fee: num.DecimalFromFloat(0.5),
		}
		err := tm.market.AmendLiquidityProvision(ctx, lpa, "party-A", vgcrypto.RandomHash())
		require.EqualError(t, err, "party is not a liquidity provider")
	})

	t.Run("should reject LP cancellation if no current LP", func(t *testing.T) {
		tm := getTestMarket(t, now, nil, nil)

		// Create a new party account with very little funding
		addAccountWithAmount(tm, "party-A", 7000)
		tm.broker.EXPECT().Send(gomock.Any()).AnyTimes()

		tm.mas.StartOpeningAuction(now, &types.AuctionDuration{Duration: 10})
		tm.mas.AuctionStarted(ctx, now)
		tm.market.EnterAuction(ctx)

		lpc := &types.LiquidityProvisionCancellation{
			MarketID: tm.market.GetID(),
		}
		err := tm.market.CancelLiquidityProvision(ctx, lpc, "party-A")
		require.EqualError(t, err, "party is not a liquidity provider")
	})
}

func newTestIDGenerator() execution.IDGenerator {
	return idgeneration.New(vgcrypto.RandomHash())
}
