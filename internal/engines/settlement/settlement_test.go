package settlement_test

import (
	"errors"
	"sync"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/internal/engines/settlement"
	"code.vegaprotocol.io/vega/internal/engines/settlement/mocks"
	"code.vegaprotocol.io/vega/internal/logging"
	types "code.vegaprotocol.io/vega/proto"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

type testEngine struct {
	*settlement.Engine
	ctrl      *gomock.Controller
	prod      *mocks.MockProduct
	positions []*mocks.MockMarketPosition
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
	t.Run("Settle positions are pushed onto the slice channel in order", testMarkToMarketOrdered)
	// -- MTM -> special case for traders getting MTM before changing positions, and trade introducing new trader
	// while existing traders still should get their MTM position updated
	t.Run("Settle MTM with new and existing trader position combo", testMTMPrefixTradePositions)
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
	trade := &types.Trade{
		Price: 10000,
		Size:  1, // for now, keep volume to 1, it's tricky to calculate the old position if not
	}
	ch := make(chan settlement.MarketPosition, 10)
	engine := getTestEngine(t)
	defer engine.Finish()
	settleCh := engine.SettleMTM(*trade, trade.Price, ch)
	close(ch)
	result := <-settleCh
	assert.Empty(t, result)
}

func testMarkToMarketOrdered(t *testing.T) {
	// this is the mark price
	trade := &types.Trade{ // {{{2
		Price: 10000,
		Size:  1,
	}
	// this is the trade we're using to trigger the change
	tradeArg := &types.Trade{
		Price:  trade.Price * 2,
		Size:   1,
		Buyer:  "trader1",
		Seller: "trader2",
	}
	data := []*types.Transfer{
		{
			Owner: "trader1",
			Size:  1,
			Amount: &types.FinancialAmount{
				Amount: 100,
			},
			Type: types.TransferType_MTM_WIN,
		},
		{
			Owner: "trader2",
			Size:  1,
			Amount: &types.FinancialAmount{
				Amount: -100,
			},
			Type: types.TransferType_MTM_LOSS,
		},
	} // }}}
	engine := getTestEngine(t)
	defer engine.Finish()
	// get initial positions (short && long)
	engine.getTestPositions(trade, data)
	positions := [][]*mocks.MockMarketPosition{
		engine.positions[:len(data)],
		engine.positions[len(data):],
	}
	// so 1 is wins 100 on MTM, the other loses 100 on MTM
	// BUT they also traded at double market price
	wg := sync.WaitGroup{}
	for _, pos := range positions {
		update := make([]settlement.MarketPosition, 0, len(positions[0]))
		for _, p := range pos {
			update = append(update, p)
		}
		engine.Update(update)
		wg.Add(1)
		ch := make(chan settlement.MarketPosition, len(pos))
		go func() {
			for _, p := range pos {
				ch <- p
			}
			wg.Done()
		}()
		// tradeArg has a different price compared to mark price here
		// this should *not* affect the output
		settleCh := engine.SettleMTM(*tradeArg, trade.Price, ch)
		wg.Wait()
		close(ch)
		result := <-settleCh
		// length is always 4
		assert.Equal(t, len(data)*2, len(result))
		// start with losses, end with wins
		assert.Equal(t, types.TransferType_MTM_LOSS, result[0].Transfer().Type)
		assert.Equal(t, types.TransferType_MTM_WIN, result[len(result)-1].Transfer().Type)
		var sum int64
		for _, r := range result {
			sum += r.Transfer().Amount.Amount
		}
		// this all balances out
		assert.Zero(t, sum)
		// assert.Equal(t, data[0].Type, result[1].Type)
		// assert.Equal(t, data[0].Amount.Amount, result[1].Amount.Amount)
		assert.Equal(t, data[1].Type, result[0].Transfer().Type, pos)
		// assert.Equal(t, data[1].Amount.Amount, result[0].Amount.Amount, tstSet)
	}
}

func testMTMPrefixTradePositions(t *testing.T) {
	engine := getTestEngine(t)
	defer engine.Finish()
	// setup {{{2
	trade := &types.Trade{
		Size:   5,
		Price:  1000,
		Buyer:  "trader1", // trader holding long position of 5@900
		Seller: "trader3", // new trader, going short @1000
	}
	// these are the initial positions, long 5@900, short 5@900
	startPos := []posValue{
		{
			trader: "trader1",
			price:  900,
			size:   5,
		},
		{
			trader: "trader2",
			price:  900,
			size:   -5,
		},
	}
	startState := engine.getExpiryPositions(startPos...)
	// initial positions for traders 1 & 2 (omitting trader 3, because that's coming from a trade-based MTM)
	// these should be set before we actually run the test
	engine.Update(startState)
	// data at the end, after a trade with new trader
	data := []posValue{
		{
			trader: "trader1",
			price:  1000,
			size:   10,
		},
		{
			trader: "trader2", // at close, this trader should end up at 1000
			price:  1000,
			size:   -5,
		},
		{
			trader: "trader3",
			price:  1000,
			size:   -5,
		},
	}
	// call to settlePreTrade won't include trader2 entry here
	preTrade := []*types.Transfer{
		{
			Owner: startPos[1].trader,
			Size:  1,
			Amount: &types.FinancialAmount{
				Amount: -500, // was 5 short at 900, trade boosts price to 1000, so this trader loses 5*-100
			},
			Type: types.TransferType_MTM_LOSS,
		},
		{
			Owner: startPos[0].trader,
			Size:  1,
			Amount: &types.FinancialAmount{
				Amount: 500, // was 5 long at 900, trade boosts to 1000 => 5*100
			},
			Type: types.TransferType_MTM_WIN,
		},
	}
	expiry := []*types.Transfer{
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
	}
	// }}}
	oraclePrice := uint64(1100)
	settleF := func(price uint64, size int64) (*types.FinancialAmount, error) {
		sp := int64((oraclePrice - price)) * size
		return &types.FinancialAmount{
			Amount: sp,
		}, nil
	}
	for _, d := range data {
		// we expect settle calls for each position
		// engine.prod.EXPECT().Settle(d.price, d.size).Times(1).DoAndReturn(settleF)
		engine.prod.EXPECT().Settle(d.price, d.size).Times(1).DoAndReturn(settleF)
	}
	// now let's set trader2 to still be at old mark price (900)
	data[1] = startPos[1]
	// these will be the positions *after* trade, we apply the MTM for the trade beforehand
	positions := engine.getExpiryPositions(data...)
	wg := sync.WaitGroup{}
	wg.Add(1)
	ch := make(chan settlement.MarketPosition, len(positions))
	go func() {
		for _, p := range positions {
			ch <- p
		}
		wg.Done()
	}()
	settleCh := engine.SettleMTM(*trade, trade.Price, ch)
	wg.Wait()
	close(ch)
	mtm := <-settleCh
	assert.NotEmpty(t, mtm)
	assert.Equal(t, len(preTrade), len(mtm))
	// ensure we get the expected Transfers (includes trader 1 and trader 2)
	for i, m := range mtm {
		assert.Equal(t, preTrade[i].Owner, m.Transfer().Owner)
		assert.Equal(t, preTrade[i].Type, m.Transfer().Type)
		assert.Equal(t, preTrade[i].Amount.Amount, m.Transfer().Amount.Amount)
	}
	// assert.Equal(t, len(preTrade), len(mtm))
	// now settle:
	got, err := engine.Settle(time.Now())
	assert.NoError(t, err)
	assert.Equal(t, len(expiry), len(got))
	for i, p := range got {
		e := expiry[i]
		assert.Equal(t, e.Size, p.Size)
		assert.Equal(t, e.Type, p.Type)
		assert.Equal(t, e.Amount.Amount, p.Amount.Amount)
	}
}

// Setup functions, makes mocking easier {{{
func (te *testEngine) getTestPositions(trade *types.Trade, data []*types.Transfer) {
	// positions double data -> wins and losses for both long and short positions
	te.positions = make([]*mocks.MockMarketPosition, 0, len(data)*2)
	shortPos := make([]*mocks.MockMarketPosition, 0, len(data))
	for _, sp := range data {
		// set up long mock
		long := mocks.NewMockMarketPosition(te.ctrl)
		short := mocks.NewMockMarketPosition(te.ctrl)
		long.EXPECT().Party().MinTimes(1).Return(sp.Owner)
		short.EXPECT().Party().MinTimes(1).Return(sp.Owner)
		// ensure we're always returning a positive amount at least once
		// and a negative one, so all tests test both possibilities (long and short)
		long.EXPECT().Size().MinTimes(1).Return(int64(1))
		short.EXPECT().Size().MinTimes(1).Return(int64(-1))
		if sp.Type == types.TransferType_MTM_WIN {
			// current position to get win with pos of +1 trade.Price - settlePosition == position price
			posPrice := uint64(int64(trade.Price) - sp.Amount.Amount)
			long.EXPECT().Price().MinTimes(1).Return(posPrice)
			posPrice = trade.Price + uint64(sp.Amount.Amount)
			short.EXPECT().Price().MinTimes(1).Return(posPrice)
		} else {
			// position is long, to get a loss, we need position price > trade.Price (mark price has gone down)
			// amount is negative -> trade.Price - (neg amount) == trade.Price + amount -> old price was greater
			posPrice := uint64(int64(trade.Price) - sp.Amount.Amount)
			long.EXPECT().Price().MinTimes(1).Return(posPrice)
			// long position -> price was lower to begin with => bad news for short
			posPrice = uint64(int64(trade.Price) + sp.Amount.Amount)
			// posPrice = trade.Price + uint64(-sp.Amount.Amount)
			short.EXPECT().Price().MinTimes(1).Return(posPrice)
		}
		// long test first
		te.positions = append(te.positions, long)
		// append short at the end
		shortPos = append(shortPos, short)
		// append long and short examples
	}
	te.positions = append(te.positions, shortPos...)
}

func (te *testEngine) getExpiryPositions(positions ...posValue) []settlement.MarketPosition {
	te.positions = make([]*mocks.MockMarketPosition, 0, len(positions))
	mpSlice := make([]settlement.MarketPosition, 0, len(positions))
	for _, p := range positions {
		pos := mocks.NewMockMarketPosition(te.ctrl)
		// these values should only be obtained once, and assigned internally
		pos.EXPECT().Party().MinTimes(1).MaxTimes(2).Return(p.trader)
		pos.EXPECT().Size().MinTimes(1).MaxTimes(2).Return(p.size)
		pos.EXPECT().Price().Times(1).Return(p.price)
		te.positions = append(te.positions, pos)
		mpSlice = append(mpSlice, pos)
	}
	return mpSlice
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
	return &testEngine{
		Engine:    settlement.New(logging.NewTestLogger(), conf, prod),
		ctrl:      ctrl,
		prod:      prod,
		positions: nil,
	}
} // }}}

//  vim: set ts=4 sw=4 tw=0 foldlevel=1 foldmethod=marker noet :
