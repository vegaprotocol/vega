package liquidity_test

import (
	"context"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/liquidity"
	"code.vegaprotocol.io/vega/liquidity/mocks"
	"code.vegaprotocol.io/vega/logging"
	types "code.vegaprotocol.io/vega/proto"
	"github.com/golang/mock/gomock"
	uuid "github.com/satori/go.uuid"
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
	ctrl         *gomock.Controller
	marketID     string
	broker       *mocks.MockBroker
	idGen        *mocks.MockIDGen
	riskModel    *mocks.MockRiskModel
	priceMonitor *mocks.MockPriceMonitor
	engine       *liquidity.Engine
}

func newTestEngine(t *testing.T, now time.Time) *testEngine {
	ctrl := gomock.NewController(t)

	log := logging.NewTestLogger()
	broker := mocks.NewMockBroker(ctrl)
	idGen := mocks.NewMockIDGen(ctrl)
	risk := mocks.NewMockRiskModel(ctrl)
	monitor := mocks.NewMockPriceMonitor(ctrl)
	market := "market-id"

	risk.EXPECT().GetProjectionHorizon().AnyTimes()

	engine := liquidity.NewEngine(
		log, broker, idGen, risk, monitor, market,
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

	lps1 := &types.LiquidityProvisionSubmission{
		MarketId: tng.marketID, CommitmentAmount: 100, Fee: "0.5",
		Buys: buyShape, Sells: sellShape,
	}

	lps2 := &types.LiquidityProvisionSubmission{
		MarketId: tng.marketID, CommitmentAmount: 200, Fee: "0.5",
		Buys: buyShape, Sells: sellShape,
	}

	lps3 := &types.LiquidityProvisionSubmission{
		MarketId: tng.marketID, CommitmentAmount: 000, Fee: "0.5",
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
		Status:           types.LiquidityProvision_STATUS_UNDEPLOYED,
		Buys: []*types.LiquidityOrderReference{
			{LiquidityOrder: buyShape[0]},
		},

		Sells: []*types.LiquidityOrderReference{
			{LiquidityOrder: sellShape[0]},
		},
	}

	// Create a submission should fire an event
	tng.broker.EXPECT().Send(
		eq(t, events.NewLiquidityProvisionEvent(ctx, expected)),
	).Times(1)
	require.NoError(t,
		tng.engine.SubmitLiquidityProvision(ctx, lps1, party, "some-id-1"),
	)
	got := tng.engine.LiquidityProvisionByPartyID(party)
	require.Equal(t, expected, got)

	// Submitting for the same market/party should update the previous
	now = now.Add(1 * time.Hour)
	expected.UpdatedAt = now.UnixNano()
	expected.CommitmentAmount = lps2.CommitmentAmount
	tng.engine.OnChainTimeUpdate(ctx, now)

	tng.broker.EXPECT().Send(
		events.NewLiquidityProvisionEvent(ctx, expected),
	).Times(1)
	require.NoError(t,
		tng.engine.SubmitLiquidityProvision(ctx, lps2, party, "some-id-2"),
	)

	got = tng.engine.LiquidityProvisionByPartyID(party)
	require.NotNil(t, got)
	require.Equal(t, lps2.CommitmentAmount, got.CommitmentAmount)
	require.True(t, got.UpdatedAt > got.CreatedAt)

	// Submit with 0 CommitmentAmount amount should remove the LP and CANCEL it
	// via event
	expected.CommitmentAmount = 0
	expected.Status = types.LiquidityProvision_STATUS_CANCELLED
	tng.broker.EXPECT().Send(
		events.NewLiquidityProvisionEvent(ctx, expected),
	).Times(1)

	require.NoError(t,
		tng.engine.SubmitLiquidityProvision(ctx, lps3, party, "some-id-3"),
	)
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
	lps := &types.LiquidityProvisionSubmission{
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

	var (
		markPrice = uint64(10)
	)

	// Now repriceFn works as expected, so initial orders should get created now
	fn := func(order *types.PeggedOrder) (uint64, error) {
		return markPrice + uint64(order.Offset), nil
	}

	// Expectations
	tng.priceMonitor.EXPECT().GetValidPriceRange().Return(0.0, 100.0).AnyTimes()
	any := gomock.Any()
	tng.riskModel.EXPECT().ProbabilityOfTrading(
		any, any, any, any, any, any, any,
	).AnyTimes().Return(0.5)
	tng.idGen.EXPECT().SetID(gomock.Any()).Do(func(order *types.Order) {
		order.Id = uuid.NewV4().String()
	}).AnyTimes()

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

	lps := &types.LiquidityProvisionSubmission{
		MarketId:         tng.marketID,
		CommitmentAmount: 0,
		Buys: []*types.LiquidityOrder{
			{
				Reference:  types.PeggedReference_PEGGED_REFERENCE_MID,
				Offset:     -1,
				Proportion: 1,
			},
		},
		Sells: []*types.LiquidityOrder{
			{
				Reference:  types.PeggedReference_PEGGED_REFERENCE_MID,
				Offset:     1,
				Proportion: 1,
			},
		},
	}
	expected := events.NewLiquidityProvisionEvent(ctx, &types.LiquidityProvision{
		Id:        "some-id",
		MarketId:  tng.marketID,
		PartyId:   party,
		CreatedAt: now.UnixNano(),
		Status:    types.LiquidityProvision_STATUS_REJECTED,
	})

	tng.broker.EXPECT().Send(eq(t, expected)).Times(1)
	err := tng.engine.SubmitLiquidityProvision(ctx,
		lps, party, "some-id")
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
	lps := &types.LiquidityProvisionSubmission{
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

	lps = &types.LiquidityProvisionSubmission{
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

	lps = &types.LiquidityProvisionSubmission{
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
	lps := &types.LiquidityProvisionSubmission{
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

	var (
		markPrice = uint64(10)
	)

	fn := func(order *types.PeggedOrder) (uint64, error) {
		return markPrice + uint64(order.Offset), nil
	}

	// Expectations
	tng.priceMonitor.EXPECT().GetValidPriceRange().Return(0.0, 100.0).AnyTimes()
	any := gomock.Any()
	tng.riskModel.EXPECT().ProbabilityOfTrading(
		any, any, any, any, any, any, any,
	).AnyTimes().Return(0.5)
	tng.idGen.EXPECT().SetID(gomock.Any()).Do(func(order *types.Order) {
		order.Id = uuid.NewV4().String()
	}).AnyTimes()

	orders := []*types.Order{
		{Id: "1", PartyId: party, Price: 10, Size: 1, Side: types.Side_SIDE_BUY, Status: types.Order_STATUS_ACTIVE},
		{Id: "2", PartyId: party, Price: 11, Size: 1, Side: types.Side_SIDE_SELL, Status: types.Order_STATUS_ACTIVE},
	}

	creates, _, err := tng.engine.CreateInitialOrders(markPrice, markPrice, party, orders, fn)
	require.NoError(t, err)
	require.Len(t, creates, 3)

	// Manual order satisfies the commitment, LiqOrders should be removed
	orders[0].Remaining, orders[0].Size = 1000, 1000
	orders[1].Remaining, orders[1].Size = 1000, 1000
	newOrders, amendments, err := tng.engine.Update(context.Background(), markPrice, markPrice, fn, orders)
	require.NoError(t, err)
	require.Len(t, newOrders, 0)
	require.Len(t, amendments, 3)
	for i, amend := range amendments {
		assert.Zero(t, creates[i].Size+uint64(amend.SizeDelta),
			"Size should be cancelled (== 0)  by the amendment",
		)
	}

	newOrders, amendments, err = tng.engine.Update(context.Background(), markPrice, markPrice, fn, orders)
	require.NoError(t, err)
	require.Len(t, newOrders, 0)
	require.Len(t, amendments, 0)
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
	lp1 := &types.LiquidityProvisionSubmission{
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

	lp2 := &types.LiquidityProvisionSubmission{
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

	lp3 := &types.LiquidityProvisionSubmission{
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

	lp1.CommitmentAmount -= 100
	require.NoError(t,
		tng.engine.SubmitLiquidityProvision(ctx, lp1, party1, "some-id1"),
	)
	suppliedStake = tng.engine.CalculateSuppliedStake()
	require.Equal(t, lp1.CommitmentAmount+lp2.CommitmentAmount+lp3.CommitmentAmount, suppliedStake)
}
