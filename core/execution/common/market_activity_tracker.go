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
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"time"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/libs/num"
	lproto "code.vegaprotocol.io/vega/libs/proto"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/protos/vega"

	"github.com/shopspring/decimal"
)

const (
	// this the maximum supported window size for any metric.
	maxWindowSize = 100
	// to avoid using decimal calculation we're scaling the time weight by the scaling factor and keep working with integers.
	scalingFactor    = int64(10000000)
	u64ScalingFactor = uint64(scalingFactor)
)

var (
	uScalingFactor = num.NewUint(u64ScalingFactor)
	dScalingFactor = num.DecimalFromInt64(scalingFactor)
)

type QuantumGetter interface {
	GetAssetQuantum(asset string) (num.Decimal, error)
	GetAllParties() []string
}

type twPosition struct {
	position               uint64    // abs last recorded position
	t                      time.Time // time of last recorded position
	currentEpochTWPosition uint64    // current epoch's running time weighted position (scaled by scaling factor)
}

type twNotional struct {
	price                  *num.Uint // last position's price
	notional               *num.Uint // last position's notional value
	t                      time.Time // time of last recorded notional position
	currentEpochTWNotional *num.Uint // current epoch's running time-weighted notional position
}

// marketTracker tracks the activity in the markets in terms of fees and value.
type marketTracker struct {
	asset             string
	makerFeesReceived map[string]*num.Uint
	makerFeesPaid     map[string]*num.Uint
	lpFees            map[string]*num.Uint
	infraFees         map[string]*num.Uint
	lpPaidFees        map[string]*num.Uint
	buybackFeesPaid   map[string]*num.Uint
	treasuryFeesPaid  map[string]*num.Uint
	markPrice         *num.Uint

	notionalVolumeForEpoch *num.Uint

	totalMakerFeesReceived *num.Uint
	totalMakerFeesPaid     *num.Uint
	totalLpFees            *num.Uint

	twPosition          map[string]*twPosition
	partyM2M            map[string]num.Decimal
	partyRealisedReturn map[string]num.Decimal
	twNotional          map[string]*twNotional

	// historical data.
	epochMakerFeesReceived      []map[string]*num.Uint
	epochMakerFeesPaid          []map[string]*num.Uint
	epochLpFees                 []map[string]*num.Uint
	epochTotalMakerFeesReceived []*num.Uint
	epochTotalMakerFeesPaid     []*num.Uint
	epochTotalLpFees            []*num.Uint
	epochTimeWeightedPosition   []map[string]uint64
	epochTimeWeightedNotional   []map[string]*num.Uint
	epochPartyM2M               []map[string]num.Decimal
	epochPartyRealisedReturn    []map[string]num.Decimal
	epochNotionalVolume         []*num.Uint

	valueTraded     *num.Uint
	proposersPaid   map[string]struct{} // identifier of payout_asset : funder : markets_in_scope
	proposer        string
	readyToDelete   bool
	allPartiesCache map[string]struct{}
	// keys of automated market makers
	ammPartiesCache map[string]struct{}
}

// MarketActivityTracker tracks how much fees are paid and received for a market by parties by epoch.
type MarketActivityTracker struct {
	log *logging.Logger

	teams              Teams
	balanceChecker     AccountBalanceChecker
	eligibilityChecker EligibilityChecker
	collateral         QuantumGetter

	currentEpoch                        uint64
	epochStartTime                      time.Time
	minEpochsInTeamForRewardEligibility uint64
	assetToMarketTrackers               map[string]map[string]*marketTracker
	partyContributionCache              map[string][]*types.PartyContributionScore
	partyTakerNotionalVolume            map[string]*num.Uint
	marketToPartyTakerNotionalVolume    map[string]map[string]*num.Uint
	takerFeesPaidInEpoch                []map[string]map[string]map[string]*num.Uint
	// maps game id to eligible parties over time window
	eligibilityInEpoch map[string][]map[string]struct{}

	ss     *snapshotState
	broker Broker
}

// NewMarketActivityTracker instantiates the fees tracker.
func NewMarketActivityTracker(log *logging.Logger, teams Teams, balanceChecker AccountBalanceChecker, broker Broker, collateral QuantumGetter) *MarketActivityTracker {
	mat := &MarketActivityTracker{
		log:                              log,
		balanceChecker:                   balanceChecker,
		teams:                            teams,
		assetToMarketTrackers:            map[string]map[string]*marketTracker{},
		partyContributionCache:           map[string][]*types.PartyContributionScore{},
		partyTakerNotionalVolume:         map[string]*num.Uint{},
		marketToPartyTakerNotionalVolume: map[string]map[string]*num.Uint{},
		ss:                               &snapshotState{},
		takerFeesPaidInEpoch:             []map[string]map[string]map[string]*num.Uint{},
		eligibilityInEpoch:               map[string][]map[string]struct{}{},
		broker:                           broker,
		collateral:                       collateral,
	}

	return mat
}

func (mat *MarketActivityTracker) OnMinEpochsInTeamForRewardEligibilityUpdated(_ context.Context, value int64) error {
	mat.minEpochsInTeamForRewardEligibility = uint64(value)
	return nil
}

// NeedsInitialisation is a heuristic migration - if there is no time weighted position data when restoring from snapshot, we will restore
// positions from the market. This will only happen on the one time migration from a version preceding the new metrics. If we're already on a
// new version, either there are no time-weighted positions and no positions or there are time weighted positions and they will not be restored.
func (mat *MarketActivityTracker) NeedsInitialisation(asset, market string) bool {
	if tracker, ok := mat.getMarketTracker(asset, market); ok {
		return len(tracker.twPosition) == 0
	}
	return false
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
		asset:                       asset,
		proposer:                    proposer,
		proposersPaid:               map[string]struct{}{},
		readyToDelete:               false,
		valueTraded:                 num.UintZero(),
		makerFeesReceived:           map[string]*num.Uint{},
		makerFeesPaid:               map[string]*num.Uint{},
		lpFees:                      map[string]*num.Uint{},
		infraFees:                   map[string]*num.Uint{},
		lpPaidFees:                  map[string]*num.Uint{},
		buybackFeesPaid:             map[string]*num.Uint{},
		treasuryFeesPaid:            map[string]*num.Uint{},
		notionalVolumeForEpoch:      num.UintZero(),
		totalMakerFeesReceived:      num.UintZero(),
		totalMakerFeesPaid:          num.UintZero(),
		totalLpFees:                 num.UintZero(),
		twPosition:                  map[string]*twPosition{},
		partyM2M:                    map[string]num.Decimal{},
		partyRealisedReturn:         map[string]num.Decimal{},
		twNotional:                  map[string]*twNotional{},
		epochTotalMakerFeesReceived: []*num.Uint{},
		epochTotalMakerFeesPaid:     []*num.Uint{},
		epochTotalLpFees:            []*num.Uint{},
		epochMakerFeesReceived:      []map[string]*num.Uint{},
		epochMakerFeesPaid:          []map[string]*num.Uint{},
		epochLpFees:                 []map[string]*num.Uint{},
		epochPartyM2M:               []map[string]num.Decimal{},
		epochPartyRealisedReturn:    []map[string]decimal.Decimal{},
		epochTimeWeightedPosition:   []map[string]uint64{},
		epochNotionalVolume:         []*num.Uint{},
		epochTimeWeightedNotional:   []map[string]*num.Uint{},
		allPartiesCache:             map[string]struct{}{},
		ammPartiesCache:             map[string]struct{}{},
	}

	if ok {
		markets[marketID] = tracker
	} else {
		mat.assetToMarketTrackers[asset] = map[string]*marketTracker{marketID: tracker}
	}
}

// UpdateMarkPrice is called for a futures market when the mark price is recalculated.
func (mat *MarketActivityTracker) UpdateMarkPrice(asset, market string, markPrice *num.Uint) {
	if amt, ok := mat.assetToMarketTrackers[asset]; ok {
		if mt, ok := amt[market]; ok {
			mt.markPrice = markPrice.Clone()
		}
	}
}

// RestoreMarkPrice is called when a market is loaded from a snapshot and will set the price of the notional to
// the mark price is none is set (for migration).
func (mat *MarketActivityTracker) RestoreMarkPrice(asset, market string, markPrice *num.Uint) {
	if amt, ok := mat.assetToMarketTrackers[asset]; ok {
		if mt, ok := amt[market]; ok {
			mt.markPrice = markPrice.Clone()
			for _, twn := range mt.twNotional {
				if twn.price == nil {
					twn.price = markPrice.Clone()
				}
			}
		}
	}
}

func (mat *MarketActivityTracker) PublishGameMetric(ctx context.Context, dispatchStrategy []*vega.DispatchStrategy, now time.Time) {
	m := map[string]map[string]map[string]*num.Uint{}

	for asset, market := range mat.assetToMarketTrackers {
		m[asset] = map[string]map[string]*num.Uint{}
		for mkt, mt := range market {
			m[asset][mkt] = mt.aggregatedFees()
			mt.processNotionalAtMilestone(mat.epochStartTime, now)
			mt.processPositionAtMilestone(mat.epochStartTime, now)
			mt.processM2MAtMilestone()
			mt.processPartyRealisedReturnAtMilestone()
			mt.calcFeesAtMilestone()
		}
	}
	mat.takerFeesPaidInEpoch = append(mat.takerFeesPaidInEpoch, m)
	for ds := range mat.eligibilityInEpoch {
		mat.eligibilityInEpoch[ds] = append(mat.eligibilityInEpoch[ds], map[string]struct{}{})
	}

	for _, ds := range dispatchStrategy {
		if ds.Metric == vega.DispatchMetric_DISPATCH_METRIC_VALIDATOR_RANKING || ds.Metric == vega.DispatchMetric_DISPATCH_METRIC_LP_FEES_RECEIVED ||
			ds.Metric == vega.DispatchMetric_DISPATCH_METRIC_MARKET_VALUE {
			continue
		}
		mat.publishMetricForDispatchStrategy(ctx, ds, now)
	}

	for _, market := range mat.assetToMarketTrackers {
		for _, mt := range market {
			mt.epochTimeWeightedNotional = mt.epochTimeWeightedNotional[:len(mt.epochTimeWeightedNotional)-1]
			mt.epochTimeWeightedPosition = mt.epochTimeWeightedPosition[:len(mt.epochTimeWeightedPosition)-1]
			mt.epochPartyM2M = mt.epochPartyM2M[:len(mt.epochPartyM2M)-1]
			mt.epochPartyRealisedReturn = mt.epochPartyRealisedReturn[:len(mt.epochPartyRealisedReturn)-1]
			mt.epochMakerFeesReceived = mt.epochMakerFeesReceived[:len(mt.epochMakerFeesReceived)-1]
			mt.epochMakerFeesPaid = mt.epochMakerFeesPaid[:len(mt.epochMakerFeesPaid)-1]
			mt.epochLpFees = mt.epochLpFees[:len(mt.epochLpFees)-1]
			mt.epochTotalMakerFeesReceived = mt.epochTotalMakerFeesReceived[:len(mt.epochTotalMakerFeesReceived)-1]
			mt.epochTotalMakerFeesPaid = mt.epochTotalMakerFeesPaid[:len(mt.epochTotalMakerFeesPaid)-1]
			mt.epochTotalLpFees = mt.epochTotalLpFees[:len(mt.epochTotalLpFees)-1]
		}
	}
	mat.takerFeesPaidInEpoch = mat.takerFeesPaidInEpoch[:len(mat.takerFeesPaidInEpoch)-1]
	mat.partyContributionCache = map[string][]*types.PartyContributionScore{}
	for ds := range mat.eligibilityInEpoch {
		mat.eligibilityInEpoch[ds] = mat.eligibilityInEpoch[ds][:len(mat.eligibilityInEpoch)-1]
	}
}

func (mat *MarketActivityTracker) publishMetricForDispatchStrategy(ctx context.Context, ds *vega.DispatchStrategy, now time.Time) {
	if ds.EntityScope == vega.EntityScope_ENTITY_SCOPE_INDIVIDUALS {
		partyScores := mat.CalculateMetricForIndividuals(ctx, ds)
		gs := events.NewPartyGameScoresEvent(ctx, int64(mat.currentEpoch), getGameID(ds), now, partyScores)
		mat.broker.Send(gs)
	} else {
		teamScores, partyScores := mat.CalculateMetricForTeams(ctx, ds)
		gs := events.NewTeamGameScoresEvent(ctx, int64(mat.currentEpoch), getGameID(ds), now, teamScores, partyScores)
		mat.broker.Send(gs)
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

// AddAMMSubAccount records sub account entries for AMM in given market.
func (mat *MarketActivityTracker) AddAMMSubAccount(asset, marketID, subAccount string) {
	markets, ok := mat.assetToMarketTrackers[asset]
	if !ok || markets[marketID] == nil {
		return
	}
	markets[marketID].ammPartiesCache[subAccount] = struct{}{}
}

// RemoveAMMParty removes amm party entries for AMM in given market.
func (mat *MarketActivityTracker) RemoveAMMParty(asset, marketID, ammParty string) {
	markets, ok := mat.assetToMarketTrackers[asset]
	if !ok || markets[marketID] == nil {
		return
	}
	delete(markets[marketID].ammPartiesCache, ammParty)
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
		sort.Strings(mkts)
	}

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
			Metric: vega.DispatchMetric_DISPATCH_METRIC_MARKET_VALUE,
		})
		totalScore = totalScore.Add(score)
	}

	mat.clipScoresAt1(scores, totalScore)
	scoresString := ""

	for _, mcs := range scores {
		scoresString += mcs.Market + ":" + mcs.Score.String() + ","
	}
	mat.log.Info("markets contributions:", logging.String("asset", asset), logging.String("metric", vega.DispatchMetric_name[int32(vega.DispatchMetric_DISPATCH_METRIC_MARKET_VALUE)]), logging.String("market-scores", scoresString[:len(scoresString)-1]))

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

// MarkPaidProposer marks the proposer of the market as having been paid proposer bonus.
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

// MarketTrackedForAsset returns whether the given market is seen to have the given asset by the tracker.
func (mat *MarketActivityTracker) MarketTrackedForAsset(market, asset string) bool {
	if markets, ok := mat.assetToMarketTrackers[asset]; ok {
		if _, ok = markets[market]; ok {
			return true
		}
	}
	return false
}

// RemoveMarket is called when the market is removed from the network. It is not immediately removed to give a chance for rewards to be paid at the end of the epoch for activity during the epoch.
// Instead it is marked for removal and will be removed at the beginning of the next epoch.
func (mat *MarketActivityTracker) RemoveMarket(asset, marketID string) {
	if markets, ok := mat.assetToMarketTrackers[asset]; ok {
		if m, ok := markets[marketID]; ok {
			m.readyToDelete = true
		}
	}
}

func (mt *marketTracker) aggregatedFees() map[string]*num.Uint {
	totalFees := map[string]*num.Uint{}
	fees := []map[string]*num.Uint{mt.infraFees, mt.lpPaidFees, mt.makerFeesPaid, mt.buybackFeesPaid, mt.treasuryFeesPaid}
	for _, fee := range fees {
		for party, paid := range fee {
			if _, ok := totalFees[party]; !ok {
				totalFees[party] = num.UintZero()
			}
			totalFees[party].AddSum(paid)
		}
	}
	return totalFees
}

// OnEpochEvent is called when the state of the epoch changes, we only care about new epochs starting.
func (mat *MarketActivityTracker) OnEpochEvent(ctx context.Context, epoch types.Epoch) {
	if epoch.Action == vega.EpochAction_EPOCH_ACTION_START {
		mat.epochStartTime = epoch.StartTime
		mat.partyContributionCache = map[string][]*types.PartyContributionScore{}
		mat.clearDeletedMarkets()
		mat.clearNotionalTakerVolume()
	} else if epoch.Action == vega.EpochAction_EPOCH_ACTION_END {
		m := map[string]map[string]map[string]*num.Uint{}
		for asset, market := range mat.assetToMarketTrackers {
			m[asset] = map[string]map[string]*num.Uint{}
			for mkt, mt := range market {
				m[asset][mkt] = mt.aggregatedFees()
				mt.processNotionalEndOfEpoch(epoch.StartTime, epoch.EndTime)
				mt.processPositionEndOfEpoch(epoch.StartTime, epoch.EndTime)
				mt.processM2MEndOfEpoch()
				mt.processPartyRealisedReturnOfEpoch()
				mt.clearFeeActivity()
				if len(mt.epochNotionalVolume) == maxWindowSize {
					mt.epochNotionalVolume = mt.epochNotionalVolume[1:]
				}
				mt.epochNotionalVolume = append(mt.epochNotionalVolume, mt.notionalVolumeForEpoch)
				mt.notionalVolumeForEpoch = num.UintZero()
			}
		}
		if len(mat.takerFeesPaidInEpoch) == maxWindowSize {
			mat.takerFeesPaidInEpoch = mat.takerFeesPaidInEpoch[1:]
		}
		mat.takerFeesPaidInEpoch = append(mat.takerFeesPaidInEpoch, m)
		for ds := range mat.eligibilityInEpoch {
			mat.eligibilityInEpoch[ds] = append(mat.eligibilityInEpoch[ds], map[string]struct{}{})
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

func (mat *MarketActivityTracker) GetNotionalVolumeForAsset(asset string, markets []string, windowSize int) *num.Uint {
	total := num.UintZero()
	trackers, ok := mat.assetToMarketTrackers[asset]
	if !ok {
		return total
	}
	marketsInScope := map[string]struct{}{}
	for _, mkt := range markets {
		marketsInScope[mkt] = struct{}{}
	}
	if len(markets) == 0 {
		for mkt := range trackers {
			marketsInScope[mkt] = struct{}{}
		}
	}
	for mkt := range marketsInScope {
		for i := 0; i < windowSize; i++ {
			idx := len(trackers[mkt].epochNotionalVolume) - i - 1
			if idx < 0 {
				break
			}
			total.AddSum(trackers[mkt].epochNotionalVolume[idx])
		}
	}
	return total
}

func (mat *MarketActivityTracker) CalculateTotalMakerContributionInQuantum(windowSize int) (map[string]*num.Uint, map[string]num.Decimal) {
	m := map[string]*num.Uint{}
	total := num.UintZero()
	for ast, trackers := range mat.assetToMarketTrackers {
		quantum, err := mat.collateral.GetAssetQuantum(ast)
		if err != nil {
			continue
		}
		for _, trckr := range trackers {
			for i := 0; i < windowSize; i++ {
				idx := len(trckr.epochMakerFeesReceived) - i - 1
				if idx < 0 {
					break
				}
				partyFees := trckr.epochMakerFeesReceived[len(trckr.epochMakerFeesReceived)-i-1]
				for party, fees := range partyFees {
					if _, ok := m[party]; !ok {
						m[party] = num.UintZero()
					}
					feesInQunatum, overflow := num.UintFromDecimal(fees.ToDecimal().Div(quantum))
					if overflow {
						continue
					}
					m[party].AddSum(feesInQunatum)
					total.AddSum(feesInQunatum)
				}
			}
		}
	}
	if total.IsZero() {
		return m, map[string]decimal.Decimal{}
	}
	totalFrac := num.DecimalZero()
	fractions := []*types.PartyContributionScore{}
	for p, f := range m {
		frac := f.ToDecimal().Div(total.ToDecimal())
		fractions = append(fractions, &types.PartyContributionScore{Party: p, Score: frac})
		totalFrac = totalFrac.Add(frac)
	}
	capAtOne(fractions, totalFrac)
	fracMap := make(map[string]num.Decimal, len(fractions))
	for _, partyFraction := range fractions {
		fracMap[partyFraction.Party] = partyFraction.Score
	}
	return m, fracMap
}

func capAtOne(partyFractions []*types.PartyContributionScore, total num.Decimal) {
	if total.LessThanOrEqual(num.DecimalOne()) {
		return
	}

	sort.SliceStable(partyFractions, func(i, j int) bool { return partyFractions[i].Score.GreaterThan(partyFractions[j].Score) })
	delta := total.Sub(num.DecimalFromInt64(1))
	partyFractions[0].Score = num.MaxD(num.DecimalZero(), partyFractions[0].Score.Sub(delta))
}

func (mt *marketTracker) calcFeesAtMilestone() {
	mt.epochMakerFeesReceived = append(mt.epochMakerFeesReceived, mt.makerFeesReceived)
	mt.epochMakerFeesPaid = append(mt.epochMakerFeesPaid, mt.makerFeesPaid)
	mt.epochLpFees = append(mt.epochLpFees, mt.lpFees)
	mt.epochTotalMakerFeesReceived = append(mt.epochTotalMakerFeesReceived, mt.totalMakerFeesReceived)
	mt.epochTotalMakerFeesPaid = append(mt.epochTotalMakerFeesPaid, mt.totalMakerFeesPaid)
	mt.epochTotalLpFees = append(mt.epochTotalLpFees, mt.totalLpFees)
}

// clearFeeActivity is called at the end of the epoch. It deletes markets that are pending to be removed and resets the fees paid for the epoch.
func (mt *marketTracker) clearFeeActivity() {
	if len(mt.epochMakerFeesReceived) == maxWindowSize {
		mt.epochMakerFeesReceived = mt.epochMakerFeesReceived[1:]
		mt.epochMakerFeesPaid = mt.epochMakerFeesPaid[1:]
		mt.epochLpFees = mt.epochLpFees[1:]
		mt.epochTotalMakerFeesReceived = mt.epochTotalMakerFeesReceived[1:]
		mt.epochTotalMakerFeesPaid = mt.epochTotalMakerFeesPaid[1:]
		mt.epochTotalLpFees = mt.epochTotalLpFees[1:]
	}
	mt.epochMakerFeesReceived = append(mt.epochMakerFeesReceived, mt.makerFeesReceived)
	mt.epochMakerFeesPaid = append(mt.epochMakerFeesPaid, mt.makerFeesPaid)
	mt.epochLpFees = append(mt.epochLpFees, mt.lpFees)
	mt.makerFeesReceived = map[string]*num.Uint{}
	mt.makerFeesPaid = map[string]*num.Uint{}
	mt.lpFees = map[string]*num.Uint{}
	mt.infraFees = map[string]*num.Uint{}
	mt.lpPaidFees = map[string]*num.Uint{}
	mt.treasuryFeesPaid = map[string]*num.Uint{}
	mt.buybackFeesPaid = map[string]*num.Uint{}

	mt.epochTotalMakerFeesReceived = append(mt.epochTotalMakerFeesReceived, mt.totalMakerFeesReceived)
	mt.epochTotalMakerFeesPaid = append(mt.epochTotalMakerFeesPaid, mt.totalMakerFeesPaid)
	mt.epochTotalLpFees = append(mt.epochTotalLpFees, mt.totalLpFees)
	mt.totalMakerFeesReceived = num.UintZero()
	mt.totalMakerFeesPaid = num.UintZero()
	mt.totalLpFees = num.UintZero()
}

// UpdateFeesFromTransfers takes a slice of transfers and if they represent fees it updates the market fee tracker.
// market is guaranteed to exist in the mapping as it is added when proposed.
func (mat *MarketActivityTracker) UpdateFeesFromTransfers(asset, market string, transfers []*types.Transfer) {
	for _, t := range transfers {
		mt, ok := mat.getMarketTracker(asset, market)
		if !ok {
			continue
		}
		mt.allPartiesCache[t.Owner] = struct{}{}
		switch t.Type {
		case types.TransferTypeMakerFeePay:
			mat.addFees(mt.makerFeesPaid, t.Owner, t.Amount.Amount, mt.totalMakerFeesPaid)
		case types.TransferTypeMakerFeeReceive:
			mat.addFees(mt.makerFeesReceived, t.Owner, t.Amount.Amount, mt.totalMakerFeesReceived)
		case types.TransferTypeLiquidityFeeNetDistribute, types.TransferTypeSlaPerformanceBonusDistribute:
			mat.addFees(mt.lpFees, t.Owner, t.Amount.Amount, mt.totalLpFees)
		case types.TransferTypeInfrastructureFeePay:
			mat.addFees(mt.infraFees, t.Owner, t.Amount.Amount, num.UintZero())
		case types.TransferTypeLiquidityFeePay:
			mat.addFees(mt.lpPaidFees, t.Owner, t.Amount.Amount, num.UintZero())
		case types.TransferTypeBuyBackFeePay:
			mat.addFees(mt.buybackFeesPaid, t.Owner, t.Amount.Amount, num.UintZero())
		case types.TransferTypeTreasuryPay:
			mat.addFees(mt.treasuryFeesPaid, t.Owner, t.Amount.Amount, num.UintZero())
		case types.TransferTypeHighMakerRebateReceive:
			// we count high maker fee receive as maker fees for that purpose.
			mat.addFees(mt.makerFeesReceived, t.Owner, t.Amount.Amount, mt.totalMakerFeesReceived)
		default:
		}
	}
}

// addFees records fees paid/received in a given metric to a given party.
func (mat *MarketActivityTracker) addFees(m map[string]*num.Uint, party string, amount *num.Uint, total *num.Uint) {
	if _, ok := m[party]; !ok {
		m[party] = amount.Clone()
		total.AddSum(amount)
		return
	}
	m[party].AddSum(amount)
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

// RestorePosition restores a position as if it were acquired at the beginning of the epoch. This is purely for migration from an old version.
func (mat *MarketActivityTracker) RestorePosition(asset, party, market string, pos int64, price *num.Uint, positionFactor num.Decimal) {
	mat.RecordPosition(asset, party, market, pos, price, positionFactor, mat.epochStartTime)
}

// RecordPosition passes the position of the party in the asset/market to the market tracker to be recorded.
func (mat *MarketActivityTracker) RecordPosition(asset, party, market string, pos int64, price *num.Uint, positionFactor num.Decimal, time time.Time) {
	if tracker, ok := mat.getMarketTracker(asset, market); ok {
		tracker.allPartiesCache[party] = struct{}{}
		absPos := uint64(0)
		if pos > 0 {
			absPos = uint64(pos)
		} else if pos < 0 {
			absPos = uint64(-pos)
		}
		notional, _ := num.UintFromDecimal(num.UintZero().Mul(num.NewUint(absPos), price).ToDecimal().Div(positionFactor))
		tracker.recordPosition(party, absPos, positionFactor, time, mat.epochStartTime)
		tracker.recordNotional(party, notional, price, time, mat.epochStartTime)
	}
}

// RecordRealisedPosition updates the market tracker on decreased position.
func (mat *MarketActivityTracker) RecordRealisedPosition(asset, party, market string, positionDecrease num.Decimal) {
	if tracker, ok := mat.getMarketTracker(asset, market); ok {
		tracker.allPartiesCache[party] = struct{}{}
		tracker.recordRealisedPosition(party, positionDecrease)
	}
}

// RecordM2M passes the mark to market win/loss transfer amount to the asset/market tracker to be recorded.
func (mat *MarketActivityTracker) RecordM2M(asset, party, market string, amount num.Decimal) {
	if tracker, ok := mat.getMarketTracker(asset, market); ok {
		tracker.allPartiesCache[party] = struct{}{}
		tracker.recordM2M(party, amount)
	}
}

// RecordFundingPayment passes the mark to market win/loss transfer amount to the asset/market tracker to be recorded.
func (mat *MarketActivityTracker) RecordFundingPayment(asset, party, market string, amount num.Decimal) {
	if tracker, ok := mat.getMarketTracker(asset, market); ok {
		tracker.allPartiesCache[party] = struct{}{}
		tracker.recordFundingPayment(party, amount)
	}
}

func (mat *MarketActivityTracker) filterParties(
	asset string,
	mkts []string,
	cacheFilter func(*marketTracker) map[string]struct{},
) map[string]struct{} {
	parties := map[string]struct{}{}
	includedMarkets := mkts
	if len(mkts) == 0 {
		includedMarkets = mat.GetAllMarketIDs()
	}
	assets := []string{}
	if len(asset) == 0 {
		assets = make([]string, 0, len(mat.assetToMarketTrackers))
		for k := range mat.assetToMarketTrackers {
			assets = append(assets, k)
		}
		sort.Strings(assets)
	} else {
		assets = append(assets, asset)
	}

	if len(includedMarkets) > 0 {
		for _, ast := range assets {
			trackers, ok := mat.assetToMarketTrackers[ast]
			if !ok {
				continue
			}
			for _, mkt := range includedMarkets {
				mt, ok := trackers[mkt]
				if !ok {
					continue
				}
				mktParties := cacheFilter(mt)
				for k := range mktParties {
					parties[k] = struct{}{}
				}
			}
		}
	}
	return parties
}

func (mat *MarketActivityTracker) getAllParties(asset string, mkts []string) map[string]struct{} {
	return mat.filterParties(asset, mkts, func(mt *marketTracker) map[string]struct{} {
		return mt.allPartiesCache
	})
}

func (mat *MarketActivityTracker) GetAllAMMParties(asset string, mkts []string) map[string]struct{} {
	return mat.filterParties(asset, mkts, func(mt *marketTracker) map[string]struct{} {
		return mt.ammPartiesCache
	})
}

func (mat *MarketActivityTracker) getPartiesInScope(ds *vega.DispatchStrategy) []string {
	var parties []string
	if ds.IndividualScope == vega.IndividualScope_INDIVIDUAL_SCOPE_IN_TEAM {
		parties = mat.teams.GetAllPartiesInTeams(mat.minEpochsInTeamForRewardEligibility)
	} else if ds.IndividualScope == vega.IndividualScope_INDIVIDUAL_SCOPE_ALL {
		if ds.Metric == vega.DispatchMetric_DISPATCH_METRIC_ELIGIBLE_ENTITIES {
			notionalReq := num.UintZero()
			stakingReq := num.UintZero()
			if len(ds.NotionalTimeWeightedAveragePositionRequirement) > 0 {
				notionalReq = num.MustUintFromString(ds.NotionalTimeWeightedAveragePositionRequirement, 10)
			}
			if len(ds.StakingRequirement) > 0 {
				stakingReq = num.MustUintFromString(ds.StakingRequirement, 10)
			}
			if !notionalReq.IsZero() {
				parties = sortedK(mat.getAllParties(ds.AssetForMetric, ds.Markets))
			} else if !stakingReq.IsZero() {
				parties = mat.balanceChecker.GetAllStakingParties()
			} else {
				parties = mat.collateral.GetAllParties()
			}
		} else {
			parties = sortedK(mat.getAllParties(ds.AssetForMetric, ds.Markets))
		}
	} else if ds.IndividualScope == vega.IndividualScope_INDIVIDUAL_SCOPE_NOT_IN_TEAM {
		parties = sortedK(excludePartiesInTeams(mat.getAllParties(ds.AssetForMetric, ds.Markets), mat.teams.GetAllPartiesInTeams(mat.minEpochsInTeamForRewardEligibility)))
	} else if ds.IndividualScope == vega.IndividualScope_INDIVIDUAL_SCOPE_AMM {
		parties = sortedK(mat.GetAllAMMParties(ds.AssetForMetric, ds.Markets))
	}
	return parties
}

func getGameID(ds *vega.DispatchStrategy) string {
	p, _ := lproto.Marshal(ds)
	return hex.EncodeToString(crypto.Hash(p))
}

func (mat *MarketActivityTracker) GameFinished(gameID string) {
	delete(mat.eligibilityInEpoch, gameID)
}

// CalculateMetricForIndividuals calculates the metric corresponding to the dispatch strategy and returns a slice of the contribution scores of the parties.
// Markets in scope are the ones passed in the dispatch strategy if any or all available markets for the asset for metric.
// Parties in scope depend on the `IndividualScope_INDIVIDUAL_SCOPE_IN_TEAM` and can include all parties, only those in teams, and only those not in teams.
func (mat *MarketActivityTracker) CalculateMetricForIndividuals(ctx context.Context, ds *vega.DispatchStrategy) []*types.PartyContributionScore {
	hash := getGameID(ds)
	if pc, ok := mat.partyContributionCache[hash]; ok {
		return pc
	}

	parties := mat.getPartiesInScope(ds)
	stakingRequirement, _ := num.UintFromString(ds.StakingRequirement, 10)
	notionalRequirement, _ := num.UintFromString(ds.NotionalTimeWeightedAveragePositionRequirement, 10)
	interval := int32(1)
	if ds.TransferInterval != nil {
		interval = *ds.TransferInterval
	}
	partyContributions := mat.calculateMetricForIndividuals(ctx, ds.AssetForMetric, parties, ds.Markets, ds.Metric, stakingRequirement, notionalRequirement, int(ds.WindowLength), hash, interval)

	// we do this calculation at the end of the epoch and clear it in the beginning of the next epoch, i.e. within the same block therefore it saves us
	// redundant calculation and has no snapshot implication
	mat.partyContributionCache[hash] = partyContributions
	return partyContributions
}

// CalculateMetricForTeams calculates the metric for teams and their respective team members for markets in scope of the dispatch strategy.
func (mat *MarketActivityTracker) CalculateMetricForTeams(ctx context.Context, ds *vega.DispatchStrategy) ([]*types.PartyContributionScore, map[string][]*types.PartyContributionScore) {
	var teamMembers map[string][]string
	interval := int32(1)
	if ds.TransferInterval != nil {
		interval = *ds.TransferInterval
	}
	paidFees := mat.GetLastEpochTakeFees(ds.AssetForMetric, ds.Markets, interval)
	if tsl := len(ds.TeamScope); tsl > 0 {
		teamMembers = make(map[string][]string, len(ds.TeamScope))
		for _, team := range ds.TeamScope {
			teamMembers[team] = mat.teams.GetTeamMembers(team, mat.minEpochsInTeamForRewardEligibility)
		}
	} else {
		teamMembers = mat.teams.GetAllTeamsWithParties(mat.minEpochsInTeamForRewardEligibility)
	}
	stakingRequirement, _ := num.UintFromString(ds.StakingRequirement, 10)
	notionalRequirement, _ := num.UintFromString(ds.NotionalTimeWeightedAveragePositionRequirement, 10)
	topNDecimal := num.MustDecimalFromString(ds.NTopPerformers)

	p, _ := lproto.Marshal(ds)
	gameID := hex.EncodeToString(crypto.Hash(p))

	return mat.calculateMetricForTeams(ctx, ds.AssetForMetric, teamMembers, ds.Markets, ds.Metric, stakingRequirement, notionalRequirement, int(ds.WindowLength), topNDecimal, gameID, paidFees)
}

func (mat *MarketActivityTracker) isEligibleForReward(ctx context.Context, asset, party string, markets []string, minStakingBalanceRequired *num.Uint, notionalTimeWeightedAveragePositionRequired *num.Uint, gameID string) (bool, *num.Uint, *num.Uint) {
	eligiblByBalance := true
	eligibleByNotional := true
	var balance, notional *num.Uint
	var err error

	balance, err = mat.balanceChecker.GetAvailableBalance(party)
	if err != nil || balance.LT(minStakingBalanceRequired) {
		eligiblByBalance = false
		if balance == nil {
			balance = num.UintZero()
		}
	}

	notional = mat.getTWNotionalPosition(asset, party, markets)
	mat.broker.Send(events.NewTimeWeightedNotionalPositionUpdated(ctx, mat.currentEpoch, asset, party, gameID, notional.String()))
	if notional.LT(notionalTimeWeightedAveragePositionRequired) {
		eligibleByNotional = false
	}

	isEligible := (eligiblByBalance || minStakingBalanceRequired.IsZero()) && (eligibleByNotional || notionalTimeWeightedAveragePositionRequired.IsZero())
	return isEligible, balance, notional
}

func getEligibilityScore(party, gameID string, eligibilityInEpoch map[string][]map[string]struct{}, balance *num.Uint, notional *num.Uint, paidFees map[string]*num.Uint, windowSize int) *types.PartyContributionScore {
	if _, ok := eligibilityInEpoch[gameID]; !ok {
		eligibilityInEpoch[gameID] = []map[string]struct{}{{}}
		eligibilityInEpoch[gameID][0][party] = struct{}{}
		return &types.PartyContributionScore{Party: party, Score: num.DecimalOne(), IsEligible: true, StakingBalance: balance, OpenVolume: notional, TotalFeesPaid: paidFees[party], RankingIndex: -1}
	}
	m := eligibilityInEpoch[gameID]
	if len(m) > windowSize {
		m = m[1:]
	}
	m[len(m)-1][party] = struct{}{}
	for _, mm := range m {
		if _, ok := mm[party]; !ok {
			return &types.PartyContributionScore{Party: party, Score: num.DecimalZero(), IsEligible: false, StakingBalance: balance, OpenVolume: notional, TotalFeesPaid: paidFees[party], RankingIndex: -1}
		}
	}
	return &types.PartyContributionScore{Party: party, Score: num.DecimalOne(), IsEligible: true, StakingBalance: balance, OpenVolume: notional, TotalFeesPaid: paidFees[party], RankingIndex: -1}
}

func (mat *MarketActivityTracker) calculateMetricForIndividuals(ctx context.Context, asset string, parties []string, markets []string, metric vega.DispatchMetric, minStakingBalanceRequired *num.Uint, notionalTimeWeightedAveragePositionRequired *num.Uint, windowSize int, gameID string, interval int32) []*types.PartyContributionScore {
	ret := make([]*types.PartyContributionScore, 0, len(parties))
	paidFees := mat.GetLastEpochTakeFees(asset, markets, interval)
	for _, party := range parties {
		eligible, balance, notional := mat.isEligibleForReward(ctx, asset, party, markets, minStakingBalanceRequired, notionalTimeWeightedAveragePositionRequired, gameID)
		if !eligible {
			ret = append(ret, &types.PartyContributionScore{Party: party, Score: num.DecimalZero(), IsEligible: eligible, StakingBalance: balance, OpenVolume: notional, TotalFeesPaid: paidFees[party], RankingIndex: -1})
			continue
		}
		if metric == vega.DispatchMetric_DISPATCH_METRIC_ELIGIBLE_ENTITIES {
			ret = append(ret, getEligibilityScore(party, gameID, mat.eligibilityInEpoch, balance, notional, paidFees, windowSize))
			continue
		}
		score, ok := mat.calculateMetricForParty(asset, party, markets, metric, windowSize)
		if !ok {
			ret = append(ret, &types.PartyContributionScore{Party: party, Score: num.DecimalZero(), IsEligible: false, StakingBalance: balance, OpenVolume: notional, TotalFeesPaid: paidFees[party], RankingIndex: -1})
			continue
		}
		ret = append(ret, &types.PartyContributionScore{Party: party, Score: score, IsEligible: true, StakingBalance: balance, OpenVolume: notional, TotalFeesPaid: paidFees[party], RankingIndex: -1})
	}
	return ret
}

// CalculateMetricForTeams returns a slice of metrics for the team and a slice of metrics for each team member.
func (mat *MarketActivityTracker) calculateMetricForTeams(ctx context.Context, asset string, teams map[string][]string, marketsInScope []string, metric vega.DispatchMetric, minStakingBalanceRequired *num.Uint, notionalTimeWeightedAveragePositionRequired *num.Uint, windowSize int, topN num.Decimal, gameID string, paidFees map[string]*num.Uint) ([]*types.PartyContributionScore, map[string][]*types.PartyContributionScore) {
	teamScores := make([]*types.PartyContributionScore, 0, len(teams))
	teamKeys := make([]string, 0, len(teams))
	for k := range teams {
		teamKeys = append(teamKeys, k)
	}
	sort.Strings(teamKeys)
	ps := make(map[string][]*types.PartyContributionScore, len(teamScores))
	for _, t := range teamKeys {
		ts, teamMemberScores := mat.calculateMetricForTeam(ctx, asset, teams[t], marketsInScope, metric, minStakingBalanceRequired, notionalTimeWeightedAveragePositionRequired, windowSize, topN, gameID, paidFees)
		if ts.IsZero() {
			continue
		}
		teamScores = append(teamScores, &types.PartyContributionScore{Party: t, Score: ts})
		ps[t] = teamMemberScores
	}

	return teamScores, ps
}

// calculateMetricForTeam returns the metric score for team and a slice of the score for each of its members.
func (mat *MarketActivityTracker) calculateMetricForTeam(ctx context.Context, asset string, parties []string, marketsInScope []string, metric vega.DispatchMetric, minStakingBalanceRequired *num.Uint, notionalTimeWeightedAveragePositionRequired *num.Uint, windowSize int, topN num.Decimal, gameID string, paidFees map[string]*num.Uint) (num.Decimal, []*types.PartyContributionScore) {
	return calculateMetricForTeamUtil(ctx, asset, parties, marketsInScope, metric, minStakingBalanceRequired, notionalTimeWeightedAveragePositionRequired, windowSize, topN, mat.isEligibleForReward, mat.calculateMetricForParty, gameID, paidFees, mat.eligibilityInEpoch)
}

func calculateMetricForTeamUtil(ctx context.Context,
	asset string,
	parties []string,
	marketsInScope []string,
	metric vega.DispatchMetric,
	minStakingBalanceRequired *num.Uint,
	notionalTimeWeightedAveragePositionRequired *num.Uint,
	windowSize int,
	topN num.Decimal,
	isEligibleForReward func(ctx context.Context, asset, party string, markets []string, minStakingBalanceRequired *num.Uint, notionalTimeWeightedAveragePositionRequired *num.Uint, gameID string) (bool, *num.Uint, *num.Uint),
	calculateMetricForParty func(asset, party string, marketsInScope []string, metric vega.DispatchMetric, windowSize int) (num.Decimal, bool),
	gameID string,
	paidFees map[string]*num.Uint,
	eligibilityInEpoch map[string][]map[string]struct{},
) (num.Decimal, []*types.PartyContributionScore) {
	teamPartyScores := []*types.PartyContributionScore{}
	eligibleTeamPartyScores := []*types.PartyContributionScore{}
	for _, party := range parties {
		eligible, balance, notional := isEligibleForReward(ctx, asset, party, marketsInScope, minStakingBalanceRequired, notionalTimeWeightedAveragePositionRequired, gameID)
		if !eligible {
			teamPartyScores = append(teamPartyScores, &types.PartyContributionScore{Party: party, Score: num.DecimalZero(), IsEligible: eligible, StakingBalance: balance, OpenVolume: notional, TotalFeesPaid: paidFees[party], RankingIndex: -1})
			continue
		}

		if metric == vega.DispatchMetric_DISPATCH_METRIC_ELIGIBLE_ENTITIES {
			score := getEligibilityScore(party, gameID, eligibilityInEpoch, balance, notional, paidFees, windowSize)
			teamPartyScores = append(teamPartyScores, score)
			if score.IsEligible {
				eligibleTeamPartyScores = append(eligibleTeamPartyScores, score)
			}
			continue
		}
		if score, ok := calculateMetricForParty(asset, party, marketsInScope, metric, windowSize); ok {
			teamPartyScores = append(teamPartyScores, &types.PartyContributionScore{Party: party, Score: score, IsEligible: eligible, StakingBalance: balance, OpenVolume: notional, TotalFeesPaid: paidFees[party], RankingIndex: -1})
			eligibleTeamPartyScores = append(eligibleTeamPartyScores, &types.PartyContributionScore{Party: party, Score: score, IsEligible: eligible, StakingBalance: balance, OpenVolume: notional, TotalFeesPaid: paidFees[party], RankingIndex: -1})
		} else {
			teamPartyScores = append(teamPartyScores, &types.PartyContributionScore{Party: party, Score: num.DecimalZero(), IsEligible: false, StakingBalance: balance, OpenVolume: notional, TotalFeesPaid: paidFees[party], RankingIndex: -1})
		}
	}

	if len(teamPartyScores) == 0 {
		return num.DecimalZero(), []*types.PartyContributionScore{}
	}

	sort.Slice(eligibleTeamPartyScores, func(i, j int) bool {
		return eligibleTeamPartyScores[i].Score.GreaterThan(eligibleTeamPartyScores[j].Score)
	})

	sort.Slice(teamPartyScores, func(i, j int) bool {
		return teamPartyScores[i].Score.GreaterThan(teamPartyScores[j].Score)
	})

	lastUsed := int64(1)
	for _, tps := range teamPartyScores {
		if tps.IsEligible {
			tps.RankingIndex = lastUsed
			lastUsed += 1
		}
	}

	maxIndex := int(topN.Mul(num.DecimalFromInt64(int64(len(parties)))).IntPart())
	// ensure non-zero, otherwise we have a divide-by-zero panic on our hands
	if maxIndex == 0 {
		maxIndex = 1
	}
	if len(eligibleTeamPartyScores) < maxIndex {
		maxIndex = len(eligibleTeamPartyScores)
	}
	if maxIndex == 0 {
		return num.DecimalZero(), teamPartyScores
	}

	total := num.DecimalZero()
	for i := 0; i < maxIndex; i++ {
		total = total.Add(eligibleTeamPartyScores[i].Score)
	}

	return total.Div(num.DecimalFromInt64(int64(maxIndex))), teamPartyScores
}

// calculateMetricForParty returns the value of a reward metric score for the given party for markets of the given assets which are in scope over the given window size.
func (mat *MarketActivityTracker) calculateMetricForParty(asset, party string, marketsInScope []string, metric vega.DispatchMetric, windowSize int) (num.Decimal, bool) {
	// exclude unsupported metrics
	if metric == vega.DispatchMetric_DISPATCH_METRIC_MARKET_VALUE {
		mat.log.Panic("unexpected dispatch metric market value here")
	}
	if metric == vega.DispatchMetric_DISPATCH_METRIC_VALIDATOR_RANKING {
		mat.log.Panic("unexpected dispatch metric validator ranking here")
	}
	total := num.DecimalZero()
	marketTotal := num.DecimalZero()
	returns := make([]*num.Decimal, windowSize)
	found := false

	assetTrackers, ok := mat.assetToMarketTrackers[asset]
	if !ok {
		return num.DecimalZero(), false
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
		if marketTracker == nil {
			continue
		}
		switch metric {
		case vega.DispatchMetric_DISPATCH_METRIC_AVERAGE_NOTIONAL:
			if t, ok := marketTracker.getNotionalMetricTotal(party, windowSize); ok {
				found = true
				total = total.Add(t)
			}
		case vega.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN:
			if t, ok := marketTracker.getRelativeReturnMetricTotal(party, windowSize); ok {
				found = true
				total = total.Add(t)
			}
		case vega.DispatchMetric_DISPATCH_METRIC_REALISED_RETURN:
			if t, ok := marketTracker.getRealisedReturnMetricTotal(party, windowSize); ok {
				found = true
				total = total.Add(t)
			}
		case vega.DispatchMetric_DISPATCH_METRIC_RETURN_VOLATILITY:
			r, ok := marketTracker.getReturns(party, windowSize)
			if !ok {
				continue
			}
			found = true
			for i, ret := range r {
				if ret != nil {
					if returns[i] != nil {
						*returns[i] = returns[i].Add(*ret)
					} else {
						returns[i] = ret
					}
				}
			}
		case vega.DispatchMetric_DISPATCH_METRIC_MAKER_FEES_PAID:
			if t, ok := getFees(marketTracker.epochMakerFeesPaid, party, windowSize); ok {
				if t.IsPositive() {
					found = true
				}
				total = total.Add(t)
			}
			marketTotal = marketTotal.Add(getTotalFees(marketTracker.epochTotalMakerFeesPaid, windowSize))
		case vega.DispatchMetric_DISPATCH_METRIC_MAKER_FEES_RECEIVED:
			if t, ok := getFees(marketTracker.epochMakerFeesReceived, party, windowSize); ok {
				if t.IsPositive() {
					found = true
				}
				total = total.Add(t)
			}
			marketTotal = marketTotal.Add(getTotalFees(marketTracker.epochTotalMakerFeesReceived, windowSize))
		case vega.DispatchMetric_DISPATCH_METRIC_LP_FEES_RECEIVED:
			if t, ok := getFees(marketTracker.epochLpFees, party, windowSize); ok {
				if t.IsPositive() {
					found = true
				}
				total = total.Add(t)
			}
			marketTotal = marketTotal.Add(getTotalFees(marketTracker.epochTotalLpFees, windowSize))
		}
	}

	switch metric {
	case vega.DispatchMetric_DISPATCH_METRIC_AVERAGE_NOTIONAL:
		// descaling the total tw position metric by dividing by the scaling factor
		v := total.Div(num.DecimalFromInt64(int64(windowSize) * scalingFactor))
		return v, found
	case vega.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, vega.DispatchMetric_DISPATCH_METRIC_REALISED_RETURN:
		return total.Div(num.DecimalFromInt64(int64(windowSize))), found
	case vega.DispatchMetric_DISPATCH_METRIC_RETURN_VOLATILITY:
		filteredReturns := []num.Decimal{}
		for _, d := range returns {
			if d != nil {
				filteredReturns = append(filteredReturns, *d)
			}
		}
		if len(filteredReturns) < 2 {
			return num.DecimalZero(), false
		}
		variance, _ := num.Variance(filteredReturns)
		if !variance.IsZero() {
			return num.DecimalOne().Div(variance), found
		}
		return variance, found
	case vega.DispatchMetric_DISPATCH_METRIC_MAKER_FEES_PAID, vega.DispatchMetric_DISPATCH_METRIC_MAKER_FEES_RECEIVED, vega.DispatchMetric_DISPATCH_METRIC_LP_FEES_RECEIVED:
		if marketTotal.IsZero() {
			return num.DecimalZero(), found
		}
		return total.Div(marketTotal), found
	default:
		mat.log.Panic("unexpected metric")
	}
	return num.DecimalZero(), found
}

func (mat *MarketActivityTracker) RecordNotionalTraded(asset, marketID string, notional *num.Uint) {
	if tracker, ok := mat.getMarketTracker(asset, marketID); ok {
		tracker.notionalVolumeForEpoch.AddSum(notional)
	}
}

func (mat *MarketActivityTracker) RecordNotionalTakerVolume(marketID string, party string, volumeToAdd *num.Uint) {
	if _, ok := mat.partyTakerNotionalVolume[party]; !ok {
		mat.partyTakerNotionalVolume[party] = volumeToAdd
	} else {
		mat.partyTakerNotionalVolume[party].AddSum(volumeToAdd)
	}

	if _, ok := mat.marketToPartyTakerNotionalVolume[marketID]; !ok {
		mat.marketToPartyTakerNotionalVolume[marketID] = map[string]*num.Uint{
			party: volumeToAdd.Clone(),
		}
	} else if _, ok := mat.marketToPartyTakerNotionalVolume[marketID][party]; !ok {
		mat.marketToPartyTakerNotionalVolume[marketID][party] = volumeToAdd.Clone()
	} else {
		mat.marketToPartyTakerNotionalVolume[marketID][party].AddSum(volumeToAdd)
	}
}

func (mat *MarketActivityTracker) clearNotionalTakerVolume() {
	mat.partyTakerNotionalVolume = map[string]*num.Uint{}
	mat.marketToPartyTakerNotionalVolume = map[string]map[string]*num.Uint{}
}

func (mat *MarketActivityTracker) NotionalTakerVolumeForAllParties() map[types.PartyID]*num.Uint {
	res := make(map[types.PartyID]*num.Uint, len(mat.partyTakerNotionalVolume))
	for k, u := range mat.partyTakerNotionalVolume {
		res[types.PartyID(k)] = u.Clone()
	}
	return res
}

func (mat *MarketActivityTracker) TeamStatsForMarkets(allMarketsForAssets, onlyTheseMarkets []string) map[string]map[string]*num.Uint {
	teams := mat.teams.GetAllTeamsWithParties(0)

	// Pre-fill stats for all teams and their members.
	partyToTeam := map[string]string{}
	teamsStats := map[string]map[string]*num.Uint{}
	for teamID, members := range teams {
		teamsStats[teamID] = map[string]*num.Uint{}
		for _, member := range members {
			teamsStats[teamID][member] = num.UintZero()
			partyToTeam[member] = teamID
		}
	}

	// Filter the markets to get data from.
	onlyMarketsStats := map[string]map[string]*num.Uint{}
	if len(onlyTheseMarkets) == 0 {
		onlyMarketsStats = mat.marketToPartyTakerNotionalVolume
	} else {
		for _, marketID := range onlyTheseMarkets {
			onlyMarketsStats[marketID] = mat.marketToPartyTakerNotionalVolume[marketID]
		}
	}

	for _, asset := range allMarketsForAssets {
		mkts, ok := mat.assetToMarketTrackers[asset]
		if !ok {
			continue
		}
		for marketID := range mkts {
			onlyMarketsStats[marketID] = mat.marketToPartyTakerNotionalVolume[marketID]
		}
	}

	// Gather only party's stats from those who are in a team.
	for _, marketStats := range onlyMarketsStats {
		for partyID, volume := range marketStats {
			teamID, inTeam := partyToTeam[partyID]
			if !inTeam {
				continue
			}
			teamsStats[teamID][partyID].AddSum(volume)
		}
	}

	return teamsStats
}

func (mat *MarketActivityTracker) NotionalTakerVolumeForParty(party string) *num.Uint {
	if _, ok := mat.partyTakerNotionalVolume[party]; !ok {
		return num.UintZero()
	}
	return mat.partyTakerNotionalVolume[party].Clone()
}

func updateNotionalOnTrade(n *twNotional, notional, price *num.Uint, t, tn int64, time time.Time) {
	tnOverT := num.UintZero()
	tnOverTComp := uScalingFactor.Clone()
	if t != 0 {
		tnOverT = num.NewUint(uint64(tn / t))
		tnOverTComp = tnOverTComp.Sub(tnOverTComp, tnOverT)
	}
	p1 := num.UintZero().Mul(n.currentEpochTWNotional, tnOverTComp)
	p2 := num.UintZero().Mul(n.notional, tnOverT)
	n.currentEpochTWNotional = num.UintZero().Div(p1.AddSum(p2), uScalingFactor)
	n.notional = notional
	n.price = price.Clone()
	n.t = time
}

func updateNotionalOnEpochEnd(n *twNotional, notional, price *num.Uint, t, tn int64, time time.Time) {
	tnOverT := num.UintZero()
	tnOverTComp := uScalingFactor.Clone()
	if t != 0 {
		tnOverT = num.NewUint(uint64(tn / t))
		tnOverTComp = tnOverTComp.Sub(tnOverTComp, tnOverT)
	}
	p1 := num.UintZero().Mul(n.currentEpochTWNotional, tnOverTComp)
	p2 := num.UintZero().Mul(notional, tnOverT)
	n.currentEpochTWNotional = num.UintZero().Div(p1.AddSum(p2), uScalingFactor)
	n.notional = notional
	if price != nil && !price.IsZero() {
		n.price = price.Clone()
	}
	n.t = time
}

func calcNotionalAt(n *twNotional, t, tn int64, markPrice *num.Uint) *num.Uint {
	tnOverT := num.UintZero()
	tnOverTComp := uScalingFactor.Clone()
	if t != 0 {
		tnOverT = num.NewUint(uint64(tn / t))
		tnOverTComp = tnOverTComp.Sub(tnOverTComp, tnOverT)
	}
	p1 := num.UintZero().Mul(n.currentEpochTWNotional, tnOverTComp)
	var notional *num.Uint
	if markPrice != nil && !markPrice.IsZero() {
		notional, _ = num.UintFromDecimal(n.notional.ToDecimal().Div(n.price.ToDecimal()).Mul(markPrice.ToDecimal()))
	} else {
		notional = n.notional
	}
	p2 := num.UintZero().Mul(notional, tnOverT)
	return num.UintZero().Div(p1.AddSum(p2), uScalingFactor)
}

// recordNotional tracks the time weighted average notional for the party per market.
// notional = abs(position) x price / position_factor
// price in asset decimals.
func (mt *marketTracker) recordNotional(party string, notional *num.Uint, price *num.Uint, time time.Time, epochStartTime time.Time) {
	if _, ok := mt.twNotional[party]; !ok {
		mt.twNotional[party] = &twNotional{
			t:                      time,
			notional:               notional,
			currentEpochTWNotional: num.UintZero(),
			price:                  price.Clone(),
		}
		return
	}
	t := int64(time.Sub(epochStartTime).Seconds())
	n := mt.twNotional[party]
	tn := int64(time.Sub(n.t).Seconds()) * scalingFactor
	updateNotionalOnTrade(n, notional, price, t, tn, time)
}

func (mt *marketTracker) processNotionalEndOfEpoch(epochStartTime time.Time, endEpochTime time.Time) {
	t := int64(endEpochTime.Sub(epochStartTime).Seconds())
	m := make(map[string]*num.Uint, len(mt.twNotional))
	for party, twNotional := range mt.twNotional {
		tn := int64(endEpochTime.Sub(twNotional.t).Seconds()) * scalingFactor
		var notional *num.Uint
		if mt.markPrice != nil && !mt.markPrice.IsZero() {
			notional, _ = num.UintFromDecimal(twNotional.notional.ToDecimal().Div(twNotional.price.ToDecimal()).Mul(mt.markPrice.ToDecimal()))
		} else {
			notional = twNotional.notional
		}
		updateNotionalOnEpochEnd(twNotional, notional, mt.markPrice, t, tn, endEpochTime)
		m[party] = twNotional.currentEpochTWNotional.Clone()
	}
	if len(mt.epochTimeWeightedNotional) == maxWindowSize {
		mt.epochTimeWeightedNotional = mt.epochTimeWeightedNotional[1:]
	}
	mt.epochTimeWeightedNotional = append(mt.epochTimeWeightedNotional, m)
	for p, twp := range mt.twNotional {
		// if the notional at the beginning of the epoch is 0 clear it so we don't keep zero notionals`` forever
		if twp.currentEpochTWNotional.IsZero() && twp.notional.IsZero() {
			delete(mt.twNotional, p)
		}
	}
}

func (mt *marketTracker) processNotionalAtMilestone(epochStartTime time.Time, milestoneTime time.Time) {
	t := int64(milestoneTime.Sub(epochStartTime).Seconds())
	m := make(map[string]*num.Uint, len(mt.twNotional))
	for party, twNotional := range mt.twNotional {
		tn := int64(milestoneTime.Sub(twNotional.t).Seconds()) * scalingFactor
		m[party] = calcNotionalAt(twNotional, t, tn, mt.markPrice)
	}
	mt.epochTimeWeightedNotional = append(mt.epochTimeWeightedNotional, m)
}

func (mat *MarketActivityTracker) getTWNotionalPosition(asset, party string, markets []string) *num.Uint {
	total := num.UintZero()
	mkts := markets
	if len(mkts) == 0 {
		mkts = make([]string, 0, len(mat.assetToMarketTrackers[asset]))
		for k := range mat.assetToMarketTrackers[asset] {
			mkts = append(mkts, k)
		}
		sort.Strings(mkts)
	}

	for _, mkt := range mkts {
		if tracker, ok := mat.getMarketTracker(asset, mkt); ok {
			if len(tracker.epochTimeWeightedNotional) <= 0 {
				continue
			}
			if twNotional, ok := tracker.epochTimeWeightedNotional[len(tracker.epochTimeWeightedNotional)-1][party]; ok {
				total.AddSum(twNotional)
			}
		}
	}
	return total
}

func updatePosition(toi *twPosition, scaledAbsPos uint64, t, tn int64, time time.Time) {
	tnOverT := uint64(0)
	if t != 0 {
		tnOverT = uint64(tn / t)
	}
	toi.currentEpochTWPosition = (toi.currentEpochTWPosition*(u64ScalingFactor-tnOverT) + (toi.position * tnOverT)) / u64ScalingFactor
	toi.position = scaledAbsPos
	toi.t = time
}

func calculatePositionAt(toi *twPosition, t, tn int64) uint64 {
	tnOverT := uint64(0)
	if t != 0 {
		tnOverT = uint64(tn / t)
	}
	return (toi.currentEpochTWPosition*(u64ScalingFactor-tnOverT) + (toi.position * tnOverT)) / u64ScalingFactor
}

// recordPosition records the current position of a party and the time of change. If there is a previous position then it is time weight updated with respect to the time
// it has been in place during the epoch.
func (mt *marketTracker) recordPosition(party string, absPos uint64, positionFactor num.Decimal, time time.Time, epochStartTime time.Time) {
	if party == "network" {
		return
	}
	// scale by scaling factor and divide by position factor
	// by design the scaling factor is greater than the max position factor which allows no loss of precision
	scaledAbsPos := num.UintZero().Mul(num.NewUint(absPos), uScalingFactor).ToDecimal().Div(positionFactor).IntPart()
	if _, ok := mt.twPosition[party]; !ok {
		mt.twPosition[party] = &twPosition{
			position:               uint64(scaledAbsPos),
			t:                      time,
			currentEpochTWPosition: 0,
		}
		return
	}
	toi := mt.twPosition[party]
	t := int64(time.Sub(epochStartTime).Seconds())
	tn := int64(time.Sub(toi.t).Seconds()) * scalingFactor

	updatePosition(toi, uint64(scaledAbsPos), t, tn, time)
}

// processPositionEndOfEpoch is called at the end of the epoch, calculates the time weight of the current position and moves it to the next epoch, and records
// the time weighted position of the current epoch in the history.
func (mt *marketTracker) processPositionEndOfEpoch(epochStartTime time.Time, endEpochTime time.Time) {
	t := int64(endEpochTime.Sub(epochStartTime).Seconds())
	m := make(map[string]uint64, len(mt.twPosition))
	for party, toi := range mt.twPosition {
		tn := int64(endEpochTime.Sub(toi.t).Seconds()) * scalingFactor
		updatePosition(toi, toi.position, t, tn, endEpochTime)
		m[party] = toi.currentEpochTWPosition
	}

	if len(mt.epochTimeWeightedPosition) == maxWindowSize {
		mt.epochTimeWeightedPosition = mt.epochTimeWeightedPosition[1:]
	}
	mt.epochTimeWeightedPosition = append(mt.epochTimeWeightedPosition, m)
	for p, twp := range mt.twPosition {
		// if the position at the beginning of the epoch is 0 clear it so we don't keep zero positions forever
		if twp.currentEpochTWPosition == 0 && twp.position == 0 {
			delete(mt.twPosition, p)
		}
	}
}

func (mt *marketTracker) processPositionAtMilestone(epochStartTime time.Time, milestoneTime time.Time) {
	t := int64(milestoneTime.Sub(epochStartTime).Seconds())
	m := make(map[string]uint64, len(mt.twPosition))
	for party, toi := range mt.twPosition {
		tn := int64(milestoneTime.Sub(toi.t).Seconds()) * scalingFactor
		m[party] = calculatePositionAt(toi, t, tn)
	}
	mt.epochTimeWeightedPosition = append(mt.epochTimeWeightedPosition, m)
}

// //// return metric //////

// recordM2M records the amount corresponding to mark to market (profit or loss).
func (mt *marketTracker) recordM2M(party string, amount num.Decimal) {
	if party == "network" || amount.IsZero() {
		return
	}
	if _, ok := mt.partyM2M[party]; !ok {
		mt.partyM2M[party] = amount
		return
	}
	mt.partyM2M[party] = mt.partyM2M[party].Add(amount)
}

func (mt *marketTracker) recordFundingPayment(party string, amount num.Decimal) {
	if party == "network" || amount.IsZero() {
		return
	}
	if _, ok := mt.partyRealisedReturn[party]; !ok {
		mt.partyRealisedReturn[party] = amount
		return
	}
	mt.partyRealisedReturn[party] = mt.partyRealisedReturn[party].Add(amount)
}

func (mt *marketTracker) recordRealisedPosition(party string, realisedPosition num.Decimal) {
	if party == "network" {
		return
	}
	if _, ok := mt.partyRealisedReturn[party]; !ok {
		mt.partyRealisedReturn[party] = realisedPosition
		return
	}
	mt.partyRealisedReturn[party] = mt.partyRealisedReturn[party].Add(realisedPosition)
}

// processM2MEndOfEpoch is called at the end of the epoch to reset the running total for the next epoch and record the total m2m in the ended epoch.
func (mt *marketTracker) processM2MEndOfEpoch() {
	m := map[string]num.Decimal{}
	for party, m2m := range mt.partyM2M {
		if _, ok := mt.twPosition[party]; !ok {
			continue
		}
		p := mt.twPosition[party].currentEpochTWPosition
		var v num.Decimal
		if p == 0 {
			v = num.DecimalZero()
		} else {
			v = m2m.Div(num.DecimalFromInt64(int64(p)).Div(dScalingFactor))
		}
		m[party] = v
	}
	if len(mt.epochPartyM2M) == maxWindowSize {
		mt.epochPartyM2M = mt.epochPartyM2M[1:]
	}
	mt.partyM2M = map[string]decimal.Decimal{}
	mt.epochPartyM2M = append(mt.epochPartyM2M, m)
}

func (mt *marketTracker) processM2MAtMilestone() {
	m := map[string]num.Decimal{}
	for party, m2m := range mt.partyM2M {
		if _, ok := mt.twPosition[party]; !ok {
			continue
		}
		p := mt.epochTimeWeightedPosition[len(mt.epochTimeWeightedPosition)-1][party]

		var v num.Decimal
		if p == 0 {
			v = num.DecimalZero()
		} else {
			v = m2m.Div(num.DecimalFromInt64(int64(p)).Div(dScalingFactor))
		}
		m[party] = v
	}
	mt.epochPartyM2M = append(mt.epochPartyM2M, m)
}

// processPartyRealisedReturnOfEpoch is called at the end of the epoch to reset the running total for the next epoch and record the total m2m in the ended epoch.
func (mt *marketTracker) processPartyRealisedReturnOfEpoch() {
	m := map[string]num.Decimal{}
	for party, realised := range mt.partyRealisedReturn {
		m[party] = realised
	}
	if len(mt.epochPartyRealisedReturn) == maxWindowSize {
		mt.epochPartyRealisedReturn = mt.epochPartyRealisedReturn[1:]
	}
	mt.epochPartyRealisedReturn = append(mt.epochPartyRealisedReturn, m)
	mt.partyRealisedReturn = map[string]decimal.Decimal{}
}

func (mt *marketTracker) processPartyRealisedReturnAtMilestone() {
	m := map[string]num.Decimal{}
	for party, realised := range mt.partyRealisedReturn {
		m[party] = realised
	}
	mt.epochPartyRealisedReturn = append(mt.epochPartyRealisedReturn, m)
}

// getReturns returns a slice of the total of the party's return by epoch in the given window.
func (mt *marketTracker) getReturns(party string, windowSize int) ([]*num.Decimal, bool) {
	returns := make([]*num.Decimal, windowSize)
	if len(mt.epochPartyM2M) == 0 {
		return []*num.Decimal{}, false
	}
	found := false
	for i := 0; i < windowSize; i++ {
		ind := len(mt.epochPartyM2M) - i - 1
		if ind < 0 {
			break
		}
		epochData := mt.epochPartyM2M[ind]
		if t, ok := epochData[party]; ok {
			found = true
			returns[i] = &t
		}
	}
	return returns, found
}

// getNotionalMetricTotal returns the sum of the epoch's time weighted notional over the time window.
func (mt *marketTracker) getNotionalMetricTotal(party string, windowSize int) (num.Decimal, bool) {
	return calcTotalForWindowU(party, mt.epochTimeWeightedNotional, windowSize)
}

// getPositionMetricTotal returns the sum of the epoch's time weighted position over the time window.
func (mt *marketTracker) getPositionMetricTotal(party string, windowSize int) (uint64, bool) {
	return calcTotalForWindowUint64(party, mt.epochTimeWeightedPosition, windowSize)
}

// getRelativeReturnMetricTotal returns the sum of the relative returns over the given window.
func (mt *marketTracker) getRelativeReturnMetricTotal(party string, windowSize int) (num.Decimal, bool) {
	return calcTotalForWindowD(party, mt.epochPartyM2M, windowSize)
}

// getRealisedReturnMetricTotal returns the sum of the relative returns over the given window.
func (mt *marketTracker) getRealisedReturnMetricTotal(party string, windowSize int) (num.Decimal, bool) {
	return calcTotalForWindowD(party, mt.epochPartyRealisedReturn, windowSize)
}

// getFees returns the total fees paid/received (depending on what feeData represents) by the party over the given window size.
func getFees(feeData []map[string]*num.Uint, party string, windowSize int) (num.Decimal, bool) {
	return calcTotalForWindowU(party, feeData, windowSize)
}

// getTotalFees returns the total fees of the given type measured over the window size.
func getTotalFees(totalFees []*num.Uint, windowSize int) num.Decimal {
	if len(totalFees) == 0 {
		return num.DecimalZero()
	}
	total := num.UintZero()
	for i := 0; i < windowSize; i++ {
		ind := len(totalFees) - i - 1
		if ind < 0 {
			return total.ToDecimal()
		}
		total.AddSum(totalFees[ind])
	}
	return total.ToDecimal()
}

func (mat *MarketActivityTracker) getEpochTakeFees(asset string, markets []string, takerFeesPaidInEpoch map[string]map[string]map[string]*num.Uint) map[string]*num.Uint {
	takerFees := map[string]*num.Uint{}
	ast, ok := takerFeesPaidInEpoch[asset]
	if !ok {
		return takerFees
	}
	mkts := markets
	if len(mkts) == 0 {
		mkts = make([]string, 0, len(ast))
		for mkt := range ast {
			mkts = append(mkts, mkt)
		}
	}

	for _, m := range mkts {
		if fees, ok := ast[m]; ok {
			for party, fees := range fees {
				if _, ok := takerFees[party]; !ok {
					takerFees[party] = num.UintZero()
				}
				takerFees[party].AddSum(fees)
			}
		}
	}
	return takerFees
}

func (mat *MarketActivityTracker) GetLastEpochTakeFees(asset string, markets []string, epochs int32) map[string]*num.Uint {
	takerFees := map[string]*num.Uint{}
	for i := 0; i < int(epochs); i++ {
		ind := len(mat.takerFeesPaidInEpoch) - i - 1
		if ind < 0 {
			break
		}
		m := mat.getEpochTakeFees(asset, markets, mat.takerFeesPaidInEpoch[ind])
		for k, v := range m {
			if _, ok := takerFees[k]; !ok {
				takerFees[k] = num.UintZero()
			}
			takerFees[k].AddSum(v)
		}
	}
	return takerFees
}

// calcTotalForWindowU returns the total relevant data from the given slice starting from the given dataIdx-1, going back <window_size> elements.
func calcTotalForWindowU(party string, data []map[string]*num.Uint, windowSize int) (num.Decimal, bool) {
	found := false
	if len(data) == 0 {
		return num.DecimalZero(), found
	}
	total := num.UintZero()
	for i := 0; i < windowSize; i++ {
		ind := len(data) - i - 1
		if ind < 0 {
			return total.ToDecimal(), found
		}
		if v, ok := data[ind][party]; ok {
			found = true
			total.AddSum(v)
		}
	}
	return total.ToDecimal(), found
}

// calcTotalForWindowD returns the total relevant data from the given slice starting from the given dataIdx-1, going back <window_size> elements.
func calcTotalForWindowD(party string, data []map[string]num.Decimal, windowSize int) (num.Decimal, bool) {
	found := false
	if len(data) == 0 {
		return num.DecimalZero(), found
	}
	total := num.DecimalZero()
	for i := 0; i < windowSize; i++ {
		ind := len(data) - i - 1
		if ind < 0 {
			return total, found
		}
		if v, ok := data[ind][party]; ok {
			found = true
			total = total.Add(v)
		}
	}
	return total, found
}

// calcTotalForWindowUint64 returns the total relevant data from the given slice starting from the given dataIdx-1, going back <window_size> elements.
func calcTotalForWindowUint64(party string, data []map[string]uint64, windowSize int) (uint64, bool) {
	found := false
	if len(data) == 0 {
		return 0, found
	}

	total := uint64(0)
	for i := 0; i < windowSize; i++ {
		ind := len(data) - i - 1
		if ind < 0 {
			return total, found
		}
		if v, ok := data[ind][party]; ok {
			found = true
			total += v
		}
	}
	return total, found
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
