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

	risk.EXPECT().GetProjectionHorizon().AnyTimes()

	engine := liquidity.NewEngine(
		log, broker, idGen, risk, monitor,
	)
	engine.OnChainTimeUpdate(context.Background(), now)

	return &testEngine{
		ctrl, broker, idGen, risk, monitor, engine,
	}
}

func TestSubmissions(t *testing.T) {
	t.Run("CreateUpdateDelete", testSubmissionCRUD)
	t.Run("CancelNonExisting", testCancelNonExistingSubmission)
	t.Run("FailWhenNoShape", testSubmissionFailWhenNoShape)
	t.Run("FailWhenShapeForOneSideOnly", testSubmissionFailWhenOneShapeSepcified)
}

func testSubmissionFailWhenOneShapeSepcified(t *testing.T) {
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
		MarketID: "test", CommitmentAmount: 100, Fee: "0.5",
		Buys: nil, Sells: sellShape,
	}

	lps2 := &types.LiquidityProvisionSubmission{
		MarketID: "test", CommitmentAmount: 100, Fee: "0.5",
		Buys: buyShape, Sells: nil,
	}

	lps3 := &types.LiquidityProvisionSubmission{
		MarketID: "test", CommitmentAmount: 100, Fee: "0.5",
		Buys: buyShape, Sells: sellShape,
	}

	// Create a submission should fire an event
	tng.broker.EXPECT().Send(gomock.Any()).Times(1)

	require.Error(t,
		tng.engine.SubmitLiquidityProvision(ctx, lps1, party, "some-id-1"),
	)

	require.Error(t,
		tng.engine.SubmitLiquidityProvision(ctx, lps2, party, "some-id-2"),
	)

	require.NoError(t,
		tng.engine.SubmitLiquidityProvision(ctx, lps3, party, "some-id-3"),
	)
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
		MarketID: "test", CommitmentAmount: 100, Fee: "0.5",
		Buys: buyShape, Sells: sellShape,
	}

	lps2 := &types.LiquidityProvisionSubmission{
		MarketID: "test", CommitmentAmount: 200, Fee: "0.5",
		Buys: buyShape, Sells: sellShape,
	}

	lps3 := &types.LiquidityProvisionSubmission{
		MarketID: "test", CommitmentAmount: 000, Fee: "0.5",
		Buys: buyShape, Sells: sellShape,
	}

	expected := &types.LiquidityProvision{
		Id:               "some-id-1",
		MarketID:         "test",
		PartyID:          party,
		CommitmentAmount: lps1.CommitmentAmount,
		CreatedAt:        now.UnixNano(),
		UpdatedAt:        now.UnixNano(),
		Status:           types.LiquidityProvision_LIQUIDITY_PROVISION_STATUS_ACTIVE,
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
	expected.Status = types.LiquidityProvision_LIQUIDITY_PROVISION_STATUS_CANCELLED
	tng.broker.EXPECT().Send(
		events.NewLiquidityProvisionEvent(ctx, expected),
	).Times(1)

	require.NoError(t,
		tng.engine.SubmitLiquidityProvision(ctx, lps3, party, "some-id-3"),
	)
	require.Nil(t, tng.engine.LiquidityProvisionByPartyID(party),
		"Party '%s' should not be a LiquidityProvider after Committing 0 amount", party)
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
		CommitmentAmount: 0,
		Buys: []*types.LiquidityOrder{
			{
				Reference:  types.PeggedReference_PEGGED_REFERENCE_MID,
				Offset:     -1,
				Proportion: 1,
			},
		},
	}

	err := tng.engine.SubmitLiquidityProvision(ctx,
		lps, party, "some-id")
	require.Error(t, err)
}

func testSubmissionFailWhenNoShape(t *testing.T) {
	var (
		party = "party-1"
		ctx   = context.Background()
		now   = time.Now()
		tng   = newTestEngine(t, now)
	)
	defer tng.ctrl.Finish()

	// Expectations
	lps := &types.LiquidityProvisionSubmission{
		CommitmentAmount: 10,
	}

	require.Error(t,
		tng.engine.SubmitLiquidityProvision(ctx, lps, party, "some-id"),
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

	// Send a submission to create the shape
	lps := &types.LiquidityProvisionSubmission{
		MarketID: "test", CommitmentAmount: 100, Fee: "0.5",
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
		{Id: "1", PartyID: party, Price: 10, Size: 1, Side: types.Side_SIDE_BUY, Status: types.Order_STATUS_ACTIVE},
		{Id: "2", PartyID: party, Price: 11, Size: 1, Side: types.Side_SIDE_SELL, Status: types.Order_STATUS_ACTIVE},
	}

	creates, updates, err := tng.engine.Update(markPrice, fn, orders)
	require.NoError(t, err)
	require.Len(t, creates, 3)
	require.Len(t, updates, 0)

	initialSizes := make([]uint64, 0, len(creates))
	for _, c := range creates {
		initialSizes = append(initialSizes, c.Size)
	}

	// Manual orders provide more liquidity, LiqOrders should decrease
	orders[0].Remaining, orders[0].Size = 10, 10
	orders[1].Remaining, orders[1].Size = 10, 10

	creates, updates, err = tng.engine.Update(markPrice, fn, orders)
	require.NoError(t, err)
	require.Len(t, creates, 0)
	require.Len(t, updates, 3)
	for i, order := range updates {
		assert.Greater(t, order.Size, uint64(0))
		require.Less(t, order.Size, initialSizes[i])
	}

	// Manual order satisfies the commitment, LiqOrders should be removed
	orders[0].Remaining, orders[0].Size = 1000, 1000
	orders[1].Remaining, orders[1].Size = 1000, 1000
	creates, updates, err = tng.engine.Update(markPrice, fn, orders)
	require.NoError(t, err)
	require.Len(t, creates, 0)
	require.Len(t, updates, 3)
	for _, order := range updates {
		assert.EqualValues(t, 0, order.Size)
	}

	creates, updates, err = tng.engine.Update(markPrice, fn, orders)
	require.NoError(t, err)
	require.Len(t, creates, 0)
	require.Len(t, updates, 0)
}

// TODO (WG): Write a test where new limit orders should reduce the liquidity deployed with shapes.
// TODO (WG): Cover the case when pegs outside valid range.
