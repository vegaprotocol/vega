package settlement_test

import (
	"errors"
	"sync"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/logging"
	types "code.vegaprotocol.io/vega/proto"
	"code.vegaprotocol.io/vega/settlement"
	"code.vegaprotocol.io/vega/settlement/mocks"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

type testEngine struct {
	*settlement.Engine
	ctrl      *gomock.Controller
	prod      *mocks.MockProduct
	positions []*mocks.MockMarketPosition
	market    string
}

type posValue struct {
	trader string
	price  uint64 // absolute Mark price
	size   int64
}

func TestMarkToMarket(t *testing.T) {
	t.Run("Settle at market expiry - success", testSettleExpiredSuccess)
	t.Run("Settle at market expiry - error", testSettleExpiryFail)
	t.Run("No settle positions if none were on channel", testMarkToMarketEmpty)
	// t.Run("Settle positions are pushed onto the slice channel in order", testMarkToMarketOrdered)
	// -- MTM -> special case for traders getting MTM before changing positions, and trade introducing new trader
	// TODO Add a test for long <-> short trades, for now we've covered the basics
	// t.Run("Settle MTM on a market with long trader going short and short trader going long", testMTMSwitchPosition)
	// t.Run("Settle MTM with new and existing trader position combo", testMTMPrefixTradePositions)
}

func testSettleExpiredSuccess(t *testing.T) {
	engine := getTestEngine(t)
	defer engine.Finish()
	// these are mark prices, product will provide the actual value
	data := []posValue{ // {{{2
		{
			trader: "trader1",
			price:  1000,
			size:   10,
		},
		{
			trader: "trader2",
			price:  1000,
			size:   -5,
		},
		{
			trader: "trader3",
			price:  1000,
			size:   -5,
		},
	}
	expect := []*types.Transfer{
		{
			Owner: data[1].trader,
			Size:  1,
			Amount: &types.FinancialAmount{
				Amount: -500,
			},
			Type: types.TransferType_LOSS,
		},
		{
			Owner: data[2].trader,
			Size:  1,
			Amount: &types.FinancialAmount{
				Amount: -500,
			},
			Type: types.TransferType_LOSS,
		},
		{
			Owner: data[0].trader,
			Size:  1,
			Amount: &types.FinancialAmount{
				Amount: 1000,
			},
			Type: types.TransferType_WIN,
		},
	} // }}}
	oraclePrice := uint64(1100)
	settleF := func(price uint64, size int64) (*types.FinancialAmount, error) {
		sp := int64((oraclePrice - price)) * size
		return &types.FinancialAmount{
			Amount: sp,
		}, nil
	}
	positions := engine.getExpiryPositions(data...)
	for _, d := range data {
		// we expect settle calls for each position
		engine.prod.EXPECT().Settle(d.price, d.size).Times(1).DoAndReturn(settleF)
	}
	// ensure positions are set
	engine.Update(positions)
	// now settle:
	got, err := engine.Settle(time.Now())
	assert.NoError(t, err)
	assert.Equal(t, len(expect), len(got))
	for i, p := range got {
		e := expect[i]
		assert.Equal(t, e.Size, p.Size)
		assert.Equal(t, e.Type, p.Type)
		assert.Equal(t, e.Amount.Amount, p.Amount.Amount)
	}
}

func testSettleExpiryFail(t *testing.T) {
	engine := getTestEngine(t)
	defer engine.Finish()
	// these are mark prices, product will provide the actual value
	data := []posValue{
		{
			trader: "trader1",
			price:  1000,
			size:   10,
		},
	}
	errExp := errors.New("product.Settle error")
	positions := engine.getExpiryPositions(data...)
	engine.prod.EXPECT().Settle(data[0].price, data[0].size).Times(1).Return(nil, errExp)
	engine.Update(positions)
	empty, err := engine.Settle(time.Now())
	assert.Empty(t, empty)
	assert.Error(t, err)
	assert.Equal(t, errExp, err)
}

func testMarkToMarketEmpty(t *testing.T) {
	markPrice := uint64(10000)
	// there's only 1 trade to test here
	data := posValue{
		price:  markPrice,
		size:   1,
		trader: "test",
	}
	engine := getTestEngine(t)
	defer engine.Finish()
	pos := mocks.NewMockMarketPosition(engine.ctrl)
	pos.EXPECT().Party().AnyTimes().Return(data.trader)
	pos.EXPECT().Size().AnyTimes().Return(data.size)
	pos.EXPECT().Price().AnyTimes().Return(markPrice)
	engine.Update([]events.MarketPosition{pos})
	result := engine.SettleMTM(markPrice, []events.MarketPosition{pos})
	assert.Empty(t, result)
}

// @TODO this test needs to be rewritten, it's rubbish
func testMarkToMarketOrdered(t *testing.T) {
	engine := getTestEngine(t)
	defer engine.Finish()
	positions := []posValue{
		{
			price:  10000,
			size:   1,
			trader: "trader1", // mocks will create 2 traders (long & short)
		},
		{
			price:  10000,
			size:   -1,
			trader: "trader2",
		},
	}
	markPrice := uint64(10000 + 1000)
	init := make([]events.MarketPosition, 0, 3)
	long := make([]events.MarketPosition, 0, 3)
	short := make([]events.MarketPosition, 0, 3)
	neutral := mocks.NewMockMarketPosition(engine.ctrl)
	neutral.EXPECT().Price().AnyTimes().Return(uint64(10000))
	neutral.EXPECT().Party().AnyTimes().Return("neutral")
	neutral.EXPECT().Size().AnyTimes().Return(int64(5))
	init = append(init, neutral)
	for _, p := range positions {
		m := mocks.NewMockMarketPosition(engine.ctrl)
		m.EXPECT().Size().AnyTimes().Return(p.size)
		m.EXPECT().Party().AnyTimes().Return(p.trader)
		m.EXPECT().Price().AnyTimes().Return(p.price)
		init = append(init, m)
		l := mocks.NewMockMarketPosition(engine.ctrl)
		l.EXPECT().Size().AnyTimes().Return(p.size * 2)
		l.EXPECT().Price().AnyTimes().Return(markPrice)
		l.EXPECT().Party().AnyTimes().Return(p.trader)
		long = append(long, l)
		s := mocks.NewMockMarketPosition(engine.ctrl)
		s.EXPECT().Size().AnyTimes().Return(p.size * -2) // long trader is going short, short trader is going long
		s.EXPECT().Price().AnyTimes().Return(markPrice)
		s.EXPECT().Party().AnyTimes().Return(p.trader)
		short = append(short, s)
	}
	engine.Update(init) // setup the initial state
	engine.Update(long)
	// add neutral to position, this hasn't changed, but we need it processed anyway
	long = append(long, neutral)
	longTransfer := engine.SettleMTM(markPrice, long)
	assert.NotEmpty(t, longTransfer)
	t.Logf("%#v\n", longTransfer)
	// now, let's update the state again as if the settlement hasn't happened
	engine.Update(init)
	engine.Update(short)
	short = append(short, neutral)
	shortTransfer := engine.SettleMTM(markPrice, short)
	assert.NotEmpty(t, shortTransfer)
	assert.Equal(t, 3, len(shortTransfer)) // all 3 traders should get updated
}

// @TODO this test makes no sense, rewrite needed (update through trade)
func testMTMSwitchPosition(t *testing.T) {
	engine := getTestEngine(t)
	defer engine.Finish()
	start := []posValue{
		{
			trader: "trader1",
			size:   5,
			price:  10000,
		},
		{
			trader: "trader2",
			size:   -5,
			price:  10000,
		},
		{
			trader: "neutral",
			size:   3,
			price:  10000,
		},
		{
			trader: "closed",
			size:   0,
			price:  10000,
		},
	}
	update := []posValue{
		{
			trader: "trader1",
			size:   -1,
			price:  11000,
		},
		{
			trader: "trader2",
			size:   1,
			price:  11000,
		},
	}
	final := []posValue{
		{
			trader: "trader1",
			size:   -1,
			price:  11000,
		},
		{
			trader: "trader2",
			size:   1,
			price:  11000,
		},
		{
			trader: "neutral",
			size:   3,
			price:  11000,
		},
		{
			trader: "closed",
			size:   0,
			price:  11000,
		},
	}
	_, init := engine.getMockMarketPositions(start)
	_, change := engine.getMockMarketPositions(update)
	_, positions := engine.getMockMarketPositions(final)
	// set the initial state
	engine.Update(init)
	engine.Update(change)
	result := engine.SettleMTM(final[0].price, positions)
	assert.NotEmpty(t, result)
	assert.Equal(t, 3, len(result)) // one for each trader with an open position
}

// {{{
func (te *testEngine) getExpiryPositions(positions ...posValue) []events.MarketPosition {
	te.positions = make([]*mocks.MockMarketPosition, 0, len(positions))
	mpSlice := make([]events.MarketPosition, 0, len(positions))
	for _, p := range positions {
		pos := mocks.NewMockMarketPosition(te.ctrl)
		// these values should only be obtained once, and assigned internally
		pos.EXPECT().Party().MinTimes(1).AnyTimes().Return(p.trader)
		pos.EXPECT().Size().MinTimes(1).AnyTimes().Return(p.size)
		pos.EXPECT().Price().Times(1).Return(p.price)
		te.positions = append(te.positions, pos)
		mpSlice = append(mpSlice, pos)
	}
	return mpSlice
}

func (te *testEngine) getMockMarketPositions(data []posValue) ([]settlement.MarketPosition, []events.MarketPosition) {
	raw, evts := make([]settlement.MarketPosition, 0, len(data)), make([]events.MarketPosition, 0, len(data))
	for _, pos := range data {
		mock := mocks.NewMockMarketPosition(te.ctrl)
		mock.EXPECT().Party().MinTimes(1).Return(pos.trader)
		mock.EXPECT().Size().MinTimes(1).Return(pos.size)
		mock.EXPECT().Price().MinTimes(1).Return(pos.price)
		raw = append(raw, mock)
		evts = append(evts, mock)
	}
	return raw, evts
}

func TestConcurrent(t *testing.T) {
	const N = 10

	engine := getTestEngine(t)
	defer engine.Finish()
	engine.prod.EXPECT().Settle(gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(func(price uint64, size int64) (*types.FinancialAmount, error) {
		return &types.FinancialAmount{Amount: 0}, nil
	})

	cfg := engine.Config
	cfg.Level.Level = logging.DebugLevel
	engine.ReloadConf(cfg)
	cfg.Level.Level = logging.InfoLevel
	engine.ReloadConf(cfg)

	var wg sync.WaitGroup

	now := time.Now()
	wg.Add(N * 3)
	for i := 0; i < N; i++ {
		data := []posValue{
			{
				trader: "testtrader1",
				price:  1234,
				size:   100,
			},
			{
				trader: "testtrader2",
				price:  1235,
				size:   0,
			},
		}
		_, evts := engine.getMockMarketPositions(data)

		go func() {
			defer wg.Done()
			// Update requires posMu
			engine.Update(evts)
		}()
		go func() {
			defer wg.Done()
			// RemoveDistressed requires posMu and closedMu
			engine.RemoveDistressed(evts)
		}()
		go func() {
			defer wg.Done()
			// Settle requires posMu
			_, err := engine.Settle(now)
			assert.NoError(t, err)
		}()
	}

	wg.Wait()
}

// Finish - call finish on controller, remove test state (positions)
func (te *testEngine) Finish() {
	te.ctrl.Finish()
	te.positions = nil
}

func getTestEngine(t *testing.T) *testEngine {
	ctrl := gomock.NewController(t)
	conf := settlement.NewDefaultConfig()
	prod := mocks.NewMockProduct(ctrl)
	market := "BTC/DEC19"
	prod.EXPECT().GetAsset().AnyTimes().Do(func() string { return "BTC" })
	return &testEngine{
		Engine:    settlement.New(logging.NewTestLogger(), conf, prod, market),
		ctrl:      ctrl,
		prod:      prod,
		positions: nil,
		market:    market,
	}
} // }}}

//  vim: set ts=4 sw=4 tw=0 foldlevel=1 foldmethod=marker noet :
