package liquidity_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/integration/stubs"

	proto "code.vegaprotocol.io/protos/vega"
	commandspb "code.vegaprotocol.io/protos/vega/commands/v1"
	bmock "code.vegaprotocol.io/vega/broker/mocks"
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
	broker       *bmock.MockBroker
	riskModel    *mocks.MockRiskModel
	priceMonitor *mocks.MockPriceMonitor
	engine       *liquidity.SnapshotEngine
	idGen        *idGenStub
}

func newTestEngineWithIDGen(t *testing.T, now time.Time, idGen *idGenStub) *testEngine {
	t.Helper()
	ctrl := gomock.NewController(t)

	log := logging.NewTestLogger()
	broker := bmock.NewMockBroker(ctrl)
	risk := mocks.NewMockRiskModel(ctrl)
	monitor := mocks.NewMockPriceMonitor(ctrl)
	market := "market-id"
	asset := "asset-id"
	liquidityConfig := liquidity.NewDefaultConfig()
	stateVarEngine := stubs.NewStateVar()
	risk.EXPECT().GetProjectionHorizon().AnyTimes()

	engine := liquidity.NewSnapshotEngine(liquidityConfig,
		log, broker, idGen, risk, monitor, asset, market, stateVarEngine,
	)
	engine.OnChainTimeUpdate(context.Background(), now)

	return &testEngine{
		ctrl:         ctrl,
		marketID:     market,
		broker:       broker,
		idGen:        idGen,
		riskModel:    risk,
		priceMonitor: monitor,
		engine:       engine,
	}
}

func newTestEngine(t *testing.T, now time.Time) *testEngine {
	t.Helper()
	idGen := &idGenStub{}
	return newTestEngineWithIDGen(t, now, idGen)
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
			Offset:     -1,
			Proportion: 1,
		},
	}
	sellShape := []*types.LiquidityOrder{
		{
			Reference:  types.PeggedReferenceMid,
			Offset:     1,
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

	expected := &types.LiquidityProvision{
		ID:               "some-id-1",
		MarketID:         tng.marketID,
		Party:            party,
		Fee:              num.DecimalFromFloat(0.5),
		CommitmentAmount: lps.CommitmentAmount.Clone(),
		CreatedAt:        now.UnixNano(),
		UpdatedAt:        now.UnixNano(),
		Status:           types.LiquidityProvisionStatusPending,
		Buys: []*types.LiquidityOrderReference{
			{LiquidityOrder: buyShape[0], OrderID: "liquidity-order-1"},
		},

		Sells: []*types.LiquidityOrderReference{
			{LiquidityOrder: sellShape[0], OrderID: "liquidity-order-2"},
		},
	}
	// Create a submission should fire an event
	tng.broker.EXPECT().Send(
		events.NewLiquidityProvisionEvent(ctx, expected),
	).Times(1)
	require.NoError(t,
		tng.engine.SubmitLiquidityProvision(ctx, lps, party, "some-id-1"),
	)
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
			{Reference: types.PeggedReferenceMid, Proportion: 20, Offset: -1},
			{Reference: types.PeggedReferenceMid, Proportion: 10, Offset: -2},
		},
		Sells: []*proto.LiquidityOrder{
			{Reference: types.PeggedReferenceMid, Proportion: 1, Offset: 1},
		},
	}
	lps, err := types.LiquidityProvisionSubmissionFromProto(lpspb)
	require.NoError(t, err)

	require.NoError(t,
		tng.engine.SubmitLiquidityProvision(ctx, lps, party, "some-id"),
	)

	require.True(t, tng.engine.IsLiquidityProvider(party))

	markPrice := num.NewUint(10)

	// Now repriceFn works as expected, so initial orders should get created now
	fn := func(order *types.PeggedOrder, _ types.Side) (*num.Uint, *types.PeggedOrder, error) {
		retPrice := markPrice.Clone()
		if order.Offset > 0 {
			return retPrice.Add(retPrice, num.NewUint(uint64(order.Offset))), order, nil
		}
		return retPrice.Sub(retPrice, num.NewUint(uint64(-order.Offset))), order, nil
	}

	// Expectations
	tng.priceMonitor.EXPECT().GetValidPriceRange().Return(num.NewWrappedDecimal(num.Zero(), num.DecimalZero()), num.NewWrappedDecimal(num.NewUint(100), num.DecimalFromInt64(100))).AnyTimes()
	any := gomock.Any()
	tng.riskModel.EXPECT().ProbabilityOfTrading(
		any, any, any, any, any, any, any,
	).AnyTimes().Return(num.DecimalFromFloat(0.5))

	newOrders, amendments, err := tng.engine.Update(context.Background(), markPrice, markPrice, fn, []*types.Order{})
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
		id    = "some-id"
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
				Offset:     -1,
				Proportion: 1,
			},
		},
	}
	lps, err := types.LiquidityProvisionSubmissionFromProto(lpspb)
	require.NoError(t, err)

	expected := events.NewLiquidityProvisionEvent(ctx, &types.LiquidityProvision{
		ID:               id,
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
					Offset:     -1,
					Proportion: 1,
				},
			},
		},
	})

	tng.broker.EXPECT().Send(eq(t, expected)).Times(1)

	require.Error(t,
		tng.engine.SubmitLiquidityProvision(ctx, lps, party, id),
	)

	lpspb = &commandspb.LiquidityProvisionSubmission{
		CommitmentAmount: "10",
		MarketId:         tng.marketID,
		Fee:              "0.2",
		Sells: []*proto.LiquidityOrder{
			{
				Reference:  types.PeggedReferenceMid,
				Offset:     -1,
				Proportion: 1,
			},
		},
	}
	lps, err = types.LiquidityProvisionSubmissionFromProto(lpspb)
	require.NoError(t, err)

	expected = events.NewLiquidityProvisionEvent(ctx, &types.LiquidityProvision{
		ID:               id,
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
					Offset:     -1,
					Proportion: 1,
				},
			},
		},
	})

	tng.broker.EXPECT().Send(eq(t, expected)).Times(1)

	require.Error(t,
		tng.engine.SubmitLiquidityProvision(ctx, lps, party, id),
	)

	lpspb = &commandspb.LiquidityProvisionSubmission{
		Fee:              "0.3",
		CommitmentAmount: "10",
		MarketId:         tng.marketID,
	}
	lps, _ = types.LiquidityProvisionSubmissionFromProto(lpspb)

	expected = events.NewLiquidityProvisionEvent(ctx, &types.LiquidityProvision{
		ID:               id,
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

	require.Error(t,
		tng.engine.SubmitLiquidityProvision(ctx, lps, party, id),
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
			{Reference: types.PeggedReferenceMid, Proportion: 20, Offset: -1},
			{Reference: types.PeggedReferenceMid, Proportion: 10, Offset: -2},
		},
		Sells: []*proto.LiquidityOrder{
			{Reference: types.PeggedReferenceMid, Proportion: 1, Offset: 1},
		},
	}
	lps, err := types.LiquidityProvisionSubmissionFromProto(lpspb)
	require.NoError(t, err)

	require.NoError(t,
		tng.engine.SubmitLiquidityProvision(ctx, lps, party, "some-id"),
	)

	markPrice := num.NewUint(10)

	fn := func(order *types.PeggedOrder, _ types.Side) (*num.Uint, *types.PeggedOrder, error) {
		retPrice := markPrice.Clone()
		if order.Offset > 0 {
			retPrice.Add(retPrice, num.NewUint(uint64(order.Offset)))
		} else {
			retPrice.Sub(retPrice, num.NewUint(uint64(-order.Offset)))
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

	creates, err := tng.engine.CreateInitialOrders(ctx, markPrice, markPrice, party, orders, fn)
	require.NoError(t, err)
	require.Len(t, creates, 3)

	// Manual order satisfies the commitment, LiqOrders should be removed
	orders[0].Remaining, orders[0].Size = 1000, 1000
	orders[1].Remaining, orders[1].Size = 1000, 1000
	newOrders, toCancels, err := tng.engine.Update(ctx, markPrice, markPrice, fn, orders)
	require.NoError(t, err)
	require.Len(t, newOrders, 0)
	require.Len(t, toCancels[0].OrderIDs, 3)
	require.Equal(t, toCancels[0].Party, party)

	newOrders, toCancels, err = tng.engine.Update(ctx, markPrice, markPrice, fn, orders)
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
			{Reference: types.PeggedReferenceMid, Proportion: 20, Offset: -1},
			{Reference: types.PeggedReferenceMid, Proportion: 10, Offset: -2},
		},
		Sells: []*proto.LiquidityOrder{
			{Reference: types.PeggedReferenceMid, Proportion: 1, Offset: 1},
		},
	}
	lp1, err := types.LiquidityProvisionSubmissionFromProto(lp1pb)
	require.NoError(t, err)

	require.NoError(t,
		tng.engine.SubmitLiquidityProvision(ctx, lp1, party1, "some-id1"),
	)
	suppliedStake := tng.engine.CalculateSuppliedStake()
	require.Equal(t, lp1.CommitmentAmount, suppliedStake)

	lp2pb := &commandspb.LiquidityProvisionSubmission{
		MarketId: tng.marketID, CommitmentAmount: "500", Fee: "0.5",
		Buys: []*proto.LiquidityOrder{
			{Reference: types.PeggedReferenceMid, Proportion: 1, Offset: -3},
		},
		Sells: []*proto.LiquidityOrder{
			{Reference: types.PeggedReferenceMid, Proportion: 1, Offset: 3},
		},
	}
	lp2, err := types.LiquidityProvisionSubmissionFromProto(lp2pb)
	require.NoError(t, err)

	require.NoError(t,
		tng.engine.SubmitLiquidityProvision(ctx, lp2, party2, "some-id2"),
	)
	suppliedStake = tng.engine.CalculateSuppliedStake()
	require.Equal(t, num.Sum(lp1.CommitmentAmount, lp2.CommitmentAmount), suppliedStake)

	lp3pb := &commandspb.LiquidityProvisionSubmission{
		MarketId: tng.marketID, CommitmentAmount: "962", Fee: "0.5",
		Buys: []*proto.LiquidityOrder{
			{Reference: types.PeggedReferenceMid, Proportion: 1, Offset: -5},
		},
		Sells: []*proto.LiquidityOrder{
			{Reference: types.PeggedReferenceMid, Proportion: 1, Offset: 1},
			{Reference: types.PeggedReferenceMid, Proportion: 1, Offset: 10},
		},
	}
	lp3, err := types.LiquidityProvisionSubmissionFromProto(lp3pb)
	require.NoError(t, err)

	require.NoError(t,
		tng.engine.SubmitLiquidityProvision(ctx, lp3, party3, "some-id3"),
	)
	suppliedStake = tng.engine.CalculateSuppliedStake()
	require.Equal(t, num.Sum(lp1.CommitmentAmount, lp2.CommitmentAmount, lp3.CommitmentAmount), suppliedStake)

	_, err = tng.engine.CancelLiquidityProvision(ctx, party1)
	require.NoError(t, err)
	suppliedStake = tng.engine.CalculateSuppliedStake()
	require.Equal(t, num.Sum(lp2.CommitmentAmount, lp3.CommitmentAmount), suppliedStake)
}

type idGenStub struct {
	id uint64
}

func (i *idGenStub) SetID(o *types.Order) {
	i.id++
	o.ID = fmt.Sprintf("liquidity-order-%d", i.id)
}
