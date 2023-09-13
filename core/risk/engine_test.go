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

package risk_test

import (
	"context"
	"fmt"
	"math"
	"sort"
	"testing"
	"time"

	bmocks "code.vegaprotocol.io/vega/core/broker/mocks"
	"code.vegaprotocol.io/vega/core/config"
	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/matching"
	"code.vegaprotocol.io/vega/core/risk"
	"code.vegaprotocol.io/vega/core/risk/mocks"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/core/types/statevar"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/logging"
	proto "code.vegaprotocol.io/vega/protos/vega"
	"github.com/stretchr/testify/require"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

var DefaultSlippageFactor = num.DecimalFromFloat(0.1)

func peggedOrderCounterForTest(int64) {}

type MLEvent interface {
	events.Event
	MarginLevels() proto.MarginLevels
}

type testEngine struct {
	*risk.Engine
	ctrl      *gomock.Controller
	model     *mocks.MockModel
	orderbook *mocks.MockOrderbook
	tsvc      *mocks.MockTimeService
	broker    *bmocks.MockBroker
	as        *mocks.MockAuctionState
}

// implements the events.Margin interface.
type testMargin struct {
	party           string
	size            int64
	buy             int64
	sell            int64
	price           uint64
	transfer        *types.Transfer
	asset           string
	margin          uint64
	general         uint64
	market          string
	buySumProduct   uint64
	sellSumProduct  uint64
	marginShortFall uint64
}

var (
	riskFactors = types.RiskFactor{
		Short: num.DecimalFromFloat(.20),
		Long:  num.DecimalFromFloat(.25),
	}

	markPrice = num.NewUint(100)
)

func TestUpdateMargins(t *testing.T) {
	t.Run("test time update", testMarginLevelsTS)
	t.Run("Top up margin test", testMarginTopup)
	t.Run("Noop margin test", testMarginNoop)
	t.Run("Margin too high (overflow)", testMarginOverflow)
	t.Run("Margin too high (overflow) - auction ending", testMarginOverflowAuctionEnd)
	t.Run("Update Margin with orders in book", testMarginWithOrderInBook)
	t.Run("Update Margin with orders in book 2", testMarginWithOrderInBook2)
	t.Run("Update Margin with orders in book after parameters update", testMarginWithOrderInBookAfterParamsUpdate)
	t.Run("Top up fail on new order", testMarginTopupOnOrderFailInsufficientFunds)
	t.Run("Margin not released in auction", testMarginNotReleasedInAuction)
	t.Run("Initial margin requirement must be met", testInitialMarginRequirement)
}

func testMarginLevelsTS(t *testing.T) {
	eng := getTestEngine(t)

	ctx, cfunc := context.WithCancel(context.Background())
	defer cfunc()
	evt := testMargin{
		party:   "party1",
		size:    1,
		price:   1000,
		asset:   "ETH",
		margin:  10,     // required margin will be > 30 so ensure we don't have enough
		general: 100000, // plenty of balance for the transfer anyway
		market:  "ETH/DEC19",
	}

	now := time.Now()
	eng.tsvc.EXPECT().GetTimeNow().DoAndReturn(
		func() time.Time {
			return now
		}).Times(1)

	eng.orderbook.EXPECT().GetCloseoutPrice(gomock.Any(), gomock.Any()).Times(1).
		DoAndReturn(func(volume uint64, side types.Side) (*num.Uint, error) {
			return markPrice.Clone(), nil
		})

	eng.as.EXPECT().InAuction().AnyTimes().Return(false)
	eng.broker.EXPECT().SendBatch(gomock.Any()).Times(1).Do(func(e []events.Event) {
		mle, ok := e[0].(MLEvent)
		assert.True(t, ok)
		ml := mle.MarginLevels()
		assert.Equal(t, now.UnixNano(), ml.Timestamp)
	})

	evts := []events.Margin{evt}
	resp := eng.UpdateMarginsOnSettlement(ctx, evts, markPrice, num.DecimalZero())
	assert.Equal(t, 1, len(resp))
	// ensure we get the correct transfer request back, correct amount etc...
	trans := resp[0].Transfer()
	assert.EqualValues(t, 20, trans.Amount.Amount.Uint64())
	// assert.EqualValues(t, 44, trans.Amount.Amount.Uint64())
	// min = 15 so we go back to maintenance level
	assert.EqualValues(t, 15, trans.MinAmount.Uint64())
	// assert.EqualValues(t, 35, trans.MinAmount.Uint64())
	assert.Equal(t, types.TransferTypeMarginLow, trans.Type)
}

func testMarginTopup(t *testing.T) {
	eng := getTestEngine(t)

	ctx, cfunc := context.WithCancel(context.Background())
	defer cfunc()
	evt := testMargin{
		party:   "party1",
		size:    1,
		price:   1000,
		asset:   "ETH",
		margin:  10,     // required margin will be > 30 so ensure we don't have enough
		general: 100000, // plenty of balance for the transfer anyway
		market:  "ETH/DEC19",
	}
	eng.tsvc.EXPECT().GetTimeNow().Times(1)
	eng.broker.EXPECT().SendBatch(gomock.Any()).AnyTimes()
	eng.as.EXPECT().InAuction().AnyTimes().Return(false)
	eng.orderbook.EXPECT().GetCloseoutPrice(gomock.Any(), gomock.Any()).Times(1).
		DoAndReturn(func(volume uint64, side types.Side) (*num.Uint, error) {
			return markPrice.Clone(), nil
		})
	evts := []events.Margin{evt}
	resp := eng.UpdateMarginsOnSettlement(ctx, evts, markPrice, num.DecimalZero())
	assert.Equal(t, 1, len(resp))
	// ensure we get the correct transfer request back, correct amount etc...
	trans := resp[0].Transfer()
	assert.EqualValues(t, 20, trans.Amount.Amount.Uint64())
	// assert.EqualValues(t, 44, trans.Amount.Amount.Uint64())
	// min = 15 so we go back to maintenance level
	assert.EqualValues(t, 15, trans.MinAmount.Uint64())
	// assert.EqualValues(t, 35, trans.MinAmount.Uint64())
	assert.Equal(t, types.TransferTypeMarginLow, trans.Type)
}

func TestMarginTopupPerpetual(t *testing.T) {
	eng := getTestEngine(t)

	ctx, cfunc := context.WithCancel(context.Background())
	defer cfunc()
	evt := testMargin{
		party:   "party1",
		size:    1,
		price:   1000,
		asset:   "ETH",
		margin:  10,     // required margin will be > 30 so ensure we don't have enough
		general: 100000, // plenty of balance for the transfer anyway
		market:  "ETH/DEC19",
	}

	// lets pretend the perpetual margin factor is 0.5 and the funding payement was 10, 5
	inc := num.DecimalFromFloat(0.5).Mul(num.DecimalFromInt64(10))

	eng.tsvc.EXPECT().GetTimeNow().Times(2)
	eng.broker.EXPECT().SendBatch(gomock.Any()).AnyTimes()
	eng.as.EXPECT().InAuction().AnyTimes().Return(false)
	eng.orderbook.EXPECT().GetCloseoutPrice(gomock.Any(), gomock.Any()).Times(2).
		DoAndReturn(func(volume uint64, side types.Side) (*num.Uint, error) {
			return markPrice.Clone(), nil
		})
	evts := []events.Margin{evt}
	resp := eng.UpdateMarginsOnSettlement(ctx, evts, markPrice, inc)
	assert.Equal(t, 1, len(resp))

	mm := resp[0].MarginLevels().MaintenanceMargin
	assert.Equal(t, "30", mm.String())

	// now do it again with the funding payment negated, the margin should be as if we were not a perp
	// and 5 less
	resp = eng.UpdateMarginsOnSettlement(ctx, evts, markPrice, inc.Neg())
	assert.Equal(t, 1, len(resp))

	mm = resp[0].MarginLevels().MaintenanceMargin
	assert.Equal(t, "25", mm.String())
}

func testMarginNotReleasedInAuction(t *testing.T) {
	eng := getTestEngine(t)

	ctx, cfunc := context.WithCancel(context.Background())
	defer cfunc()
	evt := testMargin{
		party:   "party1",
		size:    1,
		price:   1000,
		asset:   "ETH",
		margin:  70, // relese level is 35 so we need more than that
		general: 100000,
		market:  "ETH/DEC19",
	}
	eng.tsvc.EXPECT().GetTimeNow().Times(1)
	eng.broker.EXPECT().SendBatch(gomock.Any()).AnyTimes()
	eng.as.EXPECT().InAuction().AnyTimes().Return(true)
	eng.as.EXPECT().CanLeave().AnyTimes().Return(false)
	evts := []events.Margin{evt}
	resp := eng.UpdateMarginsOnSettlement(ctx, evts, markPrice, num.DecimalZero())
	assert.Equal(t, 0, len(resp))
}

func testMarginTopupOnOrderFailInsufficientFunds(t *testing.T) {
	eng := getTestEngine(t)

	_, cfunc := context.WithCancel(context.Background())
	defer cfunc()
	evt := testMargin{
		party:   "party1",
		size:    1,
		price:   1000,
		asset:   "ETH",
		margin:  10, // maring and general combined are not enough to get a sufficient margin
		general: 10,
		market:  "ETH/DEC19",
	}
	eng.tsvc.EXPECT().GetTimeNow().Times(1)
	eng.as.EXPECT().InAuction().AnyTimes().Return(false)
	eng.orderbook.EXPECT().GetCloseoutPrice(gomock.Any(), gomock.Any()).Times(1).
		DoAndReturn(func(volume uint64, side types.Side) (*num.Uint, error) {
			return markPrice.Clone(), nil
		})
	riskevt, _, err := eng.UpdateMarginOnNewOrder(context.Background(), evt, markPrice, num.DecimalZero())
	assert.Nil(t, riskevt)
	assert.NotNil(t, err)
	assert.Error(t, err, risk.ErrInsufficientFundsForInitialMargin.Error())
}

func testMarginNoop(t *testing.T) {
	eng := getTestEngine(t)

	eng.broker.EXPECT().SendBatch(gomock.Any()).AnyTimes()
	ctx, cfunc := context.WithCancel(context.Background())
	defer cfunc()
	evt := testMargin{
		party:   "party1",
		size:    1,
		price:   1000,
		asset:   "ETH",
		margin:  30,     // more than enough margin to cover the position, not enough to trigger transfer to general
		general: 100000, // plenty of balance for the transfer anyway
		market:  "ETH/DEC19",
	}
	eng.tsvc.EXPECT().GetTimeNow().Times(1)
	eng.as.EXPECT().InAuction().AnyTimes().Return(false)
	eng.orderbook.EXPECT().GetCloseoutPrice(gomock.Any(), gomock.Any()).Times(1).
		DoAndReturn(func(volume uint64, side types.Side) (*num.Uint, error) {
			return markPrice.Clone(), nil
		})

	evts := []events.Margin{evt}
	resp := eng.UpdateMarginsOnSettlement(ctx, evts, markPrice, num.DecimalZero())
	assert.Equal(t, 0, len(resp))
	// assert.Equal(t, 1, len(resp))
}

func testMarginOverflow(t *testing.T) {
	eng := getTestEngine(t)

	eng.broker.EXPECT().SendBatch(gomock.Any()).AnyTimes()
	ctx, cfunc := context.WithCancel(context.Background())
	defer cfunc()
	evt := testMargin{
		party:   "party1",
		size:    1,
		price:   1000,
		asset:   "ETH",
		margin:  500,    // required margin will be > 35 (release), so ensure we don't have enough
		general: 100000, // plenty of balance for the transfer anyway
		market:  "ETH/DEC19",
	}
	eng.tsvc.EXPECT().GetTimeNow().Times(1)
	eng.as.EXPECT().InAuction().Times(2).Return(false)
	eng.orderbook.EXPECT().GetCloseoutPrice(gomock.Any(), gomock.Any()).Times(1).
		DoAndReturn(func(volume uint64, side types.Side) (*num.Uint, error) {
			return markPrice.Clone(), nil
		})
	evts := []events.Margin{evt}
	resp := eng.UpdateMarginsOnSettlement(ctx, evts, markPrice, num.DecimalZero())
	assert.Equal(t, 1, len(resp))

	// ensure we get the correct transfer request back, correct amount etc...
	trans := resp[0].Transfer()
	assert.EqualValues(t, 470, trans.Amount.Amount.Uint64())
	// assert.EqualValues(t, 446, trans.Amount.Amount.Uint64())
	// assert.Equal(t, riskMinamount-int64(evt.margin), trans.Amount.MinAmount)
	assert.Equal(t, types.TransferTypeMarginHigh, trans.Type)
}

func testMarginOverflowAuctionEnd(t *testing.T) {
	eng := getTestEngine(t)

	eng.broker.EXPECT().SendBatch(gomock.Any()).AnyTimes()
	ctx, cfunc := context.WithCancel(context.Background())
	defer cfunc()
	evt := testMargin{
		party:   "party1",
		size:    1,
		price:   1000,
		asset:   "ETH",
		margin:  500,    // required margin will be > 35 (release), so ensure we don't have enough
		general: 100000, // plenty of balance for the transfer anyway
		market:  "ETH/DEC19",
	}
	// we're still in auction...
	eng.tsvc.EXPECT().GetTimeNow().Times(1)
	eng.as.EXPECT().InAuction().Times(2).Return(true)
	// but the auction is ending
	eng.as.EXPECT().CanLeave().Times(2).Return(true)
	// eng.as.EXPECT().InAuction().AnyTimes().Return(false)
	eng.orderbook.EXPECT().GetCloseoutPrice(gomock.Any(), gomock.Any()).Times(1).
		DoAndReturn(func(volume uint64, side types.Side) (*num.Uint, error) {
			return markPrice.Clone(), nil
		})
	evts := []events.Margin{evt}
	resp := eng.UpdateMarginsOnSettlement(ctx, evts, markPrice, num.DecimalZero())
	assert.Equal(t, 1, len(resp))

	// ensure we get the correct transfer request back, correct amount etc...
	trans := resp[0].Transfer()
	assert.EqualValues(t, 470, trans.Amount.Amount.Uint64())
	// assert.EqualValues(t, 446, trans.Amount.Amount.Uint64())
	// assert.Equal(t, riskMinamount-int64(evt.margin), trans.Amount.MinAmount)
	assert.Equal(t, types.TransferTypeMarginHigh, trans.Type)
}

func testMarginWithOrderInBook(t *testing.T) {
	// custom risk factors
	r := &types.RiskFactor{
		Short: num.DecimalFromFloat(.11),
		Long:  num.DecimalFromFloat(.10),
	}
	// custom scaling factor
	mc := &types.MarginCalculator{
		ScalingFactors: &types.ScalingFactors{
			SearchLevel:       num.DecimalFromFloat(1.1),
			InitialMargin:     num.DecimalFromFloat(1.2),
			CollateralRelease: num.DecimalFromFloat(1.3),
		},
	}

	markPrice := num.NewUint(144)

	// list of order in the book before the test happen
	ordersInBook := []struct {
		volume int64
		price  *num.Uint
		tid    string
		side   types.Side
	}{
		// asks
		// {volume: 3, price: num.NewUint(258), tid: "t1", side: types.SideSell},
		// {volume: 5, price: num.NewUint(240), tid: "t2", side: types.SideSell},
		// {volume: 3, price: num.NewUint(188), tid: "t3", side: types.SideSell},

		// bids
		{volume: 1, price: num.NewUint(120), tid: "t4", side: types.SideBuy},
		{volume: 4, price: num.NewUint(110), tid: "t5", side: types.SideBuy},
		{volume: 5, price: num.NewUint(108), tid: "t6", side: types.SideBuy},
	}

	marketID := "testingmarket"

	conf := config.NewDefaultConfig()
	log := logging.NewTestLogger()
	ctrl := gomock.NewController(t)
	model := mocks.NewMockModel(ctrl)
	ts := mocks.NewMockTimeService(ctrl)
	broker := bmocks.NewMockBroker(ctrl)
	as := mocks.NewMockAuctionState(ctrl)
	ts.EXPECT().GetTimeNow().Times(1)
	broker.EXPECT().Send(gomock.Any()).AnyTimes()

	// instantiate the book then fill it with the orders

	book := matching.NewOrderBook(log, conf.Execution.Matching, marketID, false, peggedOrderCounterForTest)

	for _, v := range ordersInBook {
		o := &types.Order{
			ID:          fmt.Sprintf("o-%v-%v", v.tid, marketID),
			MarketID:    marketID,
			Party:       "A",
			Side:        v.side,
			Price:       v.price.Clone(),
			Size:        uint64(v.volume),
			Remaining:   uint64(v.volume),
			TimeInForce: types.OrderTimeInForceGTT,
			Type:        types.OrderTypeLimit,
			Status:      types.OrderStatusActive,
			ExpiresAt:   10000,
		}
		_, err := book.SubmitOrder(o)
		assert.Nil(t, err)
	}

	model.EXPECT().DefaultRiskFactors().Return(r).Times(1)
	as.EXPECT().InAuction().AnyTimes().Return(false)
	statevar := mocks.NewMockStateVarEngine(ctrl)
	statevar.EXPECT().RegisterStateVariable(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any())
	statevar.EXPECT().NewEvent(gomock.Any(), gomock.Any(), gomock.Any())
	testE := risk.NewEngine(log, conf.Execution.Risk, mc, model, book, as, ts, broker, "mktid", "ETH", statevar, num.DecimalFromInt64(1), false, nil, DefaultSlippageFactor, DefaultSlippageFactor)
	evt := testMargin{
		party:   "tx",
		size:    10,
		buy:     4,
		sell:    8,
		price:   144,
		asset:   "ETH",
		margin:  500,
		general: 100000,
		market:  "ETH/DEC19",
	}
	// insufficient orders on the book
	riskevt, _, err := testE.UpdateMarginOnNewOrder(context.Background(), evt, markPrice.Clone(), num.DecimalZero())
	assert.NotNil(t, riskevt)
	if riskevt == nil {
		t.Fatal("expecting non nil risk update")
	}
	assert.Nil(t, err)
	margins := riskevt.MarginLevels()
	searchLevel, _ := mc.ScalingFactors.SearchLevel.Float64()
	initialMargin, _ := mc.ScalingFactors.InitialMargin.Float64()
	colRelease, _ := mc.ScalingFactors.CollateralRelease.Float64()
	assert.EqualValues(t, 542, margins.MaintenanceMargin.Uint64())
	assert.Equal(t, uint64(542*searchLevel), margins.SearchLevel.Uint64())
	assert.Equal(t, uint64(542*initialMargin), margins.InitialMargin.Uint64())
	assert.Equal(t, uint64(542*colRelease), margins.CollateralReleaseLevel.Uint64())
}

// testcase 1 from: https://drive.google.com/file/d/1B8-rLK2NB6rWvjzZX9sLtqOQzLz8s2ky/view
func testMarginWithOrderInBook2(t *testing.T) {
	// custom risk factors
	r := &types.RiskFactor{
		Short: num.DecimalFromFloat(.2),
		Long:  num.DecimalFromFloat(.1),
	}
	_ = r
	// custom scaling factor
	mc := &types.MarginCalculator{
		ScalingFactors: &types.ScalingFactors{
			SearchLevel:       num.DecimalFromFloat(3.2),
			InitialMargin:     num.DecimalFromFloat(4),
			CollateralRelease: num.DecimalFromFloat(5),
		},
	}

	// list of order in the book before the test happen
	ordersInBook := []struct {
		volume int64
		price  *num.Uint
		tid    string
		side   types.Side
	}{
		// asks
		{volume: 100, price: num.NewUint(250), tid: "t1", side: types.SideSell},
		{volume: 11, price: num.NewUint(140), tid: "t2", side: types.SideSell},
		{volume: 2, price: num.NewUint(112), tid: "t3", side: types.SideSell},
		// bids
		{volume: 1, price: num.NewUint(100), tid: "t4", side: types.SideBuy},
		{volume: 3, price: num.NewUint(96), tid: "t5", side: types.SideBuy},
		{volume: 15, price: num.NewUint(90), tid: "t6", side: types.SideBuy},
		{volume: 50, price: num.NewUint(87), tid: "t7", side: types.SideBuy},
	}

	marketID := "testingmarket"

	conf := config.NewDefaultConfig()
	log := logging.NewTestLogger()
	ctrl := gomock.NewController(t)
	model := mocks.NewMockModel(ctrl)
	ts := mocks.NewMockTimeService(ctrl)
	broker := bmocks.NewMockBroker(ctrl)
	as := mocks.NewMockAuctionState(ctrl)
	ts.EXPECT().GetTimeNow().Times(1)
	broker.EXPECT().Send(gomock.Any()).AnyTimes()

	model.EXPECT().DefaultRiskFactors().Return(r).Times(1)

	as.EXPECT().InAuction().AnyTimes().Return(false)
	// instantiate the book then fill it with the orders

	book := matching.NewOrderBook(log, conf.Execution.Matching, marketID, false, peggedOrderCounterForTest)

	for _, v := range ordersInBook {
		o := &types.Order{
			ID:          fmt.Sprintf("o-%v-%v", v.tid, marketID),
			MarketID:    marketID,
			Party:       "A",
			Side:        v.side,
			Price:       v.price.Clone(),
			Size:        uint64(v.volume),
			Remaining:   uint64(v.volume),
			TimeInForce: types.OrderTimeInForceGTT,
			Type:        types.OrderTypeLimit,
			Status:      types.OrderStatusActive,
			ExpiresAt:   10000,
		}
		_, err := book.SubmitOrder(o)
		assert.Nil(t, err)
	}

	statevar := mocks.NewMockStateVarEngine(ctrl)
	statevar.EXPECT().RegisterStateVariable(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any())
	statevar.EXPECT().NewEvent(gomock.Any(), gomock.Any(), gomock.Any())
	testE := risk.NewEngine(log, conf.Execution.Risk, mc, model, book, as, ts, broker, "mktid", "ETH", statevar, num.DecimalFromInt64(1), false, nil, DefaultSlippageFactor, DefaultSlippageFactor)
	evt := testMargin{
		party:   "tx",
		size:    13,
		buy:     0,
		sell:    0,
		price:   150,
		asset:   "ETH",
		margin:  0,
		general: 100000,
		market:  "ETH/DEC19",
	}

	previousMarkPrice := num.NewUint(103)

	riskevt, _, err := testE.UpdateMarginOnNewOrder(context.Background(), evt, previousMarkPrice, num.DecimalZero())
	assert.NotNil(t, riskevt)
	if riskevt == nil {
		t.Fatal("expecting non nil risk update")
	}
	assert.Nil(t, err)
	margins := riskevt.MarginLevels()
	searchLevel, _ := mc.ScalingFactors.SearchLevel.Float64()
	initialMargin, _ := mc.ScalingFactors.InitialMargin.Float64()
	colRelease, _ := mc.ScalingFactors.CollateralRelease.Float64()

	assert.Equal(t, uint64(277), margins.MaintenanceMargin.Uint64())
	assert.Equal(t, uint64(277*searchLevel), margins.SearchLevel.Uint64())
	assert.Equal(t, uint64(277*initialMargin), margins.InitialMargin.Uint64())
	assert.Equal(t, uint64(277*colRelease), margins.CollateralReleaseLevel.Uint64())
	// assert.Equal(t, uint64(2009), margins.MaintenanceMargin.Uint64())
	// assert.Equal(t, uint64(2009*searchLevel), margins.SearchLevel.Uint64())
	// assert.Equal(t, uint64(2009*initialMargin), margins.InitialMargin.Uint64())
	// assert.Equal(t, uint64(2009*colRelease), margins.CollateralReleaseLevel.Uint64())
}

func testMarginWithOrderInBookAfterParamsUpdate(t *testing.T) {
	// custom risk factors
	r := &types.RiskFactor{
		Short: num.DecimalFromFloat(.11),
		Long:  num.DecimalFromFloat(.10),
	}
	// custom scaling factor
	mc := &types.MarginCalculator{
		ScalingFactors: &types.ScalingFactors{
			SearchLevel:       num.DecimalFromFloat(1.1),
			InitialMargin:     num.DecimalFromFloat(1.2),
			CollateralRelease: num.DecimalFromFloat(1.3),
		},
	}

	markPrice := num.NewUint(144)

	// list of order in the book before the test happen
	ordersInBook := []struct {
		volume int64
		price  *num.Uint
		tid    string
		side   types.Side
	}{
		// asks
		// {volume: 3, price: 258, tid: "t1", side: types.Side_SIDE_SELL},
		// {volume: 5, price: 240, tid: "t2", side: types.Side_SIDE_SELL},
		// {volume: 3, price: 188, tid: "t3", side: types.Side_SIDE_SELL},
		// bids

		{volume: 1, price: num.NewUint(120), tid: "t4", side: types.SideBuy},
		{volume: 4, price: num.NewUint(110), tid: "t5", side: types.SideBuy},
		{volume: 5, price: num.NewUint(108), tid: "t6", side: types.SideBuy},
	}

	marketID := "testingmarket"

	conf := config.NewDefaultConfig()
	log := logging.NewTestLogger()
	ctrl := gomock.NewController(t)
	model := mocks.NewMockModel(ctrl)
	ts := mocks.NewMockTimeService(ctrl)
	broker := bmocks.NewMockBroker(ctrl)
	as := mocks.NewMockAuctionState(ctrl)
	ts.EXPECT().GetTimeNow().Times(2)
	broker.EXPECT().Send(gomock.Any()).AnyTimes()

	// instantiate the book then fill it with the orders

	book := matching.NewOrderBook(log, conf.Execution.Matching, marketID, false, peggedOrderCounterForTest)

	for _, v := range ordersInBook {
		o := &types.Order{
			ID:          fmt.Sprintf("o-%v-%v", v.tid, marketID),
			MarketID:    marketID,
			Party:       "A",
			Side:        v.side,
			Price:       v.price.Clone(),
			Size:        uint64(v.volume),
			Remaining:   uint64(v.volume),
			TimeInForce: types.OrderTimeInForceGTT,
			Type:        types.OrderTypeLimit,
			Status:      types.OrderStatusActive,
			ExpiresAt:   10000,
		}
		_, err := book.SubmitOrder(o)
		assert.Nil(t, err)
	}

	model.EXPECT().DefaultRiskFactors().Return(r).Times(1)
	as.EXPECT().InAuction().AnyTimes().Return(false)
	statevarEngine := mocks.NewMockStateVarEngine(ctrl)
	statevarEngine.EXPECT().RegisterStateVariable(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any())
	statevarEngine.EXPECT().NewEvent(gomock.Any(), gomock.Any(), gomock.Any())
	asset := "ETH"
	testE := risk.NewEngine(log, conf.Execution.Risk, mc, model, book, as, ts, broker, marketID, asset, statevarEngine, num.DecimalFromInt64(1), false, nil, DefaultSlippageFactor, DefaultSlippageFactor)

	evt := testMargin{
		party:   "tx",
		size:    10,
		buy:     4,
		sell:    8,
		price:   144,
		asset:   asset,
		margin:  500,
		general: 100000,
		market:  marketID,
	}
	riskevt, _, err := testE.UpdateMarginOnNewOrder(context.Background(), evt, markPrice.Clone(), num.DecimalZero())
	require.NotNil(t, riskevt)
	require.Nil(t, err)

	margins := riskevt.MarginLevels()
	searchLevel, _ := mc.ScalingFactors.SearchLevel.Float64()
	initialMargin, _ := mc.ScalingFactors.InitialMargin.Float64()
	colRelease, _ := mc.ScalingFactors.CollateralRelease.Float64()
	assert.EqualValues(t, 542, margins.MaintenanceMargin.Uint64())
	assert.Equal(t, uint64(542*searchLevel), margins.SearchLevel.Uint64())
	assert.Equal(t, uint64(542*initialMargin), margins.InitialMargin.Uint64())
	assert.Equal(t, uint64(542*colRelease), margins.CollateralReleaseLevel.Uint64())

	updatedRF := &types.RiskFactor{
		Short: num.DecimalFromFloat(.12),
		Long:  num.DecimalFromFloat(.11),
	}
	updatedMC := &types.MarginCalculator{
		ScalingFactors: &types.ScalingFactors{
			SearchLevel:       num.DecimalFromFloat(1.2),
			InitialMargin:     num.DecimalFromFloat(1.4),
			CollateralRelease: num.DecimalFromFloat(1.4),
		},
	}
	model.EXPECT().DefaultRiskFactors().Return(updatedRF).Times(1)
	statevarEngine.EXPECT().NewEvent(asset, marketID, statevar.EventTypeMarketUpdated)
	testE.UpdateModel(statevarEngine, updatedMC, model)

	evt = testMargin{
		party:   "tx",
		size:    10,
		buy:     4,
		sell:    8,
		price:   144,
		asset:   asset,
		margin:  500,
		general: 100000,
		market:  marketID,
	}
	riskevt, _, err = testE.UpdateMarginOnNewOrder(context.Background(), evt, markPrice.Clone(), num.DecimalZero())
	require.NotNil(t, riskevt)
	require.Nil(t, err)

	margins = riskevt.MarginLevels()
	searchLevel, _ = updatedMC.ScalingFactors.SearchLevel.Float64()
	initialMargin, _ = updatedMC.ScalingFactors.InitialMargin.Float64()
	colRelease, _ = updatedMC.ScalingFactors.CollateralRelease.Float64()
	assert.EqualValues(t, 562, margins.MaintenanceMargin.Uint64())
	assert.Equal(t, uint64(562*searchLevel), margins.SearchLevel.Uint64())
	assert.Equal(t, uint64(562*initialMargin), margins.InitialMargin.Uint64())
	assert.Equal(t, uint64(562*colRelease), margins.CollateralReleaseLevel.Uint64())
}

func testInitialMarginRequirement(t *testing.T) {
	eng := getTestEngine(t)

	_, cfunc := context.WithCancel(context.Background())
	defer cfunc()

	initialMargin := uint64(96)

	evt := testMargin{
		party:   "party1",
		size:    -4,
		price:   1000,
		asset:   "ETH",
		margin:  0,
		general: initialMargin - 1,
		market:  "ETH/DEC19",
	}
	eng.tsvc.EXPECT().GetTimeNow().Times(6)
	eng.as.EXPECT().InAuction().Times(2).Return(false)
	eng.orderbook.EXPECT().GetCloseoutPrice(gomock.Any(), gomock.Any()).Times(2).
		DoAndReturn(func(volume uint64, side types.Side) (*num.Uint, error) {
			return markPrice.Clone(), nil
		})
	eng.broker.EXPECT().SendBatch(gomock.Any()).Times(3)
	riskevt, _, err := eng.UpdateMarginOnNewOrder(context.Background(), evt, markPrice, num.DecimalZero())
	assert.Error(t, err, risk.ErrInsufficientFundsForInitialMargin.Error())
	assert.Nil(t, riskevt)

	evt.general = initialMargin
	riskevt, _, err = eng.UpdateMarginOnNewOrder(context.Background(), evt, markPrice, num.DecimalZero())
	assert.NoError(t, err)
	assert.NotNil(t, riskevt)
	assert.True(t, riskevt.MarginLevels().InitialMargin.EQ(num.NewUint(initialMargin)))

	eng.as.EXPECT().InAuction().Times(4).Return(true)
	eng.as.EXPECT().CanLeave().Times(4).Return(false)

	slippageFactor := DefaultSlippageFactor.InexactFloat64()
	size := math.Abs(float64(evt.size))
	rf := eng.GetRiskFactors()
	initialMarginScalingFactor := 1.2
	initialMarginAuction := math.Ceil(initialMarginScalingFactor * (size*slippageFactor + size*size*slippageFactor + size*rf.Short.InexactFloat64()) * markPrice.ToDecimal().InexactFloat64())

	evt.general = uint64(initialMarginAuction) - 1
	riskevt, _, err = eng.UpdateMarginOnNewOrder(context.Background(), evt, markPrice, num.DecimalZero())
	assert.Error(t, err, risk.ErrInsufficientFundsForInitialMargin.Error())
	assert.Nil(t, riskevt)

	evt.general = uint64(initialMarginAuction)
	riskevt, _, err = eng.UpdateMarginOnNewOrder(context.Background(), evt, markPrice, num.DecimalZero())
	assert.NoError(t, err)
	assert.NotNil(t, riskevt)
	assert.True(t, riskevt.MarginLevels().InitialMargin.EQ(num.NewUint(uint64(initialMarginAuction))))

	evt.sell = 7
	evt.sellSumProduct = 123

	ordersBit := evt.SellSumProduct().Float64() * rf.Short.InexactFloat64()
	initialMarginAuction = math.Ceil(initialMarginAuction + initialMarginScalingFactor*ordersBit)

	evt.general = uint64(initialMarginAuction) - 1
	riskevt, _, err = eng.UpdateMarginOnNewOrder(context.Background(), evt, markPrice, num.DecimalZero())
	assert.Error(t, err, risk.ErrInsufficientFundsForInitialMargin.Error())
	assert.Nil(t, riskevt)

	evt.general = uint64(math.Ceil(initialMarginAuction))
	riskevt, _, err = eng.UpdateMarginOnNewOrder(context.Background(), evt, markPrice, num.DecimalZero())
	assert.NoError(t, err)
	assert.NotNil(t, riskevt)
	assert.True(t, riskevt.MarginLevels().InitialMargin.EQ(num.NewUint(uint64(initialMarginAuction))))
}

func TestMaintenanceMarign(t *testing.T) {
	relativeTolerance := num.DecimalFromFloat(0.000001)

	testCases := []struct {
		markPrice               float64
		positionFactor          float64
		positionSize            int64
		buyOrders               []*risk.OrderInfo
		sellOrders              []*risk.OrderInfo
		linearSlippageFactor    float64
		quadraticSlippageFactor float64
		riskFactorLong          float64
		riskFactorShort         float64
		auction                 bool
	}{
		{
			markPrice:      123.4,
			positionFactor: 1,
			positionSize:   40000,
			buyOrders: []*risk.OrderInfo{
				{1, num.NewDecimalFromFloat(0), false},
			},
			sellOrders: []*risk.OrderInfo{
				{1, num.NewDecimalFromFloat(0), false},
			},
			linearSlippageFactor:    0,
			quadraticSlippageFactor: 0,
			riskFactorLong:          0.1,
			riskFactorShort:         0.1,
			auction:                 false,
		},
		{
			markPrice:      123.4,
			positionFactor: 10,
			positionSize:   40000,
			buyOrders: []*risk.OrderInfo{
				{1, num.NewDecimalFromFloat(111.1), false},
			},
			sellOrders: []*risk.OrderInfo{
				{1, num.NewDecimalFromFloat(133.3), false},
			},
			linearSlippageFactor:    0,
			quadraticSlippageFactor: 0,
			riskFactorLong:          0.1,
			riskFactorShort:         0.1,
			auction:                 true,
		},
		{
			markPrice:      123.4,
			positionFactor: 10,
			positionSize:   0,
			buyOrders: []*risk.OrderInfo{
				{1, num.NewDecimalFromFloat(111.1), false},
			},
			sellOrders: []*risk.OrderInfo{
				{1, num.NewDecimalFromFloat(133.3), false},
			},
			linearSlippageFactor:    0,
			quadraticSlippageFactor: 0,
			riskFactorLong:          0.1,
			riskFactorShort:         0.1,
			auction:                 true,
		},
		{
			markPrice:      123.4,
			positionFactor: 10,
			positionSize:   0,
			buyOrders: []*risk.OrderInfo{
				{10000, num.NewDecimalFromFloat(111.1), false},
			},
			sellOrders: []*risk.OrderInfo{
				{30000, num.NewDecimalFromFloat(133.3), false},
			},
			linearSlippageFactor:    0,
			quadraticSlippageFactor: 0,
			riskFactorLong:          0.1,
			riskFactorShort:         0.2,
			auction:                 false,
		},
		{
			markPrice:      123.4,
			positionFactor: 100,
			positionSize:   0,
			buyOrders: []*risk.OrderInfo{
				{40000, num.NewDecimalFromFloat(111.1), false},
			},
			sellOrders: []*risk.OrderInfo{
				{30000, num.NewDecimalFromFloat(133.3), false},
			},
			linearSlippageFactor:    0.5,
			quadraticSlippageFactor: 0.1,
			riskFactorLong:          0.1,
			riskFactorShort:         0.2,
			auction:                 false,
		},
		{
			markPrice:      123.4,
			positionFactor: 100,
			positionSize:   0,
			buyOrders: []*risk.OrderInfo{
				{10000, num.NewDecimalFromFloat(111.4), false},
				{30000, num.NewDecimalFromFloat(111), false},
			},
			sellOrders: []*risk.OrderInfo{
				{10000, num.NewDecimalFromFloat(133.9), false},
				{20000, num.NewDecimalFromFloat(133.0), false},
			},
			linearSlippageFactor:    0.5,
			quadraticSlippageFactor: 0.1,
			riskFactorLong:          0.1,
			riskFactorShort:         0.2,
			auction:                 false,
		},
		{
			markPrice:      123.4,
			positionFactor: 100,
			positionSize:   0,
			buyOrders: []*risk.OrderInfo{
				{10000, num.NewDecimalFromFloat(111.4), false},
				{30000, num.NewDecimalFromFloat(111), false},
				{20000, num.NewDecimalFromFloat(0), true},
			},
			sellOrders: []*risk.OrderInfo{
				{10000, num.NewDecimalFromFloat(133.9), false},
				{20000, num.NewDecimalFromFloat(133.0), false},
				{30000, num.NewDecimalFromFloat(0), true},
			},
			linearSlippageFactor:    0.5,
			quadraticSlippageFactor: 0.1,
			riskFactorLong:          0.1,
			riskFactorShort:         0.2,
			auction:                 false,
		},
	}

	for i, tc := range testCases {
		markPrice := num.DecimalFromFloat(tc.markPrice)
		positionFactor := num.DecimalFromFloat(tc.positionFactor)
		buySumProduct := num.DecimalZero()
		sellSumProduct := num.DecimalZero()
		buySize := int64(0)
		sellSize := int64(0)

		linearSlippageFactor := num.DecimalFromFloat(tc.linearSlippageFactor)
		quadraticSlippageFactor := num.DecimalFromFloat(tc.quadraticSlippageFactor)
		riskFactorLong := num.DecimalFromFloat(tc.riskFactorLong)
		riskFactorShort := num.DecimalFromFloat(tc.riskFactorShort)

		positionSize := tc.positionSize
		for _, o := range tc.buyOrders {
			s := int64(o.Size)
			if o.IsMarketOrder {
				positionSize += s
			} else {
				buySize += s
				buySumProduct = buySumProduct.Add(num.DecimalFromInt64(s).Mul(o.Price))
			}
		}

		for _, o := range tc.sellOrders {
			s := int64(o.Size)
			if o.IsMarketOrder {
				positionSize -= s
			} else {
				sellSize += s
				sellSumProduct = sellSumProduct.Add(num.DecimalFromInt64(s).Mul(o.Price))
			}
		}

		openVolume := num.DecimalFromInt64(positionSize).Div(positionFactor).Abs()
		expectedMarginShort, expectedMarginLong := num.DecimalZero(), num.DecimalZero()
		slippage := markPrice.Mul(openVolume.Mul(linearSlippageFactor).Add(openVolume.Mul(openVolume).Mul(quadraticSlippageFactor)))

		if positionSize-sellSize < 0 {
			expectedMarginShort = slippage.Add(openVolume.Mul(markPrice).Mul(riskFactorShort))
			orders := num.DecimalFromInt64(sellSize).Div(positionFactor).Abs().Mul(riskFactorShort)
			if tc.auction {
				expectedMarginShort = expectedMarginShort.Add(orders.Mul(sellSumProduct))
			} else {
				expectedMarginShort = expectedMarginShort.Add(orders.Mul(markPrice))
			}
		}
		if positionSize+buySize > 0 {
			expectedMarginLong = slippage.Add(openVolume.Mul(markPrice).Mul(riskFactorLong))
			orders := num.DecimalFromInt64(buySize).Div(positionFactor).Abs().Mul(riskFactorLong)
			if tc.auction {
				expectedMarginLong = expectedMarginLong.Add(orders.Mul(buySumProduct))
			} else {
				expectedMarginLong = expectedMarginLong.Add(orders.Mul(markPrice))
			}
		}
		expectedMargin := num.MaxD(expectedMarginShort, expectedMarginLong)

		actualMargin := risk.CalculateMaintenanceMarginWithSlippageFactors(tc.positionSize, tc.buyOrders, tc.sellOrders, markPrice, positionFactor, linearSlippageFactor, quadraticSlippageFactor, riskFactorLong, riskFactorShort, tc.auction)

		require.True(t, expectedMargin.Div(actualMargin).Sub(num.DecimalOne()).Abs().LessThan(relativeTolerance), fmt.Sprintf("Test case %v: expectedMargin=%s, actualMargin:=%s", i+1, expectedMargin, actualMargin))
	}
}

func TestLiquidationPriceWithNoOrders(t *testing.T) {
	relativeTolerance := num.DecimalFromFloat(0.000001)

	testCases := []struct {
		markPrice               float64
		positionFactor          float64
		positionSize            int64
		linearSlippageFactor    float64
		quadraticSlippageFactor float64
		riskFactorLong          float64
		riskFactorShort         float64
		collateralFactor        float64
		expectError             bool
	}{
		{
			markPrice:               123.4,
			positionFactor:          1,
			positionSize:            40000,
			linearSlippageFactor:    0,
			quadraticSlippageFactor: 0,
			riskFactorLong:          0.1,
			riskFactorShort:         0.11,
			collateralFactor:        1.7,
		},
		{
			markPrice:               1234.5,
			positionFactor:          10,
			positionSize:            40000,
			linearSlippageFactor:    0.5,
			quadraticSlippageFactor: 0,
			riskFactorLong:          0.1,
			riskFactorShort:         0.11,
			collateralFactor:        1.1,
		},
		{
			markPrice:               1234.5,
			positionFactor:          100,
			positionSize:            -40000,
			linearSlippageFactor:    0.5,
			quadraticSlippageFactor: 0.01,
			riskFactorLong:          0.1,
			riskFactorShort:         0.11,
			collateralFactor:        3,
		},
		{
			markPrice:               1234.5,
			positionFactor:          1000,
			positionSize:            -40000,
			linearSlippageFactor:    0.5,
			quadraticSlippageFactor: 0.1,
			riskFactorLong:          0.1,
			riskFactorShort:         0.11,
			collateralFactor:        0.2,
		},
		{
			markPrice:               1,
			positionFactor:          1,
			positionSize:            1,
			linearSlippageFactor:    0,
			quadraticSlippageFactor: 0,
			riskFactorLong:          1,
			riskFactorShort:         1,
			collateralFactor:        2.5,
			expectError:             true,
		},
		{
			markPrice:               110,
			positionFactor:          1,
			positionSize:            41,
			linearSlippageFactor:    0.5,
			quadraticSlippageFactor: 0.01,
			riskFactorLong:          0.1,
			riskFactorShort:         0.11,
			collateralFactor:        1000,
		},
	}

	for i, tc := range testCases {
		markPrice := num.DecimalFromFloat(tc.markPrice)
		positionFactor := num.DecimalFromFloat(tc.positionFactor)

		linearSlippageFactor := num.DecimalFromFloat(tc.linearSlippageFactor)
		quadraticSlippageFactor := num.DecimalFromFloat(tc.quadraticSlippageFactor)
		riskFactorLong := num.DecimalFromFloat(tc.riskFactorLong)
		riskFactorShort := num.DecimalFromFloat(tc.riskFactorShort)

		maintenanceMargin := risk.CalculateMaintenanceMarginWithSlippageFactors(tc.positionSize, nil, nil, markPrice, positionFactor, linearSlippageFactor, quadraticSlippageFactor, riskFactorLong, riskFactorShort, false)

		liquidationPrice, _, _, err := risk.CalculateLiquidationPriceWithSlippageFactors(tc.positionSize, nil, nil, markPrice, maintenanceMargin, positionFactor, linearSlippageFactor, quadraticSlippageFactor, riskFactorLong, riskFactorShort)
		if tc.expectError {
			require.Error(t, err)
			continue
		}
		require.NoError(t, err)

		liquidationPriceFp := liquidationPrice.InexactFloat64()
		require.GreaterOrEqual(t, liquidationPriceFp, 0.0)

		require.True(t, markPrice.Div(liquidationPrice).Sub(num.DecimalOne()).Abs().LessThan(relativeTolerance))

		collateral := maintenanceMargin.Mul(num.DecimalFromFloat(tc.collateralFactor))

		liquidationPrice, _, _, err = risk.CalculateLiquidationPriceWithSlippageFactors(tc.positionSize, nil, nil, markPrice, collateral, positionFactor, linearSlippageFactor, quadraticSlippageFactor, riskFactorLong, riskFactorShort)
		require.NoError(t, err)
		require.False(t, liquidationPrice.IsNegative())

		marginAtLiquidationPrice := risk.CalculateMaintenanceMarginWithSlippageFactors(tc.positionSize, nil, nil, liquidationPrice, positionFactor, linearSlippageFactor, quadraticSlippageFactor, riskFactorLong, riskFactorShort, false)
		openVolume := num.DecimalFromInt64(tc.positionSize).Div(positionFactor)
		mtmLoss := liquidationPrice.Sub(markPrice).Mul(openVolume)
		collateralAfterLoss := collateral.Add(mtmLoss)

		if !marginAtLiquidationPrice.IsZero() {
			require.True(t, collateralAfterLoss.Div(marginAtLiquidationPrice).Sub(num.DecimalOne()).Abs().LessThan(relativeTolerance), fmt.Sprintf("Test case %v: collateralAfterLoss=%s, marginAtLiquidationPrice:=%s", i+1, collateralAfterLoss, marginAtLiquidationPrice))
		} else {
			require.True(t, liquidationPrice.IsZero(), fmt.Sprintf("Test case %v:", i+1))
			require.True(t, collateralAfterLoss.IsNegative(), fmt.Sprintf("Test case %v:", i+1))
		}
	}
}

func TestLiquidationPriceWithOrders(t *testing.T) {
	testCases := []struct {
		markPrice               float64
		positionFactor          float64
		positionSize            int64
		buyOrders               []*risk.OrderInfo
		sellOrders              []*risk.OrderInfo
		linearSlippageFactor    float64
		quadraticSlippageFactor float64
		riskFactorLong          float64
		riskFactorShort         float64
		collateralAvailable     float64
	}{
		{
			markPrice:      123.4,
			positionFactor: 1,
			positionSize:   0,
			buyOrders: []*risk.OrderInfo{
				{1, num.NewDecimalFromFloat(110), false},
			},
			sellOrders: []*risk.OrderInfo{
				{1, num.NewDecimalFromFloat(130), false},
			},
			linearSlippageFactor:    0,
			quadraticSlippageFactor: 0,
			riskFactorLong:          0.1,
			riskFactorShort:         0.11,
			collateralAvailable:     100,
		},
		{
			markPrice:      123.4,
			positionFactor: 1,
			positionSize:   0,
			buyOrders: []*risk.OrderInfo{
				{39, num.NewDecimalFromFloat(110), false},
			},
			sellOrders: []*risk.OrderInfo{
				{40, num.NewDecimalFromFloat(130), false},
			},
			linearSlippageFactor:    0.5,
			quadraticSlippageFactor: 0.01,
			riskFactorLong:          0.1,
			riskFactorShort:         0.11,
			collateralAvailable:     50800,
		},
		{
			markPrice:      123.4,
			positionFactor: 1,
			positionSize:   20,
			buyOrders: []*risk.OrderInfo{
				{39, num.NewDecimalFromFloat(110), false},
			},
			sellOrders: []*risk.OrderInfo{
				{40, num.NewDecimalFromFloat(130), false},
			},
			linearSlippageFactor:    0.01,
			quadraticSlippageFactor: 0.000001,
			riskFactorLong:          0.1,
			riskFactorShort:         0.11,
			collateralAvailable:     50800,
		},
		{
			markPrice:      123.4,
			positionFactor: 100,
			positionSize:   -2000,
			buyOrders: []*risk.OrderInfo{
				{3900, num.NewDecimalFromFloat(110), false},
			},
			sellOrders: []*risk.OrderInfo{
				{4000, num.NewDecimalFromFloat(130), false},
			},
			linearSlippageFactor:    0.01,
			quadraticSlippageFactor: 0.000001,
			riskFactorLong:          0.1,
			riskFactorShort:         0.11,
			collateralAvailable:     50800,
		},
		{
			markPrice:      123.4,
			positionFactor: 0.1,
			positionSize:   -2,
			buyOrders: []*risk.OrderInfo{
				{3, num.NewDecimalFromFloat(110), false},
			},
			sellOrders: []*risk.OrderInfo{
				{4, num.NewDecimalFromFloat(130), false},
			},
			linearSlippageFactor:    0.01,
			quadraticSlippageFactor: 0.000001,
			riskFactorLong:          0.1,
			riskFactorShort:         0.11,
			collateralAvailable:     50800,
		},
		{
			markPrice:      101.2,
			positionFactor: 100,
			positionSize:   -2000,
			buyOrders: []*risk.OrderInfo{
				{3900, num.NewDecimalFromFloat(110), false},
			},
			sellOrders: []*risk.OrderInfo{
				{4000, num.NewDecimalFromFloat(130), false},
			},
			linearSlippageFactor:    0.01,
			quadraticSlippageFactor: 0.000001,
			riskFactorLong:          0.1,
			riskFactorShort:         0.11,
			collateralAvailable:     50800,
		},
		{
			markPrice:      144.5,
			positionFactor: 100,
			positionSize:   -2000,
			buyOrders: []*risk.OrderInfo{
				{3900, num.NewDecimalFromFloat(110), false},
			},
			sellOrders: []*risk.OrderInfo{
				{4000, num.NewDecimalFromFloat(130), false},
			},
			linearSlippageFactor:    0.01,
			quadraticSlippageFactor: 0.000001,
			riskFactorLong:          0.1,
			riskFactorShort:         0.11,
			collateralAvailable:     50800,
		},
		{
			markPrice:      123.4,
			positionFactor: 100,
			positionSize:   -2000,
			buyOrders: []*risk.OrderInfo{
				{1800, num.NewDecimalFromFloat(100), false},
				{1700, num.NewDecimalFromFloat(110), false},
			},
			sellOrders: []*risk.OrderInfo{
				{3000, num.NewDecimalFromFloat(120), false},
				{2000, num.NewDecimalFromFloat(130), false},
				{1000, num.NewDecimalFromFloat(140), false},
			},
			linearSlippageFactor:    0.01,
			quadraticSlippageFactor: 0.000001,
			riskFactorLong:          0.1,
			riskFactorShort:         0.11,
			collateralAvailable:     50800,
		},
		{
			markPrice:      123.4,
			positionFactor: 100,
			positionSize:   -2000,
			buyOrders: []*risk.OrderInfo{
				{1800, num.NewDecimalFromFloat(100), false},
				{1700, num.NewDecimalFromFloat(0), true},
			},
			sellOrders: []*risk.OrderInfo{
				{3000, num.NewDecimalFromFloat(120), false},
				{2000, num.NewDecimalFromFloat(130), false},
				{1000, num.NewDecimalFromFloat(0), true},
			},
			linearSlippageFactor:    0.01,
			quadraticSlippageFactor: 0.000001,
			riskFactorLong:          0.1,
			riskFactorShort:         0.11,
			collateralAvailable:     50800,
		},
		{
			markPrice:      123.4,
			positionFactor: 1,
			positionSize:   1,
			buyOrders:      []*risk.OrderInfo{},
			sellOrders: []*risk.OrderInfo{
				{2, num.NewDecimalFromFloat(0), true},
				{99, num.NewDecimalFromFloat(0), true},
			},
			linearSlippageFactor:    0.01,
			quadraticSlippageFactor: 0.000001,
			riskFactorLong:          0.1,
			riskFactorShort:         0.11,
			collateralAvailable:     2345,
		},
		{
			markPrice:      123.4,
			positionFactor: 1,
			positionSize:   1,
			buyOrders:      []*risk.OrderInfo{},
			sellOrders: []*risk.OrderInfo{
				{1, num.NewDecimalFromFloat(0), true},
				{100, num.NewDecimalFromFloat(0), true},
			},
			linearSlippageFactor:    0.01,
			quadraticSlippageFactor: 0.000001,
			riskFactorLong:          0.1,
			riskFactorShort:         0.11,
			collateralAvailable:     2345,
		},
	}

	for i, tc := range testCases {
		markPrice := num.DecimalFromFloat(tc.markPrice)
		positionFactor := num.DecimalFromFloat(tc.positionFactor)
		collateral := num.DecimalFromFloat(tc.collateralAvailable)

		linearSlippageFactor := num.DecimalFromFloat(tc.linearSlippageFactor)
		quadraticSlippageFactor := num.DecimalFromFloat(tc.quadraticSlippageFactor)
		riskFactorLong := num.DecimalFromFloat(tc.riskFactorLong)
		riskFactorShort := num.DecimalFromFloat(tc.riskFactorShort)

		positionOnly, withBuy, withSell, err := risk.CalculateLiquidationPriceWithSlippageFactors(tc.positionSize, tc.buyOrders, tc.sellOrders, markPrice, collateral, positionFactor, linearSlippageFactor, quadraticSlippageFactor, riskFactorLong, riskFactorShort)
		require.NoError(t, err, fmt.Sprintf("Test case %v:", i+1))

		sPositionOnly := positionOnly.String()
		sWithBuy := withBuy.String()
		sWithSell := withSell.String()

		t.Logf("positionOnly=%s, withBuy=%s, withSell=%s", sPositionOnly, sWithBuy, sWithSell)

		if tc.positionSize == 0 {
			require.True(t, positionOnly.IsZero(), fmt.Sprintf("Test case %v:", i+1))
		}
		if tc.positionSize > 0 {
			require.True(t, withBuy.GreaterThanOrEqual(positionOnly), fmt.Sprintf("Test case %v:", i+1))
		}
		if tc.positionSize < 0 {
			require.True(t, withSell.LessThanOrEqual(positionOnly), fmt.Sprintf("Test case %v:", i+1))
		}

		for _, o := range tc.buyOrders {
			if o.IsMarketOrder {
				o.Price = markPrice
			}
		}

		for _, o := range tc.sellOrders {
			if o.IsMarketOrder {
				o.Price = markPrice
			}
		}

		sort.Slice(tc.buyOrders, func(i, j int) bool {
			return tc.buyOrders[i].Price.GreaterThan(tc.buyOrders[j].Price)
		})
		sort.Slice(tc.sellOrders, func(i, j int) bool {
			return tc.sellOrders[i].Price.LessThan(tc.sellOrders[j].Price)
		})

		newPositionSize := tc.positionSize
		mtmDelta := num.DecimalZero()
		lastMarkPrice := markPrice
		for _, o := range tc.buyOrders {
			if o.Price.LessThan(withBuy) {
				break
			}
			mtmDelta = mtmDelta.Add(num.DecimalFromInt64(newPositionSize).Mul(o.Price.Sub(lastMarkPrice)))
			newPositionSize += int64(o.Size)
			lastMarkPrice = o.Price
		}
		collateralAfterMtm := collateral.Add(mtmDelta)
		newPositionOnly, _, _, err := risk.CalculateLiquidationPriceWithSlippageFactors(newPositionSize, nil, nil, lastMarkPrice, collateralAfterMtm, positionFactor, linearSlippageFactor, quadraticSlippageFactor, riskFactorLong, riskFactorShort)
		require.NoError(t, err, fmt.Sprintf("Test case %v:", i+1))
		require.True(t, withBuy.Equal(newPositionOnly), fmt.Sprintf("Test case %v: withBuy=%s, newPositionOnly=%s", i+1, withBuy.String(), newPositionOnly.String()))

		if tc.positionSize < 0 && newPositionSize > 0 {
			require.True(t, withBuy.LessThan(positionOnly), fmt.Sprintf("Test case %v:", i+1))
		}

		newPositionSize = tc.positionSize
		mtmDelta = num.DecimalZero()
		lastMarkPrice = markPrice
		for _, o := range tc.sellOrders {
			if o.Price.GreaterThan(withSell) {
				break
			}
			mtmDelta = mtmDelta.Add(num.DecimalFromInt64(newPositionSize).Div(positionFactor).Mul(o.Price.Sub(lastMarkPrice)))
			newPositionSize -= int64(o.Size)
			lastMarkPrice = o.Price
		}
		collateralAfterMtm = collateral.Add(mtmDelta)
		newPositionOnly, _, _, err = risk.CalculateLiquidationPriceWithSlippageFactors(newPositionSize, nil, nil, lastMarkPrice, collateralAfterMtm, positionFactor, linearSlippageFactor, quadraticSlippageFactor, riskFactorLong, riskFactorShort)
		require.NoError(t, err, fmt.Sprintf("Test case %v:", i+1))
		require.True(t, withSell.Equal(newPositionOnly), fmt.Sprintf("Test case %v: withSell=%s, newPositionOnly=%s", i+1, withSell.String(), newPositionOnly.String()))

		if tc.positionSize > 0 && newPositionSize < 0 {
			require.True(t, withSell.GreaterThan(positionOnly), fmt.Sprintf("Test case %v:", i+1))
		}
	}
}

func getTestEngine(t *testing.T) *testEngine {
	t.Helper()
	cpy := riskFactors
	cpyPtr := &cpy
	ctrl := gomock.NewController(t)
	model := mocks.NewMockModel(ctrl)
	conf := risk.NewDefaultConfig()
	conf.StreamMarginLevelsVerbose = true
	ob := mocks.NewMockOrderbook(ctrl)
	ts := mocks.NewMockTimeService(ctrl)
	broker := bmocks.NewMockBroker(ctrl)
	as := mocks.NewMockAuctionState(ctrl)
	model.EXPECT().DefaultRiskFactors().Return(cpyPtr).Times(1)
	statevar := mocks.NewMockStateVarEngine(ctrl)
	statevar.EXPECT().RegisterStateVariable(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any())
	statevar.EXPECT().NewEvent(gomock.Any(), gomock.Any(), gomock.Any())
	engine := risk.NewEngine(logging.NewTestLogger(),
		conf,
		getMarginCalculator(),
		model,
		ob,
		as,
		ts,
		broker,
		"mktid",
		"ETH",
		statevar,
		num.DecimalFromInt64(1),
		false,
		nil,
		DefaultSlippageFactor,
		DefaultSlippageFactor,
	)

	return &testEngine{
		Engine:    engine,
		ctrl:      ctrl,
		model:     model,
		orderbook: ob,
		tsvc:      ts,
		broker:    broker,
		as:        as,
	}
}

func getMarginCalculator() *types.MarginCalculator {
	return &types.MarginCalculator{
		ScalingFactors: &types.ScalingFactors{
			SearchLevel:       num.DecimalFromFloat(1.1),
			InitialMargin:     num.DecimalFromFloat(1.2),
			CollateralRelease: num.DecimalFromFloat(1.4),
		},
	}
}

func (m testMargin) Party() string {
	return m.party
}

func (m testMargin) MarketID() string {
	return m.market
}

func (m testMargin) Asset() string {
	return m.asset
}

func (m testMargin) MarginBalance() *num.Uint {
	return num.NewUint(m.margin)
}

func (m testMargin) GeneralBalance() *num.Uint {
	return num.NewUint(m.general)
}

func (m testMargin) BondBalance() *num.Uint {
	return num.UintZero()
}

func (m testMargin) Price() *num.Uint {
	return num.NewUint(m.price)
}

func (m testMargin) Buy() int64 {
	return m.buy
}

func (m testMargin) Sell() int64 {
	return m.sell
}

func (m testMargin) Size() int64 {
	return m.size
}

func (m testMargin) BuySumProduct() *num.Uint {
	return num.NewUint(m.buySumProduct)
}

func (m testMargin) SellSumProduct() *num.Uint {
	return num.NewUint(m.sellSumProduct)
}

func (m testMargin) VWBuy() *num.Uint {
	if m.buy == 0 {
		num.UintZero()
	}
	return num.UintZero().Div(m.BuySumProduct(), num.NewUint(uint64(m.buy)))
}

func (m testMargin) VWSell() *num.Uint {
	if m.sell == 0 {
		num.UintZero()
	}
	return num.UintZero().Div(m.SellSumProduct(), num.NewUint(uint64(m.sell)))
}

func (m testMargin) ClearPotentials() {}

func (m testMargin) Transfer() *types.Transfer {
	return m.transfer
}

func (m testMargin) MarginShortFall() *num.Uint {
	return num.NewUint(m.marginShortFall)
}
