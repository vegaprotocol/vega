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
	defer ctrl.Finish()
	riskModelMock := mocks.NewMockRangeProvider(ctrl)
	auctionStateMock := mocks.NewMockAuctionState(ctrl)
	var currentPrice uint64 = 123
	now := time.Date(1993, 2, 2, 6, 0, 0, 1, time.UTC)

	settings := types.PriceMonitoringSettings{
		Parameters: &types.PriceMonitoringParameters{
			Triggers: []*types.PriceMonitoringTrigger{},
		},
		UpdateFrequency: 1}

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
	p1 := types.PriceMonitoringTrigger{Horizon: 7200, Probability: 0.95, AuctionExtension: 300}
	p2 := types.PriceMonitoringTrigger{Horizon: 3600, Probability: 0.99, AuctionExtension: 60}

	settings := types.PriceMonitoringSettings{
		Parameters: &types.PriceMonitoringParameters{
			Triggers: []*types.PriceMonitoringTrigger{&p1, &p2},
		},
		UpdateFrequency: 600,
	}

	pm, err := price.NewMonitor(nil, settings)
	require.Error(t, err)
	require.Nil(t, pm)
}

func TestGetHorizonYearFractions(t *testing.T) {
	ctrl := gomock.NewController(t)
	riskModelMock := mocks.NewMockRangeProvider(ctrl)
	p1 := types.PriceMonitoringTrigger{Horizon: 7200, Probability: 0.95, AuctionExtension: 300}
	p2 := types.PriceMonitoringTrigger{Horizon: 3600, Probability: 0.99, AuctionExtension: 60}

	settings := types.PriceMonitoringSettings{
		Parameters: &types.PriceMonitoringParameters{
			Triggers: []*types.PriceMonitoringTrigger{&p1, &p2},
		},
		UpdateFrequency: 600,
	}

	pm, err := price.NewMonitor(riskModelMock, settings)
	require.NoError(t, err)
	require.NotNil(t, pm)

	yearFractions := pm.GetHorizonYearFractions()
	require.Equal(t, 2, len(yearFractions))
	require.Equal(t, horizonToYearFraction(p2.Horizon), yearFractions[0])
	require.Equal(t, horizonToYearFraction(p1.Horizon), yearFractions[1])
}

func TestRecordPriceChange(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	riskModelMock := mocks.NewMockRangeProvider(ctrl)
	auctionStateMock := mocks.NewMockAuctionState(ctrl)
	var currentPrice uint64 = 123
	now := time.Date(1993, 2, 2, 6, 0, 0, 1, time.UTC)
	p1 := types.PriceMonitoringTrigger{Horizon: 7200, Probability: 0.95, AuctionExtension: 300}
	p2 := types.PriceMonitoringTrigger{Horizon: 3600, Probability: 0.99, AuctionExtension: 60}
	settings := types.PriceMonitoringSettings{
		Parameters: &types.PriceMonitoringParameters{
			Triggers: []*types.PriceMonitoringTrigger{&p1, &p2},
		},
		UpdateFrequency: 600,
	}

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
	defer ctrl.Finish()
	riskModelMock := mocks.NewMockRangeProvider(ctrl)
	auctionStateMock := mocks.NewMockAuctionState(ctrl)
	var currentPrice uint64 = 123
	now := time.Date(1993, 2, 2, 6, 0, 0, 1, time.UTC)
	var p1Time int64 = 60
	var p2Time int64 = 300
	p1 := types.PriceMonitoringTrigger{Horizon: 3600, Probability: 0.99, AuctionExtension: p1Time}
	p2 := types.PriceMonitoringTrigger{Horizon: 7200, Probability: 0.95, AuctionExtension: p2Time}
	settings := types.PriceMonitoringSettings{
		Parameters: &types.PriceMonitoringParameters{
			Triggers: []*types.PriceMonitoringTrigger{&p1, &p2},
		},
		UpdateFrequency: 600,
	}

	var maxMoveDownHorizon1 uint64 = 1
	var maxMoveUpHorizon1 uint64 = 2
	var maxMoveDownHorizon2 uint64 = 3
	var maxMoveUpHorizon2 uint64 = 4
	require.True(t, maxMoveDownHorizon2 > maxMoveDownHorizon1)
	require.True(t, maxMoveUpHorizon2 > maxMoveUpHorizon1)
	riskModelMock.EXPECT().PriceRange(float64(currentPrice), horizonToYearFraction(p1.Horizon), p1.Probability).Return(float64(currentPrice-maxMoveDownHorizon1), float64(currentPrice+maxMoveUpHorizon1)).Times(6)
	riskModelMock.EXPECT().PriceRange(float64(currentPrice), horizonToYearFraction(p2.Horizon), p2.Probability).Return(float64(currentPrice-maxMoveDownHorizon2), float64(currentPrice+maxMoveUpHorizon2)).Times(6)
	auctionStateMock.EXPECT().IsFBA().Return(false).Times(16)
	auctionStateMock.EXPECT().InAuction().Return(false).Times(16)

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

	// Reinstantiate price monitoring after auction to reset internal state
	pm, err = price.NewMonitor(riskModelMock, settings)
	require.NoError(t, err)
	require.NotNil(t, pm)
	err = pm.CheckPrice(context.TODO(), auctionStateMock, currentPrice, now)
	require.NoError(t, err)

	auctionStateMock.EXPECT().StartPriceAuction(now, &end)
	err = pm.CheckPrice(context.TODO(), auctionStateMock, currentPrice-(maxMoveDownHorizon1+maxMoveDownHorizon2)/2, now)
	require.NoError(t, err)

	// Reinstantiate price monitoring after auction to reset internal state
	pm, err = price.NewMonitor(riskModelMock, settings)
	require.NoError(t, err)
	require.NotNil(t, pm)
	err = pm.CheckPrice(context.TODO(), auctionStateMock, currentPrice, now)
	require.NoError(t, err)

	auctionStateMock.EXPECT().StartPriceAuction(now, &end)
	err = pm.CheckPrice(context.TODO(), auctionStateMock, currentPrice+maxMoveUpHorizon2, now)
	require.NoError(t, err)

	// Reinstantiate price monitoring after auction to reset internal state
	pm, err = price.NewMonitor(riskModelMock, settings)
	require.NoError(t, err)
	require.NotNil(t, pm)
	err = pm.CheckPrice(context.TODO(), auctionStateMock, currentPrice, now)
	require.NoError(t, err)

	auctionStateMock.EXPECT().StartPriceAuction(now, &end)
	err = pm.CheckPrice(context.TODO(), auctionStateMock, currentPrice-maxMoveDownHorizon2, now)
	require.NoError(t, err)

	// Reinstantiate price monitoring after auction to reset internal state
	pm, err = price.NewMonitor(riskModelMock, settings)
	require.NoError(t, err)
	require.NotNil(t, pm)
	err = pm.CheckPrice(context.TODO(), auctionStateMock, currentPrice, now)
	require.NoError(t, err)

	end = types.AuctionDuration{Duration: p1.AuctionExtension + p2.AuctionExtension}
	auctionStateMock.EXPECT().StartPriceAuction(now, &end)
	err = pm.CheckPrice(context.TODO(), auctionStateMock, currentPrice+2*maxMoveUpHorizon2, now)
	require.NoError(t, err)

	// Reinstantiate price monitoring after auction to reset internal state
	pm, err = price.NewMonitor(riskModelMock, settings)
	require.NoError(t, err)
	require.NotNil(t, pm)
	err = pm.CheckPrice(context.TODO(), auctionStateMock, currentPrice, now)
	require.NoError(t, err)

	auctionStateMock.EXPECT().StartPriceAuction(now, &end)
	err = pm.CheckPrice(context.TODO(), auctionStateMock, currentPrice-2*maxMoveDownHorizon2, now)
	require.NoError(t, err)
}

func TestCheckBoundViolationsAcrossTimeWith1HorizonProbabilityPair(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	riskModelMock := mocks.NewMockRangeProvider(ctrl)
	auctionStateMock := mocks.NewMockAuctionState(ctrl)
	var price1 uint64 = 123
	initialTime := time.Date(1993, 2, 2, 6, 0, 0, 1, time.UTC)
	now := initialTime
	var p1Time int64 = 60
	p1 := types.PriceMonitoringTrigger{Horizon: 600, Probability: 0.99, AuctionExtension: p1Time}
	var boundUpdateFrequency int64 = 120
	settings := types.PriceMonitoringSettings{
		Parameters: &types.PriceMonitoringParameters{
			Triggers: []*types.PriceMonitoringTrigger{&p1},
		},
		UpdateFrequency: boundUpdateFrequency,
	}
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
	auctionStateMock.EXPECT().StartPriceAuction(now, &end).Times(1)
	err = pm.CheckPrice(context.TODO(), auctionStateMock, uint64(referencePrice)+2*maxMoveUp1, now)
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
	err = pm.CheckPrice(context.TODO(), auctionStateMock, uint64(math.Ceil(referencePrice))-2*maxMoveDown1, now)
	require.NoError(t, err)

	//Right at update time (after the auction has concluded)
	now = initialTime.Add(time.Duration(2*boundUpdateFrequency) * time.Second)
	averagePrice2 := float64(priceHistorySum) / float64(n)
	referencePrice = averagePrice2
	maxMoveDown2 := 4 * maxMoveDown1
	maxMoveUp2 := 4 * maxMoveUp1

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
	err = pm.CheckPrice(context.TODO(), auctionStateMock, uint64(math.Floor(referencePrice))+2*maxMoveUp2, now)
	require.NoError(t, err)

	// Right before update time (horizon away from averagePrice3)
	updateTime = now.Add(time.Duration(p1.Horizon) * time.Second)
	now = updateTime.Add(-time.Second)
	averagePrice3 := float64(priceHistorySum) / float64(n)
	referencePrice = averagePrice2
	maxMoveDown3 := 6 * maxMoveDown1
	maxMoveUp3 := 6 * maxMoveUp1

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
	err = pm.CheckPrice(context.TODO(), auctionStateMock, uint64(math.Floor(referencePrice))+2*maxMoveUp3, now)
	require.NoError(t, err)

	err = pm.CheckPrice(context.TODO(), auctionStateMock, uint64(math.Ceil(referencePrice))-2*maxMoveDown3, now)
	require.NoError(t, err)

	// Right at update time (horizon away from price3Average)
	now = updateTime
	referencePrice = averagePrice3

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
	err = pm.CheckPrice(context.TODO(), auctionStateMock, uint64(math.Ceil(referencePrice))-2*maxMoveDown3, now)
	require.NoError(t, err)

	//Reset price, the resetting value should become the new reference
	now = now.Add(time.Hour)
	var resetPrice uint64 = 20
	var maxMoveDown4 uint64 = 5
	var maxMoveUp4 uint64 = 120
	referencePrice = float64(resetPrice)

	//Assume in auction now
	validPriceToCheck = resetPrice
	err = pm.CheckPrice(context.TODO(), auctionStateMock, validPriceToCheck, now)
	require.NoError(t, err)

	validPriceToCheck = resetPrice + maxMoveUp4
	err = pm.CheckPrice(context.TODO(), auctionStateMock, validPriceToCheck, now)
	require.NoError(t, err)

	validPriceToCheck = resetPrice - maxMoveDown4
	err = pm.CheckPrice(context.TODO(), auctionStateMock, validPriceToCheck, now)
	require.NoError(t, err)

	end = types.AuctionDuration{Duration: p1.AuctionExtension}
	err = pm.CheckPrice(context.TODO(), auctionStateMock, uint64(referencePrice)+2*maxMoveUp4, now)
	require.NoError(t, err)
}

func TestAuctionStartedAndEndendBy1Trigger(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	riskModelMock := mocks.NewMockRangeProvider(ctrl)
	auctionStateMock := mocks.NewMockAuctionState(ctrl)
	var price1 uint64 = 123
	now := time.Date(1993, 2, 2, 6, 0, 0, 1, time.UTC)
	p1 := types.PriceMonitoringTrigger{Horizon: 600, Probability: 0.95, AuctionExtension: 60}
	p2 := types.PriceMonitoringTrigger{Horizon: 600, Probability: 0.99, AuctionExtension: 120}
	var boundUpdateFrequency int64 = 120
	settings := types.PriceMonitoringSettings{
		Parameters: &types.PriceMonitoringParameters{
			Triggers: []*types.PriceMonitoringTrigger{&p1, &p2},
		},
		UpdateFrequency: boundUpdateFrequency,
	}
	var maxMoveDownP1 uint64 = 1
	var maxMoveUpP1 uint64 = 2
	var p2Multiplier uint64 = 4
	require.True(t, (p2Multiplier-1) > 1)

	riskModelMock.EXPECT().PriceRange(float64(price1), horizonToYearFraction(p1.Horizon), p1.Probability).Return(float64(price1-maxMoveDownP1), float64(price1+maxMoveUpP1)).Times(1)
	riskModelMock.EXPECT().PriceRange(float64(price1), horizonToYearFraction(p2.Horizon), p2.Probability).Return(float64(price1-(p2Multiplier*maxMoveDownP1)), float64(price1+(p2Multiplier*maxMoveUpP1))).Times(1)
	auctionStateMock.EXPECT().IsFBA().Return(false).Times(2)
	auctionStateMock.EXPECT().InAuction().Return(false).Times(2)

	pm, err := price.NewMonitor(riskModelMock, settings)
	require.NoError(t, err)
	require.NotNil(t, pm)
	var priceHistorySum uint64 = 0
	n := 0
	referencePrice := float64(price1)
	priceToCheck := uint64(referencePrice)
	priceHistorySum += priceToCheck
	n++
	err = pm.CheckPrice(context.TODO(), auctionStateMock, price1, now)
	require.NoError(t, err)

	end := types.AuctionDuration{Duration: p1.AuctionExtension}
	auctionStateMock.EXPECT().StartPriceAuction(now, &end).Times(1)
	p1ViolatingPrice := uint64(referencePrice) + (p2Multiplier-1)*maxMoveUpP1
	err = pm.CheckPrice(context.TODO(), auctionStateMock, p1ViolatingPrice, now) //P1 violated only
	require.NoError(t, err)

	initialAuctionEnd := now.Add(time.Duration(p1.AuctionExtension) * time.Second)

	auctionStateMock.EXPECT().IsFBA().Return(false).Times(1)
	auctionStateMock.EXPECT().InAuction().Return(true).Times(1)
	auctionStateMock.EXPECT().IsOpeningAuction().Return(false).Times(1)
	auctionStateMock.EXPECT().IsPriceAuction().Return(true).Times(1)
	auctionStateMock.EXPECT().ExpiresAt().Return(&initialAuctionEnd).Times(1)
	auctionStateMock.EXPECT().EndAuction().Times(1)
	riskModelMock.EXPECT().PriceRange(float64(p1ViolatingPrice), horizonToYearFraction(p1.Horizon), p1.Probability).Return(float64(p1ViolatingPrice-maxMoveDownP1), float64(p1ViolatingPrice+maxMoveUpP1)).Times(1)
	riskModelMock.EXPECT().PriceRange(float64(p1ViolatingPrice), horizonToYearFraction(p2.Horizon), p2.Probability).Return(float64(p1ViolatingPrice-(p2Multiplier*maxMoveDownP1)), float64(p1ViolatingPrice+(p2Multiplier*maxMoveUpP1))).Times(1)

	afterInitialAuction := initialAuctionEnd.Add(time.Nanosecond)
	err = pm.CheckPrice(context.TODO(), auctionStateMock, p1ViolatingPrice, afterInitialAuction) //price should be accepted now
	require.NoError(t, err)
}

func TestAuctionStartedAndEndendBy2Triggers(t *testing.T) {
	ctrl := gomock.NewController(t)
	riskModelMock := mocks.NewMockRangeProvider(ctrl)
	auctionStateMock := mocks.NewMockAuctionState(ctrl)
	var price1 uint64 = 123
	now := time.Date(1993, 2, 2, 6, 0, 0, 1, time.UTC)
	p1 := types.PriceMonitoringTrigger{Horizon: 600, Probability: 0.95, AuctionExtension: 60}
	p2 := types.PriceMonitoringTrigger{Horizon: 600, Probability: 0.99, AuctionExtension: 120}
	var boundUpdateFrequency int64 = 120
	settings := types.PriceMonitoringSettings{
		Parameters: &types.PriceMonitoringParameters{
			Triggers: []*types.PriceMonitoringTrigger{&p1, &p2},
		},
		UpdateFrequency: boundUpdateFrequency,
	}
	var maxMoveDownP1 uint64 = 1
	var maxMoveUpP1 uint64 = 2
	var p2Multiplier uint64 = 4
	require.True(t, (p2Multiplier-1) > 1)

	riskModelMock.EXPECT().PriceRange(float64(price1), horizonToYearFraction(p1.Horizon), p1.Probability).Return(float64(price1-maxMoveDownP1), float64(price1+maxMoveUpP1)).Times(1)
	riskModelMock.EXPECT().PriceRange(float64(price1), horizonToYearFraction(p2.Horizon), p2.Probability).Return(float64(price1-(p2Multiplier*maxMoveDownP1)), float64(price1+(p2Multiplier*maxMoveUpP1))).Times(1)
	auctionStateMock.EXPECT().IsFBA().Return(false).Times(2)
	auctionStateMock.EXPECT().InAuction().Return(false).Times(2)

	pm, err := price.NewMonitor(riskModelMock, settings)
	require.NoError(t, err)
	require.NotNil(t, pm)
	var priceHistorySum uint64 = 0
	n := 0
	referencePrice := float64(price1)
	priceToCheck := uint64(referencePrice)
	priceHistorySum += priceToCheck
	n++
	err = pm.CheckPrice(context.TODO(), auctionStateMock, price1, now)
	require.NoError(t, err)

	end := types.AuctionDuration{Duration: p1.AuctionExtension + p2.AuctionExtension}
	auctionStateMock.EXPECT().StartPriceAuction(now, &end).Times(1)
	p2ViolatingPrice := uint64(referencePrice) + (p2Multiplier+1)*maxMoveUpP1
	err = pm.CheckPrice(context.TODO(), auctionStateMock, p2ViolatingPrice, now) //P1 violated only
	require.NoError(t, err)

	initialAuctionEnd := now.Add(time.Duration(p1.AuctionExtension+p2.AuctionExtension) * time.Second)

	auctionStateMock.EXPECT().IsFBA().Return(false).Times(1)
	auctionStateMock.EXPECT().InAuction().Return(true).Times(1)
	auctionStateMock.EXPECT().IsOpeningAuction().Return(false).Times(1)
	auctionStateMock.EXPECT().IsPriceAuction().Return(true).Times(1)
	auctionStateMock.EXPECT().ExpiresAt().Return(&initialAuctionEnd).Times(1)
	auctionStateMock.EXPECT().EndAuction().Times(1)
	riskModelMock.EXPECT().PriceRange(float64(p2ViolatingPrice), horizonToYearFraction(p1.Horizon), p1.Probability).Return(float64(p2ViolatingPrice-maxMoveDownP1), float64(p2ViolatingPrice+maxMoveUpP1)).Times(1)
	riskModelMock.EXPECT().PriceRange(float64(p2ViolatingPrice), horizonToYearFraction(p2.Horizon), p2.Probability).Return(float64(p2ViolatingPrice-(p2Multiplier*maxMoveDownP1)), float64(p2ViolatingPrice+(p2Multiplier*maxMoveUpP1))).Times(1)

	afterInitialAuction := initialAuctionEnd.Add(time.Nanosecond)
	err = pm.CheckPrice(context.TODO(), auctionStateMock, p2ViolatingPrice, afterInitialAuction) //price should be accepted now
	require.NoError(t, err)
}

func TestAuctionStartedAndEndendBy1TriggerAndExtendedBy2nd(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	riskModelMock := mocks.NewMockRangeProvider(ctrl)
	auctionStateMock := mocks.NewMockAuctionState(ctrl)
	var price1 uint64 = 123
	now := time.Date(1993, 2, 2, 6, 0, 0, 1, time.UTC)
	p1 := types.PriceMonitoringTrigger{Horizon: 600, Probability: 0.95, AuctionExtension: 60}
	p2 := types.PriceMonitoringTrigger{Horizon: 600, Probability: 0.99, AuctionExtension: 120}
	var boundUpdateFrequency int64 = 120
	settings := types.PriceMonitoringSettings{
		Parameters: &types.PriceMonitoringParameters{
			Triggers: []*types.PriceMonitoringTrigger{&p1, &p2},
		},
		UpdateFrequency: boundUpdateFrequency,
	}
	var maxMoveDownP1 uint64 = 1
	var maxMoveUpP1 uint64 = 2
	var p2Multiplier uint64 = 4
	require.True(t, (p2Multiplier-1) > 1)

	riskModelMock.EXPECT().PriceRange(float64(price1), horizonToYearFraction(p1.Horizon), p1.Probability).Return(float64(price1-maxMoveDownP1), float64(price1+maxMoveUpP1)).Times(1)
	riskModelMock.EXPECT().PriceRange(float64(price1), horizonToYearFraction(p2.Horizon), p2.Probability).Return(float64(price1-(p2Multiplier*maxMoveDownP1)), float64(price1+(p2Multiplier*maxMoveUpP1))).Times(1)
	auctionStateMock.EXPECT().IsFBA().Return(false).Times(2)
	auctionStateMock.EXPECT().InAuction().Return(false).Times(2)

	pm, err := price.NewMonitor(riskModelMock, settings)
	require.NoError(t, err)
	require.NotNil(t, pm)
	var priceHistorySum uint64 = 0
	n := 0
	referencePrice := float64(price1)
	priceToCheck := uint64(referencePrice)
	priceHistorySum += priceToCheck
	n++
	err = pm.CheckPrice(context.TODO(), auctionStateMock, price1, now)
	require.NoError(t, err)

	end := types.AuctionDuration{Duration: p1.AuctionExtension}
	auctionStateMock.EXPECT().StartPriceAuction(now, &end).Times(1)
	p1ViolatingPrice := uint64(referencePrice) + (p2Multiplier-1)*maxMoveUpP1
	err = pm.CheckPrice(context.TODO(), auctionStateMock, p1ViolatingPrice, now) //P1 violated only
	require.NoError(t, err)

	initialAuctionEnd := now.Add(time.Duration(p1.AuctionExtension) * time.Second)

	auctionStateMock.EXPECT().IsFBA().Return(false).Times(1)
	auctionStateMock.EXPECT().InAuction().Return(true).Times(1)
	auctionStateMock.EXPECT().IsOpeningAuction().Return(false).Times(1)
	auctionStateMock.EXPECT().IsPriceAuction().Return(true).Times(1)
	auctionStateMock.EXPECT().ExpiresAt().Return(&initialAuctionEnd).Times(1)

	afterInitialAuction := initialAuctionEnd.Add(time.Nanosecond)
	now = afterInitialAuction
	p2ViolatingPrice := uint64(referencePrice) + (p2Multiplier+1)*maxMoveUpP1
	end2 := types.AuctionDuration{Duration: p2.AuctionExtension}
	auctionStateMock.EXPECT().ExtendAuction(end2).Times(1)
	err = pm.CheckPrice(context.TODO(), auctionStateMock, p2ViolatingPrice, afterInitialAuction) //price should violated 2nd trigger and result in auction extension
	require.NoError(t, err)

	extendedAuctionEnd := now.Add(time.Duration(p1.AuctionExtension+p2.AuctionExtension) * time.Second)

	auctionStateMock.EXPECT().IsFBA().Return(false).Times(1)
	auctionStateMock.EXPECT().InAuction().Return(true).Times(1)
	auctionStateMock.EXPECT().IsOpeningAuction().Return(false).Times(1)
	auctionStateMock.EXPECT().EndAuction().Times(1)
	riskModelMock.EXPECT().PriceRange(float64(p2ViolatingPrice), horizonToYearFraction(p1.Horizon), p1.Probability).Return(float64(p2ViolatingPrice-maxMoveDownP1), float64(p2ViolatingPrice+maxMoveUpP1)).Times(1)
	riskModelMock.EXPECT().PriceRange(float64(p2ViolatingPrice), horizonToYearFraction(p2.Horizon), p2.Probability).Return(float64(p2ViolatingPrice-(p2Multiplier*maxMoveDownP1)), float64(p2ViolatingPrice+(p2Multiplier*maxMoveUpP1))).Times(1)

	afterExtendedAuction := extendedAuctionEnd.Add(time.Nanosecond)
	err = pm.CheckPrice(context.TODO(), auctionStateMock, p2ViolatingPrice, afterExtendedAuction) //price should be accepted now
	require.NoError(t, err)
}

func TestMarketInOpeningAuction(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	riskModelMock := mocks.NewMockRangeProvider(ctrl)
	auctionStateMock := mocks.NewMockAuctionState(ctrl)
	var currentPrice uint64 = 123
	p1 := types.PriceMonitoringTrigger{Horizon: 7200, Probability: 0.95, AuctionExtension: 300}
	now := time.Date(1993, 2, 2, 6, 0, 0, 1, time.UTC)
	settings := types.PriceMonitoringSettings{
		Parameters: &types.PriceMonitoringParameters{
			Triggers: []*types.PriceMonitoringTrigger{&p1},
		},
		UpdateFrequency: 1,
	}

	riskModelMock.EXPECT().PriceRange(float64(currentPrice), gomock.Any(), gomock.Any()).Return(float64(currentPrice-10), float64(currentPrice+10)).Times(1)
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
	defer ctrl.Finish()
	riskModelMock := mocks.NewMockRangeProvider(ctrl)
	auctionStateMock := mocks.NewMockAuctionState(ctrl)
	var currentPrice uint64 = 123
	p1 := types.PriceMonitoringTrigger{Horizon: 7200, Probability: 0.95, AuctionExtension: 300}
	now := time.Date(1993, 2, 2, 6, 0, 0, 1, time.UTC)

	settings := types.PriceMonitoringSettings{
		Parameters: &types.PriceMonitoringParameters{
			Triggers: []*types.PriceMonitoringTrigger{&p1},
		},
		UpdateFrequency: 1,
	}

	var maxMoveUp uint64 = 10
	var maxMoveDown uint64 = 5
	riskModelMock.EXPECT().PriceRange(float64(currentPrice), gomock.Any(), gomock.Any()).Return(float64(currentPrice-maxMoveDown), float64(currentPrice+maxMoveUp)).Times(1)
	auctionStateMock.EXPECT().IsFBA().Return(false).Times(5)
	auctionStateMock.EXPECT().InAuction().Return(true).Times(5)
	auctionStateMock.EXPECT().IsOpeningAuction().Return(false).Times(5)
	auctionStateMock.EXPECT().IsPriceAuction().Return(false).Times(4)

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
	auctionStateMock.EXPECT().ExtendAuction(extension).Times(1)

	err = pm.CheckPrice(context.TODO(), auctionStateMock, currentPrice+2*maxMoveUp, now)
	require.NoError(t, err)

	err = pm.CheckPrice(context.TODO(), auctionStateMock, currentPrice-2*maxMoveDown, now)
	require.NoError(t, err)
}

func horizonToYearFraction(horizon int64) float64 {
	return float64(horizon) / float64(365.25*24*60*60)
}

func TestGetValidPriceRange_NoTriggers(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	riskModelMock := mocks.NewMockRangeProvider(ctrl)
	auctionStateMock := mocks.NewMockAuctionState(ctrl)
	var currentPrice uint64 = 123
	now := time.Date(1993, 2, 2, 6, 0, 0, 1, time.UTC)

	settings := types.PriceMonitoringSettings{
		Parameters: &types.PriceMonitoringParameters{
			Triggers: []*types.PriceMonitoringTrigger{},
		},
		UpdateFrequency: 1}

	auctionStateMock.EXPECT().IsFBA().Return(false).Times(1)
	auctionStateMock.EXPECT().InAuction().Return(false).Times(1)

	pm, err := price.NewMonitor(riskModelMock, settings)
	require.NoError(t, err)
	require.NotNil(t, pm)

	min, max := pm.GetValidPriceRange()
	require.Equal(t, -math.MaxFloat64, min)
	require.Equal(t, math.MaxFloat64, max)

	err = pm.CheckPrice(context.TODO(), auctionStateMock, currentPrice, now)
	require.NoError(t, err)

	min, max = pm.GetValidPriceRange()
	require.Equal(t, -math.MaxFloat64, min)
	require.Equal(t, math.MaxFloat64, max)
}

func TestGetValidPriceRange_2triggers(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	riskModelMock := mocks.NewMockRangeProvider(ctrl)
	auctionStateMock := mocks.NewMockAuctionState(ctrl)
	var currentPrice uint64 = 123
	now := time.Date(1993, 2, 2, 6, 0, 0, 1, time.UTC)
	var p1Time int64 = 60
	var p2Time int64 = 300
	p1 := types.PriceMonitoringTrigger{Horizon: 3600, Probability: 0.99, AuctionExtension: p1Time}
	p2 := types.PriceMonitoringTrigger{Horizon: 7200, Probability: 0.95, AuctionExtension: p2Time}
	settings := types.PriceMonitoringSettings{
		Parameters: &types.PriceMonitoringParameters{
			Triggers: []*types.PriceMonitoringTrigger{&p1, &p2},
		},
		UpdateFrequency: 600,
	}

	var maxMoveDownHorizon1 uint64 = 1
	var maxMoveUpHorizon1 uint64 = 2
	var maxMoveDownHorizon2 uint64 = 3
	var maxMoveUpHorizon2 uint64 = 4
	require.True(t, maxMoveDownHorizon2 > maxMoveDownHorizon1)
	require.True(t, maxMoveUpHorizon2 > maxMoveUpHorizon1)
	riskModelMock.EXPECT().PriceRange(float64(currentPrice), horizonToYearFraction(p1.Horizon), p1.Probability).Return(float64(currentPrice-maxMoveDownHorizon1), float64(currentPrice+maxMoveUpHorizon1)).Times(2)
	riskModelMock.EXPECT().PriceRange(float64(currentPrice), horizonToYearFraction(p2.Horizon), p2.Probability).Return(float64(currentPrice-maxMoveDownHorizon2), float64(currentPrice+maxMoveUpHorizon2)).Times(2)
	auctionStateMock.EXPECT().IsFBA().Return(false).Times(12)
	auctionStateMock.EXPECT().InAuction().Return(false).Times(12)

	pm, err := price.NewMonitor(riskModelMock, settings)
	require.NoError(t, err)
	require.NotNil(t, pm)

	err = pm.CheckPrice(context.TODO(), auctionStateMock, currentPrice, now)
	require.NoError(t, err)

	now = now.Add(time.Second)
	err = pm.CheckPrice(context.TODO(), auctionStateMock, currentPrice+maxMoveUpHorizon1-1, now)
	require.NoError(t, err)

	now = now.Add(time.Minute)
	err = pm.CheckPrice(context.TODO(), auctionStateMock, currentPrice-maxMoveDownHorizon1+1, now)
	require.NoError(t, err)

	now = now.Add(time.Hour)
	err = pm.CheckPrice(context.TODO(), auctionStateMock, currentPrice+maxMoveUpHorizon1, now)
	require.NoError(t, err)

	now = now.Add(time.Minute)
	err = pm.CheckPrice(context.TODO(), auctionStateMock, currentPrice-maxMoveDownHorizon1, now)
	require.NoError(t, err)

	min, max := pm.GetValidPriceRange()
	minInt := uint64(math.Ceil(min))
	maxInt := uint64(math.Floor(max))

	err = pm.CheckPrice(context.TODO(), auctionStateMock, minInt, now)
	require.NoError(t, err)

	err = pm.CheckPrice(context.TODO(), auctionStateMock, maxInt, now)
	require.NoError(t, err)

	// Should trigger an auction
	auctionStateMock.EXPECT().StartPriceAuction(now, gomock.Any()).Times(1)
	err = pm.CheckPrice(context.TODO(), auctionStateMock, minInt-1, now)
	require.NoError(t, err)

	now = now.Add(time.Second)
	err = pm.CheckPrice(context.TODO(), auctionStateMock, currentPrice, now)
	require.NoError(t, err)

	min, max = pm.GetValidPriceRange()
	minInt = uint64(math.Ceil(min))
	maxInt = uint64(math.Floor(max))

	err = pm.CheckPrice(context.TODO(), auctionStateMock, minInt, now)
	require.NoError(t, err)

	err = pm.CheckPrice(context.TODO(), auctionStateMock, maxInt, now)
	require.NoError(t, err)

	// Should trigger an auction
	auctionStateMock.EXPECT().StartPriceAuction(now, gomock.Any()).Times(1)
	err = pm.CheckPrice(context.TODO(), auctionStateMock, maxInt+1, now)
	require.NoError(t, err)
}

func TestPricesValidAfterAuctionEnds(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	riskModelMock := mocks.NewMockRangeProvider(ctrl)
	auctionStateMock := mocks.NewMockAuctionState(ctrl)
	var price1 uint64 = 123
	now := time.Date(1993, 2, 2, 6, 0, 0, 1, time.UTC)
	p1 := types.PriceMonitoringTrigger{Horizon: 600, Probability: 0.95, AuctionExtension: 60}
	var boundUpdateFrequency int64 = 120
	settings := types.PriceMonitoringSettings{
		Parameters: &types.PriceMonitoringParameters{
			Triggers: []*types.PriceMonitoringTrigger{&p1},
		},
		UpdateFrequency: boundUpdateFrequency,
	}
	var maxMoveDownP1 uint64 = 1
	var maxMoveUpP1 uint64 = 2
	var p2Multiplier uint64 = 4
	require.True(t, (p2Multiplier-1) > 1)

	riskModelMock.EXPECT().PriceRange(float64(price1), horizonToYearFraction(p1.Horizon), p1.Probability).Return(float64(price1-maxMoveDownP1), float64(price1+maxMoveUpP1)).Times(1)
	auctionStateMock.EXPECT().IsFBA().Return(false).Times(2)
	auctionStateMock.EXPECT().InAuction().Return(false).Times(2)

	pm, err := price.NewMonitor(riskModelMock, settings)
	require.NoError(t, err)
	require.NotNil(t, pm)
	var priceHistorySum uint64 = 0
	n := 0
	referencePrice := float64(price1)
	priceToCheck := uint64(referencePrice)
	priceHistorySum += priceToCheck
	n++
	err = pm.CheckPrice(context.TODO(), auctionStateMock, price1, now)
	require.NoError(t, err)

	end := types.AuctionDuration{Duration: p1.AuctionExtension}
	auctionStateMock.EXPECT().StartPriceAuction(now, &end).Times(1)
	p1ViolatingPrice := uint64(referencePrice) + (p2Multiplier-1)*maxMoveUpP1
	err = pm.CheckPrice(context.TODO(), auctionStateMock, p1ViolatingPrice, now) //P1 violated only
	require.NoError(t, err)

	initialAuctionEnd := now.Add(time.Duration(p1.AuctionExtension) * time.Second)

	auctionStateMock.EXPECT().IsFBA().Return(false).Times(1)
	auctionStateMock.EXPECT().InAuction().Return(true).Times(1)
	auctionStateMock.EXPECT().IsOpeningAuction().Return(false).Times(1)
	auctionStateMock.EXPECT().IsPriceAuction().Return(true).Times(1)
	auctionStateMock.EXPECT().ExpiresAt().Return(&initialAuctionEnd).Times(1)
	auctionStateMock.EXPECT().EndAuction().Times(1)
	riskModelMock.EXPECT().PriceRange(float64(p1ViolatingPrice), horizonToYearFraction(p1.Horizon), p1.Probability).Return(float64(p1ViolatingPrice-maxMoveDownP1), float64(p1ViolatingPrice+maxMoveUpP1)).Times(1)

	afterInitialAuction := initialAuctionEnd.Add(time.Nanosecond)
	err = pm.CheckPrice(context.TODO(), auctionStateMock, p1ViolatingPrice, afterInitialAuction) //price should be accepted now
	require.NoError(t, err)

	auctionStateMock.EXPECT().IsFBA().Return(false).Times(1)
	auctionStateMock.EXPECT().InAuction().Return(false).Times(1)

	err = pm.CheckPrice(context.Background(), auctionStateMock, p1ViolatingPrice, afterInitialAuction) //price should be accepted now
	require.NoError(t, err)
}
