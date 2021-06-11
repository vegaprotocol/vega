package models_test

import (
	"testing"

	ptypes "code.vegaprotocol.io/vega/proto"
	"code.vegaprotocol.io/vega/risk/models"
	"github.com/stretchr/testify/assert"
)

func TestProbabilityOfTradingCloseToMinAndMax(t *testing.T) {
	logNormal, err := models.NewBuiltinFutures(&ptypes.LogNormalRiskModel{Params: &ptypes.LogNormalModelParams{Mu: 0, R: 0, Sigma: 1.2}, Tau: 1 / 365.25 / 24}, "test")
	assert.NoError(t, err)

	minPrice := 99.0
	maxPrice := 101.0
	currentPrice := 100.0
	bidPrice := currentPrice - 0.01
	askPrice := currentPrice + 0.01

	probBidNearMid := logNormal.ProbabilityOfTrading(currentPrice, logNormal.GetProjectionHorizon(), bidPrice, true, true, minPrice, maxPrice)
	probAskNearMid := logNormal.ProbabilityOfTrading(currentPrice, logNormal.GetProjectionHorizon(), askPrice, false, true, minPrice, maxPrice)
	porbBidAtMaxPrice := logNormal.ProbabilityOfTrading(currentPrice, logNormal.GetProjectionHorizon(), maxPrice, true, true, minPrice, maxPrice)
	probAskAtMinPrice := logNormal.ProbabilityOfTrading(currentPrice, logNormal.GetProjectionHorizon(), minPrice, false, true, minPrice, maxPrice)

	assert.InDelta(t, 0.5, probBidNearMid, 1e-2)
	assert.InDelta(t, 0.5, probAskNearMid, 1e-2)
	assert.InDelta(t, 1, porbBidAtMaxPrice, 1e-2)
	assert.InDelta(t, 1, probAskAtMinPrice, 1e-2)
}
