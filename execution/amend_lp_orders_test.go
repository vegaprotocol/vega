package execution_test

import (
	"context"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/proto"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAmendDeployedCommitment(t *testing.T) {
	now := time.Unix(10, 0)
	closingAt := time.Unix(1000000000, 0)
	ctx := context.Background()

	auctionEnd := now.Add(10001 * time.Second)
	mktCfg := getMarket(closingAt, defaultPriceMonitorSettings, &types.AuctionDuration{
		Duration: 10000,
	})
	mktCfg.Fees = &types.Fees{
		Factors: &types.FeeFactors{
			LiquidityFee:      num.DecimalFromFloat(0.001),
			InfrastructureFee: num.DecimalFromFloat(0.0005),
			MakerFee:          num.DecimalFromFloat(0.00025),
		},
	}
	mktCfg.TradableInstrument.RiskModel = &types.TradableInstrument_LogNormalRiskModel{
		LogNormalRiskModel: &types.LogNormalRiskModel{
			RiskAversionParameter: num.DecimalFromFloat(0.001),
			Tau:                   num.DecimalFromFloat(0.00011407711613050422),
			Params: &types.LogNormalModelParams{
				Mu:    num.DecimalFromFloat(0),
				R:     num.DecimalFromFloat(0.016),
				Sigma: num.DecimalFromFloat(20),
			},
		},
	}

	lpparty := "lp-party-1"

	tm := newTestMarket(t, now).Run(ctx, mktCfg)
	tm.StartOpeningAuction().
		// the liquidity provider
		WithAccountAndAmount(lpparty, 500000000000)

	tm.market.OnSuppliedStakeToObligationFactorUpdate(1.0)
	tm.market.OnChainTimeUpdate(ctx, now)

	// Add a LPSubmission
	// this is a log of stake, enough to cover all
	// the required stake for the market
	lpSubmission := &types.LiquidityProvisionSubmission{
		MarketId:         tm.market.GetID(),
		CommitmentAmount: num.NewUint(70000),
		Fee:              num.DecimalFromFloat(0.01),
		Reference:        "ref-lp-submission-1",
		Buys: []*types.LiquidityOrder{
			{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_BID, Proportion: 2, Offset: -5},
			{Reference: types.PeggedReference_PEGGED_REFERENCE_MID, Proportion: 2, Offset: -5},
		},
		Sells: []*types.LiquidityOrder{
			{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_ASK, Proportion: 13, Offset: 5},
			{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_ASK, Proportion: 13, Offset: 5},
		},
	}

	// submit our lp
	require.NoError(t,
		tm.market.SubmitLiquidityProvision(
			ctx, lpSubmission, lpparty, "liquidity-submission-1"),
	)

	t.Run("bond account is updated with the new commitment", func(t *testing.T) {
		acc, err := tm.collateralEngine.GetPartyBondAccount(tm.market.GetID(), lpparty, tm.asset)
		assert.NoError(t, err)
		assert.Equal(t, num.NewUint(70000), acc.Balance)
	})

	tm.EndOpeningAuction(t, auctionEnd, false)

	// now we will reduce our commitment
	// we will still be higher than the required stake
	lpSmallerCommitment := &types.LiquidityProvisionSubmission{
		MarketId:         tm.market.GetID(),
		CommitmentAmount: num.NewUint(60000),
		Fee:              num.DecimalFromFloat(0.01),
		Reference:        "ref-lp-submission-2",
		Buys: []*types.LiquidityOrder{
			{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_BID, Proportion: 2, Offset: -5},
			{Reference: types.PeggedReference_PEGGED_REFERENCE_MID, Proportion: 2, Offset: -5},
		},
		Sells: []*types.LiquidityOrder{
			{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_ASK, Proportion: 13, Offset: 5},
			{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_ASK, Proportion: 13, Offset: 5},
		},
	}

	tm.events = nil
	// submit our lp
	require.NoError(t,
		tm.market.SubmitLiquidityProvision(
			ctx, lpSmallerCommitment, lpparty, "liquidity-submission-2"),
	)

	t.Run("bond account is updated with the new commitment", func(t *testing.T) {
		acc, err := tm.collateralEngine.GetPartyBondAccount(tm.market.GetID(), lpparty, tm.asset)
		assert.NoError(t, err)
		assert.Equal(t, num.NewUint(60000), acc.Balance)
	})

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

		expectedStatus := map[string]types.LiquidityProvision_Status{
			"liquidity-submission-1": types.LiquidityProvision_STATUS_ACTIVE,
		}

		require.Len(t, found, len(expectedStatus))

		for k, v := range expectedStatus {
			assert.Equal(t, v.String(), found[k].Status.String())
		}
	})

	t.Run("previous LP orders to be cancelled", func(t *testing.T) {
		// First collect all the orders events
		found := []*types.Order{}
		for _, e := range tm.events {
			switch evt := e.(type) {
			case *events.Order:
				found = append(found, types.OrderFromProto(evt.Order()))
			}
		}

		require.Len(t, found, 8)

		// reference -> status
		expectedStatus := map[string]types.Order_Status{
			"ref-lp-submission-1": types.Order_STATUS_CANCELLED,
			"ref-lp-submission-2": types.Order_STATUS_ACTIVE,
		}

		totalCancelled := 0
		totalActive := 0

		for _, o := range found {
			assert.Equal(t,
				expectedStatus[o.Reference].String(),
				o.Status.String(),
			)
			if o.Status == types.Order_STATUS_CANCELLED {
				totalCancelled += 1
			}
			if o.Status == types.Order_STATUS_ACTIVE {
				totalActive += 1
			}
		}

		assert.Equal(t, totalCancelled, 4)
		assert.Equal(t, totalActive, 4)

	})

	// now we will reduce our commitment
	// we will still be higher than the required stake
	lpHigherCommitment := &types.LiquidityProvisionSubmission{
		MarketId:         tm.market.GetID(),
		CommitmentAmount: num.NewUint(80000),
		Fee:              num.DecimalFromFloat(0.01),
		Reference:        "ref-lp-submission-3",
		Buys: []*types.LiquidityOrder{
			{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_BID, Proportion: 2, Offset: -5},
			{Reference: types.PeggedReference_PEGGED_REFERENCE_MID, Proportion: 2, Offset: -5},
		},
		Sells: []*types.LiquidityOrder{
			{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_ASK, Proportion: 13, Offset: 5},
			{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_ASK, Proportion: 13, Offset: 5},
		},
	}

	tm.events = nil
	// submit our lp
	require.NoError(t,
		tm.market.SubmitLiquidityProvision(
			ctx, lpHigherCommitment, lpparty, "liquidity-submission-3"),
	)

	t.Run("bond account is updated with the new commitment", func(t *testing.T) {
		acc, err := tm.collateralEngine.GetPartyBondAccount(tm.market.GetID(), lpparty, tm.asset)
		assert.NoError(t, err)
		assert.Equal(t, num.NewUint(80000), acc.Balance)
	})

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

		expectedStatus := map[string]types.LiquidityProvision_Status{
			"liquidity-submission-1": types.LiquidityProvision_STATUS_ACTIVE,
		}

		require.Len(t, found, len(expectedStatus))

		for k, v := range expectedStatus {
			assert.Equal(t, v.String(), found[k].Status.String())
		}
	})

	t.Run("previous LP orders to be cancelled", func(t *testing.T) {
		// First collect all the orders events
		found := []*types.Order{}
		for _, e := range tm.events {
			switch evt := e.(type) {
			case *events.Order:
				found = append(found, types.OrderFromProto(evt.Order()))
			}
		}

		require.Len(t, found, 8)

		// reference -> status
		expectedStatus := map[string]types.Order_Status{
			"ref-lp-submission-2": types.Order_STATUS_CANCELLED,
			"ref-lp-submission-3": types.Order_STATUS_ACTIVE,
		}

		totalCancelled := 0
		totalActive := 0

		for _, o := range found {
			assert.Equal(t,
				expectedStatus[o.Reference].String(),
				o.Status.String(),
			)
			if o.Status == types.Order_STATUS_CANCELLED {
				totalCancelled += 1
			}
			if o.Status == types.Order_STATUS_ACTIVE {
				totalActive += 1
			}
		}

		assert.Equal(t, totalCancelled, 4)
		assert.Equal(t, totalActive, 4)

	})

	// now we will reduce our commitment
	// we will still be higher than the required stake
	lpDifferentShapeCommitment := &types.LiquidityProvisionSubmission{
		MarketId:         tm.market.GetID(),
		CommitmentAmount: num.NewUint(80000),
		Fee:              num.DecimalFromFloat(0.01),
		Reference:        "ref-lp-submission-3-bis",
		Buys: []*types.LiquidityOrder{
			{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_BID, Proportion: 2, Offset: -5},
			{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_BID, Proportion: 2, Offset: -4},
			{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_BID, Proportion: 2, Offset: -3},
			{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_BID, Proportion: 2, Offset: -2},
			{Reference: types.PeggedReference_PEGGED_REFERENCE_MID, Proportion: 2, Offset: -5},
		},
		Sells: []*types.LiquidityOrder{
			{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_ASK, Proportion: 13, Offset: 5},
		},
	}

	tm.events = nil
	// submit our lp
	require.NoError(t,
		tm.market.SubmitLiquidityProvision(
			ctx, lpDifferentShapeCommitment, lpparty, "liquidity-submission-3-bis"),
	)

	t.Run("bond account is updated with the new commitment", func(t *testing.T) {
		acc, err := tm.collateralEngine.GetPartyBondAccount(tm.market.GetID(), lpparty, tm.asset)
		assert.NoError(t, err)
		assert.Equal(t, num.NewUint(80000), acc.Balance)
	})

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

		expectedStatus := map[string]types.LiquidityProvision_Status{
			"liquidity-submission-1": types.LiquidityProvision_STATUS_ACTIVE,
		}

		require.Len(t, found, len(expectedStatus))

		for k, v := range expectedStatus {
			assert.Equal(t, v.String(), found[k].Status.String())
		}
	})

	t.Run("previous LP orders to be cancelled", func(t *testing.T) {
		// First collect all the orders events
		found := []*types.Order{}
		for _, e := range tm.events {
			switch evt := e.(type) {
			case *events.Order:
				found = append(found, types.OrderFromProto(evt.Order()))
			}
		}

		require.Len(t, found, 10)

		// reference -> status
		expectedStatus := map[string]types.Order_Status{
			"ref-lp-submission-3":     types.Order_STATUS_CANCELLED,
			"ref-lp-submission-3-bis": types.Order_STATUS_ACTIVE,
		}

		totalCancelled := 0
		totalActive := 0

		for _, o := range found {
			assert.Equal(t,
				expectedStatus[o.Reference].String(),
				o.Status.String(),
			)
			if o.Status == types.Order_STATUS_CANCELLED {
				totalCancelled += 1
			}
			if o.Status == types.Order_STATUS_ACTIVE {
				totalActive += 1
			}
		}

		assert.Equal(t, totalCancelled, 4)
		assert.Equal(t, totalActive, 6)

	})

	// now we will reduce the commitment too much so it gets under
	// the expected stake.
	// this should result into an error, and the commitment staying
	// untouched
	lpTooSmallCommitment := &types.LiquidityProvisionSubmission{
		MarketId:         tm.market.GetID(),
		CommitmentAmount: num.NewUint(30000), // required commitment is 50000
		Fee:              num.DecimalFromFloat(0.01),
		Reference:        "ref-lp-submission-4",
		Buys: []*types.LiquidityOrder{
			{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_BID, Proportion: 2, Offset: -5},
			{Reference: types.PeggedReference_PEGGED_REFERENCE_MID, Proportion: 2, Offset: -5},
		},
		Sells: []*types.LiquidityOrder{
			{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_ASK, Proportion: 13, Offset: 5},
			{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_ASK, Proportion: 13, Offset: 5},
		},
	}

	tm.events = nil
	// submit our lp
	require.EqualError(t,
		tm.market.SubmitLiquidityProvision(
			ctx, lpTooSmallCommitment, lpparty, "liquidity-submission-4"),
		"commitment submission rejected, not enough stake",
	)

	// now we will increase the commitment too much so it gets
	// at a point where we cannot fill the bond requirement
	// this should result into an error, and the commitment staying
	// untouched
	lpTooHighCommitment := &types.LiquidityProvisionSubmission{
		MarketId:         tm.market.GetID(),
		CommitmentAmount: num.NewUint(600000000000), // required commitment is 50000
		Fee:              num.DecimalFromFloat(0.01),
		Reference:        "ref-lp-submission-5",
		Buys: []*types.LiquidityOrder{
			{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_BID, Proportion: 2, Offset: -5},
			{Reference: types.PeggedReference_PEGGED_REFERENCE_MID, Proportion: 2, Offset: -5},
		},
		Sells: []*types.LiquidityOrder{
			{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_ASK, Proportion: 13, Offset: 5},
			{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_ASK, Proportion: 13, Offset: 5},
		},
	}

	tm.events = nil
	// submit our lp
	require.EqualError(t,
		tm.market.SubmitLiquidityProvision(
			ctx, lpTooHighCommitment, lpparty, "liquidity-submission-5"),
		"commitment submission not allowed",
	)

	// now we will try to cancel the LP
	// at a point where we cannot fill the bond requirement
	// this should result into an error, and the commitment staying
	// untouched
	lpCancelCommitment := &types.LiquidityProvisionSubmission{
		MarketId:         tm.market.GetID(),
		CommitmentAmount: num.NewUint(0), // required commitment is 50000
		Fee:              num.DecimalFromFloat(0.01),
		Reference:        "ref-lp-submission-6",
		Buys:             []*types.LiquidityOrder{},
		Sells:            []*types.LiquidityOrder{},
	}

	tm.events = nil
	// submit our lp
	require.EqualError(t,
		tm.market.SubmitLiquidityProvision(
			ctx, lpCancelCommitment, lpparty, "liquidity-submission-6"),
		"commitment submission rejected, not enough stake",
	)

}

func TestCancelUndeployedCommitmentDuringAuction(t *testing.T) {
	now := time.Unix(10, 0)
	closingAt := time.Unix(1000000000, 0)
	ctx := context.Background()

	// auctionEnd := now.Add(10001 * time.Second)
	mktCfg := getMarket(closingAt, defaultPriceMonitorSettings, &types.AuctionDuration{
		Duration: 10000,
	})
	mktCfg.Fees = &types.Fees{
		Factors: &types.FeeFactors{
			LiquidityFee:      num.DecimalFromFloat(0.001),
			InfrastructureFee: num.DecimalFromFloat(0.0005),
			MakerFee:          num.DecimalFromFloat(0.00025),
		},
	}
	mktCfg.TradableInstrument.RiskModel = &types.TradableInstrument_LogNormalRiskModel{
		LogNormalRiskModel: &types.LogNormalRiskModel{
			RiskAversionParameter: num.DecimalFromFloat(0.001),
			Tau:                   num.DecimalFromFloat(0.00011407711613050422),
			Params: &types.LogNormalModelParams{
				Mu:    num.DecimalFromFloat(0),
				R:     num.DecimalFromFloat(0.016),
				Sigma: num.DecimalFromFloat(20),
			},
		},
	}

	lpparty := "lp-party-1"

	tm := newTestMarket(t, now).Run(ctx, mktCfg)
	tm.StartOpeningAuction().
		// the liquidity provider
		WithAccountAndAmount(lpparty, 500000000000)

	tm.market.OnSuppliedStakeToObligationFactorUpdate(1.0)
	tm.market.OnChainTimeUpdate(ctx, now)

	// Add a LPSubmission
	// this is a log of stake, enough to cover all
	// the required stake for the market
	lpSubmission := &types.LiquidityProvisionSubmission{
		MarketId:         tm.market.GetID(),
		CommitmentAmount: num.NewUint(70000),
		Fee:              num.DecimalFromFloat(0.01),
		Reference:        "ref-lp-submission-1",
		Buys: []*types.LiquidityOrder{
			{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_BID, Proportion: 2, Offset: -5},
			{Reference: types.PeggedReference_PEGGED_REFERENCE_MID, Proportion: 2, Offset: -5},
		},
		Sells: []*types.LiquidityOrder{
			{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_ASK, Proportion: 13, Offset: 5},
			{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_ASK, Proportion: 13, Offset: 5},
		},
	}

	// submit our lp
	require.NoError(t,
		tm.market.SubmitLiquidityProvision(
			ctx, lpSubmission, lpparty, "liquidity-submission-1"),
	)

	t.Run("bond account is updated with the new commitment", func(t *testing.T) {
		acc, err := tm.collateralEngine.GetPartyBondAccount(tm.market.GetID(), lpparty, tm.asset)
		assert.NoError(t, err)
		assert.Equal(t, num.NewUint(70000), acc.Balance)
	})

	// Add a LPSubmission
	// this is a log of stake, enough to cover all
	// the required stake for the market
	lpSubmissionCancel := &types.LiquidityProvisionSubmission{
		MarketId:         tm.market.GetID(),
		CommitmentAmount: num.NewUint(0),
		Fee:              num.DecimalFromFloat(0.01),
		Reference:        "ref-lp-submission-2",
		Buys:             []*types.LiquidityOrder{},
		Sells:            []*types.LiquidityOrder{},
	}

	// submit our lp
	require.NoError(t,
		tm.market.SubmitLiquidityProvision(
			ctx, lpSubmissionCancel, lpparty, "liquidity-submission-2"),
	)

	t.Run("bond account is updated with the new commitment", func(t *testing.T) {
		acc, err := tm.collateralEngine.GetPartyBondAccount(tm.market.GetID(), lpparty, tm.asset)
		assert.NoError(t, err)
		assert.Equal(t, num.NewUint(0), acc.Balance)
	})
}

func TestDeployedCommitmentIsUndeployedWhenEnteringAuction(t *testing.T) {
	now := time.Unix(10, 0)
	closingAt := time.Unix(1000000000, 0)
	ctx := context.Background()

	auctionEnd := now.Add(10001 * time.Second)
	mktCfg := getMarket(closingAt, defaultPriceMonitorSettings, &types.AuctionDuration{
		Duration: 10000,
	})
	mktCfg.Fees = &types.Fees{
		Factors: &types.FeeFactors{
			LiquidityFee:      num.DecimalFromFloat(0.001),
			InfrastructureFee: num.DecimalFromFloat(0.0005),
			MakerFee:          num.DecimalFromFloat(0.00025),
		},
	}
	mktCfg.TradableInstrument.RiskModel = &types.TradableInstrument_LogNormalRiskModel{
		LogNormalRiskModel: &types.LogNormalRiskModel{
			RiskAversionParameter: num.DecimalFromFloat(0.001),
			Tau:                   num.DecimalFromFloat(0.00011407711613050422),
			Params: &types.LogNormalModelParams{
				Mu:    num.DecimalFromFloat(0),
				R:     num.DecimalFromFloat(0.016),
				Sigma: num.DecimalFromFloat(20),
			},
		},
	}

	lpparty := "lp-party-1"

	tm := newTestMarket(t, now).Run(ctx, mktCfg)
	tm.StartOpeningAuction().
		// the liquidity provider
		WithAccountAndAmount(lpparty, 500000000000)

	tm.market.OnSuppliedStakeToObligationFactorUpdate(0.20)
	tm.market.OnChainTimeUpdate(ctx, now)

	// Add a LPSubmission
	// this is a log of stake, enough to cover all
	// the required stake for the market
	lpSubmission := &types.LiquidityProvisionSubmission{
		MarketId:         tm.market.GetID(),
		CommitmentAmount: num.NewUint(150000),
		Fee:              num.DecimalFromFloat(0.01),
		Reference:        "ref-lp-submission-1",
		Buys: []*types.LiquidityOrder{
			{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_BID, Proportion: 2, Offset: -5},
			{Reference: types.PeggedReference_PEGGED_REFERENCE_MID, Proportion: 2, Offset: -5},
		},
		Sells: []*types.LiquidityOrder{
			{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_ASK, Proportion: 13, Offset: 5},
			{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_ASK, Proportion: 13, Offset: 5},
		},
	}

	// submit our lp
	require.NoError(t,
		tm.market.SubmitLiquidityProvision(
			ctx, lpSubmission, lpparty, "liquidity-submission-1"),
	)

	tm.events = nil
	tm.EndOpeningAuction(t, auctionEnd, false)

	t.Run("bond account is updated with the new commitment", func(t *testing.T) {
		acc, err := tm.collateralEngine.GetPartyMarginAccount(
			tm.market.GetID(), lpparty, tm.asset)
		assert.NoError(t, err)
		assert.Equal(t, num.NewUint(75129), acc.Balance)
	})

	tm.market.OnChainTimeUpdate(ctx, auctionEnd.Add(2*time.Second))
	tm.mas.StartPriceAuction(auctionEnd.Add(2*time.Second), &types.AuctionDuration{
		Duration: 30,
	})

	tm.events = nil
	tm.market.EnterAuction(ctx)

	t.Run("previous LP orders to be cancelled", func(t *testing.T) {
		// First collect all the orders events
		found := []*types.Order{}
		for _, e := range tm.events {
			switch evt := e.(type) {
			case *events.Order:
				found = append(found, types.OrderFromProto(evt.Order()))
			}
		}

		require.Len(t, found, 4)

		// only 4 cancellations
		i := 0
		for _, o := range found {
			var expectedStatus = types.Order_STATUS_CANCELLED
			assert.Equal(t,
				expectedStatus.String(),
				o.Status.String(),
			)
			i += 1
		}
	})

	// then we are leaving the auction period
	tm.events = nil
	tm.market.OnChainTimeUpdate(ctx, auctionEnd.Add(50*time.Second))

	t.Run("LP orders are re-submitted after auction", func(t *testing.T) {
		// First collect all the orders events
		found := []*types.Order{}
		for _, e := range tm.events {
			switch evt := e.(type) {
			case *events.Order:
				found = append(found, types.OrderFromProto(evt.Order()))
			}
		}

		require.Len(t, found, 4)

		for _, o := range found {
			var expectedStatus = types.Order_STATUS_ACTIVE
			assert.Equal(t,
				expectedStatus.String(),
				o.Status.String(),
			)
		}
	})
}

func TestDeployedCommitmentIsUndeployedWhenEnteringAuctionAndMarginCheckFailDuringAuction(t *testing.T) {
	now := time.Unix(10, 0)
	closingAt := time.Unix(1000000000, 0)
	ctx := context.Background()

	auctionEnd := now.Add(10001 * time.Second)
	mktCfg := getMarket(closingAt, defaultPriceMonitorSettings, &types.AuctionDuration{
		Duration: 10000,
	})
	mktCfg.Fees = &types.Fees{
		Factors: &types.FeeFactors{
			LiquidityFee:      num.DecimalFromFloat(0.001),
			InfrastructureFee: num.DecimalFromFloat(0.0005),
			MakerFee:          num.DecimalFromFloat(0.00025),
		},
	}
	mktCfg.TradableInstrument.RiskModel = &types.TradableInstrument_LogNormalRiskModel{
		LogNormalRiskModel: &types.LogNormalRiskModel{
			RiskAversionParameter: num.DecimalFromFloat(0.001),
			Tau:                   num.DecimalFromFloat(0.00011407711613050422),
			Params: &types.LogNormalModelParams{
				Mu:    num.DecimalFromFloat(0),
				R:     num.DecimalFromFloat(0.016),
				Sigma: num.DecimalFromFloat(20),
			},
		},
	}

	lpparty := "lp-party-1"

	tm := newTestMarket(t, now).Run(ctx, mktCfg)
	tm.StartOpeningAuction().
		// the liquidity provider
		WithAccountAndAmount(lpparty, 781648).
		WithAccountAndAmount("party-yolo", 1000000000).
		WithAccountAndAmount("party-yolo1", 1000000000)

	tm.market.OnSuppliedStakeToObligationFactorUpdate(1)
	tm.market.OnChainTimeUpdate(ctx, now)

	// Add a LPSubmission
	// this is a log of stake, enough to cover all
	// the required stake for the market
	lpSubmission := &types.LiquidityProvisionSubmission{
		MarketId:         tm.market.GetID(),
		CommitmentAmount: num.NewUint(150000),
		Fee:              num.DecimalFromFloat(0.01),
		Reference:        "ref-lp-submission-1",
		Buys: []*types.LiquidityOrder{
			{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_BID, Proportion: 2, Offset: -5},
			{Reference: types.PeggedReference_PEGGED_REFERENCE_MID, Proportion: 2, Offset: -5},
		},
		Sells: []*types.LiquidityOrder{
			{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_ASK, Proportion: 13, Offset: 5},
			{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_ASK, Proportion: 13, Offset: 5},
		},
	}

	// submit our lp
	require.NoError(t,
		tm.market.SubmitLiquidityProvision(
			ctx, lpSubmission, lpparty, "liquidity-submission-1"),
	)

	tm.events = nil
	tm.EndOpeningAuction(t, auctionEnd, false)

	t.Run("margin account is updated with margins", func(t *testing.T) {
		acc, err := tm.collateralEngine.GetPartyBondAccount(
			tm.market.GetID(), lpparty, tm.asset)
		assert.NoError(t, err)
		assert.Equal(t, num.NewUint(150000), acc.Balance)
		// acc, err := tm.collateralEngine.GetPartyMarginAccount(
		acc, err = tm.collateralEngine.GetPartyMarginAccount(
			tm.market.GetID(), lpparty, tm.asset)
		assert.NoError(t, err)
		assert.True(t, acc.Balance.EQ(num.NewUint(368378)))
	})

	tm.market.OnChainTimeUpdate(ctx, auctionEnd.Add(2*time.Second))
	tm.mas.StartPriceAuction(auctionEnd.Add(2*time.Second), &types.AuctionDuration{
		Duration: 30,
	})

	tm.events = nil
	tm.market.EnterAuction(ctx)

	t.Run("previous LP orders to be cancelled", func(t *testing.T) {
		// First collect all the orders events
		found := []*types.Order{}
		for _, e := range tm.events {
			switch evt := e.(type) {
			case *events.Order:
				found = append(found, types.OrderFromProto(evt.Order()))
			}
		}

		require.Len(t, found, 4)

		// 4 cancellations
		i := 0
		for _, o := range found {
			var expectedStatus = types.Order_STATUS_CANCELLED
			assert.Equal(t,
				expectedStatus.String(),
				o.Status.String(),
			)
			i += 1
		}
	})

	// commitment is being updated during auction
	lpSubmissionUpdate := &types.LiquidityProvisionSubmission{
		MarketId:         tm.market.GetID(),
		CommitmentAmount: num.NewUint(200000),
		Fee:              num.DecimalFromFloat(0.01),
		Reference:        "ref-lp-submission-2",
		Buys: []*types.LiquidityOrder{
			{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_BID, Proportion: 2, Offset: -5},
			{Reference: types.PeggedReference_PEGGED_REFERENCE_MID, Proportion: 2, Offset: -5},
		},
		Sells: []*types.LiquidityOrder{
			{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_ASK, Proportion: 13, Offset: 5},
			{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_ASK, Proportion: 13, Offset: 5},
		},
	}

	// the submission should be all OK
	// order are not deployed while still in auction
	require.EqualError(t,
		tm.market.SubmitLiquidityProvision(
			ctx, lpSubmissionUpdate, lpparty, "liquidity-submission-2"),
		"margin would be below maintenance: insufficient margin",
	)
}
