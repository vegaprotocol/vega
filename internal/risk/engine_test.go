package risk

import (
	"testing"

	"code.vegaprotocol.io/vega/internal/logging"
	types "code.vegaprotocol.io/vega/proto"

	"github.com/stretchr/testify/assert"
)

func TestRiskEngine_AddNewMarket(t *testing.T) {
	logger := logging.NewLoggerFromEnv("dev")
	defer logger.Sync()

	config := NewDefaultConfig(logger)
	re := NewRiskEngine(config)
	newMarket := &types.Market{Id: "BTC/DEC19"}
	re.AddNewMarket(newMarket)
	riskFactorLong, riskFactorShort, err := re.GetRiskFactors(newMarket.Id)
	assert.Nil(t, err)
	assert.Equal(t, 0.00550, riskFactorLong)
	assert.Equal(t, 0.00553, riskFactorShort)
}

func TestRiskEngine_CalibrateRiskModel(t *testing.T) {
	logger := logging.NewLoggerFromEnv("dev")
	defer logger.Sync()

	config := NewDefaultConfig(logger)
	re := NewRiskEngine(config)

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
