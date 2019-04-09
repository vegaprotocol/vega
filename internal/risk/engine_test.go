package risk_test

import (
	"testing"

	"code.vegaprotocol.io/vega/internal/logging"
	"code.vegaprotocol.io/vega/internal/risk"
	types "code.vegaprotocol.io/vega/proto"

	"github.com/stretchr/testify/assert"
)

func TestRiskEngine_AddNewMarket(t *testing.T) {
	logger := logging.NewLoggerFromEnv("test")
	defer logger.Sync()

	config := risk.NewDefaultConfig(logger)
	re := risk.NewRiskEngine(config)
	newMarket := &types.Market{Id: "BTC/DEC19"}
	re.AddNewMarket(newMarket)
	riskFactorLong, riskFactorShort, err := re.GetRiskFactors(newMarket.Id)
	assert.Nil(t, err)
	assert.Equal(t, 0.00550, riskFactorLong)
	assert.Equal(t, 0.00553, riskFactorShort)
}

func TestRiskEngine_CalibrateRiskModel(t *testing.T) {
	logger := logging.NewLoggerFromEnv("test")
	defer logger.Sync()

	config := risk.NewDefaultConfig(logger)
	re := risk.NewRiskEngine(config)

	newMarket := &types.Market{Id: "BTC/DEC19"}
	re.AddNewMarket(newMarket)
	riskFactorLong, riskFactorShort, err := re.GetRiskFactors(newMarket.Id)
	assert.Nil(t, err)
	assert.Equal(t, 0.00550, riskFactorLong)
	assert.Equal(t, 0.00553, riskFactorShort)

	re.RecalculateRisk()
	riskFactorLong, riskFactorShort, err = re.GetRiskFactors(newMarket.Id)
	assert.Nil(t, err)
	assert.Equal(t, 0.00550, riskFactorLong)
	assert.Equal(t, 0.00553, riskFactorShort)
}
