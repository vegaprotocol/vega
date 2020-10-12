package price_test

import (
	"context"
	"math"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/monitor/price"
	"code.vegaprotocol.io/vega/monitor/price/mocks"
	types "code.vegaprotocol.io/vega/proto"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func TestEmptyParametersList(t *testing.T) {
	ctrl := gomock.NewController(t)
	riskModelMock := mocks.NewMockRangeProvider(ctrl)
	auctionStateMock := mocks.NewMockAuctionState(ctrl)
	var currentPrice uint64 = 123
	now := time.Date(1993, 2, 2, 6, 0, 0, 1, time.UTC)

	settings := types.PriceMonitoringSettings{
		PriceMonitoringParameters: []*types.PriceMonitoringParameters{},
		UpdateFrequency:           1}

	riskModelMock.EXPECT().PriceRange(float64(currentPrice), gomock.Any(), gomock.Any()).Return(float64(currentPrice-10), float64(currentPrice+10)).Times(2)
	auctionStateMock.EXPECT().IsFBA().Return(false).Times(4)
	auctionStateMock.EXPECT().InAuction().Return(false).Times(4)

	pm, err := price.NewMonitor(riskModelMock, settings)
	require.NoError(t, err)
	require.NotNil(t, pm)

	err = pm.CheckPrice(context.TODO(), auctionStateMock, currentPrice, now)
	require.NoError(t, err)

	err = pm.CheckPrice(context.TODO(), auctionStateMock, currentPrice, now.Add(time.Second))
	require.NoError(t, err)

	err = pm.CheckPrice(context.TODO(), auctionStateMock, currentPrice, now.Add(time.Minute))
	require.NoError(t, err)

	err = pm.CheckPrice(context.TODO(), auctionStateMock, currentPrice, now.Add(time.Hour))
	require.NoError(t, err)
}

func TestErrorWithNilRiskModel(t *testing.T) {
	p1 := types.PriceMonitoringParameters{Horizon: 7200, Probability: 0.95, AuctionExtension: 300}
	p2 := types.PriceMonitoringParameters{Horizon: 3600, Probability: 0.99, AuctionExtension: 60}
	settings := types.PriceMonitoringSettings{
		PriceMonitoringParameters: []*types.PriceMonitoringParameters{&p1, &p2},
		UpdateFrequency:           600}

	pm, err := price.NewMonitor(nil, settings)
	require.Error(t, err)
	require.Nil(t, pm)
}

func TestRecordPriceChange(t *testing.T) {
	ctrl := gomock.NewController(t)
	riskModelMock := mocks.NewMockRangeProvider(ctrl)
	auctionStateMock := mocks.NewMockAuctionState(ctrl)
	var currentPrice uint64 = 123
	now := time.Date(1993, 2, 2, 6, 0, 0, 1, time.UTC)
	p1 := types.PriceMonitoringParameters{Horizon: 7200, Probability: 0.95, AuctionExtension: 300}
	p2 := types.PriceMonitoringParameters{Horizon: 3600, Probability: 0.99, AuctionExtension: 60}
	settings := types.PriceMonitoringSettings{
		PriceMonitoringParameters: []*types.PriceMonitoringParameters{&p1, &p2},
		UpdateFrequency:           600}

	riskModelMock.EXPECT().PriceRange(float64(currentPrice), gomock.Any(), gomock.Any()).Return(float64(currentPrice-10), float64(currentPrice+10)).Times(2)
	auctionStateMock.EXPECT().IsFBA().Return(false).Times(4)
	auctionStateMock.EXPECT().InAuction().Return(false).Times(4)

	pm, err := price.NewMonitor(riskModelMock, settings)
	require.NoError(t, err)
	require.NotNil(t, pm)

	err = pm.CheckPrice(context.TODO(), auctionStateMock, currentPrice, now)
	require.NoError(t, err)
	err = pm.CheckPrice(context.TODO(), auctionStateMock, currentPrice+2, now)
	require.NoError(t, err)
	err = pm.CheckPrice(context.TODO(), auctionStateMock, currentPrice+1, now)
	require.NoError(t, err)
	err = pm.CheckPrice(context.TODO(), auctionStateMock, currentPrice, now)
	require.NoError(t, err)
}

func TestCheckBoundViolationsWithinCurrentTimeWith2HorizonProbabilityPairs(t *testing.T) {
	ctrl := gomock.NewController(t)
	riskModelMock := mocks.NewMockRangeProvider(ctrl)
	auctionStateMock := mocks.NewMockAuctionState(ctrl)
	var currentPrice uint64 = 123
	now := time.Date(1993, 2, 2, 6, 0, 0, 1, time.UTC)
	var p1Time int64 = 60
	var p2Time int64 = 300
	p1 := types.PriceMonitoringParameters{Horizon: 3600, Probability: 0.99, AuctionExtension: p1Time}
	p2 := types.PriceMonitoringParameters{Horizon: 7200, Probability: 0.95, AuctionExtension: p2Time}
	settings := types.PriceMonitoringSettings{
		PriceMonitoringParameters: []*types.PriceMonitoringParameters{&p1, &p2},
		UpdateFrequency:           600}

	var maxMoveDownHorizon1 uint64 = 1
	var maxMoveUpHorizon1 uint64 = 2
	var maxMoveDownHorizon2 uint64 = 3
	var maxMoveUpHorizon2 uint64 = 4
	require.True(t, maxMoveDownHorizon2 > maxMoveDownHorizon1)
	require.True(t, maxMoveUpHorizon2 > maxMoveUpHorizon1)
	riskModelMock.EXPECT().PriceRange(float64(currentPrice), horizonToYearFraction(p1.Horizon), p1.Probability).Return(float64(currentPrice-maxMoveDownHorizon1), float64(currentPrice+maxMoveUpHorizon1))
	riskModelMock.EXPECT().PriceRange(float64(currentPrice), horizonToYearFraction(p2.Horizon), p2.Probability).Return(float64(currentPrice-maxMoveDownHorizon2), float64(currentPrice+maxMoveUpHorizon2))
	auctionStateMock.EXPECT().IsFBA().Return(false).Times(11)
	auctionStateMock.EXPECT().InAuction().Return(false).Times(11)

	pm, err := price.NewMonitor(riskModelMock, settings)
	require.NoError(t, err)
	require.NotNil(t, pm)

	err = pm.CheckPrice(context.TODO(), auctionStateMock, currentPrice, now)
	require.NoError(t, err)

	err = pm.CheckPrice(context.TODO(), auctionStateMock, currentPrice+maxMoveUpHorizon1-1, now)
	require.NoError(t, err)

	err = pm.CheckPrice(context.TODO(), auctionStateMock, currentPrice-maxMoveDownHorizon1+1, now)
	require.NoError(t, err)

	err = pm.CheckPrice(context.TODO(), auctionStateMock, currentPrice+maxMoveUpHorizon1, now)
	require.NoError(t, err)

	err = pm.CheckPrice(context.TODO(), auctionStateMock, currentPrice-maxMoveDownHorizon1, now)
	require.NoError(t, err)

	end := types.AuctionDuration{Duration: p1.AuctionExtension}
	auctionStateMock.EXPECT().StartPriceAuction(now, &end)
	err = pm.CheckPrice(context.TODO(), auctionStateMock, currentPrice+(maxMoveUpHorizon1+maxMoveUpHorizon2)/2, now)
	require.NoError(t, err)

	auctionStateMock.EXPECT().StartPriceAuction(now, &end)
	err = pm.CheckPrice(context.TODO(), auctionStateMock, currentPrice-(maxMoveDownHorizon1+maxMoveDownHorizon2)/2, now)
	require.NoError(t, err)

	auctionStateMock.EXPECT().StartPriceAuction(now, &end)
	err = pm.CheckPrice(context.TODO(), auctionStateMock, currentPrice+maxMoveUpHorizon2, now)
	require.NoError(t, err)

	auctionStateMock.EXPECT().StartPriceAuction(now, &end)
	err = pm.CheckPrice(context.TODO(), auctionStateMock, currentPrice-maxMoveDownHorizon2, now)
	require.NoError(t, err)

	end = types.AuctionDuration{Duration: p1.AuctionExtension + p2.AuctionExtension}
	auctionStateMock.EXPECT().StartPriceAuction(now, &end)
	err = pm.CheckPrice(context.TODO(), auctionStateMock, currentPrice+2*maxMoveUpHorizon2, now)
	require.NoError(t, err)

	auctionStateMock.EXPECT().StartPriceAuction(now, &end)
	err = pm.CheckPrice(context.TODO(), auctionStateMock, currentPrice-2*maxMoveDownHorizon2, now)
	require.NoError(t, err)
}

func TestCheckBoundViolationsAcrossTimeWith1HorizonProbabilityPair(t *testing.T) {
	ctrl := gomock.NewController(t)
	riskModelMock := mocks.NewMockRangeProvider(ctrl)
	auctionStateMock := mocks.NewMockAuctionState(ctrl)
	var price1 uint64 = 123
	now := time.Date(1993, 2, 2, 6, 0, 0, 1, time.UTC)
	var p1Time int64 = 60
	p1 := types.PriceMonitoringParameters{Horizon: 600, Probability: 0.99, AuctionExtension: p1Time}
	var boundUpdateFrequency int64 = 120
	settings := types.PriceMonitoringSettings{
		PriceMonitoringParameters: []*types.PriceMonitoringParameters{&p1},
		UpdateFrequency:           boundUpdateFrequency}
	var maxMoveDown1 uint64 = 1
	var maxMoveUp1 uint64 = 2

	riskModelMock.EXPECT().PriceRange(float64(price1), horizonToYearFraction(p1.Horizon), p1.Probability).Return(float64(price1-maxMoveDown1), float64(price1+maxMoveUp1))
	auctionStateMock.EXPECT().IsFBA().Return(false).Times(25)
	auctionStateMock.EXPECT().InAuction().Return(false).Times(25)

	pm, err := price.NewMonitor(riskModelMock, settings)
	require.NoError(t, err)
	require.NotNil(t, pm)
	var priceHistorySum uint64 = 0
	n := 0
	referencePrice := float64(price1)
	validPriceToCheck := uint64(referencePrice)
	priceHistorySum += validPriceToCheck
	n++
	err = pm.CheckPrice(context.TODO(), auctionStateMock, validPriceToCheck, now)
	require.NoError(t, err)

	validPriceToCheck = uint64(referencePrice) + maxMoveUp1
	priceHistorySum += validPriceToCheck
	n++
	err = pm.CheckPrice(context.TODO(), auctionStateMock, validPriceToCheck, now)
	require.NoError(t, err)

	validPriceToCheck = uint64(referencePrice) - maxMoveDown1
	priceHistorySum += validPriceToCheck
	n++
	err = pm.CheckPrice(context.TODO(), auctionStateMock, validPriceToCheck, now)
	require.NoError(t, err)

	end := types.AuctionDuration{Duration: p1.AuctionExtension}
	auctionStateMock.EXPECT().StartPriceAuction(now, &end).Times(2)
	err = pm.CheckPrice(context.TODO(), auctionStateMock, uint64(referencePrice)+2*maxMoveUp1, now)
	require.NoError(t, err)

	err = pm.CheckPrice(context.TODO(), auctionStateMock, uint64(referencePrice)-2*maxMoveDown1, now)
	require.NoError(t, err)

	//Still before update (no price change)
	updateTime := now.Add(time.Duration(boundUpdateFrequency) * time.Second)
	now = updateTime.Add(-time.Second)
	averagePrice1 := float64(priceHistorySum) / float64(n)
	referencePrice = averagePrice1
	validPriceToCheck = uint64(referencePrice)
	priceHistorySum = validPriceToCheck
	n = 1
	err = pm.CheckPrice(context.TODO(), auctionStateMock, validPriceToCheck, now)
	require.NoError(t, err)
	//Execting same behaviour as above (per reference price)
	validPriceToCheck = uint64(math.Floor(referencePrice)) + maxMoveUp1
	priceHistorySum += validPriceToCheck
	n++
	err = pm.CheckPrice(context.TODO(), auctionStateMock, validPriceToCheck, now)
	require.NoError(t, err)

	validPriceToCheck = uint64(math.Ceil(referencePrice)) - maxMoveDown1
	priceHistorySum += validPriceToCheck
	n++
	err = pm.CheckPrice(context.TODO(), auctionStateMock, validPriceToCheck, now)
	require.NoError(t, err)

	end = types.AuctionDuration{Duration: p1.AuctionExtension}
	auctionStateMock.EXPECT().StartPriceAuction(now, &end).Times(2)
	err = pm.CheckPrice(context.TODO(), auctionStateMock, uint64(math.Floor(referencePrice))+2*maxMoveUp1, now)
	require.NoError(t, err)

	err = pm.CheckPrice(context.TODO(), auctionStateMock, uint64(math.Ceil(referencePrice))-2*maxMoveDown1, now)
	require.NoError(t, err)

	//Right at update time
	now = updateTime
	averagePrice2 := float64(priceHistorySum) / float64(n)
	referencePrice = averagePrice2
	maxMoveDown2 := 4 * maxMoveDown1
	maxMoveUp2 := 4 * maxMoveUp1
	riskModelMock.EXPECT().PriceRange(referencePrice, horizonToYearFraction(p1.Horizon), p1.Probability).Return(averagePrice2-float64(maxMoveDown2), referencePrice+float64(maxMoveUp2))

	validPriceToCheck = uint64(referencePrice)
	priceHistorySum = validPriceToCheck
	n = 1
	err = pm.CheckPrice(context.TODO(), auctionStateMock, validPriceToCheck, now)
	require.NoError(t, err)

	validPriceToCheck = uint64(math.Floor(referencePrice)) + maxMoveUp2
	priceHistorySum += validPriceToCheck
	n++
	err = pm.CheckPrice(context.TODO(), auctionStateMock, validPriceToCheck, now)
	require.NoError(t, err)

	validPriceToCheck = uint64(math.Ceil(referencePrice)) - maxMoveDown2
	priceHistorySum += validPriceToCheck
	n++
	err = pm.CheckPrice(context.TODO(), auctionStateMock, validPriceToCheck, now)
	require.NoError(t, err)

	end = types.AuctionDuration{Duration: p1.AuctionExtension}
	auctionStateMock.EXPECT().StartPriceAuction(now, &end).Times(2)
	err = pm.CheckPrice(context.TODO(), auctionStateMock, uint64(math.Floor(referencePrice))+2*maxMoveUp2, now)
	require.NoError(t, err)

	err = pm.CheckPrice(context.TODO(), auctionStateMock, uint64(math.Ceil(referencePrice))-2*maxMoveDown2, now)
	require.NoError(t, err)

	// Right before update time (horizon away from averagePrice3)
	now = updateTime.Add(-time.Second).Add(time.Duration(p1.Horizon) * time.Second)
	averagePrice3 := float64(priceHistorySum) / float64(n)
	referencePrice = averagePrice2
	maxMoveDown3 := 6 * maxMoveDown1
	maxMoveUp3 := 6 * maxMoveUp1
	riskModelMock.EXPECT().PriceRange(averagePrice3, horizonToYearFraction(p1.Horizon), p1.Probability).Return(averagePrice3-float64(maxMoveDown3), averagePrice3+float64(maxMoveUp3))

	validPriceToCheck = uint64(referencePrice)
	priceHistorySum = validPriceToCheck
	n = 1
	err = pm.CheckPrice(context.TODO(), auctionStateMock, validPriceToCheck, now)
	require.NoError(t, err)

	validPriceToCheck = uint64(math.Floor(referencePrice)) + maxMoveUp3
	priceHistorySum += validPriceToCheck
	n++
	err = pm.CheckPrice(context.TODO(), auctionStateMock, validPriceToCheck, now)
	require.NoError(t, err)

	validPriceToCheck = uint64(math.Ceil(referencePrice)) - maxMoveDown3
	priceHistorySum += validPriceToCheck
	n++
	err = pm.CheckPrice(context.TODO(), auctionStateMock, validPriceToCheck, now)
	require.NoError(t, err)

	end = types.AuctionDuration{Duration: p1.AuctionExtension}
	auctionStateMock.EXPECT().StartPriceAuction(now, &end).Times(2)
	err = pm.CheckPrice(context.TODO(), auctionStateMock, uint64(math.Floor(referencePrice))+2*maxMoveUp3, now)
	require.NoError(t, err)

	err = pm.CheckPrice(context.TODO(), auctionStateMock, uint64(math.Ceil(referencePrice))-2*maxMoveDown3, now)
	require.NoError(t, err)

	// Right at update time (horizon away from price2Average)
	now = updateTime.Add(time.Duration(p1.Horizon) * time.Second)
	averagePrice4 := float64(priceHistorySum) / float64(n)
	referencePrice = averagePrice3
	riskModelMock.EXPECT().PriceRange(averagePrice4, horizonToYearFraction(p1.Horizon), p1.Probability).Return(averagePrice4-float64(maxMoveDown3), averagePrice4+float64(maxMoveUp3))

	validPriceToCheck = uint64(referencePrice)
	priceHistorySum = validPriceToCheck
	n = 1
	err = pm.CheckPrice(context.TODO(), auctionStateMock, validPriceToCheck, now)
	require.NoError(t, err)

	validPriceToCheck = uint64(math.Floor(referencePrice)) + maxMoveUp3
	priceHistorySum += validPriceToCheck
	n++
	err = pm.CheckPrice(context.TODO(), auctionStateMock, validPriceToCheck, now)
	require.NoError(t, err)

	validPriceToCheck = uint64(math.Ceil(referencePrice)) - maxMoveDown3
	priceHistorySum += validPriceToCheck
	n++
	err = pm.CheckPrice(context.TODO(), auctionStateMock, validPriceToCheck, now)
	require.NoError(t, err)

	end = types.AuctionDuration{Duration: p1.AuctionExtension}
	auctionStateMock.EXPECT().StartPriceAuction(now, &end).Times(2)
	err = pm.CheckPrice(context.TODO(), auctionStateMock, uint64(math.Floor(referencePrice))+2*maxMoveUp3, now)
	require.NoError(t, err)

	err = pm.CheckPrice(context.TODO(), auctionStateMock, uint64(math.Ceil(referencePrice))-2*maxMoveDown3, now)
	require.NoError(t, err)

	//Reset price, the resetting value should become the new reference
	now = now.Add(time.Hour)
	auctionStateMock.EXPECT().IsFBA().Return(false).Times(5)
	var resetPrice uint64 = 20
	var maxMoveDown4 uint64 = 5
	var maxMoveUp4 uint64 = 120
	referencePrice = float64(resetPrice)
	riskModelMock.EXPECT().PriceRange(referencePrice, horizonToYearFraction(p1.Horizon), p1.Probability).Return(referencePrice-float64(maxMoveDown4), referencePrice+float64(maxMoveUp4))

	//Assume in auction now
	auctionStateMock.EXPECT().InAuction().Return(true).Times(1)
	auctionStateMock.EXPECT().IsPriceAuction().Return(true).Times(1)
	auctionStateMock.EXPECT().IsOpeningAuction().Return(false).Times(1)
	auctionStateMock.EXPECT().Start().Return(now.Add(time.Duration(-(p1.Horizon + 1)) * time.Second)).Times(1)
	endBefore := now.Add(-time.Second)
	auctionStateMock.EXPECT().ExpiresAt().Times(1).Return(&endBefore)
	auctionStateMock.EXPECT().EndAuction().Times(1) // So that end=start+duration>now
	validPriceToCheck = resetPrice
	err = pm.CheckPrice(context.TODO(), auctionStateMock, validPriceToCheck, now)
	require.NoError(t, err)

	auctionStateMock.EXPECT().InAuction().Return(false).Times(4) // Now assume auction ended

	validPriceToCheck = resetPrice + maxMoveUp4
	err = pm.CheckPrice(context.TODO(), auctionStateMock, validPriceToCheck, now)
	require.NoError(t, err)

	validPriceToCheck = resetPrice - maxMoveDown4
	err = pm.CheckPrice(context.TODO(), auctionStateMock, validPriceToCheck, now)
	require.NoError(t, err)

	end = types.AuctionDuration{Duration: p1.AuctionExtension}
	auctionStateMock.EXPECT().StartPriceAuction(now, &end).Times(2)
	err = pm.CheckPrice(context.TODO(), auctionStateMock, uint64(math.Floor(referencePrice))+2*maxMoveUp4, now)
	require.NoError(t, err)

	err = pm.CheckPrice(context.TODO(), auctionStateMock, uint64(math.Ceil(referencePrice))-2*maxMoveDown4, now)
	require.NoError(t, err)
}

func TestMarketInOpeningAuction(t *testing.T) {
	ctrl := gomock.NewController(t)
	riskModelMock := mocks.NewMockRangeProvider(ctrl)
	auctionStateMock := mocks.NewMockAuctionState(ctrl)
	var currentPrice uint64 = 123
	p1 := types.PriceMonitoringParameters{Horizon: 7200, Probability: 0.95, AuctionExtension: 300}
	now := time.Date(1993, 2, 2, 6, 0, 0, 1, time.UTC)

	settings := types.PriceMonitoringSettings{
		PriceMonitoringParameters: []*types.PriceMonitoringParameters{&p1},
		UpdateFrequency:           1}

	riskModelMock.EXPECT().PriceRange(float64(currentPrice), gomock.Any(), gomock.Any()).Return(float64(currentPrice-10), float64(currentPrice+10)).Times(2)
	auctionStateMock.EXPECT().IsFBA().Return(false).Times(1)
	auctionStateMock.EXPECT().InAuction().Return(true).Times(1)
	auctionStateMock.EXPECT().IsOpeningAuction().Return(true).Times(1)

	pm, err := price.NewMonitor(riskModelMock, settings)
	require.NoError(t, err)
	require.NotNil(t, pm)

	err = pm.CheckPrice(context.TODO(), auctionStateMock, currentPrice, now)
	require.NoError(t, err)
}

func TestMarketInGenericAuction(t *testing.T) {
	ctrl := gomock.NewController(t)
	riskModelMock := mocks.NewMockRangeProvider(ctrl)
	auctionStateMock := mocks.NewMockAuctionState(ctrl)
	var currentPrice uint64 = 123
	p1 := types.PriceMonitoringParameters{Horizon: 7200, Probability: 0.95, AuctionExtension: 300}
	now := time.Date(1993, 2, 2, 6, 0, 0, 1, time.UTC)

	settings := types.PriceMonitoringSettings{
		PriceMonitoringParameters: []*types.PriceMonitoringParameters{&p1},
		UpdateFrequency:           1}

	var maxMoveUp uint64 = 10
	var maxMoveDown uint64 = 5
	riskModelMock.EXPECT().PriceRange(float64(currentPrice), gomock.Any(), gomock.Any()).Return(float64(currentPrice-maxMoveDown), float64(currentPrice+maxMoveUp)).Times(1)
	auctionStateMock.EXPECT().IsFBA().Return(false).Times(5)
	auctionStateMock.EXPECT().InAuction().Return(true).Times(5)
	auctionStateMock.EXPECT().IsOpeningAuction().Return(false).Times(5)
	auctionStateMock.EXPECT().IsPriceAuction().Return(false).Times(5)

	pm, err := price.NewMonitor(riskModelMock, settings)
	require.NoError(t, err)
	require.NotNil(t, pm)

	err = pm.CheckPrice(context.TODO(), auctionStateMock, currentPrice, now)
	require.NoError(t, err)

	err = pm.CheckPrice(context.TODO(), auctionStateMock, currentPrice+maxMoveUp, now)
	require.NoError(t, err)

	err = pm.CheckPrice(context.TODO(), auctionStateMock, currentPrice-maxMoveDown, now)
	require.NoError(t, err)

	extension := types.AuctionDuration{Duration: p1.AuctionExtension}
	auctionStateMock.EXPECT().ExtendAuction(extension).Times(2)

	err = pm.CheckPrice(context.TODO(), auctionStateMock, currentPrice+2*maxMoveUp, now)
	require.NoError(t, err)

	err = pm.CheckPrice(context.TODO(), auctionStateMock, currentPrice-2*maxMoveDown, now)
	require.NoError(t, err)
}

func horizonToYearFraction(horizon int64) float64 {
	return float64(horizon) / float64(365.25*24*60*60)
}
