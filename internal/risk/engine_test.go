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
	ctrl  *gomock.Controller
	model *mocks.MockModel
}

// implements the events.Margin interface
type testMargin struct {
	party    string
	size     int64
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
)

func TestMargin(t *testing.T) {
	eng := getTestEngine(t, nil)
	defer eng.ctrl.Finish()
	ctx, cfunc := context.WithCancel(context.Background())
	defer cfunc()
	ch := make(chan events.Margin, 1)
	// data := []events.Margin{}
	evt := testMargin{
		party:   "trader1",
		size:    1,
		price:   1000,
		asset:   "ETH",
		margin:  180,    // required margin will be > 250, so ensure we don't have enough
		general: 100000, // plenty of balance for the transfer anyway
		market:  "ETH/DEC19",
	}
	go func() {
		ch <- evt
		close(ch)
	}()
	resp := eng.UpdateMargins(ctx, ch, evt.price)
	assert.Equal(t, 1, len(resp))
}

func getTestEngine(t *testing.T, initialRisk *types.RiskResult) *testEngine {
	if initialRisk == nil {
		cpy := riskResult
		initialRisk = &cpy // this is just a shallow copy, so might be worth creating a deep copy depending on the test
	}
	ctrl := gomock.NewController(t)
	model := mocks.NewMockModel(ctrl)
	conf := risk.NewDefaultConfig()
	engine := risk.NewEngine(
		logging.NewTestLogger(),
		conf,
		model,
		initialRisk,
	)
	return &testEngine{
		Engine: engine,
		ctrl:   ctrl,
		model:  model,
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

func (m testMargin) Size() int64 {
	return m.size
}

func (m testMargin) Transfer() *types.Transfer {
	return m.transfer
}
