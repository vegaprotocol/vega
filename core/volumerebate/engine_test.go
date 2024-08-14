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

package volumerebate_test

import (
	"bytes"
	"context"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/core/volumerebate"
	"code.vegaprotocol.io/vega/core/volumerebate/mocks"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/libs/proto"
	"code.vegaprotocol.io/vega/protos/vega"
	snapshotpb "code.vegaprotocol.io/vega/protos/vega/snapshot/v1"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func assertSnapshotMatches(t *testing.T, key string, expectedHash []byte) *volumerebate.SnapshottedEngine {
	t.Helper()

	loadCtrl := gomock.NewController(t)
	loadBroker := mocks.NewMockBroker(loadCtrl)
	loadMarketActivityTracker := mocks.NewMockMarketActivityTracker(loadCtrl)
	loadEngine := volumerebate.NewSnapshottedEngine(loadBroker, loadMarketActivityTracker)

	pl := snapshotpb.Payload{}
	require.NoError(t, proto.Unmarshal(expectedHash, &pl))

	loadEngine.LoadState(context.Background(), types.PayloadFromProto(&pl))
	loadedHashEmpty, _, err := loadEngine.GetState(key)
	require.NoError(t, err)
	require.True(t, bytes.Equal(expectedHash, loadedHashEmpty))
	return loadEngine
}

func TestVolumeRebateProgramLifecycle(t *testing.T) {
	key := (&types.PayloadVolumeRebateProgram{}).Key()
	ctrl := gomock.NewController(t)
	broker := mocks.NewMockBroker(ctrl)
	marketActivityTracker := mocks.NewMockMarketActivityTracker(ctrl)
	engine := volumerebate.NewSnapshottedEngine(broker, marketActivityTracker)

	// test snapshot with empty engine
	hashEmpty, _, err := engine.GetState(key)
	require.NoError(t, err)
	assertSnapshotMatches(t, key, hashEmpty)

	now := time.Now()

	p1 := &types.VolumeRebateProgram{
		ID:                    "1",
		Version:               0,
		EndOfProgramTimestamp: now.Add(time.Hour * 1),
		WindowLength:          1,
		VolumeRebateBenefitTiers: []*types.VolumeRebateBenefitTier{
			{MinimumPartyMakerVolumeFraction: num.DecimalFromFloat(0.1000), AdditionalMakerRebate: num.DecimalFromFloat(0.1)},
			{MinimumPartyMakerVolumeFraction: num.DecimalFromFloat(0.2000), AdditionalMakerRebate: num.DecimalFromFloat(0.2)},
			{MinimumPartyMakerVolumeFraction: num.DecimalFromFloat(0.3000), AdditionalMakerRebate: num.DecimalFromFloat(0.5)},
			{MinimumPartyMakerVolumeFraction: num.DecimalFromFloat(0.4000), AdditionalMakerRebate: num.DecimalFromFloat(1)},
		},
	}
	// add the program
	engine.UpdateProgram(p1)

	// expect an event for the started program
	broker.EXPECT().Send(gomock.Any()).DoAndReturn(func(evt events.Event) {
		e := evt.(*events.VolumeRebateProgramStarted)
		require.Equal(t, p1.IntoProto(), e.GetVolumeRebateProgramStarted().Program)
	}).Times(1)

	// activate the program
	engine.OnEpoch(context.Background(), types.Epoch{Action: vega.EpochAction_EPOCH_ACTION_START, StartTime: now})

	// check snapshot with new program
	hashWithNew, _, err := engine.GetState(key)
	require.NoError(t, err)
	assertSnapshotMatches(t, key, hashWithNew)

	// add a new program
	p2 := &types.VolumeRebateProgram{
		ID:                    "1",
		Version:               1,
		EndOfProgramTimestamp: now.Add(time.Hour * 2),
		WindowLength:          1,
		VolumeRebateBenefitTiers: []*types.VolumeRebateBenefitTier{
			{MinimumPartyMakerVolumeFraction: num.DecimalFromFloat(0.2000), AdditionalMakerRebate: num.DecimalFromFloat(0.2)},
			{MinimumPartyMakerVolumeFraction: num.DecimalFromFloat(0.3000), AdditionalMakerRebate: num.DecimalFromFloat(0.5)},
			{MinimumPartyMakerVolumeFraction: num.DecimalFromFloat(0.1000), AdditionalMakerRebate: num.DecimalFromFloat(0.1)},
			{MinimumPartyMakerVolumeFraction: num.DecimalFromFloat(0.4000), AdditionalMakerRebate: num.DecimalFromFloat(1)},
		},
	}
	// add the new program
	engine.UpdateProgram(p2)

	// check snapshot with new program and current
	hashWithNewAndCurrent, _, err := engine.GetState(key)
	require.NoError(t, err)
	assertSnapshotMatches(t, key, hashWithNewAndCurrent)

	// // expect a program updated event
	broker.EXPECT().Send(gomock.Any()).DoAndReturn(func(evt events.Event) {
		e := evt.(*events.VolumeRebateProgramUpdated)
		require.Equal(t, p2.IntoProto(), e.GetVolumeRebateProgramUpdated().Program)
	}).Times(1)
	engine.OnEpoch(context.Background(), types.Epoch{Action: vega.EpochAction_EPOCH_ACTION_START, StartTime: now.Add(time.Hour * 1)})

	// // expire the program
	broker.EXPECT().Send(gomock.Any()).DoAndReturn(func(evt events.Event) {
		e := evt.(*events.VolumeRebateProgramEnded)
		require.Equal(t, p2.Version, e.GetVolumeRebateProgramEnded().Version)
	}).Times(1)
	engine.OnEpoch(context.Background(), types.Epoch{Action: vega.EpochAction_EPOCH_ACTION_START, StartTime: now.Add(time.Hour * 2)})

	// check snapshot with terminated program
	hashWithPostTermination, _, err := engine.GetState(key)
	require.NoError(t, err)
	assertSnapshotMatches(t, key, hashWithPostTermination)
}

func TestRebateFactor(t *testing.T) {
	key := (&types.PayloadVolumeRebateProgram{}).Key()
	ctrl := gomock.NewController(t)
	broker := mocks.NewMockBroker(ctrl)
	marketActivityTracker := mocks.NewMockMarketActivityTracker(ctrl)
	engine := volumerebate.NewSnapshottedEngine(broker, marketActivityTracker)
	engine.OnMarketFeeFactorsBuyBackFeeUpdate(context.Background(), num.NewDecimalFromFloat(0.5))
	engine.OnMarketFeeFactorsTreasuryFeeUpdate(context.Background(), num.NewDecimalFromFloat(0.5))
	currentTime := time.Now()

	p1 := &types.VolumeRebateProgram{
		ID:                    "1",
		Version:               0,
		EndOfProgramTimestamp: currentTime.Add(time.Hour * 1),
		WindowLength:          1,
		VolumeRebateBenefitTiers: []*types.VolumeRebateBenefitTier{
			{MinimumPartyMakerVolumeFraction: num.DecimalFromFloat(0.1000), AdditionalMakerRebate: num.DecimalFromFloat(0.1)},
			{MinimumPartyMakerVolumeFraction: num.DecimalFromFloat(0.2000), AdditionalMakerRebate: num.DecimalFromFloat(0.2)},
			{MinimumPartyMakerVolumeFraction: num.DecimalFromFloat(0.3000), AdditionalMakerRebate: num.DecimalFromFloat(0.5)},
			{MinimumPartyMakerVolumeFraction: num.DecimalFromFloat(0.4000), AdditionalMakerRebate: num.DecimalFromFloat(1)},
		},
	}
	// add the program
	engine.UpdateProgram(p1)

	// activate the program
	currentEpoch := uint64(1)
	expectProgramStarted(t, broker, p1)
	startEpoch(t, engine, currentEpoch, currentTime)

	// so now we have a program active so at the end of the epoch lets return for some parties some notional
	marketActivityTracker.EXPECT().CalculateTotalMakerContributionInQuantum(gomock.Any()).Return(map[string]*num.Uint{
		"p1": num.NewUint(900),
		"p2": num.NewUint(1000),
		"p3": num.NewUint(1001),
		"p4": num.NewUint(2000),
		"p5": num.NewUint(3000),
		"p6": num.NewUint(4000),
		"p7": num.NewUint(5000),
	},
		map[string]num.Decimal{
			"p1": num.DecimalFromFloat(0.09),
			"p2": num.DecimalFromFloat(0.1000),
			"p3": num.DecimalFromFloat(0.1001),
			"p4": num.DecimalFromFloat(0.2000),
			"p5": num.DecimalFromFloat(0.3000),
			"p6": num.DecimalFromFloat(0.4000),
			"p7": num.DecimalFromFloat(0.5000),
		}).Times(1)

	// end the epoch to get the market activity recorded
	expectStatsUpdatedWithUnqualifiedParties(t, broker)
	currentTime = currentTime.Add(1 * time.Minute)
	endEpoch(t, engine, currentEpoch, currentTime.Add(1*time.Minute))

	// start a new epoch for the rebate factors to be in place
	currentEpoch += 1
	startEpoch(t, engine, currentEpoch, currentTime)

	// check snapshot with terminated program
	hashWithEpochNotionalsData, _, err := engine.GetState(key)
	require.NoError(t, err)
	loadedEngine := assertSnapshotMatches(t, key, hashWithEpochNotionalsData)
	loadedEngine.OnMarketFeeFactorsBuyBackFeeUpdate(context.Background(), num.NewDecimalFromFloat(0.5))
	loadedEngine.OnMarketFeeFactorsTreasuryFeeUpdate(context.Background(), num.NewDecimalFromFloat(0.5))

	// party does not exist
	require.Equal(t, num.DecimalZero(), engine.VolumeRebateFactorForParty("p8"))
	require.Equal(t, num.DecimalZero(), loadedEngine.VolumeRebateFactorForParty("p8"))
	// party is not eligible
	require.Equal(t, num.DecimalZero(), engine.VolumeRebateFactorForParty("p1"))
	require.Equal(t, num.DecimalZero(), loadedEngine.VolumeRebateFactorForParty("p1"))
	// volume between 1000/2000
	require.Equal(t, "0.1", engine.VolumeRebateFactorForParty("p2").String())
	require.Equal(t, "0.1", loadedEngine.VolumeRebateFactorForParty("p2").String())
	require.Equal(t, "0.1", engine.VolumeRebateFactorForParty("p3").String())
	require.Equal(t, "0.1", loadedEngine.VolumeRebateFactorForParty("p3").String())

	// volume 2000<=x<3000
	require.Equal(t, "0.2", engine.VolumeRebateFactorForParty("p4").String())
	require.Equal(t, "0.2", loadedEngine.VolumeRebateFactorForParty("p4").String())

	// volume 3000<=x<4000
	require.Equal(t, "0.5", engine.VolumeRebateFactorForParty("p5").String())
	require.Equal(t, "0.5", loadedEngine.VolumeRebateFactorForParty("p5").String())

	// volume >=4000
	require.Equal(t, "1", engine.VolumeRebateFactorForParty("p6").String())
	require.Equal(t, "1", loadedEngine.VolumeRebateFactorForParty("p6").String())
	require.Equal(t, "1", engine.VolumeRebateFactorForParty("p7").String())
	require.Equal(t, "1", loadedEngine.VolumeRebateFactorForParty("p7").String())

	marketActivityTracker.EXPECT().CalculateTotalMakerContributionInQuantum(gomock.Any()).Return(map[string]*num.Uint{}, map[string]num.Decimal{}).Times(1)

	expectStatsUpdated(t, broker)
	currentTime = p1.EndOfProgramTimestamp
	endEpoch(t, engine, currentEpoch, currentTime)

	// terminate the program
	currentEpoch += 1
	expectProgramEnded(t, broker, p1)
	startEpoch(t, engine, currentEpoch, currentTime)

	hashAfterProgramEnded, _, err := engine.GetState(key)
	require.NoError(t, err)
	loadedEngine = assertSnapshotMatches(t, key, hashAfterProgramEnded)

	// no rebate for terminated program
	for _, p := range []string{"p1", "p2", "p3", "p4", "p5", "p6", "p7", "p8"} {
		require.Equal(t, num.DecimalZero(), engine.VolumeRebateFactorForParty(types.PartyID(p)))
		require.Equal(t, num.DecimalZero(), loadedEngine.VolumeRebateFactorForParty(types.PartyID(p)))
	}
}

func TestRebateFactorWithWindow(t *testing.T) {
	key := (&types.PayloadVolumeRebateProgram{}).Key()
	ctrl := gomock.NewController(t)
	broker := mocks.NewMockBroker(ctrl)
	marketActivityTracker := mocks.NewMockMarketActivityTracker(ctrl)
	engine := volumerebate.NewSnapshottedEngine(broker, marketActivityTracker)
	engine.OnMarketFeeFactorsBuyBackFeeUpdate(context.Background(), num.DecimalFromFloat(0.5))
	engine.OnMarketFeeFactorsTreasuryFeeUpdate(context.Background(), num.DecimalFromFloat(0.5))
	currentTime := time.Now()

	p1 := &types.VolumeRebateProgram{
		ID:                    "1",
		Version:               0,
		EndOfProgramTimestamp: currentTime.Add(time.Hour * 1),
		WindowLength:          2,
		VolumeRebateBenefitTiers: []*types.VolumeRebateBenefitTier{
			{MinimumPartyMakerVolumeFraction: num.DecimalFromFloat(0.1), AdditionalMakerRebate: num.DecimalFromFloat(0.1)},
			{MinimumPartyMakerVolumeFraction: num.DecimalFromFloat(0.2), AdditionalMakerRebate: num.DecimalFromFloat(0.2)},
			{MinimumPartyMakerVolumeFraction: num.DecimalFromFloat(0.3), AdditionalMakerRebate: num.DecimalFromFloat(0.5)},
			{MinimumPartyMakerVolumeFraction: num.DecimalFromFloat(0.4), AdditionalMakerRebate: num.DecimalFromFloat(1)},
		},
	}
	// add the program
	engine.UpdateProgram(p1)

	// expect an event for the started program
	expectProgramStarted(t, broker, p1)
	// activate the program
	currentEpoch := uint64(1)
	startEpoch(t, engine, currentEpoch, currentTime)

	// so now we have a program active so at the end of the epoch lets return for some parties some notional
	marketActivityTracker.EXPECT().CalculateTotalMakerContributionInQuantum(gomock.Any()).Return(
		map[string]*num.Uint{
			"p1": num.NewUint(900),
			"p2": num.NewUint(1000),
			"p3": num.NewUint(1001),
			"p4": num.NewUint(2000),
			"p5": num.NewUint(3000),
			"p6": num.NewUint(4000),
			"p7": num.NewUint(5000),
		}, map[string]num.Decimal{
			"p1": num.DecimalFromFloat(0.0900),
			"p2": num.DecimalFromFloat(0.1000),
			"p3": num.DecimalFromFloat(0.1001),
			"p4": num.DecimalFromFloat(0.2000),
			"p5": num.DecimalFromFloat(0.3000),
			"p6": num.DecimalFromFloat(0.4000),
			"p7": num.DecimalFromFloat(0.5000),
		}).Times(1)

	expectStatsUpdatedWithUnqualifiedParties(t, broker)
	currentTime = currentTime.Add(1 * time.Minute)
	endEpoch(t, engine, currentEpoch, currentTime)
	// start a new epoch for the rebate factors to be in place

	// party does not exist
	require.Equal(t, num.DecimalZero(), engine.VolumeRebateFactorForParty("p8"))
	// volume 900
	require.Equal(t, num.DecimalZero(), engine.VolumeRebateFactorForParty("p1"))
	// volume 1000
	require.Equal(t, "0.1", engine.VolumeRebateFactorForParty("p2").String())
	// volume 1001
	require.Equal(t, "0.1", engine.VolumeRebateFactorForParty("p3").String())
	// volume 2000
	require.Equal(t, "0.2", engine.VolumeRebateFactorForParty("p4").String())
	// volume 3000
	require.Equal(t, "0.5", engine.VolumeRebateFactorForParty("p5").String())
	// volume 4000
	require.Equal(t, "1", engine.VolumeRebateFactorForParty("p6").String())
	// volume 5000
	require.Equal(t, "1", engine.VolumeRebateFactorForParty("p7").String())

	engine.OnMarketFeeFactorsBuyBackFeeUpdate(context.Background(), num.DecimalFromFloat(0.1))
	engine.OnMarketFeeFactorsTreasuryFeeUpdate(context.Background(), num.DecimalFromFloat(0.2))

	// running for another epoch
	marketActivityTracker.EXPECT().CalculateTotalMakerContributionInQuantum(gomock.Any()).Return(map[string]*num.Uint{
		"p8": num.NewUint(2000),
		"p1": num.NewUint(1500),
		"p5": num.NewUint(4000),
		"p6": num.NewUint(4000),
	},
		map[string]num.Decimal{
			"p8": num.DecimalFromFloat(0.2000),
			"p1": num.DecimalFromFloat(0.1500),
			"p5": num.DecimalFromFloat(0.4000),
			"p6": num.DecimalFromFloat(0.4000),
		}).Times(1)

	expectStatsUpdated(t, broker)
	currentTime = currentTime.Add(1 * time.Minute)
	endEpoch(t, engine, currentEpoch, currentTime)

	currentEpoch += 1
	startEpoch(t, engine, currentEpoch, currentTime)

	hashAfter2Epochs, _, err := engine.GetState(key)
	require.NoError(t, err)
	loadedEngine := assertSnapshotMatches(t, key, hashAfter2Epochs)
	loadedEngine.OnMarketFeeFactorsBuyBackFeeUpdate(context.Background(), num.NewDecimalFromFloat(0.5))
	loadedEngine.OnMarketFeeFactorsTreasuryFeeUpdate(context.Background(), num.NewDecimalFromFloat(0.5))

	// fraction 0.2 => rebate 0.2
	require.Equal(t, "0.2", engine.VolumeRebateFactorForParty("p8").String())
	require.Equal(t, "0.2", loadedEngine.VolumeRebateFactorForParty("p8").String())
	// fraction 0.15 => rebate 0.1
	require.Equal(t, "0.1", engine.VolumeRebateFactorForParty("p1").String())
	require.Equal(t, "0.1", loadedEngine.VolumeRebateFactorForParty("p1").String())
	// nothing this time
	require.Equal(t, "0", engine.VolumeRebateFactorForParty("p2").String())
	require.Equal(t, "0", loadedEngine.VolumeRebateFactorForParty("p2").String())
	// nothing this time
	require.Equal(t, "0", engine.VolumeRebateFactorForParty("p3").String())
	require.Equal(t, "0", loadedEngine.VolumeRebateFactorForParty("p3").String())
	// nothing this time
	require.Equal(t, "0", engine.VolumeRebateFactorForParty("p4").String())
	require.Equal(t, "0", loadedEngine.VolumeRebateFactorForParty("p4").String())
	// fraction 0.4 => rebate 1 => capped at 0.3
	require.Equal(t, "0.3", engine.VolumeRebateFactorForParty("p5").String())
	require.Equal(t, "0.3", loadedEngine.VolumeRebateFactorForParty("p5").String())
	// fraction 0.4 => rebate 1 => capped at 0.3
	require.Equal(t, "0.3", engine.VolumeRebateFactorForParty("p6").String())
	require.Equal(t, "0.3", loadedEngine.VolumeRebateFactorForParty("p6").String())
	// nothing this time
	require.Equal(t, "0", engine.VolumeRebateFactorForParty("p7").String())
	require.Equal(t, "0", loadedEngine.VolumeRebateFactorForParty("p7").String())

	marketActivityTracker.EXPECT().CalculateTotalMakerContributionInQuantum(gomock.Any()).Return(map[string]*num.Uint{}, map[string]num.Decimal{}).AnyTimes()
	expectStatsUpdated(t, broker)
	currentTime = p1.EndOfProgramTimestamp
	endEpoch(t, engine, currentEpoch, currentTime)

	expectProgramEnded(t, broker, p1)
	currentEpoch += 1
	startEpoch(t, engine, currentEpoch, currentTime)

	hashAfterProgramEnded, _, err := engine.GetState(key)
	require.NoError(t, err)
	loadedEngine = assertSnapshotMatches(t, key, hashAfterProgramEnded)

	// no rebate for terminated program
	for _, p := range []string{"p1", "p2", "p3", "p4", "p5", "p6", "p7", "p8"} {
		require.Equal(t, num.DecimalZero(), engine.VolumeRebateFactorForParty(types.PartyID(p)))
		require.Equal(t, num.DecimalZero(), loadedEngine.VolumeRebateFactorForParty(types.PartyID(p)))
	}
}
