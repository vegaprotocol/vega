package core_test

import (
	"context"
	"fmt"
	"os"

	"code.vegaprotocol.io/vega/collateral"
	"code.vegaprotocol.io/vega/execution"
	"code.vegaprotocol.io/vega/integration/helpers"
	"code.vegaprotocol.io/vega/integration/steps/market"
	"code.vegaprotocol.io/vega/integration/stubs"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/oracles"
	"code.vegaprotocol.io/vega/plugins"
	types "code.vegaprotocol.io/vega/proto"

	"github.com/golang/mock/gomock"
)

var (
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

var (
	marketConfig = market.NewMarketConfig()
)

type executionTestSetup struct {
	cfg              execution.Config
	log              *logging.Logger
	ctrl             *gomock.Controller
	timeService      *stubs.TimeStub
	broker           *stubs.BrokerStub
	executionEngine  *execution.Engine
	collateralEngine *collateral.Engine
	oracleEngine     *oracles.Engine

	positionPlugin *plugins.Positions

	// save trader accounts state
	markets []types.Market

	InsurancePoolInitialBalance uint64

	errorHandler *helpers.ErrorHandler
}

func newExecutionTestSetup() *executionTestSetup {
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
	execsetup.timeService = stubs.NewTimeStub()
	execsetup.broker = stubs.NewBrokerStub()
	currentTime, _ := execsetup.timeService.GetTimeNow()
	execsetup.collateralEngine, _ = collateral.New(
		execsetup.log, collateral.NewDefaultConfig(), execsetup.broker, currentTime,
	)
	execsetup.oracleEngine = oracles.NewEngine(
		execsetup.log, oracles.NewDefaultConfig(), currentTime, execsetup.broker,
	)
	execsetup.executionEngine = execution.NewEngine(
		execsetup.log,
		execsetup.cfg,
		execsetup.timeService,
		execsetup.collateralEngine,
		execsetup.oracleEngine,
		execsetup.broker,
	)

	execsetup.positionPlugin = plugins.NewPositions(context.Background())
	execsetup.broker.Subscribe(execsetup.positionPlugin)

	execsetup.errorHandler = helpers.NewErrorHandler()

	return execsetup
}
