// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at https://www.mariadb.com/bsl11.
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

	"code.vegaprotocol.io/vega/idgeneration"
	"code.vegaprotocol.io/vega/libs/crypto"

	"code.vegaprotocol.io/vega/integration/stubs"

	proto "code.vegaprotocol.io/protos/vega"
	commandspb "code.vegaprotocol.io/protos/vega/commands/v1"
	bmocks "code.vegaprotocol.io/vega/broker/mocks"
	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/liquidity"
	"code.vegaprotocol.io/vega/liquidity/mocks"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"
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
		log, tsvc, broker, risk, monitor, asset, market, stateVarEngine, num.NewUint(100000), num.NewUint(100000), num.DecimalFromInt64(1),
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

	determisticId := crypto.RandomHash()
	idGen := idgeneration.New(determisticId)

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

	idgen := idgeneration.New(determisticId)
	require.NoError(t,
		tng.engine.SubmitLiquidityProvision(ctx, lps, party, idgen))
	got := tng.engine.LiquidityProvisionByPartyID(party)
	require.Equal(t, expected, got)

	expected.Status = types.LiquidityProvisionStatusCancelled
	tng.broker.EXPECT().Send(
		events.NewLiquidityProvisionEvent(ctx, expected),
	).Times(1)

	_, err = tng.engine.CancelLiquidityProvision(ctx, party)
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
	fn := func(order *types.PeggedOrder, side types.Side) (*num.Uint, *types.PeggedOrder, error) {
		retPrice := markPrice.Clone()
		if side == types.SideSell {
			return retPrice.Add(retPrice, order.Offset), order, nil
		}
		return retPrice.Sub(retPrice, order.Offset), order, nil
	}

	// Expectations
	tng.priceMonitor.EXPECT().GetValidPriceRange().Return(num.NewWrappedDecimal(num.Zero(), num.DecimalZero()), num.NewWrappedDecimal(num.NewUint(100), num.DecimalFromInt64(100))).AnyTimes()
	any := gomock.Any()
	tng.riskModel.EXPECT().ProbabilityOfTrading(
		any, any, any, any, any, any, any,
	).AnyTimes().Return(num.DecimalFromFloat(0.5))

	newOrders, amendments, err := tng.engine.Update(context.Background(), markPrice.ToDecimal(), markPrice.ToDecimal(), fn, []*types.Order{})
	require.NoError(t, err)
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

	_, err := tng.engine.CancelLiquidityProvision(ctx, party)
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

func TestUpdate(t *testing.T) {
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
	markPriceD := markPrice.ToDecimal()

	fn := func(order *types.PeggedOrder, side types.Side) (*num.Uint, *types.PeggedOrder, error) {
		retPrice := markPrice.Clone()
		if side == types.SideSell {
			retPrice.Add(retPrice, order.Offset)
		} else {
			retPrice.Sub(retPrice, order.Offset)
		}
		return retPrice, order, nil
	}

	// Expectations
	tng.priceMonitor.EXPECT().GetValidPriceRange().Return(num.NewWrappedDecimal(num.Zero(), num.DecimalZero()), num.NewWrappedDecimal(num.NewUint(100), num.DecimalFromInt64(100))).AnyTimes()
	any := gomock.Any()
	tng.riskModel.EXPECT().ProbabilityOfTrading(
		any, any, any, any, any, any, any,
	).AnyTimes().Return(num.DecimalFromFloat(0.5))

	orders := []*types.Order{
		{ID: "1", Party: party, Price: num.NewUint(10), Size: 1, Side: types.SideBuy, Status: types.OrderStatusActive},
		{ID: "2", Party: party, Price: num.NewUint(11), Size: 1, Side: types.SideSell, Status: types.OrderStatusActive},
	}

	creates, err := tng.engine.CreateInitialOrders(ctx, markPriceD, markPriceD, party, orders, fn)
	require.NoError(t, err)
	require.Len(t, creates, 3)

	// Manual order satisfies the commitment, LiqOrders should be removed
	orders[0].Remaining, orders[0].Size = 1000, 1000
	orders[1].Remaining, orders[1].Size = 1000, 1000
	newOrders, toCancels, err := tng.engine.Update(ctx, markPriceD, markPriceD, fn, orders)
	require.NoError(t, err)
	require.Len(t, newOrders, 0)
	require.Len(t, toCancels[0].OrderIDs, 3)
	require.Equal(t, toCancels[0].Party, party)

	newOrders, toCancels, err = tng.engine.Update(ctx, markPriceD, markPriceD, fn, orders)
	require.NoError(t, err)
	require.Len(t, newOrders, 0)
	require.Len(t, toCancels, 0)
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

	_, err = tng.engine.CancelLiquidityProvision(ctx, party1)
	require.NoError(t, err)
	suppliedStake = tng.engine.CalculateSuppliedStake()
	require.Equal(t, num.Sum(lp2.CommitmentAmount, lp3.CommitmentAmount), suppliedStake)
}
