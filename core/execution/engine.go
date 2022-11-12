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

	"code.vegaprotocol.io/vega/core/assets"
	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/metrics"
	"code.vegaprotocol.io/vega/core/monitor"
	"code.vegaprotocol.io/vega/core/oracles"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/core/types/statevar"
	"code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/protos/vega"
)

//go:generate go run github.com/golang/mock/mockgen -destination mocks/mocks.go -package mocks code.vegaprotocol.io/vega/core/execution TimeService,Assets,StateVarEngine,Collateral,OracleEngine,EpochEngine

var (
	// ErrMarketDoesNotExist is returned when the market does not exist.
	ErrMarketDoesNotExist = errors.New("market does not exist")

	// ErrNoMarketID is returned when invalid (empty) market id was supplied during market creation.
	ErrNoMarketID = errors.New("no valid market id was supplied")

	// ErrInvalidOrderCancellation is returned when an incomplete order cancellation request is used.
	ErrInvalidOrderCancellation = errors.New("invalid order cancellation")
)

// TimeService ...

type TimeService interface {
	GetTimeNow() time.Time
}

// OracleEngine ...
type OracleEngine interface {
	ListensToSigners(oracles.OracleData) bool
	Subscribe(context.Context, oracles.OracleSpec, oracles.OnMatchedOracleData) (oracles.SubscriptionID, oracles.Unsubscriber)
	Unsubscribe(context.Context, oracles.SubscriptionID)
}

// Broker (no longer need to mock this, use the broker/mocks wrapper).
type Broker interface {
	Send(event events.Event)
	SendBatch(events []events.Event)
}

type Collateral interface {
	MarketCollateral
	AssetExists(string) bool
	CreateMarketAccounts(context.Context, string, string) (string, string, error)
}

type StateVarEngine interface {
	RegisterStateVariable(asset, market, name string, converter statevar.Converter, startCalculation func(string, statevar.FinaliseCalculation), trigger []statevar.EventType, result func(context.Context, statevar.StateVariableResult) error) error
	UnregisterStateVariable(asset, market string)
	NewEvent(asset, market string, eventType statevar.EventType)
	ReadyForTimeTrigger(asset, mktID string)
}

type Assets interface {
	Get(assetID string) (*assets.Asset, error)
}

type IDGenerator interface {
	NextID() string
}

// Engine is the execution engine.
type Engine struct {
	Config
	log *logging.Logger

	markets    map[string]*Market
	marketsCpy []*Market
	collateral Collateral
	assets     Assets

	broker                Broker
	timeService           TimeService
	stateVarEngine        StateVarEngine
	marketActivityTracker *MarketActivityTracker

	oracle OracleEngine

	npv netParamsValues

	snapshotSerialised    []byte
	newGeneratedProviders []types.StateProvider // new providers generated during the last state change

	// Map of all active snapshot providers that the execution engine has generated
	generatedProviders map[string]struct{}

	maxPeggedOrders        uint64
	totalPeggedOrdersCount int64
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
	ts TimeService,
	collateral Collateral,
	oracle OracleEngine,
	broker Broker,
	stateVarEngine StateVarEngine,
	marketActivityTracker *MarketActivityTracker,
	assets Assets,
) *Engine {
	// setup logger
	log = log.Named(namedLogger)
	log.SetLevel(executionConfig.Level.Get())
	e := &Engine{
		log:                   log,
		Config:                executionConfig,
		markets:               map[string]*Market{},
		timeService:           ts,
		collateral:            collateral,
		assets:                assets,
		broker:                broker,
		oracle:                oracle,
		npv:                   defaultNetParamsValues(),
		generatedProviders:    map[string]struct{}{},
		stateVarEngine:        stateVarEngine,
		marketActivityTracker: marketActivityTracker,
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

	return mkt.StartOpeningAuction(ctx)
}

// IsEligibleForProposerBonus checks if the given value is greater than that market quantum * quantum_multiplier.
func (e *Engine) IsEligibleForProposerBonus(marketID string, value *num.Uint) bool {
	if _, ok := e.markets[marketID]; !ok {
		return false
	}
	asset, err := e.markets[marketID].mkt.GetAsset()
	if err != nil {
		return false
	}
	quantum, err := e.collateral.GetAssetQuantum(asset)
	if err != nil {
		return false
	}
	return value.ToDecimal().GreaterThan(quantum.Mul(e.npv.marketCreationQuantumMultiple))
}

// SubmitMarket submits a new market configuration to the network.
func (e *Engine) SubmitMarket(ctx context.Context, marketConfig *types.Market, proposer string) error {
	return e.submitOrRestoreMarket(ctx, marketConfig, proposer, true)
}

// RestoreMarket restores a new market from proposal checkpoint.
func (e *Engine) RestoreMarket(ctx context.Context, marketConfig *types.Market) error {
	proposer := e.marketActivityTracker.GetProposer(marketConfig.ID)
	if len(proposer) == 0 {
		return ErrMarketDoesNotExist
	}
	return e.submitOrRestoreMarket(ctx, marketConfig, "", false)
}

func (e *Engine) submitOrRestoreMarket(ctx context.Context, marketConfig *types.Market, proposer string, isNewMarket bool) error {
	if e.log.IsDebug() {
		msg := "submit market"
		if !isNewMarket {
			msg = "restore market"
		}
		e.log.Debug(msg, logging.Market(*marketConfig))
	}

	if err := e.submitMarket(ctx, marketConfig); err != nil {
		return err
	}

	if isNewMarket {
		asset, err := marketConfig.GetAsset()
		if err != nil {
			e.log.Panic("failed to get asset from market config", logging.String("market", marketConfig.ID), logging.String("error", err.Error()))
		}
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

func (e *Engine) publishNewMarketInfos(ctx context.Context, mkt *Market) {
	// we send a market data event for this market when it's created so graphql does not fail
	e.broker.Send(events.NewMarketDataEvent(ctx, mkt.GetMarketData()))
	e.broker.Send(events.NewMarketCreatedEvent(ctx, *mkt.mkt))
	e.broker.Send(events.NewMarketUpdatedEvent(ctx, *mkt.mkt))
}

func (e *Engine) publishUpdateMarketInfos(ctx context.Context, mkt *Market) {
	// we send a market data event for this market when it's created so graphql does not fail
	e.broker.Send(events.NewMarketDataEvent(ctx, mkt.GetMarketData()))
	e.broker.Send(events.NewMarketUpdatedEvent(ctx, *mkt.mkt))
}

// submitMarket will submit a new market configuration to the network.
func (e *Engine) submitMarket(ctx context.Context, marketConfig *types.Market) error {
	if len(marketConfig.ID) == 0 {
		return ErrNoMarketID
	}

	now := e.timeService.GetTimeNow()

	// ensure the asset for this new market exists
	asset, err := marketConfig.GetAsset()
	if err != nil {
		return err
	}
	if !e.collateral.AssetExists(asset) {
		e.log.Error("unable to create a market with an invalid asset",
			logging.MarketID(marketConfig.ID),
			logging.AssetID(asset))
	}

	// create market auction state
	mas := monitor.NewAuctionState(marketConfig, now)
	ad, err := e.assets.Get(asset)
	if err != nil {
		e.log.Error("Failed to create a new market, unknown asset",
			logging.MarketID(marketConfig.ID),
			logging.String("asset-id", asset),
			logging.Error(err),
		)
		return err
	}
	mkt, err := NewMarket(
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

	// we ignore the response, this cannot fail as the asset
	// is already proven to exists a few line before
	_, _, _ = e.collateral.CreateMarketAccounts(ctx, marketConfig.ID, asset)

	if err := e.propagateInitialNetParams(ctx, mkt); err != nil {
		return err
	}

	return nil
}

func (e *Engine) propagateInitialNetParams(ctx context.Context, mkt *Market) error {
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
			mkt.matching.StopSnapshots()
			mkt.position.StopSnapshots()
			mkt.liquidity.StopSnapshots()
			mkt.tsCalc.StopSnapshots()

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
	idgen IDGenerator,
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
	idgen IDGenerator,
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
	idgen IDGenerator,
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
	idgen IDGenerator,
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
		if err != nil && err != ErrTradingNotAllowed {
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
	evts := make([]events.Event, 0, len(e.marketsCpy))
	for _, mkt := range e.marketsCpy {
		mkt := mkt
		closing := mkt.OnTick(ctx, t)
		if closing {
			e.log.Info("market is closed, removing from execution engine",
				logging.MarketID(mkt.GetID()))
			toDelete = append(toDelete, mkt.GetID())
		}
		evts = append(evts, events.NewMarketDataEvent(ctx, mkt.GetMarketData()))
	}
	e.broker.SendBatch(evts)

	for _, id := range toDelete {
		e.removeMarket(id)
	}

	timer.EngineTimeCounterAdd()
}

func (e *Engine) GetMarketState(mktID string) (types.MarketState, error) {
	mkt, ok := e.markets[mktID]
	if !ok {
		return types.MarketStateUnspecified, types.ErrInvalidMarketID
	}
	return mkt.GetMarketState(), nil
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
	// we make sure to update both the copy and the actual markets for snapshots
	// although we can most likely just update the market because we're already ensuring the nextMTM value
	// is set correctly when getting state
	for _, cpMkt := range e.marketsCpy {
		mkt := e.markets[cpMkt.mkt.ID]
		mkt.OnMarkPriceUpdateMaximumFrequency(ctx, d)
		cpMkt.nextMTM = mkt.nextMTM
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

func (e *Engine) OnMarketTargetStakeScalingFactorUpdate(_ context.Context, d num.Decimal) error {
	if e.log.IsDebug() {
		e.log.Debug("update market stake scaling factor",
			logging.Decimal("scaling-factor", d),
		)
	}

	for _, mkt := range e.marketsCpy {
		if err := mkt.OnMarketTargetStakeScalingFactorUpdate(d); err != nil {
			return err
		}
	}
	return nil
}

func (e *Engine) OnMarketTargetStakeTimeWindowUpdate(_ context.Context, d time.Duration) error {
	if e.log.IsDebug() {
		e.log.Debug("update market stake time window",
			logging.Duration("time-window", d),
		)
	}

	for _, mkt := range e.marketsCpy {
		mkt.OnMarketTargetStakeTimeWindowUpdate(d)
	}
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

func (e *Engine) OnMarketLiquidityTargetStakeTriggeringRatio(ctx context.Context, d num.Decimal) error {
	if e.log.IsDebug() {
		e.log.Debug("update target stake triggering ratio",
			logging.Decimal("max-liquidity-fee", d),
		)
	}

	for _, mkt := range e.marketsCpy {
		mkt.OnMarketLiquidityTargetStakeTriggeringRatio(ctx, d)
	}
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

func (e *Engine) GetMarket(market string) (types.Market, bool) {
	mkt, ok := e.markets[market]
	if !ok {
		return types.Market{}, false
	}
	return mkt.IntoType(), true
}

// GetEquityLikeShareForMarketAndParty return the equity-like shares of the given
// party in the given market. If the market doesn't exist, it returns false.
func (e *Engine) GetEquityLikeShareForMarketAndParty(market, party string) (num.Decimal, bool) {
	mkt, ok := e.markets[market]
	if !ok {
		return num.DecimalZero(), false
	}
	return mkt.equityShares.SharesFromParty(party), true
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
