package settlement_test

import (
	"fmt"
	"sync"
	"testing"

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
	positions []*mocks.MockMarketPosition
}

func TestMarkToMarket(t *testing.T) {
	t.Run("No settle positions if none were on channel", testMarkToMarketEmpty)
	t.Run("Settle positions are pushed onto the slice channel in order", testMarkToMarketOrdered)
}

func testMarkToMarketEmpty(t *testing.T) {
	trade := &types.Trade{
		Price: 10000,
		Size:  1, // for now, keep volume to 1, it's tricky to calculate the old position if not
	}
	ch := make(chan settlement.MarketPosition, 10)
	engine := getTestEngine(t)
	defer engine.ctrl.Finish()
	settleCh := engine.SettleMTM(trade, ch)
	close(ch)
	result := <-settleCh
	assert.Empty(t, result)
}

func testMarkToMarketOrdered(t *testing.T) {
	// data is pused in the wrong order
	trade := &types.Trade{
		Price: 10000,
		Size:  1, // for now, keep volume to 1, it's tricky to calculate the old position if not
	}
	data := []*types.SettlePosition{
		{
			Owner: "trader1",
			Size:  1,
			Amount: &types.FinancialAmount{
				Amount: 100,
			},
			Type: types.SettleType_MTM_WIN,
		},
		{
			Owner: "trader1",
			Size:  1,
			Amount: &types.FinancialAmount{
				Amount: -100,
			},
			Type: types.SettleType_MTM_LOSS,
		},
	}
	engine := getTestEngine(t)
	defer engine.Finish()
	// set up test data
	engine.getTestPositions(trade, data)
	// channel only needs to be buffered for half
	// both tests should return the same values, essentially
	positions := [][]*mocks.MockMarketPosition{
		engine.positions[:len(data)],
		engine.positions[len(data):],
	}
	// responses := make([]*types.SettlePosition, 2)
	wg := sync.WaitGroup{}
	for i, pos := range positions {
		tstSet := fmt.Sprintf("positions slice %d", i)
		wg.Add(1)
		ch := make(chan settlement.MarketPosition, len(pos))
		go func() {
			for _, p := range pos {
				ch <- p
			}
			wg.Done()
		}()
		settleCh := engine.SettleMTM(trade, ch)
		wg.Wait()
		close(ch)
		result := <-settleCh
		assert.Equal(t, len(data), len(result))
		assert.Equal(t, data[0].Type, result[1].Type)
		assert.Equal(t, data[0].Amount.Amount, result[1].Amount.Amount)
		assert.Equal(t, data[1].Type, result[0].Type, tstSet)
		assert.Equal(t, data[1].Amount.Amount, result[0].Amount.Amount, tstSet)
	}
	// ensure we get the data we expect, in the correct order
}

func (te *testEngine) getTestPositions(trade *types.Trade, data []*types.SettlePosition) {
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
		if sp.Type == types.SettleType_MTM_WIN {
			// current position to get win with pos of +1 trade.Price - settlePosition == position price
			posPrice := uint64(int64(trade.Price) - sp.Amount.Amount)
			long.EXPECT().Price().MinTimes(1).Return(posPrice)
			posPrice = trade.Price + uint64(sp.Amount.Amount)
			short.EXPECT().Price().MinTimes(1).Return(posPrice)
		} else {
			// position is long, to get a loss, we need position price > trade.Price (market price has gone down)
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

// Finish - call finish on controller, remove test state (positions)
func (te *testEngine) Finish() {
	te.ctrl.Finish()
	te.positions = nil
}

func getTestEngine(t *testing.T) *testEngine {
	ctrl := gomock.NewController(t)
	conf := settlement.NewDefaultConfig()
	return &testEngine{
		Engine: settlement.New(logging.NewTestLogger(), conf),
		ctrl:   ctrl,
	}
}
