package execution

import (
	"context"
	"time"

	"code.vegaprotocol.io/vega/collateral"
	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/metrics"
	types "code.vegaprotocol.io/vega/proto"

	"github.com/pkg/errors"
)

var (
	// ErrMarketAlreadyExist signals that a market already exist
	ErrMarketAlreadyExist = errors.New("market already exist")

	// ErrUnknownProposalChange is returned if passed proposal cannot be enacted
	// because proposed changes cannot be processed by the system
	ErrUnknownProposalChange = errors.New("unknown proposal change")

	// ErrNoMarketID is returned when invalid (empty) market id was supplied during market creation
	ErrNoMarketID = errors.New("no valid market id was supplied")
)

// CandleBuf ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/candle_buf_mock.go -package mocks code.vegaprotocol.io/vega/execution CandleBuf
type CandleBuf interface {
	AddTrade(types.Trade) error
	Flush(marketID string, t time.Time) error
	Start(marketID string, t time.Time) (map[string]types.Candle, error)
}

// TimeService ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/time_service_mock.go -package mocks code.vegaprotocol.io/vega/execution TimeService
type TimeService interface {
	GetTimeNow() (time.Time, error)
	NotifyOnTick(f func(time.Time))
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
	party      *Party
	collateral *collateral.Engine
	idgen      *IDgenerator

	candleBuf CandleBuf

	broker Broker
	time   TimeService
}

// NewEngine takes stores and engines and returns
// a new execution engine to process new orders, etc.
func NewEngine(
	log *logging.Logger,
	executionConfig Config,
	time TimeService,
	candleBuf CandleBuf,
	pmkts []types.Market,
	collateral *collateral.Engine,
	broker Broker,
) *Engine {
	// setup logger
	log = log.Named(namedLogger)
	log.SetLevel(executionConfig.Level.Get())
	// this is here because we're creating some markets here
	// this isn't going to be the case in the final version
	// so I'm using Background rather than TODO
	ctx := context.Background()

	e := &Engine{
		log:        log,
		Config:     executionConfig,
		markets:    map[string]*Market{},
		candleBuf:  candleBuf,
		time:       time,
		collateral: collateral,
		party:      NewParty(log, collateral, pmkts, broker),
		idgen:      NewIDGen(),
		broker:     broker,
	}

	var err error
	// Add initial markets and flush to stores (if they're configured)
	if len(pmkts) > 0 {
		for _, mkt := range pmkts {
			mkt := mkt
			err = e.SubmitMarket(ctx, &mkt)
			if err != nil {
				e.log.Panic("Unable to submit market",
					logging.Error(err))
			}
		}
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
	for _, mkt := range e.markets {
		mkt.ReloadConf(e.Config.Matching, e.Config.Risk,
			e.Config.Collateral, e.Config.Position, e.Config.Settlement)
	}
}

// NotifyTraderAccount notify the engine to create a new account for a party
func (e *Engine) NotifyTraderAccount(ctx context.Context, notify *types.NotifyTraderAccount) error {
	return e.party.NotifyTraderAccount(ctx, notify)
}

// CreateGeneralAccounts creates new general accounts for a party
func (e *Engine) CreateGeneralAccounts(partyID string) error {
	ctx := context.TODO() // not sure if this call is used at all
	_, err := e.party.MakeGeneralAccounts(ctx, partyID)
	return err
}

func (e *Engine) Withdraw(ctx context.Context, w *types.Withdraw) error {
	err := e.collateral.Withdraw(ctx, w.PartyID, w.Asset, w.Amount)
	if err != nil {
		e.log.Error("An error occurred during withdrawal",
			logging.String("party-id", w.PartyID),
			logging.Uint64("amount", w.Amount),
			logging.Error(err),
		)
	}
	return err
}

// SubmitMarket will submit a new market configuration to the network
func (e *Engine) SubmitMarket(ctx context.Context, marketConfig *types.Market) error {
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

	mkt, err := NewMarket(
		e.log,
		e.Config.Risk,
		e.Config.Position,
		e.Config.Settlement,
		e.Config.Matching,
		e.collateral,
		e.party,
		marketConfig,
		e.candleBuf,
		now,
		e.broker,
		e.idgen,
	)
	if err != nil {
		e.log.Error("Failed to instantiate market",
			logging.String("market-id", marketConfig.Id),
			logging.Error(err),
		)
	}

	e.markets[marketConfig.Id] = mkt

	// we ignore the reponse, this cannot fail as the asset
	// is already proven to exists a few line before
	_, _, _ = e.collateral.CreateMarketAccounts(ctx, marketConfig.Id, asset, e.Config.InsurancePoolInitialBalance)

	// wire up party engine to new market
	e.party.addMarket(*mkt.mkt)

	e.broker.Send(events.NewMarketEvent(ctx, *mkt.mkt))
	return nil
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
func (e *Engine) CancelOrder(ctx context.Context, order *types.OrderCancellation) (*types.OrderCancellationConfirmation, error) {
	if e.log.GetLevel() == logging.DebugLevel {
		e.log.Debug("Cancel order", logging.String("order-id", order.OrderID))
	}
	mkt, ok := e.markets[order.MarketID]
	if !ok {
		return nil, types.ErrInvalidMarketID
	}

	conf, err := mkt.CancelOrder(ctx, order)
	if err != nil {
		return nil, err
	}
	if conf.Order.Status == types.Order_STATUS_CANCELLED {
		metrics.OrderGaugeAdd(-1, order.MarketID)
	}
	return conf, nil
}

// CancelOrderByID attempts to locate order by its Id and cancel it if exists.
func (e *Engine) CancelOrderByID(orderID string, marketID string) (*types.OrderCancellationConfirmation, error) {
	if e.log.GetLevel() == logging.DebugLevel {
		e.log.Debug("Cancel order by id", logging.String("order-id", orderID))
	}
	mkt, ok := e.markets[marketID]
	if !ok {
		return nil, types.ErrInvalidMarketID
	}
	conf, err := mkt.CancelOrderByID(orderID)
	if err != nil {
		return nil, err
	}
	if conf.Order.Status == types.Order_STATUS_CANCELLED {
		metrics.OrderGaugeAdd(-1, marketID)
	}
	return conf, nil
}

func (e *Engine) onChainTimeUpdate(t time.Time) {
	timer := metrics.NewTimeCounter("-", "execution", "onChainTimeUpdate")

	// update block time on id generator
	e.idgen.NewBatch()

	e.log.Debug("updating engine on new time update")

	// update collateral
	e.collateral.OnChainTimeUpdate(t)

	// remove expired orders
	// TODO(FIXME): this should be remove, and handled inside the market directly
	// when call with the new time (see the next for loop)
	e.removeExpiredOrders(t)

	// notify markets of the time expiration
	for mktID, mkt := range e.markets {
		mkt := mkt
		closing := mkt.OnChainTimeUpdate(t)
		if closing {
			e.log.Info("market is closed, removing from execution engine",
				logging.String("market-id", mktID))
			delete(e.markets, mktID)
		}
	}
	timer.EngineTimeCounterAdd()
}

// Process any data updates (including state changes)
// e.g. removing expired orders from matching engine.
func (e *Engine) removeExpiredOrders(t time.Time) {
	timer := metrics.NewTimeCounter("-", "execution", "removeExpiredOrders")
	if e.log.GetLevel() == logging.DebugLevel {
		e.log.Debug("Removing expiring orders from matching engine")
	}
	expiringOrders := []types.Order{}
	timeNow := t.UnixNano()
	for _, mkt := range e.markets {
		orders, err := mkt.RemoveExpiredOrders(timeNow)
		if err != nil {
			e.log.Error("unable to get remove expired orders",
				logging.String("market-id", mkt.GetID()),
				logging.Error(err))
		}
		expiringOrders = append(
			expiringOrders, orders...)
	}
	if e.log.GetLevel() == logging.DebugLevel {
		e.log.Debug("Removed expired orders from matching engine",
			logging.Int("orders-removed", len(expiringOrders)))
	}
	for _, order := range expiringOrders {
		order := order
		evt := events.NewOrderEvent(context.Background(), &order)
		e.broker.Send(evt)
		metrics.OrderGaugeAdd(-1, order.MarketID) // decrement gauge
	}
	if e.log.GetLevel() == logging.DebugLevel {
		e.log.Debug("Updated expired orders in stores",
			logging.Int("orders-removed", len(expiringOrders)))
	}
	timer.EngineTimeCounterAdd()
}

func (e *Engine) GetMarketData(mktid string) (types.MarketData, error) {
	mkt, ok := e.markets[mktid]
	if !ok {
		return types.MarketData{}, types.ErrInvalidMarketID
	}
	return mkt.GetMarketData(), nil
}

// Generate flushes any data (including storing state changes) to underlying stores (if configured).
func (e *Engine) Generate() error {
	ctx := context.TODO()

	// Market data is added to buffer on Generate
	// do this before the time event -> time event flushes
	for _, v := range e.markets {
		e.broker.Send(events.NewMarketDataEvent(ctx, v.GetMarketData()))
	}
	// Transfers
	// @TODO this event will be generated with a block context that has the trace ID
	// this will have the effect of flushing the transfer response buffer
	now, _ := e.time.GetTimeNow()
	evt := events.NewTime(ctx, now)
	e.broker.Send(evt)
	// Markets
	return nil
}
