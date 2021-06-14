package risk_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	bmock "code.vegaprotocol.io/vega/broker/mocks"
	"code.vegaprotocol.io/vega/config"
	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/matching"
	"code.vegaprotocol.io/vega/risk"
	"code.vegaprotocol.io/vega/risk/mocks"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

type MLEvent interface {
	events.Event
	MarginLevels() types.MarginLevels
}

type testEngine struct {
	*risk.Engine
	ctrl      *gomock.Controller
	model     *mocks.MockModel
	orderbook *mocks.MockOrderbook
	broker    *bmock.MockBroker
	as        *mocks.MockAuctionState
}

// implements the events.Margin interface
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
	vwBuy           uint64
	vwSell          uint64
	marginShortFall uint64
}

var (
	riskResult = types.RiskResult{
		RiskFactors: map[string]*types.RiskFactor{
			"ETH": {
				Market: "ETH/DEC19",
				Short:  .20,
				Long:   .25,
			},
		},
		PredictedNextRiskFactors: map[string]*types.RiskFactor{
			"ETH": {
				Market: "ETH/DEC19",
				Short:  .20,
				Long:   .25,
			},
		},
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
	t.Run("Top up fail on new order", testMarginTopupOnOrderFailInsufficientFunds)
}

func testMarginLevelsTS(t *testing.T) {
	eng := getTestEngine(t, nil)
	defer eng.ctrl.Finish()
	ctx, cfunc := context.WithCancel(context.Background())
	defer cfunc()
	evt := testMargin{
		party:   "trader1",
		size:    1,
		price:   1000,
		asset:   "ETH",
		margin:  10,     // required margin will be > 30 so ensure we don't have enough
		general: 100000, // plenty of balance for the transfer anyway
		market:  "ETH/DEC19",
	}

	eng.orderbook.EXPECT().GetCloseoutPrice(gomock.Any(), gomock.Any()).Times(1).
		DoAndReturn(func(volume uint64, side types.Side) (*num.Uint, error) {
			return markPrice.Clone(), nil
		})

	eng.as.EXPECT().InAuction().AnyTimes().Return(false)
	ts := time.Date(2018, time.January, 23, 0, 0, 0, 0, time.UTC)
	eng.OnTimeUpdate(ts)

	eng.broker.EXPECT().Send(gomock.Any()).Times(1).Do(func(e events.Event) {
		mle, ok := e.(MLEvent)
		assert.True(t, ok)
		ml := mle.MarginLevels()
		assert.Equal(t, ts.UnixNano(), ml.Timestamp)
	})

	evts := []events.Margin{evt}
	resp := eng.UpdateMarginsOnSettlement(ctx, evts, markPrice)
	assert.Equal(t, 1, len(resp))
	// ensure we get the correct transfer request back, correct amount etc...
	trans := resp[0].Transfer()
	assert.EqualValues(t, 20, trans.Amount.Amount.Uint64())
	// min = 15 so we go back to maintenance level
	assert.EqualValues(t, 15, trans.MinAmount.Uint64())
	assert.Equal(t, types.TransferType_TRANSFER_TYPE_MARGIN_LOW, trans.Type)
}

func testMarginTopup(t *testing.T) {
	eng := getTestEngine(t, nil)
	defer eng.ctrl.Finish()
	ctx, cfunc := context.WithCancel(context.Background())
	defer cfunc()
	evt := testMargin{
		party:   "trader1",
		size:    1,
		price:   1000,
		asset:   "ETH",
		margin:  10,     // required margin will be > 30 so ensure we don't have enough
		general: 100000, // plenty of balance for the transfer anyway
		market:  "ETH/DEC19",
	}
	eng.broker.EXPECT().Send(gomock.Any()).AnyTimes()
	eng.as.EXPECT().InAuction().AnyTimes().Return(false)
	eng.orderbook.EXPECT().GetCloseoutPrice(gomock.Any(), gomock.Any()).Times(1).
		DoAndReturn(func(volume uint64, side types.Side) (*num.Uint, error) {
			return markPrice.Clone(), nil
		})
	evts := []events.Margin{evt}
	resp := eng.UpdateMarginsOnSettlement(ctx, evts, markPrice)
	assert.Equal(t, 1, len(resp))
	// ensure we get the correct transfer request back, correct amount etc...
	trans := resp[0].Transfer()
	assert.EqualValues(t, 20, trans.Amount.Amount.Uint64())
	// min = 15 so we go back to maintenance level
	assert.EqualValues(t, 15, trans.MinAmount.Uint64())
	assert.Equal(t, types.TransferType_TRANSFER_TYPE_MARGIN_LOW, trans.Type)
}

func testMarginTopupOnOrderFailInsufficientFunds(t *testing.T) {
	eng := getTestEngine(t, nil)
	defer eng.ctrl.Finish()
	_, cfunc := context.WithCancel(context.Background())
	defer cfunc()
	evt := testMargin{
		party:   "trader1",
		size:    1,
		price:   1000,
		asset:   "ETH",
		margin:  10, // maring and general combined are not enough to get a sufficient margin
		general: 10,
		market:  "ETH/DEC19",
	}
	eng.as.EXPECT().InAuction().AnyTimes().Return(false)
	eng.orderbook.EXPECT().GetCloseoutPrice(gomock.Any(), gomock.Any()).Times(1).
		DoAndReturn(func(volume uint64, side types.Side) (*num.Uint, error) {
			return markPrice.Clone(), nil
		})
	riskevt, _, err := eng.UpdateMarginOnNewOrder(context.Background(), evt, markPrice)
	assert.Nil(t, riskevt)
	assert.NotNil(t, err)
	assert.Error(t, err, risk.ErrInsufficientFundsForMaintenanceMargin.Error())
}

func testMarginNoop(t *testing.T) {
	eng := getTestEngine(t, nil)
	defer eng.ctrl.Finish()
	eng.broker.EXPECT().Send(gomock.Any()).AnyTimes()
	ctx, cfunc := context.WithCancel(context.Background())
	defer cfunc()
	evt := testMargin{
		party:   "trader1",
		size:    1,
		price:   1000,
		asset:   "ETH",
		margin:  30,     // more than enough margin to cover the position, not enough to trigger transfer to general
		general: 100000, // plenty of balance for the transfer anyway
		market:  "ETH/DEC19",
	}
	eng.as.EXPECT().InAuction().AnyTimes().Return(false)
	eng.orderbook.EXPECT().GetCloseoutPrice(gomock.Any(), gomock.Any()).Times(1).
		DoAndReturn(func(volume uint64, side types.Side) (*num.Uint, error) {
			return markPrice.Clone(), nil
		})

	evts := []events.Margin{evt}
	resp := eng.UpdateMarginsOnSettlement(ctx, evts, markPrice)
	assert.Equal(t, 0, len(resp))
}

func testMarginOverflow(t *testing.T) {
	eng := getTestEngine(t, nil)
	defer eng.ctrl.Finish()
	eng.broker.EXPECT().Send(gomock.Any()).AnyTimes()
	ctx, cfunc := context.WithCancel(context.Background())
	defer cfunc()
	evt := testMargin{
		party:   "trader1",
		size:    1,
		price:   1000,
		asset:   "ETH",
		margin:  500,    // required margin will be > 35 (release), so ensure we don't have enough
		general: 100000, // plenty of balance for the transfer anyway
		market:  "ETH/DEC19",
	}
	eng.as.EXPECT().InAuction().Times(1).Return(false)
	// eng.as.EXPECT().InAuction().AnyTimes().Return(false)
	eng.orderbook.EXPECT().GetCloseoutPrice(gomock.Any(), gomock.Any()).Times(1).
		DoAndReturn(func(volume uint64, side types.Side) (*num.Uint, error) {
			return markPrice.Clone(), nil
		})
	evts := []events.Margin{evt}
	resp := eng.UpdateMarginsOnSettlement(ctx, evts, markPrice)
	assert.Equal(t, 1, len(resp))

	// ensure we get the correct transfer request back, correct amount etc...
	trans := resp[0].Transfer()
	assert.EqualValues(t, 470, trans.Amount.Amount.Uint64())
	// assert.Equal(t, riskMinamount-int64(evt.margin), trans.Amount.MinAmount)
	assert.Equal(t, types.TransferType_TRANSFER_TYPE_MARGIN_HIGH, trans.Type)
}

func testMarginOverflowAuctionEnd(t *testing.T) {
	eng := getTestEngine(t, nil)
	defer eng.ctrl.Finish()
	eng.broker.EXPECT().Send(gomock.Any()).AnyTimes()
	ctx, cfunc := context.WithCancel(context.Background())
	defer cfunc()
	evt := testMargin{
		party:   "trader1",
		size:    1,
		price:   1000,
		asset:   "ETH",
		margin:  500,    // required margin will be > 35 (release), so ensure we don't have enough
		general: 100000, // plenty of balance for the transfer anyway
		market:  "ETH/DEC19",
	}
	// we're still in auction...
	eng.as.EXPECT().InAuction().Times(1).Return(true)
	// but the auction is ending
	eng.as.EXPECT().CanLeave().Times(1).Return(true)
	// eng.as.EXPECT().InAuction().AnyTimes().Return(false)
	eng.orderbook.EXPECT().GetCloseoutPrice(gomock.Any(), gomock.Any()).Times(1).
		DoAndReturn(func(volume uint64, side types.Side) (*num.Uint, error) {
			return markPrice.Clone(), nil
		})
	evts := []events.Margin{evt}
	resp := eng.UpdateMarginsOnSettlement(ctx, evts, markPrice)
	assert.Equal(t, 1, len(resp))

	// ensure we get the correct transfer request back, correct amount etc...
	trans := resp[0].Transfer()
	assert.EqualValues(t, 470, trans.Amount.Amount.Uint64())
	// assert.Equal(t, riskMinamount-int64(evt.margin), trans.Amount.MinAmount)
	assert.Equal(t, types.TransferType_TRANSFER_TYPE_MARGIN_HIGH, trans.Type)
}

// implementation of the test from the specs
// https://github.com/vegaprotocol/product/blob/master/specs/0019-margin-calculator.md#pseudo-code--examples
func testMarginWithOrderInBook(t *testing.T) {
	// custom risk factors
	r := &types.RiskResult{
		RiskFactors: map[string]*types.RiskFactor{
			"ETH": {
				Market: "ETH/DEC19",
				Short:  num.DecimalFromFloat(.11),
				Long:   num.DecimalFromFloat(.10),
			},
		},
		PredictedNextRiskFactors: map[string]*types.RiskFactor{
			"ETH": {
				Market: "ETH/DEC19",
				Short:  num.DecimalFromFloat(.11),
				Long:   num.DecimalFromFloat(.10),
			},
		},
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

		{volume: 1, price: num.NewUint(120), tid: "t4", side: types.Side_SIDE_BUY},
		{volume: 4, price: num.NewUint(110), tid: "t5", side: types.Side_SIDE_BUY},
		{volume: 5, price: num.NewUint(108), tid: "t6", side: types.Side_SIDE_BUY},
	}

	marketID := "testingmarket"

	conf := config.NewDefaultConfig("")
	log := logging.NewTestLogger()
	ctrl := gomock.NewController(t)
	model := mocks.NewMockModel(ctrl)
	broker := bmock.NewMockBroker(ctrl)
	as := mocks.NewMockAuctionState(ctrl)
	broker.EXPECT().Send(gomock.Any()).AnyTimes()

	// instantiate the book then fill it with the orders

	book := matching.NewOrderBook(log, conf.Matching, marketID, false)

	for _, v := range ordersInBook {
		o := &types.Order{
			Id:          fmt.Sprintf("o-%v-%v", v.tid, marketID),
			MarketId:    marketID,
			PartyId:     "A",
			Side:        v.side,
			Price:       v.price.Clone(),
			Size:        uint64(v.volume),
			Remaining:   uint64(v.volume),
			TimeInForce: types.Order_TIME_IN_FORCE_GTT,
			Type:        types.Order_TYPE_LIMIT,
			Status:      types.Order_STATUS_ACTIVE,
			ExpiresAt:   10000,
		}
		_, err := book.SubmitOrder(o)
		assert.Nil(t, err)
	}

	as.EXPECT().InAuction().AnyTimes().Return(false)
	testE := risk.NewEngine(log, conf.Risk, mc, model, r, book, as, broker, 0, "mktid")
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
	riskevt, _, err := testE.UpdateMarginOnNewOrder(context.Background(), evt, markPrice.Clone())
	assert.NotNil(t, riskevt)
	if riskevt == nil {
		t.Fatal("expecting non nil risk update")
	}
	assert.Nil(t, err)
	margins := riskevt.MarginLevels()
	assert.EqualValues(t, 542, margins.MaintenanceMargin.Uint64())
	assert.Equal(t, uint64(542*mc.ScalingFactors.SearchLevel.Float64()), margins.SearchLevel.Uint64())
	assert.Equal(t, uint64(542*mc.ScalingFactors.InitialMargin), margins.InitialMargin.Uint64())
	assert.Equal(t, uint64(542*mc.ScalingFactors.CollateralRelease), margins.CollateralReleaseLevel.Uint64())
}

// testcase 1 from: https://drive.google.com/file/d/1B8-rLK2NB6rWvjzZX9sLtqOQzLz8s2ky/view
func testMarginWithOrderInBook2(t *testing.T) {
	// custom risk factors
	r := &types.RiskResult{
		RiskFactors: map[string]*types.RiskFactor{
			"ETH": {
				Market: "ETH/DEC19",
				Short:  num.DecimalFromFloat(.2),
				Long:   num.DecimalFromFloat(.1),
			},
		},
		PredictedNextRiskFactors: map[string]*types.RiskFactor{
			"ETH": {
				Market: "ETH/DEC19",
				Short:  num.DecimalFromFloat(.2),
				Long:   num.DecimalFromFloat(.1),
			},
		},
	}
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
		{volume: 100, price: num.NewUint(250), tid: "t1", side: types.Side_SIDE_SELL},
		{volume: 11, price: num.NewUint(140), tid: "t2", side: types.Side_SIDE_SELL},
		{volume: 2, price: num.NewUint(112), tid: "t3", side: types.Side_SIDE_SELL},
		// bids
		{volume: 1, price: num.NewUint(100), tid: "t4", side: types.Side_SIDE_BUY},
		{volume: 3, price: num.NewUint(96), tid: "t5", side: types.Side_SIDE_BUY},
		{volume: 15, price: num.NewUint(90), tid: "t6", side: types.Side_SIDE_BUY},
		{volume: 50, price: num.NewUint(87), tid: "t7", side: types.Side_SIDE_BUY},
	}

	marketID := "testingmarket"

	conf := config.NewDefaultConfig("")
	log := logging.NewTestLogger()
	ctrl := gomock.NewController(t)
	model := mocks.NewMockModel(ctrl)
	broker := bmock.NewMockBroker(ctrl)
	as := mocks.NewMockAuctionState(ctrl)
	broker.EXPECT().Send(gomock.Any()).AnyTimes()

	as.EXPECT().InAuction().AnyTimes().Return(false)
	// instantiate the book then fill it with the orders

	book := matching.NewOrderBook(log, conf.Matching, marketID, false)

	for _, v := range ordersInBook {
		o := &types.Order{
			Id:          fmt.Sprintf("o-%v-%v", v.tid, marketID),
			MarketId:    marketID,
			PartyId:     "A",
			Side:        v.side,
			Price:       v.price.Clone(),
			Size:        uint64(v.volume),
			Remaining:   uint64(v.volume),
			TimeInForce: types.Order_TIME_IN_FORCE_GTT,
			Type:        types.Order_TYPE_LIMIT,
			Status:      types.Order_STATUS_ACTIVE,
			ExpiresAt:   10000,
		}
		_, err := book.SubmitOrder(o)
		assert.Nil(t, err)
	}

	testE := risk.NewEngine(log, conf.Risk, mc, model, r, book, as, broker, 0, "mktid")
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

	riskevt, _, err := testE.UpdateMarginOnNewOrder(context.Background(), evt, previousMarkPrice)
	assert.NotNil(t, riskevt)
	if riskevt == nil {
		t.Fatal("expecting non nil risk update")
	}
	assert.Nil(t, err)
	margins := riskevt.MarginLevels()
	assert.Equal(t, uint64(277), margins.MaintenanceMargin.Uint64())
	assert.Equal(t, uint64(277*mc.ScalingFactors.SearchLevel.Float64()), margins.SearchLevel.Uint64())
	assert.Equal(t, uint64(277*mc.ScalingFactors.InitialMargin.Float64()), margins.InitialMargin.Uint64())
	assert.Equal(t, uint64(277*mc.ScalingFactors.CollateralRelease.Float64()), margins.CollateralReleaseLevel.Uint64())
}

func getTestEngine(t *testing.T, initialRisk *types.RiskResult) *testEngine {
	if initialRisk == nil {
		cpy := riskResult
		initialRisk = &cpy // this is just a shallow copy, so might be worth creating a deep copy depending on the test
	}
	ctrl := gomock.NewController(t)
	model := mocks.NewMockModel(ctrl)
	conf := risk.NewDefaultConfig()
	ob := mocks.NewMockOrderbook(ctrl)
	broker := bmock.NewMockBroker(ctrl)
	as := mocks.NewMockAuctionState(ctrl)

	engine := risk.NewEngine(
		logging.NewTestLogger(),
		conf,
		getMarginCalculator(),
		model,
		initialRisk,
		ob,
		as,
		broker,
		0,
		"mktid",
	)
	return &testEngine{
		Engine:    engine,
		ctrl:      ctrl,
		model:     model,
		orderbook: ob,
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
	return num.NewUint(0)
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

func (m testMargin) VWBuy() *num.Uint {
	return num.NewUint(m.vwBuy)
}

func (m testMargin) VWSell() *num.Uint {
	return num.NewUint(m.vwSell)
}

func (m testMargin) ClearPotentials() {}

func (m testMargin) Transfer() *types.Transfer {
	return m.transfer
}

func (m testMargin) MarginShortFall() *num.Uint {
	return num.NewUint(m.marginShortFall)
}
