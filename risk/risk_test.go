package risk

import (
	"testing"

	"vega/log"
	"vega/msg"

	"github.com/stretchr/testify/assert"
)

// this runs just once as first
func init() {
	log.InitConsoleLogger(log.DebugLevel)
}

func TestRiskEngine_AddNewMarket(t *testing.T) {
	re := New()
	newMarket := &msg.Market{Name: "BTC/DEC19"}
	re.AddNewMarket(newMarket)
	riskFactorLong, riskFactorShort, err := re.GetRiskFactors(newMarket.Name)
	assert.Nil(t, err)
	assert.Equal(t, 0.00550, riskFactorLong)
	assert.Equal(t, 0.00553, riskFactorShort)
}

func TestRiskEngine_CalibrateRiskModel(t *testing.T) {
	re := New()
	newMarket := &msg.Market{Name: "BTC/DEC19"}
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
