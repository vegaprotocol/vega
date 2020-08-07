package core_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"code.vegaprotocol.io/vega/collateral"
	"code.vegaprotocol.io/vega/execution"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/plugins"
	"code.vegaprotocol.io/vega/proto"
	types "code.vegaprotocol.io/vega/proto"

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
	market   *proto.Market
	ctrl     *gomock.Controller
	core     *execution.Market
	accounts *accStub
	proposal *ProposalStub
	votes    *VoteStub

	// accounts   *cmocks.MockAccountBuffer
	accountIDs map[string]struct{}
	traderAccs map[string]map[proto.AccountType]*proto.Account

	// we need to call this engine directly
	colE   *collateral.Engine
	broker *brokerStub
}

func getMarketTestSetup(market *proto.Market) *marketTestSetup {
	if mktsetup != nil {
		mktsetup.ctrl.Finish()
		mktsetup = nil // ready for GC
	}
	// the controller needs the reporter to report on errors or clunk out with fatal
	ctrl := gomock.NewController(&reporter)
	broker := NewBrokerStub()

	// this can happen any number of times, just set the mock up to accept all of them
	// Over time, these mocks will be replaced with stubs that store all elements to a map
	// again: allow all calls, replace with stub over time
	colE, _ := collateral.New(
		logging.NewTestLogger(),
		collateral.NewDefaultConfig(),
		broker,
		time.Now(),
	)

	tokAsset := types.Asset{
		ID:          collateral.TokenAssetSource.GetBuiltinAsset().Symbol,
		Name:        collateral.TokenAssetSource.GetBuiltinAsset().Name,
		Symbol:      collateral.TokenAssetSource.GetBuiltinAsset().Symbol,
		Decimals:    collateral.TokenAssetSource.GetBuiltinAsset().Decimals,
		TotalSupply: collateral.TokenAssetSource.GetBuiltinAsset().TotalSupply,
		Source:      collateral.TokenAssetSource,
	}
	colE.EnableAsset(context.Background(), tokAsset)

	setup := &marketTestSetup{
		market:     market,
		ctrl:       ctrl,
		accountIDs: map[string]struct{}{},
		traderAccs: map[string]map[proto.AccountType]*proto.Account{},
		colE:       colE,
		broker:     broker,
	}

	return setup
}

type executionTestSetup struct {
	engine *execution.Engine

	cfg        execution.Config
	log        *logging.Logger
	ctrl       *gomock.Controller
	timesvc    *timeStub
	proposal   *ProposalStub
	votes      *VoteStub
	collateral *collateral.Engine

	positionPlugin *plugins.Positions

	broker *brokerStub

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
		// execsetup = nil
	} else if execsetup == nil {
		execsetup = &executionTestSetup{}
	}

	ctrl := gomock.NewController(&reporter)
	execsetup.ctrl = ctrl
	execsetup.cfg = execution.NewDefaultConfig("")
	execsetup.cfg.InsurancePoolInitialBalance = execsetup.InsurancePoolInitialBalance
	execsetup.log = logging.NewTestLogger()
	execsetup.accs = map[string][]account{}
	execsetup.mkts = mkts
	execsetup.timesvc = &timeStub{now: startTime}
	execsetup.proposal = NewProposalStub()
	execsetup.votes = NewVoteStub()
	execsetup.broker = NewBrokerStub()
	execsetup.collateral, _ = collateral.New(
		execsetup.log, collateral.NewDefaultConfig(), execsetup.broker, time.Now(),
	)

	for _, mkt := range mkts {
		asset, _ := mkt.GetAsset()
		execsetup.collateral.EnableAsset(context.Background(), proto.Asset{
			ID:     asset,
			Symbol: asset,
		})
	}

	tokAsset := types.Asset{
		ID:          collateral.TokenAssetSource.GetBuiltinAsset().Symbol,
		Name:        collateral.TokenAssetSource.GetBuiltinAsset().Name,
		Symbol:      collateral.TokenAssetSource.GetBuiltinAsset().Symbol,
		Decimals:    collateral.TokenAssetSource.GetBuiltinAsset().Decimals,
		TotalSupply: collateral.TokenAssetSource.GetBuiltinAsset().TotalSupply,
		Source:      collateral.TokenAssetSource,
	}
	execsetup.collateral.EnableAsset(context.Background(), tokAsset)

	execsetup.engine = execution.NewEngine(execsetup.log, execsetup.cfg, execsetup.timesvc, mkts, execsetup.collateral, execsetup.broker)

	// instanciate position plugin
	execsetup.positionPlugin = plugins.NewPositions(context.Background())
	execsetup.broker.Subscribe(execsetup.positionPlugin)

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
		if v.Type == proto.AccountType_ACCOUNT_TYPE_MARGIN && v.Market == market {
			return v, nil
		}
	}
	return account{}, errors.New("account does not exist")
}

func getTraderGeneralAccount(accs []account, asset string) (account, error) {
	for _, v := range accs {
		if v.Type == proto.AccountType_ACCOUNT_TYPE_GENERAL && v.Asset == asset {
			return v, nil
		}
	}
	return account{}, errors.New("account does not exist")
}

func traderHaveGeneralAccount(accs []account, asset string) bool {
	for _, v := range accs {
		if v.Type == proto.AccountType_ACCOUNT_TYPE_GENERAL && v.Asset == asset {
			return true
		}
	}
	return false
}
