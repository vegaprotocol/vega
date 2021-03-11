package core_test

import (
	"context"
	"fmt"
	"os"
	"time"

	"code.vegaprotocol.io/vega/collateral"
	"code.vegaprotocol.io/vega/execution"
	"code.vegaprotocol.io/vega/integration/stubs"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/oracles"
	"code.vegaprotocol.io/vega/plugins"
	types "code.vegaprotocol.io/vega/proto"

	"github.com/golang/mock/gomock"
)

const (
	defaultMarketStart  = "2019-11-30T00:00:00Z"
	defaultMarketExpiry = "2019-12-31T23:59:59Z"
)

var (
	execsetup *executionTestSetup
	reporter  tstReporter

	marketStart  = defaultMarketStart
	marketExpiry = defaultMarketExpiry
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

type executionTestSetup struct {
	engine *execution.Engine

	cfg          execution.Config
	log          *logging.Logger
	ctrl         *gomock.Controller
	timesvc      *stubs.TimeStub
	collateral   *collateral.Engine
	oracleEngine *oracles.Engine

	positionPlugin *plugins.Positions

	broker *stubs.BrokerStub

	// save trader accounts state
	accs map[string][]account
	mkts []types.Market

	InsurancePoolInitialBalance uint64
}

func getExecutionSetupEmptyWithInsurancePoolBalance(balance uint64) *executionTestSetup {
	if execsetup == nil {
		execsetup = &executionTestSetup{}
	}
	execsetup.InsurancePoolInitialBalance = balance
	return execsetup
}

func getExecutionTestSetup(startTime time.Time, mkts []types.Market) *executionTestSetup {
	if execsetup != nil && execsetup.ctrl != nil {
		execsetup.ctrl.Finish()
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
	execsetup.timesvc = &stubs.TimeStub{Now: startTime}
	execsetup.broker = stubs.NewBrokerStub()
	currentTime := time.Now()
	execsetup.collateral, _ = collateral.New(
		execsetup.log, collateral.NewDefaultConfig(), execsetup.broker, currentTime,
	)
	execsetup.oracleEngine = oracles.NewEngine(execsetup.log, oracles.NewDefaultConfig(), currentTime, execsetup.broker)

	for _, mkt := range mkts {
		asset, _ := mkt.GetAsset()
		execsetup.collateral.EnableAsset(context.Background(), types.Asset{
			Id:     asset,
			Symbol: asset,
		})
	}

	tokAsset := types.Asset{
		Id:          "VOTE",
		Name:        "VOTE",
		Symbol:      "VOTE",
		Decimals:    5,
		TotalSupply: "1000",
		Source: &types.AssetSource{
			Source: &types.AssetSource_BuiltinAsset{
				BuiltinAsset: &types.BuiltinAsset{
					Name:        "VOTE",
					Symbol:      "VOTE",
					Decimals:    5,
					TotalSupply: "1000",
				},
			},
		},
	}
	execsetup.collateral.EnableAsset(context.Background(), tokAsset)

	execsetup.engine = execution.NewEngine(
		execsetup.log,
		execsetup.cfg,
		execsetup.timesvc,
		execsetup.collateral,
		execsetup.oracleEngine,
		execsetup.broker,
	)

	for _, mkt := range mkts {
		mkt := mkt
		execsetup.engine.SubmitMarket(context.Background(), &mkt)
	}

	// instantiate position plugin
	execsetup.positionPlugin = plugins.NewPositions(context.Background())
	execsetup.broker.Subscribe(execsetup.positionPlugin)

	return execsetup
}

type account struct {
	Balance uint64
	Type    types.AccountType
	Market  string
	Asset   string
}

func traderHaveGeneralAccount(accs []account, asset string) bool {
	for _, v := range accs {
		if v.Type == types.AccountType_ACCOUNT_TYPE_GENERAL && v.Asset == asset {
			return true
		}
	}
	return false
}
