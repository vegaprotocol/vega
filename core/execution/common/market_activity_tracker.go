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

package common

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/protos/vega"
	proto "code.vegaprotocol.io/vega/protos/vega"
)

// marketTracker tracks the activity in the markets in terms of fees and value.
type marketTracker struct {
	asset                  string
	makerFeesReceived      map[string]*num.Uint
	makerFeesPaid          map[string]*num.Uint
	lpFees                 map[string]*num.Uint
	totalMakerFeesReceived *num.Uint
	totalMakerFeesPaid     *num.Uint
	totalLPFees            *num.Uint
	valueTraded            *num.Uint
	proposersPaid          map[string]struct{} // identifier of payout_asset : funder : markets_in_scope
	proposer               string
	readyToDelete          bool
}

// MarketActivityTracker tracks how much fees are paid and received for a market by parties by epoch.
type MarketActivityTracker struct {
	log                *logging.Logger
	marketToTracker    map[string]*marketTracker
	eligibilityChecker EligibilityChecker
	currentEpoch       uint64
	ss                 *snapshotState
}

// NewFeesTracker instantiates the fees tracker.
func NewMarketActivityTracker(log *logging.Logger, epochEngine EpochEngine) *MarketActivityTracker {
	mat := &MarketActivityTracker{
		marketToTracker: map[string]*marketTracker{},
		ss:              &snapshotState{},
		log:             log,
	}
	epochEngine.NotifyOnEpoch(mat.onEpochEvent, mat.onEpochRestore)
	return mat
}

// GetProposer returns the proposer of the market or empty string if the market doesn't exist.
func (mat *MarketActivityTracker) GetProposer(market string) string {
	m, ok := mat.marketToTracker[market]
	if ok {
		return m.proposer
	}
	return ""
}

func (mat *MarketActivityTracker) SetEligibilityChecker(eligibilityChecker EligibilityChecker) {
	mat.eligibilityChecker = eligibilityChecker
}

// MarketProposed is called when the market is proposed and adds the market to the tracker.
func (mat *MarketActivityTracker) MarketProposed(asset, marketID, proposer string) {
	// if we already know about this market don't re-add it
	if _, ok := mat.marketToTracker[marketID]; ok {
		return
	}
	mat.marketToTracker[marketID] = &marketTracker{
		asset:                  asset,
		proposer:               proposer,
		proposersPaid:          map[string]struct{}{},
		readyToDelete:          false,
		valueTraded:            num.UintZero(),
		makerFeesReceived:      map[string]*num.Uint{},
		makerFeesPaid:          map[string]*num.Uint{},
		lpFees:                 map[string]*num.Uint{},
		totalMakerFeesReceived: num.UintZero(),
		totalMakerFeesPaid:     num.UintZero(),
		totalLPFees:            num.UintZero(),
	}
}

// AddValueTraded records the value of a trade done in the given market.
func (mat *MarketActivityTracker) AddValueTraded(marketID string, value *num.Uint) {
	if _, ok := mat.marketToTracker[marketID]; !ok {
		return
	}
	mat.marketToTracker[marketID].valueTraded.AddSum(value)
}

// GetMarketsWithEligibleProposer gets all the markets within the given asset (or just all the markets in scope passed as a parameter) that
// are eligible for proposer bonus.
func (mat *MarketActivityTracker) GetMarketsWithEligibleProposer(asset string, markets []string, payoutAsset string, funder string) []*types.MarketContributionScore {
	var mkts []string
	if len(markets) > 0 {
		mkts = markets
	} else {
		for m := range mat.marketToTracker {
			mkts = append(mkts, m)
		}
	}

	sort.Strings(mkts)

	eligibleMarkets := []string{}
	for _, v := range mkts {
		if t, ok := mat.marketToTracker[v]; ok && (len(asset) == 0 || t.asset == asset) && mat.IsMarketEligibleForBonus(v, payoutAsset, markets, funder) {
			eligibleMarkets = append(eligibleMarkets, v)
		}
	}
	if len(eligibleMarkets) <= 0 {
		return nil
	}
	scores := make([]*types.MarketContributionScore, 0, len(eligibleMarkets))
	numMarkets := num.DecimalFromInt64(int64(len(eligibleMarkets)))
	totalScore := num.DecimalZero()
	for _, v := range eligibleMarkets {
		score := num.DecimalFromInt64(1).Div(numMarkets)
		scores = append(scores, &types.MarketContributionScore{
			Asset:  asset,
			Market: v,
			Score:  score,
			Metric: proto.DispatchMetric_DISPATCH_METRIC_MARKET_VALUE,
		})
		totalScore = totalScore.Add(score)
	}

	mat.clipScoresAt1(scores, totalScore)
	scoresString := ""

	for _, mcs := range scores {
		scoresString += mcs.Market + ":" + mcs.Score.String() + ","
	}
	mat.log.Info("markets contibutions:", logging.String("asset", asset), logging.String("metric", proto.DispatchMetric_name[int32(proto.DispatchMetric_DISPATCH_METRIC_MARKET_VALUE)]), logging.String("market-scores", scoresString[:len(scoresString)-1]))

	return scores
}

func (mat *MarketActivityTracker) clipScoresAt1(scores []*types.MarketContributionScore, totalScore num.Decimal) {
	if totalScore.LessThanOrEqual(num.DecimalFromInt64(1)) {
		return
	}
	// if somehow the total scores are > 1 clip the largest one
	sort.SliceStable(scores, func(i, j int) bool { return scores[i].Score.GreaterThan(scores[j].Score) })
	delta := totalScore.Sub(num.DecimalFromInt64(1))
	scores[0].Score = num.MaxD(num.DecimalZero(), scores[0].Score.Sub(delta))
	// sort by market id for consistency
	sort.SliceStable(scores, func(i, j int) bool { return scores[i].Market < scores[j].Market })
}

// MarkProposerPaid marks the proposer of the market as having been paid proposer bonus.
func (mat *MarketActivityTracker) MarkPaidProposer(market, payoutAsset string, marketsInScope []string, funder string) {
	markets := strings.Join(marketsInScope[:], "_")
	if len(marketsInScope) == 0 {
		markets = "all"
	}

	if t, ok := mat.marketToTracker[market]; ok {
		ID := fmt.Sprintf("%s:%s:%s", payoutAsset, funder, markets)
		if _, ok := t.proposersPaid[ID]; !ok {
			t.proposersPaid[ID] = struct{}{}
		}
	}
}

// IsMarketEligibleForBonus returns true is the market proposer is eligible for market proposer bonus and has not been
// paid for the combination of payout asset and marketsInScope.
// The proposer is not market as having been paid until told to do so (if actually paid).
func (mat *MarketActivityTracker) IsMarketEligibleForBonus(market string, payoutAsset string, marketsInScope []string, funder string) bool {
	t, ok := mat.marketToTracker[market]
	if !ok {
		return false
	}

	markets := strings.Join(marketsInScope[:], "_")
	if len(marketsInScope) == 0 {
		markets = "all"
	}

	marketIsInScope := false
	for _, v := range marketsInScope {
		if v == market {
			marketIsInScope = true
			break
		}
	}

	if len(marketsInScope) == 0 {
		markets = "all"
		marketIsInScope = true
	}

	if !marketIsInScope {
		return false
	}

	ID := fmt.Sprintf("%s:%s:%s", payoutAsset, funder, markets)
	_, paid := t.proposersPaid[ID]

	return !paid && mat.eligibilityChecker.IsEligibleForProposerBonus(market, t.valueTraded)
}

// GetAllMarketIDs returns all the current market IDs.
func (mat *MarketActivityTracker) GetAllMarketIDs() []string {
	mIDs := make([]string, 0, len(mat.marketToTracker))
	for k := range mat.marketToTracker {
		mIDs = append(mIDs, k)
	}

	sort.Strings(mIDs)
	return mIDs
}

// removeMarket is called when the market is removed from the network. It is not immediately removed to give a chance for rewards to be paid at the end of the epoch for activity during the epoch.
// Instead it is marked for removal and will be removed at the beginning of the next epoch.
func (mat *MarketActivityTracker) RemoveMarket(marketID string) {
	if m, ok := mat.marketToTracker[marketID]; ok {
		m.readyToDelete = true
	}
}

// onEpochEvent is called when the state of the epoch changes, we only care about new epochs starting.
func (mat *MarketActivityTracker) onEpochEvent(_ context.Context, epoch types.Epoch) {
	if epoch.Action == proto.EpochAction_EPOCH_ACTION_START {
		mat.clearFeeActivity()
	}
	mat.currentEpoch = epoch.Seq
}

// clearFeeActivity is called at the beginning of a new epoch. It deletes markets that are pending to be removed and resets the fees paid for the epoch.
func (mat *MarketActivityTracker) clearFeeActivity() {
	for k, mt := range mat.marketToTracker {
		if mt.readyToDelete {
			delete(mat.marketToTracker, k)
			continue
		}
		mt.lpFees = map[string]*num.Uint{}
		mt.totalLPFees = num.UintZero()
		mt.makerFeesReceived = map[string]*num.Uint{}
		mt.totalMakerFeesReceived = num.UintZero()
		mt.makerFeesPaid = map[string]*num.Uint{}
		mt.totalMakerFeesPaid = num.UintZero()
	}
}

// GetMarketScores calculates the aggregate share of the asset/market in contribution to the metric out of either all the markets of the asset or the subset specified.
func (mat *MarketActivityTracker) GetMarketScores(asset string, markets []string, dispatchMetric vega.DispatchMetric) []*types.MarketContributionScore {
	totalFees := num.UintZero()

	// consider only markets in scope, if passed then only use those, if not passed use all the asset's markets which contributed to the metric
	marketsInScope := markets
	if len(marketsInScope) <= 0 {
		for m := range mat.marketToTracker {
			marketsInScope = append(marketsInScope, m)
		}
	}
	sort.Strings(marketsInScope)

	for _, marketInScope := range marketsInScope {
		if mt, ok := mat.marketToTracker[marketInScope]; ok && mt.asset == asset {
			totalFees.AddSum(mt.totalFees(dispatchMetric))
		}
	}
	totalFeesD := totalFees.ToDecimal()

	// calculation the contribution each market in scope made to the total metric
	scores := []*types.MarketContributionScore{}

	// if there are no fees, no need to bother.
	if totalFees.IsZero() {
		mat.log.Info("markets contibutions:", logging.String("asset", asset), logging.String("metric", proto.DispatchMetric_name[int32(dispatchMetric)]), logging.String("market-scores", "none"))
		return scores
	}

	totalScore := num.DecimalZero()
	for _, marketInScope := range marketsInScope {
		if mt, ok := mat.marketToTracker[marketInScope]; ok && asset == mt.asset {
			score := mt.totalFees(dispatchMetric).ToDecimal().Div(totalFeesD)
			if score.IsZero() {
				continue
			}
			scores = append(scores, &types.MarketContributionScore{
				Asset:  asset,
				Market: marketInScope,
				Score:  score,
				Metric: dispatchMetric,
			})
			totalScore = totalScore.Add(score)
		}
	}

	mat.clipScoresAt1(scores, totalScore)

	scoresString := ""

	for _, mcs := range scores {
		scoresString += mcs.Market + ":" + mcs.Score.String() + ","
	}
	mat.log.Info("markets contibutions:", logging.String("asset", asset), logging.String("metric", proto.DispatchMetric_name[int32(dispatchMetric)]), logging.String("market-scores", scoresString[:len(scoresString)-1]))

	return scores
}

// GetFeePartyScores returns the fraction each of the participants paid/received in the given fee of the market in the relevant period.
func (mat *MarketActivityTracker) GetFeePartyScores(market string, feeType types.TransferType) []*types.PartyContibutionScore {
	if _, ok := mat.marketToTracker[market]; !ok {
		return []*types.PartyContibutionScore{}
	}

	feesData := map[string]*num.Uint{}

	switch feeType {
	case types.TransferTypeMakerFeeReceive:
		feesData = mat.marketToTracker[market].makerFeesReceived
	case types.TransferTypeMakerFeePay:
		feesData = mat.marketToTracker[market].makerFeesPaid
	case types.TransferTypeLiquidityFeeDistribute:
		feesData = mat.marketToTracker[market].lpFees
	default:
	}

	scores := make([]*types.PartyContibutionScore, 0, len(feesData))
	parties := make([]string, 0, len(scores))
	for party := range feesData {
		parties = append(parties, party)
	}
	sort.Strings(parties)

	total := num.DecimalZero()
	for _, party := range parties {
		total = total.Add(feesData[party].ToDecimal())
	}
	for _, party := range parties {
		scores = append(scores, &types.PartyContibutionScore{Party: party, Score: feesData[party].ToDecimal().Div(total)})
	}
	return scores
}

// UpdateFeesFromTransfers takes a slice of transfers and if they represent fees it updates the market fee tracker.
// market is guaranteed to exist in the mapping as it is added when proposed.
func (mat *MarketActivityTracker) UpdateFeesFromTransfers(market string, transfers []*types.Transfer) {
	for _, t := range transfers {
		mt := mat.marketToTracker[market]
		if mt == nil {
			continue
		}
		switch t.Type {
		case types.TransferTypeMakerFeePay:
			mat.addFees(mt.makerFeesPaid, t.Owner, t.Amount.Amount, mt.totalMakerFeesPaid)
		case types.TransferTypeMakerFeeReceive:
			mat.addFees(mt.makerFeesReceived, t.Owner, t.Amount.Amount, mt.totalMakerFeesReceived)
		case types.TransferTypeLiquidityFeeDistribute:
			mat.addFees(mt.lpFees, t.Owner, t.Amount.Amount, mt.totalLPFees)
		default:
		}
	}
}

// addFees records fees paid/received in a given metric to a given party.
func (mat *MarketActivityTracker) addFees(m map[string]*num.Uint, party string, amount, total *num.Uint) {
	total.AddSum(amount)
	if _, ok := m[party]; !ok {
		m[party] = amount.Clone()
		return
	}
	m[party] = num.Sum(m[party], amount)
}

// totalFees returns the total fees corresponding to the fee metric.
func (mt *marketTracker) totalFees(metric vega.DispatchMetric) *num.Uint {
	switch metric {
	case vega.DispatchMetric_DISPATCH_METRIC_MAKER_FEES_RECEIVED:
		return mt.totalMakerFeesReceived
	case vega.DispatchMetric_DISPATCH_METRIC_MAKER_FEES_PAID:
		return mt.totalMakerFeesPaid
	case vega.DispatchMetric_DISPATCH_METRIC_LP_FEES_RECEIVED:
		return mt.totalLPFees
	default:
		return num.UintZero()
	}
}
