package plugins_test

// No race condition checks on these tests, the channels are buffered to avoid actual issues
// we are aware that the tests themselves can be written in an unsafe way, but that's the tests
// not the code itsel. The behaviour of the tests is 100% reliable
import (
	"context"
	"testing"

	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/plugins"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

type tradeStub struct {
	size  int64
	price uint64
}

type posPluginTst struct {
	*plugins.Positions
	ctrl  *gomock.Controller
	ctx   context.Context
	cfunc context.CancelFunc
}

func TestMultipleTradesOfSameSize(t *testing.T) {
	position := getPosPlugin(t)
	defer position.Finish()
	market := "market-id"
	ps := events.NewSettlePositionEvent(position.ctx, "trader1", market, 1000, []events.TradeSettlement{
		tradeStub{
			size:  -1,
			price: 1000,
		},
		tradeStub{
			size:  -1,
			price: 1000,
		},
	}, 1)
	position.Push(ps)
	pp, err := position.GetPositionsByMarket(market)
	assert.NoError(t, err)
	assert.NotZero(t, len(pp))
	// average entry price should be 1k
	assert.Equal(t, ps.Price(), pp[0].AverageEntryPrice)
}

func TestMultipleTradesAndLossSocializationTraderNoOpenVolume(t *testing.T) {
	position := getPosPlugin(t)
	defer position.Finish()
	market := "market-id"
	ps := events.NewSettlePositionEvent(position.ctx, "trader1", market, 1000, []events.TradeSettlement{
		tradeStub{
			size:  2,
			price: 1000,
		},
		tradeStub{
			size:  -2,
			price: 1500,
		},
	}, 1)
	position.Push(ps)
	pp, err := position.GetPositionsByMarket(market)
	assert.NoError(t, err)
	assert.NotZero(t, len(pp))
	// average entry price should be 1k
	// initialy calculation say the RealisedPNL should be 1000
	assert.Equal(t, 1000, int(pp[0].RealisedPNL))

	// then we process the event for LossSocialization
	lsevt := events.NewLossSocializationEvent(position.ctx, "trader1", market, -300, 1)
	position.Push(lsevt)
	pp, err = position.GetPositionsByMarket(market)
	assert.NoError(t, err)
	assert.NotZero(t, len(pp))
	// with the changes, the RealisedPNL should be 700
	assert.Equal(t, 700, int(pp[0].RealisedPNL))
	assert.Equal(t, 0, int(pp[0].UnrealisedPNL))
}

func TestDistressedTraderUpdate(t *testing.T) {
	position := getPosPlugin(t)
	defer position.Finish()
	market := "market-id"
	ps := events.NewSettlePositionEvent(position.ctx, "trader1", market, 1000, []events.TradeSettlement{
		tradeStub{
			size:  2,
			price: 1000,
		},
		tradeStub{
			size:  3,
			price: 1200,
		},
	}, 1)
	position.Push(ps)
	pp, err := position.GetPositionsByMarket(market)
	assert.NoError(t, err)
	assert.NotZero(t, len(pp))
	// average entry price should be 1k
	// initialy calculation say the RealisedPNL should be 1000
	assert.Equal(t, 0, int(pp[0].RealisedPNL))
	assert.Equal(t, -600, int(pp[0].UnrealisedPNL))

	// then we process the event for LossSocialization
	lsevt := events.NewLossSocializationEvent(position.ctx, "trader1", market, -300, 1)
	position.Push(lsevt)
	pp, err = position.GetPositionsByMarket(market)
	assert.NoError(t, err)
	assert.NotZero(t, len(pp))
	// with the changes, the RealisedPNL should be 700
	assert.Equal(t, -300, int(pp[0].RealisedPNL))
	assert.Equal(t, -600, int(pp[0].UnrealisedPNL))
	// now assume this trader is distressed, and we've taken all their funds
	sde := events.NewSettleDistressed(position.ctx, "trader1", market, 0, 100, 1)
	position.Push(sde)
	pp, err = position.GetPositionsByMarket(market)
	assert.NoError(t, err)
	assert.NotZero(t, len(pp))
	assert.Equal(t, 0, int(pp[0].UnrealisedPNL))
	assert.Equal(t, -1000, int(pp[0].RealisedPNL))
}

func TestMultipleTradesAndLossSocializationTraderWithOpenVolume(t *testing.T) {
	position := getPosPlugin(t)
	defer position.Finish()
	market := "market-id"
	ps := events.NewSettlePositionEvent(position.ctx, "trader1", market, 1000, []events.TradeSettlement{
		tradeStub{
			size:  2,
			price: 1000,
		},
		tradeStub{
			size:  3,
			price: 1200,
		},
	}, 1)
	position.Push(ps)
	pp, err := position.GetPositionsByMarket(market)
	assert.NoError(t, err)
	assert.NotZero(t, len(pp))
	// average entry price should be 1k
	// initialy calculation say the RealisedPNL should be 1000
	assert.Equal(t, 0, int(pp[0].RealisedPNL))
	assert.Equal(t, -600, int(pp[0].UnrealisedPNL))

	// then we process the event for LossSocialization
	lsevt := events.NewLossSocializationEvent(position.ctx, "trader1", market, -300, 1)
	position.Push(lsevt)
	pp, err = position.GetPositionsByMarket(market)
	assert.NoError(t, err)
	assert.NotZero(t, len(pp))
	// with the changes, the RealisedPNL should be 700
	assert.Equal(t, -300, int(pp[0].RealisedPNL))
	assert.Equal(t, -600, int(pp[0].UnrealisedPNL))
}

func getPosPlugin(t *testing.T) *posPluginTst {
	ctrl := gomock.NewController(t)
	ctx, cfunc := context.WithCancel(context.Background())
	p := plugins.NewPositions(ctx)
	tst := posPluginTst{
		Positions: p,
		ctrl:      ctrl,
		ctx:       ctx,
		cfunc:     cfunc,
	}
	return &tst
}

func (p *posPluginTst) Finish() {
	p.cfunc() // cancel context
	defer p.ctrl.Finish()
}

func (t tradeStub) Size() int64 {
	return t.size
}

func (t tradeStub) Price() uint64 {
	return t.price
}
