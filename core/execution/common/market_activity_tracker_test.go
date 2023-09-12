// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.VEGA file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package common_test

import (
	"bytes"
	"context"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/core/execution/common"
	"code.vegaprotocol.io/vega/core/execution/common/mocks"
	"code.vegaprotocol.io/vega/core/integration/stubs"
	snp "code.vegaprotocol.io/vega/core/snapshot"

	"code.vegaprotocol.io/vega/core/stats"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/libs/proto"
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

	tracker := common.NewMarketActivityTracker(logging.NewTestLogger(), &TestEpochEngine{}, teams, balanceChecker)
	tracker.SetEligibilityChecker(&EligibilityChecker{})

	tracker.MarketProposed("asset1", "market1", "me")
	tracker.MarketProposed("asset1", "market2", "me2")

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
	trackerLoad := common.NewMarketActivityTracker(logging.NewTestLogger(), &TestEpochEngine{}, teams2, balanceChecker2)
	pl := snapshotpb.Payload{}
	require.NoError(t, proto.Unmarshal(state1, &pl))

	trackerLoad.LoadState(context.Background(), types.PayloadFromProto(&pl))

	state2, _, err := trackerLoad.GetState(key)
	require.NoError(t, err)
	require.True(t, bytes.Equal(state1, state2))
}

func TestRemoveMarket(t *testing.T) {
	epochService := &TestEpochEngine{}
	ctrl := gomock.NewController(t)
	teams := mocks.NewMockTeams(ctrl)
	balanceChecker := mocks.NewMockAccountBalanceChecker(ctrl)

	tracker := common.NewMarketActivityTracker(logging.NewTestLogger(), epochService, teams, balanceChecker)
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

func TestGetScores(t *testing.T) {
	epochService := &TestEpochEngine{}
	ctrl := gomock.NewController(t)
	teams := mocks.NewMockTeams(ctrl)
	balanceChecker := mocks.NewMockAccountBalanceChecker(ctrl)
	tracker := common.NewMarketActivityTracker(logging.NewTestLogger(), epochService, teams, balanceChecker)
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
			scores := tracker.CalculateMetricForIndividuals(&vgproto.DispatchStrategy{AssetForMetric: asset, Metric: m, IndividualScope: vgproto.IndividualScope_INDIVIDUAL_SCOPE_ALL, WindowLength: 1})
			require.Equal(t, 0, len(scores))
		}
	}

	// asset1 one market in scope
	for _, market := range []string{"market1", "market2", "market4"} {
		for _, m := range []vgproto.DispatchMetric{vgproto.DispatchMetric_DISPATCH_METRIC_MAKER_FEES_PAID, vgproto.DispatchMetric_DISPATCH_METRIC_LP_FEES_RECEIVED, vgproto.DispatchMetric_DISPATCH_METRIC_MAKER_FEES_RECEIVED} {
			scores := tracker.CalculateMetricForIndividuals(&vgproto.DispatchStrategy{AssetForMetric: "asset1", Metric: m, IndividualScope: vgproto.IndividualScope_INDIVIDUAL_SCOPE_ALL, WindowLength: 1, Markets: []string{market}})
			require.Equal(t, 0, len(scores))
		}
	}

	// asset2 one market in scope
	for _, m := range []vgproto.DispatchMetric{vgproto.DispatchMetric_DISPATCH_METRIC_MAKER_FEES_PAID, vgproto.DispatchMetric_DISPATCH_METRIC_LP_FEES_RECEIVED, vgproto.DispatchMetric_DISPATCH_METRIC_MAKER_FEES_RECEIVED} {
		scores := tracker.CalculateMetricForIndividuals(&vgproto.DispatchStrategy{AssetForMetric: "asset1", Metric: m, IndividualScope: vgproto.IndividualScope_INDIVIDUAL_SCOPE_ALL, WindowLength: 1, Markets: []string{"market3"}})
		require.Equal(t, 0, len(scores))
	}

	epochService.target(context.Background(), types.Epoch{Seq: 2, Action: vgproto.EpochAction_EPOCH_ACTION_START})

	// update with a few transfers
	transfersM1 := []*types.Transfer{
		{Owner: "party1", Type: types.TransferTypeMakerFeeReceive, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.NewUint(100)}},
		{Owner: "party1", Type: types.TransferTypeMakerFeePay, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.NewUint(200)}},
		{Owner: "party1", Type: types.TransferTypeLiquidityFeeDistribute, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.NewUint(200)}},
		{Owner: "party1", Type: types.TransferTypeMakerFeeReceive, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.NewUint(400)}},
		{Owner: "party1", Type: types.TransferTypeMakerFeePay, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.NewUint(300)}},
		{Owner: "party1", Type: types.TransferTypeLiquidityFeeDistribute, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.NewUint(600)}},
		{Owner: "party2", Type: types.TransferTypeMakerFeeReceive, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.NewUint(900)}},
		{Owner: "party2", Type: types.TransferTypeMakerFeePay, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.NewUint(800)}},
		{Owner: "party2", Type: types.TransferTypeLiquidityFeeDistribute, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.NewUint(700)}},
		{Owner: "party2", Type: types.TransferTypeMakerFeeReceive, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.NewUint(600)}},
		{Owner: "party2", Type: types.TransferTypeMakerFeePay, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.NewUint(200)}},
		{Owner: "party2", Type: types.TransferTypeLiquidityFeeDistribute, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.NewUint(1000)}},
	}
	tracker.UpdateFeesFromTransfers("asset1", "market1", transfersM1)

	transfersM2 := []*types.Transfer{
		{Owner: "party1", Type: types.TransferTypeMakerFeeReceive, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.NewUint(500)}},
		{Owner: "party2", Type: types.TransferTypeMakerFeePay, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.NewUint(1500)}},
		{Owner: "party2", Type: types.TransferTypeLiquidityFeeDistribute, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.NewUint(1500)}},
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
	scores := tracker.CalculateMetricForIndividuals(&vgproto.DispatchStrategy{AssetForMetric: "asset1", Metric: vgproto.DispatchMetric_DISPATCH_METRIC_LP_FEES_RECEIVED, IndividualScope: vgproto.IndividualScope_INDIVIDUAL_SCOPE_ALL, WindowLength: 1})
	require.Equal(t, 2, len(scores))

	require.Equal(t, "party1", scores[0].Party)
	require.Equal(t, "0.2", scores[0].Score.String())
	require.Equal(t, "party2", scores[1].Party)
	require.Equal(t, "0.8", scores[1].Score.String())

	// now look only on market 1:
	// party1 = 800/2500 = 0.32
	// partt2 = 1700/2500 = 0.68
	scores = tracker.CalculateMetricForIndividuals(&vgproto.DispatchStrategy{AssetForMetric: "asset1", Metric: vgproto.DispatchMetric_DISPATCH_METRIC_LP_FEES_RECEIVED, IndividualScope: vgproto.IndividualScope_INDIVIDUAL_SCOPE_ALL, WindowLength: 1, Markets: []string{"market1"}})
	require.Equal(t, 2, len(scores))

	require.Equal(t, "party1", scores[0].Party)
	require.Equal(t, "0.32", scores[0].Score.String())
	require.Equal(t, "party2", scores[1].Party)
	require.Equal(t, "0.68", scores[1].Score.String())

	// now look only on market 2:
	scores = tracker.CalculateMetricForIndividuals(&vgproto.DispatchStrategy{AssetForMetric: "asset1", Metric: vgproto.DispatchMetric_DISPATCH_METRIC_LP_FEES_RECEIVED, IndividualScope: vgproto.IndividualScope_INDIVIDUAL_SCOPE_ALL, WindowLength: 1, Markets: []string{"market2"}})
	require.Equal(t, 1, len(scores))

	require.Equal(t, "party2", scores[0].Party)
	require.Equal(t, "1", scores[0].Score.String())

	// now look at asset2 with no market qualifer
	scores = tracker.CalculateMetricForIndividuals(&vgproto.DispatchStrategy{AssetForMetric: "asset2", Metric: vgproto.DispatchMetric_DISPATCH_METRIC_MAKER_FEES_PAID, IndividualScope: vgproto.IndividualScope_INDIVIDUAL_SCOPE_ALL, WindowLength: 1})
	require.Equal(t, 1, len(scores))
	require.Equal(t, "party1", scores[0].Party)
	require.Equal(t, "1", scores[0].Score.String())

	epochService.target(context.Background(), types.Epoch{Seq: 3, Action: vgproto.EpochAction_EPOCH_ACTION_START})
	transfersM1 = []*types.Transfer{
		{Owner: "party1", Type: types.TransferTypeLiquidityFeeDistribute, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.NewUint(1200)}},
	}
	transfersM2 = []*types.Transfer{
		{Owner: "party2", Type: types.TransferTypeLiquidityFeeDistribute, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.NewUint(800)}},
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
	scores = tracker.CalculateMetricForIndividuals(&vgproto.DispatchStrategy{AssetForMetric: "asset1", Metric: vgproto.DispatchMetric_DISPATCH_METRIC_LP_FEES_RECEIVED, IndividualScope: vgproto.IndividualScope_INDIVIDUAL_SCOPE_ALL, WindowLength: 2})
	require.Equal(t, 2, len(scores))

	require.Equal(t, "party1", scores[0].Party)
	require.Equal(t, "0.3333333333333333", scores[0].Score.String())
	require.Equal(t, "party2", scores[1].Party)
	require.Equal(t, "0.6666666666666667", scores[1].Score.String())
}

func TestGetScoresIndividualsDifferentScopes(t *testing.T) {
	epochService := &TestEpochEngine{}
	ctrl := gomock.NewController(t)
	teams := mocks.NewMockTeams(ctrl)
	balanceChecker := mocks.NewMockAccountBalanceChecker(ctrl)
	tracker := common.NewMarketActivityTracker(logging.NewTestLogger(), epochService, teams, balanceChecker)
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
			scores := tracker.CalculateMetricForIndividuals(&vgproto.DispatchStrategy{AssetForMetric: asset, Metric: m, IndividualScope: vgproto.IndividualScope_INDIVIDUAL_SCOPE_ALL, WindowLength: 1})
			require.Equal(t, 0, len(scores))
		}
	}

	// asset1 one market in scope
	for _, market := range []string{"market1", "market2", "market4"} {
		for _, m := range []vgproto.DispatchMetric{vgproto.DispatchMetric_DISPATCH_METRIC_MAKER_FEES_PAID, vgproto.DispatchMetric_DISPATCH_METRIC_LP_FEES_RECEIVED, vgproto.DispatchMetric_DISPATCH_METRIC_MAKER_FEES_RECEIVED} {
			scores := tracker.CalculateMetricForIndividuals(&vgproto.DispatchStrategy{AssetForMetric: "asset1", Metric: m, IndividualScope: vgproto.IndividualScope_INDIVIDUAL_SCOPE_ALL, WindowLength: 1, Markets: []string{market}})
			require.Equal(t, 0, len(scores))
		}
	}

	// asset2 one market in scope
	for _, m := range []vgproto.DispatchMetric{vgproto.DispatchMetric_DISPATCH_METRIC_MAKER_FEES_PAID, vgproto.DispatchMetric_DISPATCH_METRIC_LP_FEES_RECEIVED, vgproto.DispatchMetric_DISPATCH_METRIC_MAKER_FEES_RECEIVED} {
		scores := tracker.CalculateMetricForIndividuals(&vgproto.DispatchStrategy{AssetForMetric: "asset1", Metric: m, IndividualScope: vgproto.IndividualScope_INDIVIDUAL_SCOPE_ALL, WindowLength: 1, Markets: []string{"market3"}})
		require.Equal(t, 0, len(scores))
	}

	epochService.target(context.Background(), types.Epoch{Seq: 2, Action: vgproto.EpochAction_EPOCH_ACTION_START})

	// update with a few transfers
	transfersM1 := []*types.Transfer{
		{Owner: "party1", Type: types.TransferTypeMakerFeeReceive, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.NewUint(100)}},
		{Owner: "party1", Type: types.TransferTypeMakerFeePay, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.NewUint(200)}},
		{Owner: "party1", Type: types.TransferTypeLiquidityFeeDistribute, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.NewUint(200)}},
		{Owner: "party1", Type: types.TransferTypeMakerFeeReceive, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.NewUint(400)}},
		{Owner: "party1", Type: types.TransferTypeMakerFeePay, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.NewUint(300)}},
		{Owner: "party1", Type: types.TransferTypeLiquidityFeeDistribute, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.NewUint(600)}},
		{Owner: "party2", Type: types.TransferTypeMakerFeeReceive, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.NewUint(900)}},
		{Owner: "party2", Type: types.TransferTypeMakerFeePay, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.NewUint(800)}},
		{Owner: "party2", Type: types.TransferTypeLiquidityFeeDistribute, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.NewUint(700)}},
		{Owner: "party2", Type: types.TransferTypeMakerFeeReceive, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.NewUint(600)}},
		{Owner: "party2", Type: types.TransferTypeMakerFeePay, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.NewUint(200)}},
		{Owner: "party2", Type: types.TransferTypeLiquidityFeeDistribute, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.NewUint(1000)}},
	}
	tracker.UpdateFeesFromTransfers("asset1", "market1", transfersM1)

	transfersM2 := []*types.Transfer{
		{Owner: "party1", Type: types.TransferTypeMakerFeeReceive, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.NewUint(500)}},
		{Owner: "party2", Type: types.TransferTypeMakerFeePay, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.NewUint(1500)}},
		{Owner: "party2", Type: types.TransferTypeLiquidityFeeDistribute, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.NewUint(1500)}},
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
	scores := tracker.CalculateMetricForIndividuals(&vgproto.DispatchStrategy{AssetForMetric: "asset1", Metric: vgproto.DispatchMetric_DISPATCH_METRIC_LP_FEES_RECEIVED, IndividualScope: vgproto.IndividualScope_INDIVIDUAL_SCOPE_ALL, WindowLength: 1})
	require.Equal(t, 2, len(scores))

	require.Equal(t, "party1", scores[0].Party)
	require.Equal(t, "0.2", scores[0].Score.String())
	require.Equal(t, "party2", scores[1].Party)
	require.Equal(t, "0.8", scores[1].Score.String())

	// now look only on market 1:
	// party1 = 800/2500 = 0.32
	// partt2 = 1700/2500 = 0.68
	scores = tracker.CalculateMetricForIndividuals(&vgproto.DispatchStrategy{AssetForMetric: "asset1", Metric: vgproto.DispatchMetric_DISPATCH_METRIC_LP_FEES_RECEIVED, IndividualScope: vgproto.IndividualScope_INDIVIDUAL_SCOPE_ALL, WindowLength: 1, Markets: []string{"market1"}})
	require.Equal(t, 2, len(scores))

	require.Equal(t, "party1", scores[0].Party)
	require.Equal(t, "0.32", scores[0].Score.String())
	require.Equal(t, "party2", scores[1].Party)
	require.Equal(t, "0.68", scores[1].Score.String())

	// now look only on market 2:
	scores = tracker.CalculateMetricForIndividuals(&vgproto.DispatchStrategy{AssetForMetric: "asset1", Metric: vgproto.DispatchMetric_DISPATCH_METRIC_LP_FEES_RECEIVED, IndividualScope: vgproto.IndividualScope_INDIVIDUAL_SCOPE_ALL, WindowLength: 1, Markets: []string{"market2"}})
	require.Equal(t, 1, len(scores))

	require.Equal(t, "party2", scores[0].Party)
	require.Equal(t, "1", scores[0].Score.String())

	// now look at asset2 with no market qualifer
	scores = tracker.CalculateMetricForIndividuals(&vgproto.DispatchStrategy{AssetForMetric: "asset2", Metric: vgproto.DispatchMetric_DISPATCH_METRIC_MAKER_FEES_PAID, IndividualScope: vgproto.IndividualScope_INDIVIDUAL_SCOPE_ALL, WindowLength: 1})
	require.Equal(t, 1, len(scores))
	require.Equal(t, "party1", scores[0].Party)
	require.Equal(t, "1", scores[0].Score.String())

	epochService.target(context.Background(), types.Epoch{Seq: 3, Action: vgproto.EpochAction_EPOCH_ACTION_START})
	transfersM1 = []*types.Transfer{
		{Owner: "party1", Type: types.TransferTypeLiquidityFeeDistribute, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.NewUint(1200)}},
	}
	transfersM2 = []*types.Transfer{
		{Owner: "party2", Type: types.TransferTypeLiquidityFeeDistribute, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.NewUint(800)}},
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
	scores = tracker.CalculateMetricForIndividuals(&vgproto.DispatchStrategy{AssetForMetric: "asset1", Metric: vgproto.DispatchMetric_DISPATCH_METRIC_LP_FEES_RECEIVED, IndividualScope: vgproto.IndividualScope_INDIVIDUAL_SCOPE_ALL, WindowLength: 2})
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
	tracker := common.NewMarketActivityTracker(logging.NewTestLogger(), &TestEpochEngine{}, teams, balanceChecker)
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
	ctrl := gomock.NewController(t)
	teams := mocks.NewMockTeams(ctrl)
	balanceChecker := mocks.NewMockAccountBalanceChecker(ctrl)
	tracker := common.NewMarketActivityTracker(logging.NewTestLogger(), epochEngine, teams, balanceChecker)
	epochEngine.target(context.Background(), types.Epoch{Seq: 1, Action: vgproto.EpochAction_EPOCH_ACTION_START})

	tracker.MarketProposed("asset1", "market1", "me")
	transfersM1 := []*types.Transfer{
		{Owner: "party1", Type: types.TransferTypeMakerFeeReceive, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.UintZero()}},
		{Owner: "party1", Type: types.TransferTypeMakerFeePay, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.UintZero()}},
		{Owner: "party1", Type: types.TransferTypeLiquidityFeeDistribute, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.UintZero()}},
		{Owner: "party1", Type: types.TransferTypeMakerFeeReceive, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.UintZero()}},
		{Owner: "party1", Type: types.TransferTypeMakerFeePay, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.UintZero()}},
		{Owner: "party1", Type: types.TransferTypeLiquidityFeeDistribute, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.UintZero()}},
		{Owner: "party2", Type: types.TransferTypeMakerFeeReceive, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.UintZero()}},
		{Owner: "party2", Type: types.TransferTypeMakerFeePay, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.UintZero()}},
		{Owner: "party2", Type: types.TransferTypeLiquidityFeeDistribute, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.UintZero()}},
		{Owner: "party2", Type: types.TransferTypeMakerFeeReceive, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.UintZero()}},
		{Owner: "party2", Type: types.TransferTypeMakerFeePay, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.UintZero()}},
		{Owner: "party2", Type: types.TransferTypeLiquidityFeeDistribute, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.UintZero()}},
	}
	tracker.UpdateFeesFromTransfers("asset1", "market1", transfersM1)
	epochEngine.target(context.Background(), types.Epoch{Seq: 1, Action: vgproto.EpochAction_EPOCH_ACTION_END})
	scores := tracker.CalculateMetricForIndividuals(&vgproto.DispatchStrategy{AssetForMetric: "asset1", Metric: vgproto.DispatchMetric_DISPATCH_METRIC_MAKER_FEES_RECEIVED, IndividualScope: vgproto.IndividualScope_INDIVIDUAL_SCOPE_ALL, WindowLength: 1, Markets: []string{"market1"}})
	require.Equal(t, 0, len(scores))
}

func TestFeesTracker(t *testing.T) {
	epochEngine := &TestEpochEngine{}
	ctrl := gomock.NewController(t)
	teams := mocks.NewMockTeams(ctrl)
	balanceChecker := mocks.NewMockAccountBalanceChecker(ctrl)
	tracker := common.NewMarketActivityTracker(logging.NewTestLogger(), epochEngine, teams, balanceChecker)
	epochEngine.target(context.Background(), types.Epoch{Seq: 1, Action: vgproto.EpochAction_EPOCH_ACTION_START})
	tracker.SetEligibilityChecker(&EligibilityChecker{})

	partyScores := tracker.CalculateMetricForIndividuals(&vgproto.DispatchStrategy{AssetForMetric: "a1", Metric: vgproto.DispatchMetric_DISPATCH_METRIC_MAKER_FEES_RECEIVED, IndividualScope: vgproto.IndividualScope_INDIVIDUAL_SCOPE_ALL, WindowLength: 3})
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
		{Owner: "party1", Type: types.TransferTypeLiquidityFeeDistribute, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.NewUint(200)}},
		{Owner: "party1", Type: types.TransferTypeMakerFeeReceive, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.NewUint(400)}},
		{Owner: "party1", Type: types.TransferTypeMakerFeePay, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.NewUint(300)}},
		{Owner: "party1", Type: types.TransferTypeLiquidityFeeDistribute, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.NewUint(600)}},
		{Owner: "party2", Type: types.TransferTypeMakerFeeReceive, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.NewUint(900)}},
		{Owner: "party2", Type: types.TransferTypeMakerFeePay, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.NewUint(800)}},
		{Owner: "party2", Type: types.TransferTypeLiquidityFeeDistribute, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.NewUint(700)}},
		{Owner: "party2", Type: types.TransferTypeMakerFeeReceive, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.NewUint(600)}},
		{Owner: "party2", Type: types.TransferTypeMakerFeePay, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.NewUint(200)}},
		{Owner: "party2", Type: types.TransferTypeLiquidityFeeDistribute, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.NewUint(1000)}},
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
	scores := tracker.CalculateMetricForIndividuals(&vgproto.DispatchStrategy{AssetForMetric: "asset1", Metric: vgproto.DispatchMetric_DISPATCH_METRIC_MAKER_FEES_RECEIVED, IndividualScope: vgproto.IndividualScope_INDIVIDUAL_SCOPE_ALL, WindowLength: 1, Markets: []string{"market1"}})
	require.Equal(t, 2, len(scores))
	require.Equal(t, "0.25", scores[0].Score.String())
	require.Equal(t, "party1", scores[0].Party)
	require.Equal(t, "0.75", scores[1].Score.String())
	require.Equal(t, "party2", scores[1].Party)

	// asset1 TransferTypeMakerFeePay
	// party1 paid 500
	// party2 paid 1000
	scores = tracker.CalculateMetricForIndividuals(&vgproto.DispatchStrategy{AssetForMetric: "asset1", Metric: vgproto.DispatchMetric_DISPATCH_METRIC_MAKER_FEES_PAID, IndividualScope: vgproto.IndividualScope_INDIVIDUAL_SCOPE_ALL, WindowLength: 1, Markets: []string{"market1"}})
	require.Equal(t, 2, len(scores))
	require.Equal(t, "0.3333333333333333", scores[0].Score.String())
	require.Equal(t, "party1", scores[0].Party)
	require.Equal(t, "0.6666666666666667", scores[1].Score.String())
	require.Equal(t, "party2", scores[1].Party)

	// asset1 TransferTypeLiquidityFeeDistribute
	// party1 paid 800
	// party2 paid 1700
	scores = tracker.CalculateMetricForIndividuals(&vgproto.DispatchStrategy{AssetForMetric: "asset1", Metric: vgproto.DispatchMetric_DISPATCH_METRIC_LP_FEES_RECEIVED, IndividualScope: vgproto.IndividualScope_INDIVIDUAL_SCOPE_ALL, WindowLength: 1, Markets: []string{"market1"}})
	require.Equal(t, 2, len(scores))
	require.Equal(t, "0.32", scores[0].Score.String())
	require.Equal(t, "party1", scores[0].Party)
	require.Equal(t, "0.68", scores[1].Score.String())
	require.Equal(t, "party2", scores[1].Party)

	// asset2 TransferTypeMakerFeePay
	scores = tracker.CalculateMetricForIndividuals(&vgproto.DispatchStrategy{AssetForMetric: "asset1", Metric: vgproto.DispatchMetric_DISPATCH_METRIC_MAKER_FEES_RECEIVED, IndividualScope: vgproto.IndividualScope_INDIVIDUAL_SCOPE_ALL, WindowLength: 1, Markets: []string{"market2"}})
	require.Equal(t, 1, len(scores))
	require.Equal(t, "1", scores[0].Score.String())
	require.Equal(t, "party1", scores[0].Party)

	// asset2 TransferTypeMakerFeePay
	scores = tracker.CalculateMetricForIndividuals(&vgproto.DispatchStrategy{AssetForMetric: "asset1", Metric: vgproto.DispatchMetric_DISPATCH_METRIC_MAKER_FEES_PAID, IndividualScope: vgproto.IndividualScope_INDIVIDUAL_SCOPE_ALL, WindowLength: 1, Markets: []string{"market2"}})
	require.Equal(t, 1, len(scores))
	require.Equal(t, "1", scores[0].Score.String())
	require.Equal(t, "party2", scores[0].Party)

	// check state has changed
	state2, _, err := tracker.GetState(key)
	require.NoError(t, err)
	require.False(t, bytes.Equal(state1, state2))

	epochEngineLoad := &TestEpochEngine{}
	ctrl = gomock.NewController(t)
	teams = mocks.NewMockTeams(ctrl)
	balanceChecker = mocks.NewMockAccountBalanceChecker(ctrl)
	trackerLoad := common.NewMarketActivityTracker(logging.NewTestLogger(), epochEngineLoad, teams, balanceChecker)

	pl := snapshotpb.Payload{}
	require.NoError(t, proto.Unmarshal(state2, &pl))
	trackerLoad.LoadState(context.Background(), types.PayloadFromProto(&pl))

	state3, _, err := trackerLoad.GetState(key)
	require.NoError(t, err)
	require.True(t, bytes.Equal(state2, state3))

	// check a restored party exist in the restored engine
	scores = tracker.CalculateMetricForIndividuals(&vgproto.DispatchStrategy{AssetForMetric: "asset1", Metric: vgproto.DispatchMetric_DISPATCH_METRIC_MAKER_FEES_RECEIVED, IndividualScope: vgproto.IndividualScope_INDIVIDUAL_SCOPE_ALL, WindowLength: 1, Markets: []string{"market2"}})
	require.Equal(t, 1, len(scores))
	require.Equal(t, "1", scores[0].Score.String())
	require.Equal(t, "party1", scores[0].Party)

	// end the epoch
	epochEngineLoad.target(context.Background(), types.Epoch{Seq: 2, Action: vgproto.EpochAction_EPOCH_ACTION_END})

	// // NewEngine epoch should scrub the state an produce a difference hash
	state4, _, err := trackerLoad.GetState(key)
	require.NoError(t, err)
	require.False(t, bytes.Equal(state3, state4))

	// // new epoch, we expect the metrics to have been reset

	metrics := []vgproto.DispatchMetric{vgproto.DispatchMetric_DISPATCH_METRIC_MAKER_FEES_PAID, vgproto.DispatchMetric_DISPATCH_METRIC_LP_FEES_RECEIVED, vgproto.DispatchMetric_DISPATCH_METRIC_MAKER_FEES_RECEIVED}
	for _, m := range metrics {
		scores = trackerLoad.CalculateMetricForIndividuals(&vgproto.DispatchStrategy{AssetForMetric: "asset1", Metric: m, IndividualScope: vgproto.IndividualScope_INDIVIDUAL_SCOPE_ALL, WindowLength: 1, Markets: []string{"market1"}})
		require.Equal(t, 0, len(scores))
		scores = trackerLoad.CalculateMetricForIndividuals(&vgproto.DispatchStrategy{AssetForMetric: "asset1", Metric: m, IndividualScope: vgproto.IndividualScope_INDIVIDUAL_SCOPE_ALL, WindowLength: 1, Markets: []string{"market2"}})
		require.Equal(t, 0, len(scores))
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
	trackerLoad := common.NewMarketActivityTracker(logging.NewTestLogger(), &TestEpochEngine{}, teams, balanceChecker)
	pl := snapshotpb.Payload{}
	require.NoError(t, proto.Unmarshal(state1, &pl))

	trackerLoad.LoadState(context.Background(), types.PayloadFromProto(&pl))
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
	trackerLoad := common.NewMarketActivityTracker(logging.NewTestLogger(), &TestEpochEngine{}, teams, balanceChecker)
	trackerLoad.Load(context.Background(), b)

	bLoad, err := trackerLoad.Checkpoint()
	require.NoError(t, err)
	require.True(t, bytes.Equal(b, bLoad))
}

func TestSnapshotRoundTripViaEngine(t *testing.T) {
	transfersM5 := []*types.Transfer{
		{Owner: "party3", Type: types.TransferTypeMakerFeeReceive, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.NewUint(100)}},
		{Owner: "party3", Type: types.TransferTypeMakerFeePay, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.NewUint(200)}},
		{Owner: "party3", Type: types.TransferTypeLiquidityFeeDistribute, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.NewUint(200)}},
	}
	transfersM6 := []*types.Transfer{
		{Owner: "party4", Type: types.TransferTypeMakerFeeReceive, Amount: &types.FinancialAmount{Asset: "asset2", Amount: num.NewUint(500)}},
		{Owner: "party4", Type: types.TransferTypeMakerFeePay, Amount: &types.FinancialAmount{Asset: "asset2", Amount: num.NewUint(1500)}},
		{Owner: "party4", Type: types.TransferTypeLiquidityFeeDistribute, Amount: &types.FinancialAmount{Asset: "asset2", Amount: num.NewUint(1500)}},
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
	tracker2 := common.NewMarketActivityTracker(logging.NewTestLogger(), &TestEpochEngine{}, teams, balanceChecker)
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
	tracker := common.NewMarketActivityTracker(logging.NewTestLogger(), epochService, teams, balanceChecker)
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

func TestPositionMetric(t *testing.T) {
	epochService := &TestEpochEngine{}
	ctrl := gomock.NewController(t)
	teams := mocks.NewMockTeams(ctrl)
	balanceChecker := mocks.NewMockAccountBalanceChecker(ctrl)
	tracker := common.NewMarketActivityTracker(logging.NewTestLogger(), epochService, teams, balanceChecker)

	epochStartTime := time.Now()
	epochService.target(context.Background(), types.Epoch{Action: vgproto.EpochAction_EPOCH_ACTION_START, StartTime: epochStartTime})
	tracker.SetEligibilityChecker(&EligibilityChecker{})
	tracker.MarketProposed("a1", "m1", "p1")

	// 100 seconds into the epoch record a position of 100
	tracker.RecordPosition("a1", "p1", "m1", num.DecimalFromInt64(100), num.NewUint(1), epochStartTime.Add(100*time.Second))

	// 200 seconds later record another position
	// pBar = 100 * 300/400 = 75
	tracker.RecordPosition("a1", "p1", "m1", num.DecimalFromInt64(-200), num.NewUint(2), epochStartTime.Add(400*time.Second))

	// now end the epoch after 600 seconds
	// pBar = (1 - 600/1000) * 75 + 200 * 600/1000 = 150
	epochService.target(context.Background(), types.Epoch{Action: vgproto.EpochAction_EPOCH_ACTION_END, StartTime: epochStartTime, EndTime: epochStartTime.Add(1000 * time.Second)})

	scores := tracker.CalculateMetricForIndividuals(&vgproto.DispatchStrategy{AssetForMetric: "a1", Metric: vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_POSITION, IndividualScope: vgproto.IndividualScope_INDIVIDUAL_SCOPE_ALL, WindowLength: 1})
	require.Equal(t, 1, len(scores))
	require.Equal(t, "p1", scores[0].Party)
	require.Equal(t, "150", scores[0].Score.String())

	scores = tracker.CalculateMetricForIndividuals(&vgproto.DispatchStrategy{AssetForMetric: "a1", Metric: vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_POSITION, IndividualScope: vgproto.IndividualScope_INDIVIDUAL_SCOPE_ALL, WindowLength: 2})
	require.Equal(t, 1, len(scores))
	require.Equal(t, "p1", scores[0].Party)
	require.Equal(t, "75", scores[0].Score.String())

	scores = tracker.CalculateMetricForIndividuals(&vgproto.DispatchStrategy{AssetForMetric: "a1", Metric: vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_POSITION, IndividualScope: vgproto.IndividualScope_INDIVIDUAL_SCOPE_ALL, WindowLength: 3})
	require.Equal(t, 1, len(scores))
	require.Equal(t, "p1", scores[0].Party)
	require.Equal(t, "50", scores[0].Score.String())

	// qualifying the market to m1, expect the same result
	scores = tracker.CalculateMetricForIndividuals(&vgproto.DispatchStrategy{AssetForMetric: "a1", Metric: vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_POSITION, IndividualScope: vgproto.IndividualScope_INDIVIDUAL_SCOPE_ALL, WindowLength: 3, Markets: []string{"m1"}})
	require.Equal(t, 1, len(scores))
	require.Equal(t, "p1", scores[0].Party)
	require.Equal(t, "50", scores[0].Score.String())

	// qualifying the market to m2, expect the same result
	scores = tracker.CalculateMetricForIndividuals(&vgproto.DispatchStrategy{AssetForMetric: "a1", Metric: vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_POSITION, IndividualScope: vgproto.IndividualScope_INDIVIDUAL_SCOPE_ALL, WindowLength: 3, Markets: []string{"m2"}})
	require.Equal(t, 0, len(scores))

	epochService.target(context.Background(), types.Epoch{Action: vgproto.EpochAction_EPOCH_ACTION_START, StartTime: epochStartTime.Add(1000 * time.Second)})

	// 600 seconds into the epoch new position is recorded for p1
	// pBar = 0 * 150 + 1 * 200 = 200
	tracker.RecordPosition("a1", "p1", "m1", num.DecimalFromInt64(100), num.NewUint(3), epochStartTime.Add(1600*time.Second))

	// end the epoch
	// pBar = 0.6 * 200 + 0.4 * 100 = 160
	epochService.target(context.Background(), types.Epoch{Action: vgproto.EpochAction_EPOCH_ACTION_END, StartTime: epochStartTime.Add(1000 * time.Second), EndTime: epochStartTime.Add(2000 * time.Second)})
	scores = tracker.CalculateMetricForIndividuals(&vgproto.DispatchStrategy{AssetForMetric: "a1", Metric: vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_POSITION, IndividualScope: vgproto.IndividualScope_INDIVIDUAL_SCOPE_ALL, WindowLength: 1})
	require.Equal(t, 1, len(scores))
	require.Equal(t, "p1", scores[0].Party)
	require.Equal(t, "160", scores[0].Score.String())

	scores = tracker.CalculateMetricForIndividuals(&vgproto.DispatchStrategy{AssetForMetric: "a1", Metric: vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_POSITION, IndividualScope: vgproto.IndividualScope_INDIVIDUAL_SCOPE_ALL, WindowLength: 2})
	require.Equal(t, 1, len(scores))
	require.Equal(t, "p1", scores[0].Party)
	require.Equal(t, "155", scores[0].Score.String())

	scores = tracker.CalculateMetricForIndividuals(&vgproto.DispatchStrategy{AssetForMetric: "a1", Metric: vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_POSITION, IndividualScope: vgproto.IndividualScope_INDIVIDUAL_SCOPE_ALL, WindowLength: 5})
	require.Equal(t, 1, len(scores))
	require.Equal(t, "p1", scores[0].Party)
	require.Equal(t, "62", scores[0].Score.String())

	// qualify to m1
	scores = tracker.CalculateMetricForIndividuals(&vgproto.DispatchStrategy{AssetForMetric: "a1", Metric: vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_POSITION, IndividualScope: vgproto.IndividualScope_INDIVIDUAL_SCOPE_ALL, WindowLength: 5, Markets: []string{"m1"}})
	require.Equal(t, 1, len(scores))
	require.Equal(t, "p1", scores[0].Party)
	require.Equal(t, "62", scores[0].Score.String())

	// qualify to m2
	scores = tracker.CalculateMetricForIndividuals(&vgproto.DispatchStrategy{AssetForMetric: "a1", Metric: vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_POSITION, IndividualScope: vgproto.IndividualScope_INDIVIDUAL_SCOPE_ALL, WindowLength: 5, Markets: []string{"m2"}})
	require.Equal(t, 0, len(scores))

	// now lets lets at an epoch with no activity
	epochService.target(context.Background(), types.Epoch{Action: vgproto.EpochAction_EPOCH_ACTION_START, StartTime: epochStartTime.Add(2000 * time.Second)})
	// end the epoch
	// pBar = 0 * 160 + 1 * 100 = 100
	epochService.target(context.Background(), types.Epoch{Action: vgproto.EpochAction_EPOCH_ACTION_END, StartTime: epochStartTime.Add(2000 * time.Second), EndTime: epochStartTime.Add(3000 * time.Second)})
	scores = tracker.CalculateMetricForIndividuals(&vgproto.DispatchStrategy{AssetForMetric: "a1", Metric: vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_POSITION, IndividualScope: vgproto.IndividualScope_INDIVIDUAL_SCOPE_ALL, WindowLength: 1})
	require.Equal(t, 1, len(scores))
	require.Equal(t, "p1", scores[0].Party)
	require.Equal(t, "100", scores[0].Score.String())

	scores = tracker.CalculateMetricForIndividuals(&vgproto.DispatchStrategy{AssetForMetric: "a1", Metric: vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_POSITION, IndividualScope: vgproto.IndividualScope_INDIVIDUAL_SCOPE_ALL, WindowLength: 2})
	require.Equal(t, 1, len(scores))
	require.Equal(t, "p1", scores[0].Party)
	require.Equal(t, "130", scores[0].Score.String())

	scores = tracker.CalculateMetricForIndividuals(&vgproto.DispatchStrategy{AssetForMetric: "a1", Metric: vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_POSITION, IndividualScope: vgproto.IndividualScope_INDIVIDUAL_SCOPE_ALL, WindowLength: 4})
	require.Equal(t, 1, len(scores))
	require.Equal(t, "p1", scores[0].Party)
	require.Equal(t, "102.5", scores[0].Score.String())
}

func TestRelativeReturnMetric(t *testing.T) {
	epochService := &TestEpochEngine{}
	ctrl := gomock.NewController(t)
	teams := mocks.NewMockTeams(ctrl)
	balanceChecker := mocks.NewMockAccountBalanceChecker(ctrl)

	tracker := common.NewMarketActivityTracker(logging.NewTestLogger(), epochService, teams, balanceChecker)
	epochStartTime := time.Now()
	epochService.target(context.Background(), types.Epoch{Action: vgproto.EpochAction_EPOCH_ACTION_START, StartTime: epochStartTime})
	tracker.SetEligibilityChecker(&EligibilityChecker{})
	tracker.MarketProposed("a1", "m1", "p1")

	tracker.RecordPosition("a1", "p1", "m1", num.DecimalFromInt64(100), num.NewUint(1), epochStartTime.Add(100*time.Second))
	tracker.RecordPosition("a1", "p1", "m1", num.DecimalFromInt64(-200), num.NewUint(2), epochStartTime.Add(400*time.Second))

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
	scores := tracker.CalculateMetricForIndividuals(&vgproto.DispatchStrategy{AssetForMetric: "a1", Metric: vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, IndividualScope: vgproto.IndividualScope_INDIVIDUAL_SCOPE_ALL, WindowLength: 1})
	require.Equal(t, 1, len(scores))
	require.Equal(t, "p1", scores[0].Party)
	require.Equal(t, "0.1", scores[0].Score.String())

	// window size 2
	scores = tracker.CalculateMetricForIndividuals(&vgproto.DispatchStrategy{AssetForMetric: "a1", Metric: vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, IndividualScope: vgproto.IndividualScope_INDIVIDUAL_SCOPE_ALL, WindowLength: 2})
	require.Equal(t, 1, len(scores))
	require.Equal(t, "p1", scores[0].Party)
	require.Equal(t, "0.05", scores[0].Score.String())

	// window size 3
	scores = tracker.CalculateMetricForIndividuals(&vgproto.DispatchStrategy{AssetForMetric: "a1", Metric: vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, IndividualScope: vgproto.IndividualScope_INDIVIDUAL_SCOPE_ALL, WindowLength: 3})
	require.Equal(t, 1, len(scores))
	require.Equal(t, "p1", scores[0].Party)
	require.Equal(t, "0.0333333333333333", scores[0].Score.String())

	// add only this market in scope, expect same result
	scores = tracker.CalculateMetricForIndividuals(&vgproto.DispatchStrategy{AssetForMetric: "a1", Metric: vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, IndividualScope: vgproto.IndividualScope_INDIVIDUAL_SCOPE_ALL, WindowLength: 3, Markets: []string{"m1"}})
	require.Equal(t, 1, len(scores))
	require.Equal(t, "p1", scores[0].Party)
	require.Equal(t, "0.0333333333333333", scores[0].Score.String())

	// add market scope with the wrong market:
	scores = tracker.CalculateMetricForIndividuals(&vgproto.DispatchStrategy{AssetForMetric: "a1", Metric: vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, IndividualScope: vgproto.IndividualScope_INDIVIDUAL_SCOPE_ALL, WindowLength: 3, Markets: []string{"m2"}})
	require.Equal(t, 0, len(scores))

	// add a scope with a different market, expect nothing back
	scores = tracker.CalculateMetricForIndividuals(&vgproto.DispatchStrategy{AssetForMetric: "a1", Metric: vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, IndividualScope: vgproto.IndividualScope_INDIVIDUAL_SCOPE_ALL, WindowLength: 3, Markets: []string{"m2"}})
	require.Equal(t, 0, len(scores))

	// lets run another epoch and make some loss
	epochService.target(context.Background(), types.Epoch{Action: vgproto.EpochAction_EPOCH_ACTION_START, StartTime: epochStartTime.Add(1000 * time.Second)})
	tracker.RecordPosition("a1", "p1", "m1", num.DecimalFromInt64(100), num.NewUint(2), epochStartTime.Add(1600*time.Second))
	tracker.RecordM2M("a1", "p1", "m1", num.DecimalFromInt64(-8))

	// end the epoch
	epochService.target(context.Background(), types.Epoch{Action: vgproto.EpochAction_EPOCH_ACTION_END, StartTime: epochStartTime.Add(1000 * time.Second), EndTime: epochStartTime.Add(2000 * time.Second)})

	// total m2m = -8
	// the time weighted position for the epoch = 160
	// therefore r = -0.05
	// max(-0.05, 0) = 0 => nothing is returned
	scores = tracker.CalculateMetricForIndividuals(&vgproto.DispatchStrategy{AssetForMetric: "a1", Metric: vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, IndividualScope: vgproto.IndividualScope_INDIVIDUAL_SCOPE_ALL, WindowLength: 1})
	require.Equal(t, 0, len(scores))

	// with window size=2 we get (0.1-0.05)/2 = 0.025
	scores = tracker.CalculateMetricForIndividuals(&vgproto.DispatchStrategy{AssetForMetric: "a1", Metric: vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, IndividualScope: vgproto.IndividualScope_INDIVIDUAL_SCOPE_ALL, WindowLength: 2})
	require.Equal(t, 1, len(scores))
	require.Equal(t, "p1", scores[0].Party)
	require.Equal(t, "0.025", scores[0].Score.String())

	// with window size=4 we get (0.1-0.05)/4 = 0.0125
	scores = tracker.CalculateMetricForIndividuals(&vgproto.DispatchStrategy{AssetForMetric: "a1", Metric: vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, IndividualScope: vgproto.IndividualScope_INDIVIDUAL_SCOPE_ALL, WindowLength: 4})
	require.Equal(t, 1, len(scores))
	require.Equal(t, "p1", scores[0].Party)
	require.Equal(t, "0.0125", scores[0].Score.String())
}

func setupDefaultTrackerForTest(t *testing.T) *common.MarketActivityTracker {
	t.Helper()

	epochService := &TestEpochEngine{}
	ctrl := gomock.NewController(t)
	teams := mocks.NewMockTeams(ctrl)
	balanceChecker := mocks.NewMockAccountBalanceChecker(ctrl)

	tracker := common.NewMarketActivityTracker(logging.NewTestLogger(), epochService, teams, balanceChecker)
	epochStartTime := time.Now()
	epochService.target(context.Background(), types.Epoch{Action: vgproto.EpochAction_EPOCH_ACTION_START, StartTime: epochStartTime})
	tracker.SetEligibilityChecker(&EligibilityChecker{})

	tracker.MarketProposed("asset1", "market1", "me")
	tracker.MarketProposed("asset1", "market2", "me2")
	tracker.MarketProposed("asset1", "market4", "me4")
	tracker.MarketProposed("asset2", "market3", "me3")

	tracker.RecordPosition("asset1", "p1", "market1", num.DecimalFromInt64(100), num.NewUint(1), time.Now())
	tracker.RecordPosition("asset1", "p1", "market2", num.DecimalFromInt64(200), num.NewUint(2), time.Now())
	tracker.RecordPosition("asset1", "p2", "market1", num.DecimalFromInt64(300), num.NewUint(3), time.Now())
	tracker.RecordPosition("asset1", "p3", "market2", num.DecimalFromInt64(400), num.NewUint(4), time.Now())
	tracker.RecordPosition("asset1", "p3", "market4", num.DecimalFromInt64(500), num.NewUint(5), time.Now())
	tracker.RecordPosition("asset2", "p4", "market3", num.DecimalFromInt64(600), num.NewUint(6), time.Now())

	tracker.RecordM2M("asset1", "p1", "market1", num.DecimalOne())
	tracker.RecordM2M("asset1", "p1", "market2", num.DecimalFromInt64(5))

	// update with a few transfers
	transfersM1 := []*types.Transfer{
		{Owner: "party1", Type: types.TransferTypeMakerFeeReceive, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.NewUint(100)}},
		{Owner: "party1", Type: types.TransferTypeMakerFeePay, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.NewUint(200)}},
		{Owner: "party1", Type: types.TransferTypeLiquidityFeeDistribute, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.NewUint(200)}},
		{Owner: "party1", Type: types.TransferTypeMakerFeeReceive, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.NewUint(400)}},
		{Owner: "party1", Type: types.TransferTypeMakerFeePay, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.NewUint(300)}},
		{Owner: "party1", Type: types.TransferTypeLiquidityFeeDistribute, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.NewUint(600)}},
		{Owner: "party2", Type: types.TransferTypeMakerFeeReceive, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.NewUint(900)}},
		{Owner: "party2", Type: types.TransferTypeMakerFeePay, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.NewUint(800)}},
		{Owner: "party2", Type: types.TransferTypeLiquidityFeeDistribute, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.NewUint(700)}},
		{Owner: "party2", Type: types.TransferTypeMakerFeeReceive, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.NewUint(600)}},
		{Owner: "party2", Type: types.TransferTypeMakerFeePay, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.NewUint(200)}},
		{Owner: "party2", Type: types.TransferTypeLiquidityFeeDistribute, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.NewUint(1000)}},
	}
	tracker.UpdateFeesFromTransfers("asset1", "market1", transfersM1)

	transfersM2 := []*types.Transfer{
		{Owner: "party1", Type: types.TransferTypeMakerFeeReceive, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.NewUint(500)}},
		{Owner: "party2", Type: types.TransferTypeMakerFeePay, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.NewUint(1500)}},
		{Owner: "party2", Type: types.TransferTypeLiquidityFeeDistribute, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.NewUint(1500)}},
	}
	tracker.UpdateFeesFromTransfers("asset2", "market2", transfersM2)

	transfersM3 := []*types.Transfer{
		{Owner: "party1", Type: types.TransferTypeMakerFeePay, Amount: &types.FinancialAmount{Asset: "asset2", Amount: num.NewUint(500)}},
		{Owner: "party2", Type: types.TransferTypeMakerFeeReceive, Amount: &types.FinancialAmount{Asset: "asset2", Amount: num.NewUint(450)}},
	}
	tracker.UpdateFeesFromTransfers("asset2", "market3", transfersM3)
	return tracker
}
