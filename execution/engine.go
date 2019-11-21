package execution

import (
	"fmt"
	"time"

	"code.vegaprotocol.io/vega/collateral"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/metrics"
	types "code.vegaprotocol.io/vega/proto"

	"github.com/pkg/errors"
)

var (
	// ErrMarketAlreadyExist signals that a market already exist
	ErrMarketAlreadyExist = errors.New("market already exist")
)

// OrderBuf ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/order_buf_mock.go -package mocks code.vegaprotocol.io/vega/execution OrderBuf
type OrderBuf interface {
	Add(types.Order)
	Flush() error
}

// TradeBuf ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/trade_buf_mock.go -package mocks code.vegaprotocol.io/vega/execution TradeBuf
type TradeBuf interface {
	Add(types.Trade)
	Flush() error
}

// CandleBuf ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/candle_buf_mock.go -package mocks code.vegaprotocol.io/vega/execution CandleBuf
type CandleBuf interface {
	AddTrade(types.Trade) error
	Flush(marketID string, t time.Time) error
	Start(marketID string, t time.Time) (map[string]types.Candle, error)
}

// MarketBuf ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/market_buf_mock.go -package mocks code.vegaprotocol.io/vega/execution MarketBuf
type MarketBuf interface {
	Add(types.Market)
	Flush() error
}

// PartyBuf ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/party_buf_mock.go -package mocks code.vegaprotocol.io/vega/execution PartyBuf
type PartyBuf interface {
	Add(types.Party)
	Flush() error
}

// TimeService ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/time_service_mock.go -package mocks code.vegaprotocol.io/vega/execution TimeService
type TimeService interface {
	GetTimeNow() (time.Time, error)
	NotifyOnTick(f func(time.Time))
}

// TransferBuf ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/transfer_buf_mock.go -package mocks code.vegaprotocol.io/vega/execution TransferBuf
type TransferBuf interface {
	Add([]*types.TransferResponse)
	Flush() error
}

// AccountBuf ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/account_buf_mock.go -package mocks code.vegaprotocol.io/vega/execution AccountBuf
type AccountBuf interface {
	Add(types.Account)
	Flush() error
}

// Engine is the execution engine
type Engine struct {
	Config
	log *logging.Logger

	markets    map[string]*Market
	party      *Party
	collateral *collateral.Engine
	idgen      *IDgenerator

	orderBuf    OrderBuf
	tradeBuf    TradeBuf
	candleBuf   CandleBuf
	marketBuf   MarketBuf
	partyBuf    PartyBuf
	accountBuf  AccountBuf
	transferBuf TransferBuf

	time TimeService
}

// NewEngine takes stores and engines and returns
// a new execution engine to process new orders, etc.
func NewEngine(
	log *logging.Logger,
	executionConfig Config,
	time TimeService,
	orderBuf OrderBuf,
	tradeBuf TradeBuf,
	candleBuf CandleBuf,
	marketBuf MarketBuf,
	partyBuf PartyBuf,
	accountBuf AccountBuf,
	transferBuf TransferBuf,
	pmkts []types.Market,
) *Engine {
	// setup logger
	log = log.Named(namedLogger)
	log.SetLevel(executionConfig.Level.Get())

	now, err := time.GetTimeNow()
	if err != nil {
		log.Error("unable to get the time now", logging.Error(err))
		return nil
	}
	//  create collateral
	cengine, err := collateral.New(log, executionConfig.Collateral, accountBuf, now)
	if err != nil {
		log.Error("unable to initialize collateral", logging.Error(err))
		return nil
	}

	e := &Engine{
		log:         log,
		Config:      executionConfig,
		markets:     map[string]*Market{},
		candleBuf:   candleBuf,
		orderBuf:    orderBuf,
		tradeBuf:    tradeBuf,
		marketBuf:   marketBuf,
		partyBuf:    partyBuf,
		time:        time,
		collateral:  cengine,
		party:       NewParty(log, cengine, pmkts, partyBuf),
		accountBuf:  accountBuf,
		transferBuf: transferBuf,
		idgen:       NewIDGen(),
	}

	for _, mkt := range pmkts {
		mkt := mkt
		err = e.SubmitMarket(&mkt)
		if err != nil {
			e.log.Panic("Unable to submit market",
				logging.Error(err))
		}
	}

	// just flush a first time the markets
	if err := e.marketBuf.Flush(); err != nil {
		e.log.Error("unable to flush markets", logging.Error(err))
		return nil
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
func (e *Engine) NotifyTraderAccount(notif *types.NotifyTraderAccount) error {
	return e.party.NotifyTraderAccount(notif)
}

func (e *Engine) Withdraw(w *types.Withdraw) error {
	err := e.collateral.Withdraw(w.PartyID, w.Asset, w.Amount)
	if err != nil {
		e.log.Error("something happend durinmg withdrawal",
			logging.String("party-id", w.PartyID),
			logging.Uint64("amount", w.Amount),
			logging.Error(err),
		)
	}
	return err
}

// SubmitMarket will submit a new market configuration to the network
func (e *Engine) SubmitMarket(mktconfig *types.Market) error {

	// TODO: Check for existing market in MarketStore by Name.
	// if __TBC_MarketExists__(mktconfig.Name) {
	// 	return ErrMarketAlreadyExist
	// }

	var mkt *Market
	var err error

	now, _ := e.time.GetTimeNow()
	mkt, err = NewMarket(
		e.log,
		e.Config.Risk,
		e.Config.Position,
		e.Config.Settlement,
		e.Config.Matching,
		e.collateral,
		e.party,
		mktconfig,
		e.candleBuf,
		e.orderBuf,
		e.partyBuf,
		e.tradeBuf,
		e.transferBuf,
		now,
		e.idgen,
	)
	if err != nil {
		e.log.Error("Failed to instanciate market",
			logging.String("market-name", mktconfig.Name),
			logging.Error(err),
		)
	}

	e.marketBuf.Add(*mktconfig)

	e.markets[mktconfig.Id] = mkt

	// create market accounts
	asset, err := mktconfig.GetAsset()
	if err != nil {
		return err
	}

	// ignore response ids here + this cannot fail
	_, _ = e.collateral.CreateMarketAccounts(mktconfig.Id, asset, 0)

	updatedMarkets := append(e.party.markets, *mkt.mkt)
	e.party = NewParty(e.log, e.collateral, updatedMarkets, e.partyBuf)

	return nil
}

// SubmitOrder submit a new order to the vega trading core
func (e *Engine) SubmitOrder(order *types.Order) (*types.OrderConfirmation, error) {
	// order.MarketID may or may not be valid.
	timer := metrics.NewTimeCounter(order.MarketID, "execution", "SubmitOrder")

	if e.log.GetLevel() == logging.DebugLevel {
		e.log.Debug("Submit order", logging.Order(*order))
	}

	mkt, ok := e.markets[order.MarketID]
	if !ok {
		timer.EngineTimeCounterAdd()
		return nil, types.ErrInvalidMarketID
	}

	if order.Status == types.Order_Active {
		// we're submitting an active order
		metrics.OrderGaugeAdd(1, order.MarketID)
	}
	conf, err := mkt.SubmitOrder(order)
	if err != nil {
		timer.EngineTimeCounterAdd()
		return nil, err
	}
	// order was filled by submitting it to the market -> the matching engine worked
	if conf.Order.Status == types.Order_Filled {
		metrics.OrderGaugeAdd(-1, order.MarketID)
	}
	timer.EngineTimeCounterAdd()
	return conf, nil
}

// AmendOrder take order amendment details and attempts to amend the order
// if it exists and is in a state to be edited.
func (e *Engine) AmendOrder(orderAmendment *types.OrderAmendment) (*types.OrderConfirmation, error) {
	e.log.Debug("Amend order")

	mkt, ok := e.markets[orderAmendment.MarketID]
	if !ok {
		return nil, types.ErrInvalidMarketID
	}

	// we're passing a pointer here, so we need the wasActive var to be certain we're checking the original
	// order status. It's possible order.Status will reflect the new status value if we don't
	conf, err := mkt.AmendOrder(orderAmendment)
	if err != nil {
		return nil, err
	}
	// order was active, not anymore -> decrement gauge
	if conf.Order.Status != types.Order_Active {
		metrics.OrderGaugeAdd(-1, orderAmendment.MarketID)
	}
	return conf, nil
}

// CancelOrder takes order details and attempts to cancel if it exists in matching engine, stores etc.
func (e *Engine) CancelOrder(order *types.Order) (*types.OrderCancellationConfirmation, error) {
	e.log.Debug("Cancel order")
	mkt, ok := e.markets[order.MarketID]
	if !ok {
		return nil, types.ErrInvalidMarketID
	}

	// Cancel order in matching engine
	conf, err := mkt.CancelOrder(order)
	if err != nil {
		return nil, err
	}
	if conf.Order.Status == types.Order_Cancelled {
		metrics.OrderGaugeAdd(-1, order.MarketID)
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
	tnano := t.UnixNano()
	for _, mkt := range e.markets {
		ordrs, err := mkt.RemoveExpiredOrders(tnano)
		if err != nil {
			e.log.Error("unable to get remove expired orders",
				logging.String("market-id", mkt.GetID()),
				logging.Error(err))
		}
		expiringOrders = append(
			expiringOrders, ordrs...)
	}

	if e.log.GetLevel() == logging.DebugLevel {
		e.log.Debug("Removed expired orders from matching engine",
			logging.Int("orders-removed", len(expiringOrders)))
	}

	for _, order := range expiringOrders {
		order := order
		e.orderBuf.Add(order)
		// order expired, decrement gauge
		metrics.OrderGaugeAdd(-1, order.MarketID)
	}

	e.log.Debug("Updated expired orders in stores",
		logging.Int("orders-removed", len(expiringOrders)))
	timer.EngineTimeCounterAdd()
}

// Generate creates any data (including storing state changes) in the underlying stores.
// TODO(): maybe call this in onChainTimeUpdate, when the chain time is updated
func (e *Engine) Generate() error {
	err := e.accountBuf.Flush()
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("Failed to commit accounts"))
	}
	err = e.orderBuf.Flush()
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("Failed to commit orders"))
	}
	err = e.tradeBuf.Flush()
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("Failed to commit trades"))
	}
	// do not check errors here as they only happend when a party is created
	// twice, which should not be a problem
	_ = e.partyBuf.Flush()

	err = e.transferBuf.Flush()
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("Failed to commit transfers"))
	}
	err = e.marketBuf.Flush()
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("Failed to commit markets"))
	}

	return nil
}
