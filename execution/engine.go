package execution

import (
	"context"
	"errors"
	"sort"
	"strconv"
	"time"

	"code.vegaprotocol.io/protos/vega"
	"code.vegaprotocol.io/vega/assets"
	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/metrics"
	"code.vegaprotocol.io/vega/monitor"
	"code.vegaprotocol.io/vega/oracles"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"
	"code.vegaprotocol.io/vega/types/statevar"
)

var (
	// ErrMarketDoesNotExist is returned when the market does not exist.
	ErrMarketDoesNotExist = errors.New("market does not exist")

	// ErrNoMarketID is returned when invalid (empty) market id was supplied during market creation.
	ErrNoMarketID = errors.New("no valid market id was supplied")

	// ErrInvalidOrderCancellation is returned when an incomplete order cancellation request is used.
	ErrInvalidOrderCancellation = errors.New("invalid order cancellation")
)

// TimeService ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/time_service_mock.go -package mocks code.vegaprotocol.io/vega/execution TimeService
type TimeService interface {
	GetTimeNow() time.Time
	NotifyOnTick(f func(context.Context, time.Time))
}

// OracleEngine ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/oracle_engine_mock.go -package mocks code.vegaprotocol.io/vega/execution OracleEngine
type OracleEngine interface {
	Subscribe(context.Context, oracles.OracleSpec, oracles.OnMatchedOracleData) oracles.SubscriptionID
	Unsubscribe(context.Context, oracles.SubscriptionID)
}

// Broker (no longer need to mock this, use the broker/mocks wrapper).
type Broker interface {
	Send(event events.Event)
	SendBatch(events []events.Event)
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/collateral_mock.go -package mocks code.vegaprotocol.io/vega/execution Collateral
type Collateral interface {
	MarketCollateral
	AssetExists(string) bool
	CreateMarketAccounts(context.Context, string, string) (string, string, error)
	OnChainTimeUpdate(context.Context, time.Time)
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/state_var_engine_mock.go -package mocks code.vegaprotocol.io/vega/execution StateVarEngine
type StateVarEngine interface {
	RegisterStateVariable(asset, market, name string, converter statevar.Converter, startCalculation func(string, statevar.FinaliseCalculation), trigger []statevar.StateVarEventType, result func(context.Context, statevar.StateVariableResult) error) error
	NewEvent(asset, market string, eventType statevar.StateVarEventType)
	ReadyForTimeTrigger(asset, mktID string)
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/assets_mock.go -package mocks code.vegaprotocol.io/vega/execution Assets
type Assets interface {
	Get(assetID string) (*assets.Asset, error)
}

// Engine is the execution engine.
type Engine struct {
	Config
	log *logging.Logger

	markets    map[string]*Market
	marketsCpy []*Market
	collateral Collateral
	idgen      *IDgenerator
	assets     Assets

	broker         Broker
	time           TimeService
	stateVarEngine StateVarEngine
	feesTracker    *FeesTracker
	marketTracker  *MarketTracker

	oracle OracleEngine

	npv netParamsValues

	// Snapshot
	snapshotSerialised    []byte
	snapshotHash          []byte
	newGeneratedProviders []types.StateProvider // new providers generated during the last state change

	// Map of all active snapshot providers that the execution engine has generated
	generatedProviders map[string]struct{}
}

type netParamsValues struct {
	shapesMaxSize                   int64
	feeDistributionTimeStep         time.Duration
	timeWindowUpdate                time.Duration
	targetStakeScalingFactor        num.Decimal
	marketValueWindowLength         time.Duration
	suppliedStakeToObligationFactor num.Decimal
	infrastructureFee               num.Decimal
	makerFee                        num.Decimal
	scalingFactors                  *types.ScalingFactors
	maxLiquidityFee                 num.Decimal
	bondPenaltyFactor               num.Decimal
	targetStakeTriggeringRatio      num.Decimal
	auctionMinDuration              time.Duration
	probabilityOfTradingTauScaling  num.Decimal
	minProbabilityOfTradingLPOrders num.Decimal
	minLpStakeQuantumMultiple       num.Decimal
}

func defaultNetParamsValues() netParamsValues {
	return netParamsValues{
		shapesMaxSize:                   -1,
		feeDistributionTimeStep:         -1,
		timeWindowUpdate:                -1,
		targetStakeScalingFactor:        num.DecimalFromInt64(-1),
		marketValueWindowLength:         -1,
		suppliedStakeToObligationFactor: num.DecimalFromInt64(-1),
		infrastructureFee:               num.DecimalFromInt64(-1),
		makerFee:                        num.DecimalFromInt64(-1),
		scalingFactors:                  nil,
		maxLiquidityFee:                 num.DecimalFromInt64(-1),
		bondPenaltyFactor:               num.DecimalFromInt64(-1),
		targetStakeTriggeringRatio:      num.DecimalFromInt64(-1),
		auctionMinDuration:              -1,
		probabilityOfTradingTauScaling:  num.DecimalFromInt64(-1),
		minProbabilityOfTradingLPOrders: num.DecimalFromInt64(-1),
		minLpStakeQuantumMultiple:       num.DecimalFromInt64(-1),
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
	feesTracker *FeesTracker,
	marketTracker *MarketTracker,
	assets Assets,
) *Engine {
	// setup logger
	log = log.Named(namedLogger)
	log.SetLevel(executionConfig.Level.Get())
	e := &Engine{
		log:                log,
		Config:             executionConfig,
		markets:            map[string]*Market{},
		time:               ts,
		collateral:         collateral,
		assets:             assets,
		idgen:              NewIDGen(),
		broker:             broker,
		oracle:             oracle,
		npv:                defaultNetParamsValues(),
		generatedProviders: map[string]struct{}{},
		stateVarEngine:     stateVarEngine,
		feesTracker:        feesTracker,
		marketTracker:      marketTracker,
	}

	// Add time change event handler
	e.time.NotifyOnTick(e.onChainTimeUpdate)

	// set the eligibility for proposer bonus checker
	marketTracker.SetEligibilityChecker(e)

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

	hashes := make([]string, 0, len(e.markets))
	for _, m := range e.markets {
		hash := m.Hash()
		e.log.Debug("market app state hash", logging.Hash(hash), logging.String("market-id", m.GetID()))
		hashes = append(hashes, string(hash))
	}

	sort.Strings(hashes)
	bytes := []byte{}
	for _, h := range hashes {
		bytes = append(bytes, []byte(h)...)
	}
	return crypto.Hash(bytes)
}

func (e *Engine) getFakeTickSize(decimalPlaces uint64) string {
	return num.MustDecimalFromString("1e-" + strconv.Itoa(int(decimalPlaces))).String()
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
	return value.ToDecimal().GreaterThan(quantum.ToDecimal().Mul(e.npv.minLpStakeQuantumMultiple))
}

// SubmitMarketWithLiquidityProvision is submitting a market through
// the usual governance process.
func (e *Engine) SubmitMarketWithLiquidityProvision(ctx context.Context, marketConfig *types.Market, lp *types.LiquidityProvisionSubmission, party, lpID string) error {
	if e.log.IsDebug() {
		e.log.Debug("submit market with liquidity provision",
			logging.Market(*marketConfig),
			logging.LiquidityProvisionSubmission(*lp),
			logging.PartyID(party),
			logging.LiquidityID(lpID),
		)
	}

	if err := e.submitMarket(ctx, marketConfig); err != nil {
		return err
	}

	mkt := e.markets[marketConfig.ID]
	e.marketTracker.MarketProposed(marketConfig.ID, party)

	// publish market data anyway initially
	e.publishMarketInfos(ctx, mkt)

	// now we try to submit the liquidity
	if err := mkt.SubmitLiquidityProvision(ctx, lp, party, lpID); err != nil {
		e.removeMarket(marketConfig.ID)
		return err
	}

	return nil
}

// SubmitMarket will submit a new market configuration to the network.
func (e *Engine) SubmitMarket(ctx context.Context, marketConfig *types.Market) error {
	if e.log.IsDebug() {
		e.log.Debug("submit market", logging.Market(*marketConfig))
	}

	if err := e.submitMarket(ctx, marketConfig); err != nil {
		return err
	}

	// here straight away we start the OPENING_AUCTION
	mkt := e.markets[marketConfig.ID]
	_ = mkt.StartOpeningAuction(ctx)

	e.publishMarketInfos(ctx, mkt)
	return nil
}

func (e *Engine) publishMarketInfos(ctx context.Context, mkt *Market) {
	// we send a market data event for this market when it's created so graphql does not fail
	e.broker.Send(events.NewMarketDataEvent(ctx, mkt.GetMarketData()))
	e.broker.Send(events.NewMarketCreatedEvent(ctx, *mkt.mkt))
	e.broker.Send(events.NewMarketUpdatedEvent(ctx, *mkt.mkt))
}

// SubmitMarket will submit a new market configuration to the network.
func (e *Engine) submitMarket(ctx context.Context, marketConfig *types.Market) error {
	if len(marketConfig.ID) == 0 {
		return ErrNoMarketID
	}
	now := e.time.GetTimeNow()

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
		now,
		e.broker,
		e.idgen,
		mas,
		e.stateVarEngine,
		e.feesTracker,
		ad,
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

	if !e.npv.targetStakeScalingFactor.Equal(num.DecimalFromInt64(-1)) {
		if err := mkt.OnMarketTargetStakeScalingFactorUpdate(e.npv.targetStakeScalingFactor); err != nil {
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

	if e.npv.timeWindowUpdate != -1 {
		mkt.OnMarketTargetStakeTimeWindowUpdate(e.npv.timeWindowUpdate)
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
	if !e.npv.targetStakeTriggeringRatio.Equal(num.DecimalFromInt64(-1)) {
		mkt.OnMarketLiquidityTargetStakeTriggeringRatio(ctx, e.npv.targetStakeTriggeringRatio)
	}
	if !e.npv.maxLiquidityFee.Equal(num.DecimalFromInt64(-1)) {
		mkt.OnMarketLiquidityMaximumLiquidityFeeFactorLevelUpdate(e.npv.maxLiquidityFee)
	}
	return nil
}

func (e *Engine) removeMarket(mktID string) {
	delete(e.markets, mktID)
	for i, mkt := range e.marketsCpy {
		if mkt.GetID() == mktID {
			copy(e.marketsCpy[i:], e.marketsCpy[i+1:])
			e.marketsCpy[len(e.marketsCpy)-1] = nil
			e.marketsCpy = e.marketsCpy[:len(e.marketsCpy)-1]
			e.marketTracker.removeMarket(mktID)
			return
		}
	}
}

// SubmitOrder checks the incoming order and submits it to a Vega market.
func (e *Engine) SubmitOrder(
	ctx context.Context,
	submission *types.OrderSubmission,
	party string,
) (confirmation *types.OrderConfirmation, returnedErr error) {
	timer := metrics.NewTimeCounter(submission.MarketId, "execution", "SubmitOrder")
	defer func() {
		timer.EngineTimeCounterAdd()
	}()

	if e.log.IsDebug() {
		e.log.Debug("submit order", logging.OrderSubmission(submission))
	}

	mkt, ok := e.markets[submission.MarketId]
	if !ok {
		return nil, types.ErrInvalidMarketID
	}

	metrics.OrderGaugeAdd(1, submission.MarketId)
	conf, err := mkt.SubmitOrder(ctx, submission, party)
	if err != nil {
		return nil, err
	}

	e.decrementOrderGaugeMetrics(submission.MarketId, conf.Order, conf.PassiveOrdersAffected)

	return conf, nil
}

// AmendOrder takes order amendment details and attempts to amend the order
// if it exists and is in a editable state.
func (e *Engine) AmendOrder(ctx context.Context, amendment *types.OrderAmendment, party string) (confirmation *types.OrderConfirmation, returnedErr error) {
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

	conf, err := mkt.AmendOrder(ctx, amendment, party)
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
			passiveCount += 1
		}
	}
	if passiveCount > 0 {
		metrics.OrderGaugeAdd(-passiveCount, market)
	}
}

// CancelOrder takes order details and attempts to cancel if it exists in matching engine, stores etc.
func (e *Engine) CancelOrder(ctx context.Context, cancel *types.OrderCancellation, party string) (_ []*types.OrderCancellationConfirmation, returnedErr error) {
	timer := metrics.NewTimeCounter(cancel.MarketId, "execution", "CancelOrder")
	defer func() {
		timer.EngineTimeCounterAdd()
	}()

	if e.log.IsDebug() {
		e.log.Debug("cancel order", logging.OrderCancellation(cancel))
	}

	// ensure that if orderID is specified marketId is as well
	if len(cancel.OrderId) > 0 && len(cancel.MarketId) <= 0 {
		return nil, ErrInvalidOrderCancellation
	}

	if len(cancel.MarketId) > 0 {
		if len(cancel.OrderId) > 0 {
			return e.cancelOrder(ctx, party, cancel.MarketId, cancel.OrderId)
		}
		return e.cancelOrderByMarket(ctx, party, cancel.MarketId)
	}
	return e.cancelAllPartyOrders(ctx, party)
}

func (e *Engine) cancelOrder(ctx context.Context, party, market, orderID string) ([]*types.OrderCancellationConfirmation, error) {
	mkt, ok := e.markets[market]
	if !ok {
		return nil, types.ErrInvalidMarketID
	}
	conf, err := mkt.CancelOrder(ctx, party, orderID)
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
			confirmed += 1
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
				confirmed += 1
			}
		}
		metrics.OrderGaugeAdd(-confirmed, mkt.GetID())
	}
	return confirmations, nil
}

func (e *Engine) SubmitLiquidityProvision(ctx context.Context, sub *types.LiquidityProvisionSubmission, party, lpID string) (returnedErr error) {
	timer := metrics.NewTimeCounter(sub.MarketID, "execution", "LiquidityProvisionSubmission")
	defer func() {
		timer.EngineTimeCounterAdd()
	}()

	if e.log.IsDebug() {
		e.log.Debug("submit liquidity provision",
			logging.LiquidityProvisionSubmission(*sub),
			logging.PartyID(party),
			logging.LiquidityID(lpID),
		)
	}

	mkt, ok := e.markets[sub.MarketID]
	if !ok {
		return types.ErrInvalidMarketID
	}

	return mkt.SubmitLiquidityProvision(ctx, sub, party, lpID)
}

func (e *Engine) AmendLiquidityProvision(ctx context.Context, lpa *types.LiquidityProvisionAmendment, party string) (returnedErr error) {
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

	return mkt.AmendLiquidityProvision(ctx, lpa, party)
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

func (e *Engine) onChainTimeUpdate(ctx context.Context, t time.Time) {
	timer := metrics.NewTimeCounter("-", "execution", "onChainTimeUpdate")

	evts := make([]events.Event, 0, len(e.marketsCpy))
	for _, v := range e.marketsCpy {
		evts = append(evts, events.NewMarketDataEvent(ctx, v.GetMarketData()))
	}
	e.broker.SendBatch(evts)

	// update block time on id generator
	e.idgen.NewBatch()

	e.log.Debug("updating engine on new time update")

	// update collateral
	e.collateral.OnChainTimeUpdate(ctx, t)

	// remove expired orders
	// TODO(FIXME): this should be remove, and handled inside the market directly
	// when call with the new time (see the next for loop)
	e.removeExpiredOrders(ctx, t)

	// notify markets of the time expiration
	toDelete := []string{}
	for _, mkt := range e.marketsCpy {
		mkt := mkt
		closing := mkt.OnChainTimeUpdate(ctx, t)
		if closing {
			e.log.Info("market is closed, removing from execution engine",
				logging.MarketID(mkt.GetID()))
			delete(e.markets, mkt.GetID())
			toDelete = append(toDelete, mkt.GetID())
		}
	}

	for _, id := range toDelete {
		var i int
		for idx, mkt := range e.marketsCpy {
			if mkt.GetID() == id {
				i = idx
				break
			}
		}
		copy(e.marketsCpy[i:], e.marketsCpy[i+1:])
		e.marketsCpy = e.marketsCpy[:len(e.marketsCpy)-1]
	}

	timer.EngineTimeCounterAdd()
}

// Process any data updates (including state changes)
// e.g. removing expired orders from matching engine.
func (e *Engine) removeExpiredOrders(ctx context.Context, t time.Time) {
	timer := metrics.NewTimeCounter("-", "execution", "removeExpiredOrders")
	timeNow := t.UnixNano()
	for _, mkt := range e.marketsCpy {
		expired, err := mkt.RemoveExpiredOrders(ctx, timeNow)
		if err != nil {
			e.log.Error("unable to get remove expired orders",
				logging.MarketID(mkt.GetID()),
				logging.Error(err))
		}

		metrics.OrderGaugeAdd(-len(expired), mkt.GetID())
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

	e.npv.targetStakeScalingFactor = d

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

	e.npv.timeWindowUpdate = d

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
	_ context.Context, v int64) error {
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
	_ context.Context, d num.Decimal) error {
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

	e.npv.targetStakeTriggeringRatio = d

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
