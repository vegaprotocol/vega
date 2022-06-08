package price_test

import (
	"context"
	"testing"
	"time"

	proto "code.vegaprotocol.io/protos/vega"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/monitor/price"
	"code.vegaprotocol.io/vega/monitor/price/mocks"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func TestEmptyParametersList(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	riskModel := mocks.NewMockRangeProvider(ctrl)
	auctionStateMock := mocks.NewMockAuctionState(ctrl)
	currentPrice := []*types.Trade{{Price: num.NewUint(123), Size: 1}}

	now := time.Date(1993, 2, 2, 6, 0, 0, 1, time.UTC)

	settings := &types.PriceMonitoringSettings{
		Parameters: &types.PriceMonitoringParameters{
			Triggers: []*types.PriceMonitoringTrigger{},
		},
		UpdateFrequency: 1,
	}

	auctionStateMock.EXPECT().IsFBA().Return(false).Times(4)
	auctionStateMock.EXPECT().InAuction().Return(false).Times(4)
	statevar := mocks.NewMockStateVarEngine(ctrl)
	statevar.EXPECT().RegisterStateVariable(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any())

	pm, err := price.NewMonitor("asset", "market", riskModel, settings, statevar, logging.NewTestLogger())
	require.NoError(t, err)
	require.NotNil(t, pm)

	pm.OnTimeUpdate(now)
	b := pm.CheckPrice(context.TODO(), auctionStateMock, currentPrice, true)
	require.False(t, b)

	pm.OnTimeUpdate(now.Add(time.Second))
	b = pm.CheckPrice(context.TODO(), auctionStateMock, currentPrice, true)
	require.False(t, b)

	pm.OnTimeUpdate(now.Add(time.Minute))
	b = pm.CheckPrice(context.TODO(), auctionStateMock, currentPrice, true)
	require.False(t, b)

	pm.OnTimeUpdate(now.Add(time.Hour))
	b = pm.CheckPrice(context.TODO(), auctionStateMock, currentPrice, true)
	require.False(t, b)
}

func TestErrorWithNilRiskModel(t *testing.T) {
	t1 := proto.PriceMonitoringTrigger{Horizon: 7200, Probability: "0.95", AuctionExtension: 300}
	t2 := proto.PriceMonitoringTrigger{Horizon: 3600, Probability: "0.99", AuctionExtension: 60}

	pSet := &proto.PriceMonitoringSettings{
		Parameters: &proto.PriceMonitoringParameters{
			Triggers: []*proto.PriceMonitoringTrigger{&t1, &t2},
		},
		UpdateFrequency: 600,
	}
	settings := types.PriceMonitoringSettingsFromProto(pSet)
	ctrl := gomock.NewController(t)
	statevar := mocks.NewMockStateVarEngine(ctrl)
	// statevar.EXPECT().RegisterStateVariable(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any())
	pm, err := price.NewMonitor("asset", "market", nil, settings, statevar, logging.NewTestLogger())
	require.Error(t, err)
	require.Nil(t, pm)
}

func TestGetHorizonYearFractions(t *testing.T) {
	ctrl := gomock.NewController(t)
	riskModel := mocks.NewMockRangeProvider(ctrl)
	t1 := proto.PriceMonitoringTrigger{Horizon: 7200, Probability: "0.95", AuctionExtension: 300}
	t2 := proto.PriceMonitoringTrigger{Horizon: 3600, Probability: "0.99", AuctionExtension: 60}

	pSet := &proto.PriceMonitoringSettings{
		Parameters: &proto.PriceMonitoringParameters{
			Triggers: []*proto.PriceMonitoringTrigger{&t1, &t2},
		},
		UpdateFrequency: 600,
	}
	settings := types.PriceMonitoringSettingsFromProto(pSet)
	statevar := mocks.NewMockStateVarEngine(ctrl)
	statevar.EXPECT().RegisterStateVariable(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any())
	pm, err := price.NewMonitor("asset", "market", riskModel, settings, statevar, logging.NewTestLogger())
	require.NoError(t, err)
	require.NotNil(t, pm)

	yearFractions := pm.GetHorizonYearFractions()
	require.Equal(t, 2, len(yearFractions))
	require.Equal(t, horizonToYearFraction(t2.Horizon), yearFractions[0])
	require.Equal(t, horizonToYearFraction(t1.Horizon), yearFractions[1])
}

func TestRecordPriceChange(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	riskModel := mocks.NewMockRangeProvider(ctrl)
	auctionStateMock := mocks.NewMockAuctionState(ctrl)
	currentPrice := num.NewUint(123)
	cp := []*types.Trade{{Price: currentPrice, Size: 1}}
	now := time.Date(1993, 2, 2, 6, 0, 0, 1, time.UTC)
	t1 := proto.PriceMonitoringTrigger{Horizon: 7200, Probability: "0.95", AuctionExtension: 300}
	t2 := proto.PriceMonitoringTrigger{Horizon: 3600, Probability: "0.99", AuctionExtension: 60}

	pSet := &proto.PriceMonitoringSettings{
		Parameters: &proto.PriceMonitoringParameters{
			Triggers: []*proto.PriceMonitoringTrigger{&t1, &t2},
		},
		UpdateFrequency: 600,
	}
	settings := types.PriceMonitoringSettingsFromProto(pSet)

	auctionStateMock.EXPECT().IsFBA().Return(false).Times(4)
	auctionStateMock.EXPECT().InAuction().Return(false).Times(4)
	statevar := mocks.NewMockStateVarEngine(ctrl)
	statevar.EXPECT().RegisterStateVariable(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any())

	pm, err := price.NewMonitor("asset", "market", riskModel, settings, statevar, logging.NewTestLogger())
	require.NoError(t, err)
	require.NotNil(t, pm)

	pm.OnTimeUpdate(now)
	b := pm.CheckPrice(context.TODO(), auctionStateMock, cp, true)
	require.False(t, b)
	one := num.NewUint(1)
	cp1 := []*types.Trade{{Price: num.Sum(currentPrice, one), Size: 1}}      // plus 1
	cp2 := []*types.Trade{{Price: num.Sum(currentPrice, one, one), Size: 1}} // plus 2
	b = pm.CheckPrice(context.TODO(), auctionStateMock, cp2, true)
	require.False(t, b)
	b = pm.CheckPrice(context.TODO(), auctionStateMock, cp1, true)
	require.False(t, b)
	b = pm.CheckPrice(context.TODO(), auctionStateMock, cp, true)
	require.False(t, b)
}

func TestCheckBoundViolationsWithinCurrentTimeWith2HorizonProbabilityPairs(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	riskModel := mocks.NewMockRangeProvider(ctrl)
	auctionStateMock := mocks.NewMockAuctionState(ctrl)
	currentPrice := num.NewUint(123)
	cp := []*types.Trade{{Price: currentPrice, Size: 1}}
	now := time.Date(1993, 2, 2, 6, 0, 0, 1, time.UTC)
	t1Time, t2Time := int64(60), int64(300)
	t1 := proto.PriceMonitoringTrigger{Horizon: 3600, Probability: "0.99", AuctionExtension: t1Time}
	t2 := proto.PriceMonitoringTrigger{Horizon: 7200, Probability: "0.95", AuctionExtension: t2Time}
	pSet := &proto.PriceMonitoringSettings{
		Parameters: &proto.PriceMonitoringParameters{
			Triggers: []*proto.PriceMonitoringTrigger{&t1, &t2},
		},
		UpdateFrequency: 600,
	}
	settings := types.PriceMonitoringSettingsFromProto(pSet)

	maxDown1, maxUp1, maxDown2, maxUp2 := num.NewUint(1), num.NewUint(2), num.NewUint(3), num.NewUint(4)

	cpDec := num.DecimalFromUint(currentPrice)
	// get the price bounds
	pMin1 := cpDec.Sub(num.DecimalFromUint(maxDown1))
	pMin2 := cpDec.Sub(num.DecimalFromUint(maxDown2))
	pMax1 := cpDec.Add(num.DecimalFromUint(maxUp1))
	pMax2 := cpDec.Add(num.DecimalFromUint(maxUp2))
	one := num.NewUint(1) // 1, just to tweak prices when calling CheckPrice
	require.True(t, maxDown2.GT(maxDown1))
	require.True(t, maxUp2.GT(maxUp1))

	downFactors := []num.Decimal{pMin1.Div(cpDec), pMin2.Div(cpDec)}
	upFactors := []num.Decimal{pMax1.Div(cpDec), pMax2.Div(cpDec)}

	auctionStateMock.EXPECT().IsFBA().Return(false).Times(16)
	auctionStateMock.EXPECT().InAuction().Return(false).Times(16)
	statevar := mocks.NewMockStateVarEngine(ctrl)
	statevar.EXPECT().RegisterStateVariable(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

	pm, err := price.NewMonitor("asset", "market", riskModel, settings, statevar, logging.NewTestLogger())
	require.NoError(t, err)
	require.NotNil(t, pm)
	pm.UpdateTestFactors(downFactors, upFactors)

	pm.OnTimeUpdate(now)
	b := pm.CheckPrice(context.TODO(), auctionStateMock, cp, true)
	require.False(t, b)

	cPrice := num.Sum(currentPrice, maxUp1)
	cPrice = cPrice.Sub(cPrice, one)
	cp1 := []*types.Trade{{Price: cPrice, Size: 1}}
	b = pm.CheckPrice(context.TODO(), auctionStateMock, cp1, true)
	require.False(t, b)

	cPrice = num.Sum(currentPrice, one)
	cPrice = cPrice.Sub(cPrice, maxDown1)
	cp2 := []*types.Trade{{Price: cPrice, Size: 1}}
	b = pm.CheckPrice(context.TODO(), auctionStateMock, cp2, true)
	require.False(t, b)

	b = pm.CheckPrice(context.TODO(), auctionStateMock, cp2, true)
	require.False(t, b)

	cPrice = num.Sum(one, cPrice.Sub(currentPrice, maxDown1)) // add one bc price bounds are now using Ceil for min price
	cp3 := []*types.Trade{{Price: cPrice, Size: 1}}
	b = pm.CheckPrice(context.TODO(), auctionStateMock, cp3, true)
	require.False(t, b)

	// set the min duration to equal auction extension 1
	pm.SetMinDuration(time.Duration(t1.AuctionExtension) * time.Second)
	end := types.AuctionDuration{Duration: t1.AuctionExtension}
	auctionStateMock.EXPECT().StartPriceAuction(now, &end).Times(1)

	delta := num.Sum(maxUp1, maxUp2)
	cPrice = num.Sum(currentPrice, delta.Div(delta, num.Sum(one, one)))
	cp4 := []*types.Trade{{Price: cPrice, Size: 1}}
	b = pm.CheckPrice(context.TODO(), auctionStateMock, cp4, true)
	require.False(t, b)

	// Reinstantiate price monitoring after auction to reset internal state
	pm, err = price.NewMonitor("asset", "market", riskModel, settings, statevar, logging.NewTestLogger())
	require.NoError(t, err)
	require.NotNil(t, pm)
	pm.UpdateTestFactors(downFactors, upFactors)

	pm.OnTimeUpdate(now)
	b = pm.CheckPrice(context.TODO(), auctionStateMock, cp, true)
	require.False(t, b)

	auctionStateMock.EXPECT().StartPriceAuction(now, &end).Times(1)
	delta = num.Sum(maxDown1, maxDown2)
	cPrice = cPrice.Sub(currentPrice, delta.Div(delta, num.Sum(one, one)))
	cp5 := []*types.Trade{{Price: cPrice, Size: 1}}
	b = pm.CheckPrice(context.TODO(), auctionStateMock, cp5, true)
	require.False(t, b)

	// Reinstantiate price monitoring after auction to reset internal state
	pm, err = price.NewMonitor("asset", "market", riskModel, settings, statevar, logging.NewTestLogger())
	require.NoError(t, err)
	require.NotNil(t, pm)
	pm.UpdateTestFactors(downFactors, upFactors)

	pm.OnTimeUpdate(now)
	b = pm.CheckPrice(context.TODO(), auctionStateMock, cp, true)
	require.False(t, b)

	auctionStateMock.EXPECT().StartPriceAuction(now, &end).Times(1)
	cPrice = num.Sum(currentPrice, num.Zero().Sub(maxUp2, one)) // max price bound is now floored, so sub 1 to stay below second price bound
	cp6 := []*types.Trade{{Price: cPrice, Size: 1}}
	b = pm.CheckPrice(context.TODO(), auctionStateMock, cp6, true)
	require.False(t, b)

	// Reinstantiate price monitoring after auction to reset internal state
	pm, err = price.NewMonitor("asset", "market", riskModel, settings, statevar, logging.NewTestLogger())
	require.NoError(t, err)
	require.NotNil(t, pm)
	pm.UpdateTestFactors(downFactors, upFactors)

	pm.OnTimeUpdate(now)
	b = pm.CheckPrice(context.TODO(), auctionStateMock, cp, true)
	require.False(t, b)

	auctionStateMock.EXPECT().StartPriceAuction(now, &end).Times(1)
	cPrice = num.Sum(cPrice.Sub(currentPrice, maxDown2), one) // add 1 back, avoid breaching both down limits
	cp7 := []*types.Trade{{Price: cPrice, Size: 1}}
	b = pm.CheckPrice(context.TODO(), auctionStateMock, cp7, true)
	require.False(t, b)

	// Reinstantiate price monitoring after auction to reset internal state
	pm, err = price.NewMonitor("asset", "market", riskModel, settings, statevar, logging.NewTestLogger())
	require.NoError(t, err)
	require.NotNil(t, pm)
	pm.UpdateTestFactors(downFactors, upFactors)
	pm.OnTimeUpdate(now)
	b = pm.CheckPrice(context.TODO(), auctionStateMock, cp, true)
	require.False(t, b)

	end = types.AuctionDuration{Duration: t1.AuctionExtension + t2.AuctionExtension}
	auctionStateMock.EXPECT().StartPriceAuction(now, &end).Times(1)
	cPrice = num.Sum(currentPrice, maxUp2, maxUp2)
	cp8 := []*types.Trade{{Price: cPrice, Size: 1}}
	b = pm.CheckPrice(context.TODO(), auctionStateMock, cp8, true)
	require.False(t, b)

	// Reinstantiate price monitoring after auction to reset internal state
	pm, err = price.NewMonitor("asset", "market", riskModel, settings, statevar, logging.NewTestLogger())
	require.NoError(t, err)
	require.NotNil(t, pm)
	pm.UpdateTestFactors(downFactors, upFactors)
	pm.OnTimeUpdate(now)
	b = pm.CheckPrice(context.TODO(), auctionStateMock, cp, true)
	require.False(t, b)

	auctionStateMock.EXPECT().StartPriceAuction(now, &end).Times(1)
	delta = num.Sum(maxDown2, maxDown2)
	cPrice = cPrice.Sub(currentPrice, delta)
	cp9 := []*types.Trade{{Price: cPrice, Size: 1}}
	b = pm.CheckPrice(context.TODO(), auctionStateMock, cp9, true)
	require.False(t, b)
}

/*
func TestCheckBoundViolationsAcrossTimeWith1HorizonProbabilityPair(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	riskModel := mocks.NewMockRangeProvider(ctrl)
	auctionStateMock := mocks.NewMockAuctionState(ctrl)
	price1 := num.NewUint(123)
	initialTime := time.Date(1993, 2, 2, 6, 0, 0, 1, time.UTC)
	now := initialTime
	t1Time := int64(60)
	t1 := types.PriceMonitoringTrigger{Horizon: 600, Probability: 0.99, AuctionExtension: t1Time}
	boundUpdateFrequency := int64(120)
	settings := types.PriceMonitoringSettings{
		Parameters: &types.PriceMonitoringParameters{
			Triggers: []*types.PriceMonitoringTrigger{&t1},
		},
		UpdateFrequency: boundUpdateFrequency,
	}
	maxDown1, maxUp1 := num.NewUint(1), num.NewUint(2)
	p1Dec := num.DecimalFromUint(price1)
	h1 := horizonToYearFraction(t1.Horizon)
	prob1 := num.DecimalFromFloat(t1.Probability)
	min1 := p1Dec.Sub(num.DecimalFromUint(maxDown1))
	max1 := p1Dec.Add(num.DecimalFromUint(maxUp1))

	riskModel.EXPECT().PriceRange(p1Dec, h1, prob1).Return(min1, max1)
	auctionStateMock.EXPECT().IsFBA().Return(false).Times(25)
	auctionStateMock.EXPECT().InAuction().Return(false).Times(25)

	pm, err := price.NewMonitor(riskModel, settings)
	require.NoError(t, err)
	require.NotNil(t, pm)
	// for some reason this test casts to float and back...
	// it's a PITA with the num types, but let's keep it for now
	refPrice := p1Dec
	cPrice, _ := num.UintFromDecimal(refPrice)
	priceHistorySum := num.Sum(cPrice)
	n := 1
	err = pm.CheckPrice(ctx, auctionStateMock, cPrice, 1, now, true)
	require.NoError(t, err)

	cPrice = cPrice.Add(cPrice, maxUp1)
	priceHistorySum = num.Sum(priceHistorySum, cPrice)
	n++
	err = pm.CheckPrice(ctx, auctionStateMock, cPrice, 1, now, true)
	require.NoError(t, err)

	cPrice, _ = num.UintFromDecimal(refPrice) // this is silly, but the original test did this...
	cPrice = cPrice.Sub(cPrice, maxDown1)
	priceHistorySum = num.Sum(priceHistorySum, cPrice)
	n++
	err = pm.CheckPrice(ctx, auctionStateMock, cPrice, 1, now, true)
	require.NoError(t, err)

	end := types.AuctionDuration{Duration: t1.AuctionExtension}
	pm.SetMinDuration(time.Duration(t1.AuctionExtension) * time.Second)
	auctionStateMock.EXPECT().StartPriceAuction(now, &end).Times(1)
	// just as an example of the mess with refPrice as float:
	// err = pm.CheckPrice(ctx, auctionStateMock, uint64(referencePrice)+2*maxMoveUp1, 1, now, true)
	cPrice, _ = num.UintFromDecimal(refPrice)
	cPrice = num.Sum(cPrice, maxUp1, maxUp1)
	err = pm.CheckPrice(ctx, auctionStateMock, cPrice, 1, now, true)
	require.NoError(t, err)

	//Still before update (no price change)
	updateTime := now.Add(time.Duration(boundUpdateFrequency) * time.Second)
	now = updateTime.Add(-time.Second)
	avgPrice1 := priceHistorySum.ToDecimal().Div(num.DecimalFromFloat(float64(n)))
	refPrice = avgPrice1
	cPrice, _ = num.UintFromDecimal(refPrice)
	priceHistorySum = priceHistorySum.Set(cPrice)
	n = 1
	err = pm.CheckPrice(ctx, auctionStateMock, cPrice, 1, now, true)
	require.NoError(t, err)

	//Execting same behaviour as above (per reference price)
	cPrice, _ = num.UintFromDecimal(refPrice.Floor())
	cPrice.Add(cPrice, maxUp1)
	priceHistorySum.Add(priceHistorySum, cPrice)
	n++
	err = pm.CheckPrice(ctx, auctionStateMock, cPrice, 1, now, true)
	require.NoError(t, err)

	cPrice, _ = num.UintFromDecimal(refPrice.Ceil())
	cPrice.Sub(cPrice, maxDown1)
	priceHistorySum.Add(priceHistorySum, cPrice)
	n++
	err = pm.CheckPrice(ctx, auctionStateMock, cPrice, 1, now, true)
	require.NoError(t, err)

	end = types.AuctionDuration{Duration: t1.AuctionExtension}
	cPrice, _ = num.UintFromDecimal(refPrice)
	cPrice.Sub(cPrice, num.Sum(maxDown1, maxDown1))
	err = pm.CheckPrice(ctx, auctionStateMock, cPrice, 1, now, true)
	require.NoError(t, err)

	//Right at update time (after the auction has concluded)
	now = initialTime.Add(time.Duration(2*boundUpdateFrequency) * time.Second)
	// multiply by 4
	maxDown2 := num.Sum(maxDown1, maxDown1, maxDown1, maxDown1)
	maxUp2 := num.Sum(maxUp1, maxUp1, maxUp1, maxUp1)
	avgPrice2 := priceHistorySum.ToDecimal().Div(num.DecimalFromFloat(float64(n)))
	refPrice = avgPrice2
	cPrice, _ = num.UintFromDecimal(refPrice)
	priceHistorySum = priceHistorySum.Set(cPrice)
	n = 1
	err = pm.CheckPrice(ctx, auctionStateMock, cPrice, 1, now, true)
	require.NoError(t, err)

	cPrice, _ = num.UintFromDecimal(refPrice.Floor())
	cPrice.Add(cPrice, maxUp2)
	priceHistorySum.Add(priceHistorySum, cPrice)
	n++
	err = pm.CheckPrice(ctx, auctionStateMock, cPrice, 1, now, true)
	require.NoError(t, err)

	cPrice, _ = num.UintFromDecimal(refPrice.Ceil())
	cPrice.Sub(cPrice, maxDown2)
	volume := uint64(2)
	priceHistorySum = num.Sum(priceHistorySum, cPrice, cPrice) // multiplied by volume of 2
	n += int(volume)
	err = pm.CheckPrice(ctx, auctionStateMock, cPrice, volume, now, true)
	require.NoError(t, err)

	end = types.AuctionDuration{Duration: t1.AuctionExtension}
	cPrice, _ = num.UintFromDecimal(refPrice)
	cPrice = num.Sum(cPrice, maxUp2, maxUp2)
	err = pm.CheckPrice(ctx, auctionStateMock, cPrice, 1, now, true)
	require.NoError(t, err)

	// Right before update time (horizon away from averagePrice3)
	updateTime = now.Add(time.Duration(t1.Horizon) * time.Second)
	now = updateTime.Add(-time.Second)
	averagePrice3 := float64(priceHistorySum) / float64(n)
	referencePrice = averagePrice2
	maxMoveDown3 := 6 * maxMoveDown1
	maxMoveUp3 := 6 * maxMoveUp1

	validPriceToCheck = uint64(referencePrice)
	priceHistorySum = validPriceToCheck
	n = 1
	err = pm.CheckPrice(context.TODO(), auctionStateMock, validPriceToCheck, 1, now, true)
	require.NoError(t, err)

	validPriceToCheck = uint64(math.Floor(referencePrice)) + maxMoveUp3
	priceHistorySum += validPriceToCheck
	n++
	err = pm.CheckPrice(context.TODO(), auctionStateMock, validPriceToCheck, 1, now, true)
	require.NoError(t, err)

	validPriceToCheck = uint64(math.Ceil(referencePrice)) - maxMoveDown3
	priceHistorySum += validPriceToCheck
	n++
	err = pm.CheckPrice(context.TODO(), auctionStateMock, validPriceToCheck, 1, now, true)
	require.NoError(t, err)

	end = types.AuctionDuration{Duration: t1.AuctionExtension}
	err = pm.CheckPrice(context.TODO(), auctionStateMock, uint64(math.Floor(referencePrice))+2*maxMoveUp3, 1, now, true)
	require.NoError(t, err)

	err = pm.CheckPrice(context.TODO(), auctionStateMock, uint64(math.Ceil(referencePrice))-2*maxMoveDown3, 1, now, true)
	require.NoError(t, err)

	// Right at update time (horizon away from price3Average)
	now = updateTime
	referencePrice = averagePrice3

	validPriceToCheck = uint64(referencePrice)
	priceHistorySum = validPriceToCheck
	n = 1
	err = pm.CheckPrice(context.TODO(), auctionStateMock, validPriceToCheck, 1, now, true)
	require.NoError(t, err)

	validPriceToCheck = uint64(math.Floor(referencePrice)) + maxMoveUp3
	priceHistorySum += validPriceToCheck
	n++
	err = pm.CheckPrice(context.TODO(), auctionStateMock, validPriceToCheck, 1, now, true)
	require.NoError(t, err)

	validPriceToCheck = uint64(math.Ceil(referencePrice)) - maxMoveDown3
	priceHistorySum += validPriceToCheck
	n++
	err = pm.CheckPrice(context.TODO(), auctionStateMock, validPriceToCheck, 1, now, true)
	require.NoError(t, err)

	end = types.AuctionDuration{Duration: t1.AuctionExtension}
	err = pm.CheckPrice(context.TODO(), auctionStateMock, uint64(math.Ceil(referencePrice))-2*maxMoveDown3, 1, now, true)
	require.NoError(t, err)

	//Reset price, the resetting value should become the new reference
	now = now.Add(time.Hour)
	var resetPrice uint64 = 20
	var maxMoveDown4 uint64 = 5
	var maxMoveUp4 uint64 = 120
	referencePrice = float64(resetPrice)

	//Assume in auction now
	validPriceToCheck = resetPrice
	err = pm.CheckPrice(context.TODO(), auctionStateMock, validPriceToCheck, 1, now, true)
	require.NoError(t, err)

	validPriceToCheck = resetPrice + maxMoveUp4
	err = pm.CheckPrice(context.TODO(), auctionStateMock, validPriceToCheck, 1, now, true)
	require.NoError(t, err)

	validPriceToCheck = resetPrice - maxMoveDown4
	err = pm.CheckPrice(context.TODO(), auctionStateMock, validPriceToCheck, 1, now, true)
	require.NoError(t, err)

	end = types.AuctionDuration{Duration: t1.AuctionExtension}
	err = pm.CheckPrice(context.TODO(), auctionStateMock, uint64(referencePrice)+2*maxMoveUp4, 1, now, true)
	require.NoError(t, err)
}
/**/

func TestAuctionStartedAndEndendBy1Trigger(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	riskModel := mocks.NewMockRangeProvider(ctrl)
	auctionStateMock := mocks.NewMockAuctionState(ctrl)
	price1 := num.NewUint(123)
	cp := []*types.Trade{{Price: price1, Size: 1}}
	ctx := context.Background()
	now := time.Date(1993, 2, 2, 6, 0, 0, 1, time.UTC)
	t1 := proto.PriceMonitoringTrigger{Horizon: 600, Probability: "0.95", AuctionExtension: 60}
	t2 := proto.PriceMonitoringTrigger{Horizon: 600, Probability: "0.99", AuctionExtension: 120}
	var boundUpdateFrequency int64 = 120
	pSet := &proto.PriceMonitoringSettings{
		Parameters: &proto.PriceMonitoringParameters{
			Triggers: []*proto.PriceMonitoringTrigger{&t1, &t2},
		},
		UpdateFrequency: boundUpdateFrequency,
	}
	settings := types.PriceMonitoringSettingsFromProto(pSet)

	maxDown1, maxUp1 := num.NewUint(1), num.NewUint(2)
	maxDown2 := num.Sum(maxUp1, maxUp1)   // yes, maxUp -> maxUp == maxDown*2, down2 == down1*4
	maxUp2 := num.Sum(maxDown2, maxDown2) // double
	decPrice := price1.ToDecimal()
	p1Min1 := decPrice.Sub(num.DecimalFromUint(maxDown1))
	p1Min2 := decPrice.Sub(num.DecimalFromUint(maxDown2))
	p1Max1 := decPrice.Add(num.DecimalFromUint(maxUp1))
	p1Max2 := decPrice.Add(num.DecimalFromUint(maxUp2))
	downFactorsP1 := []num.Decimal{p1Min1.Div(decPrice), p1Min2.Div(decPrice)}
	upFactorsP1 := []num.Decimal{p1Max1.Div(decPrice), p1Max2.Div(decPrice)}

	auctionStateMock.EXPECT().IsFBA().Return(false).Times(2)
	auctionStateMock.EXPECT().InAuction().Return(false).Times(2)
	auctionStateMock.EXPECT().IsPriceAuction().Return(true).AnyTimes()
	statevar := mocks.NewMockStateVarEngine(ctrl)
	statevar.EXPECT().RegisterStateVariable(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any())

	pm, err := price.NewMonitor("asset", "market", riskModel, settings, statevar, logging.NewTestLogger())
	require.NoError(t, err)
	require.NotNil(t, pm)
	pm.UpdateTestFactors(downFactorsP1, upFactorsP1)
	pm.OnTimeUpdate(now)
	b := pm.CheckPrice(ctx, auctionStateMock, cp, true)
	require.False(t, b)

	end := types.AuctionDuration{Duration: t1.AuctionExtension}
	pm.SetMinDuration(time.Duration(t1.AuctionExtension) * time.Second)
	auctionStateMock.EXPECT().StartPriceAuction(now, &end).Times(1)

	delta := num.Sum().Sub(maxUp2, maxUp1)
	cPrice := num.Sum(price1, delta)
	cp1 := []*types.Trade{{Price: cPrice, Size: 1}}
	pm.OnTimeUpdate(now)
	b = pm.CheckPrice(context.TODO(), auctionStateMock, cp1, true) // t1 violated only
	require.False(t, b)

	initialAuctionEnd := now.Add(time.Duration(t1.AuctionExtension) * time.Second)

	auctionStateMock.EXPECT().IsFBA().Return(false).Times(1)
	auctionStateMock.EXPECT().InAuction().Return(true).Times(1)
	auctionStateMock.EXPECT().IsOpeningAuction().Return(false).Times(1)
	// auctionStateMock.EXPECT().IsPriceAuction().Return(true).Times(1)
	auctionStateMock.EXPECT().ExpiresAt().Return(&initialAuctionEnd).Times(1)
	auctionStateMock.EXPECT().SetReadyToLeave().Times(1)

	afterInitialAuction := initialAuctionEnd.Add(time.Nanosecond)
	pm.OnTimeUpdate(afterInitialAuction)
	b = pm.CheckPrice(ctx, auctionStateMock, cp1, true) // price should be accepted now
	require.False(t, b)
}

func TestAuctionStartedAndEndendBy2Triggers(t *testing.T) {
	ctrl := gomock.NewController(t)
	riskModel := mocks.NewMockRangeProvider(ctrl)
	auctionStateMock := mocks.NewMockAuctionState(ctrl)
	price1 := num.NewUint(123)
	cp1 := []*types.Trade{{Price: price1, Size: 1}}
	ctx := context.Background()
	now := time.Date(1993, 2, 2, 6, 0, 0, 1, time.UTC)
	t1 := proto.PriceMonitoringTrigger{Horizon: 600, Probability: "0.95", AuctionExtension: 60}
	t2 := proto.PriceMonitoringTrigger{Horizon: 600, Probability: "0.99", AuctionExtension: 120}
	var boundUpdateFrequency int64 = 120
	pSet := &proto.PriceMonitoringSettings{
		Parameters: &proto.PriceMonitoringParameters{
			Triggers: []*proto.PriceMonitoringTrigger{&t1, &t2},
		},
		UpdateFrequency: boundUpdateFrequency,
	}
	settings := types.PriceMonitoringSettingsFromProto(pSet)

	_, _, _, _, maxUp1 := getPriceBounds(price1, 1, 2)
	_, _, _, _, maxUp2 := getPriceBounds(price1, 1*4, 2*4)

	auctionStateMock.EXPECT().IsFBA().Return(false).Times(2)
	auctionStateMock.EXPECT().InAuction().Return(false).Times(2)
	statevar := mocks.NewMockStateVarEngine(ctrl)
	statevar.EXPECT().RegisterStateVariable(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any())

	pm, err := price.NewMonitor("asset", "market", riskModel, settings, statevar, logging.NewTestLogger())
	require.NoError(t, err)
	require.NotNil(t, pm)

	pm.OnTimeUpdate(now)
	b := pm.CheckPrice(ctx, auctionStateMock, cp1, true)
	require.False(t, b)

	end := types.AuctionDuration{Duration: t1.AuctionExtension + t2.AuctionExtension}
	pm.SetMinDuration(time.Duration(end.Duration) * time.Second)
	// auctionStateMock.EXPECT().StartPriceAuction(now, &end).Times(1)

	cPrice := num.Sum(price1, maxUp2, maxUp1)
	cp2 := []*types.Trade{{Price: cPrice, Size: 1}}
	// decPrice, pMin1, pMax1, _, _ := getPriceBounds(cPrice, 1, 2)
	// _, pMin2, pMax2, _, _ = getPriceBounds(cPrice, 1*4, 2*4)

	b = pm.CheckPrice(ctx, auctionStateMock, cp2, true) // t1 violated only
	require.False(t, b)

	initialAuctionEnd := now.Add(time.Duration(t1.AuctionExtension+t2.AuctionExtension) * time.Second)

	auctionStateMock.EXPECT().IsFBA().Return(false).Times(1)
	auctionStateMock.EXPECT().InAuction().Return(true).Times(1)
	auctionStateMock.EXPECT().IsOpeningAuction().Return(false).Times(1)
	auctionStateMock.EXPECT().IsPriceAuction().Return(true).Times(1)
	auctionStateMock.EXPECT().ExpiresAt().Return(&initialAuctionEnd).Times(1)
	auctionStateMock.EXPECT().SetReadyToLeave().Times(1)

	afterInitialAuction := initialAuctionEnd.Add(time.Nanosecond)
	pm.OnTimeUpdate(afterInitialAuction)
	b = pm.CheckPrice(ctx, auctionStateMock, cp2, true) // price should be accepted now
	require.False(t, b)
}

func TestAuctionStartedAndEndendBy1TriggerAndExtendedBy2nd(t *testing.T) {
	// Also verifies that GetCurrentBounds() works as expected
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	riskModel := mocks.NewMockRangeProvider(ctrl)
	auctionStateMock := mocks.NewMockAuctionState(ctrl)
	price1 := num.NewUint(123)
	cp1 := []*types.Trade{{Price: price1, Size: 1}}
	now := time.Date(1993, 2, 2, 6, 0, 0, 1, time.UTC)
	t1 := proto.PriceMonitoringTrigger{Horizon: 600, Probability: "0.95", AuctionExtension: 60}
	t2 := proto.PriceMonitoringTrigger{Horizon: 600, Probability: "0.99", AuctionExtension: 120}
	var boundUpdateFrequency int64 = 120
	pSet := &proto.PriceMonitoringSettings{
		Parameters: &proto.PriceMonitoringParameters{
			Triggers: []*proto.PriceMonitoringTrigger{&t1, &t2},
		},
		UpdateFrequency: boundUpdateFrequency,
	}
	settings := types.PriceMonitoringSettingsFromProto(pSet)
	ctx := context.Background()
	decPrice, pMin1, pMax1, _, maxUp1 := getPriceBounds(price1, 1, 2)
	_, pMin2, pMax2, _, maxUp2 := getPriceBounds(price1, 1*4, 2*4)

	one := num.NewUint(1)
	t1lb1, _ := num.UintFromDecimal(pMin1)
	t1lb1.AddSum(one) // account for value being ceil'ed
	t1ub1, _ := num.UintFromDecimal(pMax1)
	t1ub1.Sub(t1ub1, one) // floor
	t2lb1, _ := num.UintFromDecimal(pMin2)
	t2lb1.AddSum(one) // again: ceil
	t2ub1, _ := num.UintFromDecimal(pMax2)

	auctionStateMock.EXPECT().IsFBA().Return(false).Times(2)
	auctionStateMock.EXPECT().InAuction().Return(false).Times(2)
	statevar := mocks.NewMockStateVarEngine(ctrl)
	statevar.EXPECT().RegisterStateVariable(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any())

	pm, err := price.NewMonitor("asset", "market", riskModel, settings, statevar, logging.NewTestLogger())
	downFactors := []num.Decimal{pMin1.Div(decPrice), pMin2.Div(decPrice)}
	upFactors := []num.Decimal{pMax1.Div(decPrice), pMax2.Div(decPrice)}
	pm.UpdateTestFactors(downFactors, upFactors)

	require.NoError(t, err)
	require.NotNil(t, pm)

	pm.OnTimeUpdate(now)
	b := pm.CheckPrice(ctx, auctionStateMock, cp1, true)
	require.False(t, b)

	bounds := pm.GetCurrentBounds()
	require.Len(t, bounds, 2)
	require.Equal(t, *bounds[0].Trigger.IntoProto(), t1)
	require.True(t, bounds[0].MinValidPrice.EQ(t1lb1))
	require.True(t, bounds[0].MaxValidPrice.EQ(t1ub1))
	require.Equal(t, bounds[0].ReferencePrice, decPrice)
	require.Equal(t, *bounds[1].Trigger.IntoProto(), t2)
	require.True(t, bounds[1].MinValidPrice.EQ(t2lb1))
	require.True(t, bounds[1].MaxValidPrice.EQ(t2ub1))
	require.Equal(t, bounds[1].ReferencePrice, decPrice)

	end := types.AuctionDuration{Duration: t1.AuctionExtension}
	pm.SetMinDuration(time.Duration(end.Duration) * time.Second)
	auctionStateMock.EXPECT().StartPriceAuction(now, &end).Times(1)

	cPrice := num.Sum(price1, maxUp2)
	cPrice.Sub(cPrice, maxUp1)
	cp2 := []*types.Trade{{Price: cPrice, Size: 1}}
	b = pm.CheckPrice(ctx, auctionStateMock, cp2, true) // t1 violated only
	require.False(t, b)

	initialAuctionEnd := now.Add(time.Duration(t1.AuctionExtension) * time.Second)

	auctionStateMock.EXPECT().IsFBA().Return(false).Times(1)
	auctionStateMock.EXPECT().InAuction().Return(true).Times(1)
	auctionStateMock.EXPECT().IsOpeningAuction().Return(false).Times(1)
	auctionStateMock.EXPECT().IsPriceAuction().Return(true).AnyTimes()
	auctionStateMock.EXPECT().ExpiresAt().Return(&initialAuctionEnd).Times(1)

	bounds = pm.GetCurrentBounds()
	require.Len(t, bounds, 1)
	require.Equal(t, *bounds[0].Trigger.IntoProto(), t2)
	require.True(t, bounds[0].MinValidPrice.EQ(t2lb1))
	require.True(t, bounds[0].MaxValidPrice.EQ(t2ub1))
	require.Equal(t, bounds[0].ReferencePrice, decPrice)

	afterInitialAuction := initialAuctionEnd.Add(time.Nanosecond)
	now = afterInitialAuction

	cPrice = num.Sum(price1, maxUp2, maxUp1)
	end2 := types.AuctionDuration{Duration: t2.AuctionExtension}
	auctionStateMock.EXPECT().ExtendAuctionPrice(end2).Times(1)
	pm.OnTimeUpdate(afterInitialAuction)
	cp3 := []*types.Trade{{Price: cPrice, Size: 1}}
	b = pm.CheckPrice(ctx, auctionStateMock, cp3, true) // price should violated 2nd trigger and result in auction extension
	require.False(t, b)

	bounds = pm.GetCurrentBounds()
	require.Len(t, bounds, 0)

	extendedAuctionEnd := now.Add(time.Duration(t1.AuctionExtension+t2.AuctionExtension) * time.Second)

	// get new bounds
	_, pMin1, pMax1, _, _ = getPriceBounds(cPrice, 1, 2)
	_, pMin2, pMax2, _, _ = getPriceBounds(cPrice, 1*4, 2*4)

	t1lb1, _ = num.UintFromDecimal(pMin1)
	t1lb1.AddSum(one) // again ceil
	t1ub1, _ = num.UintFromDecimal(pMax1)
	t1ub1.Sub(t1ub1, one) // floor...
	t2lb1, _ = num.UintFromDecimal(pMin2)
	t2lb1.AddSum(one) // ceil...
	t2ub1, _ = num.UintFromDecimal(pMax2)
	t2ub1.Sub(t2ub1, one) // floor...

	auctionStateMock.EXPECT().IsFBA().Return(false).Times(1)
	auctionStateMock.EXPECT().InAuction().Return(true).Times(1)
	auctionStateMock.EXPECT().IsOpeningAuction().Return(false).Times(1)
	auctionStateMock.EXPECT().SetReadyToLeave().Times(1)

	afterExtendedAuction := extendedAuctionEnd.Add(time.Nanosecond)
	pm.OnTimeUpdate(afterExtendedAuction)
	b = pm.CheckPrice(ctx, auctionStateMock, cp3, true) // price should be accepted now
	require.False(t, b)
}

func TestMarketInOpeningAuction(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	riskModel := mocks.NewMockRangeProvider(ctrl)
	auctionStateMock := mocks.NewMockAuctionState(ctrl)
	currentPrice := num.NewUint(123)
	cp := []*types.Trade{{Price: currentPrice, Size: 1}}
	t1 := proto.PriceMonitoringTrigger{Horizon: 7200, Probability: "0.95", AuctionExtension: 300}
	now := time.Date(1993, 2, 2, 6, 0, 0, 1, time.UTC)
	pSet := &proto.PriceMonitoringSettings{
		Parameters: &proto.PriceMonitoringParameters{
			Triggers: []*proto.PriceMonitoringTrigger{&t1},
		},
		UpdateFrequency: 1,
	}
	settings := types.PriceMonitoringSettingsFromProto(pSet)

	ctx := context.Background()

	auctionStateMock.EXPECT().IsFBA().Return(false).Times(1)
	auctionStateMock.EXPECT().InAuction().Return(true).Times(1)
	auctionStateMock.EXPECT().IsOpeningAuction().Return(true).Times(1)
	statevar := mocks.NewMockStateVarEngine(ctrl)
	statevar.EXPECT().RegisterStateVariable(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any())

	pm, err := price.NewMonitor("asset", "market", riskModel, settings, statevar, logging.NewTestLogger())
	require.NoError(t, err)
	require.NotNil(t, pm)

	pm.OnTimeUpdate(now)
	b := pm.CheckPrice(ctx, auctionStateMock, cp, true)
	require.False(t, b)
}

func TestMarketInGenericAuction(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	riskModel := mocks.NewMockRangeProvider(ctrl)
	auctionStateMock := mocks.NewMockAuctionState(ctrl)
	currentPrice := num.NewUint(123)
	cp1 := []*types.Trade{{Price: currentPrice, Size: 1}}
	t1 := proto.PriceMonitoringTrigger{Horizon: 7200, Probability: "0.95", AuctionExtension: 300}
	now := time.Date(1993, 2, 2, 6, 0, 0, 1, time.UTC)

	pSet := &proto.PriceMonitoringSettings{
		Parameters: &proto.PriceMonitoringParameters{
			Triggers: []*proto.PriceMonitoringTrigger{&t1},
		},
		UpdateFrequency: 1,
	}
	settings := types.PriceMonitoringSettingsFromProto(pSet)

	_, _, _, maxDown, maxUp := getPriceBounds(currentPrice, 5, 10)
	one := num.NewUint(1)
	ctx := context.Background()

	// price monitoring starts with auction, not initialised, so there's no fixed price level it'll check
	auctionStateMock.EXPECT().IsFBA().Return(false).Times(5)
	auctionStateMock.EXPECT().InAuction().Return(true).Times(5)
	auctionStateMock.EXPECT().IsOpeningAuction().Return(false).Times(5)
	auctionStateMock.EXPECT().IsPriceAuction().Return(false).AnyTimes()
	auctionStateMock.EXPECT().IsPriceExtension().Return(false).AnyTimes()
	auctionStateMock.EXPECT().CanLeave().Return(false).AnyTimes()
	statevar := mocks.NewMockStateVarEngine(ctrl)
	statevar.EXPECT().RegisterStateVariable(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any())

	pm, err := price.NewMonitor("asset", "market", riskModel, settings, statevar, logging.NewTestLogger())
	require.NoError(t, err)
	require.NotNil(t, pm)

	pm.OnTimeUpdate(now)
	b := pm.CheckPrice(ctx, auctionStateMock, cp1, true)
	require.False(t, b)
	cPrice := num.Sum(currentPrice, maxUp)
	cPrice.Sub(cPrice, one)
	cp2 := []*types.Trade{{Price: cPrice, Size: 1}}
	b = pm.CheckPrice(ctx, auctionStateMock, cp2, true)
	require.False(t, b)

	cPrice.Sub(num.Sum(currentPrice, one), maxDown)
	cp3 := []*types.Trade{{Price: cPrice, Size: 1}}
	b = pm.CheckPrice(ctx, auctionStateMock, cp3, true)
	require.False(t, b)

	extension := types.AuctionDuration{Duration: t1.AuctionExtension}
	auctionStateMock.EXPECT().ExtendAuctionPrice(extension).MinTimes(1).MaxTimes(1)

	cPrice = num.Sum(currentPrice, maxUp, maxUp)
	cp4 := []*types.Trade{{Price: cPrice, Size: 1}}
	b = pm.CheckPrice(ctx, auctionStateMock, cp4, true)
	require.False(t, b)

	cPrice = num.Sum(maxDown, maxDown)
	cPrice.Sub(currentPrice, cPrice)
	cp5 := []*types.Trade{{Price: cPrice, Size: 1}}
	b = pm.CheckPrice(ctx, auctionStateMock, cp5, true)
	require.False(t, b)
}

func TestGetValidPriceRange_NoTriggers(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	riskModel := mocks.NewMockRangeProvider(ctrl)
	auctionStateMock := mocks.NewMockAuctionState(ctrl)
	currentPrice := num.NewUint(123)
	cp := []*types.Trade{{Price: currentPrice, Size: 1}}
	now := time.Date(1993, 2, 2, 6, 0, 0, 1, time.UTC)
	ctx := context.Background()

	settings := &types.PriceMonitoringSettings{
		Parameters: &types.PriceMonitoringParameters{
			Triggers: []*types.PriceMonitoringTrigger{},
		},
		UpdateFrequency: 1,
	}

	auctionStateMock.EXPECT().IsFBA().Return(false).Times(1)
	auctionStateMock.EXPECT().InAuction().Return(false).Times(1)
	statevar := mocks.NewMockStateVarEngine(ctrl)
	statevar.EXPECT().RegisterStateVariable(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any())

	pm, err := price.NewMonitor("asset", "market", riskModel, settings, statevar, logging.NewTestLogger())
	require.NoError(t, err)
	require.NotNil(t, pm)

	expMax := num.MaxUint()
	min, max := pm.GetValidPriceRange()
	require.True(t, min.Representation().IsZero())
	require.Equal(t, expMax.String(), max.Representation().String())

	pm.OnTimeUpdate(now)
	b := pm.CheckPrice(ctx, auctionStateMock, cp, true)
	require.False(t, b)

	min, max = pm.GetValidPriceRange()
	require.True(t, min.Representation().IsZero())
	require.Equal(t, expMax.String(), max.Representation().String())
}

func TestGetValidPriceRange_2triggers(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	riskModel := mocks.NewMockRangeProvider(ctrl)
	auctionStateMock := mocks.NewMockAuctionState(ctrl)
	currentPrice := num.NewUint(123)
	cp := []*types.Trade{{Price: currentPrice, Size: 1}}
	now := time.Date(1993, 2, 2, 6, 0, 0, 1, time.UTC)
	var t1Time int64 = 60
	var t2Time int64 = 300
	t1 := proto.PriceMonitoringTrigger{Horizon: 3600, Probability: "0.99", AuctionExtension: t1Time}
	t2 := proto.PriceMonitoringTrigger{Horizon: 7200, Probability: "0.95", AuctionExtension: t2Time}
	pSet := &proto.PriceMonitoringSettings{
		Parameters: &proto.PriceMonitoringParameters{
			Triggers: []*proto.PriceMonitoringTrigger{&t1, &t2},
		},
		UpdateFrequency: 600,
	}
	settings := types.PriceMonitoringSettingsFromProto(pSet)

	ctx := context.Background()
	_, pMin1, pMax1, maxDown1, maxUp1 := getPriceBounds(currentPrice, 1, 2)
	_, pMin2, pMax2, _, _ := getPriceBounds(currentPrice, 3, 4)
	one := num.NewUint(1)
	currentPriceD := currentPrice.ToDecimal()
	auctionStateMock.EXPECT().IsFBA().Return(false).Times(12)
	auctionStateMock.EXPECT().InAuction().Return(false).Times(12)
	statevar := mocks.NewMockStateVarEngine(ctrl)
	statevar.EXPECT().RegisterStateVariable(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any())

	pm, err := price.NewMonitor("asset", "market", riskModel, settings, statevar, logging.NewTestLogger())
	require.NoError(t, err)
	require.NotNil(t, pm)
	downFactors := []num.Decimal{pMin1.Div(currentPriceD), pMin2.Div(currentPriceD)}
	upFactors := []num.Decimal{pMax1.Div(currentPriceD), pMax2.Div(currentPriceD)}

	pm.UpdateTestFactors(downFactors, upFactors)

	pm.OnTimeUpdate(now)
	b := pm.CheckPrice(ctx, auctionStateMock, cp, true)
	require.False(t, b)

	_, _ = pm.GetValidPriceRange()
	now = now.Add(time.Second)
	cPrice := num.Sum(currentPrice, maxUp1)
	cPrice.Sub(cPrice, one)
	cp2 := []*types.Trade{{Price: cPrice, Size: 1}}
	pm.OnTimeUpdate(now)
	b = pm.CheckPrice(ctx, auctionStateMock, cp2, true)
	require.False(t, b)

	_, _ = pm.GetValidPriceRange()
	now = now.Add(time.Minute)
	cPrice = num.Sum(currentPrice, one)
	cPrice.Sub(cPrice, maxDown1)
	cp3 := []*types.Trade{{Price: cPrice, Size: 1}}
	pm.OnTimeUpdate(now)
	b = pm.CheckPrice(ctx, auctionStateMock, cp3, true)
	require.False(t, b)

	_, _ = pm.GetValidPriceRange()
	now = now.Add(time.Hour)
	cPrice = num.Sum(currentPrice, maxUp1)
	cPrice.Sub(cPrice, one)
	cp4 := []*types.Trade{{Price: cPrice, Size: 1}}
	pm.OnTimeUpdate(now)
	b = pm.CheckPrice(ctx, auctionStateMock, cp4, true)
	require.False(t, b)

	_, _ = pm.GetValidPriceRange()
	now = now.Add(time.Minute)
	cPrice.Sub(currentPrice, maxDown1)
	cPrice.AddSum(one)
	cp5 := []*types.Trade{{Price: cPrice, Size: 1}}
	pm.OnTimeUpdate(now)
	b = pm.CheckPrice(ctx, auctionStateMock, cp5, true)
	require.False(t, b)

	min, max := pm.GetValidPriceRange()
	cp6 := []*types.Trade{{Price: min.Representation(), Size: 1}}
	b = pm.CheckPrice(ctx, auctionStateMock, cp6, true)
	require.False(t, b)

	cp7 := []*types.Trade{{Price: max.Representation(), Size: 1}}
	b = pm.CheckPrice(ctx, auctionStateMock, cp7, true)
	require.False(t, b)

	// Should trigger an auction
	auctionStateMock.EXPECT().StartPriceAuction(now, gomock.Any()).Times(1)

	cPrice.Sub(min.Representation(), one)
	cp8 := []*types.Trade{{Price: cPrice, Size: 1}}
	b = pm.CheckPrice(ctx, auctionStateMock, cp8, true)
	require.False(t, b)

	now = now.Add(time.Second)
	pm.OnTimeUpdate(now)
	b = pm.CheckPrice(ctx, auctionStateMock, cp, true)
	require.False(t, b)

	min, max = pm.GetValidPriceRange()

	cp9 := []*types.Trade{{Price: min.Representation(), Size: 1}}
	b = pm.CheckPrice(ctx, auctionStateMock, cp9, true)
	require.False(t, b)

	cp10 := []*types.Trade{{Price: max.Representation(), Size: 1}}
	b = pm.CheckPrice(ctx, auctionStateMock, cp10, true)
	require.False(t, b)

	// Should trigger an auction
	auctionStateMock.EXPECT().StartPriceAuction(now, gomock.Any()).Times(1)
	cPrice.Add(max.Representation(), one)
	cp11 := []*types.Trade{{Price: cPrice, Size: 1}}
	b = pm.CheckPrice(ctx, auctionStateMock, cp11, true)
	require.False(t, b)
}

/*
func testPricesValidAfterAuctionEnds(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	riskModel := mocks.NewMockRangeProvider(ctrl)
	auctionStateMock := mocks.NewMockAuctionState(ctrl)
	var price1 uint64 = 123
	now := time.Date(1993, 2, 2, 6, 0, 0, 1, time.UTC)
	t1 := types.PriceMonitoringTrigger{Horizon: 600, Probability: 0.95, AuctionExtension: 60}
	var boundUpdateFrequency int64 = 120
	settings := types.PriceMonitoringSettings{
		Parameters: &types.PriceMonitoringParameters{
			Triggers: []*types.PriceMonitoringTrigger{&t1},
		},
		UpdateFrequency: boundUpdateFrequency,
	}
	var maxMoveDownt1 uint64 = 1
	var maxMoveUpt1 uint64 = 2
	var t2Multiplier uint64 = 4
	require.True(t, (t2Multiplier-1) > 1)

	riskModel.EXPECT().PriceRange(float64(price1), horizonToYearFraction(t1.Horizon), t1.Probability).Return(float64(price1-maxMoveDownt1), float64(price1+maxMoveUpt1)).Times(1)
	auctionStateMock.EXPECT().IsFBA().Return(false).Times(2)
	auctionStateMock.EXPECT().InAuction().Return(false).Times(2)

	pm, err := price.NewMonitor(riskModel, settings)
	require.NoError(t, err)
	require.NotNil(t, pm)
	var priceHistorySum uint64 = 0
	n := 0
	referencePrice := float64(price1)
	priceToCheck := uint64(referencePrice)
	priceHistorySum += priceToCheck
	n++
	err = pm.CheckPrice(context.TODO(), auctionStateMock, price1, 1, now, true)
	require.NoError(t, err)

	end := types.AuctionDuration{Duration: t1.AuctionExtension}
	pm.SetMinDuration(time.Duration(end.Duration) * time.Second)
	auctionStateMock.EXPECT().StartPriceAuction(now, &end).Times(1)
	t1ViolatingPrice := uint64(referencePrice) + (t2Multiplier-1)*maxMoveUpt1
	err = pm.CheckPrice(context.TODO(), auctionStateMock, t1ViolatingPrice, 1, now, true) //t1 violated only
	require.NoError(t, err)

	initialAuctionEnd := now.Add(time.Duration(t1.AuctionExtension) * time.Second)

	auctionStateMock.EXPECT().IsFBA().Return(false).Times(1)
	auctionStateMock.EXPECT().InAuction().Return(true).Times(1)
	auctionStateMock.EXPECT().IsOpeningAuction().Return(false).Times(1)
	auctionStateMock.EXPECT().IsPriceAuction().Return(true).Times(1)
	auctionStateMock.EXPECT().ExpiresAt().Return(&initialAuctionEnd).Times(1)
	auctionStateMock.EXPECT().SetReadyToLeave().Times(1)
	riskModel.EXPECT().PriceRange(float64(t1ViolatingPrice), horizonToYearFraction(t1.Horizon), t1.Probability).Return(float64(t1ViolatingPrice-maxMoveDownt1), float64(t1ViolatingPrice+maxMoveUpt1)).Times(1)

	afterInitialAuction := initialAuctionEnd.Add(time.Nanosecond)
	err = pm.CheckPrice(context.TODO(), auctionStateMock, t1ViolatingPrice, 1, afterInitialAuction, true) //price should be accepted now
	require.NoError(t, err)

	auctionStateMock.EXPECT().IsFBA().Return(false).Times(1)
	auctionStateMock.EXPECT().InAuction().Return(false).Times(1)

	err = pm.CheckPrice(context.Background(), auctionStateMock, t1ViolatingPrice, 1, afterInitialAuction, true) //price should be accepted now
	require.NoError(t, err)
}
*/

var secondsPerYear = num.DecimalFromFloat(365.25 * 24 * 60 * 60)

func getPriceBounds(price *num.Uint, min, max uint64) (decPr, minPr, maxPr num.Decimal, mn, mx *num.Uint) {
	decPr = price.ToDecimal()
	mn = num.NewUint(min)
	mx = num.NewUint(max)
	minPr = decPr.Sub(mn.ToDecimal())
	maxPr = decPr.Add(mx.ToDecimal())
	return
}

func horizonToYearFraction(horizon int64) num.Decimal {
	hdec := num.DecimalFromFloat(float64(horizon))
	return hdec.Div(secondsPerYear)
}
