package core_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"code.vegaprotocol.io/vega/buffer"
	"code.vegaprotocol.io/vega/collateral"
	"code.vegaprotocol.io/vega/execution"
	"code.vegaprotocol.io/vega/execution/mocks"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/plugins"
	"code.vegaprotocol.io/vega/proto"

	"github.com/golang/mock/gomock"
)

const (
	defaultMarketStart  = "2019-11-30T00:00:00Z"
	defaultMarketExpiry = "2019-12-31T23:59:59Z"
)

var (
	mktsetup  *marketTestSetup
	execsetup *executionTestSetup
	reporter  tstReporter

	marketStart  string = defaultMarketStart
	marketExpiry string = defaultMarketExpiry
)

type tstReporter struct {
	err  error
	step string
}

func (t tstReporter) Errorf(format string, args ...interface{}) {
	fmt.Printf("%s ERROR: %s", t.step, fmt.Sprintf(format, args...))
}

func (t tstReporter) Fatalf(format string, args ...interface{}) {
	fmt.Printf("%s FATAL: %s", t.step, fmt.Sprintf(format, args...))
	os.Exit(1)
}

type marketTestSetup struct {
	market          *proto.Market
	ctrl            *gomock.Controller
	core            *execution.Market
	party           *execution.Party
	candles         *mocks.MockCandleBuf
	orders          *orderStub
	trades          *tradeStub
	parties         *mocks.MockPartyBuf
	transfer        *mocks.MockTransferBuf
	accounts        *accStub
	marginLevelsBuf *marginsStub
	settle          *SettleStub
	// TODO(jeremy): will need a stub at some point for that
	lossSoc *mocks.MockLossSocializationBuf

	// accounts   *cmocks.MockAccountBuffer
	accountIDs map[string]struct{}
	traderAccs map[string]map[proto.AccountType]*proto.Account

	// we need to call this engine directly
	colE *collateral.Engine
}

func getMarketTestSetup(market *proto.Market) *marketTestSetup {
	if mktsetup != nil {
		mktsetup.ctrl.Finish()
		mktsetup = nil // ready for GC
	}
	// the controller needs the reporter to report on errors or clunk out with fatal
	ctrl := gomock.NewController(&reporter)
	candles := mocks.NewMockCandleBuf(ctrl)
	orders := NewOrderStub()
	trades := NewTradeStub()
	parties := mocks.NewMockPartyBuf(ctrl)
	lossBuf := mocks.NewMockLossSocializationBuf(ctrl)
	lossBuf.EXPECT().Add(gomock.Any()).AnyTimes()
	lossBuf.EXPECT().Flush().AnyTimes()

	// this can happen any number of times, just set the mock up to accept all of them
	// Over time, these mocks will be replaced with stubs that store all elements to a map
	parties.EXPECT().Add(gomock.Any()).AnyTimes()
	accounts := NewAccountStub()
	transfer := mocks.NewMockTransferBuf(ctrl)
	// again: allow all calls, replace with stub over time
	transfer.EXPECT().Add(gomock.Any()).AnyTimes()
	transfer.EXPECT().Flush().AnyTimes()
	colE, _ := collateral.New(
		logging.NewTestLogger(),
		collateral.NewDefaultConfig(),
		accounts,
		lossBuf,
		time.Now(),
	)
	marginLevelsBuf := NewMarginsStub()
	candles.EXPECT().AddTrade(gomock.Any()).AnyTimes().Return(nil)

	setup := &marketTestSetup{
		market:          market,
		ctrl:            ctrl,
		candles:         candles,
		orders:          orders,
		trades:          trades,
		parties:         parties,
		transfer:        transfer,
		accounts:        accounts,
		marginLevelsBuf: marginLevelsBuf,
		settle:          NewSettlementStub(),
		lossSoc:         lossBuf,
		accountIDs:      map[string]struct{}{},
		traderAccs:      map[string]map[proto.AccountType]*proto.Account{},
		colE:            colE,
	}

	return setup
}

type executionTestSetup struct {
	engine *execution.Engine

	cfg             execution.Config
	log             *logging.Logger
	ctrl            *gomock.Controller
	accounts        *accStub
	candles         *mocks.MockCandleBuf
	orders          *orderStub
	trades          *tradeStub
	parties         *mocks.MockPartyBuf
	transfers       *transferStub
	markets         *mocks.MockMarketBuf
	timesvc         *timeStub
	marketdata      *mocks.MockMarketDataBuf
	marginLevelsBuf *marginsStub
	settle          *buffer.Settlement
	lossSoc         *buffer.LossSocialization

	positionPlugin *plugins.Positions

	// save trader accounts state
	accs map[string][]account
	mkts []proto.Market

	InsurancePoolInitialBalance uint64
}

func getExecutionSetupEmptyWithInsurancePoolBalance(balance uint64) *executionTestSetup {
	if execsetup == nil {
		execsetup = &executionTestSetup{}
	}
	execsetup.InsurancePoolInitialBalance = balance
	return execsetup
}

func getExecutionTestSetup(startTime time.Time, mkts []proto.Market) *executionTestSetup {
	if execsetup != nil && execsetup.ctrl != nil {
		execsetup.ctrl.Finish()
		execsetup.positionPlugin.Stop()
		// execsetup = nil
	} else if execsetup == nil {
		execsetup = &executionTestSetup{}
	}

	ctrl := gomock.NewController(&reporter)
	execsetup.ctrl = ctrl
	execsetup.cfg = execution.NewDefaultConfig("")
	execsetup.cfg.InsurancePoolInitialBalance = execsetup.InsurancePoolInitialBalance
	execsetup.log = logging.NewTestLogger()
	execsetup.accounts = NewAccountStub()
	execsetup.candles = mocks.NewMockCandleBuf(ctrl)
	execsetup.orders = NewOrderStub()
	execsetup.trades = NewTradeStub()
	execsetup.settle = buffer.NewSettlement()
	execsetup.parties = mocks.NewMockPartyBuf(ctrl)
	execsetup.transfers = NewTransferStub()
	execsetup.markets = mocks.NewMockMarketBuf(ctrl)
	execsetup.accs = map[string][]account{}
	execsetup.mkts = mkts
	execsetup.timesvc = &timeStub{now: startTime}
	execsetup.marketdata = mocks.NewMockMarketDataBuf(ctrl)
	execsetup.marginLevelsBuf = NewMarginsStub()
	execsetup.lossSoc = buffer.NewLossSocialization()

	execsetup.marketdata.EXPECT().Flush().AnyTimes()
	execsetup.marketdata.EXPECT().Add(gomock.Any()).AnyTimes()
	execsetup.candles.EXPECT().Start(gomock.Any(), gomock.Any()).AnyTimes().Return(nil, nil)
	execsetup.candles.EXPECT().Flush(gomock.Any(), gomock.Any()).AnyTimes().Return(nil)
	execsetup.markets.EXPECT().Add(gomock.Any()).AnyTimes()
	execsetup.parties.EXPECT().Add(gomock.Any()).AnyTimes()
	execsetup.candles.EXPECT().AddTrade(gomock.Any()).AnyTimes()
	execsetup.markets.EXPECT().Flush().AnyTimes().Return(nil)

	execsetup.engine = execution.NewEngine(execsetup.log, execsetup.cfg, execsetup.timesvc, execsetup.orders, execsetup.trades, execsetup.candles, execsetup.markets, execsetup.parties, execsetup.accounts, execsetup.transfers, execsetup.marketdata, execsetup.marginLevelsBuf, execsetup.settle, execsetup.lossSoc, mkts)

	// instanciate position plugin
	execsetup.positionPlugin = plugins.NewPositions(execsetup.settle, execsetup.lossSoc)
	execsetup.positionPlugin.Start(context.Background())

	return execsetup
}

type account struct {
	Balance uint64
	Type    proto.AccountType
	Market  string
	Asset   string
}

func getTraderMarginAccount(accs []account, market string) (account, error) {
	for _, v := range accs {
		if v.Type == proto.AccountType_MARGIN && v.Market == market {
			return v, nil
		}
	}
	return account{}, errors.New("account does not exist")
}

func getTraderGeneralAccount(accs []account, asset string) (account, error) {
	for _, v := range accs {
		if v.Type == proto.AccountType_GENERAL && v.Asset == asset {
			return v, nil
		}
	}
	return account{}, errors.New("account does not exist")
}

func traderHaveGeneralAccount(accs []account, asset string) bool {
	for _, v := range accs {
		if v.Type == proto.AccountType_GENERAL && v.Asset == asset {
			return true
		}
	}
	return false
}
