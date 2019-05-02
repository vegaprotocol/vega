package settlement_test

import (
	"sync"
	"testing"

	"code.vegaprotocol.io/vega/internal/engines/settlement"
	"code.vegaprotocol.io/vega/internal/logging"
	types "code.vegaprotocol.io/vega/proto"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

type testEngine struct {
	*settlement.Engine
	ctrl *gomock.Controller
}

func TestMarkToMarket(t *testing.T) {
	t.Run("No settle positions if none were on channel", testMarkToMarketEmpty)
	t.Run("Settle positions are pushed onto the slice channel in order", testMarkToMarketOrdered)
}

func testMarkToMarketEmpty(t *testing.T) {
	ch := make(chan *types.SettlePosition, 10)
	engine := getTestEngine(t)
	defer engine.ctrl.Finish()
	settleCh := engine.MarkToMarket(ch)
	close(ch)
	result := <-settleCh
	assert.Empty(t, result)
}

func testMarkToMarketOrdered(t *testing.T) {
	// data is pused in the wrong order
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
	wg := sync.WaitGroup{}
	wg.Add(1)
	ch := make(chan *types.SettlePosition, 2)
	go func() {
		for _, d := range data {
			ch <- d
		}
		wg.Done()
	}()
	engine := getTestEngine(t)
	defer engine.ctrl.Finish()
	settleCh := engine.MarkToMarket(ch)
	wg.Wait()
	close(ch)
	result := <-settleCh
	// ensure we get the data we expect, in the correct order
	assert.Equal(t, len(data), len(result))
	assert.Equal(t, data[0], result[1])
	assert.Equal(t, data[1], result[0])
}

func getTestEngine(t *testing.T) *testEngine {
	ctrl := gomock.NewController(t)
	conf := settlement.NewDefaultConfig()
	return &testEngine{
		Engine: settlement.New(logging.NewTestLogger(), conf),
		ctrl:   ctrl,
	}
}
