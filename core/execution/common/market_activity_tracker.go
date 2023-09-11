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
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"time"

	"code.vegaprotocol.io/vega/libs/crypto"

	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	lproto "code.vegaprotocol.io/vega/libs/proto"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/protos/vega"
	proto "code.vegaprotocol.io/vega/protos/vega"
)

// this the maximum supported window size for any metric.
const maxWindowSize = 100

type twPosition struct {
	position               num.Decimal   // abs last recorded position
	t                      time.Time     // time of last recorded open interest
	currentEpochTWPosition num.Decimal   // current epoch's running time weightd open interest
	previousEpochs         []num.Decimal // previous epochs' time weighted open interest
	previousEpochsIdx      int           // index to the previousEpochs array
}

type twNotionalPosition struct {
	position               num.Decimal // abs last recorded position
	price                  *num.Uint   // last position's price
	t                      time.Time   // time of last recorded open interest
	currentEpochTWNotional num.Decimal // current epoch's running time weightd notional
}

type m2mData struct {
	runningTotal      num.Decimal
	previousEpochs    []num.Decimal
	previousEpochsIdx int
}

type feeData struct {
	runningTotal      *num.Uint
	previousEpochs    []*num.Uint
	previousEpochsIdx int
}

// marketTracker tracks the activity in the markets in terms of fees and value.
type marketTracker struct {
	asset                string
	makerFeesReceived    map[string]*feeData
	makerFeesPaid        map[string]*feeData
	lpFees               map[string]*feeData
	timeWeightedPosition map[string]*twPosition
	partyM2M             map[string]*m2mData

	totalMakerFeesReceived *feeData
	totalMakerFeesPaid     *feeData
	totalLpFees            *feeData

	twNotionalPosition map[string]*twNotionalPosition

	valueTraded   *num.Uint
	proposersPaid map[string]struct{} // identifier of payout_asset : funder : markets_in_scope
	proposer      string
	readyToDelete bool
}

// MarketActivityTracker tracks how much fees are paid and received for a market by parties by epoch.
type MarketActivityTracker struct {
	log                                 *logging.Logger
	assetToMarketTrackers               map[string]map[string]*marketTracker
	eligibilityChecker                  EligibilityChecker
	currentEpoch                        uint64
	epochStartTime                      time.Time
	ss                                  *snapshotState
	teams                               Teams
	balanceChecker                      AccountBalanceChecker
	minEpochsInTeamForRewardEligibility uint64
	partyContributionCache              map[string][]*types.PartyContibutionScore
	partyTakerNotionalVolume            map[string]*num.Uint
}

// NewMarketActivityTracker instantiates the fees tracker.
func NewMarketActivityTracker(log *logging.Logger, epochEngine EpochEngine, teams Teams, balanceChecker AccountBalanceChecker) *MarketActivityTracker {
	mat := &MarketActivityTracker{
		assetToMarketTrackers:    map[string]map[string]*marketTracker{},
		ss:                       &snapshotState{},
		log:                      log,
		balanceChecker:           balanceChecker,
		teams:                    teams,
		partyContributionCache:   map[string][]*types.PartyContibutionScore{},
		partyTakerNotionalVolume: map[string]*num.Uint{},
	}

	epochEngine.NotifyOnEpoch(mat.onEpochEvent, mat.onEpochRestore)
	return mat
}

func (mat *MarketActivityTracker) OnMinEpochsInTeamForRewardEligibilityUpdated(_ context.Context, value int64) error {
	mat.minEpochsInTeamForRewardEligibility = uint64(value)
	return nil
}

// GetProposer returns the proposer of the market or empty string if the market doesn't exist.
func (mat *MarketActivityTracker) GetProposer(market string) string {
	for _, markets := range mat.assetToMarketTrackers {
		m, ok := markets[market]
		if ok {
			return m.proposer
		}
	}
	return ""
}

func (mat *MarketActivityTracker) SetEligibilityChecker(eligibilityChecker EligibilityChecker) {
	mat.eligibilityChecker = eligibilityChecker
}

// MarketProposed is called when the market is proposed and adds the market to the tracker.
func (mat *MarketActivityTracker) MarketProposed(asset, marketID, proposer string) {
	markets, ok := mat.assetToMarketTrackers[asset]
	if ok {
		if _, ok := markets[marketID]; ok {
			return
		}
	}

	tracker := &marketTracker{
		asset:                  asset,
		proposer:               proposer,
		proposersPaid:          map[string]struct{}{},
		readyToDelete:          false,
		valueTraded:            num.UintZero(),
		makerFeesReceived:      map[string]*feeData{},
		makerFeesPaid:          map[string]*feeData{},
		lpFees:                 map[string]*feeData{},
		totalMakerFeesReceived: &feeData{runningTotal: num.UintZero(), previousEpochs: make([]*num.Uint, maxWindowSize), previousEpochsIdx: 0},
		totalMakerFeesPaid:     &feeData{runningTotal: num.UintZero(), previousEpochs: make([]*num.Uint, maxWindowSize), previousEpochsIdx: 0},
		totalLpFees:            &feeData{runningTotal: num.UintZero(), previousEpochs: make([]*num.Uint, maxWindowSize), previousEpochsIdx: 0},
		timeWeightedPosition:   map[string]*twPosition{},
		partyM2M:               map[string]*m2mData{},
		twNotionalPosition:     map[string]*twNotionalPosition{},
	}

	if ok {
		markets[marketID] = tracker
	} else {
		mat.assetToMarketTrackers[asset] = map[string]*marketTracker{marketID: tracker}
	}
}

// AddValueTraded records the value of a trade done in the given market.
func (mat *MarketActivityTracker) AddValueTraded(asset, marketID string, value *num.Uint) {
	markets, ok := mat.assetToMarketTrackers[asset]
	if !ok || markets[marketID] == nil {
		return
	}
	markets[marketID].valueTraded.AddSum(value)
}

// GetMarketsWithEligibleProposer gets all the markets within the given asset (or just all the markets in scope passed as a parameter) that
// are eligible for proposer bonus.
func (mat *MarketActivityTracker) GetMarketsWithEligibleProposer(asset string, markets []string, payoutAsset string, funder string) []*types.MarketContributionScore {
	var mkts []string
	if len(markets) > 0 {
		mkts = markets
	} else {
		if len(asset) > 0 {
			for m := range mat.assetToMarketTrackers[asset] {
				mkts = append(mkts, m)
			}
		} else {
			for _, markets := range mat.assetToMarketTrackers {
				for mkt := range markets {
					mkts = append(mkts, mkt)
				}
			}
		}
	}

	sort.Strings(mkts)

	assets := []string{}
	if len(asset) > 0 {
		assets = append(assets, asset)
	} else {
		for k := range mat.assetToMarketTrackers {
			assets = append(assets, k)
		}
		sort.Strings(assets)
	}

	eligibleMarkets := []string{}
	for _, a := range assets {
		for _, v := range mkts {
			if t, ok := mat.getMarketTracker(a, v); ok && (len(asset) == 0 || t.asset == asset) && mat.IsMarketEligibleForBonus(a, v, payoutAsset, markets, funder) {
				eligibleMarkets = append(eligibleMarkets, v)
			}
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
func (mat *MarketActivityTracker) MarkPaidProposer(asset, market, payoutAsset string, marketsInScope []string, funder string) {
	markets := strings.Join(marketsInScope[:], "_")
	if len(marketsInScope) == 0 {
		markets = "all"
	}

	if mts, ok := mat.assetToMarketTrackers[asset]; ok {
		t, ok := mts[market]
		if !ok {
			return
		}
		ID := fmt.Sprintf("%s:%s:%s", payoutAsset, funder, markets)
		if _, ok := t.proposersPaid[ID]; !ok {
			t.proposersPaid[ID] = struct{}{}
		}
	}
}

// IsMarketEligibleForBonus returns true is the market proposer is eligible for market proposer bonus and has not been
// paid for the combination of payout asset and marketsInScope.
// The proposer is not market as having been paid until told to do so (if actually paid).
func (mat *MarketActivityTracker) IsMarketEligibleForBonus(asset, market, payoutAsset string, marketsInScope []string, funder string) bool {
	t, ok := mat.getMarketTracker(asset, market)
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
	mIDs := []string{}
	for _, markets := range mat.assetToMarketTrackers {
		for k := range markets {
			mIDs = append(mIDs, k)
		}
	}

	sort.Strings(mIDs)
	return mIDs
}

// removeMarket is called when the market is removed from the network. It is not immediately removed to give a chance for rewards to be paid at the end of the epoch for activity during the epoch.
// Instead it is marked for removal and will be removed at the beginning of the next epoch.
func (mat *MarketActivityTracker) RemoveMarket(asset, marketID string) {
	if markets, ok := mat.assetToMarketTrackers[asset]; ok {
		if m, ok := markets[marketID]; ok {
			m.readyToDelete = true
		}
	}
}

// onEpochEvent is called when the state of the epoch changes, we only care about new epochs starting.
func (mat *MarketActivityTracker) onEpochEvent(_ context.Context, epoch types.Epoch) {
	if epoch.Action == proto.EpochAction_EPOCH_ACTION_START {
		mat.epochStartTime = epoch.StartTime
		mat.partyContributionCache = map[string][]*types.PartyContibutionScore{}
		mat.clearDeletedMarkets()
		mat.clearNotionalTakerVolume()
	}
	if epoch.Action == proto.EpochAction_EPOCH_ACTION_END {
		for _, market := range mat.assetToMarketTrackers {
			for _, mt := range market {
				mt.processNotionalEndOfEpoch(epoch.StartTime, epoch.EndTime)
				mt.processPositionEndOfEpoch(epoch.StartTime, epoch.EndTime)
				mt.processM2MEndOfEpoch()
				mt.clearFeeActivity()
			}
		}
	}
	mat.currentEpoch = epoch.Seq
}

func (mat *MarketActivityTracker) clearDeletedMarkets() {
	for _, mts := range mat.assetToMarketTrackers {
		for k, mt := range mts {
			if mt.readyToDelete {
				delete(mts, k)
			}
		}
	}
}

// clearFeeActivity is called at the end of the epoch. It deletes markets that are pending to be removed and resets the fees paid for the epoch.
func (mt *marketTracker) clearFeeActivity() {
	fees := []map[string]*feeData{mt.lpFees, mt.makerFeesPaid, mt.makerFeesReceived}
	for _, fee := range fees {
		for _, fd := range fee {
			fd.previousEpochs[fd.previousEpochsIdx] = fd.runningTotal
			fd.previousEpochsIdx = (fd.previousEpochsIdx + 1) % maxWindowSize
			fd.runningTotal = num.UintZero()
		}
	}
	totalFees := []*feeData{mt.totalLpFees, mt.totalMakerFeesPaid, mt.totalMakerFeesReceived}
	for _, fd := range totalFees {
		fd.previousEpochs[fd.previousEpochsIdx] = fd.runningTotal
		fd.previousEpochsIdx = (fd.previousEpochsIdx + 1) % maxWindowSize
		fd.runningTotal = num.UintZero()
	}
}

// UpdateFeesFromTransfers takes a slice of transfers and if they represent fees it updates the market fee tracker.
// market is guaranteed to exist in the mapping as it is added when proposed.
func (mat *MarketActivityTracker) UpdateFeesFromTransfers(asset, market string, transfers []*types.Transfer) {
	for _, t := range transfers {
		mt, ok := mat.getMarketTracker(asset, market)
		if !ok {
			continue
		}
		switch t.Type {
		case types.TransferTypeMakerFeePay:
			mat.addFees(mt.makerFeesPaid, t.Owner, t.Amount.Amount, mt.totalMakerFeesPaid.runningTotal)
		case types.TransferTypeMakerFeeReceive:
			mat.addFees(mt.makerFeesReceived, t.Owner, t.Amount.Amount, mt.totalMakerFeesReceived.runningTotal)
		case types.TransferTypeLiquidityFeeDistribute:
			mat.addFees(mt.lpFees, t.Owner, t.Amount.Amount, mt.totalLpFees.runningTotal)
		default:
		}
	}
}

// addFees records fees paid/received in a given metric to a given party.
func (mat *MarketActivityTracker) addFees(m map[string]*feeData, party string, amount *num.Uint, total *num.Uint) {
	if _, ok := m[party]; !ok {
		m[party] = &feeData{
			runningTotal:      amount.Clone(),
			previousEpochs:    make([]*num.Uint, maxWindowSize),
			previousEpochsIdx: 0,
		}
		total.AddSum(amount)
		return
	}
	m[party].runningTotal.AddSum(amount)
	total.AddSum(amount)
}

// getMarketTracker finds the market tracker for a market if one exists (one must exist if the market is active).
func (mat *MarketActivityTracker) getMarketTracker(asset, market string) (*marketTracker, bool) {
	if _, ok := mat.assetToMarketTrackers[asset]; !ok {
		return nil, false
	}
	tracker, ok := mat.assetToMarketTrackers[asset][market]
	if !ok {
		return nil, false
	}
	return tracker, true
}

// RecordPosition passes the position of the party in the asset/market to the market tracker to be recorded.
func (mat *MarketActivityTracker) RecordPosition(asset, party, market string, pos num.Decimal, price *num.Uint, time time.Time) {
	if tracker, ok := mat.getMarketTracker(asset, market); ok {
		tracker.recordPosition(party, pos, time, mat.epochStartTime)
		tracker.recordNotional(party, pos, price, time, mat.epochStartTime)
	}
}

// RecordM2M passes the mark to market win/loss transfer amount to the asset/market tracker to be recorded.
func (mat *MarketActivityTracker) RecordM2M(asset, party, market string, amount num.Decimal) {
	if tracker, ok := mat.getMarketTracker(asset, market); ok {
		tracker.recordM2M(party, amount)
	}
}

func (mat *MarketActivityTracker) getAllParties(asset string, mkts []string, metric vega.DispatchMetric) map[string]struct{} {
	parties := map[string]struct{}{}
	includedMarkets := mkts
	if len(mkts) == 0 {
		includedMarkets = mat.GetAllMarketIDs()
	}
	if len(includedMarkets) > 0 {
		trackers, ok := mat.assetToMarketTrackers[asset]
		if !ok {
			return map[string]struct{}{}
		}
		for _, mkt := range includedMarkets {
			mt, ok := trackers[mkt]
			if !ok {
				continue
			}
			mktParties := mt.getPartiesForMetric(metric)
			for _, v := range mktParties {
				parties[v] = struct{}{}
			}
		}
	}
	return parties
}

func (mat *MarketActivityTracker) getPartiesInScope(ds *vega.DispatchStrategy) []string {
	var parties []string
	if ds.IndividualScope == vega.IndividualScope_INDIVIDUAL_SCOPE_IN_TEAM {
		parties = mat.teams.GetAllPartiesInTeams(mat.minEpochsInTeamForRewardEligibility)
	} else if ds.IndividualScope == vega.IndividualScope_INDIVIDUAL_SCOPE_ALL {
		parties = sortedK(mat.getAllParties(ds.AssetForMetric, ds.Markets, ds.Metric))
	} else if ds.IndividualScope == vega.IndividualScope_INDIVIDUAL_SCOPE_NOT_IN_TEAM {
		parties = sortedK(excludePartiesInTeams(mat.getAllParties(ds.AssetForMetric, ds.Markets, ds.Metric), mat.teams.GetAllPartiesInTeams(mat.minEpochsInTeamForRewardEligibility)))
	}
	return parties
}

// CalculateMetricForIndividuals calculates the metric corresponding to the dispatch strategy and returns a slice of the contribution scores of the parties.
// Markets in scope are the ones passed in the dispatch strategy if any or all available markets for the asset for metric.
// Parties in scope depend on the `IndividualScope_INDIVIDUAL_SCOPE_IN_TEAM` and can include all parties, only those in teams, and only those not in teams.
func (mat *MarketActivityTracker) CalculateMetricForIndividuals(ds *vega.DispatchStrategy) []*types.PartyContibutionScore {
	p, _ := lproto.Marshal(ds)
	hash := hex.EncodeToString(crypto.Hash(p))
	if pc, ok := mat.partyContributionCache[hash]; ok {
		return pc
	}

	parties := mat.getPartiesInScope(ds)
	stakingRequirement, _ := num.UintFromString(ds.StakingRequirement, 10)
	notionalRequirement, _ := num.DecimalFromString(ds.NotionalTimeWeightedAveragePositionRequirement)
	partyContributions := mat.calculateMetricForIndividuals(ds.AssetForMetric, parties, ds.Markets, ds.Metric, stakingRequirement, notionalRequirement, int(ds.WindowLength))

	// we do this calculation at the end of the epoch and clear it in the beginning of the next epoch, i.e. within the same block therefore it saves us
	// redundant calculation and has no snapshot implication
	mat.partyContributionCache[hash] = partyContributions
	return partyContributions
}

// CalculateMetricForTeams calculates the metric for teams and their respective team members for markets in scope of the dispatch strategy.
func (mat *MarketActivityTracker) CalculateMetricForTeams(ds *vega.DispatchStrategy) ([]*types.PartyContibutionScore, map[string][]*types.PartyContibutionScore) {
	teamMembers := make(map[string][]string, len(ds.TeamScope))
	for _, team := range ds.TeamScope {
		teamMembers[team] = mat.teams.GetTeamMembers(team, mat.minEpochsInTeamForRewardEligibility)
	}
	stakingRequirement, _ := num.UintFromString(ds.StakingRequirement, 10)
	notionalRequirement, _ := num.DecimalFromString(ds.NotionalTimeWeightedAveragePositionRequirement)
	topNDecimal := num.MustDecimalFromString(ds.NTopPerformers)
	return mat.calculateMetricForTeams(ds.AssetForMetric, teamMembers, ds.Markets, ds.Metric, stakingRequirement, notionalRequirement, int(ds.WindowLength), topNDecimal)
}

func (mat *MarketActivityTracker) isEligibleForReward(asset, party string, markets []string, minStakingBalanceRequired *num.Uint, notionalTimeWeightedAveragePositionRequired num.Decimal) bool {
	if !minStakingBalanceRequired.IsZero() {
		balance, err := mat.balanceChecker.GetAvailableBalance(party)
		if err != nil || balance.LT(minStakingBalanceRequired) {
			return false
		}
	}
	if !notionalTimeWeightedAveragePositionRequired.IsZero() {
		if mat.getTWNotionalPosition(asset, party, markets).LessThan(notionalTimeWeightedAveragePositionRequired) {
			return false
		}
	}
	return true
}

func (mat *MarketActivityTracker) calculateMetricForIndividuals(asset string, parties []string, markets []string, metric vega.DispatchMetric, minStakingBalanceRequired *num.Uint, notionalTimeWeightedAveragePositionRequired num.Decimal, windowSize int) []*types.PartyContibutionScore {
	ret := make([]*types.PartyContibutionScore, 0, len(parties))
	for _, party := range parties {
		if !mat.isEligibleForReward(asset, party, markets, minStakingBalanceRequired, notionalTimeWeightedAveragePositionRequired) {
			continue
		}
		score := mat.calculateMetricForParty(asset, party, markets, metric, windowSize)
		if score.IsZero() {
			continue
		}
		ret = append(ret, &types.PartyContibutionScore{Party: party, Score: score})
	}
	return ret
}

// CalculateMetricForTeams returns a slice of metrics for the team and a slice of metrics for each team member.
func (mat *MarketActivityTracker) calculateMetricForTeams(asset string, teams map[string][]string, marketsInScope []string, metric vega.DispatchMetric, minStakingBalanceRequired *num.Uint, notionalTimeWeightedAveragePositionRequired num.Decimal, windowSize int, topN num.Decimal) ([]*types.PartyContibutionScore, map[string][]*types.PartyContibutionScore) {
	teamScores := make([]*types.PartyContibutionScore, 0, len(teams))
	teamKeys := make([]string, 0, len(teams))
	for k := range teams {
		teamKeys = append(teamKeys, k)
	}
	sort.Strings(teamKeys)

	ps := make(map[string][]*types.PartyContibutionScore, len(teamScores))
	for _, t := range teamKeys {
		ts, teamMemberScores := mat.calculateMetricForTeam(asset, teams[t], marketsInScope, metric, minStakingBalanceRequired, notionalTimeWeightedAveragePositionRequired, windowSize, topN)
		if ts.IsZero() {
			continue
		}
		teamScores = append(teamScores, &types.PartyContibutionScore{Party: t, Score: ts})
		ps[t] = teamMemberScores
	}

	return teamScores, ps
}

// calculateMetricForTeam returns the metric score for team and a slice of the score for each of its members.
func (mat *MarketActivityTracker) calculateMetricForTeam(asset string, parties []string, marketsInScope []string, metric vega.DispatchMetric, minStakingBalanceRequired *num.Uint, notionalTimeWeightedAveragePositionRequired num.Decimal, windowSize int, topN num.Decimal) (num.Decimal, []*types.PartyContibutionScore) {
	return calculateMetricForTeamUtil(asset, parties, marketsInScope, metric, minStakingBalanceRequired, notionalTimeWeightedAveragePositionRequired, windowSize, topN, mat.isEligibleForReward, mat.calculateMetricForParty)
}

func calculateMetricForTeamUtil(asset string,
	parties []string,
	marketsInScope []string,
	metric vega.DispatchMetric,
	minStakingBalanceRequired *num.Uint,
	notionalTimeWeightedAveragePositionRequired num.Decimal,
	windowSize int,
	topN num.Decimal,
	isEligibleForReward func(asset, party string, markets []string, minStakingBalanceRequired *num.Uint, notionalTimeWeightedAveragePositionRequired num.Decimal) bool,
	calculateMetricForParty func(asset, party string, marketsInScope []string, metric vega.DispatchMetric, windowSize int) num.Decimal,
) (num.Decimal, []*types.PartyContibutionScore) {
	teamPartyScores := []*types.PartyContibutionScore{}
	for _, party := range parties {
		if !isEligibleForReward(asset, party, marketsInScope, minStakingBalanceRequired, notionalTimeWeightedAveragePositionRequired) {
			continue
		}
		teamPartyScores = append(teamPartyScores, &types.PartyContibutionScore{Party: party, Score: calculateMetricForParty(asset, party, marketsInScope, metric, windowSize)})
	}

	if len(teamPartyScores) == 0 {
		return num.DecimalZero(), []*types.PartyContibutionScore{}
	}

	sort.Slice(teamPartyScores, func(i, j int) bool {
		return teamPartyScores[i].Score.GreaterThan(teamPartyScores[j].Score)
	})

	maxIndex := int(topN.Mul(num.DecimalFromInt64(int64(len(parties)))).IntPart())
	if len(teamPartyScores) < maxIndex {
		maxIndex = len(teamPartyScores)
	}

	total := num.DecimalZero()
	for i := 0; i < maxIndex; i++ {
		total = total.Add(teamPartyScores[i].Score)
	}

	return total.Div(num.DecimalFromInt64(int64(maxIndex))), teamPartyScores
}

// calculateMetricForParty returns the value of a reward metric score for the given party for markets of the givem assets which are in scope over the given window size.
func (mat *MarketActivityTracker) calculateMetricForParty(asset, party string, marketsInScope []string, metric vega.DispatchMetric, windowSize int) num.Decimal {
	// exclude unsupported metrics
	if metric == vega.DispatchMetric_DISPATCH_METRIC_MARKET_VALUE {
		mat.log.Panic("unexpected disaptch metric market value here")
	}
	if metric == vega.DispatchMetric_DISPATCH_METRIC_VALIDATOR_RANKING {
		mat.log.Panic("unexpected disaptch metric validator ranking here")
	}
	total := num.DecimalZero()
	marketTotal := num.DecimalZero()
	returns := make([]num.Decimal, windowSize)

	assetTrackers, ok := mat.assetToMarketTrackers[asset]
	if !ok {
		return num.DecimalZero()
	}

	markets := marketsInScope
	if len(markets) == 0 {
		markets = make([]string, 0, len(assetTrackers))
		for k := range assetTrackers {
			markets = append(markets, k)
		}
	}

	// for each market in scope, for each epoch in the time window get the metric entry, sum up for each epoch in the time window and divide by window size (or calculate variance - for volatility)
	for _, market := range markets {
		marketTracker := assetTrackers[market]
		switch metric {
		case vega.DispatchMetric_DISPATCH_METRIC_AVERAGE_POSITION:
			total = total.Add(marketTracker.getPositionMetricTotal(party, windowSize))
		case vega.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN:
			total = total.Add(marketTracker.getRelativeReturnMetricTotal(party, windowSize))
		case vega.DispatchMetric_DISPATCH_METRIC_RETURN_VOLATILITY:
			r, ok := marketTracker.getReturns(party, windowSize)
			if !ok {
				continue
			}
			for i, ret := range r {
				returns[i] = returns[i].Add(ret)
			}
		case vega.DispatchMetric_DISPATCH_METRIC_MAKER_FEES_PAID:
			total = total.Add(getFees(marketTracker.makerFeesPaid, party, windowSize))
			marketTotal = marketTotal.Add(getTotalFees(marketTracker.totalMakerFeesPaid, windowSize))
		case vega.DispatchMetric_DISPATCH_METRIC_MAKER_FEES_RECEIVED:
			total = total.Add(getFees(marketTracker.makerFeesReceived, party, windowSize))
			marketTotal = marketTotal.Add(getTotalFees(marketTracker.totalMakerFeesReceived, windowSize))
		case vega.DispatchMetric_DISPATCH_METRIC_LP_FEES_RECEIVED:
			total = total.Add(getFees(marketTracker.lpFees, party, windowSize))
			marketTotal = marketTotal.Add(getTotalFees(marketTracker.totalLpFees, windowSize))
		}
	}

	switch metric {
	case vega.DispatchMetric_DISPATCH_METRIC_AVERAGE_POSITION:
		return total.Div(num.DecimalFromInt64(int64(windowSize)))
	case vega.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN:
		return num.MaxD(num.DecimalZero(), total.Div(num.DecimalFromInt64(int64(windowSize))))
	case vega.DispatchMetric_DISPATCH_METRIC_RETURN_VOLATILITY:
		filteredReturns := []num.Decimal{}
		for _, d := range returns {
			if !d.IsZero() {
				filteredReturns = append(filteredReturns, d)
			}
		}
		if len(filteredReturns) == 0 {
			return num.DecimalZero()
		}
		variance, _ := num.Variance(filteredReturns)
		return variance
	case vega.DispatchMetric_DISPATCH_METRIC_MAKER_FEES_PAID, vega.DispatchMetric_DISPATCH_METRIC_MAKER_FEES_RECEIVED, vega.DispatchMetric_DISPATCH_METRIC_LP_FEES_RECEIVED:
		if marketTotal.IsZero() {
			return num.DecimalZero()
		}
		return total.Div(marketTotal)
	default:
		mat.log.Panic("unexpected metric")
	}
	return num.DecimalZero()
}

func (mat *MarketActivityTracker) RecordNotionalTakerVolume(party string, volumeToAdd *num.Uint) {
	if _, ok := mat.partyTakerNotionalVolume[party]; !ok {
		mat.partyTakerNotionalVolume[party] = volumeToAdd
		return
	}
	mat.partyTakerNotionalVolume[party].AddSum(volumeToAdd)
}

func (mat *MarketActivityTracker) clearNotionalTakerVolume() {
	mat.partyTakerNotionalVolume = map[string]*num.Uint{}
}

func (mat *MarketActivityTracker) NotionalTakerVolumeForAllParties() map[types.PartyID]*num.Uint {
	res := make(map[types.PartyID]*num.Uint, len(mat.partyTakerNotionalVolume))
	for k, u := range mat.partyTakerNotionalVolume {
		res[types.PartyID(k)] = u.Clone()
	}
	return res
}

func (mat *MarketActivityTracker) NotionalTakerVolumeForParty(party string) *num.Uint {
	if _, ok := mat.partyTakerNotionalVolume[party]; !ok {
		return num.UintZero()
	}
	return mat.partyTakerNotionalVolume[party].Clone()
}

// //// position metric //////
// recordNotional tracks the time weighted average notional for the party per market.
func (mt *marketTracker) recordNotional(party string, pos num.Decimal, price *num.Uint, time time.Time, epochStartTime time.Time) {
	if _, ok := mt.twNotionalPosition[party]; !ok {
		mt.twNotionalPosition[party] = &twNotionalPosition{
			t:                      time,
			position:               pos.Abs(),
			price:                  price,
			currentEpochTWNotional: num.DecimalZero(),
		}
		return
	}
	notional := mt.twNotionalPosition[party]
	t := num.DecimalFromInt64(int64(time.Sub(epochStartTime).Seconds()))
	tn := num.DecimalFromInt64(int64(time.Sub(notional.t).Seconds()))
	tnOverT := num.DecimalZero()
	if !t.IsZero() {
		tnOverT = tn.Div(t)
	}

	notional.currentEpochTWNotional = notional.currentEpochTWNotional.Mul(num.DecimalOne().Sub(tnOverT)).Add((notional.position).Mul(tnOverT).Mul(notional.price.ToDecimal()))
	notional.position = pos.Abs()
	notional.t = time
	notional.price = price
}

func (mt *marketTracker) processNotionalEndOfEpoch(epochStartTime time.Time, endEpochTime time.Time) {
	t := num.DecimalFromInt64(int64(endEpochTime.Sub(epochStartTime).Seconds()))
	for _, twNotional := range mt.twNotionalPosition {
		tn := num.DecimalFromInt64(int64(endEpochTime.Sub(twNotional.t).Seconds()))
		tnOverT := num.DecimalZero()
		if !t.IsZero() {
			tnOverT = tn.Div(t)
		}
		twNotional.currentEpochTWNotional = twNotional.currentEpochTWNotional.Mul(num.DecimalOne().Sub(tnOverT)).Add((twNotional.position).Mul(tnOverT).Mul(twNotional.price.ToDecimal()))
		twNotional.t = endEpochTime
	}
}

func (mat *MarketActivityTracker) getTWNotionalPosition(asset, party string, markets []string) num.Decimal {
	total := num.DecimalZero()
	for _, mkt := range markets {
		if tracker, ok := mat.getMarketTracker(asset, mkt); ok {
			if twNotional, ok := tracker.twNotionalPosition[party]; ok {
				total = total.Add(twNotional.currentEpochTWNotional)
			}
		}
	}
	return total
}

// recordPosition records the current position of a party and the time of change. If there is a previous position then it is time weight updated with respect to the time
// it has been in place during the epoch.
func (mt *marketTracker) recordPosition(party string, pos num.Decimal, time time.Time, epochStartTime time.Time) {
	if _, ok := mt.timeWeightedPosition[party]; !ok {
		mt.timeWeightedPosition[party] = &twPosition{
			position:               pos.Abs(),
			t:                      time,
			currentEpochTWPosition: num.DecimalZero(),
			previousEpochs:         make([]num.Decimal, maxWindowSize),
			previousEpochsIdx:      0,
		}
		return
	}
	toi := mt.timeWeightedPosition[party]
	t := num.DecimalFromInt64(int64(time.Sub(epochStartTime).Seconds()))
	tn := num.DecimalFromInt64(int64(time.Sub(toi.t).Seconds()))
	tnOverT := num.DecimalZero()
	if !t.IsZero() {
		tnOverT = tn.Div(t)
	}
	toi.currentEpochTWPosition = toi.currentEpochTWPosition.Mul(num.DecimalOne().Sub(tnOverT)).Add((toi.position).Mul(tnOverT))
	toi.position = pos.Abs()
	toi.t = time
}

// processPositionEndOfEpoch is called at the end of the epoch, calcualtes the time weight of the current position and moves it to the next epoch, and records
// the time weighted position of the current epoch in the history.
func (mt *marketTracker) processPositionEndOfEpoch(epochStartTime time.Time, endEpochTime time.Time) {
	t := num.DecimalFromInt64(int64(endEpochTime.Sub(epochStartTime).Seconds()))
	for _, toi := range mt.timeWeightedPosition {
		tn := num.DecimalFromInt64(int64(endEpochTime.Sub(toi.t).Seconds()))
		tnOverT := num.DecimalZero()
		if !t.IsZero() {
			tnOverT = tn.Div(t)
		}
		toi.currentEpochTWPosition = toi.currentEpochTWPosition.Mul(num.DecimalOne().Sub(tnOverT)).Add((toi.position).Mul(tnOverT))
		toi.t = endEpochTime
		toi.previousEpochs[toi.previousEpochsIdx] = toi.currentEpochTWPosition
		toi.previousEpochsIdx = (toi.previousEpochsIdx + 1) % maxWindowSize
	}
}

// //// return metric //////

// recordM2M records the amount corresponding to mark to market (profit or loss).
func (mt *marketTracker) recordM2M(party string, amount num.Decimal) {
	if _, ok := mt.partyM2M[party]; !ok {
		mt.partyM2M[party] = &m2mData{
			runningTotal:      amount,
			previousEpochs:    make([]num.Decimal, maxWindowSize),
			previousEpochsIdx: 0,
		}
		return
	}
	m2mData := mt.partyM2M[party]
	m2mData.runningTotal = m2mData.runningTotal.Add(amount)
}

// processM2MEndOfEpoch is called at the end of the epoch to reset the running total for the next epoch and record the total m2m in the ended epoch.
func (mt *marketTracker) processM2MEndOfEpoch() {
	for party, m2m := range mt.partyM2M {
		p := mt.timeWeightedPosition[party].currentEpochTWPosition
		if p.IsZero() {
			m2m.previousEpochs[m2m.previousEpochsIdx] = num.DecimalZero()
		} else {
			m2m.previousEpochs[m2m.previousEpochsIdx] = m2m.runningTotal.Div(p)
		}
		m2m.previousEpochsIdx = (m2m.previousEpochsIdx + 1) % maxWindowSize
		m2m.runningTotal = num.DecimalZero()
	}
}

// getReturns returns a slice of the total of the party's return by epoch in the given window.
func (mt *marketTracker) getReturns(party string, windowSize int) ([]num.Decimal, bool) {
	if _, ok := mt.partyM2M[party]; !ok {
		return []num.Decimal{}, false
	}
	m2mData := mt.partyM2M[party]
	returns := make([]num.Decimal, 0, windowSize)
	for i := 0; i < windowSize; i++ {
		returns = append(returns, num.MaxD(num.DecimalZero(), m2mData.previousEpochs[(m2mData.previousEpochsIdx+maxWindowSize-i-1)%maxWindowSize]))
	}
	return returns, true
}

// getPositionMetricTotal returns the sum of the epoch's time weighted position over the time window.
func (mt *marketTracker) getPositionMetricTotal(party string, windowSize int) num.Decimal {
	if _, ok := mt.timeWeightedPosition[party]; !ok {
		return num.DecimalZero()
	}
	twPos := mt.timeWeightedPosition[party]
	return calcTotalForWindowD(twPos.previousEpochs, twPos.previousEpochsIdx, windowSize)
}

// getRelativeReturnMetricTotal returns the sum of the relative returns over the given window.
func (mt *marketTracker) getRelativeReturnMetricTotal(party string, windowSize int) num.Decimal {
	if _, ok := mt.partyM2M[party]; !ok {
		return num.DecimalZero()
	}
	m2mData := mt.partyM2M[party]
	return calcTotalForWindowD(m2mData.previousEpochs, m2mData.previousEpochsIdx, windowSize)
}

// getPartiesForMetric returns a sorted slice of parties with contribution to the given metric in the market.
func (mt *marketTracker) getPartiesForMetric(metric vega.DispatchMetric) []string {
	switch metric {
	case vega.DispatchMetric_DISPATCH_METRIC_AVERAGE_POSITION:
		return sortedK(mt.timeWeightedPosition)
	case vega.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, vega.DispatchMetric_DISPATCH_METRIC_RETURN_VOLATILITY:
		return sortedK(mt.partyM2M)
	case vega.DispatchMetric_DISPATCH_METRIC_LP_FEES_RECEIVED:
		return sortedK(mt.lpFees)
	case vega.DispatchMetric_DISPATCH_METRIC_MAKER_FEES_PAID:
		return sortedK(mt.makerFeesPaid)
	case vega.DispatchMetric_DISPATCH_METRIC_MAKER_FEES_RECEIVED:
		return sortedK(mt.makerFeesReceived)
	}

	return []string{}
}

// getFees returns the total fees paid/received (depending on what feeData represents) by the party over the given window size.
func getFees(feeData map[string]*feeData, party string, windowSize int) num.Decimal {
	fees, ok := feeData[party]
	if !ok {
		return num.DecimalZero()
	}
	return calcTotalForWindowU(fees.previousEpochs, fees.previousEpochsIdx, windowSize)
}

// getTotalFees returns the total fees of the given type measured over the window size.
func getTotalFees(totalFees *feeData, windowSize int) num.Decimal {
	return calcTotalForWindowU(totalFees.previousEpochs, totalFees.previousEpochsIdx, windowSize)
}

// calcTotalForWindowU returns the total relevant data from the given slice starting from the given dataIdx-1, going back <window_size> elements.
func calcTotalForWindowU(data []*num.Uint, dataIdx int, windowSize int) num.Decimal {
	total := num.UintZero()
	for i := 0; i < windowSize; i++ {
		d := data[(dataIdx+maxWindowSize-i-1)%maxWindowSize]
		if d != nil {
			total.AddSum(d)
		}
	}
	return total.ToDecimal()
}

// calcTotalForWindowD returns the total relevant data from the given slice starting from the given dataIdx-1, going back <window_size> elements.
func calcTotalForWindowD(data []num.Decimal, dataIdx int, windowSize int) num.Decimal {
	total := num.DecimalZero()
	for i := 0; i < windowSize; i++ {
		total = total.Add(data[(dataIdx+maxWindowSize-i-1)%maxWindowSize])
	}
	return total
}

// returns the sorted slice of keys for the given map.
func sortedK[T any](m map[string]T) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// takes a set of all parties and exclude from it the given slice of parties.
func excludePartiesInTeams(allParties map[string]struct{}, partiesInTeams []string) map[string]struct{} {
	for _, v := range partiesInTeams {
		delete(allParties, v)
	}
	return allParties
}
