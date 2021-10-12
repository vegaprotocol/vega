package execution_test

import (
	"testing"

	bmock "code.vegaprotocol.io/vega/broker/mocks"
	"code.vegaprotocol.io/vega/execution"
	"code.vegaprotocol.io/vega/execution/mocks"
	"code.vegaprotocol.io/vega/logging"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func createEngine(t *testing.T) *execution.Engine {
	ctrl := gomock.NewController(t)
	log := logging.NewTestLogger()
	executionConfig := execution.NewDefaultConfig()
	broker := bmock.NewMockBroker(ctrl)
	timeService := mocks.NewMockTimeService(ctrl)
	timeService.EXPECT().NotifyOnTick(gomock.Any()).Times(1)
	collateralService := mocks.NewMockCollateral(ctrl)
	oracleService := mocks.NewMockOracleEngine(ctrl)

	return execution.NewEngine(log, executionConfig, timeService, collateralService, oracleService, broker)
}

func TestEmptyMarkets(t *testing.T) {
	engine := createEngine(t)
	assert.NotNil(t, engine)

	// Check that the starting state is empty
	bytes, err := engine.GetState("")
	assert.NoError(t, err)
	assert.Empty(t, bytes)
}

func TestValidMarketSnapshot(t *testing.T) {
	engine := createEngine(t)
	assert.NotNil(t, engine)
}
