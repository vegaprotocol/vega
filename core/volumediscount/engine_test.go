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
			{MinimumRunningNotionalTakerVolume: num.NewUint(1000), VolumeDiscountFactor: num.DecimalFromFloat(0.1)},
			{MinimumRunningNotionalTakerVolume: num.NewUint(2000), VolumeDiscountFactor: num.DecimalFromFloat(0.2)},
			{MinimumRunningNotionalTakerVolume: num.NewUint(3000), VolumeDiscountFactor: num.DecimalFromFloat(0.5)},
			{MinimumRunningNotionalTakerVolume: num.NewUint(4000), VolumeDiscountFactor: num.DecimalFromFloat(1)},
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
			{MinimumRunningNotionalTakerVolume: num.NewUint(2000), VolumeDiscountFactor: num.DecimalFromFloat(0.2)},
			{MinimumRunningNotionalTakerVolume: num.NewUint(3000), VolumeDiscountFactor: num.DecimalFromFloat(0.5)},
			{MinimumRunningNotionalTakerVolume: num.NewUint(1000), VolumeDiscountFactor: num.DecimalFromFloat(0.1)},
			{MinimumRunningNotionalTakerVolume: num.NewUint(4000), VolumeDiscountFactor: num.DecimalFromFloat(1)},
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

	now := time.Now()

	p1 := &types.VolumeDiscountProgram{
		ID:                    "1",
		Version:               0,
		EndOfProgramTimestamp: now.Add(time.Hour * 1),
		WindowLength:          1,
		VolumeBenefitTiers: []*types.VolumeBenefitTier{
			{MinimumRunningNotionalTakerVolume: num.NewUint(1000), VolumeDiscountFactor: num.DecimalFromFloat(0.1)},
			{MinimumRunningNotionalTakerVolume: num.NewUint(2000), VolumeDiscountFactor: num.DecimalFromFloat(0.2)},
			{MinimumRunningNotionalTakerVolume: num.NewUint(3000), VolumeDiscountFactor: num.DecimalFromFloat(0.5)},
			{MinimumRunningNotionalTakerVolume: num.NewUint(4000), VolumeDiscountFactor: num.DecimalFromFloat(1)},
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

	// so now we have a program active so at the end of the epoch lets return for some parties some notional
	marketActivityTracker.EXPECT().NotionalTakerVolumeForAllParties().Return(map[types.PartyID]*num.Uint{
		types.PartyID("p1"): num.NewUint(900),
		types.PartyID("p2"): num.NewUint(1000),
		types.PartyID("p3"): num.NewUint(1001),
		types.PartyID("p4"): num.NewUint(2000),
		types.PartyID("p5"): num.NewUint(3000),
		types.PartyID("p6"): num.NewUint(4000),
		types.PartyID("p7"): num.NewUint(5000),
	}).Times(1)

	// end the epoch to get the market activity recorded
	engine.OnEpoch(context.Background(), types.Epoch{Action: vega.EpochAction_EPOCH_ACTION_END})
	// start a new epoch for the discount factors to be in place
	engine.OnEpoch(context.Background(), types.Epoch{Action: vega.EpochAction_EPOCH_ACTION_START, StartTime: now.Add(time.Minute * 1)})

	// check snapshot with terminated program
	hashWithEpochNotionalsData, _, err := engine.GetState(key)
	require.NoError(t, err)
	loadedEngine := assertSnapshotMatches(t, key, hashWithEpochNotionalsData)

	// party does not exist
	require.Equal(t, num.DecimalZero(), engine.VolumeDiscountFactorForParty("p8"))
	require.Equal(t, num.DecimalZero(), loadedEngine.VolumeDiscountFactorForParty("p8"))
	// party is not eligible
	require.Equal(t, num.DecimalZero(), engine.VolumeDiscountFactorForParty("p1"))
	require.Equal(t, num.DecimalZero(), loadedEngine.VolumeDiscountFactorForParty("p1"))
	// volume between 1000/2000
	require.Equal(t, "0.1", engine.VolumeDiscountFactorForParty("p2").String())
	require.Equal(t, "0.1", loadedEngine.VolumeDiscountFactorForParty("p2").String())
	require.Equal(t, "0.1", engine.VolumeDiscountFactorForParty("p3").String())
	require.Equal(t, "0.1", loadedEngine.VolumeDiscountFactorForParty("p3").String())

	// volume 2000<=x<3000
	require.Equal(t, "0.2", engine.VolumeDiscountFactorForParty("p4").String())
	require.Equal(t, "0.2", loadedEngine.VolumeDiscountFactorForParty("p4").String())

	// volume 3000<=x<4000
	require.Equal(t, "0.5", engine.VolumeDiscountFactorForParty("p5").String())
	require.Equal(t, "0.5", loadedEngine.VolumeDiscountFactorForParty("p5").String())

	// volume >=4000
	require.Equal(t, "1", engine.VolumeDiscountFactorForParty("p6").String())
	require.Equal(t, "1", loadedEngine.VolumeDiscountFactorForParty("p6").String())
	require.Equal(t, "1", engine.VolumeDiscountFactorForParty("p7").String())
	require.Equal(t, "1", loadedEngine.VolumeDiscountFactorForParty("p7").String())

	// terminate the program
	broker.EXPECT().Send(gomock.Any()).DoAndReturn(func(evt events.Event) {
		e := evt.(*events.VolumeDiscountProgramEnded)
		require.Equal(t, p1.Version, e.GetVolumeDiscountProgramEnded().Version)
	}).Times(1)
	engine.OnEpoch(context.Background(), types.Epoch{Action: vega.EpochAction_EPOCH_ACTION_START, StartTime: now.Add(time.Hour * 1)})

	hashAfterProgramEnded, _, err := engine.GetState(key)
	require.NoError(t, err)
	loadedEngine = assertSnapshotMatches(t, key, hashAfterProgramEnded)

	// no discount for terminated program
	for _, p := range []string{"p1", "p2", "p3", "p4", "p5", "p6", "p7", "p8"} {
		require.Equal(t, num.DecimalZero(), engine.VolumeDiscountFactorForParty(types.PartyID(p)))
		require.Equal(t, num.DecimalZero(), loadedEngine.VolumeDiscountFactorForParty(types.PartyID(p)))
	}
}

func TestDiscountFactorWithWindow(t *testing.T) {
	key := (&types.PayloadVolumeDiscountProgram{}).Key()
	ctrl := gomock.NewController(t)
	broker := mocks.NewMockBroker(ctrl)
	marketActivityTracker := mocks.NewMockMarketActivityTracker(ctrl)
	engine := volumediscount.NewSnapshottedEngine(broker, marketActivityTracker)

	now := time.Now()

	p1 := &types.VolumeDiscountProgram{
		ID:                    "1",
		Version:               0,
		EndOfProgramTimestamp: now.Add(time.Hour * 1),
		WindowLength:          2,
		VolumeBenefitTiers: []*types.VolumeBenefitTier{
			{MinimumRunningNotionalTakerVolume: num.NewUint(1000), VolumeDiscountFactor: num.DecimalFromFloat(0.1)},
			{MinimumRunningNotionalTakerVolume: num.NewUint(2000), VolumeDiscountFactor: num.DecimalFromFloat(0.2)},
			{MinimumRunningNotionalTakerVolume: num.NewUint(3000), VolumeDiscountFactor: num.DecimalFromFloat(0.5)},
			{MinimumRunningNotionalTakerVolume: num.NewUint(4000), VolumeDiscountFactor: num.DecimalFromFloat(1)},
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

	// so now we have a program active so at the end of the epoch lets return for some parties some notional
	marketActivityTracker.EXPECT().NotionalTakerVolumeForAllParties().Return(map[types.PartyID]*num.Uint{
		types.PartyID("p1"): num.NewUint(900),
		types.PartyID("p2"): num.NewUint(1000),
		types.PartyID("p3"): num.NewUint(1001),
		types.PartyID("p4"): num.NewUint(2000),
		types.PartyID("p5"): num.NewUint(3000),
		types.PartyID("p6"): num.NewUint(4000),
		types.PartyID("p7"): num.NewUint(5000),
	}).Times(1)

	// end the epoch to get the market activity recorded
	engine.OnEpoch(context.Background(), types.Epoch{Action: vega.EpochAction_EPOCH_ACTION_END})
	// start a new epoch for the discount factors to be in place
	engine.OnEpoch(context.Background(), types.Epoch{Action: vega.EpochAction_EPOCH_ACTION_START, StartTime: now.Add(time.Minute * 1)})

	// party does not exist
	require.Equal(t, num.DecimalZero(), engine.VolumeDiscountFactorForParty("p8"))
	// party is not eligible
	require.Equal(t, num.DecimalZero(), engine.VolumeDiscountFactorForParty("p1"))
	// over a window of 2 party2 has 500
	require.Equal(t, num.DecimalZero(), engine.VolumeDiscountFactorForParty("p2"))
	// over a window of 2 party2 has 500.5
	require.Equal(t, num.DecimalZero(), engine.VolumeDiscountFactorForParty("p3"))
	// average volume 1000
	require.Equal(t, "0.1", engine.VolumeDiscountFactorForParty("p4").String())
	// average volume 1500
	require.Equal(t, "0.1", engine.VolumeDiscountFactorForParty("p5").String())
	// average volume 2000
	require.Equal(t, "0.2", engine.VolumeDiscountFactorForParty("p6").String())
	// average volume 2500
	require.Equal(t, "0.2", engine.VolumeDiscountFactorForParty("p7").String())

	// running for another epoch
	marketActivityTracker.EXPECT().NotionalTakerVolumeForAllParties().Return(map[types.PartyID]*num.Uint{
		types.PartyID("p8"): num.NewUint(2000),
		types.PartyID("p1"): num.NewUint(1500),
		types.PartyID("p5"): num.NewUint(4000),
		types.PartyID("p6"): num.NewUint(4000),
	}).Times(1)
	engine.OnEpoch(context.Background(), types.Epoch{Action: vega.EpochAction_EPOCH_ACTION_END})
	engine.OnEpoch(context.Background(), types.Epoch{Action: vega.EpochAction_EPOCH_ACTION_START, StartTime: now.Add(time.Minute * 2)})

	hashAfter2Epochs, _, err := engine.GetState(key)
	require.NoError(t, err)
	loadedEngine := assertSnapshotMatches(t, key, hashAfter2Epochs)

	// now p8 exists and the average notional is 1000
	require.Equal(t, "0.1", engine.VolumeDiscountFactorForParty("p8").String())
	require.Equal(t, "0.1", loadedEngine.VolumeDiscountFactorForParty("p8").String())
	// party1 now has a total of 2000 with average of 1000 they're not eligible
	require.Equal(t, "0.1", engine.VolumeDiscountFactorForParty("p1").String())
	require.Equal(t, "0.1", loadedEngine.VolumeDiscountFactorForParty("p1").String())
	// over a window of 2 party2 has 500 so not eligible
	require.Equal(t, num.DecimalZero(), engine.VolumeDiscountFactorForParty("p2"))
	require.Equal(t, num.DecimalZero(), loadedEngine.VolumeDiscountFactorForParty("p2"))
	// over a window of 2 party2 has 500.5 so not eligible
	require.Equal(t, num.DecimalZero(), engine.VolumeDiscountFactorForParty("p3"))
	require.Equal(t, num.DecimalZero(), loadedEngine.VolumeDiscountFactorForParty("p3"))
	// average volume 1000
	require.Equal(t, "0.1", engine.VolumeDiscountFactorForParty("p4").String())
	require.Equal(t, "0.1", loadedEngine.VolumeDiscountFactorForParty("p4").String())
	// average volume 3500
	require.Equal(t, "0.5", engine.VolumeDiscountFactorForParty("p5").String())
	require.Equal(t, "0.5", loadedEngine.VolumeDiscountFactorForParty("p5").String())
	// average volume 4000
	require.Equal(t, "1", engine.VolumeDiscountFactorForParty("p6").String())
	require.Equal(t, "1", loadedEngine.VolumeDiscountFactorForParty("p6").String())
	// average volume 2500
	require.Equal(t, "0.2", engine.VolumeDiscountFactorForParty("p7").String())
	require.Equal(t, "0.2", loadedEngine.VolumeDiscountFactorForParty("p7").String())

	broker.EXPECT().Send(gomock.Any()).DoAndReturn(func(evt events.Event) {
		e := evt.(*events.VolumeDiscountProgramEnded)
		require.Equal(t, p1.Version, e.GetVolumeDiscountProgramEnded().Version)
	}).Times(1)
	engine.OnEpoch(context.Background(), types.Epoch{Action: vega.EpochAction_EPOCH_ACTION_START, StartTime: now.Add(time.Hour * 1)})

	hashAfterProgramEnded, _, err := engine.GetState(key)
	require.NoError(t, err)
	loadedEngine = assertSnapshotMatches(t, key, hashAfterProgramEnded)

	// no discount for terminated program
	for _, p := range []string{"p1", "p2", "p3", "p4", "p5", "p6", "p7", "p8"} {
		require.Equal(t, num.DecimalZero(), engine.VolumeDiscountFactorForParty(types.PartyID(p)))
		require.Equal(t, num.DecimalZero(), loadedEngine.VolumeDiscountFactorForParty(types.PartyID(p)))
	}
}
