package liquidity_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	bmock "code.vegaprotocol.io/vega/broker/mocks"
	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/liquidity"
	"code.vegaprotocol.io/vega/liquidity/mocks"
	"code.vegaprotocol.io/vega/logging"
	commandspb "code.vegaprotocol.io/vega/proto/commands/v1"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// eq implements a gomock.Matcher with a better diff output
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

func eq(t *testing.T, x interface{}) eqMatcher { return eqMatcher{t, x} }

type testEngine struct {
	ctrl     *gomock.Controller
	marketID string
	broker   *bmock.MockBroker
	//idGen        *mocks.MockIDGen
	riskModel    *mocks.MockRiskModel
	priceMonitor *mocks.MockPriceMonitor
	engine       *liquidity.Engine
}

func newTestEngine(t *testing.T, now time.Time) *testEngine {
	ctrl := gomock.NewController(t)

	log := logging.NewTestLogger()
	broker := bmock.NewMockBroker(ctrl)
	// idGen := mocks.NewMockIDGen(ctrl)
	idGen := &idGenStub{}
	risk := mocks.NewMockRiskModel(ctrl)
	monitor := mocks.NewMockPriceMonitor(ctrl)
	market := "market-id"
	liquidityConfig := liquidity.NewDefaultConfig()

	risk.EXPECT().GetProjectionHorizon().AnyTimes()

	engine := liquidity.NewEngine(liquidityConfig,
		log, broker, idGen, risk, monitor, market,
	)
	engine.OnChainTimeUpdate(context.Background(), now)

	return &testEngine{
		ctrl:     ctrl,
		marketID: market,
		broker:   broker,
		// idGen:        idGen,
		riskModel:    risk,
		priceMonitor: monitor,
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
			Reference:  types.PeggedReference_PEGGED_REFERENCE_MID,
			Offset:     -1,
			Proportion: 1,
		},
	}
	sellShape := []*types.LiquidityOrder{
		{
			Reference:  types.PeggedReference_PEGGED_REFERENCE_MID,
			Offset:     1,
			Proportion: 1,
		},
	}

	lps1 := &commandspb.LiquidityProvisionSubmission{
		MarketId: tng.marketID, CommitmentAmount: 100, Fee: "0.5",
		Buys: buyShape, Sells: sellShape,
	}

	expected := &types.LiquidityProvision{
		Id:               "some-id-1",
		MarketId:         tng.marketID,
		PartyId:          party,
		Fee:              "0.5",
		CommitmentAmount: lps1.CommitmentAmount,
		CreatedAt:        now.UnixNano(),
		UpdatedAt:        now.UnixNano(),
		Status:           types.LiquidityProvision_STATUS_PENDING,
		Buys: []*types.LiquidityOrderReference{
			{LiquidityOrder: buyShape[0], OrderId: "liquidity-order-1"},
		},

		Sells: []*types.LiquidityOrderReference{
			{LiquidityOrder: sellShape[0], OrderId: "liquidity-order-2"},
		},
	}
	// Create a submission should fire an event
	tng.broker.EXPECT().Send(
		events.NewLiquidityProvisionEvent(ctx, expected),
	).Times(1)
	require.NoError(t,
		tng.engine.SubmitLiquidityProvision(ctx, lps1, party, "some-id-1"),
	)
	got := tng.engine.LiquidityProvisionByPartyID(party)
	require.Equal(t, expected, got)

	expected.Status = types.LiquidityProvision_STATUS_CANCELLED
	tng.broker.EXPECT().Send(
		events.NewLiquidityProvisionEvent(ctx, expected),
	).Times(1)

	_, err := tng.engine.CancelLiquidityProvision(ctx, party)
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
	lps := &commandspb.LiquidityProvisionSubmission{
		MarketId: tng.marketID, CommitmentAmount: 100, Fee: "0.5",
		Buys: []*types.LiquidityOrder{
			{Reference: types.PeggedReference_PEGGED_REFERENCE_MID, Proportion: 20, Offset: -1},
			{Reference: types.PeggedReference_PEGGED_REFERENCE_MID, Proportion: 10, Offset: -2},
		},
		Sells: []*types.LiquidityOrder{
			{Reference: types.PeggedReference_PEGGED_REFERENCE_MID, Proportion: 1, Offset: 1},
		},
	}
	require.NoError(t,
		tng.engine.SubmitLiquidityProvision(ctx, lps, party, "some-id"),
	)

	require.True(t, tng.engine.IsLiquidityProvider(party))

	var markPrice = num.NewUint(10)

	// Now repriceFn works as expected, so initial orders should get created now
	fn := func(order *types.PeggedOrder, _ types.Side) (*num.Uint, *types.PeggedOrder, error) {
		retPrice := markPrice.Clone()
		if order.Offset > 0 {
			return retPrice.Add(retPrice, num.NewUint(uint64(order.Offset))), order, nil
		}
		return retPrice.Sub(retPrice, num.NewUint(uint64(-order.Offset))), order, nil
	}

	// Expectations
	tng.priceMonitor.EXPECT().GetValidPriceRange().Return(num.NewUint(0), num.NewUint(100)).AnyTimes()
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
	lps := &commandspb.LiquidityProvisionSubmission{
		CommitmentAmount: 10,
		MarketId:         tng.marketID,
		Fee:              "0.1",
		Buys: []*types.LiquidityOrder{
			{
				Reference:  types.PeggedReference_PEGGED_REFERENCE_MID,
				Offset:     -1,
				Proportion: 1,
			},
		},
	}

	expected := events.NewLiquidityProvisionEvent(ctx, &types.LiquidityProvision{
		Id:               id,
		MarketId:         tng.marketID,
		PartyId:          party,
		CreatedAt:        now.UnixNano(),
		Status:           types.LiquidityProvision_STATUS_REJECTED,
		Fee:              "0.1",
		CommitmentAmount: 10,
		Sells:            []*types.LiquidityOrderReference{},
		Buys: []*types.LiquidityOrderReference{
			{
				LiquidityOrder: &types.LiquidityOrder{
					Reference:  types.PeggedReference_PEGGED_REFERENCE_MID,
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

	lps = &commandspb.LiquidityProvisionSubmission{
		CommitmentAmount: 10,
		MarketId:         tng.marketID,
		Fee:              "0.2",
		Sells: []*types.LiquidityOrder{
			{
				Reference:  types.PeggedReference_PEGGED_REFERENCE_MID,
				Offset:     -1,
				Proportion: 1,
			},
		},
	}

	expected = events.NewLiquidityProvisionEvent(ctx, &types.LiquidityProvision{
		Id:               id,
		Fee:              "0.2",
		MarketId:         tng.marketID,
		PartyId:          party,
		CreatedAt:        now.UnixNano(),
		CommitmentAmount: 10,
		Status:           types.LiquidityProvision_STATUS_REJECTED,
		Buys:             []*types.LiquidityOrderReference{},
		Sells: []*types.LiquidityOrderReference{
			{
				LiquidityOrder: &types.LiquidityOrder{
					Reference:  types.PeggedReference_PEGGED_REFERENCE_MID,
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

	lps = &commandspb.LiquidityProvisionSubmission{
		Fee:              "0.3",
		CommitmentAmount: 10,
		MarketId:         tng.marketID,
	}

	expected = events.NewLiquidityProvisionEvent(ctx, &types.LiquidityProvision{
		Id:               id,
		MarketId:         tng.marketID,
		Fee:              "0.3",
		PartyId:          party,
		CreatedAt:        now.UnixNano(),
		CommitmentAmount: 10,
		Status:           types.LiquidityProvision_STATUS_REJECTED,
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
	lps := &commandspb.LiquidityProvisionSubmission{
		MarketId: tng.marketID, CommitmentAmount: 100, Fee: "0.5",
		Buys: []*types.LiquidityOrder{
			{Reference: types.PeggedReference_PEGGED_REFERENCE_MID, Proportion: 20, Offset: -1},
			{Reference: types.PeggedReference_PEGGED_REFERENCE_MID, Proportion: 10, Offset: -2},
		},
		Sells: []*types.LiquidityOrder{
			{Reference: types.PeggedReference_PEGGED_REFERENCE_MID, Proportion: 1, Offset: 1},
		},
	}
	require.NoError(t,
		tng.engine.SubmitLiquidityProvision(ctx, lps, party, "some-id"),
	)

	var markPrice = num.NewUint(10)

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
	tng.priceMonitor.EXPECT().GetValidPriceRange().Return(num.NewUint(0), num.NewUint(100)).AnyTimes()
	any := gomock.Any()
	tng.riskModel.EXPECT().ProbabilityOfTrading(
		any, any, any, any, any, any, any,
	).AnyTimes().Return(num.DecimalFromFloat(0.5))

	orders := []*types.Order{
		{Id: "1", PartyId: party, Price: num.NewUint(10), Size: 1, Side: types.Side_SIDE_BUY, Status: types.Order_STATUS_ACTIVE},
		{Id: "2", PartyId: party, Price: num.NewUint(11), Size: 1, Side: types.Side_SIDE_SELL, Status: types.Order_STATUS_ACTIVE},
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
	lp1 := &commandspb.LiquidityProvisionSubmission{
		MarketId: tng.marketID, CommitmentAmount: 100, Fee: "0.5",
		Buys: []*types.LiquidityOrder{
			{Reference: types.PeggedReference_PEGGED_REFERENCE_MID, Proportion: 20, Offset: -1},
			{Reference: types.PeggedReference_PEGGED_REFERENCE_MID, Proportion: 10, Offset: -2},
		},
		Sells: []*types.LiquidityOrder{
			{Reference: types.PeggedReference_PEGGED_REFERENCE_MID, Proportion: 1, Offset: 1},
		},
	}

	require.NoError(t,
		tng.engine.SubmitLiquidityProvision(ctx, lp1, party1, "some-id1"),
	)
	suppliedStake := tng.engine.CalculateSuppliedStake()
	require.Equal(t, lp1.CommitmentAmount, suppliedStake)

	lp2 := &commandspb.LiquidityProvisionSubmission{
		MarketId: tng.marketID, CommitmentAmount: 500, Fee: "0.5",
		Buys: []*types.LiquidityOrder{
			{Reference: types.PeggedReference_PEGGED_REFERENCE_MID, Proportion: 1, Offset: -3},
		},
		Sells: []*types.LiquidityOrder{
			{Reference: types.PeggedReference_PEGGED_REFERENCE_MID, Proportion: 1, Offset: 3},
		},
	}

	require.NoError(t,
		tng.engine.SubmitLiquidityProvision(ctx, lp2, party2, "some-id2"),
	)
	suppliedStake = tng.engine.CalculateSuppliedStake()
	require.Equal(t, lp1.CommitmentAmount+lp2.CommitmentAmount, suppliedStake)

	lp3 := &commandspb.LiquidityProvisionSubmission{
		MarketId: tng.marketID, CommitmentAmount: 962, Fee: "0.5",
		Buys: []*types.LiquidityOrder{
			{Reference: types.PeggedReference_PEGGED_REFERENCE_MID, Proportion: 1, Offset: -5},
		},
		Sells: []*types.LiquidityOrder{
			{Reference: types.PeggedReference_PEGGED_REFERENCE_MID, Proportion: 1, Offset: 1},
			{Reference: types.PeggedReference_PEGGED_REFERENCE_MID, Proportion: 1, Offset: 10},
		},
	}

	require.NoError(t,
		tng.engine.SubmitLiquidityProvision(ctx, lp3, party3, "some-id3"),
	)
	suppliedStake = tng.engine.CalculateSuppliedStake()
	require.Equal(t, lp1.CommitmentAmount+lp2.CommitmentAmount+lp3.CommitmentAmount, suppliedStake)

	_, err := tng.engine.CancelLiquidityProvision(ctx, party1)
	require.NoError(t, err)
	suppliedStake = tng.engine.CalculateSuppliedStake()
	require.Equal(t, lp2.CommitmentAmount+lp3.CommitmentAmount, suppliedStake)
}

type idGenStub struct {
	id uint64
}

func (i *idGenStub) SetID(o *types.Order) {
	i.id++
	o.Id = fmt.Sprintf("liquidity-order-%d", i.id)
}
