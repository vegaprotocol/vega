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

package common

import (
	"context"
	"testing"
	"time"

	bmocks "code.vegaprotocol.io/vega/core/broker/mocks"
	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/execution/common/mocks"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/logging"
	vgproto "code.vegaprotocol.io/vega/protos/vega"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func TestPublishGameMetricAverageNotional(t *testing.T) {
	ctx := context.Background()
	epochService := &DummyEpochEngine{}
	ctrl := gomock.NewController(t)
	teams := mocks.NewMockTeams(ctrl)
	balanceChecker := mocks.NewMockAccountBalanceChecker(ctrl)
	broker := bmocks.NewMockBroker(ctrl)
	gameScoreEvents := []events.Event{}

	broker.EXPECT().SendBatch(gomock.Any()).AnyTimes()
	broker.EXPECT().Send(gomock.Any()).Do(func(evt events.Event) {
		if evt.StreamMessage().GetGameScores() != nil {
			gameScoreEvents = append(gameScoreEvents, evt)
		}
	}).AnyTimes()
	tracker := NewMarketActivityTracker(logging.NewTestLogger(), teams, balanceChecker, broker, DummyCollateralEngine{})
	epochService.NotifyOnEpoch(tracker.OnEpochEvent, tracker.OnEpochRestore)
	tracker.SetEligibilityChecker(&DummyEligibilityChecker{})
	epochService.target(context.Background(), types.Epoch{Seq: 1, Action: vgproto.EpochAction_EPOCH_ACTION_START, StartTime: time.Time{}})

	// add some markets for 2 different assets
	tracker.MarketProposed("a1", "m1", "z1")
	tracker.MarketProposed("a1", "m2", "z2")
	tracker.MarketProposed("a1", "m3", "z3")

	// add some markets for 2 different assets
	tracker.MarketProposed("a1", "m1", "z1")
	tracker.MarketProposed("a1", "m2", "z2")
	tracker.MarketProposed("a1", "m3", "z3")

	// record some values for all metrics
	tracker.RecordPosition("a1", "p1", "m1", 10, num.NewUint(1), num.DecimalOne(), time.Unix(5, 0))
	tracker.RecordPosition("a1", "p1", "m2", 20, num.NewUint(2), num.DecimalOne(), time.Unix(20, 0))
	tracker.RecordPosition("a1", "p1", "m3", 30, num.NewUint(3), num.DecimalOne(), time.Unix(30, 0))
	tracker.RecordPosition("a1", "p2", "m1", 100, num.NewUint(10), num.DecimalOne(), time.Unix(15, 0))
	tracker.RecordPosition("a1", "p2", "m2", 200, num.NewUint(20), num.DecimalOne(), time.Unix(25, 0))
	tracker.RecordPosition("a1", "p2", "m3", 300, num.NewUint(30), num.DecimalOne(), time.Unix(45, 0))

	// end epoch1
	epochService.target(context.Background(), types.Epoch{Seq: 1, Action: vgproto.EpochAction_EPOCH_ACTION_END, StartTime: time.Unix(0, 0), EndTime: time.Unix(60, 0)})

	// get metrics for market m1 with window size=1
	balanceChecker.EXPECT().GetAvailableBalance(gomock.Any()).Return(num.NewUint(0), nil).AnyTimes()

	ds1 := &vgproto.DispatchStrategy{
		AssetForMetric:       "a1",
		Metric:               vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_NOTIONAL,
		Markets:              []string{"m1"},
		EntityScope:          vgproto.EntityScope_ENTITY_SCOPE_INDIVIDUALS,
		IndividualScope:      vgproto.IndividualScope_INDIVIDUAL_SCOPE_ALL,
		WindowLength:         1,
		DistributionStrategy: vgproto.DistributionStrategy_DISTRIBUTION_STRATEGY_PRO_RATA,
	}
	ds2 := &vgproto.DispatchStrategy{
		AssetForMetric:       "a1",
		Metric:               vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_NOTIONAL,
		Markets:              []string{"m2"},
		EntityScope:          vgproto.EntityScope_ENTITY_SCOPE_INDIVIDUALS,
		IndividualScope:      vgproto.IndividualScope_INDIVIDUAL_SCOPE_ALL,
		WindowLength:         1,
		DistributionStrategy: vgproto.DistributionStrategy_DISTRIBUTION_STRATEGY_PRO_RATA,
	}
	ds3 := &vgproto.DispatchStrategy{
		AssetForMetric:       "a1",
		Metric:               vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_NOTIONAL,
		Markets:              []string{"m3"},
		EntityScope:          vgproto.EntityScope_ENTITY_SCOPE_INDIVIDUALS,
		IndividualScope:      vgproto.IndividualScope_INDIVIDUAL_SCOPE_ALL,
		WindowLength:         1,
		DistributionStrategy: vgproto.DistributionStrategy_DISTRIBUTION_STRATEGY_PRO_RATA,
	}
	ds4 := &vgproto.DispatchStrategy{
		AssetForMetric:       "a1",
		Metric:               vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_NOTIONAL,
		Markets:              []string{"m1", "m2"},
		EntityScope:          vgproto.EntityScope_ENTITY_SCOPE_INDIVIDUALS,
		IndividualScope:      vgproto.IndividualScope_INDIVIDUAL_SCOPE_ALL,
		WindowLength:         1,
		DistributionStrategy: vgproto.DistributionStrategy_DISTRIBUTION_STRATEGY_PRO_RATA,
	}
	ds5 := &vgproto.DispatchStrategy{
		AssetForMetric:       "a1",
		Metric:               vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_NOTIONAL,
		EntityScope:          vgproto.EntityScope_ENTITY_SCOPE_INDIVIDUALS,
		IndividualScope:      vgproto.IndividualScope_INDIVIDUAL_SCOPE_ALL,
		WindowLength:         1,
		DistributionStrategy: vgproto.DistributionStrategy_DISTRIBUTION_STRATEGY_PRO_RATA,
	}

	// calculate intermediate scores
	tracker.PublishGameMetric(ctx, []*vgproto.DispatchStrategy{ds1, ds2, ds3, ds4, ds5}, time.Unix(60, 0))
	require.Equal(t, 5, len(gameScoreEvents))
	ps1 := gameScoreEvents[0].StreamMessage().GetGameScores().PartyScores
	require.Equal(t, "p1", ps1[0].Party)
	require.Equal(t, "p2", ps1[1].Party)
	require.Equal(t, "0.0000009", ps1[0].Score)
	require.Equal(t, "0.000075", ps1[1].Score)

	ps2 := gameScoreEvents[1].StreamMessage().GetGameScores().PartyScores
	require.Equal(t, "p1", ps2[0].Party)
	require.Equal(t, "p2", ps2[1].Party)
	require.Equal(t, "0.0000026", ps2[0].Score)
	require.Equal(t, "0.0002333", ps2[1].Score)

	ps3 := gameScoreEvents[2].StreamMessage().GetGameScores().PartyScores
	require.Equal(t, "p1", ps3[0].Party)
	require.Equal(t, "p2", ps3[1].Party)
	require.Equal(t, "0.0000045", ps3[0].Score)
	require.Equal(t, "0.000225", ps3[1].Score)

	ps4 := gameScoreEvents[3].StreamMessage().GetGameScores().PartyScores
	require.Equal(t, "p1", ps4[0].Party)
	require.Equal(t, "p2", ps4[1].Party)
	require.Equal(t, "0.0000035", ps4[0].Score)
	require.Equal(t, "0.0003083", ps4[1].Score)

	ps5 := gameScoreEvents[4].StreamMessage().GetGameScores().PartyScores
	require.Equal(t, "p1", ps5[0].Party)
	require.Equal(t, "p2", ps5[1].Party)
	require.Equal(t, "0.000008", ps5[0].Score)
	require.Equal(t, "0.0005333", ps5[1].Score)

	// now we end the epoch and make sure that we get the exact same results
	gameScoreEvents = []events.Event{}
	epochService.target(context.Background(), types.Epoch{Seq: 1, Action: vgproto.EpochAction_EPOCH_ACTION_END, StartTime: time.Unix(0, 0), EndTime: time.Unix(60, 0)})

	// we expect that if we take a snapshot of scores now, it looks identical because we didn't change the time
	tracker.PublishGameMetric(ctx, []*vgproto.DispatchStrategy{ds1, ds2, ds3, ds4, ds5}, time.Unix(60, 0))
	require.Equal(t, 5, len(gameScoreEvents))
	ps1 = gameScoreEvents[0].StreamMessage().GetGameScores().PartyScores
	require.Equal(t, "p1", ps1[0].Party)
	require.Equal(t, "p2", ps1[1].Party)
	require.Equal(t, "0.0000009", ps1[0].Score)
	require.Equal(t, "0.000075", ps1[1].Score)

	ps2 = gameScoreEvents[1].StreamMessage().GetGameScores().PartyScores
	require.Equal(t, "p1", ps2[0].Party)
	require.Equal(t, "p2", ps2[1].Party)
	require.Equal(t, "0.0000026", ps2[0].Score)
	require.Equal(t, "0.0002333", ps2[1].Score)

	ps3 = gameScoreEvents[2].StreamMessage().GetGameScores().PartyScores
	require.Equal(t, "p1", ps3[0].Party)
	require.Equal(t, "p2", ps3[1].Party)
	require.Equal(t, "0.0000045", ps3[0].Score)
	require.Equal(t, "0.000225", ps3[1].Score)

	ps4 = gameScoreEvents[3].StreamMessage().GetGameScores().PartyScores
	require.Equal(t, "p1", ps4[0].Party)
	require.Equal(t, "p2", ps4[1].Party)
	require.Equal(t, "0.0000035", ps4[0].Score)
	require.Equal(t, "0.0003083", ps4[1].Score)

	ps5 = gameScoreEvents[4].StreamMessage().GetGameScores().PartyScores
	require.Equal(t, "p1", ps5[0].Party)
	require.Equal(t, "p2", ps5[1].Party)
	require.Equal(t, "0.000008", ps5[0].Score)
	require.Equal(t, "0.0005333", ps5[1].Score)

	// start epoch 2 and record some activity
	epochService.target(context.Background(), types.Epoch{Seq: 2, Action: vgproto.EpochAction_EPOCH_ACTION_START, StartTime: time.Unix(60, 0)})
	tracker.RecordPosition("a1", "p1", "m1", 20, num.NewUint(5), num.DecimalOne(), time.Unix(90, 0))
	tracker.RecordPosition("a1", "p2", "m2", 10, num.NewUint(10), num.DecimalOne(), time.Unix(75, 0))
	tracker.RecordPosition("a2", "p1", "m3", 20, num.NewUint(5), num.DecimalOne(), time.Unix(90, 0))
	tracker.RecordPosition("a2", "p2", "m2", 10, num.NewUint(10), num.DecimalOne(), time.Unix(75, 0))

	gameScoreEvents = []events.Event{}

	// lets look at the events when the window size is 1
	tracker.PublishGameMetric(ctx, []*vgproto.DispatchStrategy{ds1, ds2, ds3, ds4, ds5}, time.Unix(120, 0))
	require.Equal(t, 5, len(gameScoreEvents))
	ps1 = gameScoreEvents[0].StreamMessage().GetGameScores().PartyScores
	require.Equal(t, "p1", ps1[0].Party)
	require.Equal(t, "p2", ps1[1].Party)
	require.Equal(t, "0.0000055", ps1[0].Score)
	require.Equal(t, "0.0001", ps1[1].Score)

	ps2 = gameScoreEvents[1].StreamMessage().GetGameScores().PartyScores
	require.Equal(t, "p1", ps2[0].Party)
	require.Equal(t, "p2", ps2[1].Party)
	require.Equal(t, "0.000004", ps2[0].Score)
	require.Equal(t, "0.0001075", ps2[1].Score)

	ps3 = gameScoreEvents[2].StreamMessage().GetGameScores().PartyScores
	require.Equal(t, "p1", ps3[0].Party)
	require.Equal(t, "p2", ps3[1].Party)
	require.Equal(t, "0.000009", ps3[0].Score)
	require.Equal(t, "0.0009", ps3[1].Score)

	ps4 = gameScoreEvents[3].StreamMessage().GetGameScores().PartyScores
	require.Equal(t, "p1", ps4[0].Party)
	require.Equal(t, "p2", ps4[1].Party)
	require.Equal(t, "0.0000095", ps4[0].Score)
	require.Equal(t, "0.0002075", ps4[1].Score)

	ps5 = gameScoreEvents[4].StreamMessage().GetGameScores().PartyScores
	require.Equal(t, "p1", ps5[0].Party)
	require.Equal(t, "p2", ps5[1].Party)
	require.Equal(t, "0.0000185", ps5[0].Score)
	require.Equal(t, "0.0011075", ps5[1].Score)

	// now lets change the window to 2:
	ds1.WindowLength = 2
	ds2.WindowLength = 2
	ds3.WindowLength = 2
	ds4.WindowLength = 2
	ds5.WindowLength = 2

	gameScoreEvents = []events.Event{}

	tracker.PublishGameMetric(ctx, []*vgproto.DispatchStrategy{ds1, ds2, ds3, ds4, ds5}, time.Unix(120, 0))
	require.Equal(t, 5, len(gameScoreEvents))
	ps1 = gameScoreEvents[0].StreamMessage().GetGameScores().PartyScores
	require.Equal(t, "p1", ps1[0].Party)
	require.Equal(t, "p2", ps1[1].Party)
	require.Equal(t, "0.0000032", ps1[0].Score)
	require.Equal(t, "0.0000875", ps1[1].Score)

	ps2 = gameScoreEvents[1].StreamMessage().GetGameScores().PartyScores
	require.Equal(t, "p1", ps2[0].Party)
	require.Equal(t, "p2", ps2[1].Party)
	require.Equal(t, "0.0000033", ps2[0].Score)
	require.Equal(t, "0.0001704", ps2[1].Score)

	ps3 = gameScoreEvents[2].StreamMessage().GetGameScores().PartyScores
	require.Equal(t, "p1", ps3[0].Party)
	require.Equal(t, "p2", ps3[1].Party)
	require.Equal(t, "0.00000675", ps3[0].Score)
	require.Equal(t, "0.0005625", ps3[1].Score)

	ps4 = gameScoreEvents[3].StreamMessage().GetGameScores().PartyScores
	require.Equal(t, "p1", ps4[0].Party)
	require.Equal(t, "p2", ps4[1].Party)
	require.Equal(t, "0.0000065", ps4[0].Score)
	require.Equal(t, "0.0002579", ps4[1].Score)

	ps5 = gameScoreEvents[4].StreamMessage().GetGameScores().PartyScores
	require.Equal(t, "p1", ps5[0].Party)
	require.Equal(t, "p2", ps5[1].Party)
	require.Equal(t, "0.00001325", ps5[0].Score)
	require.Equal(t, "0.0008204", ps5[1].Score)
}

func TestPublishGameMetricReturnVolatility(t *testing.T) {
	ctx := context.Background()
	epochService := &DummyEpochEngine{}
	ctrl := gomock.NewController(t)
	teams := mocks.NewMockTeams(ctrl)
	balanceChecker := mocks.NewMockAccountBalanceChecker(ctrl)
	broker := bmocks.NewMockBroker(ctrl)
	gameScoreEvents := []events.Event{}
	broker.EXPECT().SendBatch(gomock.Any()).AnyTimes()
	broker.EXPECT().Send(gomock.Any()).Do(func(evt events.Event) {
		if evt.StreamMessage().GetGameScores() != nil {
			gameScoreEvents = append(gameScoreEvents, evt)
		}
	}).AnyTimes()
	tracker := NewMarketActivityTracker(logging.NewTestLogger(), teams, balanceChecker, broker, DummyCollateralEngine{})
	epochService.NotifyOnEpoch(tracker.OnEpochEvent, tracker.OnEpochRestore)
	tracker.SetEligibilityChecker(&DummyEligibilityChecker{})
	epochService.target(context.Background(), types.Epoch{Seq: 1, Action: vgproto.EpochAction_EPOCH_ACTION_START, StartTime: time.Time{}})

	// add some markets for 2 different assets
	tracker.MarketProposed("a1", "m1", "z1")
	tracker.MarketProposed("a1", "m2", "z2")
	tracker.MarketProposed("a1", "m3", "z3")

	// add some markets for 2 different assets
	tracker.MarketProposed("a1", "m1", "z1")
	tracker.MarketProposed("a1", "m2", "z2")
	tracker.MarketProposed("a1", "m3", "z3")

	// record some values for all metrics
	tracker.RecordPosition("a1", "p1", "m1", 10, num.NewUint(1), num.DecimalOne(), time.Unix(5, 0))
	tracker.RecordPosition("a1", "p1", "m2", 20, num.NewUint(2), num.DecimalOne(), time.Unix(20, 0))
	tracker.RecordPosition("a1", "p1", "m3", 30, num.NewUint(3), num.DecimalOne(), time.Unix(30, 0))
	tracker.RecordPosition("a1", "p2", "m1", 100, num.NewUint(10), num.DecimalOne(), time.Unix(15, 0))
	tracker.RecordPosition("a1", "p2", "m2", 200, num.NewUint(20), num.DecimalOne(), time.Unix(25, 0))
	tracker.RecordPosition("a1", "p2", "m3", 300, num.NewUint(30), num.DecimalOne(), time.Unix(45, 0))
	tracker.RecordPosition("a1", "p3", "m1", 10, num.NewUint(1), num.DecimalOne(), time.Unix(10, 0))
	tracker.RecordPosition("a1", "p3", "m2", 20, num.NewUint(2), num.DecimalOne(), time.Unix(10, 0))
	tracker.RecordPosition("a1", "p3", "m3", 30, num.NewUint(3), num.DecimalOne(), time.Unix(10, 0))

	tracker.RecordM2M("a1", "p1", "m1", num.DecimalFromInt64(80))
	tracker.RecordM2M("a1", "p2", "m1", num.DecimalFromInt64(20))
	tracker.RecordM2M("a1", "p3", "m1", num.DecimalFromInt64(-100))
	tracker.RecordM2M("a1", "p1", "m1", num.DecimalFromInt64(10))
	tracker.RecordM2M("a1", "p2", "m1", num.DecimalFromInt64(-10))
	tracker.RecordM2M("a1", "p1", "m2", num.DecimalFromInt64(50))
	tracker.RecordM2M("a1", "p2", "m2", num.DecimalFromInt64(-5))
	tracker.RecordM2M("a1", "p3", "m2", num.DecimalFromInt64(-45))
	tracker.RecordM2M("a1", "p1", "m3", num.DecimalFromInt64(-35))
	tracker.RecordM2M("a1", "p2", "m3", num.DecimalFromInt64(35))

	// get metrics for market m1 with window size=1
	balanceChecker.EXPECT().GetAvailableBalance(gomock.Any()).Return(num.NewUint(0), nil).AnyTimes()

	ds1 := &vgproto.DispatchStrategy{
		AssetForMetric:       "a1",
		Metric:               vgproto.DispatchMetric_DISPATCH_METRIC_RETURN_VOLATILITY,
		Markets:              []string{"m1"},
		EntityScope:          vgproto.EntityScope_ENTITY_SCOPE_INDIVIDUALS,
		IndividualScope:      vgproto.IndividualScope_INDIVIDUAL_SCOPE_ALL,
		WindowLength:         1,
		DistributionStrategy: vgproto.DistributionStrategy_DISTRIBUTION_STRATEGY_PRO_RATA,
	}
	ds2 := &vgproto.DispatchStrategy{
		AssetForMetric:       "a1",
		Metric:               vgproto.DispatchMetric_DISPATCH_METRIC_RETURN_VOLATILITY,
		Markets:              []string{"m2"},
		EntityScope:          vgproto.EntityScope_ENTITY_SCOPE_INDIVIDUALS,
		IndividualScope:      vgproto.IndividualScope_INDIVIDUAL_SCOPE_ALL,
		WindowLength:         1,
		DistributionStrategy: vgproto.DistributionStrategy_DISTRIBUTION_STRATEGY_PRO_RATA,
	}
	ds3 := &vgproto.DispatchStrategy{
		AssetForMetric:       "a1",
		Metric:               vgproto.DispatchMetric_DISPATCH_METRIC_RETURN_VOLATILITY,
		Markets:              []string{"m3"},
		EntityScope:          vgproto.EntityScope_ENTITY_SCOPE_INDIVIDUALS,
		IndividualScope:      vgproto.IndividualScope_INDIVIDUAL_SCOPE_ALL,
		WindowLength:         1,
		DistributionStrategy: vgproto.DistributionStrategy_DISTRIBUTION_STRATEGY_PRO_RATA,
	}
	ds4 := &vgproto.DispatchStrategy{
		AssetForMetric:       "a1",
		Metric:               vgproto.DispatchMetric_DISPATCH_METRIC_RETURN_VOLATILITY,
		Markets:              []string{"m1", "m2"},
		EntityScope:          vgproto.EntityScope_ENTITY_SCOPE_INDIVIDUALS,
		IndividualScope:      vgproto.IndividualScope_INDIVIDUAL_SCOPE_ALL,
		WindowLength:         1,
		DistributionStrategy: vgproto.DistributionStrategy_DISTRIBUTION_STRATEGY_PRO_RATA,
	}
	ds5 := &vgproto.DispatchStrategy{
		AssetForMetric:       "a1",
		Metric:               vgproto.DispatchMetric_DISPATCH_METRIC_RETURN_VOLATILITY,
		EntityScope:          vgproto.EntityScope_ENTITY_SCOPE_INDIVIDUALS,
		IndividualScope:      vgproto.IndividualScope_INDIVIDUAL_SCOPE_ALL,
		WindowLength:         1,
		DistributionStrategy: vgproto.DistributionStrategy_DISTRIBUTION_STRATEGY_PRO_RATA,
	}

	tracker.PublishGameMetric(ctx, []*vgproto.DispatchStrategy{ds1, ds2, ds3, ds4, ds5}, time.Unix(60, 0))
	require.Equal(t, 5, len(gameScoreEvents))
	ps1 := gameScoreEvents[0].StreamMessage().GetGameScores().PartyScores
	require.Equal(t, "p1", ps1[0].Party)
	require.Equal(t, "p2", ps1[1].Party)
	require.Equal(t, "0", ps1[0].Score)
	require.Equal(t, "0", ps1[1].Score)

	ps2 := gameScoreEvents[1].StreamMessage().GetGameScores().PartyScores
	require.Equal(t, "p1", ps2[0].Party)
	require.Equal(t, "p2", ps2[1].Party)
	require.Equal(t, "0", ps2[0].Score)
	require.Equal(t, "0", ps2[1].Score)

	ps3 := gameScoreEvents[2].StreamMessage().GetGameScores().PartyScores
	require.Equal(t, "p1", ps3[0].Party)
	require.Equal(t, "p2", ps3[1].Party)
	require.Equal(t, "0", ps3[0].Score)
	require.Equal(t, "0", ps3[1].Score)

	ps4 := gameScoreEvents[3].StreamMessage().GetGameScores().PartyScores
	require.Equal(t, "p1", ps4[0].Party)
	require.Equal(t, "p2", ps4[1].Party)
	require.Equal(t, "0", ps4[0].Score)
	require.Equal(t, "0", ps4[1].Score)

	ps5 := gameScoreEvents[4].StreamMessage().GetGameScores().PartyScores
	require.Equal(t, "p1", ps5[0].Party)
	require.Equal(t, "p2", ps5[1].Party)
	require.Equal(t, "0", ps5[0].Score)
	require.Equal(t, "0", ps5[1].Score)

	epochService.target(context.Background(), types.Epoch{Seq: 1, Action: vgproto.EpochAction_EPOCH_ACTION_END, StartTime: time.Unix(0, 0), EndTime: time.Unix(60, 0)})
	epochService.target(context.Background(), types.Epoch{Seq: 2, Action: vgproto.EpochAction_EPOCH_ACTION_START, StartTime: time.Unix(60, 0)})
	tracker.RecordPosition("a1", "p1", "m1", 20, num.NewUint(5), num.DecimalOne(), time.Unix(90, 0))
	tracker.RecordPosition("a1", "p2", "m2", 10, num.NewUint(10), num.DecimalOne(), time.Unix(75, 0))
	tracker.RecordPosition("a1", "p3", "m1", 10, num.NewUint(10), num.DecimalOne(), time.Unix(75, 0))
	tracker.RecordPosition("a2", "p1", "m3", 20, num.NewUint(5), num.DecimalOne(), time.Unix(90, 0))
	tracker.RecordPosition("a2", "p2", "m2", 10, num.NewUint(10), num.DecimalOne(), time.Unix(75, 0))

	tracker.RecordM2M("a1", "p1", "m1", num.DecimalFromInt64(45))
	tracker.RecordM2M("a1", "p3", "m1", num.DecimalFromInt64(-45))
	tracker.RecordM2M("a1", "p1", "m2", num.DecimalFromInt64(-10))
	tracker.RecordM2M("a1", "p2", "m2", num.DecimalFromInt64(10))
	// nothing in m3

	ds1.WindowLength = 2
	ds2.WindowLength = 2
	ds3.WindowLength = 2
	ds4.WindowLength = 2
	ds5.WindowLength = 2

	gameScoreEvents = []events.Event{}
	tracker.PublishGameMetric(ctx, []*vgproto.DispatchStrategy{ds1, ds2, ds3, ds4, ds5}, time.Unix(120, 0))
	require.Equal(t, 5, len(gameScoreEvents))
	ps1 = gameScoreEvents[0].StreamMessage().GetGameScores().PartyScores
	require.Equal(t, "p1", ps1[0].Party)
	require.Equal(t, "p2", ps1[1].Party)
	require.Equal(t, "0.086044426422046", ps1[0].Score)
	require.Equal(t, "0", ps1[1].Score)
	require.Equal(t, true, ps1[0].IsEligible)
	require.Equal(t, false, ps1[1].IsEligible)

	ps2 = gameScoreEvents[1].StreamMessage().GetGameScores().PartyScores
	require.Equal(t, "p1", ps2[0].Party)
	require.Equal(t, "p2", ps2[1].Party)
	require.Equal(t, "0.2214532481172412", ps2[0].Score)
	require.Equal(t, "85.1257359604949139", ps2[1].Score)
	require.Equal(t, true, ps2[0].IsEligible)
	require.Equal(t, true, ps2[1].IsEligible)

	ps3 = gameScoreEvents[2].StreamMessage().GetGameScores().PartyScores
	require.Equal(t, "p1", ps3[0].Party)
	require.Equal(t, "p2", ps3[1].Party)
	require.Equal(t, "0", ps3[0].Score)
	require.Equal(t, "0", ps3[1].Score)
	require.Equal(t, false, ps3[0].IsEligible)
	require.Equal(t, false, ps1[1].IsEligible)

	ps4 = gameScoreEvents[3].StreamMessage().GetGameScores().PartyScores
	require.Equal(t, "p1", ps4[0].Party)
	require.Equal(t, "p2", ps4[1].Party)
	require.Equal(t, "0.0326518156928779", ps4[0].Score)
	require.Equal(t, "574.5715725244936759", ps4[1].Score)
	require.Equal(t, true, ps4[0].IsEligible)
	require.Equal(t, true, ps4[1].IsEligible)

	ps5 = gameScoreEvents[4].StreamMessage().GetGameScores().PartyScores
	require.Equal(t, "p1", ps5[0].Party)
	require.Equal(t, "p2", ps5[1].Party)
	require.Equal(t, "0.0524262906455334", ps5[0].Score)
	require.Equal(t, "27.2358805547724978", ps5[1].Score)
	require.Equal(t, true, ps5[0].IsEligible)
	require.Equal(t, true, ps5[1].IsEligible)

	// now end the epoch properly
	epochService.target(context.Background(), types.Epoch{Seq: 2, Action: vgproto.EpochAction_EPOCH_ACTION_END, StartTime: time.Unix(60, 0), EndTime: time.Unix(120, 0)})
	epochService.target(context.Background(), types.Epoch{Seq: 3, Action: vgproto.EpochAction_EPOCH_ACTION_START, StartTime: time.Unix(120, 0)})

	// record some m2ms
	tracker.RecordM2M("a1", "p1", "m1", num.DecimalFromInt64(20))
	tracker.RecordM2M("a1", "p3", "m1", num.DecimalFromInt64(-25))
	tracker.RecordM2M("a1", "p1", "m2", num.DecimalFromInt64(-15))
	tracker.RecordM2M("a1", "p2", "m2", num.DecimalFromInt64(15))

	gameScoreEvents = []events.Event{}
	tracker.PublishGameMetric(ctx, []*vgproto.DispatchStrategy{ds1, ds2, ds3, ds4, ds5}, time.Unix(150, 0))
	require.Equal(t, 5, len(gameScoreEvents))
	ps1 = gameScoreEvents[0].StreamMessage().GetGameScores().PartyScores
	require.Equal(t, "p1", ps1[0].Party)
	require.Equal(t, "p2", ps1[1].Party)
	require.Equal(t, "1", ps1[0].Score)
	require.Equal(t, "0", ps1[1].Score)
	require.Equal(t, true, ps1[0].IsEligible)
	require.Equal(t, false, ps1[1].IsEligible)

	ps2 = gameScoreEvents[1].StreamMessage().GetGameScores().PartyScores
	require.Equal(t, "p1", ps2[0].Party)
	require.Equal(t, "p2", ps2[1].Party)
	require.Equal(t, "64", ps2[0].Score)
	require.Equal(t, "2.2746573501746843", ps2[1].Score)
	require.Equal(t, true, ps2[0].IsEligible)
	require.Equal(t, true, ps2[1].IsEligible)

	ps3 = gameScoreEvents[2].StreamMessage().GetGameScores().PartyScores
	require.Equal(t, "p1", ps3[0].Party)
	require.Equal(t, "p2", ps3[1].Party)
	require.Equal(t, "0", ps3[0].Score)
	require.Equal(t, "0", ps3[1].Score)
	require.Equal(t, false, ps3[0].IsEligible)
	require.Equal(t, false, ps1[1].IsEligible)

	ps4 = gameScoreEvents[3].StreamMessage().GetGameScores().PartyScores
	require.Equal(t, "p1", ps4[0].Party)
	require.Equal(t, "p2", ps4[1].Party)
	require.Equal(t, "0.7901234567901235", ps4[0].Score)
	require.Equal(t, "2.2746573501746843", ps4[1].Score)
	require.Equal(t, true, ps4[0].IsEligible)
	require.Equal(t, true, ps4[1].IsEligible)

	ps5 = gameScoreEvents[4].StreamMessage().GetGameScores().PartyScores
	require.Equal(t, "p1", ps5[0].Party)
	require.Equal(t, "p2", ps5[1].Party)
	require.Equal(t, "0.7901234567901235", ps5[0].Score)
	require.Equal(t, "2.2746573501746843", ps5[1].Score)
	require.Equal(t, true, ps5[0].IsEligible)
	require.Equal(t, true, ps5[1].IsEligible)
}

func TestPublishGameMetricRelativeReturn(t *testing.T) {
	ctx := context.Background()
	epochService := &DummyEpochEngine{}
	ctrl := gomock.NewController(t)
	teams := mocks.NewMockTeams(ctrl)
	balanceChecker := mocks.NewMockAccountBalanceChecker(ctrl)
	broker := bmocks.NewMockBroker(ctrl)
	gameScoreEvents := []events.Event{}
	broker.EXPECT().SendBatch(gomock.Any()).AnyTimes()
	broker.EXPECT().Send(gomock.Any()).Do(func(evt events.Event) {
		if evt.StreamMessage().GetGameScores() != nil {
			gameScoreEvents = append(gameScoreEvents, evt)
		}
	}).AnyTimes()
	tracker := NewMarketActivityTracker(logging.NewTestLogger(), teams, balanceChecker, broker, DummyCollateralEngine{})
	epochService.NotifyOnEpoch(tracker.OnEpochEvent, tracker.OnEpochRestore)
	tracker.SetEligibilityChecker(&DummyEligibilityChecker{})
	epochService.target(context.Background(), types.Epoch{Seq: 1, Action: vgproto.EpochAction_EPOCH_ACTION_START, StartTime: time.Unix(0, 0)})

	balanceChecker.EXPECT().GetAvailableBalance(gomock.Any()).Return(num.NewUint(0), nil).AnyTimes()

	ds1 := &vgproto.DispatchStrategy{
		AssetForMetric:       "a1",
		Metric:               vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN,
		Markets:              []string{"m1"},
		EntityScope:          vgproto.EntityScope_ENTITY_SCOPE_INDIVIDUALS,
		IndividualScope:      vgproto.IndividualScope_INDIVIDUAL_SCOPE_ALL,
		WindowLength:         1,
		DistributionStrategy: vgproto.DistributionStrategy_DISTRIBUTION_STRATEGY_PRO_RATA,
	}
	ds2 := &vgproto.DispatchStrategy{
		AssetForMetric:       "a1",
		Metric:               vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN,
		Markets:              []string{"m2"},
		EntityScope:          vgproto.EntityScope_ENTITY_SCOPE_INDIVIDUALS,
		IndividualScope:      vgproto.IndividualScope_INDIVIDUAL_SCOPE_ALL,
		WindowLength:         1,
		DistributionStrategy: vgproto.DistributionStrategy_DISTRIBUTION_STRATEGY_PRO_RATA,
	}
	ds3 := &vgproto.DispatchStrategy{
		AssetForMetric:       "a1",
		Metric:               vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN,
		Markets:              []string{"m3"},
		EntityScope:          vgproto.EntityScope_ENTITY_SCOPE_INDIVIDUALS,
		IndividualScope:      vgproto.IndividualScope_INDIVIDUAL_SCOPE_ALL,
		WindowLength:         1,
		DistributionStrategy: vgproto.DistributionStrategy_DISTRIBUTION_STRATEGY_PRO_RATA,
	}
	ds4 := &vgproto.DispatchStrategy{
		AssetForMetric:       "a1",
		Metric:               vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN,
		Markets:              []string{"m1", "m2"},
		EntityScope:          vgproto.EntityScope_ENTITY_SCOPE_INDIVIDUALS,
		IndividualScope:      vgproto.IndividualScope_INDIVIDUAL_SCOPE_ALL,
		WindowLength:         1,
		DistributionStrategy: vgproto.DistributionStrategy_DISTRIBUTION_STRATEGY_PRO_RATA,
	}
	ds5 := &vgproto.DispatchStrategy{
		AssetForMetric:       "a1",
		Metric:               vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN,
		EntityScope:          vgproto.EntityScope_ENTITY_SCOPE_INDIVIDUALS,
		IndividualScope:      vgproto.IndividualScope_INDIVIDUAL_SCOPE_ALL,
		WindowLength:         1,
		DistributionStrategy: vgproto.DistributionStrategy_DISTRIBUTION_STRATEGY_PRO_RATA,
	}

	// add some markets for 2 different assets
	tracker.MarketProposed("a1", "m1", "z1")
	tracker.MarketProposed("a1", "m2", "z2")
	tracker.MarketProposed("a1", "m3", "z3")

	// record some values for all metrics
	tracker.RecordPosition("a1", "p1", "m1", 10, num.NewUint(1), num.DecimalOne(), time.Unix(5, 0))
	tracker.RecordPosition("a1", "p1", "m2", 20, num.NewUint(2), num.DecimalOne(), time.Unix(20, 0))
	tracker.RecordPosition("a1", "p1", "m3", 30, num.NewUint(3), num.DecimalOne(), time.Unix(30, 0))
	tracker.RecordPosition("a1", "p2", "m1", 100, num.NewUint(10), num.DecimalOne(), time.Unix(15, 0))
	tracker.RecordPosition("a1", "p2", "m2", 200, num.NewUint(20), num.DecimalOne(), time.Unix(25, 0))
	tracker.RecordPosition("a1", "p2", "m3", 300, num.NewUint(30), num.DecimalOne(), time.Unix(45, 0))

	tracker.RecordM2M("a1", "p1", "m1", num.DecimalFromInt64(-100))
	tracker.RecordM2M("a1", "p2", "m1", num.DecimalFromInt64(100))
	tracker.RecordM2M("a1", "p1", "m1", num.DecimalFromInt64(250))
	tracker.RecordM2M("a1", "p2", "m1", num.DecimalFromInt64(-250))
	tracker.RecordM2M("a1", "p1", "m2", num.DecimalFromInt64(-50))
	tracker.RecordM2M("a1", "p2", "m2", num.DecimalFromInt64(50))
	tracker.RecordM2M("a1", "p1", "m3", num.DecimalFromInt64(100))
	tracker.RecordM2M("a1", "p2", "m3", num.DecimalFromInt64(-100))

	tracker.PublishGameMetric(ctx, []*vgproto.DispatchStrategy{ds1, ds2, ds3, ds4, ds5}, time.Unix(60, 0))
	require.Equal(t, 5, len(gameScoreEvents))
	ps1 := gameScoreEvents[0].StreamMessage().GetGameScores().PartyScores
	require.Equal(t, "p1", ps1[0].Party)
	require.Equal(t, "p2", ps1[1].Party)
	require.Equal(t, "16.3636375537190948", ps1[0].Score)
	require.Equal(t, "-2", ps1[1].Score)

	ps2 := gameScoreEvents[1].StreamMessage().GetGameScores().PartyScores
	require.Equal(t, "p1", ps2[0].Party)
	require.Equal(t, "p2", ps2[1].Party)
	require.Equal(t, "-3.7500003750000375", ps2[0].Score)
	require.Equal(t, "0.4285714530612259", ps2[1].Score)

	ps3 := gameScoreEvents[2].StreamMessage().GetGameScores().PartyScores
	require.Equal(t, "p1", ps3[0].Party)
	require.Equal(t, "p2", ps3[1].Party)
	require.Equal(t, "6.6666666666666667", ps3[0].Score)
	require.Equal(t, "-1.3333333333333333", ps3[1].Score)

	ps4 := gameScoreEvents[3].StreamMessage().GetGameScores().PartyScores
	require.Equal(t, "p1", ps4[0].Party)
	require.Equal(t, "p2", ps4[1].Party)
	require.Equal(t, "12.6136371787190573", ps4[0].Score)
	require.Equal(t, "-1.5714285469387741", ps4[1].Score)

	ps5 := gameScoreEvents[4].StreamMessage().GetGameScores().PartyScores
	require.Equal(t, "p1", ps5[0].Party)
	require.Equal(t, "p2", ps5[1].Party)
	require.Equal(t, "19.280303845385724", ps5[0].Score)
	require.Equal(t, "-2.9047618802721074", ps5[1].Score)

	epochService.target(context.Background(), types.Epoch{Seq: 1, Action: vgproto.EpochAction_EPOCH_ACTION_END, StartTime: time.Unix(0, 0), EndTime: time.Unix(60, 0)})
	epochService.target(context.Background(), types.Epoch{Seq: 2, Action: vgproto.EpochAction_EPOCH_ACTION_START, StartTime: time.Unix(60, 0)})

	tracker.RecordPosition("a1", "p1", "m1", 20, num.NewUint(5), num.DecimalOne(), time.Unix(90, 0))
	tracker.RecordPosition("a1", "p2", "m2", 10, num.NewUint(10), num.DecimalOne(), time.Unix(75, 0))
	tracker.RecordPosition("a2", "p1", "m3", 20, num.NewUint(5), num.DecimalOne(), time.Unix(90, 0))
	tracker.RecordPosition("a2", "p2", "m2", 10, num.NewUint(10), num.DecimalOne(), time.Unix(75, 0))

	tracker.RecordM2M("a1", "p1", "m1", num.DecimalFromInt64(450))
	tracker.RecordM2M("a1", "p2", "m1", num.DecimalFromInt64(-450))
	tracker.RecordM2M("a1", "p1", "m2", num.DecimalFromInt64(-100))
	tracker.RecordM2M("a1", "p2", "m2", num.DecimalFromInt64(100))
	// nothing in m3

	gameScoreEvents = []events.Event{}
	tracker.PublishGameMetric(ctx, []*vgproto.DispatchStrategy{ds1, ds2, ds3, ds4, ds5}, time.Unix(120, 0))
	require.Equal(t, 5, len(gameScoreEvents))
	ps1 = gameScoreEvents[0].StreamMessage().GetGameScores().PartyScores
	require.Equal(t, "p1", ps1[0].Party)
	require.Equal(t, "p2", ps1[1].Party)
	require.Equal(t, "30", ps1[0].Score)
	require.Equal(t, "-4.5", ps1[1].Score)

	ps2 = gameScoreEvents[1].StreamMessage().GetGameScores().PartyScores
	require.Equal(t, "p1", ps2[0].Party)
	require.Equal(t, "p2", ps2[1].Party)
	require.Equal(t, "-5", ps2[0].Score)
	require.Equal(t, "1.7391304347826087", ps2[1].Score)

	ps3 = gameScoreEvents[2].StreamMessage().GetGameScores().PartyScores
	require.Equal(t, "p1", ps3[0].Party)
	require.Equal(t, "p2", ps3[1].Party)
	require.Equal(t, "0", ps3[0].Score)
	require.Equal(t, "0", ps3[1].Score)
	require.False(t, ps3[0].IsEligible)
	require.False(t, ps3[1].IsEligible)

	ps4 = gameScoreEvents[3].StreamMessage().GetGameScores().PartyScores
	require.Equal(t, "p1", ps4[0].Party)
	require.Equal(t, "p2", ps4[1].Party)
	require.Equal(t, "25", ps4[0].Score)
	require.Equal(t, "-2.7608695652173913", ps4[1].Score)

	ps5 = gameScoreEvents[4].StreamMessage().GetGameScores().PartyScores
	require.Equal(t, "p1", ps5[0].Party)
	require.Equal(t, "p2", ps5[1].Party)
	require.Equal(t, "25", ps5[0].Score)
	require.Equal(t, "-2.7608695652173913", ps5[1].Score)

	// check with window length = 2
	ds1.WindowLength = 2
	ds2.WindowLength = 2
	ds3.WindowLength = 2
	ds4.WindowLength = 2
	ds5.WindowLength = 2

	gameScoreEvents = []events.Event{}
	tracker.PublishGameMetric(ctx, []*vgproto.DispatchStrategy{ds1, ds2, ds3, ds4, ds5}, time.Unix(120, 0))
	require.Equal(t, 5, len(gameScoreEvents))
	ps1 = gameScoreEvents[0].StreamMessage().GetGameScores().PartyScores
	require.Equal(t, "p1", ps1[0].Party)
	require.Equal(t, "p2", ps1[1].Party)
	require.Equal(t, "23.1818187768595474", ps1[0].Score)
	require.Equal(t, "-3.25", ps1[1].Score)
	require.Equal(t, true, ps1[0].IsEligible)
	require.Equal(t, true, ps1[1].IsEligible)

	ps2 = gameScoreEvents[1].StreamMessage().GetGameScores().PartyScores
	require.Equal(t, "p1", ps2[0].Party)
	require.Equal(t, "p2", ps2[1].Party)
	require.Equal(t, "-4.3750001875000188", ps2[0].Score)
	require.Equal(t, "1.0838509439219173", ps2[1].Score)
	require.Equal(t, true, ps2[0].IsEligible)
	require.Equal(t, true, ps2[1].IsEligible)

	ps3 = gameScoreEvents[2].StreamMessage().GetGameScores().PartyScores
	require.Equal(t, "p1", ps3[0].Party)
	require.Equal(t, "p2", ps3[1].Party)
	require.Equal(t, "3.3333333333333334", ps3[0].Score)
	require.Equal(t, "-0.6666666666666667", ps3[1].Score)
	require.Equal(t, true, ps3[0].IsEligible)
	require.Equal(t, true, ps1[1].IsEligible)

	ps4 = gameScoreEvents[3].StreamMessage().GetGameScores().PartyScores
	require.Equal(t, "p1", ps4[0].Party)
	require.Equal(t, "p2", ps4[1].Party)
	require.Equal(t, "18.8068185893595287", ps4[0].Score)
	require.Equal(t, "-2.1661490560780827", ps4[1].Score)
	require.Equal(t, true, ps4[0].IsEligible)
	require.Equal(t, true, ps4[1].IsEligible)

	ps5 = gameScoreEvents[4].StreamMessage().GetGameScores().PartyScores
	require.Equal(t, "p1", ps5[0].Party)
	require.Equal(t, "p2", ps5[1].Party)
	require.Equal(t, "22.140151922692862", ps5[0].Score)
	require.Equal(t, "-2.8328157227447494", ps5[1].Score)
	require.Equal(t, true, ps5[0].IsEligible)
	require.Equal(t, true, ps5[1].IsEligible)
}

func TestPublishGameMetricRealisedReturn(t *testing.T) {
	ctx := context.Background()
	epochService := &DummyEpochEngine{}
	ctrl := gomock.NewController(t)
	teams := mocks.NewMockTeams(ctrl)
	balanceChecker := mocks.NewMockAccountBalanceChecker(ctrl)
	broker := bmocks.NewMockBroker(ctrl)
	gameScoreEvents := []events.Event{}
	broker.EXPECT().SendBatch(gomock.Any()).AnyTimes()
	broker.EXPECT().Send(gomock.Any()).Do(func(evt events.Event) {
		if evt.StreamMessage().GetGameScores() != nil {
			gameScoreEvents = append(gameScoreEvents, evt)
		}
	}).AnyTimes()
	tracker := NewMarketActivityTracker(logging.NewTestLogger(), teams, balanceChecker, broker, DummyCollateralEngine{})
	epochService.NotifyOnEpoch(tracker.OnEpochEvent, tracker.OnEpochRestore)
	tracker.SetEligibilityChecker(&DummyEligibilityChecker{})
	epochService.target(context.Background(), types.Epoch{Seq: 1, Action: vgproto.EpochAction_EPOCH_ACTION_START, StartTime: time.Unix(0, 0)})

	balanceChecker.EXPECT().GetAvailableBalance(gomock.Any()).Return(num.NewUint(0), nil).AnyTimes()

	tracker.MarketProposed("a1", "m1", "z1")

	ds1 := &vgproto.DispatchStrategy{
		AssetForMetric:       "a1",
		Metric:               vgproto.DispatchMetric_DISPATCH_METRIC_REALISED_RETURN,
		Markets:              []string{"m1"},
		EntityScope:          vgproto.EntityScope_ENTITY_SCOPE_INDIVIDUALS,
		IndividualScope:      vgproto.IndividualScope_INDIVIDUAL_SCOPE_ALL,
		WindowLength:         1,
		DistributionStrategy: vgproto.DistributionStrategy_DISTRIBUTION_STRATEGY_PRO_RATA,
	}
	ds2 := &vgproto.DispatchStrategy{
		AssetForMetric:       "a1",
		Metric:               vgproto.DispatchMetric_DISPATCH_METRIC_REALISED_RETURN,
		EntityScope:          vgproto.EntityScope_ENTITY_SCOPE_INDIVIDUALS,
		IndividualScope:      vgproto.IndividualScope_INDIVIDUAL_SCOPE_ALL,
		WindowLength:         1,
		DistributionStrategy: vgproto.DistributionStrategy_DISTRIBUTION_STRATEGY_PRO_RATA,
	}

	tracker.RecordFundingPayment("a1", "p1", "m1", num.DecimalFromInt64(100))
	tracker.RecordRealisedPosition("a1", "p1", "m1", num.DecimalFromInt64(-50))
	tracker.RecordFundingPayment("a1", "p1", "m1", num.DecimalFromInt64(-200))
	tracker.RecordRealisedPosition("a1", "p1", "m1", num.DecimalFromInt64(20))
	tracker.RecordFundingPayment("a1", "p2", "m1", num.DecimalFromInt64(-100))
	tracker.RecordRealisedPosition("a1", "p2", "m1", num.DecimalFromInt64(-10))
	tracker.RecordRealisedPosition("a1", "p2", "m1", num.DecimalFromInt64(20))
	tracker.RecordFundingPayment("a1", "p3", "m1", num.DecimalFromInt64(200))

	tracker.PublishGameMetric(ctx, []*vgproto.DispatchStrategy{ds1, ds2}, time.Unix(60, 0))
	require.Equal(t, 2, len(gameScoreEvents))
	ps1 := gameScoreEvents[0].StreamMessage().GetGameScores().PartyScores
	require.Equal(t, "p1", ps1[0].Party)
	require.Equal(t, "p2", ps1[1].Party)
	require.Equal(t, "p3", ps1[2].Party)
	require.Equal(t, "-130", ps1[0].Score)
	require.Equal(t, "-90", ps1[1].Score)
	require.Equal(t, "200", ps1[2].Score)

	ps2 := gameScoreEvents[1].StreamMessage().GetGameScores().PartyScores
	require.Equal(t, "p1", ps2[0].Party)
	require.Equal(t, "p2", ps2[1].Party)
	require.Equal(t, "p3", ps2[2].Party)
	require.Equal(t, "-130", ps2[0].Score)
	require.Equal(t, "-90", ps2[1].Score)
	require.Equal(t, "200", ps2[2].Score)

	epochService.target(context.Background(), types.Epoch{Seq: 1, Action: vgproto.EpochAction_EPOCH_ACTION_END, StartTime: time.Unix(0, 0), EndTime: time.Unix(60, 0)})
	epochService.target(context.Background(), types.Epoch{Seq: 2, Action: vgproto.EpochAction_EPOCH_ACTION_START, StartTime: time.Unix(60, 0)})

	tracker.RecordFundingPayment("a1", "p1", "m1", num.DecimalFromInt64(-30))
	tracker.RecordRealisedPosition("a1", "p2", "m1", num.DecimalFromInt64(70))
	tracker.RecordRealisedPosition("a1", "p2", "m1", num.DecimalFromInt64(80))
	tracker.RecordRealisedPosition("a1", "p3", "m1", num.DecimalFromInt64(-50))

	// with window size1
	gameScoreEvents = []events.Event{}
	tracker.PublishGameMetric(ctx, []*vgproto.DispatchStrategy{ds1, ds2}, time.Unix(120, 0))
	require.Equal(t, 2, len(gameScoreEvents))

	ps1 = gameScoreEvents[0].StreamMessage().GetGameScores().PartyScores
	require.Equal(t, "p1", ps1[0].Party)
	require.Equal(t, "p2", ps1[1].Party)
	require.Equal(t, "p3", ps1[2].Party)
	require.Equal(t, "-30", ps1[0].Score)
	require.Equal(t, "150", ps1[1].Score)
	require.Equal(t, "-50", ps1[2].Score)

	ps2 = gameScoreEvents[1].StreamMessage().GetGameScores().PartyScores
	require.Equal(t, "p1", ps2[0].Party)
	require.Equal(t, "p2", ps2[1].Party)
	require.Equal(t, "p3", ps2[2].Party)
	require.Equal(t, "-30", ps2[0].Score)
	require.Equal(t, "150", ps2[1].Score)
	require.Equal(t, "-50", ps2[2].Score)

	// check with window length = 2
	ds1.WindowLength = 2
	ds2.WindowLength = 2

	gameScoreEvents = []events.Event{}
	tracker.PublishGameMetric(ctx, []*vgproto.DispatchStrategy{ds1, ds2}, time.Unix(120, 0))
	require.Equal(t, 2, len(gameScoreEvents))

	ps1 = gameScoreEvents[0].StreamMessage().GetGameScores().PartyScores
	require.Equal(t, "p1", ps1[0].Party)
	require.Equal(t, "p2", ps1[1].Party)
	require.Equal(t, "p3", ps1[2].Party)
	require.Equal(t, "-80", ps1[0].Score)
	require.Equal(t, "30", ps1[1].Score)
	require.Equal(t, "75", ps1[2].Score)

	ps2 = gameScoreEvents[1].StreamMessage().GetGameScores().PartyScores
	require.Equal(t, "p1", ps2[0].Party)
	require.Equal(t, "p2", ps2[1].Party)
	require.Equal(t, "p3", ps2[2].Party)
	require.Equal(t, "-80", ps2[0].Score)
	require.Equal(t, "30", ps2[1].Score)
	require.Equal(t, "75", ps2[2].Score)
}

func TestPublishGameMetricFees(t *testing.T) {
	ctx := context.Background()
	epochService := &DummyEpochEngine{}
	ctrl := gomock.NewController(t)
	teams := mocks.NewMockTeams(ctrl)
	balanceChecker := mocks.NewMockAccountBalanceChecker(ctrl)
	broker := bmocks.NewMockBroker(ctrl)
	gameScoreEvents := []events.Event{}
	broker.EXPECT().SendBatch(gomock.Any()).AnyTimes()
	broker.EXPECT().Send(gomock.Any()).Do(func(evt events.Event) {
		if evt.StreamMessage().GetGameScores() != nil {
			gameScoreEvents = append(gameScoreEvents, evt)
		}
	}).AnyTimes()
	tracker := NewMarketActivityTracker(logging.NewTestLogger(), teams, balanceChecker, broker, DummyCollateralEngine{})
	epochService.NotifyOnEpoch(tracker.OnEpochEvent, tracker.OnEpochRestore)
	tracker.SetEligibilityChecker(&DummyEligibilityChecker{})
	epochService.target(context.Background(), types.Epoch{Seq: 1, Action: vgproto.EpochAction_EPOCH_ACTION_START, StartTime: time.Unix(0, 0)})

	balanceChecker.EXPECT().GetAvailableBalance(gomock.Any()).Return(num.NewUint(0), nil).AnyTimes()

	tracker.MarketProposed("a1", "m1", "me")
	tracker.MarketProposed("a1", "m2", "me2")

	ds1 := &vgproto.DispatchStrategy{
		AssetForMetric:       "a1",
		Metric:               vgproto.DispatchMetric_DISPATCH_METRIC_MAKER_FEES_PAID,
		Markets:              []string{"m1"},
		EntityScope:          vgproto.EntityScope_ENTITY_SCOPE_INDIVIDUALS,
		IndividualScope:      vgproto.IndividualScope_INDIVIDUAL_SCOPE_ALL,
		WindowLength:         1,
		DistributionStrategy: vgproto.DistributionStrategy_DISTRIBUTION_STRATEGY_PRO_RATA,
	}
	ds2 := &vgproto.DispatchStrategy{
		AssetForMetric:       "a1",
		Metric:               vgproto.DispatchMetric_DISPATCH_METRIC_MAKER_FEES_PAID,
		Markets:              []string{"m2"},
		EntityScope:          vgproto.EntityScope_ENTITY_SCOPE_INDIVIDUALS,
		IndividualScope:      vgproto.IndividualScope_INDIVIDUAL_SCOPE_ALL,
		WindowLength:         1,
		DistributionStrategy: vgproto.DistributionStrategy_DISTRIBUTION_STRATEGY_PRO_RATA,
	}
	ds3 := &vgproto.DispatchStrategy{
		AssetForMetric:       "a1",
		Metric:               vgproto.DispatchMetric_DISPATCH_METRIC_MAKER_FEES_PAID,
		Markets:              []string{"m1", "m2"},
		EntityScope:          vgproto.EntityScope_ENTITY_SCOPE_INDIVIDUALS,
		IndividualScope:      vgproto.IndividualScope_INDIVIDUAL_SCOPE_ALL,
		WindowLength:         1,
		DistributionStrategy: vgproto.DistributionStrategy_DISTRIBUTION_STRATEGY_PRO_RATA,
	}
	ds4 := &vgproto.DispatchStrategy{
		AssetForMetric:       "a1",
		Metric:               vgproto.DispatchMetric_DISPATCH_METRIC_MAKER_FEES_PAID,
		Markets:              []string{},
		EntityScope:          vgproto.EntityScope_ENTITY_SCOPE_INDIVIDUALS,
		IndividualScope:      vgproto.IndividualScope_INDIVIDUAL_SCOPE_ALL,
		WindowLength:         1,
		DistributionStrategy: vgproto.DistributionStrategy_DISTRIBUTION_STRATEGY_PRO_RATA,
	}
	ds5 := &vgproto.DispatchStrategy{
		AssetForMetric:       "a1",
		Metric:               vgproto.DispatchMetric_DISPATCH_METRIC_MAKER_FEES_RECEIVED,
		Markets:              []string{"m1"},
		EntityScope:          vgproto.EntityScope_ENTITY_SCOPE_INDIVIDUALS,
		IndividualScope:      vgproto.IndividualScope_INDIVIDUAL_SCOPE_ALL,
		WindowLength:         1,
		DistributionStrategy: vgproto.DistributionStrategy_DISTRIBUTION_STRATEGY_PRO_RATA,
	}
	ds6 := &vgproto.DispatchStrategy{
		AssetForMetric:       "a1",
		Metric:               vgproto.DispatchMetric_DISPATCH_METRIC_MAKER_FEES_RECEIVED,
		Markets:              []string{"m2"},
		EntityScope:          vgproto.EntityScope_ENTITY_SCOPE_INDIVIDUALS,
		IndividualScope:      vgproto.IndividualScope_INDIVIDUAL_SCOPE_ALL,
		WindowLength:         1,
		DistributionStrategy: vgproto.DistributionStrategy_DISTRIBUTION_STRATEGY_PRO_RATA,
	}
	ds7 := &vgproto.DispatchStrategy{
		AssetForMetric:       "a1",
		Metric:               vgproto.DispatchMetric_DISPATCH_METRIC_MAKER_FEES_RECEIVED,
		Markets:              []string{"m1", "m2"},
		EntityScope:          vgproto.EntityScope_ENTITY_SCOPE_INDIVIDUALS,
		IndividualScope:      vgproto.IndividualScope_INDIVIDUAL_SCOPE_ALL,
		WindowLength:         1,
		DistributionStrategy: vgproto.DistributionStrategy_DISTRIBUTION_STRATEGY_PRO_RATA,
	}
	ds8 := &vgproto.DispatchStrategy{
		AssetForMetric:       "a1",
		Metric:               vgproto.DispatchMetric_DISPATCH_METRIC_MAKER_FEES_RECEIVED,
		Markets:              []string{},
		EntityScope:          vgproto.EntityScope_ENTITY_SCOPE_INDIVIDUALS,
		IndividualScope:      vgproto.IndividualScope_INDIVIDUAL_SCOPE_ALL,
		WindowLength:         1,
		DistributionStrategy: vgproto.DistributionStrategy_DISTRIBUTION_STRATEGY_PRO_RATA,
	}

	epochService.target(context.Background(), types.Epoch{Seq: 1, Action: vgproto.EpochAction_EPOCH_ACTION_START})

	tracker.MarketProposed("a1", "market1", "me")
	tracker.MarketProposed("a1", "market2", "me2")

	// update with a few transfers
	transfersM1 := []*types.Transfer{
		{Owner: "p1", Type: types.TransferTypeMakerFeeReceive, Amount: &types.FinancialAmount{Asset: "a1", Amount: num.NewUint(100)}},
		{Owner: "p1", Type: types.TransferTypeMakerFeePay, Amount: &types.FinancialAmount{Asset: "a1", Amount: num.NewUint(200)}},
		{Owner: "p1", Type: types.TransferTypeMakerFeeReceive, Amount: &types.FinancialAmount{Asset: "a1", Amount: num.NewUint(400)}},
		{Owner: "p1", Type: types.TransferTypeMakerFeePay, Amount: &types.FinancialAmount{Asset: "a1", Amount: num.NewUint(300)}},
		{Owner: "p2", Type: types.TransferTypeMakerFeeReceive, Amount: &types.FinancialAmount{Asset: "a1", Amount: num.NewUint(900)}},
		{Owner: "p2", Type: types.TransferTypeMakerFeePay, Amount: &types.FinancialAmount{Asset: "a1", Amount: num.NewUint(800)}},
		{Owner: "p2", Type: types.TransferTypeMakerFeeReceive, Amount: &types.FinancialAmount{Asset: "a1", Amount: num.NewUint(600)}},
		{Owner: "p2", Type: types.TransferTypeMakerFeePay, Amount: &types.FinancialAmount{Asset: "a1", Amount: num.NewUint(200)}},
	}
	tracker.UpdateFeesFromTransfers("a1", "m1", transfersM1)

	transfersM2 := []*types.Transfer{
		{Owner: "p1", Type: types.TransferTypeMakerFeeReceive, Amount: &types.FinancialAmount{Asset: "a2", Amount: num.NewUint(150)}},
		{Owner: "p2", Type: types.TransferTypeMakerFeePay, Amount: &types.FinancialAmount{Asset: "a2", Amount: num.NewUint(150)}},
	}
	tracker.UpdateFeesFromTransfers("a1", "m2", transfersM2)

	tracker.PublishGameMetric(ctx, []*vgproto.DispatchStrategy{ds1, ds2, ds3, ds4, ds5, ds6, ds7, ds8}, time.Unix(10, 0))
	require.Equal(t, 8, len(gameScoreEvents))

	ps1 := gameScoreEvents[0].StreamMessage().GetGameScores().PartyScores
	require.Equal(t, "p1", ps1[0].Party)
	require.Equal(t, "0.3333333333333333", ps1[0].Score)
	require.Equal(t, "p2", ps1[1].Party)
	require.Equal(t, "0.6666666666666667", ps1[1].Score)

	ps2 := gameScoreEvents[1].StreamMessage().GetGameScores().PartyScores
	require.Equal(t, "p1", ps2[0].Party)
	require.Equal(t, "0", ps2[0].Score)
	require.False(t, ps2[0].IsEligible)
	require.Equal(t, "p2", ps2[1].Party)
	require.Equal(t, "1", ps2[1].Score)

	ps3 := gameScoreEvents[2].StreamMessage().GetGameScores().PartyScores
	require.Equal(t, "p1", ps3[0].Party)
	require.Equal(t, "0.303030303030303", ps3[0].Score)
	require.Equal(t, "p2", ps3[1].Party)
	require.Equal(t, "0.696969696969697", ps3[1].Score)

	ps4 := gameScoreEvents[3].StreamMessage().GetGameScores().PartyScores
	require.Equal(t, "p1", ps4[0].Party)
	require.Equal(t, "0.303030303030303", ps4[0].Score)
	require.Equal(t, "p2", ps4[1].Party)
	require.Equal(t, "0.696969696969697", ps4[1].Score)

	ps5 := gameScoreEvents[4].StreamMessage().GetGameScores().PartyScores
	require.Equal(t, "p1", ps5[0].Party)
	require.Equal(t, "0.25", ps5[0].Score)
	require.Equal(t, "p2", ps5[1].Party)
	require.Equal(t, "0.75", ps5[1].Score)

	ps6 := gameScoreEvents[5].StreamMessage().GetGameScores().PartyScores
	require.Equal(t, "p1", ps6[0].Party)
	require.Equal(t, "1", ps6[0].Score)
	require.Equal(t, "p2", ps6[1].Party)
	require.Equal(t, "0", ps6[1].Score)
	require.False(t, ps6[1].IsEligible)

	ps7 := gameScoreEvents[6].StreamMessage().GetGameScores().PartyScores
	require.Equal(t, "p1", ps7[0].Party)
	require.Equal(t, "0.3023255813953488", ps7[0].Score)
	require.Equal(t, "p2", ps7[1].Party)
	require.Equal(t, "0.6976744186046512", ps7[1].Score)

	ps8 := gameScoreEvents[7].StreamMessage().GetGameScores().PartyScores
	require.Equal(t, "p1", ps8[0].Party)
	require.Equal(t, "0.3023255813953488", ps8[0].Score)
	require.Equal(t, "p2", ps8[1].Party)
	require.Equal(t, "0.6976744186046512", ps8[1].Score)

	epochService.target(context.Background(), types.Epoch{Seq: 1, Action: vgproto.EpochAction_EPOCH_ACTION_END})
	epochService.target(context.Background(), types.Epoch{Seq: 2, Action: vgproto.EpochAction_EPOCH_ACTION_START})

	ds1.WindowLength = 2
	ds2.WindowLength = 2
	ds3.WindowLength = 2
	ds4.WindowLength = 2
	ds5.WindowLength = 2
	ds6.WindowLength = 2
	ds7.WindowLength = 2
	ds8.WindowLength = 2

	// pay/receive some fees in me for the new epoch
	transfersM1 = []*types.Transfer{
		{Owner: "p1", Type: types.TransferTypeMakerFeeReceive, Amount: &types.FinancialAmount{Asset: "a1", Amount: num.NewUint(300)}},
		{Owner: "p1", Type: types.TransferTypeMakerFeePay, Amount: &types.FinancialAmount{Asset: "a1", Amount: num.NewUint(100)}},
		{Owner: "p2", Type: types.TransferTypeMakerFeeReceive, Amount: &types.FinancialAmount{Asset: "a1", Amount: num.NewUint(900)}},
	}
	tracker.UpdateFeesFromTransfers("a1", "m1", transfersM1)

	gameScoreEvents = []events.Event{}
	tracker.PublishGameMetric(ctx, []*vgproto.DispatchStrategy{ds1, ds2, ds3, ds4, ds5, ds6, ds7, ds8}, time.Unix(20, 0))
	require.Equal(t, 8, len(gameScoreEvents))

	ps1 = gameScoreEvents[0].StreamMessage().GetGameScores().PartyScores
	require.Equal(t, "p1", ps1[0].Party)
	require.Equal(t, "0.375", ps1[0].Score)
	require.Equal(t, "p2", ps1[1].Party)
	require.Equal(t, "0.625", ps1[1].Score)

	ps2 = gameScoreEvents[1].StreamMessage().GetGameScores().PartyScores
	require.Equal(t, "p1", ps2[0].Party)
	require.Equal(t, "0", ps2[0].Score)
	require.False(t, ps2[0].IsEligible)
	require.Equal(t, "p2", ps2[1].Party)
	require.Equal(t, "1", ps2[1].Score)

	ps3 = gameScoreEvents[2].StreamMessage().GetGameScores().PartyScores
	require.Equal(t, "p1", ps3[0].Party)
	require.Equal(t, "0.3428571428571429", ps3[0].Score)
	require.Equal(t, "p2", ps3[1].Party)
	require.Equal(t, "0.6571428571428571", ps3[1].Score)

	ps4 = gameScoreEvents[3].StreamMessage().GetGameScores().PartyScores
	require.Equal(t, "p1", ps4[0].Party)
	require.Equal(t, "0.3428571428571429", ps4[0].Score)
	require.Equal(t, "p2", ps4[1].Party)
	require.Equal(t, "0.6571428571428571", ps4[1].Score)

	ps5 = gameScoreEvents[4].StreamMessage().GetGameScores().PartyScores
	require.Equal(t, "p1", ps5[0].Party)
	require.Equal(t, "0.25", ps5[0].Score)
	require.Equal(t, "p2", ps5[1].Party)
	require.Equal(t, "0.75", ps5[1].Score)

	ps6 = gameScoreEvents[5].StreamMessage().GetGameScores().PartyScores
	require.Equal(t, "p1", ps6[0].Party)
	require.Equal(t, "1", ps6[0].Score)
	require.Equal(t, "p2", ps6[1].Party)
	require.Equal(t, "0", ps6[1].Score)
	require.False(t, ps6[1].IsEligible)

	ps7 = gameScoreEvents[6].StreamMessage().GetGameScores().PartyScores
	require.Equal(t, "p1", ps7[0].Party)
	require.Equal(t, "0.2835820895522388", ps7[0].Score)
	require.Equal(t, "p2", ps7[1].Party)
	require.Equal(t, "0.7164179104477612", ps7[1].Score)

	ps8 = gameScoreEvents[7].StreamMessage().GetGameScores().PartyScores
	require.Equal(t, "p1", ps8[0].Party)
	require.Equal(t, "0.2835820895522388", ps8[0].Score)
	require.Equal(t, "p2", ps8[1].Party)
	require.Equal(t, "0.7164179104477612", ps8[1].Score)
}
