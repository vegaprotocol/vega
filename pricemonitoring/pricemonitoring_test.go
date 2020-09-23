package pricemonitoring_test

import (
	"testing"
	"time"

	"code.vegaprotocol.io/vega/pricemonitoring"
	"code.vegaprotocol.io/vega/pricemonitoring/mocks"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestConstructor(t *testing.T) {
	ctrl := gomock.NewController(t)
	riskModelMock := mocks.NewMockPriceRangeProvider(ctrl)
	var currentPrice uint64 = 123
	currentTime := time.Date(1993, 2, 2, 6, 0, 0, 1, time.UTC)
	horizon1, err := pricemonitoring.NewHorizonProbabilityLevelPair(time.Hour, 0.99)
	assert.NoError(t, err)
	horizon2, err := pricemonitoring.NewHorizonProbabilityLevelPair(2*time.Hour, 0.95)
	assert.NoError(t, err)
	horizonProbabilityPairs := []pricemonitoring.HorizonProbabilityLevelPair{*horizon1, *horizon2}
	riskModelMock.EXPECT().PriceRange(float64(currentPrice), gomock.Any(), gomock.Any()).Return(float64(currentPrice-1), float64(currentPrice+1)).Times(2)

	pm, err := pricemonitoring.NewPriceMonitoring(riskModelMock, horizonProbabilityPairs, 10*time.Minute, currentPrice, currentTime)
	assert.NoError(t, err)
	assert.NotNil(t, pm)

	negativeUpdateFrequency := -time.Minute
	pm, err = pricemonitoring.NewPriceMonitoring(riskModelMock, horizonProbabilityPairs, negativeUpdateFrequency, currentPrice, currentTime)
	assert.Error(t, err)
	assert.Nil(t, pm)

	negativeHorizon := pricemonitoring.HorizonProbabilityLevelPair{Horizon: -time.Microsecond, ProbabilityLevel: 0.5}
	pm, err = pricemonitoring.NewPriceMonitoring(riskModelMock, []pricemonitoring.HorizonProbabilityLevelPair{negativeHorizon}, 10*time.Minute, currentPrice, currentTime)
	assert.Error(t, err)
	assert.Nil(t, pm)

	invalidProbability := pricemonitoring.HorizonProbabilityLevelPair{Horizon: time.Microsecond, ProbabilityLevel: 1.1}
	pm, err = pricemonitoring.NewPriceMonitoring(riskModelMock, []pricemonitoring.HorizonProbabilityLevelPair{invalidProbability}, 10*time.Minute, currentPrice, currentTime)
	assert.Error(t, err)
	assert.Nil(t, pm)

	invalidProbability = pricemonitoring.HorizonProbabilityLevelPair{Horizon: time.Microsecond, ProbabilityLevel: -0.1}
	pm, err = pricemonitoring.NewPriceMonitoring(riskModelMock, []pricemonitoring.HorizonProbabilityLevelPair{invalidProbability}, 10*time.Minute, currentPrice, currentTime)
	assert.Error(t, err)
	assert.Nil(t, pm)

}

func TestHorizonProbablityLevelPairsSorted(t *testing.T) {
	ctrl := gomock.NewController(t)
	riskModelMock := mocks.NewMockPriceRangeProvider(ctrl)
	var currentPrice uint64 = 123
	currentTime := time.Date(1993, 2, 2, 6, 0, 0, 1, time.UTC)
	horizon1, err := pricemonitoring.NewHorizonProbabilityLevelPair(time.Hour, 0.99)
	assert.NoError(t, err)
	horizon2, err := pricemonitoring.NewHorizonProbabilityLevelPair(2*time.Hour, 0.99)
	assert.NoError(t, err)
	horizon3, err := pricemonitoring.NewHorizonProbabilityLevelPair(2*time.Hour, 0.95)
	assert.NoError(t, err)
	horizon4, err := pricemonitoring.NewHorizonProbabilityLevelPair(10*time.Hour, 0.95)
	assert.NoError(t, err)
	riskModelMock.EXPECT().PriceRange(float64(currentPrice), gomock.Any(), gomock.Any()).Return(float64(currentPrice-1), float64(currentPrice+1)).Times(4)

	expectedHorizonProbabilityPairs := []pricemonitoring.HorizonProbabilityLevelPair{*horizon1, *horizon2, *horizon3, *horizon4}
	horizonProbabilityPairs := []pricemonitoring.HorizonProbabilityLevelPair{*horizon4, *horizon2, *horizon3, *horizon1}

	pm, err := pricemonitoring.NewPriceMonitoring(riskModelMock, horizonProbabilityPairs, 10*time.Minute, currentPrice, currentTime)

	assert.NoError(t, err)
	assert.NotNil(t, pm)

	boundViolations := pm.CheckBoundViolations(currentPrice)
	i := 0
	for key := range boundViolations {
		assert.Equal(t, expectedHorizonProbabilityPairs[i], key)
		i++
	}
}

func TestUpdateTime(t *testing.T) {
	ctrl := gomock.NewController(t)
	riskModelMock := mocks.NewMockPriceRangeProvider(ctrl)
	var currentPrice uint64 = 123
	currentTime := time.Date(1993, 2, 2, 6, 0, 0, 1, time.UTC)
	horizon1, err := pricemonitoring.NewHorizonProbabilityLevelPair(time.Hour, 0.99)
	assert.Nil(t, err)
	horizon2, err := pricemonitoring.NewHorizonProbabilityLevelPair(2*time.Hour, 0.95)
	assert.Nil(t, err)
	horizonProbabilityPairs := []pricemonitoring.HorizonProbabilityLevelPair{*horizon1, *horizon2}
	riskModelMock.EXPECT().PriceRange(float64(currentPrice), gomock.Any(), gomock.Any()).Return(float64(currentPrice-1), float64(currentPrice+1)).Times(2)

	pm, err := pricemonitoring.NewPriceMonitoring(riskModelMock, horizonProbabilityPairs, 10*time.Minute, currentPrice, currentTime)
	assert.NoError(t, err)
	assert.NotNil(t, pm)

	future := currentTime.Add(time.Microsecond)
	err = pm.RecordTimeChange(future)
	assert.NoError(t, err)

	past := currentTime
	err = pm.RecordTimeChange(past)
	assert.Error(t, err)
}

func TestRecordPriceChange(t *testing.T) {
	ctrl := gomock.NewController(t)
	riskModelMock := mocks.NewMockPriceRangeProvider(ctrl)
	var currentPrice uint64 = 123
	currentTime := time.Date(1993, 2, 2, 6, 0, 0, 1, time.UTC)
	horizon1, err := pricemonitoring.NewHorizonProbabilityLevelPair(time.Hour, 0.99)
	assert.Nil(t, err)
	horizon2, err := pricemonitoring.NewHorizonProbabilityLevelPair(2*time.Hour, 0.95)
	assert.Nil(t, err)
	horizonProbabilityPairs := []pricemonitoring.HorizonProbabilityLevelPair{*horizon1, *horizon2}
	riskModelMock.EXPECT().PriceRange(float64(currentPrice), gomock.Any(), gomock.Any()).Return(float64(currentPrice-1), float64(currentPrice+1)).Times(2)

	pm, err := pricemonitoring.NewPriceMonitoring(riskModelMock, horizonProbabilityPairs, 10*time.Minute, currentPrice, currentTime)
	assert.NoError(t, err)
	assert.NotNil(t, pm)

	pm.RecordPriceChange(currentPrice + 2)
	pm.RecordPriceChange(currentPrice + 1)
	pm.RecordPriceChange(currentPrice)
}

func TestCheckBoundViolationsWithinCurrentTimeWith2HorizonProbabilityPairs(t *testing.T) {
	ctrl := gomock.NewController(t)
	riskModelMock := mocks.NewMockPriceRangeProvider(ctrl)
	var currentPrice uint64 = 123
	currentTime := time.Date(1993, 2, 2, 6, 0, 0, 1, time.UTC)
	horizon1, err := pricemonitoring.NewHorizonProbabilityLevelPair(time.Hour, 0.99)
	assert.Nil(t, err)
	horizon2, err := pricemonitoring.NewHorizonProbabilityLevelPair(2*time.Hour, 0.95)
	assert.Nil(t, err)
	horizonProbabilityPairs := []pricemonitoring.HorizonProbabilityLevelPair{*horizon1, *horizon2}
	var maxMoveDownHorizon1 uint64 = 1
	var maxMoveUpHorizon1 uint64 = 2
	var maxMoveDownHorizon2 uint64 = 3
	var maxMoveUpHorizon2 uint64 = 4
	assert.True(t, maxMoveDownHorizon2 > maxMoveDownHorizon1)
	assert.True(t, maxMoveUpHorizon2 > maxMoveUpHorizon1)
	riskModelMock.EXPECT().PriceRange(float64(currentPrice), horizonToYearFraction(horizon1.Horizon), horizon1.ProbabilityLevel).Return(float64(currentPrice-maxMoveDownHorizon1), float64(currentPrice+maxMoveUpHorizon1))
	riskModelMock.EXPECT().PriceRange(float64(currentPrice), horizonToYearFraction(horizon2.Horizon), horizon2.ProbabilityLevel).Return(float64(currentPrice-maxMoveDownHorizon2), float64(currentPrice+maxMoveUpHorizon2))

	pm, err := pricemonitoring.NewPriceMonitoring(riskModelMock, horizonProbabilityPairs, 10*time.Minute, currentPrice, currentTime)
	assert.NoError(t, err)
	assert.NotNil(t, pm)

	violations := pm.CheckBoundViolations(currentPrice + maxMoveUpHorizon1 - 1)
	assert.False(t, violations[*horizon1])
	assert.False(t, violations[*horizon2])
	violations = pm.CheckBoundViolations(currentPrice - maxMoveDownHorizon1 + 1)
	assert.False(t, violations[*horizon1])
	assert.False(t, violations[*horizon2])
	violations = pm.CheckBoundViolations(currentPrice + maxMoveUpHorizon1)
	assert.False(t, violations[*horizon1])
	assert.False(t, violations[*horizon2])
	violations = pm.CheckBoundViolations(currentPrice - maxMoveDownHorizon1)
	assert.False(t, violations[*horizon1])
	assert.False(t, violations[*horizon2])
	violations = pm.CheckBoundViolations(currentPrice + (maxMoveUpHorizon1+maxMoveUpHorizon2)/2)
	assert.True(t, violations[*horizon1])
	assert.False(t, violations[*horizon2])
	violations = pm.CheckBoundViolations(currentPrice - (maxMoveDownHorizon1+maxMoveDownHorizon2)/2)
	assert.True(t, violations[*horizon1])
	assert.False(t, violations[*horizon2])
	violations = pm.CheckBoundViolations(currentPrice + maxMoveUpHorizon2)
	assert.True(t, violations[*horizon1])
	assert.False(t, violations[*horizon2])
	violations = pm.CheckBoundViolations(currentPrice - maxMoveDownHorizon2)
	assert.True(t, violations[*horizon1])
	assert.False(t, violations[*horizon2])
	violations = pm.CheckBoundViolations(currentPrice + 2*maxMoveUpHorizon2)
	assert.True(t, violations[*horizon1])
	assert.True(t, violations[*horizon2])
	violations = pm.CheckBoundViolations(currentPrice - 2*maxMoveDownHorizon2)
	assert.True(t, violations[*horizon1])
	assert.True(t, violations[*horizon2])
}

func TestCheckBoundViolationsAcrossTimeWith1HorizonProbabilityPair(t *testing.T) {
	ctrl := gomock.NewController(t)
	riskModelMock := mocks.NewMockPriceRangeProvider(ctrl)
	var price1 uint64 = 123
	currentTime := time.Date(1993, 2, 2, 6, 0, 0, 1, time.UTC)
	horizon, err := pricemonitoring.NewHorizonProbabilityLevelPair(10*time.Minute, 0.99)
	assert.Nil(t, err)
	horizonProbabilityPairs := []pricemonitoring.HorizonProbabilityLevelPair{*horizon}
	var maxMoveDown1 uint64 = 1
	var maxMoveUp1 uint64 = 2
	boundUpdateFrequency := 2 * time.Minute
	riskModelMock.EXPECT().PriceRange(float64(price1), horizonToYearFraction(horizon.Horizon), horizon.ProbabilityLevel).Return(float64(price1-maxMoveDown1), float64(price1+maxMoveUp1))

	pm, err := pricemonitoring.NewPriceMonitoring(riskModelMock, horizonProbabilityPairs, boundUpdateFrequency, price1, currentTime)
	assert.NoError(t, err)
	assert.NotNil(t, pm)

	violations := pm.CheckBoundViolations(price1 + maxMoveUp1)
	assert.False(t, violations[*horizon])
	violations = pm.CheckBoundViolations(price1 - maxMoveDown1)
	assert.False(t, violations[*horizon])
	violations = pm.CheckBoundViolations(price1 + 2*maxMoveUp1)
	assert.True(t, violations[*horizon])
	violations = pm.CheckBoundViolations(price1 - 2*maxMoveDown1)
	assert.True(t, violations[*horizon])

	updateTime := currentTime.Add(boundUpdateFrequency)
	//Still before update
	pm.RecordTimeChange(updateTime.Add(-time.Nanosecond))
	//Execting same behaviour as above (per reference price)
	violations = pm.CheckBoundViolations(price1 + maxMoveUp1)
	assert.False(t, violations[*horizon])
	violations = pm.CheckBoundViolations(price1 - maxMoveDown1)
	assert.False(t, violations[*horizon])
	violations = pm.CheckBoundViolations(price1 + 2*maxMoveUp1)
	assert.True(t, violations[*horizon])
	violations = pm.CheckBoundViolations(price1 - 2*maxMoveDown1)
	assert.True(t, violations[*horizon])

	//Right at update time
	price2Intermediate1 := 2 * price1
	maxMoveDown2 := 4 * maxMoveDown1
	maxMoveUp2 := 4 * maxMoveUp1
	price2Intermediate2 := 4 * price1
	pm.RecordPriceChange(price2Intermediate1)
	pm.RecordPriceChange(price2Intermediate2)
	price2Average := float64(price2Intermediate1+price2Intermediate2) / 2.0
	riskModelMock.EXPECT().PriceRange(float64(price2Average), horizonToYearFraction(horizon.Horizon), horizon.ProbabilityLevel).Return(price2Average-float64(maxMoveDown2), price2Average+float64(maxMoveUp2))
	pm.RecordTimeChange(updateTime)
	referencePrice := price1
	violations = pm.CheckBoundViolations(referencePrice + maxMoveUp2)
	assert.False(t, violations[*horizon])
	violations = pm.CheckBoundViolations(referencePrice - maxMoveDown2)
	assert.False(t, violations[*horizon])
	violations = pm.CheckBoundViolations(referencePrice + 2*maxMoveUp2)
	assert.True(t, violations[*horizon])
	violations = pm.CheckBoundViolations(referencePrice - 2*maxMoveDown2)
	assert.True(t, violations[*horizon])

	//Right before update time (horizon away from price2Average)
	price3Intermediate1 := 5 * price1
	maxMoveDown3 := 6 * maxMoveDown1
	maxMoveUp3 := 6 * maxMoveUp1
	price3Intermediate2 := 7 * price1
	pm.RecordPriceChange(price3Intermediate1)
	pm.RecordPriceChange(price3Intermediate2)
	price3Average := float64(price3Intermediate1+price3Intermediate2) / 2.0
	riskModelMock.EXPECT().PriceRange(float64(price3Average), horizonToYearFraction(horizon.Horizon), horizon.ProbabilityLevel).Return(price3Average-float64(maxMoveDown3), price3Average+float64(maxMoveUp3))
	pm.RecordTimeChange(updateTime.Add(-time.Nanosecond).Add(horizon.Horizon))
	referencePrice = uint64(price2Average)
	violations = pm.CheckBoundViolations(referencePrice + maxMoveUp3)
	assert.False(t, violations[*horizon])
	violations = pm.CheckBoundViolations(referencePrice - maxMoveDown3)
	assert.False(t, violations[*horizon])
	violations = pm.CheckBoundViolations(referencePrice + 2*maxMoveUp3)
	assert.True(t, violations[*horizon])
	violations = pm.CheckBoundViolations(referencePrice - 2*maxMoveDown3)
	assert.True(t, violations[*horizon])

	//Right at update time (horizon away from price2Average)
	pm.RecordTimeChange(updateTime.Add(horizon.Horizon))
	referencePrice = uint64(price3Average)
	violations = pm.CheckBoundViolations(referencePrice + maxMoveUp3)
	assert.False(t, violations[*horizon])
	violations = pm.CheckBoundViolations(referencePrice - maxMoveDown3)
	assert.False(t, violations[*horizon])
	violations = pm.CheckBoundViolations(referencePrice + 2*maxMoveUp3)
	assert.True(t, violations[*horizon])
	violations = pm.CheckBoundViolations(referencePrice - 2*maxMoveDown3)
	assert.True(t, violations[*horizon])

	//Reset price, the resetting value should become the new reference
	var resetPrice uint64 = 20
	var maxMoveDown4 uint64 = 5
	var maxMoveUp4 uint64 = 120
	riskModelMock.EXPECT().PriceRange(float64(resetPrice), horizonToYearFraction(horizon.Horizon), horizon.ProbabilityLevel).Return(float64(resetPrice-maxMoveDown4), float64(resetPrice+maxMoveUp4))

	pm.Reset(resetPrice, updateTime.Add(horizon.Horizon).Add(time.Second))
	violations = pm.CheckBoundViolations(resetPrice + maxMoveUp4)
	assert.False(t, violations[*horizon])
	violations = pm.CheckBoundViolations(resetPrice - maxMoveDown4)
	assert.False(t, violations[*horizon])
	violations = pm.CheckBoundViolations(resetPrice + 2*maxMoveUp4)
	assert.True(t, violations[*horizon])
	violations = pm.CheckBoundViolations(resetPrice - 2*maxMoveDown4)
	assert.True(t, violations[*horizon])
}

func horizonToYearFraction(horizon time.Duration) float64 {
	return float64(horizon.Nanoseconds()) / float64((365.25 * 24 * time.Hour).Nanoseconds())
}
