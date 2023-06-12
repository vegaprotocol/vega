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
	"sort"
	"time"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/execution/common"
	"code.vegaprotocol.io/vega/core/execution/future"
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
)

// Engine is the execution engine.
type Engine struct {
	Config
	log *logging.Logger

	markets    map[string]*future.Market
	marketsCpy []*future.Market
	collateral common.Collateral
	assets     common.Assets

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
}

type netParamsValues struct {
	shapesMaxSize                   int64
	feeDistributionTimeStep         time.Duration
	marketValueWindowLength         time.Duration
	suppliedStakeToObligationFactor num.Decimal
	infrastructureFee               num.Decimal
	makerFee                        num.Decimal
	scalingFactors                  *types.ScalingFactors
	maxLiquidityFee                 num.Decimal
	bondPenaltyFactor               num.Decimal
	auctionMinDuration              time.Duration
	probabilityOfTradingTauScaling  num.Decimal
	minProbabilityOfTradingLPOrders num.Decimal
	minLpStakeQuantumMultiple       num.Decimal
	marketCreationQuantumMultiple   num.Decimal
	markPriceUpdateMaximumFrequency time.Duration
}

func defaultNetParamsValues() netParamsValues {
	return netParamsValues{
		shapesMaxSize:                   -1,
		feeDistributionTimeStep:         -1,
		marketValueWindowLength:         -1,
		suppliedStakeToObligationFactor: num.DecimalFromInt64(-1),
		infrastructureFee:               num.DecimalFromInt64(-1),
		makerFee:                        num.DecimalFromInt64(-1),
		scalingFactors:                  nil,
		maxLiquidityFee:                 num.DecimalFromInt64(-1),
		bondPenaltyFactor:               num.DecimalFromInt64(-1),
		auctionMinDuration:              -1,
		probabilityOfTradingTauScaling:  num.DecimalFromInt64(-1),
		minProbabilityOfTradingLPOrders: num.DecimalFromInt64(-1),
		minLpStakeQuantumMultiple:       num.DecimalFromInt64(-1),
		marketCreationQuantumMultiple:   num.DecimalFromInt64(-1),
		markPriceUpdateMaximumFrequency: 5 * time.Second, // default is 5 seconds, should come from net params though
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
) *Engine {
	// setup logger
	log = log.Named(namedLogger)
	log.SetLevel(executionConfig.Level.Get())
	e := &Engine{
		log:                   log,
		Config:                executionConfig,
		markets:               map[string]*future.Market{},
		timeService:           ts,
		collateral:            collateral,
		assets:                assets,
		broker:                broker,
		oracle:                oracle,
		npv:                   defaultNetParamsValues(),
		generatedProviders:    map[string]struct{}{},
		stateVarEngine:        stateVarEngine,
		marketActivityTracker: marketActivityTracker,
		marketCPStates:        map[string]*types.CPMarketState{},
		successors:            map[string][]string{},
		isSuccessor:           map[string]string{},
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
	for _, mkt := range e.marketsCpy {
		mkt.ReloadConf(e.Matching, e.Risk, e.Position, e.Settlement, e.Fee)
	}
}

func (e *Engine) SpotsMarketsEnabled() bool {
	// TODO replace with real implementation
	return false
}

func (e *Engine) Hash() []byte {
	e.log.Debug("hashing markets")

	hashes := make([]string, 0, len(e.marketsCpy))
	for _, m := range e.marketsCpy {
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

	mkt, ok := e.markets[marketID]
	if !ok {
		return ErrMarketDoesNotExist
	}

	if err := mkt.Reject(ctx); err != nil {
		return err
	}

	e.removeMarket(marketID)
	// a market rejection can have a knock-on effect for proposed markets which were supposed to succeed this market
	// they should be purged here, and @TODO handle any errors
	if successors, ok := e.successors[marketID]; ok {
		delete(e.successors, marketID)
		for _, sID := range successors {
			_ = e.RejectMarket(ctx, sID)
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

	mkt, ok := e.markets[marketID]
	if !ok {
		return ErrMarketDoesNotExist
	}
	err := mkt.StartOpeningAuction(ctx)
	if err != nil {
		return err
	}
	// proposal was accepted, the parent market should not be checked here, it'll reach the correct state
	// either before this (in case of restore) or when calling SucceedMarket. If neither are called,
	// this proposal doesn't need to have the ParentMarketID checked.
	return nil
}

func (e *Engine) SucceedMarket(ctx context.Context, successor, parent string) error {
	return e.succeedOrRestore(ctx, successor, parent, false)
}

func (e *Engine) restoreOwnState(ctx context.Context, mID string) (bool, error) {
	mkt, ok := e.markets[mID]
	if !ok {
		return false, ErrMarketDoesNotExist
	}
	if state, ok := e.marketCPStates[mID]; ok {
		// set ELS state and the like
		mkt.InheritParent(ctx, state)
		// if there was state of the market to restore, then check if this is a successor market
		if pid := mkt.GetParentMarketID(); len(pid) > 0 {
			// mark parent market as being succeeded
			if pMkt, ok := e.markets[pid]; ok {
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
	mkt, ok := e.markets[successor]
	if !ok {
		// this can happen if a proposal vote closed, but the proposal had an enactment time in the future.
		// Between the proposal being accepted and enacted, another proposal may be enacted first.
		// Whenever the parent is succeeded, all other markets are rejected and removed from the map here,
		// nevertheless the proposal is still valid, and updated by the governance engine.
		return ErrMarketDoesNotExist
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
		mkt.ResetParentIDAndInsurancePoolFraction()
		return nil
	}
	// if parent market is active, mark as succeeded
	if pmo, ok := e.markets[parent]; ok {
		// succeeding a parent market before it was enacted is not allowed
		if pmo.Mkt().State == types.MarketStateProposed {
			e.RejectMarket(ctx, successor)
			return ErrParentMarketNotEnactedYet
		}
	}
	// successor market set up accordingly, clean up the state
	// first reject all pending successors proposed for the same parent
	return nil
}

// IsEligibleForProposerBonus checks if the given value is greater than that market quantum * quantum_multiplier.
func (e *Engine) IsEligibleForProposerBonus(marketID string, value *num.Uint) bool {
	if _, ok := e.markets[marketID]; !ok {
		return false
	}
	quantum, err := e.collateral.GetAssetQuantum(e.markets[marketID].GetSettlementAsset())
	if err != nil {
		return false
	}
	return value.ToDecimal().GreaterThan(quantum.Mul(e.npv.marketCreationQuantumMultiple))
}

// SubmitMarket submits a new market configuration to the network.
func (e *Engine) SubmitMarket(ctx context.Context, marketConfig *types.Market, proposer string, oos time.Time) error {
	return e.submitOrRestoreMarket(ctx, marketConfig, proposer, true, oos)
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
	// attempt to restore market state. The restoreOwnState call handles both parent and successor markets
	ok, err := e.restoreOwnState(ctx, marketConfig.ID)
	if ok || err != nil {
		return err
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
		asset := assets[0]
		e.marketActivityTracker.MarketProposed(asset, marketConfig.ID, proposer)
	}

	// keep state in pending, opening auction is triggered when proposal is enacted
	mkt := e.markets[marketConfig.ID]

	e.publishNewMarketInfos(ctx, mkt)
	return nil
}

// UpdateMarket will update an existing market configuration.
func (e *Engine) UpdateMarket(ctx context.Context, marketConfig *types.Market) error {
	e.log.Info("update market", logging.Market(*marketConfig))

	mkt := e.markets[marketConfig.ID]

	if err := mkt.Update(ctx, marketConfig, e.oracle); err != nil {
		return err
	}

	e.publishUpdateMarketInfos(ctx, mkt)

	return nil
}

func (e *Engine) publishNewMarketInfos(ctx context.Context, mkt *future.Market) {
	// send a market data event for this market when it's created so graphql does not fail
	e.broker.Send(events.NewMarketDataEvent(ctx, mkt.GetMarketData()))
	e.broker.Send(events.NewMarketCreatedEvent(ctx, *mkt.Mkt()))
	e.broker.Send(events.NewMarketUpdatedEvent(ctx, *mkt.Mkt()))
}

func (e *Engine) publishUpdateMarketInfos(ctx context.Context, mkt *future.Market) {
	// send a market data event for this market when it's created so graphql does not fail
	e.broker.Send(events.NewMarketDataEvent(ctx, mkt.GetMarketData()))
	e.broker.Send(events.NewMarketUpdatedEvent(ctx, *mkt.Mkt()))
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
	)
	if err != nil {
		e.log.Error("failed to instantiate market",
			logging.MarketID(marketConfig.ID),
			logging.Error(err),
		)
		return err
	}

	e.markets[marketConfig.ID] = mkt
	e.marketsCpy = append(e.marketsCpy, mkt)

	// ignore the response, this cannot fail as the asset
	// is already proven to exists a few line before
	_, _, _ = e.collateral.CreateMarketAccounts(ctx, marketConfig.ID, asset)

	return e.propagateInitialNetParams(ctx, mkt)
}

func (e *Engine) propagateInitialNetParams(ctx context.Context, mkt *future.Market) error {
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
	if e.npv.shapesMaxSize != -1 {
		if err := mkt.OnMarketLiquidityProvisionShapesMaxSizeUpdate(e.npv.shapesMaxSize); err != nil {
			return err
		}
	}

	if !e.npv.infrastructureFee.Equal(num.DecimalFromInt64(-1)) {
		if err := mkt.OnFeeFactorsInfrastructureFeeUpdate(ctx, e.npv.infrastructureFee); err != nil {
			return err
		}
	}

	if !e.npv.makerFee.Equal(num.DecimalFromInt64(-1)) {
		if err := mkt.OnFeeFactorsMakerFeeUpdate(ctx, e.npv.makerFee); err != nil {
			return err
		}
	}

	if e.npv.scalingFactors != nil {
		if err := mkt.OnMarginScalingFactorsUpdate(ctx, e.npv.scalingFactors); err != nil {
			return err
		}
	}

	if e.npv.feeDistributionTimeStep != -1 {
		mkt.OnMarketLiquidityProvidersFeeDistribitionTimeStep(e.npv.feeDistributionTimeStep)
	}

	if e.npv.marketValueWindowLength != -1 {
		mkt.OnMarketValueWindowLengthUpdate(e.npv.marketValueWindowLength)
	}

	if !e.npv.suppliedStakeToObligationFactor.Equal(num.DecimalFromInt64(-1)) {
		mkt.OnSuppliedStakeToObligationFactorUpdate(e.npv.suppliedStakeToObligationFactor)
	}
	if !e.npv.bondPenaltyFactor.Equal(num.DecimalFromInt64(-1)) {
		mkt.BondPenaltyFactorUpdate(ctx, e.npv.bondPenaltyFactor)
	}

	if !e.npv.maxLiquidityFee.Equal(num.DecimalFromInt64(-1)) {
		mkt.OnMarketLiquidityMaximumLiquidityFeeFactorLevelUpdate(e.npv.maxLiquidityFee)
	}
	if e.npv.markPriceUpdateMaximumFrequency > 0 {
		mkt.OnMarkPriceUpdateMaximumFrequency(ctx, e.npv.markPriceUpdateMaximumFrequency)
	}
	return nil
}

func (e *Engine) removeMarket(mktID string) {
	e.log.Debug("removing market", logging.String("id", mktID))

	delete(e.markets, mktID)
	for i, mkt := range e.marketsCpy {
		if mkt.GetID() == mktID {
			mkt.StopSnapshots()

			copy(e.marketsCpy[i:], e.marketsCpy[i+1:])
			e.marketsCpy[len(e.marketsCpy)-1] = nil
			e.marketsCpy = e.marketsCpy[:len(e.marketsCpy)-1]
			e.marketActivityTracker.RemoveMarket(mktID)
			e.log.Debug("removed in total", logging.String("id", mktID))
			return
		}
	}
}

func (e *Engine) peggedOrderCountUpdated(added int64) {
	e.totalPeggedOrdersCount += added
}

func (e *Engine) canSubmitPeggedOrder() bool {
	return uint64(e.totalPeggedOrdersCount) < e.maxPeggedOrders
}

// SubmitOrder checks the incoming order and submits it to a Vega market.
func (e *Engine) SubmitOrder(
	ctx context.Context,
	submission *types.OrderSubmission,
	party string,
	idgen common.IDGenerator,
	orderID string,
) (confirmation *types.OrderConfirmation, returnedErr error) {
	timer := metrics.NewTimeCounter(submission.MarketID, "execution", "SubmitOrder")
	defer func() {
		timer.EngineTimeCounterAdd()
	}()

	if e.log.IsDebug() {
		e.log.Debug("submit order", logging.OrderSubmission(submission))
	}

	mkt, ok := e.markets[submission.MarketID]
	if !ok {
		return nil, types.ErrInvalidMarketID
	}

	if submission.PeggedOrder != nil && !e.canSubmitPeggedOrder() {
		return nil, &types.ErrTooManyPeggedOrders
	}

	metrics.OrderGaugeAdd(1, submission.MarketID)
	conf, err := mkt.SubmitOrderWithIDGeneratorAndOrderID(
		ctx, submission, party, idgen, orderID)
	if err != nil {
		return nil, err
	}

	e.decrementOrderGaugeMetrics(submission.MarketID, conf.Order, conf.PassiveOrdersAffected)

	return conf, nil
}

// AmendOrder takes order amendment details and attempts to amend the order
// if it exists and is in a editable state.
func (e *Engine) AmendOrder(
	ctx context.Context,
	amendment *types.OrderAmendment,
	party string,
	idgen common.IDGenerator,
) (confirmation *types.OrderConfirmation, returnedErr error) {
	timer := metrics.NewTimeCounter(amendment.MarketID, "execution", "AmendOrder")
	defer func() {
		timer.EngineTimeCounterAdd()
	}()

	if e.log.IsDebug() {
		e.log.Debug("amend order", logging.OrderAmendment(amendment))
	}

	mkt, ok := e.markets[amendment.MarketID]
	if !ok {
		return nil, types.ErrInvalidMarketID
	}

	conf, err := mkt.AmendOrderWithIDGenerator(ctx, amendment, party, idgen)
	if err != nil {
		return nil, err
	}

	e.decrementOrderGaugeMetrics(amendment.MarketID, conf.Order, conf.PassiveOrdersAffected)

	return conf, nil
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

func (e *Engine) cancelOrder(
	ctx context.Context,
	party, market, orderID string,
	idgen common.IDGenerator,
) ([]*types.OrderCancellationConfirmation, error) {
	mkt, ok := e.markets[market]
	if !ok {
		return nil, types.ErrInvalidMarketID
	}
	conf, err := mkt.CancelOrderWithIDGenerator(ctx, party, orderID, idgen)
	if err != nil {
		return nil, err
	}
	if conf.Order.Status == types.OrderStatusCancelled {
		metrics.OrderGaugeAdd(-1, market)
	}
	return []*types.OrderCancellationConfirmation{conf}, nil
}

func (e *Engine) cancelOrderByMarket(ctx context.Context, party, market string) ([]*types.OrderCancellationConfirmation, error) {
	mkt, ok := e.markets[market]
	if !ok {
		return nil, types.ErrInvalidMarketID
	}
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

func (e *Engine) cancelAllPartyOrders(ctx context.Context, party string) ([]*types.OrderCancellationConfirmation, error) {
	confirmations := []*types.OrderCancellationConfirmation{}

	for _, mkt := range e.marketsCpy {
		confs, err := mkt.CancelAllOrders(ctx, party)
		if err != nil && err != common.ErrTradingNotAllowed {
			return nil, err
		}
		confirmations = append(confirmations, confs...)
		var confirmed int
		for _, conf := range confs {
			if conf.Order.Status == types.OrderStatusCancelled {
				confirmed++
			}
		}
		metrics.OrderGaugeAdd(-confirmed, mkt.GetID())
	}
	return confirmations, nil
}

func (e *Engine) SubmitLiquidityProvision(
	ctx context.Context,
	sub *types.LiquidityProvisionSubmission,
	party, deterministicID string,
) (returnedErr error) {
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

	mkt, ok := e.markets[sub.MarketID]
	if !ok {
		return types.ErrInvalidMarketID
	}

	return mkt.SubmitLiquidityProvision(ctx, sub, party, deterministicID)
}

func (e *Engine) AmendLiquidityProvision(ctx context.Context, lpa *types.LiquidityProvisionAmendment, party string,
	deterministicID string,
) (returnedErr error) {
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

	mkt, ok := e.markets[lpa.MarketID]
	if !ok {
		return types.ErrInvalidMarketID
	}

	return mkt.AmendLiquidityProvision(ctx, lpa, party, deterministicID)
}

func (e *Engine) CancelLiquidityProvision(ctx context.Context, cancel *types.LiquidityProvisionCancellation, party string) (returnedErr error) {
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

	mkt, ok := e.markets[cancel.MarketID]
	if !ok {
		return types.ErrInvalidMarketID
	}

	return mkt.CancelLiquidityProvision(ctx, cancel, party)
}

func (e *Engine) OnTick(ctx context.Context, t time.Time) {
	timer := metrics.NewTimeCounter("-", "execution", "OnTick")

	e.log.Debug("updating engine on new time update")

	// notify markets of the time expiration
	toDelete := []string{}
	parentStates := e.getParentStates()
	evts := make([]events.Event, 0, len(e.marketsCpy))
	for _, mkt := range e.marketsCpy {
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
			if pmkt, ok := e.markets[pid]; ok {
				pmkt.SetSucceeded()
			} else {
				asset := mkt.GetSettlementAsset()
				// clear parent market insurance pool
				if clearTransfers, _ := e.collateral.ClearInsurancepool(ctx, pid, asset, true); len(clearTransfers) > 0 {
					e.broker.Send(events.NewLedgerMovements(ctx, clearTransfers))
				}
			}
			// reject other pending successors
			for _, sid := range e.successors[pid] {
				delete(e.isSuccessor, sid)
				if id == sid {
					continue
				}
				e.RejectMarket(ctx, sid)
			}
			// remove data used to indicate that the parent market has pending successors
			delete(e.successors, pid)
			delete(e.marketCPStates, pid)
		} else if isSuccessor {
			// this call can be made even if the market has left opening auction, but checking this here, too, is better than
			// relying on how this is implemented
			mkt.RollbackInherit(ctx)
		}
		if _, ok := e.successors[id]; ok || !mkt.IsSucceeded() {
			// update the market state used to set the successor market accordingly
			// if the market was closed, the checkpoint data will include the full market definition
			cps := mkt.GetCPState()
			// set until what time this state is considered valid.
			cps.TTL = t.Add(e.successorWindow)
			e.marketCPStates[id] = cps
		} else {
			// just in case it's still around -> remove the state, this would mean there is no pending successor, and the market is
			// marked as succeeded, this state should not be here
			delete(e.marketCPStates, id)
		}
		evts = append(evts, events.NewMarketDataEvent(ctx, mkt.GetMarketData()))
	}
	e.broker.SendBatch(evts)

	for _, id := range toDelete {
		e.removeMarket(id)
	}
	// clear slice
	toDelete = toDelete[:]
	// find state that should expire
	for id, cpm := range e.marketCPStates {
		if !cpm.TTL.After(t) {
			toDelete = append(toDelete, id)
			assets, _ := cpm.Market.GetAssets()
			if clearTransfers, _ := e.collateral.ClearInsurancepool(ctx, id, assets[0], true); len(clearTransfers) > 0 {
				e.broker.Send(events.NewLedgerMovements(ctx, clearTransfers))
			}
		}
	}
	for _, id := range toDelete {
		delete(e.marketCPStates, id)
		if ss, ok := e.successors[id]; ok {
			// parent market expired, remove parent ID
			for _, s := range ss {
				delete(e.isSuccessor, s)
				if mkt, ok := e.markets[s]; ok {
					mkt.ResetParentIDAndInsurancePoolFraction()
				}
			}
		}
		delete(e.successors, id)
	}

	timer.EngineTimeCounterAdd()
}

func (e *Engine) getParentStates() map[string]*types.CPMarketState {
	// all successor markets need to have a reference to the parent state
	states := make(map[string]*types.CPMarketState, len(e.isSuccessor))
	// for each parent market, get the successors
	for pid, successors := range e.successors {
		state, sok := e.marketCPStates[pid]
		if !sok {
			if pmkt, ok := e.markets[pid]; ok {
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
	for _, mkt := range e.marketsCpy {
		mkt.BlockEnd(ctx)
	}
}

func (e *Engine) GetMarketState(mktID string) (types.MarketState, error) {
	mkt, ok := e.markets[mktID]
	if !ok {
		return types.MarketStateUnspecified, types.ErrInvalidMarketID
	}
	return mkt.GetMarketState(), nil
}

func (e *Engine) IsSucceeded(mktID string) bool {
	if mkt, ok := e.markets[mktID]; ok {
		return mkt.IsSucceeded()
	}
	// checking marketCPStates is pointless. The parent market could not be found to validate the proposal, so it will be rejected outright
	// if the market is no longer in e.markets, it will be set in marketCPStates, and therefore the successor proposal must be accepted.
	return false
}

func (e *Engine) GetMarketData(mktID string) (types.MarketData, error) {
	mkt, ok := e.markets[mktID]
	if !ok {
		return types.MarketData{}, types.ErrInvalidMarketID
	}
	return mkt.GetMarketData(), nil
}

func (e *Engine) OnMarketAuctionMinimumDurationUpdate(ctx context.Context, d time.Duration) error {
	for _, mkt := range e.markets {
		mkt.OnMarketAuctionMinimumDurationUpdate(ctx, d)
	}
	e.npv.auctionMinDuration = d
	return nil
}

func (e *Engine) OnMarkPriceUpdateMaximumFrequency(ctx context.Context, d time.Duration) error {
	for _, mkt := range e.markets {
		mkt.OnMarkPriceUpdateMaximumFrequency(ctx, d)
	}
	e.npv.markPriceUpdateMaximumFrequency = d
	return nil
}

func (e *Engine) OnMarketLiquidityBondPenaltyUpdate(ctx context.Context, d num.Decimal) error {
	if e.log.IsDebug() {
		e.log.Debug("update market liquidity bond penalty",
			logging.Decimal("bond-penalty-factor", d),
		)
	}

	for _, mkt := range e.markets {
		mkt.BondPenaltyFactorUpdate(ctx, d)
	}

	e.npv.bondPenaltyFactor = d

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
	for _, mkt := range e.marketsCpy {
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

	for _, mkt := range e.marketsCpy {
		if err := mkt.OnFeeFactorsMakerFeeUpdate(ctx, d); err != nil {
			return err
		}
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

	for _, mkt := range e.marketsCpy {
		if err := mkt.OnFeeFactorsInfrastructureFeeUpdate(ctx, d); err != nil {
			return err
		}
	}

	e.npv.infrastructureFee = d

	return nil
}

func (e *Engine) OnSuppliedStakeToObligationFactorUpdate(_ context.Context, d num.Decimal) error {
	if e.log.IsDebug() {
		e.log.Debug("update supplied stake to obligation factor",
			logging.Decimal("factor", d),
		)
	}

	for _, mkt := range e.marketsCpy {
		mkt.OnSuppliedStakeToObligationFactorUpdate(d)
	}

	e.npv.suppliedStakeToObligationFactor = d

	return nil
}

func (e *Engine) OnMarketValueWindowLengthUpdate(_ context.Context, d time.Duration) error {
	if e.log.IsDebug() {
		e.log.Debug("update market value window length",
			logging.Duration("window-length", d),
		)
	}

	for _, mkt := range e.marketsCpy {
		mkt.OnMarketValueWindowLengthUpdate(d)
	}

	e.npv.marketValueWindowLength = d

	return nil
}

func (e *Engine) OnMarketLiquidityProvidersFeeDistributionTimeStep(_ context.Context, d time.Duration) error {
	if e.log.IsDebug() {
		e.log.Debug("update liquidity providers fee distribution time step",
			logging.Duration("time-window", d),
		)
	}

	for _, mkt := range e.marketsCpy {
		mkt.OnMarketLiquidityProvidersFeeDistribitionTimeStep(d)
	}

	e.npv.feeDistributionTimeStep = d

	return nil
}

func (e *Engine) OnMarketLiquidityProvisionShapesMaxSizeUpdate(
	_ context.Context, v int64,
) error {
	if e.log.IsDebug() {
		e.log.Debug("update liquidity provision max shape",
			logging.Int64("max-shape", v),
		)
	}

	for _, mkt := range e.marketsCpy {
		_ = mkt.OnMarketLiquidityProvisionShapesMaxSizeUpdate(v)
	}

	e.npv.shapesMaxSize = v

	return nil
}

func (e *Engine) OnMarketLiquidityMaximumLiquidityFeeFactorLevelUpdate(
	_ context.Context, d num.Decimal,
) error {
	if e.log.IsDebug() {
		e.log.Debug("update liquidity provision max liquidity fee factor",
			logging.Decimal("max-liquidity-fee", d),
		)
	}

	for _, mkt := range e.marketsCpy {
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

	for _, mkt := range e.marketsCpy {
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

	for _, mkt := range e.marketsCpy {
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

	for _, mkt := range e.marketsCpy {
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
	_, ok := e.markets[market]
	return ok
}

func (e *Engine) GetMarket(market string, settled bool) (types.Market, bool) {
	if mkt, ok := e.markets[market]; ok {
		return mkt.IntoType(), true
	}
	// market wasn't found in the markets map, if a successor market was proposed after parent market
	// was settled/closed, then check the checkpoint states map for the parent market definition.
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
	mkt, ok := e.markets[market]
	if !ok {
		return num.DecimalZero(), false
	}
	return mkt.GetEquityShares().SharesFromParty(party), true
}

func (e *Engine) GetAsset(assetID string) (types.Asset, bool) {
	a, err := e.assets.Get(assetID)
	if err != nil {
		return types.Asset{}, false
	}
	return *a.ToAssetType(), true
}

// GetMarketCounters returns the per-market counts used for gas estimation.
func (e *Engine) GetMarketCounters() map[string]*types.MarketCounters {
	counters := map[string]*types.MarketCounters{}
	for k, m := range e.markets {
		counters[k] = &types.MarketCounters{
			PeggedOrderCounter:  m.GetTotalPeggedOrderCount(),
			OrderbookLevelCount: m.GetTotalOrderBookLevelCount(),
			PositionCount:       m.GetTotalOpenPositionCount(),
			LPShapeCount:        m.GetTotalLPShapeCount(),
		}
	}
	return counters
}

func (e *Engine) OnSuccessorMarketTimeWindowUpdate(ctx context.Context, window time.Duration) error {
	// change in succession window length
	delta := window - e.successorWindow
	if delta != 0 {
		torm := []string{}
		now := e.timeService.GetTimeNow()
		for id, cpm := range e.marketCPStates {
			cpm.TTL = cpm.TTL.Add(delta)
			if cpm.TTL.After(now) {
				continue
			}
			torm = append(torm, id)
		}
		for _, id := range torm {
			delete(e.marketCPStates, id)
		}
	}
	e.successorWindow = window
	return nil
}
