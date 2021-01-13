package execution

import (
	"context"
	"sort"
	"time"

	"code.vegaprotocol.io/vega/collateral"
	"code.vegaprotocol.io/vega/crypto"
	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/metrics"
	"code.vegaprotocol.io/vega/monitor"
	types "code.vegaprotocol.io/vega/proto"

	"github.com/pkg/errors"
)

var (
	// ErrMarketAlreadyExist signals that a market already exist
	ErrMarketAlreadyExist = errors.New("market already exist")

	// ErrMarketAlreadyExist signals that a market already exist
	ErrMarketDoesNotExist = errors.New("market does not exist")

	// ErrUnknownProposalChange is returned if passed proposal cannot be enacted
	// because proposed changes cannot be processed by the system
	ErrUnknownProposalChange = errors.New("unknown proposal change")

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
}

// NewEngine takes stores and engines and returns
// a new execution engine to process new orders, etc.
func NewEngine(
	log *logging.Logger,
	executionConfig Config,
	ts TimeService,
	collateral *collateral.Engine,
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
	}

	// Add time change event handler
	e.time.NotifyOnTick(e.onChainTimeUpdate)

	return e
}

// ReloadConf updates the internal configuration of the execution
// engine and its dependencies
func (e *Engine) ReloadConf(cfg Config) {
	e.log.Info("reloading configuration")
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
	var tickSize string = "0."
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
func (e *Engine) RejectMarket(ctx context.Context, marketid string) error {
	mkt, ok := e.markets[marketid]
	if !ok {
		return ErrMarketDoesNotExist
	}

	if err := mkt.Reject(ctx); err != nil {
		return err
	}

	e.removeMarket(marketid)
	e.broker.Send(events.NewMarketUpdatedEvent(ctx, *mkt.mkt))
	return nil
}

// StartOpeningAuction will start the opening auction of the given market.
// This will work only if the market is currently in a PROPOSED state
func (e *Engine) StartOpeningAuction(ctx context.Context, marketid string) error {
	mkt, ok := e.markets[marketid]
	if !ok {
		return ErrMarketDoesNotExist
	}

	defer e.broker.Send(events.NewMarketUpdatedEvent(ctx, *mkt.mkt))
	return mkt.StartOpeningAuction(ctx)
}

// SubmitMarketWithLiquidityProvision is submitting a market through
// the usual governance process
func (e *Engine) SubmitMarketWithLiquidityProvision(ctx context.Context, marketConfig *types.Market, lp *types.LiquidityProvisionSubmission, party, lpid string) error {
	if err := e.submitMarket(ctx, marketConfig); err != nil {
		return err
	}

	// now we try to submit the liquidity
	mkt := e.markets[marketConfig.Id]
	if err := mkt.SubmitLiquidityProvision(ctx, lp, party, lpid); err != nil {
		e.removeMarket(marketConfig.Id)
		return err
	}

	e.publishMarketInfos(ctx, marketConfig.Id)
	return nil
}

// SubmitMarket will submit a new market configuration to the network
func (e *Engine) SubmitMarket(ctx context.Context, marketConfig *types.Market) error {
	if err := e.submitMarket(ctx, marketConfig); err != nil {
		return err
	}

	// here straight away we start the OPENING_AUCTION
	mkt := e.markets[marketConfig.Id]
	mkt.StartOpeningAuction(ctx)

	e.publishMarketInfos(ctx, marketConfig.Id)
	return nil
}

func (e *Engine) publishMarketInfos(ctx context.Context, marketid string) {
	mkt := e.markets[marketid]

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

	// ensure the asset for this new market exisrts
	asset, err := marketConfig.GetAsset()
	if err != nil {
		return err
	}
	if !e.collateral.AssetExists(asset) {
		e.log.Error("unable to create a market with an invalid asset",
			logging.String("market-id", marketConfig.Id),
			logging.String("asset-id", asset))
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
		e.collateral,
		marketConfig,
		now,
		e.broker,
		e.idgen,
		mas,
	)
	if err != nil {
		e.log.Error("Failed to instantiate market",
			logging.String("market-id", marketConfig.Id),
			logging.Error(err),
		)
	}

	e.markets[marketConfig.Id] = mkt
	e.marketsCpy = append(e.marketsCpy, mkt)

	// we ignore the reponse, this cannot fail as the asset
	// is already proven to exists a few line before
	_, _, _ = e.collateral.CreateMarketAccounts(ctx, marketConfig.Id, asset, e.Config.InsurancePoolInitialBalance)
	return nil
}

func (e *Engine) removeMarket(mktid string) {
	delete(e.markets, mktid)
	for i, mkt := range e.marketsCpy {
		if mkt.GetID() == mktid {
			copy(e.marketsCpy[i:], e.marketsCpy[i+1:])
			e.marketsCpy[len(e.marketsCpy)-1] = nil
			e.marketsCpy = e.marketsCpy[:len(e.marketsCpy)-1]
			return
		}
	}

}

// SubmitOrder checks the incoming order and submits it to a Vega market.
func (e *Engine) SubmitOrder(ctx context.Context, order *types.Order) (*types.OrderConfirmation, error) {
	timer := metrics.NewTimeCounter(order.MarketID, "execution", "SubmitOrder")

	if e.log.GetLevel() == logging.DebugLevel {
		e.log.Debug("Submit order", logging.Order(*order))
	}

	mkt, ok := e.markets[order.MarketID]
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
		metrics.OrderGaugeAdd(1, order.MarketID)
	}

	conf, err := mkt.SubmitOrder(ctx, order)
	if err != nil {
		timer.EngineTimeCounterAdd()
		return nil, err
	}

	if conf.Order.Status == types.Order_STATUS_FILLED {
		metrics.OrderGaugeAdd(-1, order.MarketID)
	}

	timer.EngineTimeCounterAdd()
	return conf, nil
}

// AmendOrder takes order amendment details and attempts to amend the order
// if it exists and is in a editable state.
func (e *Engine) AmendOrder(ctx context.Context, orderAmendment *types.OrderAmendment) (*types.OrderConfirmation, error) {
	if e.log.GetLevel() == logging.DebugLevel {
		e.log.Debug("Amend order", logging.OrderAmendment(orderAmendment))
	}

	mkt, ok := e.markets[orderAmendment.MarketID]
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
		metrics.OrderGaugeAdd(-1, orderAmendment.MarketID)
	}
	return conf, nil
}

// CancelOrder takes order details and attempts to cancel if it exists in matching engine, stores etc.
func (e *Engine) CancelOrder(ctx context.Context, order *types.OrderCancellation) ([]*types.OrderCancellationConfirmation, error) {
	if e.log.GetLevel() == logging.DebugLevel {
		e.log.Debug("Cancel order", logging.String("order-id", order.OrderID))
	}

	// ensure that if orderID is specified marketId is as well
	if len(order.OrderID) > 0 && len(order.MarketID) <= 0 {
		return nil, ErrInvalidOrderCancellation
	}

	if len(order.PartyID) > 0 {
		if len(order.MarketID) > 0 {
			if len(order.OrderID) > 0 {
				return e.cancelOrder(ctx, order.PartyID, order.MarketID, order.OrderID)
			}
			return e.cancelOrderByMarket(ctx, order.PartyID, order.MarketID)
		}
		return e.cancelAllPartyOrders(ctx, order.PartyID)
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
	confs, err := mkt.CancelAllOrders(ctx, party)
	if err != nil {
		return nil, err
	}
	var confirmed int
	for _, conf := range confs {
		if conf.Order.Status == types.Order_STATUS_CANCELLED {
			confirmed += 1
		}
	}
	metrics.OrderGaugeAdd(-confirmed, market)
	return confs, nil
}

func (e *Engine) cancelAllPartyOrders(ctx context.Context, party string) ([]*types.OrderCancellationConfirmation, error) {
	confirmations := []*types.OrderCancellationConfirmation{}

	for _, mkt := range e.marketsCpy {
		confs, err := mkt.CancelAllOrders(ctx, party)
		if err != nil {
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
				logging.String("market-id", mkt.GetID()))
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
		orders, err := mkt.RemoveExpiredOrders(timeNow)
		if err != nil {
			e.log.Error("unable to get remove expired orders",
				logging.String("market-id", mkt.GetID()),
				logging.Error(err))
		}
		expiringOrders = append(
			expiringOrders, orders...)
	}
	evts := make([]events.Event, 0, len(expiringOrders))
	for _, order := range expiringOrders {
		order := order
		evts = append(evts, events.NewOrderEvent(ctx, &order))
		metrics.OrderGaugeAdd(-1, order.MarketID) // decrement gauge
	}
	e.broker.SendBatch(evts)
	timer.EngineTimeCounterAdd()
}

func (e *Engine) GetMarketData(mktid string) (types.MarketData, error) {
	mkt, ok := e.markets[mktid]
	if !ok {
		return types.MarketData{}, types.ErrInvalidMarketID
	}
	return mkt.GetMarketData(), nil
}

func (e *Engine) SubmitLiquidityProvision(ctx context.Context, sub *types.LiquidityProvisionSubmission, party, id string) error {
	mkt, ok := e.markets[sub.MarketID]
	if !ok {
		return types.ErrInvalidMarketID
	}

	return mkt.SubmitLiquidityProvision(ctx, sub, party, id)
}

func (e *Engine) OnMarketMarginScalingFactorsUpdate(ctx context.Context, v interface{}) error {
	scalingFactors, ok := v.(*types.ScalingFactors)
	if !ok {
		return errors.New("invalid types for Margin ScalingFactors")
	}

	for _, mkt := range e.marketsCpy {
		if err := mkt.OnMarginScalingFactorsUpdate(ctx, scalingFactors); err != nil {
			return err
		}
	}
	return nil
}

func (e *Engine) OnMarketFeeFactorsMakerFeeUpdate(ctx context.Context, f float64) error {
	for _, mkt := range e.marketsCpy {
		if err := mkt.OnFeeFactorsMakerFeeUpdate(ctx, f); err != nil {
			return err
		}
	}
	return nil
}

func (e *Engine) OnMarketFeeFactorsInfrastructureFeeUpdate(ctx context.Context, f float64) error {
	for _, mkt := range e.marketsCpy {
		if err := mkt.OnFeeFactorsInfrastructureFeeUpdate(ctx, f); err != nil {
			return err
		}
	}
	return nil
}

func (e *Engine) OnSuppliedStakeToObligationFactorUpdate(_ context.Context, v float64) error {
	for _, mkt := range e.marketsCpy {
		mkt.OnSuppliedStakeToObligationFactorUpdate(v)
	}
	return nil
}

func (e *Engine) OnMarketValueWindowLengthUpdate(_ context.Context, d time.Duration) error {
	for _, mkt := range e.marketsCpy {
		mkt.OnMarketValueWindowLengthUpdate(d)
	}
	return nil
}
