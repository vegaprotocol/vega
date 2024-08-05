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

package common_test

import (
	"bytes"
	"context"
	"testing"
	"time"

	bmocks "code.vegaprotocol.io/vega/core/broker/mocks"
	"code.vegaprotocol.io/vega/core/execution/common"
	"code.vegaprotocol.io/vega/core/execution/common/mocks"
	"code.vegaprotocol.io/vega/core/integration/stubs"
	snp "code.vegaprotocol.io/vega/core/snapshot"
	"code.vegaprotocol.io/vega/core/stats"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/libs/proto"
	vgrand "code.vegaprotocol.io/vega/libs/rand"
	vgtest "code.vegaprotocol.io/vega/libs/test"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/paths"
	vgproto "code.vegaprotocol.io/vega/protos/vega"
	snapshotpb "code.vegaprotocol.io/vega/protos/vega/snapshot/v1"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type TestEpochEngine struct {
	target func(context.Context, types.Epoch)
}

func (e *TestEpochEngine) NotifyOnEpoch(f func(context.Context, types.Epoch), _ func(context.Context, types.Epoch)) {
	e.target = f
}

type EligibilityChecker struct{}

func (e *EligibilityChecker) IsEligibleForProposerBonus(marketID string, volumeTraded *num.Uint) bool {
	return volumeTraded.GT(num.NewUint(5000))
}

func TestMarketTracker(t *testing.T) {
	ctrl := gomock.NewController(t)
	teams := mocks.NewMockTeams(ctrl)
	balanceChecker := mocks.NewMockAccountBalanceChecker(ctrl)

	broker := bmocks.NewMockBroker(ctrl)
	tracker := common.NewMarketActivityTracker(logging.NewTestLogger(), teams, balanceChecker, broker)
	tracker.SetEligibilityChecker(&EligibilityChecker{})

	tracker.MarketProposed("asset1", "market1", "me")
	tracker.MarketProposed("asset1", "market2", "me2")

	assert.True(t, tracker.MarketTrackedForAsset("market1", "asset1"))
	assert.False(t, tracker.MarketTrackedForAsset("market1", "asset2"))

	require.Equal(t, false, tracker.IsMarketEligibleForBonus("asset1", "market1", "VEGA", []string{}, "zohar"))
	require.Equal(t, false, tracker.IsMarketEligibleForBonus("asset1", "market2", "VEGA", []string{}, "zohar"))

	tracker.AddValueTraded("asset1", "market1", num.NewUint(1000))
	require.Equal(t, false, tracker.IsMarketEligibleForBonus("asset1", "market1", "VEGA", []string{}, "zohar"))
	require.Equal(t, false, tracker.IsMarketEligibleForBonus("asset1", "market2", "VEGA", []string{}, "zohar"))

	tracker.AddValueTraded("asset1", "market2", num.NewUint(4000))
	require.Equal(t, false, tracker.IsMarketEligibleForBonus("asset1", "market1", "VEGA", []string{}, "zohar"))
	require.Equal(t, false, tracker.IsMarketEligibleForBonus("asset1", "market2", "VEGA", []string{}, "zohar"))

	tracker.AddValueTraded("asset1", "market2", num.NewUint(1001))
	tracker.AddValueTraded("asset1", "market1", num.NewUint(4001))

	require.Equal(t, true, tracker.IsMarketEligibleForBonus("asset1", "market1", "VEGA", []string{}, "zohar"))
	require.Equal(t, true, tracker.IsMarketEligibleForBonus("asset1", "market2", "VEGA", []string{}, "zohar"))

	// mark as paid
	tracker.MarkPaidProposer("asset1", "market1", "VEGA", []string{}, "zohar")
	tracker.MarkPaidProposer("asset1", "market2", "VEGA", []string{}, "zohar")

	// check if eligible for the same combo, expect false
	require.Equal(t, false, tracker.IsMarketEligibleForBonus("asset1", "market1", "VEGA", []string{}, "zohar"))
	require.Equal(t, false, tracker.IsMarketEligibleForBonus("asset1", "market2", "VEGA", []string{}, "zohar"))

	// now check for another funder
	require.Equal(t, true, tracker.IsMarketEligibleForBonus("asset1", "market1", "VEGA", []string{}, "jeremy"))
	require.Equal(t, true, tracker.IsMarketEligibleForBonus("asset1", "market2", "VEGA", []string{}, "jeremy"))

	// mark as paid
	tracker.MarkPaidProposer("asset1", "market1", "VEGA", []string{}, "jeremy")
	tracker.MarkPaidProposer("asset1", "market2", "VEGA", []string{}, "jeremy")

	require.Equal(t, false, tracker.IsMarketEligibleForBonus("asset1", "market1", "VEGA", []string{}, "jeremy"))
	require.Equal(t, false, tracker.IsMarketEligibleForBonus("asset1", "market2", "VEGA", []string{}, "jeremy"))

	// check for another payout asset
	require.Equal(t, true, tracker.IsMarketEligibleForBonus("asset1", "market1", "USDC", []string{}, "zohar"))
	require.Equal(t, true, tracker.IsMarketEligibleForBonus("asset1", "market2", "USDC", []string{}, "zohar"))

	tracker.MarkPaidProposer("asset1", "market1", "USDC", []string{}, "zohar")
	tracker.MarkPaidProposer("asset1", "market2", "USDC", []string{}, "zohar")

	require.Equal(t, false, tracker.IsMarketEligibleForBonus("asset1", "market1", "USDC", []string{}, "zohar"))
	require.Equal(t, false, tracker.IsMarketEligibleForBonus("asset1", "market2", "USDC", []string{}, "zohar"))

	// check for another market scope
	require.Equal(t, true, tracker.IsMarketEligibleForBonus("asset1", "market1", "USDC", []string{"market1"}, "zohar"))
	require.Equal(t, true, tracker.IsMarketEligibleForBonus("asset1", "market2", "USDC", []string{"market2"}, "zohar"))
	require.Equal(t, true, tracker.IsMarketEligibleForBonus("asset1", "market1", "USDC", []string{"market1", "market2"}, "zohar"))
	require.Equal(t, true, tracker.IsMarketEligibleForBonus("asset1", "market2", "USDC", []string{"market2", "market2"}, "zohar"))

	tracker.MarkPaidProposer("asset1", "market1", "USDC", []string{"market1"}, "zohar")
	tracker.MarkPaidProposer("asset1", "market2", "USDC", []string{"market2"}, "zohar")
	tracker.MarkPaidProposer("asset1", "market1", "USDC", []string{"market1", "market2"}, "zohar")
	tracker.MarkPaidProposer("asset1", "market2", "USDC", []string{"market1", "market2"}, "zohar")

	require.Equal(t, false, tracker.IsMarketEligibleForBonus("asset1", "market1", "USDC", []string{"market1"}, "zohar"))
	require.Equal(t, false, tracker.IsMarketEligibleForBonus("asset1", "market2", "USDC", []string{"market2"}, "zohar"))
	require.Equal(t, false, tracker.IsMarketEligibleForBonus("asset1", "market1", "USDC", []string{"market1", "market2"}, "zohar"))
	require.Equal(t, false, tracker.IsMarketEligibleForBonus("asset1", "market2", "USDC", []string{"market1", "market2"}, "zohar"))

	// take a snapshot
	key := (&types.PayloadMarketActivityTracker{}).Key()
	state1, _, err := tracker.GetState(key)
	require.NoError(t, err)
	teams2 := mocks.NewMockTeams(ctrl)
	balanceChecker2 := mocks.NewMockAccountBalanceChecker(ctrl)
	broker = bmocks.NewMockBroker(ctrl)
	trackerLoad := common.NewMarketActivityTracker(logging.NewTestLogger(), teams2, balanceChecker2, broker)
	pl := snapshotpb.Payload{}
	require.NoError(t, proto.Unmarshal(state1, &pl))

	additionalProvider, err := trackerLoad.LoadState(context.Background(), types.PayloadFromProto(&pl))
	require.NoError(t, err)
	assert.Nil(t, additionalProvider)

	state2, _, err := trackerLoad.GetState(key)
	require.NoError(t, err)
	require.True(t, bytes.Equal(state1, state2))
}

func TestRemoveMarket(t *testing.T) {
	epochService := &TestEpochEngine{}
	ctrl := gomock.NewController(t)
	teams := mocks.NewMockTeams(ctrl)
	balanceChecker := mocks.NewMockAccountBalanceChecker(ctrl)

	broker := bmocks.NewMockBroker(ctrl)
	tracker := common.NewMarketActivityTracker(logging.NewTestLogger(), teams, balanceChecker, broker)
	epochService.NotifyOnEpoch(tracker.OnEpochEvent, tracker.OnEpochRestore)
	tracker.SetEligibilityChecker(&EligibilityChecker{})
	tracker.MarketProposed("asset1", "market1", "me")
	tracker.MarketProposed("asset1", "market2", "me2")
	require.Equal(t, 2, len(tracker.GetAllMarketIDs()))
	require.Equal(t, "market1", tracker.GetAllMarketIDs()[0])
	require.Equal(t, "market2", tracker.GetAllMarketIDs()[1])

	// remove the market - this should only mark the market for removal
	tracker.RemoveMarket("asset1", "market1")
	require.Equal(t, 2, len(tracker.GetAllMarketIDs()))
	require.Equal(t, "market1", tracker.GetAllMarketIDs()[0])
	require.Equal(t, "market2", tracker.GetAllMarketIDs()[1])
	epochService.target(context.Background(), types.Epoch{Action: vgproto.EpochAction_EPOCH_ACTION_START})

	require.Equal(t, 1, len(tracker.GetAllMarketIDs()))
	require.Equal(t, "market2", tracker.GetAllMarketIDs()[0])
}

func TestAddRemoveAMM(t *testing.T) {
	epochService := &TestEpochEngine{}
	ctrl := gomock.NewController(t)
	teams := mocks.NewMockTeams(ctrl)
	broker := bmocks.NewMockBroker(ctrl)
	balanceChecker := mocks.NewMockAccountBalanceChecker(ctrl)

	tracker := common.NewMarketActivityTracker(logging.NewTestLogger(), teams, balanceChecker, broker)
	epochService.NotifyOnEpoch(tracker.OnEpochEvent, tracker.OnEpochRestore)
	tracker.SetEligibilityChecker(&EligibilityChecker{})
	tracker.MarketProposed("asset1", "market1", "me")
	tracker.MarketProposed("asset1", "market2", "me2")
	require.Equal(t, 2, len(tracker.GetAllMarketIDs()))
	require.Equal(t, "market1", tracker.GetAllMarketIDs()[0])
	require.Equal(t, "market2", tracker.GetAllMarketIDs()[1])

	tracker.AddAMMSubAccount("asset1", "market1", "sub1")
	tracker.AddAMMSubAccount("asset1", "market1", "sub2")

	require.Equal(t, map[string]struct{}{"sub1": {}, "sub2": {}}, tracker.GetAllAMMParties("asset1", nil))

	tracker.RemoveAMMParty("asset1", "market1", "sub2")
	require.Equal(t, map[string]struct{}{"sub1": {}}, tracker.GetAllAMMParties("asset1", nil))

	tracker.RemoveAMMParty("asset1", "market1", "sub1")
	require.Equal(t, map[string]struct{}{}, tracker.GetAllAMMParties("asset1", nil))
}

func TestGetScores(t *testing.T) {
	ctx := context.Background()
	epochService := &TestEpochEngine{}
	ctrl := gomock.NewController(t)
	teams := mocks.NewMockTeams(ctrl)
	balanceChecker := mocks.NewMockAccountBalanceChecker(ctrl)
	balanceChecker.EXPECT().GetAvailableBalance(gomock.Any()).Return(num.UintZero(), nil).AnyTimes()
	broker := bmocks.NewMockBroker(ctrl)
	broker.EXPECT().SendBatch(gomock.Any()).AnyTimes()
	broker.EXPECT().Send(gomock.Any()).AnyTimes()
	tracker := common.NewMarketActivityTracker(logging.NewTestLogger(), teams, balanceChecker, broker)
	epochService.NotifyOnEpoch(tracker.OnEpochEvent, tracker.OnEpochRestore)
	tracker.SetEligibilityChecker(&EligibilityChecker{})
	tracker.MarketProposed("asset1", "market1", "me")
	tracker.MarketProposed("asset1", "market2", "me2")
	tracker.MarketProposed("asset1", "market4", "me4")
	tracker.MarketProposed("asset2", "market3", "me3")

	// no fees generated expect empty slice
	epochService.target(context.Background(), types.Epoch{Seq: 1, Action: vgproto.EpochAction_EPOCH_ACTION_START})
	epochService.target(context.Background(), types.Epoch{Seq: 1, Action: vgproto.EpochAction_EPOCH_ACTION_END})

	// asset1, asset2 no market scoping

	for _, asset := range []string{"asset1", "asset2"} {
		for _, m := range []vgproto.DispatchMetric{vgproto.DispatchMetric_DISPATCH_METRIC_MAKER_FEES_PAID, vgproto.DispatchMetric_DISPATCH_METRIC_LP_FEES_RECEIVED, vgproto.DispatchMetric_DISPATCH_METRIC_MAKER_FEES_RECEIVED} {
			scores := tracker.CalculateMetricForIndividuals(ctx, &vgproto.DispatchStrategy{AssetForMetric: asset, Metric: m, IndividualScope: vgproto.IndividualScope_INDIVIDUAL_SCOPE_ALL, WindowLength: 1})
			require.Equal(t, 0, len(scores))
		}
	}

	// asset1 one market in scope
	for _, market := range []string{"market1", "market2", "market4"} {
		for _, m := range []vgproto.DispatchMetric{vgproto.DispatchMetric_DISPATCH_METRIC_MAKER_FEES_PAID, vgproto.DispatchMetric_DISPATCH_METRIC_LP_FEES_RECEIVED, vgproto.DispatchMetric_DISPATCH_METRIC_MAKER_FEES_RECEIVED} {
			scores := tracker.CalculateMetricForIndividuals(ctx, &vgproto.DispatchStrategy{AssetForMetric: "asset1", Metric: m, IndividualScope: vgproto.IndividualScope_INDIVIDUAL_SCOPE_ALL, WindowLength: 1, Markets: []string{market}})
			require.Equal(t, 0, len(scores))
		}
	}

	// asset2 one market in scope
	for _, m := range []vgproto.DispatchMetric{vgproto.DispatchMetric_DISPATCH_METRIC_MAKER_FEES_PAID, vgproto.DispatchMetric_DISPATCH_METRIC_LP_FEES_RECEIVED, vgproto.DispatchMetric_DISPATCH_METRIC_MAKER_FEES_RECEIVED} {
		scores := tracker.CalculateMetricForIndividuals(ctx, &vgproto.DispatchStrategy{AssetForMetric: "asset1", Metric: m, IndividualScope: vgproto.IndividualScope_INDIVIDUAL_SCOPE_ALL, WindowLength: 1, Markets: []string{"market3"}})
		require.Equal(t, 0, len(scores))
	}

	epochService.target(context.Background(), types.Epoch{Seq: 2, Action: vgproto.EpochAction_EPOCH_ACTION_START})

	// update with a few transfers
	transfersM1 := []*types.Transfer{
		{Owner: "party1", Type: types.TransferTypeMakerFeeReceive, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.NewUint(100)}},
		{Owner: "party1", Type: types.TransferTypeMakerFeePay, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.NewUint(200)}},
		{Owner: "party1", Type: types.TransferTypeLiquidityFeeNetDistribute, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.NewUint(200)}},
		{Owner: "party1", Type: types.TransferTypeMakerFeeReceive, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.NewUint(400)}},
		{Owner: "party1", Type: types.TransferTypeMakerFeePay, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.NewUint(300)}},
		{Owner: "party1", Type: types.TransferTypeLiquidityFeeNetDistribute, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.NewUint(600)}},
		{Owner: "party2", Type: types.TransferTypeMakerFeeReceive, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.NewUint(900)}},
		{Owner: "party2", Type: types.TransferTypeMakerFeePay, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.NewUint(800)}},
		{Owner: "party2", Type: types.TransferTypeLiquidityFeeNetDistribute, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.NewUint(700)}},
		{Owner: "party2", Type: types.TransferTypeMakerFeeReceive, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.NewUint(600)}},
		{Owner: "party2", Type: types.TransferTypeMakerFeePay, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.NewUint(200)}},
		{Owner: "party2", Type: types.TransferTypeLiquidityFeeNetDistribute, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.NewUint(1000)}},
	}
	tracker.UpdateFeesFromTransfers("asset1", "market1", transfersM1)

	transfersM2 := []*types.Transfer{
		{Owner: "party1", Type: types.TransferTypeMakerFeeReceive, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.NewUint(500)}},
		{Owner: "party2", Type: types.TransferTypeMakerFeePay, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.NewUint(1500)}},
		{Owner: "party2", Type: types.TransferTypeLiquidityFeeNetDistribute, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.NewUint(1500)}},
	}
	tracker.UpdateFeesFromTransfers("asset1", "market2", transfersM2)

	transfersM3 := []*types.Transfer{
		{Owner: "party1", Type: types.TransferTypeMakerFeePay, Amount: &types.FinancialAmount{Asset: "asset2", Amount: num.NewUint(500)}},
		{Owner: "party2", Type: types.TransferTypeMakerFeeReceive, Amount: &types.FinancialAmount{Asset: "asset2", Amount: num.NewUint(450)}},
	}
	tracker.UpdateFeesFromTransfers("asset2", "market3", transfersM3)

	epochService.target(context.Background(), types.Epoch{Seq: 2, Action: vgproto.EpochAction_EPOCH_ACTION_END})

	// looking across all markets in asset 1 with window length 1:
	// party1: 800
	// partt2: 3200
	// total = 4000
	// party1 = 800/4000 = 0.2
	// party2 = 3200/4000 = 0.8
	scores := tracker.CalculateMetricForIndividuals(ctx, &vgproto.DispatchStrategy{AssetForMetric: "asset1", Metric: vgproto.DispatchMetric_DISPATCH_METRIC_LP_FEES_RECEIVED, IndividualScope: vgproto.IndividualScope_INDIVIDUAL_SCOPE_ALL, WindowLength: 1})
	require.Equal(t, 2, len(scores))

	require.Equal(t, "party1", scores[0].Party)
	require.Equal(t, "0.2", scores[0].Score.String())
	require.Equal(t, "party2", scores[1].Party)
	require.Equal(t, "0.8", scores[1].Score.String())

	// now look only on market 1:
	// party1 = 800/2500 = 0.32
	// partt2 = 1700/2500 = 0.68
	scores = tracker.CalculateMetricForIndividuals(ctx, &vgproto.DispatchStrategy{AssetForMetric: "asset1", Metric: vgproto.DispatchMetric_DISPATCH_METRIC_LP_FEES_RECEIVED, IndividualScope: vgproto.IndividualScope_INDIVIDUAL_SCOPE_ALL, WindowLength: 1, Markets: []string{"market1"}})
	require.Equal(t, 2, len(scores))

	require.Equal(t, "party1", scores[0].Party)
	require.Equal(t, "0.32", scores[0].Score.String())
	require.Equal(t, "party2", scores[1].Party)
	require.Equal(t, "0.68", scores[1].Score.String())

	// now look only on market 2:
	scores = tracker.CalculateMetricForIndividuals(ctx, &vgproto.DispatchStrategy{AssetForMetric: "asset1", Metric: vgproto.DispatchMetric_DISPATCH_METRIC_LP_FEES_RECEIVED, IndividualScope: vgproto.IndividualScope_INDIVIDUAL_SCOPE_ALL, WindowLength: 1, Markets: []string{"market2"}})
	require.Equal(t, 2, len(scores))
	require.Equal(t, "party2", scores[1].Party)
	require.Equal(t, "1", scores[1].Score.String())
	require.Equal(t, true, scores[1].IsEligible)
	require.Equal(t, "party1", scores[0].Party)
	require.Equal(t, false, scores[0].IsEligible)
	require.Equal(t, "0", scores[0].Score.String())

	// now look at asset2 with no market qualifer
	scores = tracker.CalculateMetricForIndividuals(ctx, &vgproto.DispatchStrategy{AssetForMetric: "asset2", Metric: vgproto.DispatchMetric_DISPATCH_METRIC_MAKER_FEES_PAID, IndividualScope: vgproto.IndividualScope_INDIVIDUAL_SCOPE_ALL, WindowLength: 1})
	require.Equal(t, 2, len(scores))
	require.Equal(t, "party1", scores[0].Party)
	require.Equal(t, "1", scores[0].Score.String())
	require.Equal(t, true, scores[0].IsEligible)
	require.Equal(t, "party2", scores[1].Party)
	require.Equal(t, "0", scores[1].Score.String())
	require.Equal(t, false, scores[1].IsEligible)

	epochService.target(context.Background(), types.Epoch{Seq: 3, Action: vgproto.EpochAction_EPOCH_ACTION_START})
	transfersM1 = []*types.Transfer{
		{Owner: "party1", Type: types.TransferTypeLiquidityFeeNetDistribute, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.NewUint(1200)}},
	}
	transfersM2 = []*types.Transfer{
		{Owner: "party2", Type: types.TransferTypeLiquidityFeeNetDistribute, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.NewUint(800)}},
	}
	tracker.UpdateFeesFromTransfers("asset1", "market1", transfersM1)
	tracker.UpdateFeesFromTransfers("asset1", "market2", transfersM2)
	epochService.target(context.Background(), types.Epoch{Seq: 3, Action: vgproto.EpochAction_EPOCH_ACTION_END})

	// looking across all markets in asset 1 with window length 2:
	// party1: 800 + 1200 = 2000
	// partt2: 3200 + 800 = 4000
	// total = 4000 + 2000 = 6000
	// party1 = 2000/6000 = 1/3
	// party2 = 4000/6000 = 2/3
	scores = tracker.CalculateMetricForIndividuals(ctx, &vgproto.DispatchStrategy{AssetForMetric: "asset1", Metric: vgproto.DispatchMetric_DISPATCH_METRIC_LP_FEES_RECEIVED, IndividualScope: vgproto.IndividualScope_INDIVIDUAL_SCOPE_ALL, WindowLength: 2})
	require.Equal(t, 2, len(scores))

	require.Equal(t, "party1", scores[0].Party)
	require.Equal(t, "0.3333333333333333", scores[0].Score.String())
	require.Equal(t, "party2", scores[1].Party)
	require.Equal(t, "0.6666666666666667", scores[1].Score.String())
}

func TestGetScoresIndividualsDifferentScopes(t *testing.T) {
	ctx := context.Background()
	epochService := &TestEpochEngine{}
	ctrl := gomock.NewController(t)
	teams := mocks.NewMockTeams(ctrl)
	balanceChecker := mocks.NewMockAccountBalanceChecker(ctrl)
	balanceChecker.EXPECT().GetAvailableBalance(gomock.Any()).Return(num.UintZero(), nil).AnyTimes()
	broker := bmocks.NewMockBroker(ctrl)
	broker.EXPECT().SendBatch(gomock.Any()).AnyTimes()
	broker.EXPECT().Send(gomock.Any()).AnyTimes()
	tracker := common.NewMarketActivityTracker(logging.NewTestLogger(), teams, balanceChecker, broker)
	epochService.NotifyOnEpoch(tracker.OnEpochEvent, tracker.OnEpochRestore)
	tracker.SetEligibilityChecker(&EligibilityChecker{})
	tracker.MarketProposed("asset1", "market1", "me")
	tracker.MarketProposed("asset1", "market2", "me2")
	tracker.MarketProposed("asset1", "market4", "me4")
	tracker.MarketProposed("asset2", "market3", "me3")

	// no fees generated expect empty slice
	epochService.target(context.Background(), types.Epoch{Seq: 1, Action: vgproto.EpochAction_EPOCH_ACTION_START})
	epochService.target(context.Background(), types.Epoch{Seq: 1, Action: vgproto.EpochAction_EPOCH_ACTION_END})

	// asset1, asset2 no market scoping

	for _, asset := range []string{"asset1", "asset2"} {
		for _, m := range []vgproto.DispatchMetric{vgproto.DispatchMetric_DISPATCH_METRIC_MAKER_FEES_PAID, vgproto.DispatchMetric_DISPATCH_METRIC_LP_FEES_RECEIVED, vgproto.DispatchMetric_DISPATCH_METRIC_MAKER_FEES_RECEIVED} {
			scores := tracker.CalculateMetricForIndividuals(ctx, &vgproto.DispatchStrategy{AssetForMetric: asset, Metric: m, IndividualScope: vgproto.IndividualScope_INDIVIDUAL_SCOPE_ALL, WindowLength: 1})
			require.Equal(t, 0, len(scores))
		}
	}

	// asset1 one market in scope
	for _, market := range []string{"market1", "market2", "market4"} {
		for _, m := range []vgproto.DispatchMetric{vgproto.DispatchMetric_DISPATCH_METRIC_MAKER_FEES_PAID, vgproto.DispatchMetric_DISPATCH_METRIC_LP_FEES_RECEIVED, vgproto.DispatchMetric_DISPATCH_METRIC_MAKER_FEES_RECEIVED} {
			scores := tracker.CalculateMetricForIndividuals(ctx, &vgproto.DispatchStrategy{AssetForMetric: "asset1", Metric: m, IndividualScope: vgproto.IndividualScope_INDIVIDUAL_SCOPE_ALL, WindowLength: 1, Markets: []string{market}})
			require.Equal(t, 0, len(scores))
		}
	}

	// asset2 one market in scope
	for _, m := range []vgproto.DispatchMetric{vgproto.DispatchMetric_DISPATCH_METRIC_MAKER_FEES_PAID, vgproto.DispatchMetric_DISPATCH_METRIC_LP_FEES_RECEIVED, vgproto.DispatchMetric_DISPATCH_METRIC_MAKER_FEES_RECEIVED} {
		scores := tracker.CalculateMetricForIndividuals(ctx, &vgproto.DispatchStrategy{AssetForMetric: "asset1", Metric: m, IndividualScope: vgproto.IndividualScope_INDIVIDUAL_SCOPE_ALL, WindowLength: 1, Markets: []string{"market3"}})
		require.Equal(t, 0, len(scores))
	}

	epochService.target(context.Background(), types.Epoch{Seq: 2, Action: vgproto.EpochAction_EPOCH_ACTION_START})

	tracker.AddAMMSubAccount("asset1", "market1", "party1")

	// update with a few transfers
	transfersM1 := []*types.Transfer{
		{Owner: "party1", Type: types.TransferTypeMakerFeeReceive, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.NewUint(100)}},
		{Owner: "party1", Type: types.TransferTypeMakerFeePay, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.NewUint(200)}},
		{Owner: "party1", Type: types.TransferTypeLiquidityFeeNetDistribute, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.NewUint(200)}},
		{Owner: "party1", Type: types.TransferTypeMakerFeeReceive, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.NewUint(400)}},
		{Owner: "party1", Type: types.TransferTypeMakerFeePay, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.NewUint(300)}},
		{Owner: "party1", Type: types.TransferTypeLiquidityFeeNetDistribute, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.NewUint(600)}},
		{Owner: "party2", Type: types.TransferTypeMakerFeeReceive, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.NewUint(900)}},
		{Owner: "party2", Type: types.TransferTypeMakerFeePay, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.NewUint(800)}},
		{Owner: "party2", Type: types.TransferTypeLiquidityFeeNetDistribute, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.NewUint(700)}},
		{Owner: "party2", Type: types.TransferTypeMakerFeeReceive, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.NewUint(600)}},
		{Owner: "party2", Type: types.TransferTypeMakerFeePay, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.NewUint(200)}},
		{Owner: "party2", Type: types.TransferTypeLiquidityFeeNetDistribute, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.NewUint(1000)}},
	}
	tracker.UpdateFeesFromTransfers("asset1", "market1", transfersM1)

	transfersM2 := []*types.Transfer{
		{Owner: "party1", Type: types.TransferTypeMakerFeeReceive, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.NewUint(500)}},
		{Owner: "party2", Type: types.TransferTypeMakerFeePay, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.NewUint(1500)}},
		{Owner: "party2", Type: types.TransferTypeLiquidityFeeNetDistribute, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.NewUint(1500)}},
	}
	tracker.UpdateFeesFromTransfers("asset1", "market2", transfersM2)

	transfersM3 := []*types.Transfer{
		{Owner: "party1", Type: types.TransferTypeMakerFeePay, Amount: &types.FinancialAmount{Asset: "asset2", Amount: num.NewUint(500)}},
		{Owner: "party2", Type: types.TransferTypeMakerFeeReceive, Amount: &types.FinancialAmount{Asset: "asset2", Amount: num.NewUint(450)}},
	}
	tracker.UpdateFeesFromTransfers("asset2", "market3", transfersM3)

	epochService.target(context.Background(), types.Epoch{Seq: 2, Action: vgproto.EpochAction_EPOCH_ACTION_END})

	// looking across all markets in asset 1 with window length 1:
	// party1: 800
	// partt2: 3200
	// total = 4000
	// party1 = 800/4000 = 0.2
	// party2 = 3200/4000 = 0.8
	scores := tracker.CalculateMetricForIndividuals(ctx, &vgproto.DispatchStrategy{AssetForMetric: "asset1", Metric: vgproto.DispatchMetric_DISPATCH_METRIC_LP_FEES_RECEIVED, IndividualScope: vgproto.IndividualScope_INDIVIDUAL_SCOPE_ALL, WindowLength: 1})
	require.Equal(t, 2, len(scores))

	require.Equal(t, "party1", scores[0].Party)
	require.Equal(t, "0.2", scores[0].Score.String())
	require.Equal(t, "party2", scores[1].Party)
	require.Equal(t, "0.8", scores[1].Score.String())

	// looking across all markets in asset 1 with window length 1 and AMM scope:
	// party1: 800
	// partt2: 3200
	// total = 4000
	// party1 = 800/4000 = 0.2
	scores = tracker.CalculateMetricForIndividuals(ctx, &vgproto.DispatchStrategy{AssetForMetric: "asset1", Metric: vgproto.DispatchMetric_DISPATCH_METRIC_LP_FEES_RECEIVED, IndividualScope: vgproto.IndividualScope_INDIVIDUAL_SCOPE_AMM, WindowLength: 1})
	require.Equal(t, 1, len(scores))

	require.Equal(t, "party1", scores[0].Party)
	require.Equal(t, "0.2", scores[0].Score.String())

	// now look only on market 1:
	// party1 = 800/2500 = 0.32
	// partt2 = 1700/2500 = 0.68
	scores = tracker.CalculateMetricForIndividuals(ctx, &vgproto.DispatchStrategy{AssetForMetric: "asset1", Metric: vgproto.DispatchMetric_DISPATCH_METRIC_LP_FEES_RECEIVED, IndividualScope: vgproto.IndividualScope_INDIVIDUAL_SCOPE_ALL, WindowLength: 1, Markets: []string{"market1"}})
	require.Equal(t, 2, len(scores))

	require.Equal(t, "party1", scores[0].Party)
	require.Equal(t, "0.32", scores[0].Score.String())
	require.Equal(t, "party2", scores[1].Party)
	require.Equal(t, "0.68", scores[1].Score.String())

	// now look only on market 2:
	scores = tracker.CalculateMetricForIndividuals(ctx, &vgproto.DispatchStrategy{AssetForMetric: "asset1", Metric: vgproto.DispatchMetric_DISPATCH_METRIC_LP_FEES_RECEIVED, IndividualScope: vgproto.IndividualScope_INDIVIDUAL_SCOPE_ALL, WindowLength: 1, Markets: []string{"market2"}})
	require.Equal(t, 2, len(scores))

	require.Equal(t, "party1", scores[0].Party)
	require.Equal(t, "0", scores[0].Score.String())
	require.Equal(t, false, scores[0].IsEligible)
	require.Equal(t, "party2", scores[1].Party)
	require.Equal(t, "1", scores[1].Score.String())
	require.Equal(t, true, scores[1].IsEligible)

	// now look at asset2 with no market qualifer
	scores = tracker.CalculateMetricForIndividuals(ctx, &vgproto.DispatchStrategy{AssetForMetric: "asset2", Metric: vgproto.DispatchMetric_DISPATCH_METRIC_MAKER_FEES_PAID, IndividualScope: vgproto.IndividualScope_INDIVIDUAL_SCOPE_ALL, WindowLength: 1})
	require.Equal(t, 2, len(scores))
	require.Equal(t, "party1", scores[0].Party)
	require.Equal(t, true, scores[0].IsEligible)
	require.Equal(t, "1", scores[0].Score.String())
	require.Equal(t, "party2", scores[1].Party)
	require.Equal(t, "0", scores[1].Score.String())
	require.Equal(t, false, scores[1].IsEligible)

	epochService.target(context.Background(), types.Epoch{Seq: 3, Action: vgproto.EpochAction_EPOCH_ACTION_START})
	transfersM1 = []*types.Transfer{
		{Owner: "party1", Type: types.TransferTypeLiquidityFeeNetDistribute, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.NewUint(1200)}},
	}
	transfersM2 = []*types.Transfer{
		{Owner: "party2", Type: types.TransferTypeLiquidityFeeNetDistribute, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.NewUint(800)}},
	}
	tracker.UpdateFeesFromTransfers("asset1", "market1", transfersM1)
	tracker.UpdateFeesFromTransfers("asset1", "market2", transfersM2)
	epochService.target(context.Background(), types.Epoch{Seq: 3, Action: vgproto.EpochAction_EPOCH_ACTION_END})

	// looking across all markets in asset 1 with window length 2:
	// party1: 800 + 1200 = 2000
	// partt2: 3200 + 800 = 4000
	// total = 4000 + 2000 = 6000
	// party1 = 2000/6000 = 1/3
	// party2 = 4000/6000 = 2/3
	scores = tracker.CalculateMetricForIndividuals(ctx, &vgproto.DispatchStrategy{AssetForMetric: "asset1", Metric: vgproto.DispatchMetric_DISPATCH_METRIC_LP_FEES_RECEIVED, IndividualScope: vgproto.IndividualScope_INDIVIDUAL_SCOPE_ALL, WindowLength: 2})
	require.Equal(t, 2, len(scores))

	require.Equal(t, "party1", scores[0].Party)
	require.Equal(t, "0.3333333333333333", scores[0].Score.String())
	require.Equal(t, "party2", scores[1].Party)
	require.Equal(t, "0.6666666666666667", scores[1].Score.String())
}

func TestMarketTrackerStateChange(t *testing.T) {
	key := (&types.PayloadMarketActivityTracker{}).Key()

	ctrl := gomock.NewController(t)
	teams := mocks.NewMockTeams(ctrl)
	balanceChecker := mocks.NewMockAccountBalanceChecker(ctrl)
	broker := bmocks.NewMockBroker(ctrl)
	broker.EXPECT().SendBatch(gomock.Any()).AnyTimes()
	tracker := common.NewMarketActivityTracker(logging.NewTestLogger(), teams, balanceChecker, broker)
	tracker.SetEligibilityChecker(&EligibilityChecker{})

	state1, _, err := tracker.GetState(key)
	require.NoError(t, err)

	tracker.MarketProposed("asset1", "market1", "me")
	tracker.MarketProposed("asset1", "market2", "me2")

	state2, _, err := tracker.GetState(key)
	require.NoError(t, err)
	require.False(t, bytes.Equal(state1, state2))

	tracker.AddValueTraded("asset1", "market1", num.NewUint(1000))
	require.False(t, tracker.IsMarketEligibleForBonus("asset1", "market1", "VEGA", []string{}, "zohar"))
	require.False(t, tracker.IsMarketEligibleForBonus("asset1", "market2", "VEGA", []string{}, "zohar"))

	state3, _, err := tracker.GetState(key)
	require.NoError(t, err)
	require.False(t, bytes.Equal(state1, state3))
}

func TestFeesTrackerWith0(t *testing.T) {
	epochEngine := &TestEpochEngine{}
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	teams := mocks.NewMockTeams(ctrl)
	balanceChecker := mocks.NewMockAccountBalanceChecker(ctrl)
	balanceChecker.EXPECT().GetAvailableBalance(gomock.Any()).Return(num.UintZero(), nil).AnyTimes()
	broker := bmocks.NewMockBroker(ctrl)
	broker.EXPECT().Send(gomock.Any()).AnyTimes()
	tracker := common.NewMarketActivityTracker(logging.NewTestLogger(), teams, balanceChecker, broker)
	epochEngine.NotifyOnEpoch(tracker.OnEpochEvent, tracker.OnEpochRestore)
	epochEngine.target(context.Background(), types.Epoch{Seq: 1, Action: vgproto.EpochAction_EPOCH_ACTION_START})

	tracker.MarketProposed("asset1", "market1", "me")
	transfersM1 := []*types.Transfer{
		{Owner: "party1", Type: types.TransferTypeMakerFeeReceive, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.UintZero()}},
		{Owner: "party1", Type: types.TransferTypeMakerFeePay, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.UintZero()}},
		{Owner: "party1", Type: types.TransferTypeLiquidityFeeNetDistribute, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.UintZero()}},
		{Owner: "party1", Type: types.TransferTypeMakerFeeReceive, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.UintZero()}},
		{Owner: "party1", Type: types.TransferTypeMakerFeePay, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.UintZero()}},
		{Owner: "party1", Type: types.TransferTypeLiquidityFeeNetDistribute, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.UintZero()}},
		{Owner: "party2", Type: types.TransferTypeMakerFeeReceive, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.UintZero()}},
		{Owner: "party2", Type: types.TransferTypeMakerFeePay, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.UintZero()}},
		{Owner: "party2", Type: types.TransferTypeLiquidityFeeNetDistribute, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.UintZero()}},
		{Owner: "party2", Type: types.TransferTypeMakerFeeReceive, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.UintZero()}},
		{Owner: "party2", Type: types.TransferTypeMakerFeePay, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.UintZero()}},
		{Owner: "party2", Type: types.TransferTypeLiquidityFeeNetDistribute, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.UintZero()}},
	}
	tracker.UpdateFeesFromTransfers("asset1", "market1", transfersM1)
	epochEngine.target(context.Background(), types.Epoch{Seq: 1, Action: vgproto.EpochAction_EPOCH_ACTION_END})
	scores := tracker.CalculateMetricForIndividuals(ctx, &vgproto.DispatchStrategy{AssetForMetric: "asset1", Metric: vgproto.DispatchMetric_DISPATCH_METRIC_MAKER_FEES_RECEIVED, IndividualScope: vgproto.IndividualScope_INDIVIDUAL_SCOPE_ALL, WindowLength: 1, Markets: []string{"market1"}})
	require.Equal(t, 2, len(scores))
	require.Equal(t, false, scores[0].IsEligible)
	require.Equal(t, false, scores[1].IsEligible)
}

func TestGetLastEpochTakeFees(t *testing.T) {
	epochEngine := &TestEpochEngine{}
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	teams := mocks.NewMockTeams(ctrl)
	balanceChecker := mocks.NewMockAccountBalanceChecker(ctrl)
	broker := bmocks.NewMockBroker(ctrl)
	tracker := common.NewMarketActivityTracker(logging.NewTestLogger(), teams, balanceChecker, broker)
	epochEngine.NotifyOnEpoch(tracker.OnEpochEvent, tracker.OnEpochRestore)
	epochEngine.target(context.Background(), types.Epoch{Seq: 1, Action: vgproto.EpochAction_EPOCH_ACTION_START})
	tracker.SetEligibilityChecker(&EligibilityChecker{})

	partyScores := tracker.CalculateMetricForIndividuals(ctx, &vgproto.DispatchStrategy{AssetForMetric: "a1", Metric: vgproto.DispatchMetric_DISPATCH_METRIC_MAKER_FEES_RECEIVED, IndividualScope: vgproto.IndividualScope_INDIVIDUAL_SCOPE_ALL, WindowLength: 3})
	require.Equal(t, 0, len(partyScores))

	tracker.MarketProposed("asset1", "market1", "me")
	tracker.MarketProposed("asset1", "market2", "me2")

	// update with a few transfers
	transfersM1 := []*types.Transfer{
		{Owner: "party1", Type: types.TransferTypeMakerFeeReceive, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.NewUint(100)}},
		{Owner: "party1", Type: types.TransferTypeMakerFeePay, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.NewUint(200)}},
		{Owner: "party1", Type: types.TransferTypeInfrastructureFeePay, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.NewUint(110)}},
		{Owner: "party1", Type: types.TransferTypeLiquidityFeePay, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.NewUint(10)}},
		{Owner: "party1", Type: types.TransferTypeLiquidityFeeNetDistribute, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.NewUint(200)}},
		{Owner: "party1", Type: types.TransferTypeMakerFeeReceive, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.NewUint(400)}},
		{Owner: "party1", Type: types.TransferTypeMakerFeePay, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.NewUint(300)}},
		{Owner: "party1", Type: types.TransferTypeLiquidityFeeNetDistribute, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.NewUint(600)}},
		{Owner: "party2", Type: types.TransferTypeMakerFeeReceive, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.NewUint(900)}},
		{Owner: "party2", Type: types.TransferTypeMakerFeePay, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.NewUint(800)}},
		{Owner: "party2", Type: types.TransferTypeLiquidityFeeNetDistribute, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.NewUint(700)}},
		{Owner: "party2", Type: types.TransferTypeMakerFeeReceive, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.NewUint(600)}},
		{Owner: "party2", Type: types.TransferTypeMakerFeePay, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.NewUint(200)}},
		{Owner: "party2", Type: types.TransferTypeLiquidityFeeNetDistribute, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.NewUint(1000)}},
	}
	tracker.UpdateFeesFromTransfers("asset1", "market1", transfersM1)

	transfersM2 := []*types.Transfer{
		{Owner: "party1", Type: types.TransferTypeMakerFeeReceive, Amount: &types.FinancialAmount{Asset: "asset2", Amount: num.NewUint(150)}},
		{Owner: "party2", Type: types.TransferTypeMakerFeePay, Amount: &types.FinancialAmount{Asset: "asset2", Amount: num.NewUint(150)}},
	}
	tracker.UpdateFeesFromTransfers("asset1", "market2", transfersM2)

	epochEngine.target(context.Background(), types.Epoch{Seq: 1, Action: vgproto.EpochAction_EPOCH_ACTION_END})

	m1 := tracker.GetLastEpochTakeFees("asset1", []string{"market1"}, 1)
	require.Equal(t, 2, len(m1))
	require.Equal(t, "620", m1["party1"].String())
	require.Equal(t, "1000", m1["party2"].String())
	m2 := tracker.GetLastEpochTakeFees("asset1", []string{"market2"}, 1)
	require.Equal(t, 1, len(m2))
	require.Equal(t, "150", m2["party2"].String())

	mAll := tracker.GetLastEpochTakeFees("asset1", []string{"market1", "market2"}, 1)
	require.Equal(t, "620", mAll["party1"].String())
	require.Equal(t, "1150", mAll["party2"].String())

	mNoMarkets := tracker.GetLastEpochTakeFees("asset1", []string{}, 1)
	require.Equal(t, "620", mNoMarkets["party1"].String())
	require.Equal(t, "1150", mNoMarkets["party2"].String())

	require.Equal(t, mAll, mNoMarkets)
}

func TestGetLastEpochTakeFeesMultiEpochWindow(t *testing.T) {
	epochEngine := &TestEpochEngine{}
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	teams := mocks.NewMockTeams(ctrl)
	balanceChecker := mocks.NewMockAccountBalanceChecker(ctrl)
	broker := bmocks.NewMockBroker(ctrl)
	tracker := common.NewMarketActivityTracker(logging.NewTestLogger(), teams, balanceChecker, broker)
	epochEngine.NotifyOnEpoch(tracker.OnEpochEvent, tracker.OnEpochRestore)
	epochEngine.target(context.Background(), types.Epoch{Seq: 1, Action: vgproto.EpochAction_EPOCH_ACTION_START})
	tracker.SetEligibilityChecker(&EligibilityChecker{})

	partyScores := tracker.CalculateMetricForIndividuals(ctx, &vgproto.DispatchStrategy{AssetForMetric: "a1", Metric: vgproto.DispatchMetric_DISPATCH_METRIC_MAKER_FEES_RECEIVED, IndividualScope: vgproto.IndividualScope_INDIVIDUAL_SCOPE_ALL, WindowLength: 3})
	require.Equal(t, 0, len(partyScores))

	tracker.MarketProposed("asset1", "market1", "me")
	tracker.MarketProposed("asset1", "market2", "me2")

	// update with a few transfers
	transfersM1 := []*types.Transfer{
		{Owner: "party1", Type: types.TransferTypeMakerFeeReceive, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.NewUint(100)}},
		{Owner: "party1", Type: types.TransferTypeMakerFeePay, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.NewUint(200)}},
		{Owner: "party1", Type: types.TransferTypeInfrastructureFeePay, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.NewUint(110)}},
		{Owner: "party1", Type: types.TransferTypeLiquidityFeePay, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.NewUint(10)}},
		{Owner: "party1", Type: types.TransferTypeLiquidityFeeNetDistribute, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.NewUint(200)}},
		{Owner: "party1", Type: types.TransferTypeMakerFeeReceive, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.NewUint(400)}},
		{Owner: "party1", Type: types.TransferTypeMakerFeePay, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.NewUint(300)}},
		{Owner: "party1", Type: types.TransferTypeLiquidityFeeNetDistribute, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.NewUint(600)}},
		{Owner: "party2", Type: types.TransferTypeMakerFeeReceive, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.NewUint(900)}},
		{Owner: "party2", Type: types.TransferTypeMakerFeePay, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.NewUint(800)}},
		{Owner: "party2", Type: types.TransferTypeLiquidityFeeNetDistribute, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.NewUint(700)}},
		{Owner: "party2", Type: types.TransferTypeMakerFeeReceive, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.NewUint(600)}},
		{Owner: "party2", Type: types.TransferTypeMakerFeePay, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.NewUint(200)}},
		{Owner: "party2", Type: types.TransferTypeLiquidityFeeNetDistribute, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.NewUint(1000)}},
	}
	tracker.UpdateFeesFromTransfers("asset1", "market1", transfersM1)

	transfersM2 := []*types.Transfer{
		{Owner: "party1", Type: types.TransferTypeMakerFeeReceive, Amount: &types.FinancialAmount{Asset: "asset2", Amount: num.NewUint(150)}},
		{Owner: "party2", Type: types.TransferTypeMakerFeePay, Amount: &types.FinancialAmount{Asset: "asset2", Amount: num.NewUint(150)}},
	}
	tracker.UpdateFeesFromTransfers("asset1", "market2", transfersM2)

	epochEngine.target(context.Background(), types.Epoch{Seq: 1, Action: vgproto.EpochAction_EPOCH_ACTION_END})
	epochEngine.target(context.Background(), types.Epoch{Seq: 2, Action: vgproto.EpochAction_EPOCH_ACTION_START})

	// double the fees for a window of 2
	tracker.UpdateFeesFromTransfers("asset1", "market1", transfersM1)
	tracker.UpdateFeesFromTransfers("asset1", "market2", transfersM2)

	epochEngine.target(context.Background(), types.Epoch{Seq: 2, Action: vgproto.EpochAction_EPOCH_ACTION_END})
	m1 := tracker.GetLastEpochTakeFees("asset1", []string{"market1"}, 1)
	require.Equal(t, 2, len(m1))
	require.Equal(t, "620", m1["party1"].String())
	require.Equal(t, "1000", m1["party2"].String())

	m1 = tracker.GetLastEpochTakeFees("asset1", []string{"market1"}, 2)
	require.Equal(t, 2, len(m1))
	require.Equal(t, "1240", m1["party1"].String())
	require.Equal(t, "2000", m1["party2"].String())

	m2 := tracker.GetLastEpochTakeFees("asset1", []string{"market2"}, 1)
	require.Equal(t, 1, len(m2))
	require.Equal(t, "150", m2["party2"].String())

	m2 = tracker.GetLastEpochTakeFees("asset1", []string{"market2"}, 2)
	require.Equal(t, 1, len(m2))
	require.Equal(t, "300", m2["party2"].String())

	mAll := tracker.GetLastEpochTakeFees("asset1", []string{"market1", "market2"}, 1)
	require.Equal(t, "620", mAll["party1"].String())
	require.Equal(t, "1150", mAll["party2"].String())

	mAll = tracker.GetLastEpochTakeFees("asset1", []string{"market1", "market2"}, 2)
	require.Equal(t, "1240", mAll["party1"].String())
	require.Equal(t, "2300", mAll["party2"].String())

	mNoMarkets := tracker.GetLastEpochTakeFees("asset1", []string{}, 1)
	require.Equal(t, "620", mNoMarkets["party1"].String())
	require.Equal(t, "1150", mNoMarkets["party2"].String())

	mNoMarkets = tracker.GetLastEpochTakeFees("asset1", []string{}, 2)
	require.Equal(t, "1240", mNoMarkets["party1"].String())
	require.Equal(t, "2300", mNoMarkets["party2"].String())
	require.Equal(t, mAll, mNoMarkets)
}

func TestFeesTracker(t *testing.T) {
	epochEngine := &TestEpochEngine{}
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	teams := mocks.NewMockTeams(ctrl)
	balanceChecker := mocks.NewMockAccountBalanceChecker(ctrl)
	balanceChecker.EXPECT().GetAvailableBalance(gomock.Any()).Return(num.UintZero(), nil).AnyTimes()
	broker := bmocks.NewMockBroker(ctrl)
	broker.EXPECT().Send(gomock.Any()).AnyTimes()
	broker.EXPECT().SendBatch(gomock.Any()).AnyTimes()
	tracker := common.NewMarketActivityTracker(logging.NewTestLogger(), teams, balanceChecker, broker)
	epochEngine.NotifyOnEpoch(tracker.OnEpochEvent, tracker.OnEpochRestore)
	epochEngine.target(context.Background(), types.Epoch{Seq: 1, Action: vgproto.EpochAction_EPOCH_ACTION_START})
	tracker.SetEligibilityChecker(&EligibilityChecker{})

	partyScores := tracker.CalculateMetricForIndividuals(ctx, &vgproto.DispatchStrategy{AssetForMetric: "a1", Metric: vgproto.DispatchMetric_DISPATCH_METRIC_MAKER_FEES_RECEIVED, IndividualScope: vgproto.IndividualScope_INDIVIDUAL_SCOPE_ALL, WindowLength: 3})
	require.Equal(t, 0, len(partyScores))

	key := (&types.PayloadMarketActivityTracker{}).Key()
	state1, _, err := tracker.GetState(key)
	require.NoError(t, err)

	tracker.MarketProposed("asset1", "market1", "me")
	tracker.MarketProposed("asset1", "market2", "me2")

	// update with a few transfers
	transfersM1 := []*types.Transfer{
		{Owner: "party1", Type: types.TransferTypeMakerFeeReceive, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.NewUint(100)}},
		{Owner: "party1", Type: types.TransferTypeMakerFeePay, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.NewUint(200)}},
		{Owner: "party1", Type: types.TransferTypeLiquidityFeeNetDistribute, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.NewUint(200)}},
		{Owner: "party1", Type: types.TransferTypeMakerFeeReceive, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.NewUint(400)}},
		{Owner: "party1", Type: types.TransferTypeMakerFeePay, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.NewUint(300)}},
		{Owner: "party1", Type: types.TransferTypeLiquidityFeeNetDistribute, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.NewUint(600)}},
		{Owner: "party2", Type: types.TransferTypeMakerFeeReceive, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.NewUint(900)}},
		{Owner: "party2", Type: types.TransferTypeMakerFeePay, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.NewUint(800)}},
		{Owner: "party2", Type: types.TransferTypeLiquidityFeeNetDistribute, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.NewUint(700)}},
		{Owner: "party2", Type: types.TransferTypeMakerFeeReceive, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.NewUint(600)}},
		{Owner: "party2", Type: types.TransferTypeMakerFeePay, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.NewUint(200)}},
		{Owner: "party2", Type: types.TransferTypeLiquidityFeeNetDistribute, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.NewUint(1000)}},
	}
	tracker.UpdateFeesFromTransfers("asset1", "market1", transfersM1)

	transfersM2 := []*types.Transfer{
		{Owner: "party1", Type: types.TransferTypeMakerFeeReceive, Amount: &types.FinancialAmount{Asset: "asset2", Amount: num.NewUint(150)}},
		{Owner: "party2", Type: types.TransferTypeMakerFeePay, Amount: &types.FinancialAmount{Asset: "asset2", Amount: num.NewUint(150)}},
	}
	tracker.UpdateFeesFromTransfers("asset1", "market2", transfersM2)

	epochEngine.target(context.Background(), types.Epoch{Seq: 1, Action: vgproto.EpochAction_EPOCH_ACTION_END})
	epochEngine.target(context.Background(), types.Epoch{Seq: 2, Action: vgproto.EpochAction_EPOCH_ACTION_START})

	// asset1, types.TransferTypeMakerFeeReceive
	// party1 received 500
	// party2 received 1500
	scores := tracker.CalculateMetricForIndividuals(ctx, &vgproto.DispatchStrategy{AssetForMetric: "asset1", Metric: vgproto.DispatchMetric_DISPATCH_METRIC_MAKER_FEES_RECEIVED, IndividualScope: vgproto.IndividualScope_INDIVIDUAL_SCOPE_ALL, WindowLength: 1, Markets: []string{"market1"}})
	require.Equal(t, 2, len(scores))
	require.Equal(t, "0.25", scores[0].Score.String())
	require.Equal(t, "party1", scores[0].Party)
	require.Equal(t, "0.75", scores[1].Score.String())
	require.Equal(t, "party2", scores[1].Party)

	// asset1 TransferTypeMakerFeePay
	// party1 paid 500
	// party2 paid 1000
	scores = tracker.CalculateMetricForIndividuals(ctx, &vgproto.DispatchStrategy{AssetForMetric: "asset1", Metric: vgproto.DispatchMetric_DISPATCH_METRIC_MAKER_FEES_PAID, IndividualScope: vgproto.IndividualScope_INDIVIDUAL_SCOPE_ALL, WindowLength: 1, Markets: []string{"market1"}})
	require.Equal(t, 2, len(scores))
	require.Equal(t, "0.3333333333333333", scores[0].Score.String())
	require.Equal(t, "party1", scores[0].Party)
	require.Equal(t, "0.6666666666666667", scores[1].Score.String())
	require.Equal(t, "party2", scores[1].Party)

	// asset1 TransferTypeLiquidityFeeNetDistribute
	// party1 paid 800
	// party2 paid 1700
	scores = tracker.CalculateMetricForIndividuals(ctx, &vgproto.DispatchStrategy{AssetForMetric: "asset1", Metric: vgproto.DispatchMetric_DISPATCH_METRIC_LP_FEES_RECEIVED, IndividualScope: vgproto.IndividualScope_INDIVIDUAL_SCOPE_ALL, WindowLength: 1, Markets: []string{"market1"}})
	require.Equal(t, 2, len(scores))
	require.Equal(t, "0.32", scores[0].Score.String())
	require.Equal(t, "party1", scores[0].Party)
	require.Equal(t, "0.68", scores[1].Score.String())
	require.Equal(t, "party2", scores[1].Party)

	// asset2 TransferTypeMakerFeePay
	scores = tracker.CalculateMetricForIndividuals(ctx, &vgproto.DispatchStrategy{AssetForMetric: "asset1", Metric: vgproto.DispatchMetric_DISPATCH_METRIC_MAKER_FEES_RECEIVED, IndividualScope: vgproto.IndividualScope_INDIVIDUAL_SCOPE_ALL, WindowLength: 1, Markets: []string{"market2"}})
	require.Equal(t, 2, len(scores))
	require.Equal(t, "1", scores[0].Score.String())
	require.Equal(t, "party1", scores[0].Party)
	require.Equal(t, true, scores[0].IsEligible)
	require.Equal(t, "0", scores[1].Score.String())
	require.Equal(t, "party2", scores[1].Party)
	require.Equal(t, false, scores[1].IsEligible)

	// asset2 TransferTypeMakerFeePay
	scores = tracker.CalculateMetricForIndividuals(ctx, &vgproto.DispatchStrategy{AssetForMetric: "asset1", Metric: vgproto.DispatchMetric_DISPATCH_METRIC_MAKER_FEES_PAID, IndividualScope: vgproto.IndividualScope_INDIVIDUAL_SCOPE_ALL, WindowLength: 1, Markets: []string{"market2"}})
	require.Equal(t, 2, len(scores))
	require.Equal(t, "0", scores[0].Score.String())
	require.Equal(t, "party1", scores[0].Party)
	require.Equal(t, false, scores[0].IsEligible)
	require.Equal(t, "1", scores[1].Score.String())
	require.Equal(t, "party2", scores[1].Party)
	require.Equal(t, true, scores[1].IsEligible)

	// check state has changed
	state2, _, err := tracker.GetState(key)
	require.NoError(t, err)
	require.False(t, bytes.Equal(state1, state2))

	epochEngineLoad := &TestEpochEngine{}
	ctrl = gomock.NewController(t)
	teams = mocks.NewMockTeams(ctrl)
	balanceChecker = mocks.NewMockAccountBalanceChecker(ctrl)
	balanceChecker.EXPECT().GetAvailableBalance(gomock.Any()).Return(num.UintZero(), nil).AnyTimes()
	broker.EXPECT().Send(gomock.Any()).AnyTimes()
	trackerLoad := common.NewMarketActivityTracker(logging.NewTestLogger(), teams, balanceChecker, broker)
	epochEngineLoad.NotifyOnEpoch(trackerLoad.OnEpochEvent, trackerLoad.OnEpochRestore)

	pl := snapshotpb.Payload{}
	require.NoError(t, proto.Unmarshal(state2, &pl))

	additionalProvider, err := trackerLoad.LoadState(context.Background(), types.PayloadFromProto(&pl))
	require.NoError(t, err)
	assert.Nil(t, additionalProvider)

	state3, _, err := trackerLoad.GetState(key)
	require.NoError(t, err)
	require.True(t, bytes.Equal(state2, state3))

	// check a restored party exist in the restored engine
	scores = tracker.CalculateMetricForIndividuals(ctx, &vgproto.DispatchStrategy{AssetForMetric: "asset1", Metric: vgproto.DispatchMetric_DISPATCH_METRIC_MAKER_FEES_RECEIVED, IndividualScope: vgproto.IndividualScope_INDIVIDUAL_SCOPE_ALL, WindowLength: 1, Markets: []string{"market2"}})
	require.Equal(t, 2, len(scores))
	require.Equal(t, "1", scores[0].Score.String())
	require.Equal(t, "party1", scores[0].Party)
	require.Equal(t, true, scores[0].IsEligible)
	require.Equal(t, "0", scores[1].Score.String())
	require.Equal(t, "party2", scores[1].Party)
	require.Equal(t, false, scores[1].IsEligible)

	// end the epoch
	epochEngineLoad.target(context.Background(), types.Epoch{Seq: 2, Action: vgproto.EpochAction_EPOCH_ACTION_END})

	// // NewEngine epoch should scrub the state an produce a difference hash
	state4, _, err := trackerLoad.GetState(key)
	require.NoError(t, err)
	require.False(t, bytes.Equal(state3, state4))

	// // new epoch, we expect the metrics to have been reset

	metrics := []vgproto.DispatchMetric{vgproto.DispatchMetric_DISPATCH_METRIC_MAKER_FEES_PAID, vgproto.DispatchMetric_DISPATCH_METRIC_LP_FEES_RECEIVED, vgproto.DispatchMetric_DISPATCH_METRIC_MAKER_FEES_RECEIVED}
	for _, m := range metrics {
		scores = trackerLoad.CalculateMetricForIndividuals(ctx, &vgproto.DispatchStrategy{AssetForMetric: "asset1", Metric: m, IndividualScope: vgproto.IndividualScope_INDIVIDUAL_SCOPE_ALL, WindowLength: 1, Markets: []string{"market1"}})
		require.Equal(t, 2, len(scores))
		require.Equal(t, "0", scores[0].Score.String())
		require.Equal(t, "party1", scores[0].Party)
		require.Equal(t, false, scores[0].IsEligible)
		require.Equal(t, "0", scores[1].Score.String())
		require.Equal(t, "party2", scores[1].Party)
		require.Equal(t, false, scores[1].IsEligible)
		scores = trackerLoad.CalculateMetricForIndividuals(ctx, &vgproto.DispatchStrategy{AssetForMetric: "asset1", Metric: m, IndividualScope: vgproto.IndividualScope_INDIVIDUAL_SCOPE_ALL, WindowLength: 1, Markets: []string{"market2"}})
		require.Equal(t, 2, len(scores))
		require.Equal(t, "0", scores[0].Score.String())
		require.Equal(t, "party1", scores[0].Party)
		require.Equal(t, false, scores[0].IsEligible)
		require.Equal(t, "0", scores[1].Score.String())
		require.Equal(t, "party2", scores[1].Party)
		require.Equal(t, false, scores[1].IsEligible)
	}
}

func TestDecimalSerialisation(t *testing.T) {
	d := num.DecimalE()
	b, err := d.MarshalBinary()
	require.NoError(t, err)
	dd, err := num.UnmarshalBinaryDecimal(b)
	require.NoError(t, err)
	require.Equal(t, d, dd)
}

func TestUintSerialisation(t *testing.T) {
	ui, _ := num.UintFromString("1000000000000000000", 10)
	b := ui.Bytes()
	bb := b[:]
	uiLoad := num.UintFromBytes(bb)
	require.Equal(t, ui, uiLoad)
}

func TestSnapshot(t *testing.T) {
	tracker := setupDefaultTrackerForTest(t)

	// take a snapshot
	key := (&types.PayloadMarketActivityTracker{}).Key()
	state1, _, err := tracker.GetState(key)
	require.NoError(t, err)

	ctrl := gomock.NewController(t)
	teams := mocks.NewMockTeams(ctrl)
	balanceChecker := mocks.NewMockAccountBalanceChecker(ctrl)
	broker := bmocks.NewMockBroker(ctrl)
	trackerLoad := common.NewMarketActivityTracker(logging.NewTestLogger(), teams, balanceChecker, broker)
	pl := snapshotpb.Payload{}
	require.NoError(t, proto.Unmarshal(state1, &pl))

	additionalProvider, err := trackerLoad.LoadState(context.Background(), types.PayloadFromProto(&pl))
	require.NoError(t, err)
	assert.Nil(t, additionalProvider)

	state2, _, err := trackerLoad.GetState(key)
	require.NoError(t, err)
	require.True(t, bytes.Equal(state1, state2))
}

func TestCheckpoint(t *testing.T) {
	tracker := setupDefaultTrackerForTest(t)

	b, err := tracker.Checkpoint()
	require.NoError(t, err)

	ctrl := gomock.NewController(t)
	teams := mocks.NewMockTeams(ctrl)
	balanceChecker := mocks.NewMockAccountBalanceChecker(ctrl)
	broker := bmocks.NewMockBroker(ctrl)
	trackerLoad := common.NewMarketActivityTracker(logging.NewTestLogger(), teams, balanceChecker, broker)

	require.NoError(t, trackerLoad.Load(context.Background(), b))

	bLoad, err := trackerLoad.Checkpoint()
	require.NoError(t, err)
	require.True(t, bytes.Equal(b, bLoad))
}

func TestSnapshotRoundTripViaEngine(t *testing.T) {
	transfersM5 := []*types.Transfer{
		{Owner: "party3", Type: types.TransferTypeMakerFeeReceive, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.NewUint(100)}},
		{Owner: "party3", Type: types.TransferTypeMakerFeePay, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.NewUint(200)}},
		{Owner: "party3", Type: types.TransferTypeLiquidityFeeNetDistribute, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.NewUint(200)}},
	}
	transfersM6 := []*types.Transfer{
		{Owner: "party4", Type: types.TransferTypeMakerFeeReceive, Amount: &types.FinancialAmount{Asset: "asset2", Amount: num.NewUint(500)}},
		{Owner: "party4", Type: types.TransferTypeMakerFeePay, Amount: &types.FinancialAmount{Asset: "asset2", Amount: num.NewUint(1500)}},
		{Owner: "party4", Type: types.TransferTypeLiquidityFeeNetDistribute, Amount: &types.FinancialAmount{Asset: "asset2", Amount: num.NewUint(1500)}},
	}

	ctx := vgtest.VegaContext("chainid", 100)
	tracker1 := setupDefaultTrackerForTest(t)
	now := time.Now()
	log := logging.NewTestLogger()
	timeService := stubs.NewTimeStub()
	timeService.SetTime(now)
	statsData := stats.New(log, stats.NewDefaultConfig())
	config := snp.DefaultConfig()
	vegaPath := paths.New(t.TempDir())

	snapshotEngine1, err := snp.NewEngine(vegaPath, config, log, timeService, statsData.Blockchain)
	require.NoError(t, err)
	snapshotEngine1CloseFn := vgtest.OnlyOnce(snapshotEngine1.Close)
	defer snapshotEngine1CloseFn()

	snapshotEngine1.AddProviders(tracker1)

	require.NoError(t, snapshotEngine1.Start(ctx))

	hash1, err := snapshotEngine1.SnapshotNow(ctx)
	require.NoError(t, err)

	tracker1.SetEligibilityChecker(&EligibilityChecker{})
	tracker1.MarketProposed("asset1", "market5", "meeeee")
	tracker1.MarketProposed("asset2", "market6", "meeeeeee")
	tracker1.UpdateFeesFromTransfers("asset1", "market5", transfersM5)
	tracker1.UpdateFeesFromTransfers("asset2", "market6", transfersM6)

	state1 := map[string][]byte{}
	for _, key := range tracker1.Keys() {
		state, additionalProvider, err := tracker1.GetState(key)
		require.NoError(t, err)
		assert.Empty(t, additionalProvider)
		state1[key] = state
	}

	snapshotEngine1CloseFn()

	ctrl := gomock.NewController(t)
	teams := mocks.NewMockTeams(ctrl)
	balanceChecker := mocks.NewMockAccountBalanceChecker(ctrl)
	broker := bmocks.NewMockBroker(ctrl)
	tracker2 := common.NewMarketActivityTracker(logging.NewTestLogger(), teams, balanceChecker, broker)
	snapshotEngine2, err := snp.NewEngine(vegaPath, config, log, timeService, statsData.Blockchain)
	require.NoError(t, err)
	defer snapshotEngine2.Close()

	snapshotEngine2.AddProviders(tracker2)

	// This triggers the state restoration from the local snapshot.
	require.NoError(t, snapshotEngine2.Start(ctx))

	// Comparing the hash after restoration, to ensure it produces the same result.
	hash2, _, _ := snapshotEngine2.Info()
	require.Equal(t, hash1, hash2)

	tracker2.MarketProposed("asset1", "market5", "meeeee")
	tracker2.MarketProposed("asset2", "market6", "meeeeeee")
	tracker2.UpdateFeesFromTransfers("asset1", "market5", transfersM5)
	tracker2.UpdateFeesFromTransfers("asset2", "market6", transfersM6)

	state2 := map[string][]byte{}
	for _, key := range tracker2.Keys() {
		state, additionalProvider, err := tracker2.GetState(key)
		require.NoError(t, err)
		assert.Empty(t, additionalProvider)
		state2[key] = state
	}

	for key := range state1 {
		assert.Equalf(t, state1[key], state2[key], "Key %q does not have the same data", key)
	}
}

func TestMarketProposerBonusScenarios(t *testing.T) {
	epochService := &TestEpochEngine{}
	ctrl := gomock.NewController(t)
	teams := mocks.NewMockTeams(ctrl)
	balanceChecker := mocks.NewMockAccountBalanceChecker(ctrl)
	broker := bmocks.NewMockBroker(ctrl)
	tracker := common.NewMarketActivityTracker(logging.NewTestLogger(), teams, balanceChecker, broker)
	epochService.NotifyOnEpoch(tracker.OnEpochEvent, tracker.OnEpochRestore)
	tracker.SetEligibilityChecker(&EligibilityChecker{})

	// setup 4 market for settlement asset1 2 of them proposed by the same proposer, and 2 markets for settlement asset 2
	tracker.MarketProposed("asset1", "market1", "me")
	tracker.MarketProposed("asset1", "market2", "me")
	tracker.MarketProposed("asset1", "market3", "me2")
	tracker.MarketProposed("asset1", "market4", "me3")
	tracker.MarketProposed("asset2", "market5", "me")
	tracker.MarketProposed("asset2", "market6", "me2")

	// no trading done so far so expect no one to be eligible for bonus
	require.Equal(t, 0, len(tracker.GetMarketsWithEligibleProposer("asset1", []string{}, "VEGA", "zohar")))
	require.Equal(t, 0, len(tracker.GetMarketsWithEligibleProposer("asset2", []string{}, "VEGA", "zohar")))

	// market1 goes above the threshold only it should be eligible
	tracker.AddValueTraded("asset1", "market1", num.NewUint(5001))
	require.Equal(t, 1, len(tracker.GetMarketsWithEligibleProposer("asset1", []string{"market1", "market2", "market3"}, "VEGA", "zohar")))
	require.True(t, tracker.IsMarketEligibleForBonus("asset1", "market1", "VEGA", []string{"market1", "market2", "market3"}, "zohar"))
	require.False(t, tracker.IsMarketEligibleForBonus("asset1", "market2", "VEGA", []string{"market1", "market2", "market3"}, "zohar"))
	require.False(t, tracker.IsMarketEligibleForBonus("asset1", "market3", "VEGA", []string{"market1", "market2", "market3"}, "zohar"))
	require.False(t, tracker.IsMarketEligibleForBonus("asset1", "market4", "VEGA", []string{"market1", "market2", "market3"}, "zohar"))
	tracker.MarkPaidProposer("asset1", "market1", "VEGA", []string{"market1", "market2", "market3"}, "zohar")

	// now market 2 and 3 become eligible
	tracker.AddValueTraded("asset1", "market2", num.NewUint(5001))
	tracker.AddValueTraded("asset1", "market3", num.NewUint(5001))
	require.Equal(t, 2, len(tracker.GetMarketsWithEligibleProposer("asset1", []string{"market1", "market2", "market3"}, "VEGA", "zohar")))

	// show that only markets 2 and 3 are now eligible with this combo
	require.False(t, tracker.IsMarketEligibleForBonus("asset1", "market1", "VEGA", []string{"market1", "market2", "market3"}, "zohar"))
	require.True(t, tracker.IsMarketEligibleForBonus("asset1", "market2", "VEGA", []string{"market1", "market2", "market3"}, "zohar"))
	require.True(t, tracker.IsMarketEligibleForBonus("asset1", "market3", "VEGA", []string{"market1", "market2", "market3"}, "zohar"))
	require.False(t, tracker.IsMarketEligibleForBonus("asset1", "market4", "VEGA", []string{"market1", "market2", "market3"}, "zohar"))
	tracker.MarkPaidProposer("asset1", "market2", "VEGA", []string{"market1", "market2", "market3"}, "zohar")
	tracker.MarkPaidProposer("asset1", "market3", "VEGA", []string{"market1", "market2", "market3"}, "zohar")

	// now market4 goes above the threshold but no one gets paid by this combo
	tracker.AddValueTraded("asset1", "market4", num.NewUint(5001))
	require.Equal(t, 0, len(tracker.GetMarketsWithEligibleProposer("asset1", []string{"market1", "market2", "market3"}, "VEGA", "zohar")))
	require.False(t, tracker.IsMarketEligibleForBonus("asset1", "market1", "VEGA", []string{"market1", "market2", "market3"}, "zohar"))
	require.False(t, tracker.IsMarketEligibleForBonus("asset1", "market2", "VEGA", []string{"market1", "market2", "market3"}, "zohar"))
	require.False(t, tracker.IsMarketEligibleForBonus("asset1", "market3", "VEGA", []string{"market1", "market2", "market3"}, "zohar"))
	require.False(t, tracker.IsMarketEligibleForBonus("asset1", "market4", "VEGA", []string{"market1", "market2", "market3"}, "zohar"))

	// now "all" is funded by zohar
	require.Equal(t, 4, len(tracker.GetMarketsWithEligibleProposer("asset1", []string{}, "VEGA", "zohar")))
	require.True(t, tracker.IsMarketEligibleForBonus("asset1", "market1", "VEGA", []string{}, "zohar"))
	require.True(t, tracker.IsMarketEligibleForBonus("asset1", "market2", "VEGA", []string{}, "zohar"))
	require.True(t, tracker.IsMarketEligibleForBonus("asset1", "market3", "VEGA", []string{}, "zohar"))
	require.True(t, tracker.IsMarketEligibleForBonus("asset1", "market4", "VEGA", []string{}, "zohar"))

	tracker.MarkPaidProposer("asset1", "market1", "VEGA", []string{}, "zohar")
	tracker.MarkPaidProposer("asset1", "market2", "VEGA", []string{}, "zohar")
	tracker.MarkPaidProposer("asset1", "market3", "VEGA", []string{}, "zohar")
	tracker.MarkPaidProposer("asset1", "market4", "VEGA", []string{}, "zohar")

	// everyone were paid so next time no one is eligible
	require.Equal(t, 0, len(tracker.GetMarketsWithEligibleProposer("asset1", []string{}, "VEGA", "zohar")))

	// a new market is proposed and gets over the limit
	tracker.MarketProposed("asset1", "market7", "mememe")
	tracker.AddValueTraded("asset1", "market7", num.NewUint(5001))

	// only the new market should be eligible for the "all" combo funded by zohar
	require.Equal(t, 1, len(tracker.GetMarketsWithEligibleProposer("asset1", []string{}, "VEGA", "zohar")))
	require.False(t, tracker.IsMarketEligibleForBonus("asset1", "market1", "VEGA", []string{}, "zohar"))
	require.False(t, tracker.IsMarketEligibleForBonus("asset1", "market2", "VEGA", []string{}, "zohar"))
	require.False(t, tracker.IsMarketEligibleForBonus("asset1", "market3", "VEGA", []string{}, "zohar"))
	require.False(t, tracker.IsMarketEligibleForBonus("asset1", "market4", "VEGA", []string{}, "zohar"))
	require.True(t, tracker.IsMarketEligibleForBonus("asset1", "market7", "VEGA", []string{}, "zohar"))
	tracker.MarkPaidProposer("asset1", "market7", "VEGA", []string{}, "zohar")

	// check that they are no longer eligible for this combo of all
	require.Equal(t, 0, len(tracker.GetMarketsWithEligibleProposer("asset1", []string{}, "VEGA", "zohar")))

	// check new combo
	require.Equal(t, 3, len(tracker.GetMarketsWithEligibleProposer("asset1", []string{"market1", "market3", "market7"}, "VEGA", "zohar")))
	require.True(t, tracker.IsMarketEligibleForBonus("asset1", "market1", "VEGA", []string{"market1", "market3", "market7"}, "zohar"))
	require.False(t, tracker.IsMarketEligibleForBonus("asset1", "market2", "VEGA", []string{"market1", "market3", "market7"}, "zohar"))
	require.True(t, tracker.IsMarketEligibleForBonus("asset1", "market3", "VEGA", []string{"market1", "market3", "market7"}, "zohar"))
	require.False(t, tracker.IsMarketEligibleForBonus("asset1", "market4", "VEGA", []string{"market1", "market3", "market7"}, "zohar"))
	require.True(t, tracker.IsMarketEligibleForBonus("asset1", "market7", "VEGA", []string{"market1", "market3", "market7"}, "zohar"))

	tracker.MarkPaidProposer("asset1", "market1", "VEGA", []string{"market1", "market3", "market7"}, "zohar")
	tracker.MarkPaidProposer("asset1", "market3", "VEGA", []string{"market1", "market3", "market7"}, "zohar")
	tracker.MarkPaidProposer("asset1", "market7", "VEGA", []string{"market1", "market3", "market7"}, "zohar")

	// now that they're marked as paid check they're no longer eligible
	require.Equal(t, 0, len(tracker.GetMarketsWithEligibleProposer("asset1", []string{"market1", "market3", "market7"}, "VEGA", "zohar")))

	// check new asset for the same combo
	require.Equal(t, 3, len(tracker.GetMarketsWithEligibleProposer("asset1", []string{"market1", "market3", "market7"}, "USDC", "zohar")))
	require.True(t, tracker.IsMarketEligibleForBonus("asset1", "market1", "USDC", []string{"market1", "market3", "market7"}, "zohar"))
	require.False(t, tracker.IsMarketEligibleForBonus("asset1", "market2", "USDC", []string{"market1", "market3", "market7"}, "zohar"))
	require.True(t, tracker.IsMarketEligibleForBonus("asset1", "market3", "USDC", []string{"market1", "market3", "market7"}, "zohar"))
	require.False(t, tracker.IsMarketEligibleForBonus("asset1", "market4", "USDC", []string{"market1", "market3", "market7"}, "zohar"))
	require.True(t, tracker.IsMarketEligibleForBonus("asset1", "market7", "USDC", []string{"market1", "market3", "market7"}, "zohar"))

	tracker.MarkPaidProposer("asset1", "market1", "USDC", []string{"market1", "market3", "market7"}, "zohar")
	tracker.MarkPaidProposer("asset1", "market3", "USDC", []string{"market1", "market3", "market7"}, "zohar")
	tracker.MarkPaidProposer("asset1", "market7", "USDC", []string{"market1", "market3", "market7"}, "zohar")

	// now that they're marked as paid check they're no longer eligible
	require.Equal(t, 0, len(tracker.GetMarketsWithEligibleProposer("asset1", []string{"market1", "market3", "market7"}, "USDC", "zohar")))

	// check new funder for the all combo
	require.Equal(t, 5, len(tracker.GetMarketsWithEligibleProposer("asset1", []string{}, "VEGA", "jeremy")))
	require.True(t, tracker.IsMarketEligibleForBonus("asset1", "market1", "VEGA", []string{}, "jeremy"))
	require.True(t, tracker.IsMarketEligibleForBonus("asset1", "market2", "VEGA", []string{}, "jeremy"))
	require.True(t, tracker.IsMarketEligibleForBonus("asset1", "market3", "VEGA", []string{}, "jeremy"))
	require.True(t, tracker.IsMarketEligibleForBonus("asset1", "market4", "VEGA", []string{}, "jeremy"))
	require.True(t, tracker.IsMarketEligibleForBonus("asset1", "market7", "VEGA", []string{}, "jeremy"))

	tracker.MarkPaidProposer("asset1", "market1", "VEGA", []string{}, "jeremy")
	tracker.MarkPaidProposer("asset1", "market2", "VEGA", []string{}, "jeremy")
	tracker.MarkPaidProposer("asset1", "market3", "VEGA", []string{}, "jeremy")
	tracker.MarkPaidProposer("asset1", "market4", "VEGA", []string{}, "jeremy")
	tracker.MarkPaidProposer("asset1", "market7", "VEGA", []string{}, "jeremy")
	require.Equal(t, 0, len(tracker.GetMarketsWithEligibleProposer("asset1", []string{}, "VEGA", "jeremy")))
}

func TestNotionalMetric(t *testing.T) {
	epochService := &TestEpochEngine{}
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	teams := mocks.NewMockTeams(ctrl)
	balanceChecker := mocks.NewMockAccountBalanceChecker(ctrl)
	balanceChecker.EXPECT().GetAvailableBalance(gomock.Any()).Return(num.UintZero(), nil).AnyTimes()
	broker := bmocks.NewMockBroker(ctrl)
	broker.EXPECT().SendBatch(gomock.Any()).AnyTimes()
	broker.EXPECT().Send(gomock.Any()).AnyTimes()
	tracker := common.NewMarketActivityTracker(logging.NewTestLogger(), teams, balanceChecker, broker)
	epochService.NotifyOnEpoch(tracker.OnEpochEvent, tracker.OnEpochRestore)

	epochStartTime := time.Now()
	epochService.target(context.Background(), types.Epoch{Action: vgproto.EpochAction_EPOCH_ACTION_START, StartTime: epochStartTime})
	tracker.SetEligibilityChecker(&EligibilityChecker{})
	tracker.MarketProposed("a1", "m1", "p1")

	// 100 seconds into the epoch record a position of 100
	tracker.RecordPosition("a1", "p1", "m1", 100, num.NewUint(1), num.DecimalOne(), epochStartTime.Add(100*time.Second))

	// 200 seconds later record another position
	// pBar = 100 * 300/400 = 75
	tracker.RecordPosition("a1", "p1", "m1", -200, num.NewUint(2), num.DecimalOne(), epochStartTime.Add(400*time.Second))

	// now end the epoch after 600 seconds
	// pBar = (1 - 600/1000) * 75 + 200 * 600/1000 = 150
	epochService.target(context.Background(), types.Epoch{Action: vgproto.EpochAction_EPOCH_ACTION_END, StartTime: epochStartTime, EndTime: epochStartTime.Add(1000 * time.Second)})

	scores := tracker.CalculateMetricForIndividuals(ctx, &vgproto.DispatchStrategy{AssetForMetric: "a1", Metric: vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_NOTIONAL, IndividualScope: vgproto.IndividualScope_INDIVIDUAL_SCOPE_ALL, WindowLength: 1})
	require.Equal(t, 1, len(scores))
	require.Equal(t, "p1", scores[0].Party)
	require.Equal(t, "0.000027", scores[0].Score.String())

	scores = tracker.CalculateMetricForIndividuals(ctx, &vgproto.DispatchStrategy{AssetForMetric: "a1", Metric: vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_NOTIONAL, IndividualScope: vgproto.IndividualScope_INDIVIDUAL_SCOPE_ALL, WindowLength: 2})
	require.Equal(t, 1, len(scores))
	require.Equal(t, "p1", scores[0].Party)
	require.Equal(t, "0.0000135", scores[0].Score.String())

	scores = tracker.CalculateMetricForIndividuals(ctx, &vgproto.DispatchStrategy{AssetForMetric: "a1", Metric: vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_NOTIONAL, IndividualScope: vgproto.IndividualScope_INDIVIDUAL_SCOPE_ALL, WindowLength: 3})
	require.Equal(t, 1, len(scores))
	require.Equal(t, "p1", scores[0].Party)
	require.Equal(t, "0.000009", scores[0].Score.String())

	// qualifying the market to m1, expect the same result
	scores = tracker.CalculateMetricForIndividuals(ctx, &vgproto.DispatchStrategy{AssetForMetric: "a1", Metric: vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_NOTIONAL, IndividualScope: vgproto.IndividualScope_INDIVIDUAL_SCOPE_ALL, WindowLength: 3, Markets: []string{"m1"}})
	require.Equal(t, 1, len(scores))
	require.Equal(t, "p1", scores[0].Party)
	require.Equal(t, "0.000009", scores[0].Score.String())

	// qualifying the market to m2, expect the same result
	scores = tracker.CalculateMetricForIndividuals(ctx, &vgproto.DispatchStrategy{AssetForMetric: "a1", Metric: vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_NOTIONAL, IndividualScope: vgproto.IndividualScope_INDIVIDUAL_SCOPE_ALL, WindowLength: 3, Markets: []string{"m2"}})
	require.Equal(t, 0, len(scores))

	epochService.target(context.Background(), types.Epoch{Action: vgproto.EpochAction_EPOCH_ACTION_START, StartTime: epochStartTime.Add(1000 * time.Second)})

	// 600 seconds into the epoch new position is recorded for p1
	// pBar = 0 * 150 + 1 * 200 = 200
	tracker.RecordPosition("a1", "p1", "m1", 100, num.NewUint(3), num.DecimalOne(), epochStartTime.Add(1600*time.Second))

	// end the epoch
	// pBar = 0.6 * 200 + 0.4 * 100 = 160
	epochService.target(context.Background(), types.Epoch{Action: vgproto.EpochAction_EPOCH_ACTION_END, StartTime: epochStartTime.Add(1000 * time.Second), EndTime: epochStartTime.Add(2000 * time.Second)})
	scores = tracker.CalculateMetricForIndividuals(ctx, &vgproto.DispatchStrategy{AssetForMetric: "a1", Metric: vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_NOTIONAL, IndividualScope: vgproto.IndividualScope_INDIVIDUAL_SCOPE_ALL, WindowLength: 1})
	require.Equal(t, 1, len(scores))
	require.Equal(t, "p1", scores[0].Party)
	require.Equal(t, "0.000036", scores[0].Score.String())

	scores = tracker.CalculateMetricForIndividuals(ctx, &vgproto.DispatchStrategy{AssetForMetric: "a1", Metric: vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_NOTIONAL, IndividualScope: vgproto.IndividualScope_INDIVIDUAL_SCOPE_ALL, WindowLength: 2})
	require.Equal(t, 1, len(scores))
	require.Equal(t, "p1", scores[0].Party)
	require.Equal(t, "0.0000315", scores[0].Score.String())

	scores = tracker.CalculateMetricForIndividuals(ctx, &vgproto.DispatchStrategy{AssetForMetric: "a1", Metric: vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_NOTIONAL, IndividualScope: vgproto.IndividualScope_INDIVIDUAL_SCOPE_ALL, WindowLength: 5})
	require.Equal(t, 1, len(scores))
	require.Equal(t, "p1", scores[0].Party)
	require.Equal(t, "0.0000126", scores[0].Score.String())

	// qualify to m1
	scores = tracker.CalculateMetricForIndividuals(ctx, &vgproto.DispatchStrategy{AssetForMetric: "a1", Metric: vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_NOTIONAL, IndividualScope: vgproto.IndividualScope_INDIVIDUAL_SCOPE_ALL, WindowLength: 5, Markets: []string{"m1"}})
	require.Equal(t, 1, len(scores))
	require.Equal(t, "p1", scores[0].Party)
	require.Equal(t, "0.0000126", scores[0].Score.String())

	// qualify to m2
	scores = tracker.CalculateMetricForIndividuals(ctx, &vgproto.DispatchStrategy{AssetForMetric: "a1", Metric: vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_NOTIONAL, IndividualScope: vgproto.IndividualScope_INDIVIDUAL_SCOPE_ALL, WindowLength: 5, Markets: []string{"m2"}})
	require.Equal(t, 0, len(scores))

	// now lets lets at an epoch with no activity
	epochService.target(context.Background(), types.Epoch{Action: vgproto.EpochAction_EPOCH_ACTION_START, StartTime: epochStartTime.Add(2000 * time.Second)})
	// end the epoch
	// pBar = 0 * 160 + 1 * 100 = 100
	epochService.target(context.Background(), types.Epoch{Action: vgproto.EpochAction_EPOCH_ACTION_END, StartTime: epochStartTime.Add(2000 * time.Second), EndTime: epochStartTime.Add(3000 * time.Second)})
	scores = tracker.CalculateMetricForIndividuals(ctx, &vgproto.DispatchStrategy{AssetForMetric: "a1", Metric: vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_NOTIONAL, IndividualScope: vgproto.IndividualScope_INDIVIDUAL_SCOPE_ALL, WindowLength: 1})
	require.Equal(t, 1, len(scores))
	require.Equal(t, "p1", scores[0].Party)
	require.Equal(t, "0.00003", scores[0].Score.String())

	scores = tracker.CalculateMetricForIndividuals(ctx, &vgproto.DispatchStrategy{AssetForMetric: "a1", Metric: vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_NOTIONAL, IndividualScope: vgproto.IndividualScope_INDIVIDUAL_SCOPE_ALL, WindowLength: 2})
	require.Equal(t, 1, len(scores))
	require.Equal(t, "p1", scores[0].Party)
	require.Equal(t, "0.000033", scores[0].Score.String())

	scores = tracker.CalculateMetricForIndividuals(ctx, &vgproto.DispatchStrategy{AssetForMetric: "a1", Metric: vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_NOTIONAL, IndividualScope: vgproto.IndividualScope_INDIVIDUAL_SCOPE_ALL, WindowLength: 4})
	require.Equal(t, 1, len(scores))
	require.Equal(t, "p1", scores[0].Party)
	require.Equal(t, "0.00002325", scores[0].Score.String())
}

func TestRealisedReturnMetric(t *testing.T) {
	epochService := &TestEpochEngine{}
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	teams := mocks.NewMockTeams(ctrl)
	balanceChecker := mocks.NewMockAccountBalanceChecker(ctrl)
	balanceChecker.EXPECT().GetAvailableBalance(gomock.Any()).Return(num.UintZero(), nil).AnyTimes()
	broker := bmocks.NewMockBroker(ctrl)
	broker.EXPECT().SendBatch(gomock.Any()).AnyTimes()
	broker.EXPECT().Send(gomock.Any()).AnyTimes()
	tracker := common.NewMarketActivityTracker(logging.NewTestLogger(), teams, balanceChecker, broker)
	epochService.NotifyOnEpoch(tracker.OnEpochEvent, tracker.OnEpochRestore)

	epochStartTime := time.Now()
	epochService.target(context.Background(), types.Epoch{Action: vgproto.EpochAction_EPOCH_ACTION_START, StartTime: epochStartTime})
	tracker.SetEligibilityChecker(&EligibilityChecker{})
	tracker.MarketProposed("a1", "m1", "p1")

	tracker.RecordFundingPayment("a1", "p1", "m1", num.NewDecimalFromFloat(100))
	tracker.RecordFundingPayment("a1", "p1", "m1", num.NewDecimalFromFloat(-200))
	tracker.RecordRealisedPosition("a1", "p1", "m1", num.DecimalFromFloat(130))

	// now end the epoch after 600 seconds
	epochService.target(context.Background(), types.Epoch{Action: vgproto.EpochAction_EPOCH_ACTION_END, StartTime: epochStartTime, EndTime: epochStartTime.Add(1000 * time.Second)})

	scores := tracker.CalculateMetricForIndividuals(ctx, &vgproto.DispatchStrategy{AssetForMetric: "a1", Metric: vgproto.DispatchMetric_DISPATCH_METRIC_REALISED_RETURN, IndividualScope: vgproto.IndividualScope_INDIVIDUAL_SCOPE_ALL, WindowLength: 1})
	require.Equal(t, 1, len(scores))
	require.Equal(t, "p1", scores[0].Party)
	require.Equal(t, "30", scores[0].Score.String())

	scores = tracker.CalculateMetricForIndividuals(ctx, &vgproto.DispatchStrategy{AssetForMetric: "a1", Metric: vgproto.DispatchMetric_DISPATCH_METRIC_REALISED_RETURN, IndividualScope: vgproto.IndividualScope_INDIVIDUAL_SCOPE_ALL, WindowLength: 2})
	require.Equal(t, 1, len(scores))
	require.Equal(t, "p1", scores[0].Party)
	require.Equal(t, "15", scores[0].Score.String())

	scores = tracker.CalculateMetricForIndividuals(ctx, &vgproto.DispatchStrategy{AssetForMetric: "a1", Metric: vgproto.DispatchMetric_DISPATCH_METRIC_REALISED_RETURN, IndividualScope: vgproto.IndividualScope_INDIVIDUAL_SCOPE_ALL, WindowLength: 3})
	require.Equal(t, 1, len(scores))
	require.Equal(t, "p1", scores[0].Party)
	require.Equal(t, "10", scores[0].Score.String())

	// qualifying the market to m1, expect the same result
	scores = tracker.CalculateMetricForIndividuals(ctx, &vgproto.DispatchStrategy{AssetForMetric: "a1", Metric: vgproto.DispatchMetric_DISPATCH_METRIC_REALISED_RETURN, IndividualScope: vgproto.IndividualScope_INDIVIDUAL_SCOPE_ALL, WindowLength: 3, Markets: []string{"m1"}})
	require.Equal(t, 1, len(scores))
	require.Equal(t, "p1", scores[0].Party)
	require.Equal(t, "10", scores[0].Score.String())

	// qualifying the market to m2, expect the same result
	scores = tracker.CalculateMetricForIndividuals(ctx, &vgproto.DispatchStrategy{AssetForMetric: "a1", Metric: vgproto.DispatchMetric_DISPATCH_METRIC_REALISED_RETURN, IndividualScope: vgproto.IndividualScope_INDIVIDUAL_SCOPE_ALL, WindowLength: 3, Markets: []string{"m2"}})
	require.Equal(t, 0, len(scores))

	epochService.target(context.Background(), types.Epoch{Action: vgproto.EpochAction_EPOCH_ACTION_START, StartTime: epochStartTime.Add(1000 * time.Second)})

	tracker.RecordRealisedPosition("a1", "p1", "m1", num.DecimalFromFloat(-45))

	// end the epoch
	epochService.target(context.Background(), types.Epoch{Action: vgproto.EpochAction_EPOCH_ACTION_END, StartTime: epochStartTime.Add(1000 * time.Second), EndTime: epochStartTime.Add(2000 * time.Second)})

	scores = tracker.CalculateMetricForIndividuals(ctx, &vgproto.DispatchStrategy{AssetForMetric: "a1", Metric: vgproto.DispatchMetric_DISPATCH_METRIC_REALISED_RETURN, IndividualScope: vgproto.IndividualScope_INDIVIDUAL_SCOPE_ALL, WindowLength: 1})
	require.Equal(t, 1, len(scores))
	require.Equal(t, "p1", scores[0].Party)
	require.Equal(t, "-45", scores[0].Score.String())

	scores = tracker.CalculateMetricForIndividuals(ctx, &vgproto.DispatchStrategy{AssetForMetric: "a1", Metric: vgproto.DispatchMetric_DISPATCH_METRIC_REALISED_RETURN, IndividualScope: vgproto.IndividualScope_INDIVIDUAL_SCOPE_ALL, WindowLength: 2})
	require.Equal(t, 1, len(scores))
	require.Equal(t, "p1", scores[0].Party)
	require.Equal(t, "-7.5", scores[0].Score.String())

	scores = tracker.CalculateMetricForIndividuals(ctx, &vgproto.DispatchStrategy{AssetForMetric: "a1", Metric: vgproto.DispatchMetric_DISPATCH_METRIC_REALISED_RETURN, IndividualScope: vgproto.IndividualScope_INDIVIDUAL_SCOPE_ALL, WindowLength: 5})
	require.Equal(t, 1, len(scores))
	require.Equal(t, "p1", scores[0].Party)
	require.Equal(t, "-3", scores[0].Score.String())

	// qualify to m1
	scores = tracker.CalculateMetricForIndividuals(ctx, &vgproto.DispatchStrategy{AssetForMetric: "a1", Metric: vgproto.DispatchMetric_DISPATCH_METRIC_REALISED_RETURN, IndividualScope: vgproto.IndividualScope_INDIVIDUAL_SCOPE_ALL, WindowLength: 5, Markets: []string{"m1"}})
	require.Equal(t, 1, len(scores))
	require.Equal(t, "p1", scores[0].Party)
	require.Equal(t, "-3", scores[0].Score.String())

	// qualify to m2
	scores = tracker.CalculateMetricForIndividuals(ctx, &vgproto.DispatchStrategy{AssetForMetric: "a1", Metric: vgproto.DispatchMetric_DISPATCH_METRIC_REALISED_RETURN, IndividualScope: vgproto.IndividualScope_INDIVIDUAL_SCOPE_ALL, WindowLength: 5, Markets: []string{"m2"}})
	require.Equal(t, 0, len(scores))

	// now lets lets at an epoch with no activity
	epochService.target(context.Background(), types.Epoch{Action: vgproto.EpochAction_EPOCH_ACTION_START, StartTime: epochStartTime.Add(2000 * time.Second)})
	// end the epoch
	epochService.target(context.Background(), types.Epoch{Action: vgproto.EpochAction_EPOCH_ACTION_END, StartTime: epochStartTime.Add(2000 * time.Second), EndTime: epochStartTime.Add(3000 * time.Second)})
	scores = tracker.CalculateMetricForIndividuals(ctx, &vgproto.DispatchStrategy{AssetForMetric: "a1", Metric: vgproto.DispatchMetric_DISPATCH_METRIC_REALISED_RETURN, IndividualScope: vgproto.IndividualScope_INDIVIDUAL_SCOPE_ALL, WindowLength: 1})
	require.Equal(t, 1, len(scores))
	require.Equal(t, "p1", scores[0].Party)
	require.Equal(t, "0", scores[0].Score.String())
	require.Equal(t, false, scores[0].IsEligible)

	scores = tracker.CalculateMetricForIndividuals(ctx, &vgproto.DispatchStrategy{AssetForMetric: "a1", Metric: vgproto.DispatchMetric_DISPATCH_METRIC_REALISED_RETURN, IndividualScope: vgproto.IndividualScope_INDIVIDUAL_SCOPE_ALL, WindowLength: 2})
	require.Equal(t, 1, len(scores))
	require.Equal(t, "p1", scores[0].Party)
	require.Equal(t, "-22.5", scores[0].Score.String())

	scores = tracker.CalculateMetricForIndividuals(ctx, &vgproto.DispatchStrategy{AssetForMetric: "a1", Metric: vgproto.DispatchMetric_DISPATCH_METRIC_REALISED_RETURN, IndividualScope: vgproto.IndividualScope_INDIVIDUAL_SCOPE_ALL, WindowLength: 4})
	require.Equal(t, 1, len(scores))
	require.Equal(t, "p1", scores[0].Party)
	require.Equal(t, "-3.75", scores[0].Score.String())
}

func TestRelativeReturnMetric(t *testing.T) {
	ctx := context.Background()
	epochService := &TestEpochEngine{}
	ctrl := gomock.NewController(t)
	teams := mocks.NewMockTeams(ctrl)
	balanceChecker := mocks.NewMockAccountBalanceChecker(ctrl)
	balanceChecker.EXPECT().GetAvailableBalance(gomock.Any()).Return(num.UintZero(), nil).AnyTimes()

	broker := bmocks.NewMockBroker(ctrl)
	broker.EXPECT().Send(gomock.Any()).AnyTimes()
	broker.EXPECT().SendBatch(gomock.Any()).AnyTimes()
	tracker := common.NewMarketActivityTracker(logging.NewTestLogger(), teams, balanceChecker, broker)
	epochService.NotifyOnEpoch(tracker.OnEpochEvent, tracker.OnEpochRestore)

	epochStartTime := time.Now()
	epochService.target(context.Background(), types.Epoch{Action: vgproto.EpochAction_EPOCH_ACTION_START, StartTime: epochStartTime})
	tracker.SetEligibilityChecker(&EligibilityChecker{})
	tracker.MarketProposed("a1", "m1", "p1")

	tracker.RecordPosition("a1", "p1", "m1", 100, num.NewUint(1), num.DecimalOne(), epochStartTime.Add(100*time.Second))
	tracker.RecordPosition("a1", "p1", "m1", -200, num.NewUint(2), num.DecimalOne(), epochStartTime.Add(400*time.Second))

	tracker.RecordM2M("a1", "p1", "m1", num.DecimalFromInt64(100))
	tracker.RecordM2M("a1", "p1", "m1", num.DecimalFromInt64(-120))
	tracker.RecordM2M("a1", "p1", "m1", num.DecimalFromInt64(150))
	tracker.RecordM2M("a1", "p1", "m1", num.DecimalFromInt64(-115))

	// end the epoch
	epochService.target(context.Background(), types.Epoch{Action: vgproto.EpochAction_EPOCH_ACTION_END, StartTime: epochStartTime, EndTime: epochStartTime.Add(1000 * time.Second)})

	// the total m2m = 15
	// the time weighted position for the epoch = 150
	// therefore the r = 15/150 = 0.1

	// window size 1
	scores := tracker.CalculateMetricForIndividuals(ctx, &vgproto.DispatchStrategy{AssetForMetric: "a1", Metric: vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, IndividualScope: vgproto.IndividualScope_INDIVIDUAL_SCOPE_ALL, WindowLength: 1})
	require.Equal(t, 1, len(scores))
	require.Equal(t, "p1", scores[0].Party)
	require.Equal(t, "0.1", scores[0].Score.String())

	// window size 2
	scores = tracker.CalculateMetricForIndividuals(ctx, &vgproto.DispatchStrategy{AssetForMetric: "a1", Metric: vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, IndividualScope: vgproto.IndividualScope_INDIVIDUAL_SCOPE_ALL, WindowLength: 2})
	require.Equal(t, 1, len(scores))
	require.Equal(t, "p1", scores[0].Party)
	require.Equal(t, "0.05", scores[0].Score.String())

	// window size 3
	scores = tracker.CalculateMetricForIndividuals(ctx, &vgproto.DispatchStrategy{AssetForMetric: "a1", Metric: vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, IndividualScope: vgproto.IndividualScope_INDIVIDUAL_SCOPE_ALL, WindowLength: 3})
	require.Equal(t, 1, len(scores))
	require.Equal(t, "p1", scores[0].Party)
	require.Equal(t, "0.0333333333333333", scores[0].Score.String())

	// add only this market in scope, expect same result
	scores = tracker.CalculateMetricForIndividuals(ctx, &vgproto.DispatchStrategy{AssetForMetric: "a1", Metric: vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, IndividualScope: vgproto.IndividualScope_INDIVIDUAL_SCOPE_ALL, WindowLength: 3, Markets: []string{"m1"}})
	require.Equal(t, 1, len(scores))
	require.Equal(t, "p1", scores[0].Party)
	require.Equal(t, "0.0333333333333333", scores[0].Score.String())

	// add market scope with the wrong market:
	scores = tracker.CalculateMetricForIndividuals(ctx, &vgproto.DispatchStrategy{AssetForMetric: "a1", Metric: vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, IndividualScope: vgproto.IndividualScope_INDIVIDUAL_SCOPE_ALL, WindowLength: 3, Markets: []string{"m2"}})
	require.Equal(t, 0, len(scores))

	// add a scope with a different market, expect nothing back
	scores = tracker.CalculateMetricForIndividuals(ctx, &vgproto.DispatchStrategy{AssetForMetric: "a1", Metric: vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, IndividualScope: vgproto.IndividualScope_INDIVIDUAL_SCOPE_ALL, WindowLength: 3, Markets: []string{"m2"}})
	require.Equal(t, 0, len(scores))

	// lets run another epoch and make some loss
	epochService.target(context.Background(), types.Epoch{Action: vgproto.EpochAction_EPOCH_ACTION_START, StartTime: epochStartTime.Add(1000 * time.Second)})
	tracker.RecordPosition("a1", "p1", "m1", 100, num.NewUint(2), num.DecimalOne(), epochStartTime.Add(1600*time.Second))
	tracker.RecordM2M("a1", "p1", "m1", num.DecimalFromInt64(-8))

	// end the epoch
	epochService.target(context.Background(), types.Epoch{Action: vgproto.EpochAction_EPOCH_ACTION_END, StartTime: epochStartTime.Add(1000 * time.Second), EndTime: epochStartTime.Add(2000 * time.Second)})

	// total m2m = -8
	// the time weighted position for the epoch = 160
	// therefore r = -0.05
	scores = tracker.CalculateMetricForIndividuals(ctx, &vgproto.DispatchStrategy{AssetForMetric: "a1", Metric: vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, IndividualScope: vgproto.IndividualScope_INDIVIDUAL_SCOPE_ALL, WindowLength: 1})
	require.Equal(t, 1, len(scores))
	require.Equal(t, "p1", scores[0].Party)
	require.Equal(t, "-0.05", scores[0].Score.String())

	// with window size=2 we get (0.1-0.05)/2 = 0.025
	scores = tracker.CalculateMetricForIndividuals(ctx, &vgproto.DispatchStrategy{AssetForMetric: "a1", Metric: vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, IndividualScope: vgproto.IndividualScope_INDIVIDUAL_SCOPE_ALL, WindowLength: 2})
	require.Equal(t, 1, len(scores))
	require.Equal(t, "p1", scores[0].Party)
	require.Equal(t, "0.025", scores[0].Score.String())

	// with window size=4 we get (0.1-0.05)/4 = 0.0125
	scores = tracker.CalculateMetricForIndividuals(ctx, &vgproto.DispatchStrategy{AssetForMetric: "a1", Metric: vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, IndividualScope: vgproto.IndividualScope_INDIVIDUAL_SCOPE_ALL, WindowLength: 4})
	require.Equal(t, 1, len(scores))
	require.Equal(t, "p1", scores[0].Party)
	require.Equal(t, "0.0125", scores[0].Score.String())
}

func TestTeamStatsForMarkets(t *testing.T) {
	ctrl := gomock.NewController(t)
	teams := mocks.NewMockTeams(ctrl)
	balanceChecker := mocks.NewMockAccountBalanceChecker(ctrl)
	broker := bmocks.NewMockBroker(ctrl)
	tracker := common.NewMarketActivityTracker(logging.NewTestLogger(), teams, balanceChecker, broker)

	asset1 := vgrand.RandomStr(5)
	asset2 := vgrand.RandomStr(5)
	asset3 := vgrand.RandomStr(5)

	market1 := "market1"
	market2 := "market2"
	market3 := "market3"
	market4 := "market4"
	market5 := "market5"
	market6 := "market6"
	market7 := "market7"

	team1 := "team1"
	team2 := "team2"
	member11 := "member11"
	member12 := "member12"
	member21 := "member21"
	member22 := "member22"
	lonewolf1 := "lone-wolf1"
	lonewolf2 := "lone-wolf2"

	// 1. Need markets with different assets.
	tracker.MarketProposed(asset1, market1, vgrand.RandomStr(5))
	tracker.MarketProposed(asset1, market2, vgrand.RandomStr(5))
	tracker.MarketProposed(asset1, market3, vgrand.RandomStr(5))

	tracker.MarketProposed(asset2, market4, vgrand.RandomStr(5))
	tracker.MarketProposed(asset2, market5, vgrand.RandomStr(5))

	tracker.MarketProposed(asset3, market6, vgrand.RandomStr(5))
	tracker.MarketProposed(asset3, market7, vgrand.RandomStr(5))

	// 2. Need to define teams.
	teams.EXPECT().GetAllTeamsWithParties(uint64(0)).Return(map[string][]string{
		team1: {member11, member12},
		team2: {member21, member22},
	}).Times(1)

	// 3. Need parties generating volume on these markets.
	tracker.RecordNotionalTakerVolume(market1, member11, num.NewUint(1))
	tracker.RecordNotionalTakerVolume(market1, member12, num.NewUint(2))
	tracker.RecordNotionalTakerVolume(market1, member21, num.NewUint(3))
	tracker.RecordNotionalTakerVolume(market1, member22, num.NewUint(4))
	tracker.RecordNotionalTakerVolume(market1, lonewolf1, num.NewUint(5))
	tracker.RecordNotionalTakerVolume(market1, lonewolf2, num.NewUint(6))

	tracker.RecordNotionalTakerVolume(market2, member11, num.NewUint(1))
	tracker.RecordNotionalTakerVolume(market2, member12, num.NewUint(2))
	tracker.RecordNotionalTakerVolume(market2, member21, num.NewUint(3))
	tracker.RecordNotionalTakerVolume(market2, member22, num.NewUint(4))
	tracker.RecordNotionalTakerVolume(market2, lonewolf1, num.NewUint(5))
	tracker.RecordNotionalTakerVolume(market2, lonewolf2, num.NewUint(6))

	// No participation of team 2 in market 3.
	tracker.RecordNotionalTakerVolume(market3, member11, num.NewUint(1))
	tracker.RecordNotionalTakerVolume(market3, member12, num.NewUint(2))
	tracker.RecordNotionalTakerVolume(market3, lonewolf1, num.NewUint(5))
	tracker.RecordNotionalTakerVolume(market3, lonewolf2, num.NewUint(6))

	// No participation of team 1 in market 4.
	tracker.RecordNotionalTakerVolume(market4, member21, num.NewUint(3))
	tracker.RecordNotionalTakerVolume(market4, member22, num.NewUint(4))
	tracker.RecordNotionalTakerVolume(market4, lonewolf1, num.NewUint(5))
	tracker.RecordNotionalTakerVolume(market4, lonewolf2, num.NewUint(6))

	// Market 5 is not expected to be filtered on, so none of these volume
	// should show up in the stats.
	tracker.RecordNotionalTakerVolume(market5, member11, num.NewUint(1000))
	tracker.RecordNotionalTakerVolume(market5, member12, num.NewUint(2000))
	tracker.RecordNotionalTakerVolume(market5, member12, num.NewUint(3000))
	tracker.RecordNotionalTakerVolume(market5, member22, num.NewUint(4000))
	tracker.RecordNotionalTakerVolume(market5, lonewolf1, num.NewUint(5000))
	tracker.RecordNotionalTakerVolume(market5, lonewolf2, num.NewUint(6000))

	// Nobody likes market 6. So, no participation of any kind.

	// Only lone-wolves in market 7.
	tracker.RecordNotionalTakerVolume(market7, lonewolf1, num.NewUint(5))
	tracker.RecordNotionalTakerVolume(market7, lonewolf2, num.NewUint(6))

	// Regarding the dataset above, this should result in gathering the data from
	// the market 1, 2, 3, 4, and 7, but not 5 and 6, because:
	//   - we want all markets from asset 1 -> market 1, 2, and 3.
	//   - we want specific market 1, 3, 4, and 7.
	//
	// NB: It's on purpose we have duplicated references to the market 1 and 3, so
	// we can ensure it's duplicated and we don't add up stats from a market multiple
	// times.
	teamsStats := tracker.TeamStatsForMarkets([]string{asset1}, []string{market1, market3, market4, market7})

	assert.Equal(t, map[string]map[string]*num.Uint{
		team1: {
			member11: num.NewUint(3),
			member12: num.NewUint(6),
		},
		team2: {
			member21: num.NewUint(9),
			member22: num.NewUint(12),
		},
	}, teamsStats)
}

func setupDefaultTrackerForTest(t *testing.T) *common.MarketActivityTracker {
	t.Helper()

	epochService := &TestEpochEngine{}
	ctrl := gomock.NewController(t)
	teams := mocks.NewMockTeams(ctrl)
	balanceChecker := mocks.NewMockAccountBalanceChecker(ctrl)
	broker := bmocks.NewMockBroker(ctrl)

	tracker := common.NewMarketActivityTracker(logging.NewTestLogger(), teams, balanceChecker, broker)
	epochService.NotifyOnEpoch(tracker.OnEpochEvent, tracker.OnEpochRestore)

	epochStartTime := time.Now()
	epochService.target(context.Background(), types.Epoch{Action: vgproto.EpochAction_EPOCH_ACTION_START, StartTime: epochStartTime})
	tracker.SetEligibilityChecker(&EligibilityChecker{})

	tracker.MarketProposed("asset1", "market1", "me")
	tracker.MarketProposed("asset1", "market2", "me2")
	tracker.MarketProposed("asset1", "market4", "me4")
	tracker.MarketProposed("asset2", "market3", "me3")

	tracker.RecordPosition("asset1", "p1", "market1", 100, num.NewUint(1), num.DecimalOne(), time.Now())
	tracker.RecordPosition("asset1", "p1", "market2", 200, num.NewUint(2), num.DecimalOne(), time.Now())
	tracker.RecordPosition("asset1", "p2", "market1", 300, num.NewUint(3), num.DecimalOne(), time.Now())
	tracker.RecordPosition("asset1", "p3", "market2", 400, num.NewUint(4), num.DecimalOne(), time.Now())
	tracker.RecordPosition("asset1", "p3", "market4", 500, num.NewUint(5), num.DecimalOne(), time.Now())
	tracker.RecordPosition("asset2", "p4", "market3", 600, num.NewUint(6), num.DecimalOne(), time.Now())

	tracker.RecordM2M("asset1", "p1", "market1", num.DecimalOne())
	tracker.RecordM2M("asset1", "p1", "market2", num.DecimalFromInt64(5))

	tracker.RecordNotionalTakerVolume("market1", "p1", num.NewUint(10))
	tracker.RecordNotionalTakerVolume("market1", "p2", num.NewUint(10))
	tracker.RecordNotionalTakerVolume("market1", "p3", num.NewUint(10))
	tracker.RecordNotionalTakerVolume("market1", "p4", num.NewUint(10))
	tracker.RecordNotionalTakerVolume("market2", "p1", num.NewUint(20))
	tracker.RecordNotionalTakerVolume("market2", "p2", num.NewUint(20))
	tracker.RecordNotionalTakerVolume("market2", "p3", num.NewUint(20))
	tracker.RecordNotionalTakerVolume("market2", "p4", num.NewUint(20))
	tracker.RecordNotionalTakerVolume("market3", "p1", num.NewUint(30))
	tracker.RecordNotionalTakerVolume("market3", "p2", num.NewUint(30))
	tracker.RecordNotionalTakerVolume("market3", "p3", num.NewUint(30))
	tracker.RecordNotionalTakerVolume("market3", "p4", num.NewUint(30))
	tracker.RecordNotionalTakerVolume("market4", "p1", num.NewUint(40))
	tracker.RecordNotionalTakerVolume("market4", "p2", num.NewUint(40))
	tracker.RecordNotionalTakerVolume("market4", "p3", num.NewUint(40))
	tracker.RecordNotionalTakerVolume("market4", "p4", num.NewUint(40))

	// update with a few transfers
	transfersM1 := []*types.Transfer{
		{Owner: "party1", Type: types.TransferTypeMakerFeeReceive, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.NewUint(100)}},
		{Owner: "party1", Type: types.TransferTypeMakerFeePay, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.NewUint(200)}},
		{Owner: "party1", Type: types.TransferTypeLiquidityFeeNetDistribute, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.NewUint(200)}},
		{Owner: "party1", Type: types.TransferTypeMakerFeeReceive, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.NewUint(400)}},
		{Owner: "party1", Type: types.TransferTypeMakerFeePay, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.NewUint(300)}},
		{Owner: "party1", Type: types.TransferTypeLiquidityFeeNetDistribute, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.NewUint(600)}},
		{Owner: "party2", Type: types.TransferTypeMakerFeeReceive, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.NewUint(900)}},
		{Owner: "party2", Type: types.TransferTypeMakerFeePay, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.NewUint(800)}},
		{Owner: "party2", Type: types.TransferTypeLiquidityFeeNetDistribute, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.NewUint(700)}},
		{Owner: "party2", Type: types.TransferTypeMakerFeeReceive, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.NewUint(600)}},
		{Owner: "party2", Type: types.TransferTypeMakerFeePay, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.NewUint(200)}},
		{Owner: "party2", Type: types.TransferTypeLiquidityFeeNetDistribute, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.NewUint(1000)}},
	}
	tracker.UpdateFeesFromTransfers("asset1", "market1", transfersM1)

	transfersM2 := []*types.Transfer{
		{Owner: "party1", Type: types.TransferTypeMakerFeeReceive, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.NewUint(500)}},
		{Owner: "party2", Type: types.TransferTypeMakerFeePay, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.NewUint(1500)}},
		{Owner: "party2", Type: types.TransferTypeLiquidityFeeNetDistribute, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.NewUint(1500)}},
	}
	tracker.UpdateFeesFromTransfers("asset2", "market2", transfersM2)

	transfersM3 := []*types.Transfer{
		{Owner: "party1", Type: types.TransferTypeMakerFeePay, Amount: &types.FinancialAmount{Asset: "asset2", Amount: num.NewUint(500)}},
		{Owner: "party2", Type: types.TransferTypeMakerFeeReceive, Amount: &types.FinancialAmount{Asset: "asset2", Amount: num.NewUint(450)}},
	}
	tracker.UpdateFeesFromTransfers("asset2", "market3", transfersM3)

	return tracker
}
