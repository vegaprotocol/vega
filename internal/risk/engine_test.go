package risk_test

import (
	"context"
	"testing"

	"code.vegaprotocol.io/vega/internal/events"
	"code.vegaprotocol.io/vega/internal/logging"
	"code.vegaprotocol.io/vega/internal/risk"
	"code.vegaprotocol.io/vega/internal/risk/mocks"
	types "code.vegaprotocol.io/vega/proto"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

type testEngine struct {
	*risk.Engine
	ctrl      *gomock.Controller
	model     *mocks.MockModel
	orderbook *mocks.MockOrderbook
}

// implements the events.Margin interface
type testMargin struct {
	party    string
	size     int64
	buy      int64
	sell     int64
	price    uint64
	transfer *types.Transfer
	asset    string
	margin   uint64
	general  uint64
	market   string
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

	riskMinamount      int64  = 250
	riskRequiredMargin int64  = 300
	markPrice          uint64 = 100
)

func TestUpdateMargins(t *testing.T) {
	t.Run("Top up margin test", testMarginTopup)
	t.Run("Noop margin test", testMarginNoop)
	t.Run("Margin too high (overflow)", testMarginOverflow)
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
	eng.orderbook.EXPECT().GetCloseoutPrice(gomock.Any(), gomock.Any()).Times(1).
		DoAndReturn(func(volume uint64, side types.Side) (uint64, error) {
			return markPrice, nil
		})
	evts := []events.Margin{evt}
	resp := eng.UpdateMarginsOnSettlement(ctx, evts, markPrice)
	assert.Equal(t, 1, len(resp))
	// ensure we get the correct transfer request back, correct amount etc...
	trans := resp[0].Transfer()
	assert.Equal(t, int64(20), trans.Amount.Amount)
	// min = 15 so we go back to search level
	assert.Equal(t, int64(15), trans.Amount.MinAmount)
	assert.Equal(t, types.TransferType_MARGIN_LOW, trans.Type)
}

func testMarginNoop(t *testing.T) {
	eng := getTestEngine(t, nil)
	defer eng.ctrl.Finish()
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
	eng.orderbook.EXPECT().GetCloseoutPrice(gomock.Any(), gomock.Any()).Times(1).
		DoAndReturn(func(volume uint64, side types.Side) (uint64, error) {
			return markPrice, nil
		})

	evts := []events.Margin{evt}
	resp := eng.UpdateMarginsOnSettlement(ctx, evts, markPrice)
	assert.Equal(t, 0, len(resp))
}

func testMarginOverflow(t *testing.T) {
	eng := getTestEngine(t, nil)
	defer eng.ctrl.Finish()
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
	eng.orderbook.EXPECT().GetCloseoutPrice(gomock.Any(), gomock.Any()).Times(1).
		DoAndReturn(func(volume uint64, side types.Side) (uint64, error) {
			return markPrice, nil
		})
	evts := []events.Margin{evt}
	resp := eng.UpdateMarginsOnSettlement(ctx, evts, markPrice)
	assert.Equal(t, 1, len(resp))
	// ensure we get the correct transfer request back, correct amount etc...
	trans := resp[0].Transfer()
	assert.Equal(t, int64(465), trans.Amount.Amount)
	// assert.Equal(t, riskMinamount-int64(evt.margin), trans.Amount.MinAmount)
	assert.Equal(t, types.TransferType_MARGIN_HIGH, trans.Type)
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
	engine := risk.NewEngine(
		logging.NewTestLogger(),
		conf,
		getMarginCalculator(),
		model,
		initialRisk,
		ob,
	)
	return &testEngine{
		Engine:    engine,
		ctrl:      ctrl,
		model:     model,
		orderbook: ob,
	}
}

func getMarginCalculator() *types.MarginCalculator {
	return &types.MarginCalculator{
		ScalingFactors: &types.ScalingFactors{
			SearchLevel:       1.1,
			InitialMargin:     1.2,
			CollateralRelease: 1.4,
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

func (m testMargin) MarginBalance() uint64 {
	return m.margin
}

func (m testMargin) GeneralBalance() uint64 {
	return m.general
}

func (m testMargin) Price() uint64 {
	return m.price
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

func (m testMargin) Transfer() *types.Transfer {
	return m.transfer
}
