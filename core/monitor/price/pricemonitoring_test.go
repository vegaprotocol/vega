// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package price_test

import (
	"context"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/core/monitor/price"
	"code.vegaprotocol.io/vega/core/monitor/price/mocks"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/logging"
	proto "code.vegaprotocol.io/vega/protos/vega"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func TestEmptyParametersList(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	riskModel := mocks.NewMockRangeProvider(ctrl)
	auctionStateMock := mocks.NewMockAuctionState(ctrl)
	currentPrice := num.NewUint(123)

	now := time.Date(1993, 2, 2, 6, 0, 0, 1, time.UTC)

	settings := &types.PriceMonitoringSettings{
		Parameters: &types.PriceMonitoringParameters{
			Triggers: []*types.PriceMonitoringTrigger{},
		},
	}

	auctionStateMock.EXPECT().IsFBA().Return(false).Times(4)
	auctionStateMock.EXPECT().InAuction().Return(false).Times(4)
	auctionStateMock.EXPECT().IsPriceAuction().Return(false).Times(1)
	statevar := mocks.NewMockStateVarEngine(ctrl)
	statevar.EXPECT().RegisterStateVariable(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any())

	pm, err := price.NewMonitor("asset", "market", riskModel, auctionStateMock, settings, statevar, logging.NewTestLogger())
	require.NoError(t, err)
	require.NotNil(t, pm)

	pm.OnTimeUpdate(now)
	b := pm.CheckPrice(context.TODO(), auctionStateMock, currentPrice, true, true)
	require.False(t, b)

	pm.OnTimeUpdate(now.Add(time.Second))
	b = pm.CheckPrice(context.TODO(), auctionStateMock, currentPrice, true, true)
	require.False(t, b)

	pm.OnTimeUpdate(now.Add(time.Minute))
	b = pm.CheckPrice(context.TODO(), auctionStateMock, currentPrice, true, true)
	require.False(t, b)

	pm.OnTimeUpdate(now.Add(time.Hour))
	b = pm.CheckPrice(context.TODO(), auctionStateMock, currentPrice, true, true)
	require.False(t, b)
}

func TestErrorWithNilRiskModel(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	t1 := proto.PriceMonitoringTrigger{Horizon: 7200, Probability: "0.95", AuctionExtension: 300}
	t2 := proto.PriceMonitoringTrigger{Horizon: 3600, Probability: "0.99", AuctionExtension: 60}

	pSet := &proto.PriceMonitoringSettings{
		Parameters: &proto.PriceMonitoringParameters{
			Triggers: []*proto.PriceMonitoringTrigger{&t1, &t2},
		},
	}
	auctionStateMock := mocks.NewMockAuctionState(ctrl)
	settings := types.PriceMonitoringSettingsFromProto(pSet)
	statevar := mocks.NewMockStateVarEngine(ctrl)
	// statevar.EXPECT().RegisterStateVariable(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any())
	// auctionStateMock.EXPECT().IsPriceAuction().Times(1).Return(false)
	pm, err := price.NewMonitor("asset", "market", nil, auctionStateMock, settings, statevar, logging.NewTestLogger())
	require.Error(t, err)
	require.Nil(t, pm)
}

func TestGetHorizonYearFractions(t *testing.T) {
	ctrl := gomock.NewController(t)
	riskModel := mocks.NewMockRangeProvider(ctrl)
	auctionStateMock := mocks.NewMockAuctionState(ctrl)
	t1 := proto.PriceMonitoringTrigger{Horizon: 7200, Probability: "0.95", AuctionExtension: 300}
	t2 := proto.PriceMonitoringTrigger{Horizon: 3600, Probability: "0.99", AuctionExtension: 60}

	pSet := &proto.PriceMonitoringSettings{
		Parameters: &proto.PriceMonitoringParameters{
			Triggers: []*proto.PriceMonitoringTrigger{&t1, &t2},
		},
	}
	settings := types.PriceMonitoringSettingsFromProto(pSet)
	statevar := mocks.NewMockStateVarEngine(ctrl)
	statevar.EXPECT().RegisterStateVariable(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any())
	auctionStateMock.EXPECT().IsPriceAuction().Times(1).Return(false)
	pm, err := price.NewMonitor("asset", "market", riskModel, auctionStateMock, settings, statevar, logging.NewTestLogger())
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
	cp := num.NewUint(123)
	now := time.Date(1993, 2, 2, 6, 0, 0, 1, time.UTC)
	t1 := proto.PriceMonitoringTrigger{Horizon: 7200, Probability: "0.95", AuctionExtension: 300}
	t2 := proto.PriceMonitoringTrigger{Horizon: 3600, Probability: "0.99", AuctionExtension: 60}

	pSet := &proto.PriceMonitoringSettings{
		Parameters: &proto.PriceMonitoringParameters{
			Triggers: []*proto.PriceMonitoringTrigger{&t1, &t2},
		},
	}
	settings := types.PriceMonitoringSettingsFromProto(pSet)

	auctionStateMock.EXPECT().IsFBA().Return(false).Times(4)
	auctionStateMock.EXPECT().InAuction().Return(false).Times(4)
	auctionStateMock.EXPECT().IsPriceAuction().Return(false).Times(1)
	statevar := mocks.NewMockStateVarEngine(ctrl)
	statevar.EXPECT().RegisterStateVariable(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any())

	pm, err := price.NewMonitor("asset", "market", riskModel, auctionStateMock, settings, statevar, logging.NewTestLogger())
	require.NoError(t, err)
	require.NotNil(t, pm)

	pm.OnTimeUpdate(now)
	b := pm.CheckPrice(context.TODO(), auctionStateMock, cp, true, true)
	require.False(t, b)
	one := num.NewUint(1)
	cp1 := num.Sum(cp, one)      // plus 1
	cp2 := num.Sum(cp, one, one) // plus 2
	b = pm.CheckPrice(context.TODO(), auctionStateMock, cp2, true, true)
	require.False(t, b)
	b = pm.CheckPrice(context.TODO(), auctionStateMock, cp1, true, true)
	require.False(t, b)
	b = pm.CheckPrice(context.TODO(), auctionStateMock, cp, true, true)
	require.False(t, b)
}

func TestCheckBoundViolationsWithinCurrentTimeWith2HorizonProbabilityPairs(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	riskModel := mocks.NewMockRangeProvider(ctrl)
	auctionStateMock := mocks.NewMockAuctionState(ctrl)
	cp := num.NewUint(123)
	now := time.Date(1993, 2, 2, 6, 0, 0, 1, time.UTC)
	t1Time, t2Time := int64(60), int64(300)
	t1 := proto.PriceMonitoringTrigger{Horizon: 3600, Probability: "0.99", AuctionExtension: t1Time}
	t2 := proto.PriceMonitoringTrigger{Horizon: 7200, Probability: "0.95", AuctionExtension: t2Time}
	pSet := &proto.PriceMonitoringSettings{
		Parameters: &proto.PriceMonitoringParameters{
			Triggers: []*proto.PriceMonitoringTrigger{&t1, &t2},
		},
	}
	settings := types.PriceMonitoringSettingsFromProto(pSet)

	maxDown1, maxUp1, maxDown2, maxUp2 := num.NewUint(1), num.NewUint(2), num.NewUint(3), num.NewUint(4)

	cpDec := num.DecimalFromUint(cp)
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

	auctionStateMock.EXPECT().IsFBA().Return(false).AnyTimes()
	auctionStateMock.EXPECT().IsPriceAuction().Return(false).AnyTimes()
	auctionStateMock.EXPECT().InAuction().Return(false).Times(14)
	statevar := mocks.NewMockStateVarEngine(ctrl)
	statevar.EXPECT().RegisterStateVariable(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

	pm, err := price.NewMonitor("asset", "market", riskModel, auctionStateMock, settings, statevar, logging.NewTestLogger())
	require.NoError(t, err)
	require.NotNil(t, pm)
	pm.UpdateTestFactors(downFactors, upFactors)

	pm.OnTimeUpdate(now)
	b := pm.CheckPrice(context.TODO(), auctionStateMock, cp, true, true)
	require.False(t, b)

	cPrice := num.Sum(cp, maxUp1)
	cPrice = cPrice.Sub(cPrice, one)
	b = pm.CheckPrice(context.TODO(), auctionStateMock, cPrice, true, true)
	require.False(t, b)

	cPrice = num.Sum(cp, one)
	cPrice = cPrice.Sub(cPrice, maxDown1)
	b = pm.CheckPrice(context.TODO(), auctionStateMock, cPrice, true, true)
	require.False(t, b)

	b = pm.CheckPrice(context.TODO(), auctionStateMock, cPrice, true, true)
	require.False(t, b)

	cPrice = num.Sum(one, cPrice.Sub(cp, maxDown1)) // add one bc price bounds are now using Ceil for min price
	b = pm.CheckPrice(context.TODO(), auctionStateMock, cPrice, true, true)
	require.False(t, b)

	// set the min duration to equal auction extension 1
	pm.SetMinDuration(time.Duration(t1.AuctionExtension) * time.Second)
	end := types.AuctionDuration{Duration: t1.AuctionExtension}
	auctionStateMock.EXPECT().StartPriceAuction(now, &end).Times(1)

	delta := num.Sum(maxUp1, maxUp2)
	cPrice = num.Sum(cp, delta.Div(delta, num.Sum(one, one)))
	b = pm.CheckPrice(context.TODO(), auctionStateMock, cPrice, true, true)
	require.False(t, b)

	// Reinstantiate price monitoring after auction to reset internal state
	pm, err = price.NewMonitor("asset", "market", riskModel, auctionStateMock, settings, statevar, logging.NewTestLogger())
	require.NoError(t, err)
	require.NotNil(t, pm)
	pm.UpdateTestFactors(downFactors, upFactors)

	pm.OnTimeUpdate(now)
	b = pm.CheckPrice(context.TODO(), auctionStateMock, cp, true, true)
	require.False(t, b)

	auctionStateMock.EXPECT().StartPriceAuction(now, &end).Times(1)
	delta = num.Sum(maxDown1, maxDown2)
	cPrice = cPrice.Sub(cp, delta.Div(delta, num.Sum(one, one)))
	b = pm.CheckPrice(context.TODO(), auctionStateMock, cPrice, true, true)
	require.False(t, b)

	// Reinstantiate price monitoring after auction to reset internal state
	pm, err = price.NewMonitor("asset", "market", riskModel, auctionStateMock, settings, statevar, logging.NewTestLogger())
	require.NoError(t, err)
	require.NotNil(t, pm)
	pm.UpdateTestFactors(downFactors, upFactors)

	pm.OnTimeUpdate(now)
	b = pm.CheckPrice(context.TODO(), auctionStateMock, cp, true, true)
	require.False(t, b)

	auctionStateMock.EXPECT().StartPriceAuction(now, &end).Times(1)
	cPrice = num.Sum(cp, num.UintZero().Sub(maxUp2, one)) // max price bound is now floored, so sub 1 to stay below second price bound
	b = pm.CheckPrice(context.TODO(), auctionStateMock, cPrice, true, true)
	require.False(t, b)

	// Reinstantiate price monitoring after auction to reset internal state
	pm, err = price.NewMonitor("asset", "market", riskModel, auctionStateMock, settings, statevar, logging.NewTestLogger())
	require.NoError(t, err)
	require.NotNil(t, pm)
	pm.UpdateTestFactors(downFactors, upFactors)

	pm.OnTimeUpdate(now)
	b = pm.CheckPrice(context.TODO(), auctionStateMock, cp, true, true)
	require.False(t, b)

	auctionStateMock.EXPECT().StartPriceAuction(now, &end).Times(1)
	cPrice = num.Sum(cPrice.Sub(cp, maxDown2), one) // add 1 back, avoid breaching both down limits
	b = pm.CheckPrice(context.TODO(), auctionStateMock, cPrice, true, true)
	require.False(t, b)

	// Reinstantiate price monitoring after auction to reset internal state
	pm, err = price.NewMonitor("asset", "market", riskModel, auctionStateMock, settings, statevar, logging.NewTestLogger())
	require.NoError(t, err)
	require.NotNil(t, pm)
	pm.UpdateTestFactors(downFactors, upFactors)
	pm.OnTimeUpdate(now)
	b = pm.CheckPrice(context.TODO(), auctionStateMock, cp, true, true)
	require.False(t, b)

	end = types.AuctionDuration{Duration: t1.AuctionExtension}
	auctionStateMock.EXPECT().StartPriceAuction(now, &end).Times(1)
	cPrice = num.Sum(cp, maxUp2, maxUp2)
	b = pm.CheckPrice(context.TODO(), auctionStateMock, cPrice, true, true)
	require.False(t, b)
	// recheck with same price, 2nd trigger should get breached now
	end2 := types.AuctionDuration{Duration: t2.AuctionExtension}
	auctionStateMock.EXPECT().InAuction().Return(true).Times(1)
	auctionStateMock.EXPECT().ExtendAuctionPrice(end2).Times(1)

	auctionEnd := now.Add(time.Duration(end.Duration) * time.Second)
	now = auctionEnd.Add(time.Second)
	pm.OnTimeUpdate(now)
	b = pm.CheckPrice(context.TODO(), auctionStateMock, cPrice, true, true)
	require.False(t, b)

	// recheck with same price, auction should end now
	auctionStateMock.EXPECT().InAuction().Return(true).Times(1)
	auctionStateMock.EXPECT().SetReadyToLeave().Times(1)
	auctionEnd = auctionEnd.Add(time.Duration(end2.Duration) * time.Second)
	auctionStateMock.EXPECT().ExpiresAt().Return(&auctionEnd)
	now = auctionEnd.Add(time.Second)
	pm.OnTimeUpdate(now)
	b = pm.CheckPrice(context.TODO(), auctionStateMock, cPrice, true, true)
	require.False(t, b)

	// Check with same price again after exiting, should not start auction now
	auctionStateMock.EXPECT().InAuction().Return(false).Times(3)
	b = pm.CheckPrice(context.TODO(), auctionStateMock, cPrice, true, true)
	require.False(t, b)

	// Update factors and check again, should still be fine
	pm.UpdateTestFactors(downFactors, upFactors)
	pm.OnTimeUpdate(now)
	b = pm.CheckPrice(context.TODO(), auctionStateMock, cPrice, true, true)
	require.False(t, b)

	auctionStateMock.EXPECT().StartPriceAuction(now, &end).Times(1)
	delta = num.Sum(maxDown2, maxDown2)
	cPrice = cPrice.Sub(cp, delta)
	b = pm.CheckPrice(context.TODO(), auctionStateMock, cPrice, true, true)
	require.False(t, b)
}

func TestAuctionStartedAndEndendBy1Trigger(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	riskModel := mocks.NewMockRangeProvider(ctrl)
	auctionStateMock := mocks.NewMockAuctionState(ctrl)
	price1 := num.NewUint(123)
	ctx := context.Background()
	now := time.Date(1993, 2, 2, 6, 0, 0, 1, time.UTC)
	t1 := proto.PriceMonitoringTrigger{Horizon: 600, Probability: "0.95", AuctionExtension: 60}
	t2 := proto.PriceMonitoringTrigger{Horizon: 600, Probability: "0.99", AuctionExtension: 120}
	pSet := &proto.PriceMonitoringSettings{
		Parameters: &proto.PriceMonitoringParameters{
			Triggers: []*proto.PriceMonitoringTrigger{&t1, &t2},
		},
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

	auctionStateMock.EXPECT().IsFBA().Return(false).AnyTimes()
	auctionStateMock.EXPECT().InAuction().Return(false).Times(2)
	auctionStateMock.EXPECT().IsPriceAuction().Return(false).Times(1)
	statevar := mocks.NewMockStateVarEngine(ctrl)
	statevar.EXPECT().RegisterStateVariable(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any())

	pm, err := price.NewMonitor("asset", "market", riskModel, auctionStateMock, settings, statevar, logging.NewTestLogger())
	require.NoError(t, err)
	require.NotNil(t, pm)
	pm.UpdateTestFactors(downFactorsP1, upFactorsP1)
	pm.OnTimeUpdate(now)
	b := pm.CheckPrice(ctx, auctionStateMock, price1, true, true)
	require.False(t, b)

	end := types.AuctionDuration{Duration: t1.AuctionExtension}
	pm.SetMinDuration(time.Duration(t1.AuctionExtension) * time.Second)
	auctionStateMock.EXPECT().StartPriceAuction(now, &end).Times(1)

	delta := num.Sum().Sub(maxUp2, maxUp1)
	cPrice := num.Sum(price1, delta)
	pm.OnTimeUpdate(now)
	b = pm.CheckPrice(context.TODO(), auctionStateMock, cPrice, true, true) // t1 violated only
	require.False(t, b)

	initialAuctionEnd := now.Add(time.Duration(t1.AuctionExtension) * time.Second)

	auctionStateMock.EXPECT().InAuction().Return(true).Times(1)
	auctionStateMock.EXPECT().ExpiresAt().Return(&initialAuctionEnd).Times(1)
	auctionStateMock.EXPECT().SetReadyToLeave().Times(1)

	afterInitialAuction := initialAuctionEnd.Add(time.Nanosecond)
	pm.OnTimeUpdate(afterInitialAuction)
	b = pm.CheckPrice(ctx, auctionStateMock, cPrice, true, true) // price should be accepted now
	require.False(t, b)
}

func TestAuctionStartedAndEndendBy2Triggers(t *testing.T) {
	ctrl := gomock.NewController(t)
	riskModel := mocks.NewMockRangeProvider(ctrl)
	auctionStateMock := mocks.NewMockAuctionState(ctrl)
	price1 := num.NewUint(123)
	ctx := context.Background()
	now := time.Date(1993, 2, 2, 6, 0, 0, 1, time.UTC)
	t1 := proto.PriceMonitoringTrigger{Horizon: 600, Probability: "0.95", AuctionExtension: 60}
	t2 := proto.PriceMonitoringTrigger{Horizon: 600, Probability: "0.99", AuctionExtension: 120}
	pSet := &proto.PriceMonitoringSettings{
		Parameters: &proto.PriceMonitoringParameters{
			Triggers: []*proto.PriceMonitoringTrigger{&t1, &t2},
		},
	}
	settings := types.PriceMonitoringSettingsFromProto(pSet)

	_, _, _, _, maxUp1 := getPriceBounds(price1, 1, 2)
	_, _, _, _, maxUp2 := getPriceBounds(price1, 1*4, 2*4)

	auctionStateMock.EXPECT().IsFBA().Return(false).Times(2)
	auctionStateMock.EXPECT().InAuction().Return(false).Times(2)
	auctionStateMock.EXPECT().IsPriceAuction().Return(false).Times(1)
	statevar := mocks.NewMockStateVarEngine(ctrl)
	statevar.EXPECT().RegisterStateVariable(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any())

	pm, err := price.NewMonitor("asset", "market", riskModel, auctionStateMock, settings, statevar, logging.NewTestLogger())
	require.NoError(t, err)
	require.NotNil(t, pm)

	pm.OnTimeUpdate(now)
	b := pm.CheckPrice(ctx, auctionStateMock, price1, true, true)
	require.False(t, b)

	end := types.AuctionDuration{Duration: t1.AuctionExtension + t2.AuctionExtension}
	pm.SetMinDuration(time.Duration(end.Duration) * time.Second)
	// auctionStateMock.EXPECT().StartPriceAuction(now, &end).Times(1)

	cPrice := num.Sum(price1, maxUp2, maxUp1)
	b = pm.CheckPrice(ctx, auctionStateMock, cPrice, true, true) // t1 violated only
	require.False(t, b)

	initialAuctionEnd := now.Add(time.Duration(t1.AuctionExtension+t2.AuctionExtension) * time.Second)

	auctionStateMock.EXPECT().IsFBA().Return(false).Times(1)
	auctionStateMock.EXPECT().InAuction().Return(true).Times(1)
	auctionStateMock.EXPECT().ExpiresAt().Return(&initialAuctionEnd).Times(1)
	auctionStateMock.EXPECT().SetReadyToLeave().Times(1)

	afterInitialAuction := initialAuctionEnd.Add(time.Nanosecond)
	pm.OnTimeUpdate(afterInitialAuction)
	b = pm.CheckPrice(ctx, auctionStateMock, cPrice, true, true) // price should be accepted now
	require.False(t, b)
}

func TestAuctionStartedAndEndendBy1TriggerAndExtendedBy2nd(t *testing.T) {
	// Also verifies that GetCurrentBounds() works as expected
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	riskModel := mocks.NewMockRangeProvider(ctrl)
	auctionStateMock := mocks.NewMockAuctionState(ctrl)
	price1 := num.NewUint(123)
	now := time.Date(1993, 2, 2, 6, 0, 0, 1, time.UTC)
	t1 := proto.PriceMonitoringTrigger{Horizon: 600, Probability: "0.95", AuctionExtension: 60}
	t2 := proto.PriceMonitoringTrigger{Horizon: 600, Probability: "0.99", AuctionExtension: 120}
	pSet := &proto.PriceMonitoringSettings{
		Parameters: &proto.PriceMonitoringParameters{
			Triggers: []*proto.PriceMonitoringTrigger{&t1, &t2},
		},
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
	auctionStateMock.EXPECT().IsPriceAuction().Return(false).Times(1)
	auctionStateMock.EXPECT().InAuction().Return(false).Times(2)
	statevar := mocks.NewMockStateVarEngine(ctrl)
	statevar.EXPECT().RegisterStateVariable(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any())

	pm, err := price.NewMonitor("asset", "market", riskModel, auctionStateMock, settings, statevar, logging.NewTestLogger())
	downFactors := []num.Decimal{pMin1.Div(decPrice), pMin2.Div(decPrice)}
	upFactors := []num.Decimal{pMax1.Div(decPrice), pMax2.Div(decPrice)}
	pm.UpdateTestFactors(downFactors, upFactors)

	require.NoError(t, err)
	require.NotNil(t, pm)

	pm.OnTimeUpdate(now)
	b := pm.CheckPrice(ctx, auctionStateMock, price1, true, true)
	require.False(t, b)

	bounds := pm.GetCurrentBounds()
	require.Len(t, bounds, 2)
	require.Equal(t, bounds[0].Trigger.IntoProto(), &t1)
	require.True(t, bounds[0].MinValidPrice.EQ(t1lb1))
	require.True(t, bounds[0].MaxValidPrice.EQ(t1ub1))
	require.Equal(t, bounds[0].ReferencePrice, decPrice)
	require.Equal(t, bounds[1].Trigger.IntoProto(), &t2)
	require.True(t, bounds[1].MinValidPrice.EQ(t2lb1))
	require.True(t, bounds[1].MaxValidPrice.EQ(t2ub1))
	require.Equal(t, bounds[1].ReferencePrice, decPrice)

	end := types.AuctionDuration{Duration: t1.AuctionExtension}
	pm.SetMinDuration(time.Duration(end.Duration) * time.Second)
	auctionStateMock.EXPECT().StartPriceAuction(now, &end).Times(1)

	cPrice := num.Sum(price1, maxUp2)
	cPrice.Sub(cPrice, maxUp1)
	cp2 := cPrice
	b = pm.CheckPrice(ctx, auctionStateMock, cp2, true, true) // t1 violated only
	require.False(t, b)

	initialAuctionEnd := now.Add(time.Duration(t1.AuctionExtension) * time.Second)

	auctionStateMock.EXPECT().IsFBA().Return(false).Times(1)
	auctionStateMock.EXPECT().InAuction().Return(true).Times(1)
	auctionStateMock.EXPECT().ExpiresAt().Return(&initialAuctionEnd).Times(1)

	bounds = pm.GetCurrentBounds()
	require.Len(t, bounds, 1)
	require.Equal(t, bounds[0].Trigger.IntoProto(), &t2)
	require.True(t, bounds[0].MinValidPrice.EQ(t2lb1))
	require.True(t, bounds[0].MaxValidPrice.EQ(t2ub1))
	require.Equal(t, bounds[0].ReferencePrice, decPrice)

	afterInitialAuction := initialAuctionEnd.Add(time.Nanosecond)
	now = afterInitialAuction

	cPrice = num.Sum(price1, maxUp2, maxUp1)
	end2 := types.AuctionDuration{Duration: t2.AuctionExtension}
	auctionStateMock.EXPECT().ExtendAuctionPrice(end2).Times(1)
	pm.OnTimeUpdate(afterInitialAuction)
	cp3 := cPrice
	b = pm.CheckPrice(ctx, auctionStateMock, cp3, true, true) // price should violated 2nd trigger and result in auction extension
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
	auctionStateMock.EXPECT().SetReadyToLeave().Times(1)

	afterExtendedAuction := extendedAuctionEnd.Add(time.Nanosecond)
	pm.OnTimeUpdate(afterExtendedAuction)
	b = pm.CheckPrice(ctx, auctionStateMock, cp3, true, true) // price should be accepted now
	require.False(t, b)
}

func TestAuctionStartedBy1TriggerAndNotExtendedBy2ndStaleTrigger(t *testing.T) {
	// Also verifies that GetCurrentBounds() works as expected
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	riskModel := mocks.NewMockRangeProvider(ctrl)
	auctionStateMock := mocks.NewMockAuctionState(ctrl)
	price1 := num.NewUint(123)
	now := time.Date(1993, 2, 2, 6, 0, 0, 1, time.UTC)
	t1 := proto.PriceMonitoringTrigger{Horizon: 6, Probability: "0.95", AuctionExtension: 60}
	t2 := proto.PriceMonitoringTrigger{Horizon: 6, Probability: "0.99", AuctionExtension: 120}
	pSet := &proto.PriceMonitoringSettings{
		Parameters: &proto.PriceMonitoringParameters{
			Triggers: []*proto.PriceMonitoringTrigger{&t1, &t2},
		},
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
	auctionStateMock.EXPECT().IsPriceAuction().Return(false).Times(1)
	auctionStateMock.EXPECT().InAuction().Return(false).Times(2)
	statevar := mocks.NewMockStateVarEngine(ctrl)
	statevar.EXPECT().RegisterStateVariable(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any())

	pm, err := price.NewMonitor("asset", "market", riskModel, auctionStateMock, settings, statevar, logging.NewTestLogger())
	downFactors := []num.Decimal{pMin1.Div(decPrice), pMin2.Div(decPrice)}
	upFactors := []num.Decimal{pMax1.Div(decPrice), pMax2.Div(decPrice)}
	pm.UpdateTestFactors(downFactors, upFactors)

	require.NoError(t, err)
	require.NotNil(t, pm)

	pm.OnTimeUpdate(now)
	b := pm.CheckPrice(ctx, auctionStateMock, price1, true, true)
	require.False(t, b)

	bounds := pm.GetCurrentBounds()
	require.Len(t, bounds, 2)
	require.Equal(t, bounds[0].Trigger.IntoProto(), &t1)
	require.True(t, bounds[0].MinValidPrice.EQ(t1lb1))
	require.True(t, bounds[0].MaxValidPrice.EQ(t1ub1))
	require.Equal(t, bounds[0].ReferencePrice, decPrice)
	require.Equal(t, bounds[1].Trigger.IntoProto(), &t2)
	require.True(t, bounds[1].MinValidPrice.EQ(t2lb1))
	require.True(t, bounds[1].MaxValidPrice.EQ(t2ub1))
	require.Equal(t, bounds[1].ReferencePrice, decPrice)

	end := types.AuctionDuration{Duration: t1.AuctionExtension}
	pm.SetMinDuration(time.Duration(end.Duration) * time.Second)
	auctionStateMock.EXPECT().StartPriceAuction(now, &end).Times(1)

	cPrice := num.Sum(price1, maxUp2)
	cPrice.Sub(cPrice, maxUp1)
	cp2 := cPrice
	b = pm.CheckPrice(ctx, auctionStateMock, cp2, true, true) // t1 violated only
	require.False(t, b)

	initialAuctionEnd := now.Add(time.Duration(t1.AuctionExtension) * time.Second)

	auctionStateMock.EXPECT().IsFBA().Return(false).Times(1)
	auctionStateMock.EXPECT().InAuction().Return(true).Times(1)

	bounds = pm.GetCurrentBounds()
	require.Len(t, bounds, 1)
	require.Equal(t, bounds[0].Trigger.IntoProto(), &t2)
	require.True(t, bounds[0].MinValidPrice.EQ(t2lb1))
	require.True(t, bounds[0].MaxValidPrice.EQ(t2ub1))
	require.Equal(t, bounds[0].ReferencePrice, decPrice)

	afterInitialAuction := initialAuctionEnd.Add(time.Nanosecond)
	now = afterInitialAuction

	auctionStateMock.EXPECT().ExtendAuctionPrice(gomock.Any()).Times(1)

	cPrice = num.Sum(price1, maxUp2, maxUp1)
	pm.OnTimeUpdate(afterInitialAuction)
	cp3 := cPrice
	b = pm.CheckPrice(ctx, auctionStateMock, cp3, true, true) // price should violated 2nd trigger and result in auction extension
	require.False(t, b)
}

func TestMarketInOpeningAuction(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	riskModel := mocks.NewMockRangeProvider(ctrl)
	auctionStateMock := mocks.NewMockAuctionState(ctrl)
	currentPrice := num.NewUint(123)
	t1 := proto.PriceMonitoringTrigger{Horizon: 7200, Probability: "0.95", AuctionExtension: 300}
	now := time.Date(1993, 2, 2, 6, 0, 0, 1, time.UTC)
	pSet := &proto.PriceMonitoringSettings{
		Parameters: &proto.PriceMonitoringParameters{
			Triggers: []*proto.PriceMonitoringTrigger{&t1},
		},
	}
	settings := types.PriceMonitoringSettingsFromProto(pSet)

	ctx := context.Background()

	auctionStateMock.EXPECT().IsFBA().Return(false).Times(1)
	auctionStateMock.EXPECT().IsPriceAuction().Return(false).Times(1)
	auctionStateMock.EXPECT().InAuction().Return(true).Times(1)
	end := now.Add(time.Second)
	auctionStateMock.EXPECT().ExpiresAt().Return(&end).Times(1)
	statevar := mocks.NewMockStateVarEngine(ctrl)
	statevar.EXPECT().RegisterStateVariable(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any())

	pm, err := price.NewMonitor("asset", "market", riskModel, auctionStateMock, settings, statevar, logging.NewTestLogger())
	require.NoError(t, err)
	require.NotNil(t, pm)

	pm.OnTimeUpdate(now)
	b := pm.CheckPrice(ctx, auctionStateMock, currentPrice, true, true)
	require.False(t, b)
}

func TestMarketInGenericAuction(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	riskModel := mocks.NewMockRangeProvider(ctrl)
	auctionStateMock := mocks.NewMockAuctionState(ctrl)
	currentPrice := num.NewUint(123)
	t1 := proto.PriceMonitoringTrigger{Horizon: 7200, Probability: "0.95", AuctionExtension: 300}
	now := time.Date(1993, 2, 2, 6, 0, 0, 1, time.UTC)

	pSet := &proto.PriceMonitoringSettings{
		Parameters: &proto.PriceMonitoringParameters{
			Triggers: []*proto.PriceMonitoringTrigger{&t1},
		},
	}
	settings := types.PriceMonitoringSettingsFromProto(pSet)

	_, _, _, maxDown, maxUp := getPriceBounds(currentPrice, 5, 10)
	one := num.NewUint(1)
	ctx := context.Background()

	// price monitoring starts with auction, not initialised, so there's no fixed price level it'll check
	auctionStateMock.EXPECT().IsFBA().Return(false).AnyTimes()
	auctionStateMock.EXPECT().InAuction().Return(true).Times(4)
	auctionStateMock.EXPECT().IsPriceAuction().Return(false).Times(1)
	auctionStateMock.EXPECT().CanLeave().Return(false).AnyTimes()
	statevar := mocks.NewMockStateVarEngine(ctrl)
	statevar.EXPECT().RegisterStateVariable(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any())
	end := now.Add(time.Second)
	auctionStateMock.EXPECT().ExpiresAt().Return(&end).AnyTimes()
	pm, err := price.NewMonitor("asset", "market", riskModel, auctionStateMock, settings, statevar, logging.NewTestLogger())
	require.NoError(t, err)
	require.NotNil(t, pm)

	pm.OnTimeUpdate(now)
	pm.ResetPriceHistory(currentPrice)

	cPrice := num.Sum(currentPrice, maxUp)
	cPrice.Sub(cPrice, one)
	b := pm.CheckPrice(ctx, auctionStateMock, cPrice, true, true)
	require.False(t, b)

	cPrice.Sub(num.Sum(currentPrice, one), maxDown)
	cp3 := cPrice
	b = pm.CheckPrice(ctx, auctionStateMock, cp3, true, true)
	require.False(t, b)

	extension := types.AuctionDuration{Duration: t1.AuctionExtension}
	auctionStateMock.EXPECT().ExtendAuctionPrice(extension).Times(1)
	cPrice = num.Sum(currentPrice, maxUp, maxUp)
	cp4 := cPrice
	b = pm.CheckPrice(ctx, auctionStateMock, cp4, true, true)
	require.False(t, b)

	cPrice = num.Sum(maxDown, maxDown)
	cPrice.Sub(currentPrice, cPrice)
	cp5 := cPrice
	b = pm.CheckPrice(ctx, auctionStateMock, cp5, true, true)
	require.False(t, b)
}

func TestGetValidPriceRange_NoTriggers(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	riskModel := mocks.NewMockRangeProvider(ctrl)
	auctionStateMock := mocks.NewMockAuctionState(ctrl)
	currentPrice := num.NewUint(123)
	now := time.Date(1993, 2, 2, 6, 0, 0, 1, time.UTC)
	ctx := context.Background()

	settings := &types.PriceMonitoringSettings{
		Parameters: &types.PriceMonitoringParameters{
			Triggers: []*types.PriceMonitoringTrigger{},
		},
	}

	auctionStateMock.EXPECT().IsFBA().Return(false).Times(1)
	auctionStateMock.EXPECT().InAuction().Return(false).Times(1)
	auctionStateMock.EXPECT().IsPriceAuction().Return(false).Times(1)
	statevar := mocks.NewMockStateVarEngine(ctrl)
	statevar.EXPECT().RegisterStateVariable(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any())

	pm, err := price.NewMonitor("asset", "market", riskModel, auctionStateMock, settings, statevar, logging.NewTestLogger())
	require.NoError(t, err)
	require.NotNil(t, pm)

	expMax := num.MaxUint()
	min, max := pm.GetValidPriceRange()
	require.True(t, min.Representation().IsZero())
	require.Equal(t, expMax.String(), max.Representation().String())

	pm.OnTimeUpdate(now)
	b := pm.CheckPrice(ctx, auctionStateMock, currentPrice, true, true)
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
	now := time.Date(1993, 2, 2, 6, 0, 0, 1, time.UTC)
	var t1Time int64 = 60
	var t2Time int64 = 300
	t1 := proto.PriceMonitoringTrigger{Horizon: 3600, Probability: "0.99", AuctionExtension: t1Time}
	t2 := proto.PriceMonitoringTrigger{Horizon: 7200, Probability: "0.95", AuctionExtension: t2Time}
	pSet := &proto.PriceMonitoringSettings{
		Parameters: &proto.PriceMonitoringParameters{
			Triggers: []*proto.PriceMonitoringTrigger{&t1, &t2},
		},
	}
	settings := types.PriceMonitoringSettingsFromProto(pSet)

	ctx := context.Background()
	_, pMin1, pMax1, maxDown1, maxUp1 := getPriceBounds(currentPrice, 1, 2)
	_, pMin2, pMax2, _, _ := getPriceBounds(currentPrice, 3, 4)
	one := num.NewUint(1)
	currentPriceD := currentPrice.ToDecimal()
	auctionStateMock.EXPECT().IsFBA().Return(false).Times(12)
	auctionStateMock.EXPECT().InAuction().Return(false).Times(12)
	auctionStateMock.EXPECT().IsPriceAuction().Return(false).Times(1)
	statevar := mocks.NewMockStateVarEngine(ctrl)
	statevar.EXPECT().RegisterStateVariable(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any())

	pm, err := price.NewMonitor("asset", "market", riskModel, auctionStateMock, settings, statevar, logging.NewTestLogger())
	require.NoError(t, err)
	require.NotNil(t, pm)
	downFactors := []num.Decimal{pMin1.Div(currentPriceD), pMin2.Div(currentPriceD)}
	upFactors := []num.Decimal{pMax1.Div(currentPriceD), pMax2.Div(currentPriceD)}

	pm.UpdateTestFactors(downFactors, upFactors)

	pm.OnTimeUpdate(now)
	b := pm.CheckPrice(ctx, auctionStateMock, currentPrice, true, true)
	require.False(t, b)

	_, _ = pm.GetValidPriceRange()
	now = now.Add(time.Second)
	cPrice := num.Sum(currentPrice, maxUp1)
	cPrice.Sub(cPrice, one)
	pm.OnTimeUpdate(now)
	b = pm.CheckPrice(ctx, auctionStateMock, cPrice, true, true)
	require.False(t, b)

	_, _ = pm.GetValidPriceRange()
	now = now.Add(time.Minute)
	cPrice = num.Sum(currentPrice, one)
	cPrice.Sub(cPrice, maxDown1)
	pm.OnTimeUpdate(now)
	b = pm.CheckPrice(ctx, auctionStateMock, cPrice, true, true)
	require.False(t, b)

	_, _ = pm.GetValidPriceRange()
	now = now.Add(time.Hour)
	cPrice = num.Sum(currentPrice, maxUp1)
	cPrice.Sub(cPrice, one)
	pm.OnTimeUpdate(now)
	b = pm.CheckPrice(ctx, auctionStateMock, cPrice, true, true)
	require.False(t, b)

	_, _ = pm.GetValidPriceRange()
	now = now.Add(time.Minute)
	cPrice.Sub(currentPrice, maxDown1)
	cPrice.AddSum(one)
	pm.OnTimeUpdate(now)
	b = pm.CheckPrice(ctx, auctionStateMock, cPrice, true, true)
	require.False(t, b)

	min, max := pm.GetValidPriceRange()
	b = pm.CheckPrice(ctx, auctionStateMock, min.Representation(), true, true)
	require.False(t, b)

	b = pm.CheckPrice(ctx, auctionStateMock, max.Representation(), true, true)
	require.False(t, b)

	// Should trigger an auction
	auctionStateMock.EXPECT().StartPriceAuction(now, gomock.Any()).Times(1)

	cPrice.Sub(min.Representation(), one)
	b = pm.CheckPrice(ctx, auctionStateMock, cPrice, true, true)
	require.False(t, b)

	now = now.Add(time.Second)
	pm.OnTimeUpdate(now)
	b = pm.CheckPrice(ctx, auctionStateMock, currentPrice, true, true)
	require.False(t, b)

	min, max = pm.GetValidPriceRange()

	b = pm.CheckPrice(ctx, auctionStateMock, min.Representation(), true, true)
	require.False(t, b)

	b = pm.CheckPrice(ctx, auctionStateMock, max.Representation(), true, true)
	require.False(t, b)

	// Should trigger an auction
	auctionStateMock.EXPECT().StartPriceAuction(now, gomock.Any()).Times(1)
	cPrice.Add(max.Representation(), one)
	cp11 := cPrice
	b = pm.CheckPrice(ctx, auctionStateMock, cp11, true, true)
	require.False(t, b)
}

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
