package risk

import (
	"testing"
	types "vega/proto"
	"github.com/stretchr/testify/assert"
	"vega/internal/logging"
)

func TestRiskEngine_AddNewMarket(t *testing.T) {
	logger := logging.NewLoggerFromEnv("dev")
	defer logger.Sync()

	config := NewConfig(logger)
	re := NewRiskEngine(config)
	newMarket := &types.Market{Name: "BTC/DEC19"}
	re.AddNewMarket(newMarket)
	riskFactorLong, riskFactorShort, err := re.GetRiskFactors(newMarket.Name)
	assert.Nil(t, err)
	assert.Equal(t, 0.00550, riskFactorLong)
	assert.Equal(t, 0.00553, riskFactorShort)
}

func TestRiskEngine_CalibrateRiskModel(t *testing.T) {
	logger := logging.NewLoggerFromEnv("dev")
	defer logger.Sync()

	config := NewConfig(logger)
	re := NewRiskEngine(config)

	newMarket := &types.Market{Name: "BTC/DEC19"}
	re.AddNewMarket(newMarket)
	riskFactorLong, riskFactorShort, err := re.GetRiskFactors(newMarket.Name)
	assert.Nil(t, err)
	assert.Equal(t, 0.00550, riskFactorLong)
	assert.Equal(t, 0.00553, riskFactorShort)

	re.RecalculateRisk()
	riskFactorLong, riskFactorShort, err = re.GetRiskFactors(newMarket.Name)
	assert.Nil(t, err)
	assert.Equal(t, 0.00550, riskFactorLong)
	assert.Equal(t, 0.00553, riskFactorShort)
}
