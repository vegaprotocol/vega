package core_test

import (
	"errors"
	"fmt"
	"os"
	"time"

	"code.vegaprotocol.io/vega/collateral"
	"code.vegaprotocol.io/vega/execution"
	"code.vegaprotocol.io/vega/execution/mocks"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/proto"

	"github.com/golang/mock/gomock"
)

var (
	mktsetup  *marketTestSetup
	execsetup *executionTestSetup
	reporter  tstReporter
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
	market   *proto.Market
	ctrl     *gomock.Controller
	core     *execution.Market
	party    *execution.Party
	candles  *mocks.MockCandleBuf
	orders   *orderStub
	trades   *tradeStub
	parties  *mocks.MockPartyBuf
	transfer *mocks.MockTransferBuf
	accounts *accStub
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
		time.Now(),
	)
	// mock call to get the last candle
	candles.EXPECT().Start(gomock.Any(), gomock.Any()).MinTimes(1).Return(nil, nil)
	candles.EXPECT().AddTrade(gomock.Any()).AnyTimes().Return(nil)

	setup := &marketTestSetup{
		market:     market,
		ctrl:       ctrl,
		candles:    candles,
		orders:     orders,
		trades:     trades,
		parties:    parties,
		transfer:   transfer,
		accounts:   accounts,
		accountIDs: map[string]struct{}{},
		traderAccs: map[string]map[proto.AccountType]*proto.Account{},
		colE:       colE,
	}

	return setup
}

type executionTestSetup struct {
	engine *execution.Engine

	cfg       execution.Config
	log       *logging.Logger
	ctrl      *gomock.Controller
	accounts  *accStub
	candles   *mocks.MockCandleBuf
	orders    *orderStub
	trades    *tradeStub
	parties   *mocks.MockPartyBuf
	transfers *transferStub
	markets   *mocks.MockMarketBuf
	timesvc   *mocks.MockTimeService

	// save trader accounts state
	accs map[string][]account
	mkts []proto.Market
}

func getExecutionTestSetup(mkts []proto.Market) *executionTestSetup {
	if execsetup != nil {
		execsetup.ctrl.Finish()
		execsetup = nil
	}

	ctrl := gomock.NewController(&reporter)
	setup := executionTestSetup{
		ctrl:      ctrl,
		cfg:       execution.NewDefaultConfig(""),
		log:       logging.NewTestLogger(),
		accounts:  NewAccountStub(),
		candles:   mocks.NewMockCandleBuf(ctrl),
		orders:    NewOrderStub(),
		trades:    NewTradeStub(),
		parties:   mocks.NewMockPartyBuf(ctrl),
		transfers: NewTransferStub(),
		markets:   mocks.NewMockMarketBuf(ctrl),
		timesvc:   mocks.NewMockTimeService(ctrl),
		accs:      map[string][]account{},
		mkts:      mkts,
	}

	setup.timesvc.EXPECT().GetTimeNow().AnyTimes().Return(time.Now(), nil)
	setup.timesvc.EXPECT().NotifyOnTick(gomock.Any()).AnyTimes()
	setup.candles.EXPECT().Start(gomock.Any(), gomock.Any()).AnyTimes().Return(nil, nil)
	setup.markets.EXPECT().Add(gomock.Any()).AnyTimes()
	setup.parties.EXPECT().Add(gomock.Any()).AnyTimes()
	setup.candles.EXPECT().AddTrade(gomock.Any()).AnyTimes()
	setup.markets.EXPECT().Flush().AnyTimes().Return(nil)

	setup.engine = execution.NewEngine(setup.log, setup.cfg, setup.timesvc, setup.orders, setup.trades, setup.candles, setup.markets, setup.parties, setup.accounts, setup.transfers, mkts)

	return &setup
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
