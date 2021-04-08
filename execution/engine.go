package execution

import (
	"context"
	"errors"
	"sort"
	"time"

	"code.vegaprotocol.io/vega/collateral"
	"code.vegaprotocol.io/vega/crypto"
	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/metrics"
	"code.vegaprotocol.io/vega/monitor"
	"code.vegaprotocol.io/vega/products"
	types "code.vegaprotocol.io/vega/proto"
)

var (
	// ErrMarketAlreadyExist signals that a market already exist
	ErrMarketDoesNotExist = errors.New("market does not exist")

	// ErrNoMarketID is returned when invalid (empty) market id was supplied during market creation
	ErrNoMarketID = errors.New("no valid market id was supplied")

	// ErrInvalidOrderCancellation is returned when an incomplete order cancellation request is used
	ErrInvalidOrderCancellation = errors.New("invalid order cancellation")
)

// TimeService ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/time_service_mock.go -package mocks code.vegaprotocol.io/vega/execution TimeService
type TimeService interface {
	GetTimeNow() (time.Time, error)
	NotifyOnTick(f func(context.Context, time.Time))
}

// Broker ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/event_broker_mock.go -package mocks code.vegaprotocol.io/vega/execution Broker
type Broker interface {
	Send(event events.Event)
	SendBatch(events []events.Event)
}

// Engine is the execution engine
type Engine struct {
	Config
	log *logging.Logger

	markets    map[string]*Market
	marketsCpy []*Market
	collateral *collateral.Engine
	idgen      *IDgenerator

	broker Broker
	time   TimeService

	oracle products.OracleEngine

	npv netParamsValues
}

type netParamsValues struct {
	shapesMaxSize                   int64
	feeDistributionTimeStep         time.Duration
	timeWindowUpdate                time.Duration
	targetStakeScalingFactor        float64
	marketValueWindowLength         time.Duration
	suppliedStakeToObligationFactor float64
	infrastructureFee               float64
	makerFee                        float64
	scalingFactors                  *types.ScalingFactors
	maxLiquidityFee                 float64
	bondPenaltyFactor               float64
	targetStakeTriggeringRatio      float64
	auctionMinDuration              time.Duration
}

func defaultNetParamsValues() netParamsValues {
	return netParamsValues{
		shapesMaxSize:                   -1,
		feeDistributionTimeStep:         -1,
		timeWindowUpdate:                -1,
		targetStakeScalingFactor:        -1,
		marketValueWindowLength:         -1,
		suppliedStakeToObligationFactor: -1,
		infrastructureFee:               -1,
		makerFee:                        -1,
		scalingFactors:                  nil,
		maxLiquidityFee:                 -1,
		bondPenaltyFactor:               -1,
		targetStakeTriggeringRatio:      -1,
		auctionMinDuration:              -1,
	}
}

// NewEngine takes stores and engines and returns
// a new execution engine to process new orders, etc.
func NewEngine(
	log *logging.Logger,
	executionConfig Config,
	ts TimeService,
	collateral *collateral.Engine,
	oracle products.OracleEngine,
	broker Broker,
) *Engine {
	// setup logger
	log = log.Named(namedLogger)
	log.SetLevel(executionConfig.Level.Get())
	e := &Engine{
		log:        log,
		Config:     executionConfig,
		markets:    map[string]*Market{},
		time:       ts,
		collateral: collateral,
		idgen:      NewIDGen(),
		broker:     broker,
		oracle:     oracle,
		npv:        defaultNetParamsValues(),
	}

	// Add time change event handler
	e.time.NotifyOnTick(e.onChainTimeUpdate)

	return e
}

// ReloadConf updates the internal configuration of the execution
// engine and its dependencies
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
		mkt.ReloadConf(e.Config.Matching, e.Config.Risk,
			e.Config.Position, e.Config.Settlement, e.Config.Fee)
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
	var tickSize = "0."
	for decimalPlaces > 1 {
		tickSize += "0"
		decimalPlaces--
	}
	tickSize += "1"
	return tickSize
}

// RejectMarket will stop the execution of the market
// and refund into the general account any funds in margins accounts from any parties
// This works only if the market is in a PROPOSED STATE
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
// This will work only if the market is currently in a PROPOSED state
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

// SubmitMarketWithLiquidityProvision is submitting a market through
// the usual governance process
func (e *Engine) SubmitMarketWithLiquidityProvision(ctx context.Context, marketConfig *types.Market, lp *types.LiquidityProvisionSubmission, party, lpID string) error {
	if e.log.IsDebug() {
		e.log.Debug("submit market with liquidity provision",
			logging.Market(*marketConfig),
			// FIXME(JEREMY): this would crash in some cases at the moment.
			// as the lp can be nil as it's optional for now in order
			// to ease the transitions for the tests to use LP commitment
			// submission with every new market.
			// uncomment when this is not needed anymore
			// logging.LiquidityProvisionSubmission(*lp),
			logging.PartyID(party),
			logging.LiquidityID(lpID),
		)
	}

	if err := e.submitMarket(ctx, marketConfig); err != nil {
		return err
	}
	// publish market data anyway initially
	e.publishMarketInfos(ctx, mkt)

	mkt := e.markets[marketConfig.Id]
	// TODO(): remove check once LiquidityProvision is required
	// for now it is optional
	if lp != nil {
		// now we try to submit the liquidity
		if err := mkt.SubmitLiquidityProvision(ctx, lp, party, lpID); err != nil {
			e.removeMarket(marketConfig.Id)
			return err
		}
	}

	return nil
}

// SubmitMarket will submit a new market configuration to the network
func (e *Engine) SubmitMarket(ctx context.Context, marketConfig *types.Market) error {
	if e.log.IsDebug() {
		e.log.Debug("submit market", logging.Market(*marketConfig))
	}

	if err := e.submitMarket(ctx, marketConfig); err != nil {
		return err
	}

	// here straight away we start the OPENING_AUCTION
	mkt := e.markets[marketConfig.Id]
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

// SubmitMarket will submit a new market configuration to the network
func (e *Engine) submitMarket(ctx context.Context, marketConfig *types.Market) error {
	if len(marketConfig.Id) == 0 {
		return ErrNoMarketID
	}
	now, err := e.time.GetTimeNow()
	if err != nil {
		e.log.Error("Failed to get current Vega network time", logging.Error(err))
		return err
	}

	// ensure the asset for this new market exists
	asset, err := marketConfig.GetAsset()
	if err != nil {
		return err
	}
	if !e.collateral.AssetExists(asset) {
		e.log.Error("unable to create a market with an invalid asset",
			logging.MarketID(marketConfig.Id),
			logging.AssetID(asset))
	}

	// set a fake tick size to the continuous trading if it's continuous
	switch tmod := marketConfig.TradingModeConfig.(type) {
	case *types.Market_Continuous:
		tmod.Continuous.TickSize = e.getFakeTickSize(marketConfig.DecimalPlaces)
	case *types.Market_Discrete:
		tmod.Discrete.TickSize = e.getFakeTickSize(marketConfig.DecimalPlaces)
	}

	// create market auction state
	mas := monitor.NewAuctionState(marketConfig, now)
	mkt, err := NewMarket(
		ctx,
		e.log,
		e.Config.Risk,
		e.Config.Position,
		e.Config.Settlement,
		e.Config.Matching,
		e.Config.Fee,
		e.Config.Liquidity,
		e.collateral,
		e.oracle,
		marketConfig,
		now,
		e.broker,
		e.idgen,
		mas,
	)
	if err != nil {
		e.log.Error("failed to instantiate market",
			logging.MarketID(marketConfig.Id),
			logging.Error(err),
		)
		return err
	}

	e.markets[marketConfig.Id] = mkt
	e.marketsCpy = append(e.marketsCpy, mkt)

	// we ignore the response, this cannot fail as the asset
	// is already proven to exists a few line before
	_, _, _ = e.collateral.CreateMarketAccounts(ctx, marketConfig.Id, asset, e.Config.InsurancePoolInitialBalance)

	if err := e.propagateInitialNetParams(ctx, mkt); err != nil {
		return err
	}

	return nil
}

func (e *Engine) propagateInitialNetParams(ctx context.Context, mkt *Market) error {
	if e.npv.auctionMinDuration != -1 {
		mkt.OnMarketAuctionMinimumDurationUpdate(ctx, e.npv.auctionMinDuration)
	}
	if e.npv.shapesMaxSize != -1 {
		if err := mkt.OnMarketLiquidityProvisionShapesMaxSizeUpdate(e.npv.shapesMaxSize); err != nil {
			return err
		}
	}

	if e.npv.targetStakeScalingFactor != -1 {
		if err := mkt.OnMarketTargetStakeScalingFactorUpdate(e.npv.targetStakeScalingFactor); err != nil {
			return err
		}
	}

	if e.npv.infrastructureFee != -1 {
		if err := mkt.OnFeeFactorsInfrastructureFeeUpdate(ctx, e.npv.infrastructureFee); err != nil {
			return err
		}
	}

	if e.npv.makerFee != -1 {
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

	if e.npv.suppliedStakeToObligationFactor != -1 {
		mkt.OnSuppliedStakeToObligationFactorUpdate(e.npv.suppliedStakeToObligationFactor)
	}
	if e.npv.bondPenaltyFactor != -1 {
		mkt.BondPenaltyFactorUpdate(ctx, e.npv.bondPenaltyFactor)
	}
	if e.npv.targetStakeTriggeringRatio != -1 {
		mkt.OnMarketLiquidityTargetStakeTriggeringRatio(ctx, e.npv.targetStakeTriggeringRatio)
	}
	if e.npv.maxLiquidityFee != -1 {
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
			return
		}
	}

}

// SubmitOrder checks the incoming order and submits it to a Vega market.
func (e *Engine) SubmitOrder(ctx context.Context, order *types.Order) (*types.OrderConfirmation, error) {
	if e.log.IsDebug() {
		e.log.Debug("submit order", logging.Order(*order))
	}

	timer := metrics.NewTimeCounter(order.MarketId, "execution", "SubmitOrder")

	mkt, ok := e.markets[order.MarketId]
	if !ok {
		e.idgen.SetID(order)

		// adding rejected order to the buf
		order.Status = types.Order_STATUS_REJECTED
		order.Reason = types.OrderError_ORDER_ERROR_INVALID_MARKET_ID
		evt := events.NewOrderEvent(ctx, order)
		e.broker.Send(evt)

		timer.EngineTimeCounterAdd()
		return nil, types.ErrInvalidMarketID
	}

	if order.Status == types.Order_STATUS_ACTIVE {
		metrics.OrderGaugeAdd(1, order.MarketId)
	}

	conf, err := mkt.SubmitOrder(ctx, order)
	if err != nil {
		timer.EngineTimeCounterAdd()
		return nil, err
	}

	if conf.Order.Status == types.Order_STATUS_FILLED {
		metrics.OrderGaugeAdd(-1, order.MarketId)
	}

	timer.EngineTimeCounterAdd()
	return conf, nil
}

// AmendOrder takes order amendment details and attempts to amend the order
// if it exists and is in a editable state.
func (e *Engine) AmendOrder(ctx context.Context, orderAmendment *types.OrderAmendment) (confirmation *types.OrderConfirmation, returnedErr error) {
	defer func() {
		e.notifyFailureOnError(ctx, returnedErr, orderAmendment.PartyId, orderAmendment)
	}()

	if e.log.IsDebug() {
		e.log.Debug("amend order", logging.OrderAmendment(orderAmendment))
	}

	mkt, ok := e.markets[orderAmendment.MarketId]
	if !ok {
		return nil, types.ErrInvalidMarketID
	}

	// we're passing a pointer here, so we need the wasActive var to be certain we're checking the original
	// order status. It's possible order.Status will reflect the new status value if we don't
	conf, err := mkt.AmendOrder(ctx, orderAmendment)
	if err != nil {
		return nil, err
	}
	// order was active, not anymore -> decrement gauge
	if conf.Order.Status != types.Order_STATUS_ACTIVE {
		metrics.OrderGaugeAdd(-1, orderAmendment.MarketId)
	}
	return conf, nil
}

// CancelOrder takes order details and attempts to cancel if it exists in matching engine, stores etc.
func (e *Engine) CancelOrder(ctx context.Context, order *types.OrderCancellation) ([]*types.OrderCancellationConfirmation, error) {
	if e.log.IsDebug() {
		e.log.Debug("cancel order", logging.OrderCancellation(order))
	}

	// ensure that if orderID is specified marketId is as well
	if len(order.OrderId) > 0 && len(order.MarketId) <= 0 {
		return nil, ErrInvalidOrderCancellation
	}

	if len(order.PartyId) > 0 {
		if len(order.MarketId) > 0 {
			if len(order.OrderId) > 0 {
				return e.cancelOrder(ctx, order.PartyId, order.MarketId, order.OrderId)
			}
			return e.cancelOrderByMarket(ctx, order.PartyId, order.MarketId)
		}
		return e.cancelAllPartyOrders(ctx, order.PartyId)
	}

	return nil, ErrInvalidOrderCancellation
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
	if conf.Order.Status == types.Order_STATUS_CANCELLED {
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
		if conf.Order.Status == types.Order_STATUS_CANCELLED {
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
			if conf.Order.Status == types.Order_STATUS_CANCELLED {
				confirmed += 1
			}
		}
		metrics.OrderGaugeAdd(-confirmed, mkt.GetID())
	}
	return confirmations, nil
}

func (e *Engine) onChainTimeUpdate(ctx context.Context, t time.Time) {
	timer := metrics.NewTimeCounter("-", "execution", "onChainTimeUpdate")

	evts := make([]events.Event, 0, len(e.marketsCpy))
	for _, v := range e.marketsCpy {
		evts = append(evts, events.NewMarketDataEvent(ctx, v.GetMarketData()))
	}
	e.broker.SendBatch(evts)
	evt := events.NewTime(ctx, t)
	e.broker.Send(evt)

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
	expiringOrders := []types.Order{}
	timeNow := t.UnixNano()
	for _, mkt := range e.marketsCpy {
		orders, err := mkt.RemoveExpiredOrders(ctx, timeNow)
		if err != nil {
			e.log.Error("unable to get remove expired orders",
				logging.MarketID(mkt.GetID()),
				logging.Error(err))
		}
		expiringOrders = append(
			expiringOrders, orders...)
	}
	evts := make([]events.Event, 0, len(expiringOrders))
	for _, order := range expiringOrders {
		order := order
		evts = append(evts, events.NewOrderEvent(ctx, &order))
		metrics.OrderGaugeAdd(-1, order.MarketId) // decrement gauge
	}
	e.broker.SendBatch(evts)
	timer.EngineTimeCounterAdd()
}

func (e *Engine) SubmitLiquidityProvision(ctx context.Context, sub *types.LiquidityProvisionSubmission, party, lpID string) error {
	if e.log.IsDebug() {
		e.log.Debug("submit liquidity provision",
			logging.LiquidityProvisionSubmission(*sub),
			logging.PartyID(party),
			logging.LiquidityID(lpID),
		)
	}

	mkt, ok := e.markets[sub.MarketId]
	if !ok {
		return types.ErrInvalidMarketID
	}

	return mkt.SubmitLiquidityProvision(ctx, sub, party, lpID)
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

func (e *Engine) OnMarketLiquidityBondPenaltyUpdate(ctx context.Context, v float64) error {
	if e.log.IsDebug() {
		e.log.Debug("update market liquidity bond penalty",
			logging.Float64("bond-penalty-factor", v),
		)
	}

	for _, mkt := range e.markets {
		mkt.BondPenaltyFactorUpdate(ctx, v)
	}

	e.npv.bondPenaltyFactor = v

	return nil
}

func (e *Engine) OnMarketMarginScalingFactorsUpdate(ctx context.Context, v interface{}) error {
	if e.log.IsDebug() {
		e.log.Debug("update market scaling factors",
			logging.Reflect("scaling-factors", v),
		)
	}

	scalingFactors, ok := v.(*types.ScalingFactors)
	if !ok {
		return errors.New("invalid types for Margin ScalingFactors")
	}

	for _, mkt := range e.marketsCpy {
		if err := mkt.OnMarginScalingFactorsUpdate(ctx, scalingFactors); err != nil {
			return err
		}
	}

	e.npv.scalingFactors = scalingFactors

	return nil
}

func (e *Engine) OnMarketFeeFactorsMakerFeeUpdate(ctx context.Context, f float64) error {
	if e.log.IsDebug() {
		e.log.Debug("update maker fee in market fee factors",
			logging.Float64("maker-fee", f),
		)
	}

	for _, mkt := range e.marketsCpy {
		if err := mkt.OnFeeFactorsMakerFeeUpdate(ctx, f); err != nil {
			return err
		}
	}

	e.npv.makerFee = f

	return nil
}

func (e *Engine) OnMarketFeeFactorsInfrastructureFeeUpdate(ctx context.Context, f float64) error {
	if e.log.IsDebug() {
		e.log.Debug("update infrastructure fee in market fee factors",
			logging.Float64("infrastructure-fee", f),
		)
	}

	for _, mkt := range e.marketsCpy {
		if err := mkt.OnFeeFactorsInfrastructureFeeUpdate(ctx, f); err != nil {
			return err
		}
	}

	e.npv.infrastructureFee = f

	return nil
}

func (e *Engine) OnSuppliedStakeToObligationFactorUpdate(_ context.Context, v float64) error {
	if e.log.IsDebug() {
		e.log.Debug("update supplied stake to obligation factor",
			logging.Float64("factor", v),
		)
	}

	for _, mkt := range e.marketsCpy {
		mkt.OnSuppliedStakeToObligationFactorUpdate(v)
	}

	e.npv.suppliedStakeToObligationFactor = v

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

func (e *Engine) OnMarketTargetStakeScalingFactorUpdate(_ context.Context, v float64) error {
	if e.log.IsDebug() {
		e.log.Debug("update market stake scaling factor",
			logging.Float64("scaling-factor", v),
		)
	}

	for _, mkt := range e.marketsCpy {
		if err := mkt.OnMarketTargetStakeScalingFactorUpdate(v); err != nil {
			return err
		}
	}

	e.npv.targetStakeScalingFactor = v

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
	_ context.Context, f float64) error {
	if e.log.IsDebug() {
		e.log.Debug("update liquidity provision max liquidity fee factor",
			logging.Float64("max-liquidity-fee", f),
		)
	}

	for _, mkt := range e.marketsCpy {
		mkt.OnMarketLiquidityMaximumLiquidityFeeFactorLevelUpdate(f)
	}

	e.npv.maxLiquidityFee = f

	return nil
}

func (e *Engine) OnMarketLiquidityTargetStakeTriggeringRatio(ctx context.Context, v float64) error {
	if e.log.IsDebug() {
		e.log.Debug("update target stake triggering ratio",
			logging.Float64("max-liquidity-fee", v),
		)
	}

	for _, mkt := range e.marketsCpy {
		mkt.OnMarketLiquidityTargetStakeTriggeringRatio(ctx, v)
	}

	e.npv.targetStakeTriggeringRatio = v

	return nil
}

func (e *Engine) notifyFailureOnError(ctx context.Context, err error, partyID string, tx interface{}) {
	if err != nil {
		e.broker.Send(events.NewTxErrEvent(ctx, err, partyID, tx))
	}
}
