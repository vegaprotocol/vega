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

package volumediscount_test

import (
	"bytes"
	"context"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/core/volumediscount"
	"code.vegaprotocol.io/vega/core/volumediscount/mocks"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/libs/proto"
	"code.vegaprotocol.io/vega/protos/vega"
	snapshotpb "code.vegaprotocol.io/vega/protos/vega/snapshot/v1"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func assertSnapshotMatches(t *testing.T, key string, expectedHash []byte) *volumediscount.SnapshottedEngine {
	t.Helper()

	loadCtrl := gomock.NewController(t)
	loadBroker := mocks.NewMockBroker(loadCtrl)
	loadMarketActivityTracker := mocks.NewMockMarketActivityTracker(loadCtrl)
	loadEngine := volumediscount.NewSnapshottedEngine(loadBroker, loadMarketActivityTracker)

	pl := snapshotpb.Payload{}
	require.NoError(t, proto.Unmarshal(expectedHash, &pl))

	loadEngine.LoadState(context.Background(), types.PayloadFromProto(&pl))
	loadedHashEmpty, _, err := loadEngine.GetState(key)
	require.NoError(t, err)
	require.True(t, bytes.Equal(expectedHash, loadedHashEmpty))
	return loadEngine
}

func TestVolumeDiscountProgramLifecycle(t *testing.T) {
	key := (&types.PayloadVolumeDiscountProgram{}).Key()
	ctrl := gomock.NewController(t)
	broker := mocks.NewMockBroker(ctrl)
	marketActivityTracker := mocks.NewMockMarketActivityTracker(ctrl)
	engine := volumediscount.NewSnapshottedEngine(broker, marketActivityTracker)

	// test snapshot with empty engine
	hashEmpty, _, err := engine.GetState(key)
	require.NoError(t, err)
	assertSnapshotMatches(t, key, hashEmpty)

	now := time.Now()

	p1 := &types.VolumeDiscountProgram{
		ID:                    "1",
		Version:               0,
		EndOfProgramTimestamp: now.Add(time.Hour * 1),
		WindowLength:          1,
		VolumeBenefitTiers: []*types.VolumeBenefitTier{
			{MinimumRunningNotionalTakerVolume: num.NewUint(1000), VolumeDiscountFactors: types.Factors{
				Infra:     num.DecimalFromFloat(0.1),
				Maker:     num.DecimalFromFloat(0.1),
				Liquidity: num.DecimalFromFloat(0.1),
			}},
			{MinimumRunningNotionalTakerVolume: num.NewUint(2000), VolumeDiscountFactors: types.Factors{
				Infra:     num.DecimalFromFloat(0.2),
				Maker:     num.DecimalFromFloat(0.2),
				Liquidity: num.DecimalFromFloat(0.2),
			}},
			{MinimumRunningNotionalTakerVolume: num.NewUint(3000), VolumeDiscountFactors: types.Factors{
				Infra:     num.DecimalFromFloat(0.5),
				Maker:     num.DecimalFromFloat(0.5),
				Liquidity: num.DecimalFromFloat(0.5),
			}},
			{MinimumRunningNotionalTakerVolume: num.NewUint(4000), VolumeDiscountFactors: types.Factors{
				Infra:     num.DecimalFromFloat(1),
				Maker:     num.DecimalFromFloat(1),
				Liquidity: num.DecimalFromFloat(1),
			}},
		},
	}
	// add the program
	engine.UpdateProgram(p1)

	// expect an event for the started program
	broker.EXPECT().Send(gomock.Any()).DoAndReturn(func(evt events.Event) {
		e := evt.(*events.VolumeDiscountProgramStarted)
		require.Equal(t, p1.IntoProto(), e.GetVolumeDiscountProgramStarted().Program)
	}).Times(1)

	// activate the program
	engine.OnEpoch(context.Background(), types.Epoch{Action: vega.EpochAction_EPOCH_ACTION_START, StartTime: now})

	// check snapshot with new program
	hashWithNew, _, err := engine.GetState(key)
	require.NoError(t, err)
	assertSnapshotMatches(t, key, hashWithNew)

	// add a new program
	p2 := &types.VolumeDiscountProgram{
		ID:                    "1",
		Version:               1,
		EndOfProgramTimestamp: now.Add(time.Hour * 2),
		WindowLength:          1,
		VolumeBenefitTiers: []*types.VolumeBenefitTier{
			{MinimumRunningNotionalTakerVolume: num.NewUint(2000), VolumeDiscountFactors: types.Factors{
				Maker:     num.DecimalFromFloat(0.2),
				Infra:     num.DecimalFromFloat(0.2),
				Liquidity: num.DecimalFromFloat(0.2),
			}},
			{MinimumRunningNotionalTakerVolume: num.NewUint(3000), VolumeDiscountFactors: types.Factors{
				Maker:     num.DecimalFromFloat(0.5),
				Infra:     num.DecimalFromFloat(0.5),
				Liquidity: num.DecimalFromFloat(0.5),
			}},
			{MinimumRunningNotionalTakerVolume: num.NewUint(1000), VolumeDiscountFactors: types.Factors{
				Maker:     num.DecimalFromFloat(0.1),
				Infra:     num.DecimalFromFloat(0.1),
				Liquidity: num.DecimalFromFloat(0.1),
			}},
			{MinimumRunningNotionalTakerVolume: num.NewUint(4000), VolumeDiscountFactors: types.Factors{
				Maker:     num.DecimalFromFloat(1),
				Infra:     num.DecimalFromFloat(1),
				Liquidity: num.DecimalFromFloat(1),
			}},
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
		e := evt.(*events.VolumeDiscountProgramUpdated)
		require.Equal(t, p2.IntoProto(), e.GetVolumeDiscountProgramUpdated().Program)
	}).Times(1)
	engine.OnEpoch(context.Background(), types.Epoch{Action: vega.EpochAction_EPOCH_ACTION_START, StartTime: now.Add(time.Hour * 1)})

	// // expire the program
	broker.EXPECT().Send(gomock.Any()).DoAndReturn(func(evt events.Event) {
		e := evt.(*events.VolumeDiscountProgramEnded)
		require.Equal(t, p2.Version, e.GetVolumeDiscountProgramEnded().Version)
	}).Times(1)
	engine.OnEpoch(context.Background(), types.Epoch{Action: vega.EpochAction_EPOCH_ACTION_START, StartTime: now.Add(time.Hour * 2)})

	// check snapshot with terminated program
	hashWithPostTermination, _, err := engine.GetState(key)
	require.NoError(t, err)
	assertSnapshotMatches(t, key, hashWithPostTermination)
}

func TestDiscountFactor(t *testing.T) {
	key := (&types.PayloadVolumeDiscountProgram{}).Key()
	ctrl := gomock.NewController(t)
	broker := mocks.NewMockBroker(ctrl)
	marketActivityTracker := mocks.NewMockMarketActivityTracker(ctrl)
	engine := volumediscount.NewSnapshottedEngine(broker, marketActivityTracker)

	currentTime := time.Now()

	p1 := &types.VolumeDiscountProgram{
		ID:                    "1",
		Version:               0,
		EndOfProgramTimestamp: currentTime.Add(time.Hour * 1),
		WindowLength:          1,
		VolumeBenefitTiers: []*types.VolumeBenefitTier{
			{MinimumRunningNotionalTakerVolume: num.NewUint(1000), VolumeDiscountFactors: types.Factors{
				Maker:     num.DecimalFromFloat(0.1),
				Infra:     num.DecimalFromFloat(0.1),
				Liquidity: num.DecimalFromFloat(0.1),
			}},
			{MinimumRunningNotionalTakerVolume: num.NewUint(2000), VolumeDiscountFactors: types.Factors{
				Maker:     num.DecimalFromFloat(0.2),
				Infra:     num.DecimalFromFloat(0.2),
				Liquidity: num.DecimalFromFloat(0.2),
			}},
			{MinimumRunningNotionalTakerVolume: num.NewUint(3000), VolumeDiscountFactors: types.Factors{
				Maker:     num.DecimalFromFloat(0.5),
				Infra:     num.DecimalFromFloat(0.5),
				Liquidity: num.DecimalFromFloat(0.5),
			}},
			{MinimumRunningNotionalTakerVolume: num.NewUint(4000), VolumeDiscountFactors: types.Factors{
				Maker:     num.DecimalFromFloat(1),
				Infra:     num.DecimalFromFloat(1),
				Liquidity: num.DecimalFromFloat(1),
			}},
		},
	}
	// add the program
	engine.UpdateProgram(p1)

	// activate the program
	currentEpoch := uint64(1)
	expectProgramStarted(t, broker, p1)
	startEpoch(t, engine, currentEpoch, currentTime)

	// so now we have a program active so at the end of the epoch lets return for some parties some notional
	marketActivityTracker.EXPECT().NotionalTakerVolumeForAllParties().Return(map[types.PartyID]*num.Uint{
		"p1": num.NewUint(900),
		"p2": num.NewUint(1000),
		"p3": num.NewUint(1001),
		"p4": num.NewUint(2000),
		"p5": num.NewUint(3000),
		"p6": num.NewUint(4000),
		"p7": num.NewUint(5000),
	}).Times(1)

	// end the epoch to get the market activity recorded
	expectStatsUpdatedWithUnqualifiedParties(t, broker)
	currentTime = currentTime.Add(1 * time.Minute)
	endEpoch(t, engine, currentEpoch, currentTime.Add(1*time.Minute))

	// start a new epoch for the discount factors to be in place
	currentEpoch += 1
	startEpoch(t, engine, currentEpoch, currentTime)

	// check snapshot with terminated program
	hashWithEpochNotionalsData, _, err := engine.GetState(key)
	require.NoError(t, err)
	loadedEngine := assertSnapshotMatches(t, key, hashWithEpochNotionalsData)

	// party does not exist
	require.Equal(t, num.DecimalZero(), engine.VolumeDiscountFactorForParty("p8").Infra)
	require.Equal(t, num.DecimalZero(), loadedEngine.VolumeDiscountFactorForParty("p8").Infra)
	// party is not eligible
	require.Equal(t, num.DecimalZero(), engine.VolumeDiscountFactorForParty("p1").Infra)
	require.Equal(t, num.DecimalZero(), loadedEngine.VolumeDiscountFactorForParty("p1").Infra)
	// volume between 1000/2000
	require.Equal(t, "0.1", engine.VolumeDiscountFactorForParty("p2").Infra.String())
	require.Equal(t, "0.1", loadedEngine.VolumeDiscountFactorForParty("p2").Infra.String())
	require.Equal(t, "0.1", engine.VolumeDiscountFactorForParty("p3").Infra.String())
	require.Equal(t, "0.1", loadedEngine.VolumeDiscountFactorForParty("p3").Infra.String())

	// volume 2000<=x<3000
	require.Equal(t, "0.2", engine.VolumeDiscountFactorForParty("p4").Infra.String())
	require.Equal(t, "0.2", loadedEngine.VolumeDiscountFactorForParty("p4").Infra.String())

	// volume 3000<=x<4000
	require.Equal(t, "0.5", engine.VolumeDiscountFactorForParty("p5").Infra.String())
	require.Equal(t, "0.5", loadedEngine.VolumeDiscountFactorForParty("p5").Infra.String())

	// volume >=4000
	require.Equal(t, "1", engine.VolumeDiscountFactorForParty("p6").Infra.String())
	require.Equal(t, "1", loadedEngine.VolumeDiscountFactorForParty("p6").Infra.String())
	require.Equal(t, "1", engine.VolumeDiscountFactorForParty("p7").Infra.String())
	require.Equal(t, "1", loadedEngine.VolumeDiscountFactorForParty("p7").Infra.String())

	marketActivityTracker.EXPECT().NotionalTakerVolumeForAllParties().Return(map[types.PartyID]*num.Uint{}).Times(1)

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

	// no discount for terminated program
	for _, p := range []string{"p1", "p2", "p3", "p4", "p5", "p6", "p7", "p8"} {
		require.Equal(t, num.DecimalZero(), engine.VolumeDiscountFactorForParty(types.PartyID(p)).Infra)
		require.Equal(t, num.DecimalZero(), loadedEngine.VolumeDiscountFactorForParty(types.PartyID(p)).Infra)
	}
}

func TestDiscountFactorWithWindow(t *testing.T) {
	key := (&types.PayloadVolumeDiscountProgram{}).Key()
	ctrl := gomock.NewController(t)
	broker := mocks.NewMockBroker(ctrl)
	marketActivityTracker := mocks.NewMockMarketActivityTracker(ctrl)
	engine := volumediscount.NewSnapshottedEngine(broker, marketActivityTracker)

	currentTime := time.Now()

	p1 := &types.VolumeDiscountProgram{
		ID:                    "1",
		Version:               0,
		EndOfProgramTimestamp: currentTime.Add(time.Hour * 1),
		WindowLength:          2,
		VolumeBenefitTiers: []*types.VolumeBenefitTier{
			{MinimumRunningNotionalTakerVolume: num.NewUint(1000), VolumeDiscountFactors: types.Factors{
				Maker:     num.DecimalFromFloat(0.1),
				Infra:     num.DecimalFromFloat(0.1),
				Liquidity: num.DecimalFromFloat(0.1),
			}},
			{MinimumRunningNotionalTakerVolume: num.NewUint(2000), VolumeDiscountFactors: types.Factors{
				Maker:     num.DecimalFromFloat(0.2),
				Infra:     num.DecimalFromFloat(0.2),
				Liquidity: num.DecimalFromFloat(0.2),
			}},
			{MinimumRunningNotionalTakerVolume: num.NewUint(3000), VolumeDiscountFactors: types.Factors{
				Maker:     num.DecimalFromFloat(0.5),
				Infra:     num.DecimalFromFloat(0.5),
				Liquidity: num.DecimalFromFloat(0.5),
			}},
			{MinimumRunningNotionalTakerVolume: num.NewUint(4000), VolumeDiscountFactors: types.Factors{
				Maker:     num.DecimalFromFloat(1),
				Infra:     num.DecimalFromFloat(1),
				Liquidity: num.DecimalFromFloat(1),
			}},
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
	marketActivityTracker.EXPECT().NotionalTakerVolumeForAllParties().Return(map[types.PartyID]*num.Uint{
		"p1": num.NewUint(900),
		"p2": num.NewUint(1000),
		"p3": num.NewUint(1001),
		"p4": num.NewUint(2000),
		"p5": num.NewUint(3000),
		"p6": num.NewUint(4000),
		"p7": num.NewUint(5000),
	}).Times(1)

	expectStatsUpdatedWithUnqualifiedParties(t, broker)
	currentTime = currentTime.Add(1 * time.Minute)
	endEpoch(t, engine, currentEpoch, currentTime)
	// start a new epoch for the discount factors to be in place

	// party does not exist
	require.Equal(t, num.DecimalZero(), engine.VolumeDiscountFactorForParty("p8").Infra)
	// volume 900
	require.Equal(t, num.DecimalZero(), engine.VolumeDiscountFactorForParty("p1").Infra)
	// volume 1000
	require.Equal(t, "0.1", engine.VolumeDiscountFactorForParty("p2").Infra.String())
	// volume 1001
	require.Equal(t, "0.1", engine.VolumeDiscountFactorForParty("p3").Infra.String())
	// volume 2000
	require.Equal(t, "0.2", engine.VolumeDiscountFactorForParty("p4").Infra.String())
	// volume 3000
	require.Equal(t, "0.5", engine.VolumeDiscountFactorForParty("p5").Infra.String())
	// volume 4000
	require.Equal(t, "1", engine.VolumeDiscountFactorForParty("p6").Infra.String())
	// volume 5000
	require.Equal(t, "1", engine.VolumeDiscountFactorForParty("p7").Infra.String())

	// running for another epoch
	marketActivityTracker.EXPECT().NotionalTakerVolumeForAllParties().Return(map[types.PartyID]*num.Uint{
		"p8": num.NewUint(2000),
		"p1": num.NewUint(1500),
		"p5": num.NewUint(4000),
		"p6": num.NewUint(4000),
	}).Times(1)

	expectStatsUpdated(t, broker)
	currentTime = currentTime.Add(1 * time.Minute)
	endEpoch(t, engine, currentEpoch, currentTime)

	currentEpoch += 1
	startEpoch(t, engine, currentEpoch, currentTime)

	hashAfter2Epochs, _, err := engine.GetState(key)
	require.NoError(t, err)
	loadedEngine := assertSnapshotMatches(t, key, hashAfter2Epochs)

	// now p8 exists and the volume is 2000
	require.Equal(t, "0.2", engine.VolumeDiscountFactorForParty("p8").Infra.String())
	require.Equal(t, "0.2", loadedEngine.VolumeDiscountFactorForParty("p8").Infra.String())
	// volume 2400
	require.Equal(t, "0.2", engine.VolumeDiscountFactorForParty("p1").Infra.String())
	require.Equal(t, "0.2", loadedEngine.VolumeDiscountFactorForParty("p1").Infra.String())
	// volume 1000
	require.Equal(t, "0.1", engine.VolumeDiscountFactorForParty("p2").Infra.String())
	require.Equal(t, "0.1", loadedEngine.VolumeDiscountFactorForParty("p2").Infra.String())
	// volume 1001
	require.Equal(t, "0.1", engine.VolumeDiscountFactorForParty("p3").Infra.String())
	require.Equal(t, "0.1", loadedEngine.VolumeDiscountFactorForParty("p3").Infra.String())
	// volume 2000
	require.Equal(t, "0.2", engine.VolumeDiscountFactorForParty("p4").Infra.String())
	require.Equal(t, "0.2", loadedEngine.VolumeDiscountFactorForParty("p4").Infra.String())
	// volume 7000
	require.Equal(t, "1", engine.VolumeDiscountFactorForParty("p5").Infra.String())
	require.Equal(t, "1", loadedEngine.VolumeDiscountFactorForParty("p5").Infra.String())
	// volume 8000
	require.Equal(t, "1", engine.VolumeDiscountFactorForParty("p6").Infra.String())
	require.Equal(t, "1", loadedEngine.VolumeDiscountFactorForParty("p6").Infra.String())
	// volume 5000
	require.Equal(t, "1", engine.VolumeDiscountFactorForParty("p7").Infra.String())
	require.Equal(t, "1", loadedEngine.VolumeDiscountFactorForParty("p7").Infra.String())

	marketActivityTracker.EXPECT().NotionalTakerVolumeForAllParties().Return(map[types.PartyID]*num.Uint{}).Times(1)

	expectStatsUpdated(t, broker)
	currentTime = p1.EndOfProgramTimestamp
	endEpoch(t, engine, currentEpoch, currentTime)

	expectProgramEnded(t, broker, p1)
	currentEpoch += 1
	startEpoch(t, engine, currentEpoch, currentTime)

	hashAfterProgramEnded, _, err := engine.GetState(key)
	require.NoError(t, err)
	loadedEngine = assertSnapshotMatches(t, key, hashAfterProgramEnded)

	// no discount for terminated program
	for _, p := range []string{"p1", "p2", "p3", "p4", "p5", "p6", "p7", "p8"} {
		require.Equal(t, num.DecimalZero(), engine.VolumeDiscountFactorForParty(types.PartyID(p)).Infra)
		require.Equal(t, num.DecimalZero(), loadedEngine.VolumeDiscountFactorForParty(types.PartyID(p)).Infra)
	}
}
