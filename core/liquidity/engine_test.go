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

package liquidity_test

import (
	"context"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/core/idgeneration"
	"code.vegaprotocol.io/vega/libs/crypto"

	"code.vegaprotocol.io/vega/core/integration/stubs"

	bmocks "code.vegaprotocol.io/vega/core/broker/mocks"
	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/liquidity"
	"code.vegaprotocol.io/vega/core/liquidity/mocks"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/logging"
	proto "code.vegaprotocol.io/vega/protos/vega"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// eq implements a gomock.Matcher with a better diff output.
type eqMatcher struct {
	t        *testing.T
	expected interface{}
}

func (eq eqMatcher) Matches(v interface{}) bool {
	return assert.Equal(eq.t, eq.expected, v)
}

func (eqMatcher) String() string {
	return "assert.Equal(expected, got)"
}

func eq(t *testing.T, x interface{}) eqMatcher {
	t.Helper()
	return eqMatcher{t, x}
}

type testEngine struct {
	ctrl         *gomock.Controller
	marketID     string
	tsvc         *mocks.MockTimeService
	broker       *bmocks.MockBroker
	riskModel    *mocks.MockRiskModel
	priceMonitor *mocks.MockPriceMonitor
	orderbook    *mocks.MockOrderBook
	engine       *liquidity.SnapshotEngine
	stateVar     *stubs.StateVarStub
}

func newTestEngine(t *testing.T, now time.Time) *testEngine {
	t.Helper()
	ctrl := gomock.NewController(t)

	log := logging.NewTestLogger()
	tsvc := mocks.NewMockTimeService(ctrl)
	tsvc.EXPECT().GetTimeNow().DoAndReturn(
		func() time.Time {
			return now
		}).AnyTimes()

	broker := bmocks.NewMockBroker(ctrl)
	risk := mocks.NewMockRiskModel(ctrl)
	monitor := mocks.NewMockPriceMonitor(ctrl)
	orderbook := mocks.NewMockOrderBook(ctrl)
	market := "market-id"
	asset := "asset-id"
	liquidityConfig := liquidity.NewDefaultConfig()
	stateVarEngine := stubs.NewStateVar()
	risk.EXPECT().GetProjectionHorizon().AnyTimes()

	engine := liquidity.NewSnapshotEngine(liquidityConfig,
		log, tsvc, broker, risk, monitor, orderbook, asset, market, stateVarEngine, num.NewUint(100000), num.DecimalFromInt64(1),
	)

	return &testEngine{
		ctrl:         ctrl,
		marketID:     market,
		tsvc:         tsvc,
		broker:       broker,
		riskModel:    risk,
		priceMonitor: monitor,
		orderbook:    orderbook,
		engine:       engine,
		stateVar:     stateVarEngine,
	}
}

func TestSubmissions(t *testing.T) {
	t.Run("CreateUpdateDelete", testSubmissionCRUD)
	t.Run("CancelNonExisting", testCancelNonExistingSubmission)
	t.Run("FailWhenWithoutBothShapes", testSubmissionFailWithoutBothShapes)
}

func testSubmissionCRUD(t *testing.T) {
	var (
		party = "party-1"
		ctx   = context.Background()
		now   = time.Now()
		tng   = newTestEngine(t, now)
	)
	defer tng.ctrl.Finish()

	require.Nil(t, tng.engine.LiquidityProvisionByPartyID("some-party"))

	buyShape := []*types.LiquidityOrder{
		{
			Reference:  types.PeggedReferenceMid,
			Offset:     num.NewUint(1),
			Proportion: 1,
		},
	}
	sellShape := []*types.LiquidityOrder{
		{
			Reference:  types.PeggedReferenceMid,
			Offset:     num.NewUint(1),
			Proportion: 1,
		},
	}

	pbBuys := make([]*proto.LiquidityOrder, 0, len(buyShape))
	pbSells := make([]*proto.LiquidityOrder, 0, len(sellShape))
	for _, b := range buyShape {
		pbBuys = append(pbBuys, b.IntoProto())
	}
	for _, s := range sellShape {
		pbSells = append(pbSells, s.IntoProto())
	}

	lps1 := &commandspb.LiquidityProvisionSubmission{
		MarketId: tng.marketID, CommitmentAmount: "100", Fee: "0.5",
		Buys: pbBuys, Sells: pbSells,
	}
	lps, err := types.LiquidityProvisionSubmissionFromProto(lps1)
	require.NoError(t, err)

	deterministicID := crypto.RandomHash()
	idGen := idgeneration.New(deterministicID)

	lpID := idGen.NextID()
	order1 := &types.Order{}
	order2 := &types.Order{}
	order1.ID = idGen.NextID()
	order2.ID = idGen.NextID()

	expected := &types.LiquidityProvision{
		ID:               lpID,
		MarketID:         tng.marketID,
		Party:            party,
		Fee:              num.DecimalFromFloat(0.5),
		CommitmentAmount: lps.CommitmentAmount.Clone(),
		CreatedAt:        now.UnixNano(),
		UpdatedAt:        now.UnixNano(),
		Status:           types.LiquidityProvisionStatusPending,
		Version:          1,
		Buys: []*types.LiquidityOrderReference{
			{LiquidityOrder: buyShape[0], OrderID: order1.ID},
		},

		Sells: []*types.LiquidityOrderReference{
			{LiquidityOrder: sellShape[0], OrderID: order2.ID},
		},
	}

	// Create a submission should fire an event
	tng.broker.EXPECT().Send(
		events.NewLiquidityProvisionEvent(ctx, expected),
	).Times(1)

	idgen := idgeneration.New(deterministicID)
	require.NoError(t,
		tng.engine.SubmitLiquidityProvision(ctx, lps, party, idgen))
	got := tng.engine.LiquidityProvisionByPartyID(party)
	require.Equal(t, expected, got)

	expected.Status = types.LiquidityProvisionStatusCancelled
	tng.broker.EXPECT().Send(
		events.NewLiquidityProvisionEvent(ctx, expected),
	).Times(1)

	err = tng.engine.CancelLiquidityProvision(ctx, party)
	require.NoError(t, err)
	require.Nil(t, tng.engine.LiquidityProvisionByPartyID(party),
		"Party '%s' should not be a LiquidityProvider after Committing 0 amount", party)
}

func TestInitialDeployFailsWorksLater(t *testing.T) {
	var (
		party = "party-1"
		ctx   = context.Background()
		now   = time.Now()
		tng   = newTestEngine(t, now)
	)
	defer tng.ctrl.Finish()

	// We don't care about the following calls
	tng.broker.EXPECT().Send(gomock.Any()).AnyTimes()
	tng.broker.EXPECT().SendBatch(gomock.Any()).AnyTimes()
	tng.orderbook.EXPECT().GetOrdersPerParty(party).Times(1)
	tng.orderbook.EXPECT().GetLiquidityOrders(gomock.Any()).AnyTimes()

	// Send a submission to create the shape
	lpspb := &commandspb.LiquidityProvisionSubmission{
		MarketId: tng.marketID, CommitmentAmount: "100", Fee: "0.5",
		Buys: []*proto.LiquidityOrder{
			{Reference: types.PeggedReferenceMid, Proportion: 20, Offset: "1"},
			{Reference: types.PeggedReferenceMid, Proportion: 10, Offset: "2"},
		},
		Sells: []*proto.LiquidityOrder{
			{Reference: types.PeggedReferenceMid, Proportion: 1, Offset: "1"},
		},
	}
	lps, err := types.LiquidityProvisionSubmissionFromProto(lpspb)
	require.NoError(t, err)

	idgen := idgeneration.New(crypto.RandomHash())
	require.NoError(t,
		tng.engine.SubmitLiquidityProvision(ctx, lps, party, idgen),
	)

	require.True(t, tng.engine.IsLiquidityProvider(party))

	markPrice := num.NewUint(10)

	// Now repriceFn works as expected, so initial orders should get created now
	fn := func(side types.Side, ref types.PeggedReference, offset *num.Uint) (*num.Uint, error) {
		retPrice := markPrice.Clone()
		if side == types.SideSell {
			return retPrice.Add(retPrice, offset), nil
		}
		return retPrice.Sub(retPrice, offset), nil
	}

	newOrders, amendments := tng.engine.Update(context.Background(), num.UintOne(), num.NewUint(100), fn)
	require.Len(t, newOrders, 3)
	require.Len(t, amendments, 0)
}

func testCancelNonExistingSubmission(t *testing.T) {
	var (
		party = "party-1"
		ctx   = context.Background()
		now   = time.Now()
		tng   = newTestEngine(t, now)
	)
	defer tng.ctrl.Finish()

	err := tng.engine.CancelLiquidityProvision(ctx, party)
	require.Error(t, err)
}

func testSubmissionFailWithoutBothShapes(t *testing.T) {
	var (
		party = "party-1"
		ctx   = context.Background()
		now   = time.Now()
		tng   = newTestEngine(t, now)
	)
	defer tng.ctrl.Finish()

	// Expectations
	lpspb := &commandspb.LiquidityProvisionSubmission{
		CommitmentAmount: "10",
		MarketId:         tng.marketID,
		Fee:              "0.1",
		Buys: []*proto.LiquidityOrder{
			{
				Reference:  types.PeggedReferenceMid,
				Offset:     "1",
				Proportion: 1,
			},
		},
	}
	lps, err := types.LiquidityProvisionSubmissionFromProto(lpspb)
	require.NoError(t, err)

	lpID := crypto.RandomHash()
	expected := events.NewLiquidityProvisionEvent(ctx, &types.LiquidityProvision{
		ID:               lpID,
		MarketID:         tng.marketID,
		Party:            party,
		CreatedAt:        now.UnixNano(),
		Status:           types.LiquidityProvisionStatusRejected,
		Fee:              num.DecimalFromFloat(0.1),
		CommitmentAmount: num.NewUint(10),
		Sells:            []*types.LiquidityOrderReference{},
		Buys: []*types.LiquidityOrderReference{
			{
				LiquidityOrder: &types.LiquidityOrder{
					Reference:  types.PeggedReferenceMid,
					Offset:     num.NewUint(1),
					Proportion: 1,
				},
			},
		},
	})

	tng.broker.EXPECT().Send(eq(t, expected)).Times(1)

	idgen := idgeneration.New(lpID)
	require.Error(t,
		tng.engine.SubmitLiquidityProvision(ctx, lps, party, idgen),
	)

	lpspb = &commandspb.LiquidityProvisionSubmission{
		CommitmentAmount: "10",
		MarketId:         tng.marketID,
		Fee:              "0.2",
		Sells: []*proto.LiquidityOrder{
			{
				Reference:  types.PeggedReferenceMid,
				Offset:     "1",
				Proportion: 1,
			},
		},
	}
	lps, err = types.LiquidityProvisionSubmissionFromProto(lpspb)
	require.NoError(t, err)

	expected = events.NewLiquidityProvisionEvent(ctx, &types.LiquidityProvision{
		ID:               lpID,
		Fee:              num.DecimalFromFloat(0.2),
		MarketID:         tng.marketID,
		Party:            party,
		CreatedAt:        now.UnixNano(),
		CommitmentAmount: num.NewUint(10),
		Status:           types.LiquidityProvisionStatusRejected,
		Buys:             []*types.LiquidityOrderReference{},
		Sells: []*types.LiquidityOrderReference{
			{
				LiquidityOrder: &types.LiquidityOrder{
					Reference:  types.PeggedReferenceMid,
					Offset:     num.NewUint(1),
					Proportion: 1,
				},
			},
		},
	})

	tng.broker.EXPECT().Send(eq(t, expected)).Times(1)

	idgen = idgeneration.New(lpID)
	require.Error(t,
		tng.engine.SubmitLiquidityProvision(ctx, lps, party, idgen),
	)

	lpspb = &commandspb.LiquidityProvisionSubmission{
		Fee:              "0.3",
		CommitmentAmount: "10",
		MarketId:         tng.marketID,
	}
	lps, _ = types.LiquidityProvisionSubmissionFromProto(lpspb)

	expected = events.NewLiquidityProvisionEvent(ctx, &types.LiquidityProvision{
		ID:               lpID,
		MarketID:         tng.marketID,
		Fee:              num.DecimalFromFloat(0.3),
		Party:            party,
		CreatedAt:        now.UnixNano(),
		CommitmentAmount: num.NewUint(10),
		Status:           types.LiquidityProvisionStatusRejected,
		Buys:             []*types.LiquidityOrderReference{},
		Sells:            []*types.LiquidityOrderReference{},
	})

	tng.broker.EXPECT().Send(eq(t, expected)).Times(1)

	idgen = idgeneration.New(lpID)
	require.Error(t,
		tng.engine.SubmitLiquidityProvision(ctx, lps, party, idgen),
	)
}

func TestUpdateAndUndeploy(t *testing.T) {
	var (
		party = "party-1"
		ctx   = context.Background()
		now   = time.Now()
		tng   = newTestEngine(t, now)
	)
	defer tng.ctrl.Finish()

	// We don't care about the following calls
	tng.broker.EXPECT().Send(gomock.Any()).AnyTimes()
	tng.orderbook.EXPECT().GetLiquidityOrders(gomock.Any()).Times(2)

	// Send a submission to create the shape
	lpspb := &commandspb.LiquidityProvisionSubmission{
		MarketId: tng.marketID, CommitmentAmount: "100", Fee: "0.5",
		Buys: []*proto.LiquidityOrder{
			{Reference: types.PeggedReferenceMid, Proportion: 20, Offset: "1"},
			{Reference: types.PeggedReferenceMid, Proportion: 10, Offset: "2"},
		},
		Sells: []*proto.LiquidityOrder{
			{Reference: types.PeggedReferenceMid, Proportion: 1, Offset: "1"},
		},
	}
	lps, err := types.LiquidityProvisionSubmissionFromProto(lpspb)
	require.NoError(t, err)

	idgen := idgeneration.New(crypto.RandomHash())
	require.NoError(t,
		tng.engine.SubmitLiquidityProvision(ctx, lps, party, idgen),
	)

	markPrice := num.NewUint(10)

	fn := func(side types.Side, ref types.PeggedReference, offset *num.Uint) (*num.Uint, error) {
		retPrice := markPrice.Clone()
		if side == types.SideSell {
			return retPrice.Add(retPrice, offset), nil
		}
		return retPrice.Sub(retPrice, offset), nil
	}

	// Expectations
	orders := []*types.Order{
		{ID: "1", Party: party, Price: num.NewUint(10), Size: 1, Side: types.SideBuy, Status: types.OrderStatusActive},
		{ID: "2", Party: party, Price: num.NewUint(11), Size: 1, Side: types.SideSell, Status: types.OrderStatusActive},
	}
	tng.orderbook.EXPECT().GetOrdersPerParty(party).Times(3).Return(orders)
	creates := tng.engine.CreateInitialOrders(ctx, num.UintOne(), num.NewUint(100), party, fn)
	require.Len(t, creates, 3)

	tng.orderbook.EXPECT().GetLiquidityOrders(gomock.Any()).Times(2).Return(creates)

	// Manual order satisfies the commitment, LiqOrders should be removed
	orders[0].Remaining, orders[0].Size = 1000, 1000
	orders[1].Remaining, orders[1].Size = 1000, 1000
	newOrders, toCancels := tng.engine.Update(ctx, num.UintOne(), num.NewUint(100), fn)
	require.Len(t, newOrders, 0)
	require.Len(t, toCancels[0].OrderIDs, 3)
	require.Equal(t, toCancels[0].Party, party)

	tng.orderbook.EXPECT().GetLiquidityOrders(gomock.Any()).Times(2)
	newOrders, toCancels = tng.engine.Update(ctx, num.UintOne(), num.NewUint(100), fn)
	require.Len(t, newOrders, 0)
	require.Len(t, toCancels, 0)

	tng.orderbook.EXPECT().GetLiquidityOrders(gomock.Any()).Times(2)
	tng.broker.EXPECT().SendBatch(gomock.Any()).Times(1)
	tng.engine.UndeployLPs(ctx, nil)
	lp := tng.engine.LiquidityProvisionByPartyID(party)
	require.Equal(t, types.LiquidityProvisionStatusUndeployed, lp.Status)
}

func TestCalculateSuppliedStake(t *testing.T) {
	var (
		party1 = "party-1"
		party2 = "party-2"
		party3 = "party-3"
		ctx    = context.Background()
		now    = time.Now()
		tng    = newTestEngine(t, now)
	)
	defer tng.ctrl.Finish()

	// We don't care about the following calls
	tng.broker.EXPECT().Send(gomock.Any()).AnyTimes()

	// Send a submission to create the shape
	lp1pb := &commandspb.LiquidityProvisionSubmission{
		MarketId: tng.marketID, CommitmentAmount: "100", Fee: "0.5",
		Buys: []*proto.LiquidityOrder{
			{Reference: types.PeggedReferenceMid, Proportion: 20, Offset: "1"},
			{Reference: types.PeggedReferenceMid, Proportion: 10, Offset: "2"},
		},
		Sells: []*proto.LiquidityOrder{
			{Reference: types.PeggedReferenceMid, Proportion: 1, Offset: "1"},
		},
	}
	lp1, err := types.LiquidityProvisionSubmissionFromProto(lp1pb)
	require.NoError(t, err)

	idgen := idgeneration.New(crypto.RandomHash())
	require.NoError(t,
		tng.engine.SubmitLiquidityProvision(ctx, lp1, party1, idgen),
	)
	suppliedStake := tng.engine.CalculateSuppliedStake()
	require.Equal(t, lp1.CommitmentAmount, suppliedStake)

	lp2pb := &commandspb.LiquidityProvisionSubmission{
		MarketId: tng.marketID, CommitmentAmount: "500", Fee: "0.5",
		Buys: []*proto.LiquidityOrder{
			{Reference: types.PeggedReferenceMid, Proportion: 1, Offset: "3"},
		},
		Sells: []*proto.LiquidityOrder{
			{Reference: types.PeggedReferenceMid, Proportion: 1, Offset: "3"},
		},
	}
	lp2, err := types.LiquidityProvisionSubmissionFromProto(lp2pb)
	require.NoError(t, err)

	idgen = idgeneration.New(crypto.RandomHash())
	require.NoError(t,
		tng.engine.SubmitLiquidityProvision(ctx, lp2, party2, idgen),
	)
	suppliedStake = tng.engine.CalculateSuppliedStake()
	require.Equal(t, num.Sum(lp1.CommitmentAmount, lp2.CommitmentAmount), suppliedStake)

	lp3pb := &commandspb.LiquidityProvisionSubmission{
		MarketId: tng.marketID, CommitmentAmount: "962", Fee: "0.5",
		Buys: []*proto.LiquidityOrder{
			{Reference: types.PeggedReferenceMid, Proportion: 1, Offset: "5"},
		},
		Sells: []*proto.LiquidityOrder{
			{Reference: types.PeggedReferenceMid, Proportion: 1, Offset: "1"},
			{Reference: types.PeggedReferenceMid, Proportion: 1, Offset: "10"},
		},
	}
	lp3, err := types.LiquidityProvisionSubmissionFromProto(lp3pb)
	require.NoError(t, err)

	idgen = idgeneration.New(crypto.RandomHash())
	require.NoError(t,
		tng.engine.SubmitLiquidityProvision(ctx, lp3, party3, idgen),
	)
	suppliedStake = tng.engine.CalculateSuppliedStake()
	require.Equal(t, num.Sum(lp1.CommitmentAmount, lp2.CommitmentAmount, lp3.CommitmentAmount), suppliedStake)

	err = tng.engine.CancelLiquidityProvision(ctx, party1)
	require.NoError(t, err)
	suppliedStake = tng.engine.CalculateSuppliedStake()
	require.Equal(t, num.Sum(lp2.CommitmentAmount, lp3.CommitmentAmount), suppliedStake)
}

func TestLiquidityScoresMechanics(t *testing.T) {
	var (
		party1     = "party-1"
		party2     = "party-2"
		party3     = "party-3"
		party4     = "party-4"
		ctx        = context.Background()
		now        = time.Now()
		tng        = newTestEngine(t, now)
		bestBid    = num.NewDecimalFromFloat(95)
		bestAsk    = num.NewDecimalFromFloat(105)
		minLpPrice = num.NewUint(90)
		maxLpPrice = num.NewUint(110)
		minPmPrice = num.NewWrappedDecimal(num.NewUint(85), num.DecimalFromFloat(85))
		maxPmPrice = num.NewWrappedDecimal(num.NewUint(115), num.DecimalFromFloat(115))
		commitment = 1000000
		offset     = num.NewUint(2)
	)
	defer tng.ctrl.Finish()
	tng.priceMonitor.EXPECT().GetValidPriceRange().AnyTimes().Return(minPmPrice, maxPmPrice).AnyTimes()

	gomock.InOrder(
		tng.riskModel.EXPECT().ProbabilityOfTrading(bestBid, gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(num.DecimalFromFloat(0.5)),
		tng.riskModel.EXPECT().ProbabilityOfTrading(bestBid, gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(num.DecimalFromFloat(0.4)),
		tng.riskModel.EXPECT().ProbabilityOfTrading(bestBid, gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(num.DecimalFromFloat(0.3)),
		tng.riskModel.EXPECT().ProbabilityOfTrading(bestBid, gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(num.DecimalFromFloat(0.2)),
		tng.riskModel.EXPECT().ProbabilityOfTrading(bestBid, gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(num.DecimalFromFloat(0.1)),
		tng.riskModel.EXPECT().ProbabilityOfTrading(bestBid, gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(num.DecimalFromFloat(0.0)),
	)
	gomock.InOrder(
		tng.riskModel.EXPECT().ProbabilityOfTrading(bestAsk, gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(num.DecimalFromFloat(0.5)),
		tng.riskModel.EXPECT().ProbabilityOfTrading(bestAsk, gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(num.DecimalFromFloat(0.4)),
		tng.riskModel.EXPECT().ProbabilityOfTrading(bestAsk, gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(num.DecimalFromFloat(0.3)),
		tng.riskModel.EXPECT().ProbabilityOfTrading(bestAsk, gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(num.DecimalFromFloat(0.2)),
		tng.riskModel.EXPECT().ProbabilityOfTrading(bestAsk, gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(num.DecimalFromFloat(0.1)),
		tng.riskModel.EXPECT().ProbabilityOfTrading(bestAsk, gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(num.DecimalFromFloat(0.0)),
	)

	// We don't care about the following calls
	tng.broker.EXPECT().Send(gomock.Any()).AnyTimes()

	// initialise PoT
	tng.engine.SetGetStaticPricesFunc(func() (num.Decimal, num.Decimal, error) { return bestBid, bestAsk, nil })
	tng.stateVar.OnTick(ctx, now)
	require.True(t, tng.engine.IsPoTInitialised())

	// party1 submission
	tng.sortOutLpSubAndOrders(t, ctx, party1, commitment, offset, minLpPrice, maxLpPrice, bestBid, bestAsk, 9)

	cLiq1, t1 := tng.engine.GetCurrentLiquidityScores(bestBid, bestAsk, minLpPrice, maxLpPrice)
	require.Len(t, cLiq1, 1)
	require.True(t, t1.GreaterThan(num.DecimalZero()))

	tng.engine.UpdateAverageLiquidityScores(bestBid, bestAsk, minLpPrice, maxLpPrice)
	lScores1 := tng.engine.GetAverageLiquidityScores()
	require.Len(t, lScores1, 1)
	lScoresSumTo1(t, lScores1)

	// party2 submission with 3*commitment
	tng.sortOutLpSubAndOrders(t, ctx, party2, 3*commitment, offset, minLpPrice, maxLpPrice, bestBid, bestAsk, 100)

	cLiq2, t2 := tng.engine.GetCurrentLiquidityScores(bestBid, bestAsk, minLpPrice, maxLpPrice)
	require.Len(t, cLiq2, 2)
	require.True(t, t2.GreaterThan(num.DecimalZero()))

	p1cLiq := cLiq2[party1].Copy()
	p2cLiqExp := p1cLiq.Mul(num.DecimalFromFloat(3))
	// there's some ceiling going on when creating order volumes from commitment so check results within delta
	expFP, _ := p2cLiqExp.Float64()
	actFP, _ := cLiq2[party2].Float64()
	require.InDelta(t, expFP, actFP, 1e-4*float64(commitment))

	tng.engine.UpdateAverageLiquidityScores(bestBid, bestAsk, minLpPrice, maxLpPrice)
	lScores2 := tng.engine.GetAverageLiquidityScores()
	require.Len(t, lScores2, 2)
	lScoresSumTo1(t, lScores2)

	// party3 submission with 3*offset
	offsetTimes3 := num.UintZero().Mul(offset, num.NewUint(3))
	tng.sortOutLpSubAndOrders(t, ctx, party3, commitment, offsetTimes3, minLpPrice, maxLpPrice, bestBid, bestAsk, 100)

	cLiq3, t3 := tng.engine.GetCurrentLiquidityScores(bestBid, bestAsk, minLpPrice, maxLpPrice)
	require.Len(t, cLiq3, 3)
	require.True(t, t3.GreaterThan(num.DecimalZero()))
	require.True(t, cLiq3[party1].GreaterThan(cLiq3[party3]))

	tng.engine.UpdateAverageLiquidityScores(bestBid, bestAsk, minLpPrice, maxLpPrice)
	lScores3 := tng.engine.GetAverageLiquidityScores()
	require.Len(t, lScores3, 3)
	lScoresSumTo1(t, lScores3)

	// now add 1 LP, remove 1 LP and change
	//    remove party3
	require.NoError(t, tng.engine.CancelLiquidityProvision(ctx, party3))
	//    add same submission as party3, but by party4
	tng.sortOutLpSubAndOrders(t, ctx, party4, commitment, offsetTimes3, minLpPrice, maxLpPrice, bestBid, bestAsk, 100)

	cLiq4, t4 := tng.engine.GetCurrentLiquidityScores(bestBid, bestAsk, minLpPrice, maxLpPrice)
	require.Len(t, cLiq4, 3)
	require.True(t, t4.GreaterThan(num.DecimalZero()))
	// should get same value for party4 as for party3 in previous round
	require.True(t, cLiq4[party4].Equal(cLiq3[party3]))

	tng.engine.UpdateAverageLiquidityScores(bestBid, bestAsk, minLpPrice, maxLpPrice)
	lScores4 := tng.engine.GetAverageLiquidityScores()
	require.Len(t, lScores4, 3)
	lScoresSumTo1(t, lScores4)

	keys := make([]string, 0, len(lScores4))
	for k := range lScores4 {
		keys = append(keys, k)
	}
	activeParties := []string{party1, party2, party4}
	require.ElementsMatch(t, activeParties, keys)

	tng.sortOutLpAmendmentAndOrders(t, ctx, party1, 3*commitment, offset, minLpPrice, maxLpPrice, bestBid, bestAsk)

	cLiq5, t5 := tng.engine.GetCurrentLiquidityScores(bestBid, bestAsk, minLpPrice, maxLpPrice)
	require.Len(t, cLiq5, 3)
	require.True(t, t5.GreaterThan(num.DecimalZero()))
	// commitment size should have almost no impact on score (only via relative order size differences due to ceiling)
	expFP, _ = cLiq4[party1].Float64()
	actFP, _ = cLiq5[party1].Float64()
	require.InDelta(t, expFP, actFP, 1e-4)

	tng.engine.UpdateAverageLiquidityScores(bestBid, bestAsk, minLpPrice, maxLpPrice)
	lScores5 := tng.engine.GetAverageLiquidityScores()
	require.Len(t, lScores5, 3)
	lScoresSumTo1(t, lScores5)

	// check running average
	n := num.DecimalFromInt64(5)
	nMinus1 := n.Sub(num.DecimalOne())
	nMinus1overN := nMinus1.Div(n)
	expectedScore := (lScores4[party1].Mul(nMinus1overN).Add(cLiq5[party1].Div(t5).Div(n))).Round(10)
	require.True(t, expectedScore.Equal(lScores5[party1]))

	// now reset scores and do another round
	tng.engine.ResetAverageLiquidityScores()
	tng.engine.UpdateAverageLiquidityScores(bestBid, bestAsk, minLpPrice, maxLpPrice)

	lScores6 := tng.engine.GetAverageLiquidityScores()
	require.Len(t, lScores6, 3)
	lScoresSumTo1(t, lScores6)
	for _, p := range activeParties {
		// we've just reset so running average should be same as previous observation normalised
		require.True(t, lScores6[p].Equal((cLiq5[p].Div(t5)).Round(10)))
	}
}

func (tng *testEngine) sortOutLpSubAndOrders(
	t *testing.T,
	ctx context.Context,
	party string,
	commitment int,
	offset *num.Uint,
	minLpPrice, maxLpPrice *num.Uint,
	bestBid, bestAsk num.Decimal,
	maxTimes int,
) {
	t.Helper()
	fn := func(side types.Side, ref types.PeggedReference, offset *num.Uint) (*num.Uint, error) {
		r := bestBid.Copy()
		if ref == types.PeggedReferenceBestAsk {
			r = bestAsk.Copy()
		}
		if ref == types.PeggedReferenceMid {
			r = r.Add(bestAsk).Div(num.DecimalFromInt64(2))
		}
		retPrice, _ := num.UintFromDecimal(r)
		if side == types.SideSell {
			return retPrice.Add(retPrice, offset), nil
		}
		return retPrice.Sub(retPrice, offset), nil
	}
	lps := &types.LiquidityProvisionSubmission{
		MarketID:         tng.marketID,
		CommitmentAmount: num.NewUint(uint64(commitment)),
		Fee:              num.DecimalFromFloat(0.5),
		Buys: []*types.LiquidityOrder{
			{Reference: types.PeggedReferenceMid, Proportion: 1, Offset: offset},
			{Reference: types.PeggedReferenceBestBid, Proportion: 1, Offset: offset},
		},
		Sells: []*types.LiquidityOrder{
			{Reference: types.PeggedReferenceMid, Proportion: 1, Offset: offset},
			{Reference: types.PeggedReferenceBestAsk, Proportion: 1, Offset: offset},
		},
	}

	require.NoError(t,
		tng.engine.SubmitLiquidityProvision(ctx, lps, party, idgeneration.New(crypto.RandomHash())),
	)
	tng.orderbook.EXPECT().GetOrdersPerParty(party).Return([]*types.Order{}).Times(1)
	tng.orderbook.EXPECT().GetLiquidityOrders(party).Return([]*types.Order{}).AnyTimes()
	partyOrders := tng.engine.CreateInitialOrders(ctx, minLpPrice, maxLpPrice, party, fn)

	for _, o := range partyOrders {
		// set status to active to mock up deployment
		o.Status = types.OrderStatusActive
	}

	require.Len(t, partyOrders, len(lps.Buys)+len(lps.Sells))
	require.Equal(t, types.LiquidityProvisionStatusActive, tng.engine.LiquidityProvisionByPartyID(party).Status)
	tng.orderbook.EXPECT().GetOrdersPerParty(party).Return(partyOrders).MaxTimes(maxTimes)
}

func (tng *testEngine) sortOutLpAmendmentAndOrders(
	t *testing.T,
	ctx context.Context,
	party string,
	commitment int,
	offset *num.Uint,
	minLpPrice, maxLpPrice *num.Uint,
	bestBid, bestAsk num.Decimal,
) {
	t.Helper()
	fn := func(side types.Side, ref types.PeggedReference, offset *num.Uint) (*num.Uint, error) {
		r := bestBid.Copy()
		if ref == types.PeggedReferenceBestAsk {
			r = bestAsk.Copy()
		}
		if ref == types.PeggedReferenceMid {
			r = r.Add(bestAsk).Div(num.DecimalFromInt64(2))
		}
		retPrice, _ := num.UintFromDecimal(r)
		if side == types.SideSell {
			return retPrice.Add(retPrice, offset), nil
		}
		return retPrice.Sub(retPrice, offset), nil
	}
	lpa := &types.LiquidityProvisionAmendment{
		MarketID:         tng.marketID,
		CommitmentAmount: num.NewUint(uint64(commitment)),
		Fee:              num.DecimalFromFloat(0.5),
		Buys: []*types.LiquidityOrder{
			{Reference: types.PeggedReferenceMid, Proportion: 1, Offset: offset},
			{Reference: types.PeggedReferenceBestBid, Proportion: 1, Offset: offset},
		},
		Sells: []*types.LiquidityOrder{
			{Reference: types.PeggedReferenceMid, Proportion: 1, Offset: offset},
			{Reference: types.PeggedReferenceBestAsk, Proportion: 1, Offset: offset},
		},
	}

	// tng.orderbook.EXPECT().GetLiquidityOrders(party).Return([]*types.Order{}).Times(2)
	_, err := tng.engine.AmendLiquidityProvision(ctx, lpa, party, idgeneration.New(crypto.RandomHash()))
	require.NoError(t, err)

	allUpdates, _ := tng.engine.Update(ctx, minLpPrice, maxLpPrice, fn)

	partyNewOrders := []*types.Order{}
	for _, o := range allUpdates {
		if o.Party == party {
			// set status to active to mock up deployment
			o.Status = types.OrderStatusActive
			partyNewOrders = append(partyNewOrders, o)
		}
	}

	require.Len(t, partyNewOrders, len(lpa.Buys)+len(lpa.Sells))
	require.Equal(t, types.LiquidityProvisionStatusActive, tng.engine.LiquidityProvisionByPartyID(party).Status)
	tng.orderbook.EXPECT().GetOrdersPerParty(party).Return(partyNewOrders).AnyTimes()
}

func lScoresSumTo1(t *testing.T, lScores map[string]num.Decimal) {
	t.Helper()

	goTo0 := num.DecimalOne()
	for _, v := range lScores {
		goTo0 = goTo0.Sub(v)
	}

	zeroFp, _ := goTo0.Float64()

	require.InDelta(t, 0, zeroFp, 1e-8)
}
