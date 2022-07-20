// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package execution_test

import (
	"bytes"
	"context"
	"testing"
	"time"

	vgproto "code.vegaprotocol.io/protos/vega"
	snapshotpb "code.vegaprotocol.io/protos/vega/snapshot/v1"
	"code.vegaprotocol.io/shared/paths"
	"code.vegaprotocol.io/vega/execution"
	"code.vegaprotocol.io/vega/integration/stubs"
	vgcontext "code.vegaprotocol.io/vega/libs/context"
	"code.vegaprotocol.io/vega/libs/proto"
	"code.vegaprotocol.io/vega/logging"
	snp "code.vegaprotocol.io/vega/snapshot"
	"code.vegaprotocol.io/vega/stats"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"
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
	tracker := execution.NewMarketActivityTracker(logging.NewTestLogger(), &TestEpochEngine{})
	tracker.SetEligibilityChecker(&EligibilityChecker{})

	tracker.MarketProposed("asset1", "market1", "me")
	tracker.MarketProposed("asset1", "market2", "me2")

	require.Equal(t, 0, len(tracker.GetEligibleProposers("market1")))
	require.Equal(t, 0, len(tracker.GetEligibleProposers("market2")))

	tracker.AddValueTraded("market1", num.NewUint(1000))
	require.Equal(t, 0, len(tracker.GetEligibleProposers("market1")))
	require.Equal(t, 0, len(tracker.GetEligibleProposers("market2")))

	tracker.AddValueTraded("market2", num.NewUint(4000))
	require.Equal(t, 0, len(tracker.GetEligibleProposers("market1")))
	require.Equal(t, 0, len(tracker.GetEligibleProposers("market2")))

	tracker.AddValueTraded("market2", num.NewUint(1001))
	tracker.AddValueTraded("market1", num.NewUint(4001))

	proposers1 := tracker.GetEligibleProposers("market1")
	require.Equal(t, 1, len(proposers1))
	require.Equal(t, "me", proposers1[0])
	proposers2 := tracker.GetEligibleProposers("market2")
	require.Equal(t, 1, len(proposers2))
	require.Equal(t, "me2", proposers2[0])

	// ask again and expect nothing to be returned
	tracker.MarkPaidProposer("market1")
	tracker.MarkPaidProposer("market2")
	require.Equal(t, 0, len(tracker.GetEligibleProposers("market1")))
	require.Equal(t, 0, len(tracker.GetEligibleProposers("market2")))

	// take a snapshot
	key := (&types.PayloadMarketActivityTracker{}).Key()
	state1, _, err := tracker.GetState(key)
	require.NoError(t, err)

	trackerLoad := execution.NewMarketActivityTracker(logging.NewTestLogger(), &TestEpochEngine{})
	pl := snapshotpb.Payload{}
	require.NoError(t, proto.Unmarshal(state1, &pl))

	trackerLoad.LoadState(context.Background(), types.PayloadFromProto(&pl))

	state2, _, err := trackerLoad.GetState(key)
	require.NoError(t, err)
	require.True(t, bytes.Equal(state1, state2))
}

func TestRemoveMarket(t *testing.T) {
	epochService := &TestEpochEngine{}
	tracker := execution.NewMarketActivityTracker(logging.NewTestLogger(), epochService)
	tracker.SetEligibilityChecker(&EligibilityChecker{})
	tracker.MarketProposed("asset1", "market1", "me")
	tracker.MarketProposed("asset1", "market2", "me2")
	require.Equal(t, 2, len(tracker.GetAllMarketIDs()))
	require.Equal(t, "market1", tracker.GetAllMarketIDs()[0])
	require.Equal(t, "market2", tracker.GetAllMarketIDs()[1])

	// remove the market - this should only mark the market for removal
	tracker.RemoveMarket("market1")
	require.Equal(t, 2, len(tracker.GetAllMarketIDs()))
	require.Equal(t, "market1", tracker.GetAllMarketIDs()[0])
	require.Equal(t, "market2", tracker.GetAllMarketIDs()[1])
	epochService.target(context.Background(), types.Epoch{Action: vgproto.EpochAction_EPOCH_ACTION_START})

	require.Equal(t, 1, len(tracker.GetAllMarketIDs()))
	require.Equal(t, "market2", tracker.GetAllMarketIDs()[0])
}

func TestGetMarketScores(t *testing.T) {
	epochService := &TestEpochEngine{}
	tracker := execution.NewMarketActivityTracker(logging.NewTestLogger(), epochService)
	tracker.SetEligibilityChecker(&EligibilityChecker{})
	tracker.MarketProposed("asset1", "market1", "me")
	tracker.MarketProposed("asset1", "market2", "me2")
	tracker.MarketProposed("asset1", "market4", "me4")
	tracker.MarketProposed("asset2", "market3", "me3")

	// no fees generated expect empty slice
	require.Equal(t, 0, len(tracker.GetMarketScores("asset1", []string{}, vgproto.DispatchMetric_DISPATCH_METRIC_LP_FEES_RECEIVED)))
	require.Equal(t, 0, len(tracker.GetMarketScores("asset1", []string{}, vgproto.DispatchMetric_DISPATCH_METRIC_MAKER_FEES_RECEIVED)))
	require.Equal(t, 0, len(tracker.GetMarketScores("asset1", []string{}, vgproto.DispatchMetric_DISPATCH_METRIC_TAKER_FEES_PAID)))

	require.Equal(t, 0, len(tracker.GetMarketScores("asset1", []string{"market1"}, vgproto.DispatchMetric_DISPATCH_METRIC_LP_FEES_RECEIVED)))
	require.Equal(t, 0, len(tracker.GetMarketScores("asset1", []string{"market1"}, vgproto.DispatchMetric_DISPATCH_METRIC_MAKER_FEES_RECEIVED)))
	require.Equal(t, 0, len(tracker.GetMarketScores("asset1", []string{"market1"}, vgproto.DispatchMetric_DISPATCH_METRIC_TAKER_FEES_PAID)))

	require.Equal(t, 0, len(tracker.GetMarketScores("asset1", []string{"market2"}, vgproto.DispatchMetric_DISPATCH_METRIC_LP_FEES_RECEIVED)))
	require.Equal(t, 0, len(tracker.GetMarketScores("asset1", []string{"market2"}, vgproto.DispatchMetric_DISPATCH_METRIC_MAKER_FEES_RECEIVED)))
	require.Equal(t, 0, len(tracker.GetMarketScores("asset1", []string{"market2"}, vgproto.DispatchMetric_DISPATCH_METRIC_TAKER_FEES_PAID)))

	require.Equal(t, 0, len(tracker.GetMarketScores("asset2", []string{"market3"}, vgproto.DispatchMetric_DISPATCH_METRIC_LP_FEES_RECEIVED)))
	require.Equal(t, 0, len(tracker.GetMarketScores("asset2", []string{"market3"}, vgproto.DispatchMetric_DISPATCH_METRIC_MAKER_FEES_RECEIVED)))
	require.Equal(t, 0, len(tracker.GetMarketScores("asset2", []string{"market3"}, vgproto.DispatchMetric_DISPATCH_METRIC_TAKER_FEES_PAID)))

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
	tracker.UpdateFeesFromTransfers("market1", transfersM1)

	transfersM2 := []*types.Transfer{
		{Owner: "party1", Type: types.TransferTypeMakerFeeReceive, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.NewUint(500)}},
		{Owner: "party2", Type: types.TransferTypeMakerFeePay, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.NewUint(1500)}},
		{Owner: "party2", Type: types.TransferTypeLiquidityFeeDistribute, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.NewUint(1500)}},
	}
	tracker.UpdateFeesFromTransfers("market2", transfersM2)

	transfersM3 := []*types.Transfer{
		{Owner: "party1", Type: types.TransferTypeMakerFeePay, Amount: &types.FinancialAmount{Asset: "asset2", Amount: num.NewUint(500)}},
		{Owner: "party2", Type: types.TransferTypeMakerFeeReceive, Amount: &types.FinancialAmount{Asset: "asset2", Amount: num.NewUint(450)}},
	}
	tracker.UpdateFeesFromTransfers("market3", transfersM3)

	// in market1: 2500
	// in market2: 1500
	// in market4: 0 => it is not included in the scores.
	require.Equal(t, 2, len(tracker.GetMarketScores("asset1", []string{}, vgproto.DispatchMetric_DISPATCH_METRIC_LP_FEES_RECEIVED)))
	LPMarket1 := &types.MarketContributionScore{
		Asset:  "asset1",
		Market: "market1",
		Metric: vgproto.DispatchMetric_DISPATCH_METRIC_LP_FEES_RECEIVED,
		Score:  num.MustDecimalFromString("0.625"),
	}
	LPMarket2 := &types.MarketContributionScore{
		Asset:  "asset1",
		Market: "market2",
		Metric: vgproto.DispatchMetric_DISPATCH_METRIC_LP_FEES_RECEIVED,
		Score:  num.MustDecimalFromString("0.375"),
	}
	require.Equal(t, 2, len(tracker.GetMarketScores("asset1", []string{}, vgproto.DispatchMetric_DISPATCH_METRIC_LP_FEES_RECEIVED)))
	assertMarketContributionScore(t, LPMarket1, tracker.GetMarketScores("asset1", []string{}, vgproto.DispatchMetric_DISPATCH_METRIC_LP_FEES_RECEIVED)[0])
	assertMarketContributionScore(t, LPMarket2, tracker.GetMarketScores("asset1", []string{}, vgproto.DispatchMetric_DISPATCH_METRIC_LP_FEES_RECEIVED)[1])

	// scope only market1:
	require.Equal(t, 1, len(tracker.GetMarketScores("asset1", []string{"market1"}, vgproto.DispatchMetric_DISPATCH_METRIC_LP_FEES_RECEIVED)))
	LPMarket1.Score = num.DecimalFromInt64(1)
	assertMarketContributionScore(t, LPMarket1, tracker.GetMarketScores("asset1", []string{"market1"}, vgproto.DispatchMetric_DISPATCH_METRIC_LP_FEES_RECEIVED)[0])

	// scope only market2:
	require.Equal(t, 1, len(tracker.GetMarketScores("asset1", []string{"market2"}, vgproto.DispatchMetric_DISPATCH_METRIC_LP_FEES_RECEIVED)))
	LPMarket2.Score = num.DecimalFromInt64(1)
	assertMarketContributionScore(t, LPMarket2, tracker.GetMarketScores("asset1", []string{"market2"}, vgproto.DispatchMetric_DISPATCH_METRIC_LP_FEES_RECEIVED)[0])

	// try to scope market3: doesn't exist in the asset
	require.Equal(t, 0, len(tracker.GetMarketScores("asset1", []string{"market3"}, vgproto.DispatchMetric_DISPATCH_METRIC_LP_FEES_RECEIVED)))

	// try to get the market from the wrong asset
	require.Equal(t, 0, len(tracker.GetMarketScores("asset2", []string{"market1"}, vgproto.DispatchMetric_DISPATCH_METRIC_LP_FEES_RECEIVED)))

	// in market1: 2000
	// in market2: 500
	require.Equal(t, 2, len(tracker.GetMarketScores("asset1", []string{}, vgproto.DispatchMetric_DISPATCH_METRIC_MAKER_FEES_RECEIVED)))
	LPMarket1 = &types.MarketContributionScore{
		Asset:  "asset1",
		Market: "market1",
		Metric: vgproto.DispatchMetric_DISPATCH_METRIC_MAKER_FEES_RECEIVED,
		Score:  num.MustDecimalFromString("0.8"),
	}
	LPMarket2 = &types.MarketContributionScore{
		Asset:  "asset1",
		Market: "market2",
		Metric: vgproto.DispatchMetric_DISPATCH_METRIC_MAKER_FEES_RECEIVED,
		Score:  num.MustDecimalFromString("0.2"),
	}
	require.Equal(t, 2, len(tracker.GetMarketScores("asset1", []string{}, vgproto.DispatchMetric_DISPATCH_METRIC_MAKER_FEES_RECEIVED)))
	assertMarketContributionScore(t, LPMarket1, tracker.GetMarketScores("asset1", []string{}, vgproto.DispatchMetric_DISPATCH_METRIC_MAKER_FEES_RECEIVED)[0])
	assertMarketContributionScore(t, LPMarket2, tracker.GetMarketScores("asset1", []string{}, vgproto.DispatchMetric_DISPATCH_METRIC_MAKER_FEES_RECEIVED)[1])

	// scope only market1:
	require.Equal(t, 1, len(tracker.GetMarketScores("asset1", []string{"market1"}, vgproto.DispatchMetric_DISPATCH_METRIC_MAKER_FEES_RECEIVED)))
	LPMarket1.Score = num.DecimalFromInt64(1)
	assertMarketContributionScore(t, LPMarket1, tracker.GetMarketScores("asset1", []string{"market1"}, vgproto.DispatchMetric_DISPATCH_METRIC_MAKER_FEES_RECEIVED)[0])

	// scope only market2:
	require.Equal(t, 1, len(tracker.GetMarketScores("asset1", []string{"market2"}, vgproto.DispatchMetric_DISPATCH_METRIC_MAKER_FEES_RECEIVED)))
	LPMarket2.Score = num.DecimalFromInt64(1)
	assertMarketContributionScore(t, LPMarket2, tracker.GetMarketScores("asset1", []string{"market2"}, vgproto.DispatchMetric_DISPATCH_METRIC_MAKER_FEES_RECEIVED)[0])

	// try to scope market3: doesn't exist in the asset
	require.Equal(t, 0, len(tracker.GetMarketScores("asset1", []string{"market3"}, vgproto.DispatchMetric_DISPATCH_METRIC_MAKER_FEES_RECEIVED)))

	// try to get the market from the wrong asset
	require.Equal(t, 0, len(tracker.GetMarketScores("asset2", []string{"market1"}, vgproto.DispatchMetric_DISPATCH_METRIC_MAKER_FEES_RECEIVED)))

	// in market1: 1500
	// in market2: 1500
	require.Equal(t, 2, len(tracker.GetMarketScores("asset1", []string{}, vgproto.DispatchMetric_DISPATCH_METRIC_TAKER_FEES_PAID)))
	LPMarket1 = &types.MarketContributionScore{
		Asset:  "asset1",
		Market: "market1",
		Metric: vgproto.DispatchMetric_DISPATCH_METRIC_TAKER_FEES_PAID,
		Score:  num.MustDecimalFromString("0.5"),
	}
	LPMarket2 = &types.MarketContributionScore{
		Asset:  "asset1",
		Market: "market2",
		Metric: vgproto.DispatchMetric_DISPATCH_METRIC_TAKER_FEES_PAID,
		Score:  num.MustDecimalFromString("0.5"),
	}
	require.Equal(t, 2, len(tracker.GetMarketScores("asset1", []string{}, vgproto.DispatchMetric_DISPATCH_METRIC_TAKER_FEES_PAID)))
	assertMarketContributionScore(t, LPMarket1, tracker.GetMarketScores("asset1", []string{}, vgproto.DispatchMetric_DISPATCH_METRIC_TAKER_FEES_PAID)[0])
	assertMarketContributionScore(t, LPMarket2, tracker.GetMarketScores("asset1", []string{}, vgproto.DispatchMetric_DISPATCH_METRIC_TAKER_FEES_PAID)[1])

	// scope only market1:
	require.Equal(t, 1, len(tracker.GetMarketScores("asset1", []string{"market1"}, vgproto.DispatchMetric_DISPATCH_METRIC_TAKER_FEES_PAID)))
	LPMarket1.Score = num.DecimalFromInt64(1)
	assertMarketContributionScore(t, LPMarket1, tracker.GetMarketScores("asset1", []string{"market1"}, vgproto.DispatchMetric_DISPATCH_METRIC_TAKER_FEES_PAID)[0])

	// scope only market2:
	require.Equal(t, 1, len(tracker.GetMarketScores("asset1", []string{"market2"}, vgproto.DispatchMetric_DISPATCH_METRIC_TAKER_FEES_PAID)))
	LPMarket2.Score = num.DecimalFromInt64(1)
	assertMarketContributionScore(t, LPMarket2, tracker.GetMarketScores("asset1", []string{"market2"}, vgproto.DispatchMetric_DISPATCH_METRIC_TAKER_FEES_PAID)[0])

	// try to scope market3: doesn't exist in the asset
	require.Equal(t, 0, len(tracker.GetMarketScores("asset1", []string{"market3"}, vgproto.DispatchMetric_DISPATCH_METRIC_TAKER_FEES_PAID)))

	// try to get the market from the wrong asset
	require.Equal(t, 0, len(tracker.GetMarketScores("asset2", []string{"market1"}, vgproto.DispatchMetric_DISPATCH_METRIC_TAKER_FEES_PAID)))
}

func TestGetMarketsWithEligibleProposer(t *testing.T) {
	tracker := execution.NewMarketActivityTracker(logging.NewTestLogger(), &TestEpochEngine{})
	tracker.SetEligibilityChecker(&EligibilityChecker{})
	tracker.MarketProposed("asset1", "market1", "me")
	tracker.MarketProposed("asset1", "market2", "me2")

	tracker.AddValueTraded("market2", num.NewUint(1001))
	tracker.AddValueTraded("market1", num.NewUint(4001))

	// the threshold is 5000 so expect at this point no market should be returned
	require.Equal(t, 0, len(tracker.GetMarketsWithEligibleProposer("asset1", []string{})))
	require.Equal(t, 0, len(tracker.GetMarketsWithEligibleProposer("asset1", []string{"market1"})))
	require.Equal(t, 0, len(tracker.GetMarketsWithEligibleProposer("asset1", []string{"market2"})))
	require.Equal(t, 0, len(tracker.GetMarketsWithEligibleProposer("asset1", []string{"market1", "market2"})))

	// market1 goes above the threshold
	tracker.AddValueTraded("market1", num.NewUint(1000))
	expectedScoreMarket1Full := &types.MarketContributionScore{
		Asset:  "asset1",
		Market: "market1",
		Score:  num.DecimalFromInt64(1),
		Metric: vgproto.DispatchMetric_DISPATCH_METRIC_MARKET_VALUE,
	}
	require.Equal(t, 1, len(tracker.GetMarketsWithEligibleProposer("asset1", []string{})))
	assertMarketContributionScore(t, expectedScoreMarket1Full, tracker.GetMarketsWithEligibleProposer("asset1", []string{})[0])
	require.Equal(t, 1, len(tracker.GetMarketsWithEligibleProposer("asset1", []string{"market1"})))
	assertMarketContributionScore(t, expectedScoreMarket1Full, tracker.GetMarketsWithEligibleProposer("asset1", []string{"market1"})[0])
	require.Equal(t, 0, len(tracker.GetMarketsWithEligibleProposer("asset1", []string{"market2"})))
	require.Equal(t, 1, len(tracker.GetMarketsWithEligibleProposer("asset1", []string{"market1", "market2"})))
	assertMarketContributionScore(t, expectedScoreMarket1Full, tracker.GetMarketsWithEligibleProposer("asset1", []string{"market1", "market2"})[0])

	// now market 2 goes above the threshold as well so expect the scores to be 0.5 for each
	tracker.AddValueTraded("market2", num.NewUint(4000))
	expectedScoreMarket1Half := &types.MarketContributionScore{
		Asset:  "asset1",
		Market: "market1",
		Score:  num.MustDecimalFromString("0.5"),
		Metric: vgproto.DispatchMetric_DISPATCH_METRIC_MARKET_VALUE,
	}
	expectedScoreMarket2Half := &types.MarketContributionScore{
		Asset:  "asset1",
		Market: "market2",
		Score:  num.MustDecimalFromString("0.5"),
		Metric: vgproto.DispatchMetric_DISPATCH_METRIC_MARKET_VALUE,
	}
	expectedScoreMarket2Full := &types.MarketContributionScore{
		Asset:  "asset1",
		Market: "market2",
		Score:  num.DecimalFromInt64(1),
		Metric: vgproto.DispatchMetric_DISPATCH_METRIC_MARKET_VALUE,
	}
	require.Equal(t, 2, len(tracker.GetMarketsWithEligibleProposer("asset1", []string{})))
	assertMarketContributionScore(t, expectedScoreMarket1Half, tracker.GetMarketsWithEligibleProposer("asset1", []string{})[0])
	assertMarketContributionScore(t, expectedScoreMarket2Half, tracker.GetMarketsWithEligibleProposer("asset1", []string{})[1])
	require.Equal(t, 1, len(tracker.GetMarketsWithEligibleProposer("asset1", []string{"market1"})))
	assertMarketContributionScore(t, expectedScoreMarket1Full, tracker.GetMarketsWithEligibleProposer("asset1", []string{"market1"})[0])
	require.Equal(t, 1, len(tracker.GetMarketsWithEligibleProposer("asset1", []string{"market2"})))
	assertMarketContributionScore(t, expectedScoreMarket2Full, tracker.GetMarketsWithEligibleProposer("asset1", []string{"market2"})[0])
	require.Equal(t, 2, len(tracker.GetMarketsWithEligibleProposer("asset1", []string{"market1", "market2"})))
	assertMarketContributionScore(t, expectedScoreMarket1Half, tracker.GetMarketsWithEligibleProposer("asset1", []string{"market1", "market2"})[0])
	assertMarketContributionScore(t, expectedScoreMarket2Half, tracker.GetMarketsWithEligibleProposer("asset1", []string{"market1", "market2"})[1])

	// asset with no markets
	require.Equal(t, 0, len(tracker.GetMarketsWithEligibleProposer("asset2", []string{})))
}

func assertMarketContributionScore(t *testing.T, expected, actual *types.MarketContributionScore) {
	t.Helper()
	require.Equal(t, expected.Asset, actual.Asset)
	require.Equal(t, expected.Market, actual.Market)
	require.Equal(t, expected.Score.String(), actual.Score.String())
	require.Equal(t, expected.Metric, actual.Metric)
}

func TestMarketTrackerStateChange(t *testing.T) {
	key := (&types.PayloadMarketActivityTracker{}).Key()

	tracker := execution.NewMarketActivityTracker(logging.NewTestLogger(), &TestEpochEngine{})
	tracker.SetEligibilityChecker(&EligibilityChecker{})

	state1, _, err := tracker.GetState(key)
	require.NoError(t, err)

	tracker.MarketProposed("asset1", "market1", "me")
	tracker.MarketProposed("asset1", "market2", "me2")

	state2, _, err := tracker.GetState(key)
	require.NoError(t, err)
	require.False(t, bytes.Equal(state1, state2))

	tracker.AddValueTraded("market1", num.NewUint(1000))
	require.Equal(t, 0, len(tracker.GetEligibleProposers("market1")))
	require.Equal(t, 0, len(tracker.GetEligibleProposers("market2")))

	state3, _, err := tracker.GetState(key)
	require.NoError(t, err)
	require.False(t, bytes.Equal(state1, state3))
}

func TestFeesTracker(t *testing.T) {
	epochEngine := &TestEpochEngine{}
	tracker := execution.NewMarketActivityTracker(logging.NewTestLogger(), epochEngine)
	epochEngine.target(context.Background(), types.Epoch{Seq: 1})

	partyScores := tracker.GetFeePartyScores("does not exist", types.TransferTypeMakerFeeReceive)
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
	tracker.UpdateFeesFromTransfers("market1", transfersM1)

	transfersM2 := []*types.Transfer{
		{Owner: "party1", Type: types.TransferTypeMakerFeeReceive, Amount: &types.FinancialAmount{Asset: "asset2", Amount: num.NewUint(150)}},
		{Owner: "party2", Type: types.TransferTypeMakerFeePay, Amount: &types.FinancialAmount{Asset: "asset2", Amount: num.NewUint(150)}},
	}
	tracker.UpdateFeesFromTransfers("market2", transfersM2)

	// asset1, types.TransferTypeMakerFeeReceive
	// party1 received 500
	// party2 received 1500
	scores := tracker.GetFeePartyScores("market1", types.TransferTypeMakerFeeReceive)
	require.Equal(t, "0.25", scores[0].Score.String())
	require.Equal(t, "party1", scores[0].Party)
	require.Equal(t, "0.75", scores[1].Score.String())
	require.Equal(t, "party2", scores[1].Party)

	// asset1 TransferTypeMakerFeePay
	// party1 paid 500
	// party2 paid 1000
	scores = tracker.GetFeePartyScores("market1", types.TransferTypeMakerFeePay)
	require.Equal(t, "0.3333333333333333", scores[0].Score.String())
	require.Equal(t, "party1", scores[0].Party)
	require.Equal(t, "0.6666666666666667", scores[1].Score.String())
	require.Equal(t, "party2", scores[1].Party)

	// asset1 TransferTypeLiquidityFeeDistribute
	// party1 paid 800
	// party2 paid 1700
	scores = tracker.GetFeePartyScores("market1", types.TransferTypeLiquidityFeeDistribute)
	require.Equal(t, "0.32", scores[0].Score.String())
	require.Equal(t, "party1", scores[0].Party)
	require.Equal(t, "0.68", scores[1].Score.String())
	require.Equal(t, "party2", scores[1].Party)

	// asset2 TransferTypeMakerFeePay
	scores = tracker.GetFeePartyScores("market2", types.TransferTypeMakerFeeReceive)
	require.Equal(t, 1, len(scores))
	require.Equal(t, "1", scores[0].Score.String())
	require.Equal(t, "party1", scores[0].Party)

	// asset2 TransferTypeMakerFeePay
	scores = tracker.GetFeePartyScores("market2", types.TransferTypeMakerFeePay)
	require.Equal(t, 1, len(scores))
	require.Equal(t, "1", scores[0].Score.String())
	require.Equal(t, "party2", scores[0].Party)

	// check state has changed
	state2, _, err := tracker.GetState(key)
	require.NoError(t, err)
	require.False(t, bytes.Equal(state1, state2))

	epochEngineLoad := &TestEpochEngine{}
	trackerLoad := execution.NewMarketActivityTracker(logging.NewTestLogger(), epochEngineLoad)
	epochEngineLoad.target(context.Background(), types.Epoch{Seq: 1})

	pl := snapshotpb.Payload{}
	require.NoError(t, proto.Unmarshal(state2, &pl))
	trackerLoad.LoadState(context.Background(), types.PayloadFromProto(&pl))

	state3, _, err := trackerLoad.GetState(key)
	require.NoError(t, err)
	require.True(t, bytes.Equal(state2, state3))

	// check a restored party exist in the restored engine
	scores = trackerLoad.GetFeePartyScores("market2", types.TransferTypeMakerFeeReceive)
	require.Equal(t, 1, len(scores))
	require.Equal(t, "1", scores[0].Score.String())
	require.Equal(t, "party1", scores[0].Party)

	// New epoch should scrub the state an produce a difference hash
	epochEngineLoad.target(context.Background(), types.Epoch{Seq: 2, Action: vgproto.EpochAction_EPOCH_ACTION_START})
	state4, _, err := trackerLoad.GetState(key)
	require.NoError(t, err)
	require.False(t, bytes.Equal(state3, state4))

	// new epoch, we expect the metrics to have been reset
	for _, metric := range []types.TransferType{types.TransferTypeMakerFeePay, types.TransferTypeMakerFeeReceive, types.TransferTypeLiquidityFeeDistribute} {
		require.Equal(t, 0, len(trackerLoad.GetFeePartyScores("market1", metric)))
		require.Equal(t, 0, len(trackerLoad.GetFeePartyScores("market2", metric)))
	}
}

func TestSnapshot(t *testing.T) {
	tracker := setupDefaultTrackerForTest(t)

	// take a snapshot
	key := (&types.PayloadMarketActivityTracker{}).Key()
	state1, _, err := tracker.GetState(key)
	require.NoError(t, err)

	trackerLoad := execution.NewMarketActivityTracker(logging.NewTestLogger(), &TestEpochEngine{})
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

	trackerLoad := execution.NewMarketActivityTracker(logging.NewTestLogger(), &TestEpochEngine{})
	trackerLoad.Load(context.Background(), b)

	bLoad, err := trackerLoad.Checkpoint()
	require.NoError(t, err)
	require.True(t, bytes.Equal(b, bLoad))
}

func setupDefaultTrackerForTest(t *testing.T) *execution.MarketActivityTracker {
	t.Helper()
	tracker := execution.NewMarketActivityTracker(logging.NewTestLogger(), &TestEpochEngine{})
	tracker.SetEligibilityChecker(&EligibilityChecker{})

	tracker.MarketProposed("asset1", "market1", "me")
	tracker.MarketProposed("asset1", "market2", "me2")
	tracker.MarketProposed("asset1", "market4", "me4")
	tracker.MarketProposed("asset2", "market3", "me3")

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
	tracker.UpdateFeesFromTransfers("market1", transfersM1)

	transfersM2 := []*types.Transfer{
		{Owner: "party1", Type: types.TransferTypeMakerFeeReceive, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.NewUint(500)}},
		{Owner: "party2", Type: types.TransferTypeMakerFeePay, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.NewUint(1500)}},
		{Owner: "party2", Type: types.TransferTypeLiquidityFeeDistribute, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.NewUint(1500)}},
	}
	tracker.UpdateFeesFromTransfers("market2", transfersM2)

	transfersM3 := []*types.Transfer{
		{Owner: "party1", Type: types.TransferTypeMakerFeePay, Amount: &types.FinancialAmount{Asset: "asset2", Amount: num.NewUint(500)}},
		{Owner: "party2", Type: types.TransferTypeMakerFeeReceive, Amount: &types.FinancialAmount{Asset: "asset2", Amount: num.NewUint(450)}},
	}
	tracker.UpdateFeesFromTransfers("market3", transfersM3)
	return tracker
}

func TestSnapshotRoundtripViaEngine(t *testing.T) {
	ctx := vgcontext.WithTraceID(vgcontext.WithBlockHeight(context.Background(), 100), "0xDEADBEEF")
	ctx = vgcontext.WithChainID(ctx, "chainid")
	tracker := setupDefaultTrackerForTest(t)
	now := time.Now()
	log := logging.NewTestLogger()
	timeService := stubs.NewTimeStub()
	timeService.SetTime(now)
	statsData := stats.New(log, stats.NewDefaultConfig(), "", "")
	config := snp.NewDefaultConfig()
	config.Storage = "memory"
	snapshotEngine, _ := snp.New(context.Background(), &paths.DefaultPaths{}, config, log, timeService, statsData.Blockchain)
	snapshotEngine.AddProviders(tracker)
	snapshotEngine.ClearAndInitialise()
	defer snapshotEngine.Close()

	_, err := snapshotEngine.Snapshot(ctx)
	require.NoError(t, err)
	snaps, err := snapshotEngine.List()
	require.NoError(t, err)
	snap1 := snaps[0]

	trackerLoad := execution.NewMarketActivityTracker(logging.NewTestLogger(), &TestEpochEngine{})
	tracker.SetEligibilityChecker(&EligibilityChecker{})
	snapshotEngineLoad, _ := snp.New(context.Background(), &paths.DefaultPaths{}, config, log, timeService, statsData.Blockchain)
	snapshotEngineLoad.AddProviders(trackerLoad)
	snapshotEngineLoad.ClearAndInitialise()
	snapshotEngineLoad.ReceiveSnapshot(snap1)
	snapshotEngineLoad.ApplySnapshot(ctx)
	snapshotEngineLoad.CheckLoaded()
	defer snapshotEngineLoad.Close()

	b, err := snapshotEngine.Snapshot(ctx)
	require.NoError(t, err)
	bLoad, err := snapshotEngineLoad.Snapshot(ctx)
	require.NoError(t, err)
	require.True(t, bytes.Equal(b, bLoad))

	// now lets get some activity going and verify they still match
	tracker.MarketProposed("asset1", "market5", "meeeee")
	tracker.MarketProposed("asset2", "market6", "meeeeeee")
	trackerLoad.MarketProposed("asset1", "market5", "meeeee")
	trackerLoad.MarketProposed("asset2", "market6", "meeeeeee")

	transfersM5 := []*types.Transfer{
		{Owner: "party3", Type: types.TransferTypeMakerFeeReceive, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.NewUint(100)}},
		{Owner: "party3", Type: types.TransferTypeMakerFeePay, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.NewUint(200)}},
		{Owner: "party3", Type: types.TransferTypeLiquidityFeeDistribute, Amount: &types.FinancialAmount{Asset: "asset1", Amount: num.NewUint(200)}},
	}
	tracker.UpdateFeesFromTransfers("market5", transfersM5)
	trackerLoad.UpdateFeesFromTransfers("market5", transfersM5)

	transfersM6 := []*types.Transfer{
		{Owner: "party4", Type: types.TransferTypeMakerFeeReceive, Amount: &types.FinancialAmount{Asset: "asset2", Amount: num.NewUint(500)}},
		{Owner: "party4", Type: types.TransferTypeMakerFeePay, Amount: &types.FinancialAmount{Asset: "asset2", Amount: num.NewUint(1500)}},
		{Owner: "party4", Type: types.TransferTypeLiquidityFeeDistribute, Amount: &types.FinancialAmount{Asset: "asset2", Amount: num.NewUint(1500)}},
	}
	tracker.UpdateFeesFromTransfers("market6", transfersM6)
	trackerLoad.UpdateFeesFromTransfers("market6", transfersM6)

	b, err = snapshotEngine.Snapshot(ctx)
	require.NoError(t, err)
	bLoad, err = snapshotEngineLoad.Snapshot(ctx)
	require.NoError(t, err)
	require.True(t, bytes.Equal(b, bLoad))
}
