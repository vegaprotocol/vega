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

package future_test

import (
	"context"
	"testing"
	"time"

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

		// Submitting a zero or smaller fee should cause a reject
		lps := &types.LiquidityProvisionSubmission{
			Fee:              num.DecimalFromFloat(-0.50),
			MarketID:         tm.market.GetID(),
			CommitmentAmount: num.NewUint(1000),
		}

		err := tm.market.SubmitLiquidityProvision(ctx, lps, "party-A", vgcrypto.RandomHash())
		require.Error(t, err)
		assert.Equal(t, 0, tm.market.GetLPSCount())

		// Submitting a fee greater than 1.0 should cause a reject
		lps = &types.LiquidityProvisionSubmission{
			Fee:              num.DecimalFromFloat(1.01),
			MarketID:         tm.market.GetID(),
			CommitmentAmount: num.NewUint(1000),
		}

		err = tm.market.SubmitLiquidityProvision(ctx, lps, "party-A", vgcrypto.RandomHash())
		require.Error(t, err)
		assert.Equal(t, 0, tm.market.GetLPSCount())
	})

	t.Run("test liquidity provision fee validation", func(t *testing.T) {
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

		tm.market.OnTick(ctx, tm.now)

		// Add a LPSubmission
		// this is a log of stake, enough to cover all
		// the required stake for the market
		lpSubmission := &types.LiquidityProvisionSubmission{
			MarketID:         tm.market.GetID(),
			CommitmentAmount: num.NewUint(70000),
			Fee:              num.DecimalFromFloat(-0.1),
			Reference:        "ref-lp-submission-1",
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

		lpParty := "lp-party-1"
		addAccountWithAmount(tm, lpParty, 5000000)

		// ensure LP is set
		lps := &types.LiquidityProvisionSubmission{
			Fee:              num.DecimalFromFloat(0.01),
			MarketID:         tm.market.GetID(),
			CommitmentAmount: num.NewUint(5000000),
		}
		require.NoError(t, tm.market.SubmitLiquidityProvision(
			context.Background(), lps, lpParty, vgcrypto.RandomHash(),
		))

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

		// Submitting a correct entry
		lps2 := &types.LiquidityProvisionSubmission{
			Fee:              num.DecimalFromFloat(0.01),
			MarketID:         tm.market.GetID(),
			CommitmentAmount: num.NewUint(1000),
		}

		err = tm.market.SubmitLiquidityProvision(ctx, lps2, "party-A", vgcrypto.RandomHash())

		tm.market.OnEpochEvent(ctx, types.Epoch{Action: proto.EpochAction_EPOCH_ACTION_START})
		require.NoError(t, err)
		require.Equal(t, types.LiquidityProvisionStatusActive.String(), tm.market.GetLPSState("party-A").String())
		// Only 3 pegged orders as one fails due to price monitoring
		assert.Equal(t, 0, tm.market.GetPeggedOrderCount())
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
		}

		// submit our lp
		tm.events = nil
		require.NoError(t,
			tm.market.SubmitLiquidityProvision(
				ctx, lpSubmission, lpparty, vgcrypto.RandomHash()),
		)

		tm.market.OnEpochEvent(ctx, types.Epoch{Action: proto.EpochAction_EPOCH_ACTION_START})

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
		}

		// submit our lp
		tm.events = nil
		require.NoError(t,
			tm.market.SubmitLiquidityProvision(
				ctx, lpSubmission2, lpparty2, vgcrypto.RandomHash()),
		)

		tm.market.OnEpochEvent(ctx, types.Epoch{Action: proto.EpochAction_EPOCH_ACTION_START})

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
	})

	// Liquidity fee must be updated when new LP submissions are added or existing ones
	// removed.
	t.Run("check that LP fee is correct after changes", func(t *testing.T) {
		t.Skip()
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

		// Submitting a correct entry
		lps := &types.LiquidityProvisionSubmission{
			Fee:              num.DecimalFromFloat(0.01),
			MarketID:         tm.market.GetID(),
			CommitmentAmount: num.NewUint(1000),
		}

		err := tm.market.SubmitLiquidityProvision(ctx, lps, "party-A", vgcrypto.RandomHash())
		require.NoError(t, err)

		tm.market.OnEpochEvent(ctx, types.Epoch{Action: proto.EpochAction_EPOCH_ACTION_START})

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
		t.Skip()
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

		// Submitting a correct entry
		lps := &types.LiquidityProvisionSubmission{
			Fee:              num.DecimalFromFloat(0.01),
			MarketID:         tm.market.GetID(),
			CommitmentAmount: num.NewUint(1000),
		}

		err = tm.market.SubmitLiquidityProvision(ctx, lps, "party-A", vgcrypto.RandomHash())
		require.NoError(t, err)
		assert.Equal(t, 1, tm.market.GetLPSCount())

		// Try to reduce our commitment to below the minimum level
		lpa := &types.LiquidityProvisionAmendment{
			Fee:              num.DecimalFromFloat(0.01),
			MarketID:         tm.market.GetID(),
			CommitmentAmount: num.NewUint(1),
		}

		err = tm.market.AmendLiquidityProvision(ctx, lpa, "party-A", vgcrypto.RandomHash())
		require.Error(t, err)
		assert.Equal(t, 1, tm.market.GetLPSCount())
	})

	t.Run("check that changing LP during auction works", func(t *testing.T) {
		t.Skip()
		tm := getTestMarket(t, now, nil, nil)

		// Create a new party account with very little funding
		addAccountWithAmount(tm, "party-A", 7000)
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

		o31 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "Order031", types.SideSell, "party-C", 1, 30)
		o31conf, err := tm.market.SubmitOrder(ctx, o31)
		require.NotNil(t, o31conf)
		require.NoError(t, err)

		// Submitting a correct entry
		lps := &types.LiquidityProvisionSubmission{
			Fee:              num.DecimalFromFloat(0.01),
			MarketID:         tm.market.GetID(),
			CommitmentAmount: num.NewUint(1000),
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
		}
		err = tm.market.AmendLiquidityProvision(ctx, lpa, "party-A", vgcrypto.RandomHash())
		require.NoError(t, err)

		tm.market.OnEpochEvent(ctx, types.Epoch{Action: proto.EpochAction_EPOCH_ACTION_START})

		// Check we have the right amount of bond balance
		assert.Equal(t, num.NewUint(2000), tm.market.GetBondAccountBalance(ctx, "party-A", tm.market.GetID(), tm.asset))

		// Amend the commitment
		lpa.CommitmentAmount = num.NewUint(500)
		err = tm.market.AmendLiquidityProvision(ctx, lpa, "party-A", vgcrypto.RandomHash())
		require.NoError(t, err)

		// Check we have the right amount of bond balance
		assert.Equal(t, num.NewUint(500), tm.market.GetBondAccountBalance(ctx, "party-A", tm.market.GetID(), tm.asset))

		err = tm.market.AmendLiquidityProvision(ctx, lpa, "party-A", vgcrypto.RandomHash())
		require.NoError(t, err)
		assert.Equal(t, 0, tm.market.GetPeggedOrderCount())
		assert.Equal(t, 0, tm.market.GetParkedOrderCount())
	})

	// Check that we are unable to directly cancel or amend a pegged order that was
	// created by the LP system.
	t.Run("check that it is not possible to cancel or amend LP order", func(t *testing.T) {
		t.Skip()
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

		// Submitting a correct entry
		lps := &types.LiquidityProvisionSubmission{
			Fee:              num.DecimalFromFloat(0.01),
			MarketID:         tm.market.GetID(),
			CommitmentAmount: num.NewUint(1000),
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
		t.Skip()
		tm := getTestMarket(t, now, nil, nil)

		// Create a new party account with very little funding
		addAccountWithAmount(tm, "party-A", 7000)
		addAccountWithAmount(tm, "party-B", 10000000)
		addAccountWithAmount(tm, "party-C", 10000000)
		tm.broker.EXPECT().Send(gomock.Any()).AnyTimes()

		tm.mas.StartOpeningAuction(now, &types.AuctionDuration{Duration: 10})
		tm.mas.AuctionStarted(ctx, now)
		tm.market.EnterAuction(ctx)

		// Submitting a correct entry
		lps := &types.LiquidityProvisionSubmission{
			Fee:              num.DecimalFromFloat(0.01),
			MarketID:         tm.market.GetID(),
			CommitmentAmount: num.NewUint(1000),
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
		}

		err = tm.market.AmendLiquidityProvision(ctx, lpa, "party-A", vgcrypto.RandomHash())
		require.NoError(t, err)

		// Now attempt to amend the LP submission with empty fee and commitment amount
		lpa = &types.LiquidityProvisionAmendment{
			MarketID: lps.MarketID,
		}

		err = tm.market.AmendLiquidityProvision(ctx, lpa, "party-A", vgcrypto.RandomHash())
		require.NoError(t, err)

		// Now attempt to amend the LP submission with empty buys
		lpa = &types.LiquidityProvisionAmendment{
			Fee:              lps.Fee,
			MarketID:         lps.MarketID,
			CommitmentAmount: lps.CommitmentAmount,
		}

		err = tm.market.AmendLiquidityProvision(ctx, lpa, "party-A", vgcrypto.RandomHash())
		require.NoError(t, err)

		// Now attempt to amend the LP submission with no changes with nil buys and nil sells
		lpa = &types.LiquidityProvisionAmendment{
			Fee:              num.DecimalZero(),
			MarketID:         lps.MarketID,
			CommitmentAmount: num.UintZero(),
		}

		err = tm.market.AmendLiquidityProvision(ctx, lpa, "party-A", vgcrypto.RandomHash())
		require.EqualError(t, err, "empty liquidity provision amendment content")

		// Now attempt to amend the LP submission with no changes with sells and buys empty lists
		lpa = &types.LiquidityProvisionAmendment{
			Fee:              num.DecimalZero(),
			MarketID:         lps.MarketID,
			CommitmentAmount: num.UintZero(),
		}

		err = tm.market.AmendLiquidityProvision(ctx, lpa, "party-A", vgcrypto.RandomHash())
		require.EqualError(t, err, "empty liquidity provision amendment content")

		// Check that the original LP submission is still working fine
		require.Equal(t, types.LiquidityProvisionStatusPending.String(), tm.market.GetLPSState("party-A").String())
	})

	// Reference must be updated when LP submissions are amended.
	t.Run("check reference is correct after changes", func(t *testing.T) {
		t.Skip()
		tm := getTestMarket(t, now, nil, nil)

		// Create a new party account with very little funding
		addAccountWithAmount(tm, "party-A", 7000)
		tm.broker.EXPECT().Send(gomock.Any()).AnyTimes()

		tm.mas.StartOpeningAuction(now, &types.AuctionDuration{Duration: 10})
		tm.mas.AuctionStarted(ctx, now)
		tm.market.EnterAuction(ctx)

		// Submitting a correct entry
		lps := &types.LiquidityProvisionSubmission{
			Fee:              num.DecimalFromFloat(0.01),
			MarketID:         tm.market.GetID(),
			CommitmentAmount: num.NewUint(1000),
			Reference:        "ref-lp-1",
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
		t.Skip()
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
		t.Skip()
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
