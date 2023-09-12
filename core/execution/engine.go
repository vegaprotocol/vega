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

package execution

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/execution/common"
	"code.vegaprotocol.io/vega/core/execution/future"
	"code.vegaprotocol.io/vega/core/execution/spot"
	"code.vegaprotocol.io/vega/core/fee"
	"code.vegaprotocol.io/vega/core/metrics"
	"code.vegaprotocol.io/vega/core/monitor"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/protos/vega"
)

var (
	// ErrMarketDoesNotExist is returned when the market does not exist.
	ErrMarketDoesNotExist = errors.New("market does not exist")

	// ErrNoMarketID is returned when invalid (empty) market id was supplied during market creation.
	ErrNoMarketID = errors.New("no valid market id was supplied")

	// ErrInvalidOrderCancellation is returned when an incomplete order cancellation request is used.
	ErrInvalidOrderCancellation = errors.New("invalid order cancellation")

	// ErrSuccessorMarketDoesNotExists is returned when SucceedMarket call is made with an invalid successor market ID.
	ErrSuccessorMarketDoesNotExist = errors.New("successor market does not exist")

	// ErrParentMarketNotEnactedYEt is returned when trying to enact a successor market that is still in proposed state.
	ErrParentMarketNotEnactedYet = errors.New("parent market in proposed state, can't enact successor")

	// ErrInvalidStopOrdersCancellation is returned when an incomplete stop orders cancellation request is used.
	ErrInvalidStopOrdersCancellation = errors.New("invalid stop orders cancellation")

	// ErrMarketIDRequiredWhenOrderIDSpecified is returned when a stop order cancellation is emitted without an order id.
	ErrMarketIDRequiredWhenOrderIDSpecified = errors.New("market id required when order id specified")
)

// Engine is the execution engine.
type Engine struct {
	Config
	log *logging.Logger

	futureMarkets    map[string]*future.Market
	futureMarketsCpy []*future.Market

	spotMarkets    map[string]*spot.Market
	spotMarketsCpy []*spot.Market

	allMarkets    map[string]common.CommonMarket
	allMarketsCpy []common.CommonMarket

	collateral                    common.Collateral
	assets                        common.Assets
	referralDiscountRewardService fee.ReferralDiscountRewardService
	volumeDiscountService         fee.VolumeDiscountService

	broker                common.Broker
	timeService           common.TimeService
	stateVarEngine        common.StateVarEngine
	marketActivityTracker *common.MarketActivityTracker

	oracle common.OracleEngine

	npv netParamsValues

	snapshotSerialised    []byte
	newGeneratedProviders []types.StateProvider // new providers generated during the last state change

	// Map of all active snapshot providers that the execution engine has generated
	generatedProviders map[string]struct{}

	maxPeggedOrders        uint64
	totalPeggedOrdersCount int64

	marketCPStates map[string]*types.CPMarketState
	// a map of all successor markets under parent ID
	// used to manage pending markets once a successor takes over
	successors      map[string][]string
	isSuccessor     map[string]string
	successorWindow time.Duration
	// only used once, during CP restore, this doesn't need to be included in a snapshot or checkpoint.
	skipRestoreSuccessors map[string]struct{}
}

type netParamsValues struct {
	feeDistributionTimeStep              time.Duration
	marketValueWindowLength              time.Duration
	suppliedStakeToObligationFactor      num.Decimal
	infrastructureFee                    num.Decimal
	makerFee                             num.Decimal
	scalingFactors                       *types.ScalingFactors
	maxLiquidityFee                      num.Decimal
	bondPenaltyFactor                    num.Decimal
	auctionMinDuration                   time.Duration
	probabilityOfTradingTauScaling       num.Decimal
	minProbabilityOfTradingLPOrders      num.Decimal
	minLpStakeQuantumMultiple            num.Decimal
	marketCreationQuantumMultiple        num.Decimal
	markPriceUpdateMaximumFrequency      time.Duration
	marketPartiesMaximumStopOrdersUpdate *num.Uint

	// Liquidity version 2.
	liquidityV2BondPenaltyFactor                 num.Decimal
	liquidityV2EarlyExitPenalty                  num.Decimal
	liquidityV2MaxLiquidityFee                   num.Decimal
	liquidityV2SLANonPerformanceBondPenaltyMax   num.Decimal
	liquidityV2SLANonPerformanceBondPenaltySlope num.Decimal
	liquidityV2StakeToCCYVolume                  num.Decimal
	liquidityV2ProvidersFeeCalculationTimeStep   time.Duration
}

func defaultNetParamsValues() netParamsValues {
	return netParamsValues{
		feeDistributionTimeStep:         -1,
		marketValueWindowLength:         -1,
		suppliedStakeToObligationFactor: num.DecimalFromInt64(-1),
		infrastructureFee:               num.DecimalFromInt64(-1),
		makerFee:                        num.DecimalFromInt64(-1),
		scalingFactors:                  nil,
		maxLiquidityFee:                 num.DecimalFromInt64(-1),
		bondPenaltyFactor:               num.DecimalFromInt64(-1),

		auctionMinDuration:                   -1,
		probabilityOfTradingTauScaling:       num.DecimalFromInt64(-1),
		minProbabilityOfTradingLPOrders:      num.DecimalFromInt64(-1),
		minLpStakeQuantumMultiple:            num.DecimalFromInt64(-1),
		marketCreationQuantumMultiple:        num.DecimalFromInt64(-1),
		markPriceUpdateMaximumFrequency:      5 * time.Second, // default is 5 seconds, should come from net params though
		marketPartiesMaximumStopOrdersUpdate: num.UintZero(),

		// Liquidity version 2.
		liquidityV2BondPenaltyFactor:                 num.DecimalFromInt64(-1),
		liquidityV2EarlyExitPenalty:                  num.DecimalFromInt64(-1),
		liquidityV2MaxLiquidityFee:                   num.DecimalFromInt64(-1),
		liquidityV2SLANonPerformanceBondPenaltyMax:   num.DecimalFromInt64(-1),
		liquidityV2SLANonPerformanceBondPenaltySlope: num.DecimalFromInt64(-1),
		liquidityV2StakeToCCYVolume:                  num.DecimalFromInt64(-1),
		liquidityV2ProvidersFeeCalculationTimeStep:   time.Second * 5,
	}
}

// NewEngine takes stores and engines and returns
// a new execution engine to process new orders, etc.
func NewEngine(
	log *logging.Logger,
	executionConfig Config,
	ts common.TimeService,
	collateral common.Collateral,
	oracle common.OracleEngine,
	broker common.Broker,
	stateVarEngine common.StateVarEngine,
	marketActivityTracker *common.MarketActivityTracker,
	assets common.Assets,
	referralDiscountRewardService fee.ReferralDiscountRewardService,
	volumeDiscountService fee.VolumeDiscountService,
) *Engine {
	// setup logger
	log = log.Named(namedLogger)
	log.SetLevel(executionConfig.Level.Get())
	e := &Engine{
		log:                           log,
		Config:                        executionConfig,
		futureMarkets:                 map[string]*future.Market{},
		spotMarkets:                   map[string]*spot.Market{},
		allMarkets:                    map[string]common.CommonMarket{},
		timeService:                   ts,
		collateral:                    collateral,
		assets:                        assets,
		broker:                        broker,
		oracle:                        oracle,
		npv:                           defaultNetParamsValues(),
		generatedProviders:            map[string]struct{}{},
		stateVarEngine:                stateVarEngine,
		marketActivityTracker:         marketActivityTracker,
		marketCPStates:                map[string]*types.CPMarketState{},
		successors:                    map[string][]string{},
		isSuccessor:                   map[string]string{},
		skipRestoreSuccessors:         map[string]struct{}{},
		referralDiscountRewardService: referralDiscountRewardService,
		volumeDiscountService:         volumeDiscountService,
	}

	// set the eligibility for proposer bonus checker
	e.marketActivityTracker.SetEligibilityChecker(e)

	return e
}

// ReloadConf updates the internal configuration of the execution
// engine and its dependencies.
func (e *Engine) ReloadConf(cfg Config) {
	e.log.Debug("reloading configuration")

	if e.log.GetLevel() != cfg.Level.Get() {
		e.log.Info("updating log level",
			logging.String("old", e.log.GetLevel().String()),
			logging.String("new", cfg.Level.String()),
		)
		e.log.SetLevel(cfg.Level.Get())
	}

	e.Config = cfg
	for _, mkt := range e.futureMarketsCpy {
		mkt.ReloadConf(e.Matching, e.Risk, e.Position, e.Settlement, e.Fee)
	}

	for _, mkt := range e.spotMarketsCpy {
		mkt.ReloadConf(e.Matching, e.Fee)
	}
}

func (e *Engine) OnEpochEvent(ctx context.Context, epoch types.Epoch) {
	for _, m := range e.allMarketsCpy {
		m.OnEpochEvent(ctx, epoch)
	}
}

func (e *Engine) OnEpochRestore(ctx context.Context, epoch types.Epoch) {
	for _, m := range e.allMarketsCpy {
		m.OnEpochRestore(ctx, epoch)
	}
}

func (e *Engine) Hash() []byte {
	e.log.Debug("hashing markets")

	hashes := make([]string, 0, len(e.allMarketsCpy))

	for _, m := range e.allMarketsCpy {
		hash := m.Hash()
		e.log.Debug("market app state hash", logging.Hash(hash), logging.String("market-id", m.GetID()))
		hashes = append(hashes, string(hash))
	}
	sort.Strings(hashes)

	// get the accounts hash + add it at end of all markets hash
	accountsHash := e.collateral.Hash()
	e.log.Debug("accounts state hash", logging.Hash(accountsHash))

	bytes := []byte{}
	for _, h := range append(hashes, string(accountsHash)) {
		bytes = append(bytes, []byte(h)...)
	}
	return crypto.Hash(bytes)
}

// RejectMarket will stop the execution of the market
// and refund into the general account any funds in margins accounts from any parties
// This works only if the market is in a PROPOSED STATE.
func (e *Engine) RejectMarket(ctx context.Context, marketID string) error {
	if e.log.IsDebug() {
		e.log.Debug("reject market", logging.MarketID(marketID))
	}

	_, isFuture := e.futureMarkets[marketID]
	if _, ok := e.allMarkets[marketID]; !ok {
		return ErrMarketDoesNotExist
	}
	mkt := e.allMarkets[marketID]
	if err := mkt.Reject(ctx); err != nil {
		return err
	}

	// send market data event so market data and markets API are consistent.
	e.broker.Send(events.NewMarketDataEvent(ctx, mkt.GetMarketData()))
	e.removeMarket(marketID)

	if !isFuture {
		return nil
	}

	// a market rejection can have a knock-on effect for proposed markets which were supposed to succeed this market
	// they should be purged here, and @TODO handle any errors
	if successors, ok := e.successors[marketID]; ok {
		delete(e.successors, marketID)
		for _, sID := range successors {
			e.RejectMarket(ctx, sID)
			delete(e.isSuccessor, sID)
		}
	}
	// remove entries in succession maps
	delete(e.isSuccessor, marketID)
	// and clear out any state that may exist
	delete(e.marketCPStates, marketID)
	return nil
}

// StartOpeningAuction will start the opening auction of the given market.
// This will work only if the market is currently in a PROPOSED state.
func (e *Engine) StartOpeningAuction(ctx context.Context, marketID string) error {
	if e.log.IsDebug() {
		e.log.Debug("start opening auction", logging.MarketID(marketID))
	}

	if mkt, ok := e.allMarkets[marketID]; ok {
		return mkt.StartOpeningAuction(ctx)
	}

	return ErrMarketDoesNotExist
}

func (e *Engine) SucceedMarket(ctx context.Context, successor, parent string) error {
	return e.succeedOrRestore(ctx, successor, parent, false)
}

func (e *Engine) restoreOwnState(ctx context.Context, mID string) (bool, error) {
	mkt, ok := e.futureMarkets[mID]
	if !ok {
		return false, ErrMarketDoesNotExist
	}
	if state, ok := e.marketCPStates[mID]; ok {
		// set ELS state and the like
		mkt.RestoreELS(ctx, state)
		// if there was state of the market to restore, then check if this is a successor market
		if pid := mkt.GetParentMarketID(); len(pid) > 0 {
			// mark parent market as being succeeded
			if pMkt, ok := e.futureMarkets[pid]; ok {
				pMkt.SetSucceeded()
			}
			for _, pending := range e.successors[pid] {
				if pending == mID {
					continue
				}
				e.RejectMarket(ctx, pending)
			}
			delete(e.successors, pid)
			delete(e.isSuccessor, mID)
		}
		return true, nil
	}
	return false, nil
}

func (e *Engine) succeedOrRestore(ctx context.Context, successor, parent string, restore bool) error {
	mkt, ok := e.futureMarkets[successor]
	if !ok {
		// this can happen if a proposal vote closed, but the proposal had an enactment time in the future.
		// Between the proposal being accepted and enacted, another proposal may be enacted first.
		// Whenever the parent is succeeded, all other markets are rejected and removed from the map here,
		// nevertheless the proposal is still valid, and updated by the governance engine.
		return ErrMarketDoesNotExist
	}
	if restore {
		// first up: when restoring markets, check to see if this successor should be rejected
		if _, ok := e.skipRestoreSuccessors[parent]; ok {
			_ = e.RejectMarket(ctx, successor)
			delete(e.successors, parent)
			delete(e.isSuccessor, successor)
			// no error: we just do not care about this market anymore
			return nil
		}
	}
	// if this is a market restore, first check to see if there is some state already
	_, ok = e.GetMarket(parent, true)
	if !ok && !restore {
		// a successor market that has passed the vote, but the parent market either already was succeeded
		// or the proposal vote closed when the parent market was still around, but it wasn't enacted until now
		// and since then the parent market state expired. This shouldn't really happen save for checkpoints,
		// but then the proposal will be rejected/closed later on.
		mkt.ResetParentIDAndInsurancePoolFraction()
		return nil
	}
	_, sok := e.marketCPStates[parent]
	// restoring a market, but no state of the market nor parent market exists. Treat market as parent.
	if restore && !sok && !ok {
		// restoring a market, but the market state and parent market both are missing
		// this market, upon leaving opening auction, cannot possibly succeed a market that no longer exists
		// now we should reset
		mkt.ResetParentIDAndInsurancePoolFraction()
		// remove from maps
		delete(e.successors, parent)
		delete(e.isSuccessor, successor)
		return nil
	}
	// succeeding a parent market before it was enacted is not allowed
	if pmo, ok := e.futureMarkets[parent]; ok && !restore && pmo.Mkt().State == types.MarketStateProposed {
		e.RejectMarket(ctx, successor)
		return ErrParentMarketNotEnactedYet
	}
	// successor market set up accordingly, clean up the state
	// first reject all pending successors proposed for the same parent
	return nil
}

// IsEligibleForProposerBonus checks if the given value is greater than that market quantum * quantum_multiplier.
func (e *Engine) IsEligibleForProposerBonus(marketID string, value *num.Uint) bool {
	if _, ok := e.allMarkets[marketID]; ok {
		quantum, err := e.collateral.GetAssetQuantum(e.futureMarkets[marketID].GetAssetForProposerBonus())
		if err != nil {
			return false
		}
		return value.ToDecimal().GreaterThan(quantum.Mul(e.npv.marketCreationQuantumMultiple))
	}
	return false
}

// SubmitMarket submits a new market configuration to the network.
func (e *Engine) SubmitMarket(ctx context.Context, marketConfig *types.Market, proposer string, oos time.Time) error {
	return e.submitOrRestoreMarket(ctx, marketConfig, proposer, true, oos)
}

// SubmitSpotMarket submits a new spot market configuration to the network.
func (e *Engine) SubmitSpotMarket(ctx context.Context, marketConfig *types.Market, proposer string, oos time.Time) error {
	return e.submitOrRestoreSpotMarket(ctx, marketConfig, proposer, true, oos)
}

// RestoreMarket restores a new market from proposal checkpoint.
func (e *Engine) RestoreMarket(ctx context.Context, marketConfig *types.Market) error {
	proposer := e.marketActivityTracker.GetProposer(marketConfig.ID)
	if len(proposer) == 0 {
		return ErrMarketDoesNotExist
	}
	// restoring a market means starting it as though the proposal was accepted now.
	if err := e.submitOrRestoreMarket(ctx, marketConfig, "", false, e.timeService.GetTimeNow()); err != nil {
		return err
	}
	// attempt to restore market state from checkpoint, returns true if state (ELS) was restored
	// error if the market doesn't exist
	ok, err := e.restoreOwnState(ctx, marketConfig.ID)
	if err != nil {
		return err
	}
	if ok {
		// existing state has been restored. This means a potential parent market has been succeeded
		// the parent market may no longer be present. In that case, remove the reference to the parent market
		if len(marketConfig.ParentMarketID) == 0 {
			return nil
		}
		// successor had state to restore, meaning it left opening auction, and no other successors with the same parent market
		// can be restored after this point.
		e.skipRestoreSuccessors[marketConfig.ParentMarketID] = struct{}{}
		// any pending successors that didn't manage to leave opening auction should be rejected at this point:
		pendingSuccessors := e.successors[marketConfig.ParentMarketID]
		for _, sid := range pendingSuccessors {
			_ = e.RejectMarket(ctx, sid)
		}
		// check to see if the parent market can be found, remove from the successor maps if the parent is gone
		// the market itself should still hold the reference because state was restored
		pmkt, ok := e.futureMarkets[marketConfig.ParentMarketID]
		if ok {
			// market parent as having been succeeded
			pmkt.SetSucceeded()
		}
		// remove the parent from the successors map
		delete(e.successors, marketConfig.ParentMarketID)
		// remove from the isSuccessor map, do not reset the parent ID reference to preserve the reference in the events.
		delete(e.isSuccessor, marketConfig.ID)
		return nil
	}
	// this is a successor market, handle accordingly
	if pid := marketConfig.ParentMarketID; len(pid) > 0 {
		return e.succeedOrRestore(ctx, marketConfig.ID, pid, true)
	}
	return nil
}

func (e *Engine) submitOrRestoreMarket(ctx context.Context, marketConfig *types.Market, proposer string, isNewMarket bool, oos time.Time) error {
	if e.log.IsDebug() {
		msg := "submit market"
		if !isNewMarket {
			msg = "restore market"
		}
		e.log.Debug(msg, logging.Market(*marketConfig))
	}

	if err := e.submitMarket(ctx, marketConfig, oos); err != nil {
		return err
	}
	if pid := marketConfig.ParentMarketID; len(pid) > 0 {
		ss, ok := e.successors[pid]
		if !ok {
			ss = make([]string, 0, 5)
		}
		id := marketConfig.ID
		// add successor market to the successors, to track which markets to get rid off once one successor is enacted
		e.successors[pid] = append(ss, id)
		e.isSuccessor[id] = pid
	}

	if isNewMarket {
		assets, err := marketConfig.GetAssets()
		if err != nil {
			e.log.Panic("failed to get asset from market config", logging.String("market", marketConfig.ID), logging.String("error", err.Error()))
		}
		e.marketActivityTracker.MarketProposed(assets[0], marketConfig.ID, proposer)
	}

	// keep state in pending, opening auction is triggered when proposal is enacted
	mkt := e.futureMarkets[marketConfig.ID]
	e.publishNewMarketInfos(ctx, mkt.GetMarketData(), *mkt.Mkt())
	return nil
}

func (e *Engine) submitOrRestoreSpotMarket(ctx context.Context, marketConfig *types.Market, proposer string, isNewMarket bool, oos time.Time) error {
	if e.log.IsDebug() {
		msg := "submit spot market"
		if !isNewMarket {
			msg = "restore spot market"
		}
		e.log.Debug(msg, logging.Market(*marketConfig))
	}

	if err := e.submitSpotMarket(ctx, marketConfig, oos); err != nil {
		return err
	}

	if isNewMarket {
		assets, err := marketConfig.GetAssets()
		if err != nil {
			e.log.Panic("failed to get asset from market config", logging.String("market", marketConfig.ID), logging.String("error", err.Error()))
		}
		e.marketActivityTracker.MarketProposed(assets[1], marketConfig.ID, proposer)
	}

	// keep state in pending, opening auction is triggered when proposal is enacted
	mkt := e.spotMarkets[marketConfig.ID]
	e.publishNewMarketInfos(ctx, mkt.GetMarketData(), *mkt.Mkt())
	return nil
}

// UpdateSpotMarket will update an existing market configuration.
func (e *Engine) UpdateSpotMarket(ctx context.Context, marketConfig *types.Market) error {
	e.log.Info("update spot market", logging.Market(*marketConfig))

	mkt := e.spotMarkets[marketConfig.ID]
	if err := mkt.Update(ctx, marketConfig); err != nil {
		return err
	}
	e.publishUpdateMarketInfos(ctx, mkt.GetMarketData(), *mkt.Mkt())
	return nil
}

func (e *Engine) VerifyUpdateMarketState(changes *types.MarketStateUpdateConfiguration) error {
	// futures or perps market
	if market, ok := e.futureMarkets[changes.MarketID]; ok {
		if changes.SettlementPrice == nil && changes.UpdateType == types.MarketStateUpdateTypeTerminate {
			return fmt.Errorf("missing settlement price for governance initiated futures market termination")
		}
		state := market.GetMarketState()
		if state == types.MarketStateCancelled || state == types.MarketStateClosed || state == types.MarketStateRejected || state == types.MarketStateSettled || state == types.MarketStateTradingTerminated {
			return fmt.Errorf("invalid state update request. Market is already in a terminal state")
		}
		if changes.UpdateType == types.MarketStateUpdateTypeResume && state != types.MarketStateSuspendedViaGovernance {
			return fmt.Errorf("invalid state update request. Market for resume is not suspended")
		}
		return nil
	}

	// spot market
	if market, ok := e.spotMarkets[changes.MarketID]; ok {
		if changes.SettlementPrice != nil && changes.UpdateType == types.MarketStateUpdateTypeTerminate {
			return fmt.Errorf("settlement price is not needed for governance initiated spot market termination")
		}
		state := market.GetMarketState()
		if state == types.MarketStateCancelled || state == types.MarketStateClosed || state == types.MarketStateRejected || state == types.MarketStateTradingTerminated {
			return fmt.Errorf("invalid state update request. Market is already in a terminal state")
		}
		if changes.UpdateType == types.MarketStateUpdateTypeResume && state != types.MarketStateSuspendedViaGovernance {
			return fmt.Errorf("invalid state update request. Market for resume is not suspended")
		}
		return nil
	}
	return ErrMarketDoesNotExist
}

func (e *Engine) UpdateMarketState(ctx context.Context, changes *types.MarketStateUpdateConfiguration) error {
	if market, ok := e.allMarkets[changes.MarketID]; ok {
		return market.UpdateMarketState(ctx, changes)
	}
	return ErrMarketDoesNotExist
}

// UpdateMarket will update an existing market configuration.
func (e *Engine) UpdateMarket(ctx context.Context, marketConfig *types.Market) error {
	e.log.Info("update market", logging.Market(*marketConfig))
	mkt := e.futureMarkets[marketConfig.ID]
	if err := mkt.Update(ctx, marketConfig, e.oracle); err != nil {
		return err
	}
	e.publishUpdateMarketInfos(ctx, mkt.GetMarketData(), *mkt.Mkt())
	return nil
}

func (e *Engine) publishNewMarketInfos(ctx context.Context, data types.MarketData, mkt types.Market) {
	// we send a market data event for this market when it's created so graphql does not fail
	e.broker.Send(events.NewMarketDataEvent(ctx, data))
	e.broker.Send(events.NewMarketCreatedEvent(ctx, mkt))
	e.broker.Send(events.NewMarketUpdatedEvent(ctx, mkt))
}

func (e *Engine) publishUpdateMarketInfos(ctx context.Context, data types.MarketData, mkt types.Market) {
	// we send a market data event for this market when it's created so graphql does not fail
	e.broker.Send(events.NewMarketDataEvent(ctx, data))
	e.broker.Send(events.NewMarketUpdatedEvent(ctx, mkt))
}

// submitMarket will submit a new market configuration to the network.
func (e *Engine) submitMarket(ctx context.Context, marketConfig *types.Market, oos time.Time) error {
	if len(marketConfig.ID) == 0 {
		return ErrNoMarketID
	}

	// ensure the asset for this new market exists
	assets, err := marketConfig.GetAssets()
	if err != nil {
		return err
	}
	asset := assets[0]

	if !e.collateral.AssetExists(asset) {
		e.log.Error("unable to create a market with an invalid asset",
			logging.MarketID(marketConfig.ID),
			logging.AssetID(asset))
	}

	// ignore the response, this cannot fail as the asset
	// is already proven to exists a few line before
	_, _, _ = e.collateral.CreateMarketAccounts(ctx, marketConfig.ID, asset)

	// create market auction state
	mas := monitor.NewAuctionState(marketConfig, oos)
	ad, err := e.assets.Get(asset)
	if err != nil {
		e.log.Error("Failed to create a new market, unknown asset",
			logging.MarketID(marketConfig.ID),
			logging.String("asset-id", asset),
			logging.Error(err),
		)
		return err
	}
	mkt, err := future.NewMarket(
		ctx,
		e.log,
		e.Risk,
		e.Position,
		e.Settlement,
		e.Matching,
		e.Fee,
		e.Liquidity,
		e.collateral,
		e.oracle,
		marketConfig,
		e.timeService,
		e.broker,
		mas,
		e.stateVarEngine,
		e.marketActivityTracker,
		ad,
		e.peggedOrderCountUpdated,
		e.referralDiscountRewardService,
		e.volumeDiscountService,
	)
	if err != nil {
		e.log.Error("failed to instantiate market",
			logging.MarketID(marketConfig.ID),
			logging.Error(err),
		)
		return err
	}

	e.futureMarkets[marketConfig.ID] = mkt
	e.futureMarketsCpy = append(e.futureMarketsCpy, mkt)
	e.allMarkets[marketConfig.ID] = mkt
	e.allMarketsCpy = append(e.allMarketsCpy, mkt)

	return e.propagateInitialNetParamsToFutureMarket(ctx, mkt)
}

// submitMarket will submit a new market configuration to the network.
func (e *Engine) submitSpotMarket(ctx context.Context, marketConfig *types.Market, oos time.Time) error {
	if len(marketConfig.ID) == 0 {
		return ErrNoMarketID
	}

	// ensure the asset for this new market exists
	assets, err := marketConfig.GetAssets()
	if err != nil {
		return err
	}
	baseAsset := assets[spot.BaseAssetIndex]
	if !e.collateral.AssetExists(baseAsset) {
		e.log.Error("unable to create a spot market with an invalid base asset",
			logging.MarketID(marketConfig.ID),
			logging.AssetID(baseAsset))
	}

	quoteAsset := assets[spot.QuoteAssetIndex]
	if !e.collateral.AssetExists(quoteAsset) {
		e.log.Error("unable to create a spot market with an invalid quote asset",
			logging.MarketID(marketConfig.ID),
			logging.AssetID(quoteAsset))
	}

	// create market auction state
	mas := monitor.NewAuctionState(marketConfig, oos)
	bad, err := e.assets.Get(baseAsset)
	if err != nil {
		e.log.Error("Failed to create a new market, unknown asset",
			logging.MarketID(marketConfig.ID),
			logging.String("asset-id", baseAsset),
			logging.Error(err),
		)
		return err
	}
	qad, err := e.assets.Get(quoteAsset)
	if err != nil {
		e.log.Error("Failed to create a new market, unknown asset",
			logging.MarketID(marketConfig.ID),
			logging.String("asset-id", quoteAsset),
			logging.Error(err),
		)
		return err
	}
	mkt, err := spot.NewMarket(
		e.log,
		e.Matching,
		e.Fee,
		e.Liquidity,
		e.collateral,
		marketConfig,
		e.timeService,
		e.broker,
		mas,
		e.stateVarEngine,
		e.marketActivityTracker,
		bad,
		qad,
		e.peggedOrderCountUpdated,
		e.referralDiscountRewardService,
		e.volumeDiscountService,
	)
	if err != nil {
		e.log.Error("failed to instantiate market",
			logging.MarketID(marketConfig.ID),
			logging.Error(err),
		)
		return err
	}

	e.spotMarkets[marketConfig.ID] = mkt
	e.spotMarketsCpy = append(e.spotMarketsCpy, mkt)
	e.allMarkets[marketConfig.ID] = mkt
	e.allMarketsCpy = append(e.allMarketsCpy, mkt)

	e.collateral.CreateSpotMarketAccounts(ctx, marketConfig.ID, quoteAsset)

	if err := e.propagateSpotInitialNetParams(ctx, mkt); err != nil {
		return err
	}

	return nil
}

func (e *Engine) propagateSpotInitialNetParams(ctx context.Context, mkt *spot.Market) error {
	if !e.npv.minLpStakeQuantumMultiple.Equal(num.DecimalFromInt64(-1)) {
		mkt.OnMarketMinLpStakeQuantumMultipleUpdate(ctx, e.npv.minLpStakeQuantumMultiple)
	}
	if e.npv.auctionMinDuration != -1 {
		mkt.OnMarketAuctionMinimumDurationUpdate(ctx, e.npv.auctionMinDuration)
	}
	if !e.npv.infrastructureFee.Equal(num.DecimalFromInt64(-1)) {
		mkt.OnFeeFactorsInfrastructureFeeUpdate(ctx, e.npv.infrastructureFee)
	}

	if !e.npv.makerFee.Equal(num.DecimalFromInt64(-1)) {
		mkt.OnFeeFactorsMakerFeeUpdate(ctx, e.npv.makerFee)
	}

	if e.npv.marketValueWindowLength != -1 {
		mkt.OnMarketValueWindowLengthUpdate(e.npv.marketValueWindowLength)
	}

	if e.npv.markPriceUpdateMaximumFrequency > 0 {
		mkt.OnMarkPriceUpdateMaximumFrequency(ctx, e.npv.markPriceUpdateMaximumFrequency)
	}

	if !e.npv.liquidityV2EarlyExitPenalty.Equal(num.DecimalFromInt64(-1)) { //nolint:staticcheck
		mkt.OnMarketLiquidityV2EarlyExitPenaltyUpdate(e.npv.liquidityV2EarlyExitPenalty)
	}

	if !e.npv.liquidityV2MaxLiquidityFee.Equal(num.DecimalFromInt64(-1)) { //nolint:staticcheck
		mkt.OnMarketLiquidityV2MaximumLiquidityFeeFactorLevelUpdate(e.npv.liquidityV2MaxLiquidityFee)
	}

	if !e.npv.liquidityV2SLANonPerformanceBondPenaltySlope.Equal(num.DecimalFromInt64(-1)) { //nolint:staticcheck
		mkt.OnMarketLiquidityV2SLANonPerformanceBondPenaltySlopeUpdate(e.npv.liquidityV2SLANonPerformanceBondPenaltySlope)
	}

	if !e.npv.liquidityV2SLANonPerformanceBondPenaltyMax.Equal(num.DecimalFromInt64(-1)) { //nolint:staticcheck
		mkt.OnMarketLiquidityV2SLANonPerformanceBondPenaltyMaxUpdate(e.npv.liquidityV2SLANonPerformanceBondPenaltyMax)
	}

	if !e.npv.liquidityV2StakeToCCYVolume.Equal(num.DecimalFromInt64(-1)) { //nolint:staticcheck
		mkt.OnMarketLiquidityV2StakeToCCYVolume(e.npv.liquidityV2StakeToCCYVolume)
	}
	return nil
}

func (e *Engine) propagateInitialNetParamsToFutureMarket(ctx context.Context, mkt *future.Market) error {
	if !e.npv.probabilityOfTradingTauScaling.Equal(num.DecimalFromInt64(-1)) {
		mkt.OnMarketProbabilityOfTradingTauScalingUpdate(ctx, e.npv.probabilityOfTradingTauScaling)
	}
	if !e.npv.minProbabilityOfTradingLPOrders.Equal(num.DecimalFromInt64(-1)) {
		mkt.OnMarketMinProbabilityOfTradingLPOrdersUpdate(ctx, e.npv.minProbabilityOfTradingLPOrders)
	}
	if !e.npv.minLpStakeQuantumMultiple.Equal(num.DecimalFromInt64(-1)) {
		mkt.OnMarketMinLpStakeQuantumMultipleUpdate(ctx, e.npv.minLpStakeQuantumMultiple)
	}
	if e.npv.auctionMinDuration != -1 {
		mkt.OnMarketAuctionMinimumDurationUpdate(ctx, e.npv.auctionMinDuration)
	}

	if !e.npv.infrastructureFee.Equal(num.DecimalFromInt64(-1)) {
		mkt.OnFeeFactorsInfrastructureFeeUpdate(ctx, e.npv.infrastructureFee)
	}

	if !e.npv.makerFee.Equal(num.DecimalFromInt64(-1)) {
		mkt.OnFeeFactorsMakerFeeUpdate(ctx, e.npv.makerFee)
	}

	if e.npv.scalingFactors != nil {
		if err := mkt.OnMarginScalingFactorsUpdate(ctx, e.npv.scalingFactors); err != nil {
			return err
		}
	}

	if e.npv.marketValueWindowLength != -1 {
		mkt.OnMarketValueWindowLengthUpdate(e.npv.marketValueWindowLength)
	}

	if !e.npv.maxLiquidityFee.Equal(num.DecimalFromInt64(-1)) {
		mkt.OnMarketLiquidityMaximumLiquidityFeeFactorLevelUpdate(e.npv.maxLiquidityFee)
	}
	if e.npv.markPriceUpdateMaximumFrequency > 0 {
		mkt.OnMarkPriceUpdateMaximumFrequency(ctx, e.npv.markPriceUpdateMaximumFrequency)
	}

	mkt.OnMarketPartiesMaximumStopOrdersUpdate(ctx, e.npv.marketPartiesMaximumStopOrdersUpdate)

	e.propagateSLANetParams(ctx, mkt)

	return nil
}

func (e *Engine) propagateSLANetParams(_ context.Context, mkt *future.Market) {
	if !e.npv.liquidityV2BondPenaltyFactor.Equal(num.DecimalFromInt64(-1)) { //nolint:staticcheck
		mkt.OnMarketLiquidityV2BondPenaltyFactorUpdate(e.npv.liquidityV2BondPenaltyFactor)
	}

	if !e.npv.liquidityV2EarlyExitPenalty.Equal(num.DecimalFromInt64(-1)) { //nolint:staticcheck
		mkt.OnMarketLiquidityV2EarlyExitPenaltyUpdate(e.npv.liquidityV2EarlyExitPenalty)
	}

	if !e.npv.liquidityV2MaxLiquidityFee.Equal(num.DecimalFromInt64(-1)) { //nolint:staticcheck
		mkt.OnMarketLiquidityV2MaximumLiquidityFeeFactorLevelUpdate(e.npv.liquidityV2MaxLiquidityFee)
	}

	if !e.npv.liquidityV2SLANonPerformanceBondPenaltySlope.Equal(num.DecimalFromInt64(-1)) { //nolint:staticcheck
		mkt.OnMarketLiquidityV2SLANonPerformanceBondPenaltySlopeUpdate(e.npv.liquidityV2SLANonPerformanceBondPenaltySlope)
	}

	if !e.npv.liquidityV2SLANonPerformanceBondPenaltyMax.Equal(num.DecimalFromInt64(-1)) { //nolint:staticcheck
		mkt.OnMarketLiquidityV2SLANonPerformanceBondPenaltyMaxUpdate(e.npv.liquidityV2SLANonPerformanceBondPenaltyMax)
	}

	if !e.npv.liquidityV2StakeToCCYVolume.Equal(num.DecimalFromInt64(-1)) { //nolint:staticcheck
		mkt.OnMarketLiquidityV2StakeToCCYVolume(e.npv.liquidityV2StakeToCCYVolume)
	}

	mkt.OnMarketLiquidityV2ProvidersFeeCalculationTimeStep(e.npv.liquidityV2ProvidersFeeCalculationTimeStep)
}

func (e *Engine) removeMarket(mktID string) {
	e.log.Debug("removing market", logging.String("id", mktID))
	delete(e.allMarkets, mktID)
	for i, mkt := range e.allMarketsCpy {
		if mkt.GetID() == mktID {
			copy(e.allMarketsCpy[i:], e.allMarketsCpy[i+1:])
			e.allMarketsCpy[len(e.allMarketsCpy)-1] = nil
			e.allMarketsCpy = e.allMarketsCpy[:len(e.allMarketsCpy)-1]
			break
		}
	}
	if _, ok := e.futureMarkets[mktID]; ok {
		delete(e.futureMarkets, mktID)
		for i, mkt := range e.futureMarketsCpy {
			if mkt.GetID() == mktID {
				mkt.StopSnapshots()

				copy(e.futureMarketsCpy[i:], e.futureMarketsCpy[i+1:])
				e.futureMarketsCpy[len(e.futureMarketsCpy)-1] = nil
				e.futureMarketsCpy = e.futureMarketsCpy[:len(e.futureMarketsCpy)-1]
				e.marketActivityTracker.RemoveMarket(mkt.GetSettlementAsset(), mktID)
				e.log.Debug("removed in total", logging.String("id", mktID))
				return
			}
		}
		return
	}
	if _, ok := e.spotMarkets[mktID]; ok {
		delete(e.spotMarkets, mktID)
		for i, mkt := range e.spotMarketsCpy {
			if mkt.GetID() == mktID {
				mkt.StopSnapshots()
				copy(e.spotMarketsCpy[i:], e.spotMarketsCpy[i+1:])
				e.spotMarketsCpy[len(e.spotMarketsCpy)-1] = nil
				e.spotMarketsCpy = e.spotMarketsCpy[:len(e.spotMarketsCpy)-1]
				e.marketActivityTracker.RemoveMarket(mkt.GetAssetForProposerBonus(), mktID)
				e.log.Debug("removed in total", logging.String("id", mktID))
				return
			}
		}
	}
}

func (e *Engine) peggedOrderCountUpdated(added int64) {
	e.totalPeggedOrdersCount += added
}

func (e *Engine) canSubmitPeggedOrder() bool {
	return uint64(e.totalPeggedOrdersCount) < e.maxPeggedOrders
}

func (e *Engine) SubmitStopOrders(
	ctx context.Context,
	submission *types.StopOrdersSubmission,
	party string,
	idgen common.IDGenerator,
	fallsBelowID *string,
	risesAboveID *string,
) (*types.OrderConfirmation, error) {
	var market string
	if submission.FallsBelow != nil {
		market = submission.FallsBelow.OrderSubmission.MarketID
	} else {
		market = submission.RisesAbove.OrderSubmission.MarketID
	}

	if mkt, ok := e.allMarkets[market]; ok {
		conf, err := mkt.SubmitStopOrdersWithIDGeneratorAndOrderIDs(
			ctx, submission, party, idgen, fallsBelowID, risesAboveID)
		if err != nil {
			return nil, err
		}

		// not necessary going to trade on submission, could be nil
		if conf != nil {
			// increasing the gauge, just because we reuse the
			// decrement function, and it required the order + passive
			metrics.OrderGaugeAdd(1, market)
			e.decrementOrderGaugeMetrics(market, conf.Order, conf.PassiveOrdersAffected)
		}

		return conf, nil
	}
	return nil, ErrMarketDoesNotExist
}

func (e *Engine) CancelStopOrders(ctx context.Context, cancel *types.StopOrdersCancellation, party string, idgen common.IDGenerator) error {
	// ensure that if orderID is specified marketId is as well
	if len(cancel.OrderID) > 0 && len(cancel.MarketID) <= 0 {
		return ErrMarketIDRequiredWhenOrderIDSpecified
	}

	if len(cancel.MarketID) > 0 {
		if len(cancel.OrderID) > 0 {
			return e.cancelStopOrders(ctx, party, cancel.MarketID, cancel.OrderID, idgen)
		}
		return e.cancelStopOrdersByMarket(ctx, party, cancel.MarketID)
	}
	return e.cancelAllPartyStopOrders(ctx, party)
}

func (e *Engine) cancelStopOrders(ctx context.Context, party, market, orderID string, _ common.IDGenerator) error {
	if mkt, ok := e.allMarkets[market]; ok {
		err := mkt.CancelStopOrder(ctx, party, orderID)
		if err != nil {
			return err
		}
		return nil
	}
	return types.ErrInvalidMarketID
}

func (e *Engine) cancelStopOrdersByMarket(ctx context.Context, party, market string) error {
	if mkt, ok := e.allMarkets[market]; ok {
		err := mkt.CancelAllStopOrders(ctx, party)
		if err != nil {
			return err
		}
	}
	return types.ErrInvalidMarketID
}

func (e *Engine) cancelAllPartyStopOrders(ctx context.Context, party string) error {
	for _, mkt := range e.allMarketsCpy {
		err := mkt.CancelAllStopOrders(ctx, party)
		if err != nil && err != common.ErrTradingNotAllowed {
			return err
		}
	}
	return nil
}

// SubmitOrder checks the incoming order and submits it to a Vega market.
func (e *Engine) SubmitOrder(ctx context.Context, submission *types.OrderSubmission, party string, idgen common.IDGenerator, orderID string) (*types.OrderConfirmation, error) {
	timer := metrics.NewTimeCounter(submission.MarketID, "execution", "SubmitOrder")
	defer func() {
		timer.EngineTimeCounterAdd()
	}()

	if e.log.IsDebug() {
		e.log.Debug("submit order", logging.OrderSubmission(submission))
	}

	if mkt, ok := e.allMarkets[submission.MarketID]; ok {
		if submission.PeggedOrder != nil && !e.canSubmitPeggedOrder() {
			return nil, &types.ErrTooManyPeggedOrders
		}

		metrics.OrderGaugeAdd(1, submission.MarketID)
		conf, err := mkt.SubmitOrderWithIDGeneratorAndOrderID(
			ctx, submission, party, idgen, orderID, true)
		if err != nil {
			return nil, err
		}

		e.decrementOrderGaugeMetrics(submission.MarketID, conf.Order, conf.PassiveOrdersAffected)
		return conf, nil
	}
	return nil, types.ErrInvalidMarketID
}

// AmendOrder takes order amendment details and attempts to amend the order
// if it exists and is in a editable state.
func (e *Engine) AmendOrder(ctx context.Context, amendment *types.OrderAmendment, party string, idgen common.IDGenerator) (*types.OrderConfirmation, error) {
	timer := metrics.NewTimeCounter(amendment.MarketID, "execution", "AmendOrder")
	defer func() {
		timer.EngineTimeCounterAdd()
	}()

	if e.log.IsDebug() {
		e.log.Debug("amend order", logging.OrderAmendment(amendment))
	}

	if mkt, ok := e.allMarkets[amendment.MarketID]; ok {
		conf, err := mkt.AmendOrderWithIDGenerator(ctx, amendment, party, idgen)
		if err != nil {
			return nil, err
		}

		e.decrementOrderGaugeMetrics(amendment.MarketID, conf.Order, conf.PassiveOrdersAffected)
		return conf, nil
	}
	return nil, types.ErrInvalidMarketID
}

func (e *Engine) decrementOrderGaugeMetrics(
	market string,
	order *types.Order,
	passive []*types.Order,
) {
	// order was active, not anymore -> decrement gauge
	if order.Status != types.OrderStatusActive {
		metrics.OrderGaugeAdd(-1, market)
	}
	var passiveCount int
	for _, v := range passive {
		if v.IsFinished() {
			passiveCount++
		}
	}
	if passiveCount > 0 {
		metrics.OrderGaugeAdd(-passiveCount, market)
	}
}

// CancelOrder takes order details and attempts to cancel if it exists in matching engine, stores etc.
func (e *Engine) CancelOrder(
	ctx context.Context,
	cancel *types.OrderCancellation,
	party string,
	idgen common.IDGenerator,
) (_ []*types.OrderCancellationConfirmation, returnedErr error) {
	timer := metrics.NewTimeCounter(cancel.MarketID, "execution", "CancelOrder")
	defer func() {
		timer.EngineTimeCounterAdd()
	}()

	if e.log.IsDebug() {
		e.log.Debug("cancel order", logging.OrderCancellation(cancel))
	}

	// ensure that if orderID is specified marketId is as well
	if len(cancel.OrderID) > 0 && len(cancel.MarketID) <= 0 {
		return nil, ErrInvalidOrderCancellation
	}

	if len(cancel.MarketID) > 0 {
		if len(cancel.OrderID) > 0 {
			return e.cancelOrder(ctx, party, cancel.MarketID, cancel.OrderID, idgen)
		}
		return e.cancelOrderByMarket(ctx, party, cancel.MarketID)
	}
	return e.cancelAllPartyOrders(ctx, party)
}

func (e *Engine) cancelOrder(ctx context.Context, party, market, orderID string, idgen common.IDGenerator) ([]*types.OrderCancellationConfirmation, error) {
	if mkt, ok := e.allMarkets[market]; ok {
		conf, err := mkt.CancelOrderWithIDGenerator(ctx, party, orderID, idgen)
		if err != nil {
			return nil, err
		}
		if conf.Order.Status == types.OrderStatusCancelled {
			metrics.OrderGaugeAdd(-1, market)
		}
		return []*types.OrderCancellationConfirmation{conf}, nil
	}
	return nil, types.ErrInvalidMarketID
}

func (e *Engine) cancelOrderByMarket(ctx context.Context, party, market string) ([]*types.OrderCancellationConfirmation, error) {
	if mkt, ok := e.allMarkets[market]; ok {
		confirmations, err := mkt.CancelAllOrders(ctx, party)
		if err != nil {
			return nil, err
		}
		var confirmed int
		for _, conf := range confirmations {
			if conf.Order.Status == types.OrderStatusCancelled {
				confirmed++
			}
		}
		metrics.OrderGaugeAdd(-confirmed, market)
		return confirmations, nil
	}
	return nil, types.ErrInvalidMarketID
}

func (e *Engine) cancelAllPartyOrdersForMarket(ctx context.Context, ID, party string, mkt common.CommonMarket) ([]*types.OrderCancellationConfirmation, error) {
	confs, err := mkt.CancelAllOrders(ctx, party)
	if err != nil && err != common.ErrTradingNotAllowed {
		return nil, err
	}
	var confirmed int
	for _, conf := range confs {
		if conf.Order.Status == types.OrderStatusCancelled {
			confirmed++
		}
	}
	metrics.OrderGaugeAdd(-confirmed, ID)
	return confs, nil
}

func (e *Engine) cancelAllPartyOrders(ctx context.Context, party string) ([]*types.OrderCancellationConfirmation, error) {
	confirmations := []*types.OrderCancellationConfirmation{}

	for _, mkt := range e.allMarketsCpy {
		confs, err := e.cancelAllPartyOrdersForMarket(ctx, mkt.GetID(), party, mkt)
		if err != nil && err != common.ErrTradingNotAllowed {
			return nil, err
		}
		confirmations = append(confirmations, confs...)
	}
	return confirmations, nil
}

func (e *Engine) SubmitLiquidityProvision(ctx context.Context, sub *types.LiquidityProvisionSubmission, party, deterministicID string) error {
	timer := metrics.NewTimeCounter(sub.MarketID, "execution", "LiquidityProvisionSubmission")
	defer func() {
		timer.EngineTimeCounterAdd()
	}()

	if e.log.IsDebug() {
		e.log.Debug("submit liquidity provision",
			logging.LiquidityProvisionSubmission(*sub),
			logging.PartyID(party),
			logging.LiquidityID(deterministicID),
		)
	}

	if mkt, ok := e.allMarkets[sub.MarketID]; ok {
		return mkt.SubmitLiquidityProvision(ctx, sub, party, deterministicID)
	}
	return types.ErrInvalidMarketID
}

func (e *Engine) AmendLiquidityProvision(ctx context.Context, lpa *types.LiquidityProvisionAmendment, party string, deterministicID string) error {
	timer := metrics.NewTimeCounter(lpa.MarketID, "execution", "LiquidityProvisionAmendment")
	defer func() {
		timer.EngineTimeCounterAdd()
	}()

	if e.log.IsDebug() {
		e.log.Debug("amend liquidity provision",
			logging.LiquidityProvisionAmendment(*lpa),
			logging.PartyID(party),
			logging.MarketID(lpa.MarketID),
		)
	}

	if mkt, ok := e.allMarkets[lpa.MarketID]; ok {
		return mkt.AmendLiquidityProvision(ctx, lpa, party, deterministicID)
	}
	return types.ErrInvalidMarketID
}

func (e *Engine) CancelLiquidityProvision(ctx context.Context, cancel *types.LiquidityProvisionCancellation, party string) error {
	timer := metrics.NewTimeCounter(cancel.MarketID, "execution", "LiquidityProvisionCancellation")
	defer func() {
		timer.EngineTimeCounterAdd()
	}()

	if e.log.IsDebug() {
		e.log.Debug("cancel liquidity provision",
			logging.LiquidityProvisionCancellation(*cancel),
			logging.PartyID(party),
			logging.MarketID(cancel.MarketID),
		)
	}

	if mkt, ok := e.allMarkets[cancel.MarketID]; ok {
		return mkt.CancelLiquidityProvision(ctx, cancel, party)
	}
	return types.ErrInvalidMarketID
}

func (e *Engine) OnTick(ctx context.Context, t time.Time) {
	timer := metrics.NewTimeCounter("-", "execution", "OnTick")

	e.log.Debug("updating engine on new time update")

	// notify markets of the time expiration
	toDelete := []string{}
	parentStates := e.getParentStates()
	evts := make([]events.Event, 0, len(e.futureMarketsCpy))
	toSkip := map[int]string{}
	for i, mkt := range e.futureMarketsCpy {
		// we can skip successor markets which reference a parent market that has been succeeded
		if _, ok := toSkip[i]; ok {
			continue
		}
		mkt := mkt
		id := mkt.GetID()
		mdef := mkt.Mkt()
		pstate, isSuccessor := parentStates[id]
		inOA := isSuccessor && mdef.State == types.MarketStatePending
		// this market was a successor, but has no parent state (parent state likely expired
		// although this currently is not possible, better check here.
		if isSuccessor && inOA {
			if pstate == nil {
				delete(e.isSuccessor, id)
				delete(e.successors, mdef.ParentMarketID)
				mkt.ResetParentIDAndInsurancePoolFraction()
				isSuccessor = false
			} else {
				// update parent state in market prior to potentially leaving opening auction
				mkt.InheritParent(ctx, pstate)
			}
		}
		closing := mkt.OnTick(ctx, t)
		// successor market has left opening auction
		leftOA := inOA && mdef.State == types.MarketStateActive
		if closing {
			e.log.Info("market is closed, removing from execution engine",
				logging.MarketID(id))
			toDelete = append(toDelete, id)
		}
		// this can only be true if mkt was a successor, and the successor market has left the opening auction
		if leftOA {
			pid := mdef.ParentMarketID
			// transfer insurance pool balance
			if !mdef.InsurancePoolFraction.IsZero() {
				lm := e.collateral.SuccessorInsuranceFraction(ctx, id, pid, mkt.GetSettlementAsset(), mdef.InsurancePoolFraction)
				if lm != nil {
					e.broker.Send(events.NewLedgerMovements(ctx, []*types.LedgerMovement{lm}))
				}
			}
			// set parent market as succeeded, clear insurance pool account if needed
			if pmkt, ok := e.futureMarkets[pid]; ok {
				pmkt.SetSucceeded()
			} else {
				asset := mkt.GetSettlementAsset()
				// clear parent market insurance pool
				if clearTransfers, _ := e.collateral.ClearInsurancepool(ctx, pid, asset, true); len(clearTransfers) > 0 {
					e.broker.Send(events.NewLedgerMovements(ctx, clearTransfers))
				}
			}
			// add other markets that need to be rejected to the skip list
			toSkip = e.getPendingSuccessorsToReject(pid, id, toSkip)
			// remove data used to indicate that the parent market has pending successors
			delete(e.isSuccessor, id)
			delete(e.successors, pid)
			delete(e.marketCPStates, pid)
		} else if isSuccessor {
			// this call can be made even if the market has left opening auction, but checking this here, too, is better than
			// relying on how this is implemented
			mkt.RollbackInherit(ctx)
		}
		if !mkt.IsSucceeded() {
			// the market was not yet succeeded -> capture state
			cps := mkt.GetCPState()
			// set until what time this state is considered valid.
			cps.TTL = t.Add(e.successorWindow)
			e.marketCPStates[id] = cps
		} else {
			// market was succeeded
			delete(e.marketCPStates, id)
		}
		evts = append(evts, events.NewMarketDataEvent(ctx, mkt.GetMarketData()))
	}

	for _, mkt := range e.spotMarketsCpy {
		closing := mkt.OnTick(ctx, t)
		if closing {
			e.log.Info("spot market is closed, removing from execution engine",
				logging.MarketID(mkt.GetID()))
			toDelete = append(toDelete, mkt.GetID())
		}
		evts = append(evts, events.NewMarketDataEvent(ctx, mkt.GetMarketData()))
	}
	e.broker.SendBatch(evts)
	// reject successor markets in the toSkip list
	for _, sid := range toSkip {
		e.RejectMarket(ctx, sid)
	}

	rmCPStates := make([]string, 0, len(toDelete))
	for _, id := range toDelete {
		// a cancelled market cannot be succeeded, so remove it from the CP state immediately
		if m, ok := e.futureMarkets[id]; ok && m.Mkt().State == types.MarketStateCancelled {
			rmCPStates = append(rmCPStates, id)
		}
		e.removeMarket(id)
	}

	// sort the marketCPStates by ID since the order we clear insurance pools
	// changes the division when we split it across remaining markets and
	// who the remainder ends up with if it doesn't divide equally.
	allIDs := []string{}
	for id := range e.marketCPStates {
		allIDs = append(allIDs, id)
	}
	sort.Strings(allIDs)

	// find state that should expire
	for _, id := range allIDs {
		// market field will be nil if the market is still current (ie not closed/settled)
		cpm := e.marketCPStates[id]
		if !cpm.TTL.Before(t) {
			// CP data has not expired yet
			continue
		}
		if cpm.Market == nil {
			// expired, and yet somehow the market is gone, this is stale data, must be removed
			if _, ok := e.futureMarkets[id]; !ok {
				rmCPStates = append(rmCPStates, id)
			}
		} else {
			// market state was set, so this is a closed/settled market that was not succeeded in time
			rmCPStates = append(rmCPStates, id)
			assets, _ := cpm.Market.GetAssets()
			if clearTransfers, _ := e.collateral.ClearInsurancepool(ctx, id, assets[0], true); len(clearTransfers) > 0 {
				e.broker.Send(events.NewLedgerMovements(ctx, clearTransfers))
			}
		}
	}
	for _, id := range rmCPStates {
		delete(e.marketCPStates, id)
		if ss, ok := e.successors[id]; ok {
			// parent market expired, remove parent ID
			for _, s := range ss {
				delete(e.isSuccessor, s)
				if mkt, ok := e.futureMarkets[s]; ok {
					mkt.ResetParentIDAndInsurancePoolFraction()
				}
			}
		}
		delete(e.successors, id)
	}

	timer.EngineTimeCounterAdd()
}

func (e *Engine) getPendingSuccessorsToReject(parent, successor string, toSkip map[int]string) map[int]string {
	ss, ok := e.successors[parent]
	if !ok {
		return toSkip
	}
	// iterate over all pending successors for the given parent
	for _, sid := range ss {
		// ignore the actual successor
		if sid == successor {
			continue
		}
		if _, ok := e.futureMarkets[sid]; !ok {
			continue
		}
		for i, mkt := range e.futureMarketsCpy {
			if mkt.GetID() == sid {
				toSkip[i] = sid
			}
		}
	}
	return toSkip
}

func (e *Engine) getParentStates() map[string]*types.CPMarketState {
	// all successor markets need to have a reference to the parent state
	states := make(map[string]*types.CPMarketState, len(e.isSuccessor))
	// for each parent market, get the successors
	for pid, successors := range e.successors {
		state, sok := e.marketCPStates[pid]
		if !sok {
			if pmkt, ok := e.futureMarkets[pid]; ok {
				state = pmkt.GetCPState()
			}
		}
		// if the state does not exist, then there is nothing to inherit. This is handled elsewhere
		// include nil states in the map
		for _, sid := range successors {
			states[sid] = state
		}
	}
	return states
}

func (e *Engine) BlockEnd(ctx context.Context) {
	for _, mkt := range e.allMarketsCpy {
		mkt.BlockEnd(ctx)
	}
}

func (e *Engine) GetMarketState(mktID string) (types.MarketState, error) {
	if mkt, ok := e.allMarkets[mktID]; ok {
		return mkt.GetMarketState(), nil
	}
	return types.MarketStateUnspecified, types.ErrInvalidMarketID
}

func (e *Engine) IsSucceeded(mktID string) bool {
	if mkt, ok := e.futureMarkets[mktID]; ok {
		return mkt.IsSucceeded()
	}
	// checking marketCPStates is pointless. The parent market could not be found to validate the proposal, so it will be rejected outright
	// if the market is no longer in e.markets, it will be set in marketCPStates, and therefore the successor proposal must be accepted.
	return false
}

func (e *Engine) GetMarketData(mktID string) (types.MarketData, error) {
	if mkt, ok := e.allMarkets[mktID]; ok {
		return mkt.GetMarketData(), nil
	}
	return types.MarketData{}, types.ErrInvalidMarketID
}

func (e *Engine) OnMarketAuctionMinimumDurationUpdate(ctx context.Context, d time.Duration) error {
	for _, mkt := range e.allMarketsCpy {
		mkt.OnMarketAuctionMinimumDurationUpdate(ctx, d)
	}
	e.npv.auctionMinDuration = d
	return nil
}

func (e *Engine) OnMarkPriceUpdateMaximumFrequency(ctx context.Context, d time.Duration) error {
	for _, mkt := range e.allMarketsCpy {
		mkt.OnMarkPriceUpdateMaximumFrequency(ctx, d)
	}
	e.npv.markPriceUpdateMaximumFrequency = d
	return nil
}

// SLA liquidity - currently only used in spots.
func (e *Engine) OnMarketLiquidityV2BondPenaltyUpdate(_ context.Context, d num.Decimal) error {
	if e.log.IsDebug() {
		e.log.Debug("update market liquidity bond penalty (liquidity v2)",
			logging.Decimal("bond-penalty-factor", d),
		)
	}

	for _, m := range e.futureMarketsCpy {
		m.OnMarketLiquidityV2BondPenaltyFactorUpdate(d)
	}
	e.npv.liquidityV2BondPenaltyFactor = d

	return nil
}

// SLA liquidity - currently only used in spots.
func (e *Engine) OnMarketLiquidityV2EarlyExitPenaltyUpdate(_ context.Context, d num.Decimal) error {
	if e.log.IsDebug() {
		e.log.Debug("update market liquidity early exit penalty (liquidity v2)",
			logging.Decimal("early-exit-penalty", d),
		)
	}

	for _, m := range e.allMarketsCpy {
		m.OnMarketLiquidityV2EarlyExitPenaltyUpdate(d)
	}
	e.npv.liquidityV2EarlyExitPenalty = d
	return nil
}

// SLA liquidity - currently only used in spots.
func (e *Engine) OnMarketLiquidityV2MaximumLiquidityFeeFactorLevelUpdate(_ context.Context, d num.Decimal) error {
	if e.log.IsDebug() {
		e.log.Debug("update liquidity provision max liquidity fee factor (liquidity v2)",
			logging.Decimal("max-liquidity-fee", d),
		)
	}

	for _, m := range e.allMarketsCpy {
		m.OnMarketLiquidityV2MaximumLiquidityFeeFactorLevelUpdate(d)
	}
	e.npv.liquidityV2MaxLiquidityFee = d

	return nil
}

// SLA liquidity - currently only used in spots.
func (e *Engine) OnMarketLiquidityV2SLANonPerformanceBondPenaltySlopeUpdate(_ context.Context, d num.Decimal) error {
	if e.log.IsDebug() {
		e.log.Debug("update market SLA non performance bond penalty slope (liquidity v2)",
			logging.Decimal("bond-penalty-slope", d),
		)
	}

	for _, m := range e.allMarketsCpy {
		m.OnMarketLiquidityV2SLANonPerformanceBondPenaltySlopeUpdate(d)
	}
	e.npv.liquidityV2SLANonPerformanceBondPenaltySlope = d

	return nil
}

// SLA liquidity - currently only used in spots.
func (e *Engine) OnMarketLiquidityV2SLANonPerformanceBondPenaltyMaxUpdate(_ context.Context, d num.Decimal) error {
	if e.log.IsDebug() {
		e.log.Debug("update market SLA non performance bond penalty max (liquidity v2)",
			logging.Decimal("bond-penalty-max", d),
		)
	}

	for _, m := range e.allMarketsCpy {
		m.OnMarketLiquidityV2SLANonPerformanceBondPenaltyMaxUpdate(d)
	}
	e.npv.liquidityV2SLANonPerformanceBondPenaltyMax = d

	return nil
}

func (e *Engine) OnMarketLiquidityV2StakeToCCYVolumeUpdate(_ context.Context, d num.Decimal) error {
	if e.log.IsDebug() {
		e.log.Debug("update market stake to CCYVolume (liquidity v2)",
			logging.Decimal("stake-to-ccy-volume", d),
		)
	}

	for _, m := range e.allMarketsCpy {
		m.OnMarketLiquidityV2StakeToCCYVolume(d)
	}
	e.npv.liquidityV2StakeToCCYVolume = d
	return nil
}

func (e *Engine) OnMarketLiquidityV2ProvidersFeeCalculationTimeStep(_ context.Context, d time.Duration) error {
	if e.log.IsDebug() {
		e.log.Debug("update market SLA providers fee calculation time step (liquidity v2)",
			logging.Duration("providersFeeCalculationTimeStep", d),
		)
	}
	for _, m := range e.allMarketsCpy {
		m.OnMarketLiquidityV2ProvidersFeeCalculationTimeStep(d)
	}
	e.npv.liquidityV2ProvidersFeeCalculationTimeStep = d
	return nil
}

func (e *Engine) OnMarketMarginScalingFactorsUpdate(ctx context.Context, v interface{}) error {
	if e.log.IsDebug() {
		e.log.Debug("update market scaling factors",
			logging.Reflect("scaling-factors", v),
		)
	}

	pscalingFactors, ok := v.(*vega.ScalingFactors)
	if !ok {
		return errors.New("invalid types for Margin ScalingFactors")
	}
	scalingFactors := types.ScalingFactorsFromProto(pscalingFactors)
	for _, mkt := range e.futureMarketsCpy {
		if err := mkt.OnMarginScalingFactorsUpdate(ctx, scalingFactors); err != nil {
			return err
		}
	}
	e.npv.scalingFactors = scalingFactors
	return nil
}

func (e *Engine) OnMarketFeeFactorsMakerFeeUpdate(ctx context.Context, d num.Decimal) error {
	if e.log.IsDebug() {
		e.log.Debug("update maker fee in market fee factors",
			logging.Decimal("maker-fee", d),
		)
	}

	for _, mkt := range e.allMarketsCpy {
		mkt.OnFeeFactorsMakerFeeUpdate(ctx, d)
	}
	e.npv.makerFee = d
	return nil
}

func (e *Engine) OnMarketFeeFactorsInfrastructureFeeUpdate(ctx context.Context, d num.Decimal) error {
	if e.log.IsDebug() {
		e.log.Debug("update infrastructure fee in market fee factors",
			logging.Decimal("infrastructure-fee", d),
		)
	}
	for _, mkt := range e.allMarketsCpy {
		mkt.OnFeeFactorsInfrastructureFeeUpdate(ctx, d)
	}
	e.npv.infrastructureFee = d
	return nil
}

func (e *Engine) OnMarketValueWindowLengthUpdate(_ context.Context, d time.Duration) error {
	if e.log.IsDebug() {
		e.log.Debug("update market value window length",
			logging.Duration("window-length", d),
		)
	}

	for _, mkt := range e.allMarketsCpy {
		mkt.OnMarketValueWindowLengthUpdate(d)
	}
	e.npv.marketValueWindowLength = d
	return nil
}

// to be removed and replaced by its v2 counterpart. in use only for future.
func (e *Engine) OnMarketLiquidityMaximumLiquidityFeeFactorLevelUpdate(_ context.Context, d num.Decimal) error {
	if e.log.IsDebug() {
		e.log.Debug("update liquidity provision max liquidity fee factor",
			logging.Decimal("max-liquidity-fee", d),
		)
	}

	for _, mkt := range e.futureMarketsCpy {
		mkt.OnMarketLiquidityMaximumLiquidityFeeFactorLevelUpdate(d)
	}
	e.npv.maxLiquidityFee = d

	return nil
}

func (e *Engine) OnMarketProbabilityOfTradingTauScalingUpdate(ctx context.Context, d num.Decimal) error {
	if e.log.IsDebug() {
		e.log.Debug("update probability of trading tau scaling",
			logging.Decimal("probability-of-trading-tau-scaling", d),
		)
	}
	for _, mkt := range e.allMarketsCpy {
		mkt.OnMarketProbabilityOfTradingTauScalingUpdate(ctx, d)
	}
	e.npv.probabilityOfTradingTauScaling = d
	return nil
}

func (e *Engine) OnMarketMinProbabilityOfTradingForLPOrdersUpdate(ctx context.Context, d num.Decimal) error {
	if e.log.IsDebug() {
		e.log.Debug("update min probability of trading tau scaling",
			logging.Decimal("min-probability-of-trading-lp-orders", d),
		)
	}

	for _, mkt := range e.allMarketsCpy {
		mkt.OnMarketMinProbabilityOfTradingLPOrdersUpdate(ctx, d)
	}
	e.npv.minProbabilityOfTradingLPOrders = d
	return nil
}

func (e *Engine) OnMinLpStakeQuantumMultipleUpdate(ctx context.Context, d num.Decimal) error {
	if e.log.IsDebug() {
		e.log.Debug("update min lp stake quantum multiple",
			logging.Decimal("min-lp-stake-quantum-multiple", d),
		)
	}
	for _, mkt := range e.allMarketsCpy {
		mkt.OnMarketMinLpStakeQuantumMultipleUpdate(ctx, d)
	}
	e.npv.minLpStakeQuantumMultiple = d
	return nil
}

func (e *Engine) OnMarketCreationQuantumMultipleUpdate(ctx context.Context, d num.Decimal) error {
	if e.log.IsDebug() {
		e.log.Debug("update market creation quantum multiple",
			logging.Decimal("market-creation-quantum-multiple", d),
		)
	}
	e.npv.marketCreationQuantumMultiple = d
	return nil
}

func (e *Engine) OnMarketPartiesMaximumStopOrdersUpdate(ctx context.Context, u *num.Uint) error {
	if e.log.IsDebug() {
		e.log.Debug("update market parties maxiumum stop orders",
			logging.BigUint("value", u),
		)
	}
	e.npv.marketPartiesMaximumStopOrdersUpdate = u
	for _, mkt := range e.allMarketsCpy {
		mkt.OnMarketPartiesMaximumStopOrdersUpdate(ctx, u)
	}
	return nil
}

func (e *Engine) OnMaxPeggedOrderUpdate(ctx context.Context, max *num.Uint) error {
	if e.log.IsDebug() {
		e.log.Debug("update max pegged orders",
			logging.Uint64("max-pegged-orders", max.Uint64()),
		)
	}
	e.maxPeggedOrders = max.Uint64()
	return nil
}

func (e *Engine) MarketExists(market string) bool {
	_, ok := e.allMarkets[market]
	return ok
}

func (e *Engine) GetMarket(market string, settled bool) (types.Market, bool) {
	if mkt, ok := e.allMarkets[market]; ok {
		return mkt.IntoType(), true
	}
	// market wasn't found in the markets map, if a successor market was proposed after parent market
	// was settled/closed, then we should check the checkpoint states map for the parent market definition.
	if settled {
		if mcp, ok := e.marketCPStates[market]; ok && mcp.Market != nil {
			cpy := mcp.Market.DeepClone()
			return *cpy, true
		}
	}
	return types.Market{}, false
}

// GetEquityLikeShareForMarketAndParty return the equity-like shares of the given
// party in the given market. If the market doesn't exist, it returns false.
func (e *Engine) GetEquityLikeShareForMarketAndParty(market, party string) (num.Decimal, bool) {
	if mkt, ok := e.allMarkets[market]; ok {
		return mkt.GetEquityShares().SharesFromParty(party), true
	}
	return num.DecimalZero(), false
}

// GetMarketCounters returns the per-market counts used for gas estimation.
func (e *Engine) GetMarketCounters() map[string]*types.MarketCounters {
	counters := map[string]*types.MarketCounters{}
	for k, m := range e.allMarkets {
		counters[k] = m.GetMarketCounters()
	}
	return counters
}

func (e *Engine) GetMarketStats() map[string]*types.MarketStats {
	stats := map[string]*types.MarketStats{}
	for id, cm := range e.allMarkets {
		if s := cm.GetPartiesStats(); s != nil {
			stats[id] = s
		}
	}

	return stats
}

func (e *Engine) OnSuccessorMarketTimeWindowUpdate(ctx context.Context, window time.Duration) error {
	// change in succession window length
	delta := window - e.successorWindow
	if delta != 0 {
		for _, cpm := range e.marketCPStates {
			cpm.TTL = cpm.TTL.Add(delta)
		}
	}
	e.successorWindow = window
	return nil
}
