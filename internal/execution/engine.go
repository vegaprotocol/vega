package execution

import (
	"context"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"
	"time"

	"github.com/golang/protobuf/jsonpb"
	"github.com/pkg/errors"

	"code.vegaprotocol.io/vega/internal/buffer"
	"code.vegaprotocol.io/vega/internal/collateral"
	"code.vegaprotocol.io/vega/internal/logging"
	"code.vegaprotocol.io/vega/internal/metrics"
	"code.vegaprotocol.io/vega/internal/storage"

	types "code.vegaprotocol.io/vega/proto"
)

var (
	ErrMarketAlreadyExist = errors.New("market already exist")
)

//go:generate go run github.com/golang/mock/mockgen -destination mocks/order_store_mock.go -package mocks code.vegaprotocol.io/vega/internal/execution OrderStore
type OrderStore interface {
	GetByPartyAndId(ctx context.Context, party string, id string) (*types.Order, error)
	Post(order types.Order) error
	Put(order types.Order) error
	Commit() error
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/trade_store_mock.go -package mocks code.vegaprotocol.io/vega/internal/execution TradeStore
type TradeStore interface {
	Commit() error
	Post(trade *types.Trade) error
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/candle_store_mock.go -package mocks code.vegaprotocol.io/vega/internal/execution CandleStore
type CandleStore interface {
	GenerateCandlesFromBuffer(market string, buf map[string]types.Candle) error
	FetchLastCandle(marketID string, interval types.Interval) (*types.Candle, error)
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/market_store_mock.go -package mocks code.vegaprotocol.io/vega/internal/execution MarketStore
type MarketStore interface {
	Post(party *types.Market) error
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/party_store_mock.go -package mocks code.vegaprotocol.io/vega/internal/execution PartyStore
type PartyStore interface {
	GetByID(id string) (*types.Party, error)
	Post(party *types.Party) error
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/time_service_mock.go -package mocks code.vegaprotocol.io/vega/internal/execution TimeService
type TimeService interface {
	GetTimeNow() (time.Time, error)
	NotifyOnTick(f func(time.Time))
}

type Engine struct {
	log *logging.Logger
	Config

	markets      map[string]*Market
	party        *Party
	orderStore   OrderStore
	tradeStore   TradeStore
	candleStore  CandleStore
	marketStore  MarketStore
	partyStore   PartyStore
	time         TimeService
	collateral   *collateral.Engine
	accountBuf   *buffer.Account
	accountStore *storage.Account
}

// NewEngine takes stores and engines and returns
// a new execution engine to process new orders, etc.
func NewEngine(
	log *logging.Logger,
	executionConfig Config,
	time TimeService,
	orderStore OrderStore,
	tradeStore TradeStore,
	candleStore CandleStore,
	marketStore MarketStore,
	partyStore PartyStore,
	accountStore *storage.Account,
) *Engine {
	// setup logger
	log = log.Named(namedLogger)
	log.SetLevel(executionConfig.Level.Get())

	accountBuf := buffer.NewAccount(accountStore)

	//  create collateral
	cengine, err := collateral.New(log, executionConfig.Collateral, accountBuf)
	if err != nil {
		log.Error("unable to initialize collateral", logging.Error(err))
		return nil
	}

	e := &Engine{
		log:          log,
		Config:       executionConfig,
		markets:      map[string]*Market{},
		candleStore:  candleStore,
		orderStore:   orderStore,
		tradeStore:   tradeStore,
		marketStore:  marketStore,
		partyStore:   partyStore,
		time:         time,
		collateral:   cengine,
		accountStore: accountStore,
		accountBuf:   accountBuf,
	}

	pmkts := []types.Market{}
	// loads markets from configuration
	for _, v := range executionConfig.Markets.Configs {
		path := filepath.Join(executionConfig.Markets.Path, v)
		buf, err := ioutil.ReadFile(path)
		if err != nil {
			e.log.Panic("Unable to read market configuration",
				logging.Error(err),
				logging.String("config-path", path))
		}

		mkt := types.Market{}
		err = jsonpb.Unmarshal(strings.NewReader(string(buf)), &mkt)
		if err != nil {
			e.log.Panic("Unable to unmarshal market configuration",
				logging.Error(err),
				logging.String("config-path", path))
		}

		e.log.Info("NewModel market loaded from configuation",
			logging.String("market-config", path),
			logging.String("market-id", mkt.Id))

		err = e.SubmitMarket(&mkt)
		if err != nil {
			e.log.Panic("Unable to submit market",
				logging.Error(err))
		}
		pmkts = append(pmkts, mkt)
	}

	// create the party engine
	e.party = NewParty(log, e.collateral, pmkts, e.partyStore)

	e.time.NotifyOnTick(e.onChainTimeUpdate)

	return e
}

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

func (e *Engine) NotifyTraderAccount(notif *types.NotifyTraderAccount) error {
	return e.party.NotifyTraderAccount(notif)
}

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
		mktconfig,
		e.candleStore,
		e.orderStore,
		e.partyStore,
		e.tradeStore,
		e.accountStore,
		now,
		uint64(len(e.markets)),
	)
	if err != nil {
		e.log.Error("Failed to instanciate market",
			logging.String("market-name", mktconfig.Name),
			logging.Error(err),
		)
	}

	err = e.marketStore.Post(mktconfig)
	if err != nil {
		e.log.Error("Failed to add default market to market store",
			logging.String("market-name", mktconfig.Name),
			logging.Error(err),
		)
	}

	e.markets[mktconfig.Id] = mkt

	// create market accounts
	asset, err := mktconfig.GetAsset()
	if err != nil {
		return err
	}

	// ignore response ids here + this cannot fail
	_, _ = e.collateral.CreateMarketAccounts(mktconfig.Id, asset, 0)

	return nil
}

func (e *Engine) SubmitOrder(order *types.Order) (*types.OrderConfirmation, error) {
	if e.log.GetLevel() == logging.DebugLevel {
		e.log.Debug("Submit order", logging.Order(*order))
	}

	mkt, ok := e.markets[order.MarketID]
	if !ok {
		return nil, types.ErrInvalidMarketID
	}

	if order.Status == types.Order_Active {
		// we're submitting an active order
		metrics.OrderGaugeAdd(1, order.MarketID)
	}
	conf, err := mkt.SubmitOrder(order)
	if err != nil {
		return nil, err
	}
	// order was filled by submitting it to the market -> the matching engine worked
	if conf.Order.Status == types.Order_Filled {
		metrics.OrderGaugeAdd(-1, order.MarketID)
	}
	return conf, nil
}

// AmendOrder take order amendment details and attempts to amend the order
// if it exists and is in a state to be edited.
func (e *Engine) AmendOrder(orderAmendment *types.OrderAmendment) (*types.OrderConfirmation, error) {
	e.log.Debug("Amend order")
	// try to get the order first
	order, err := e.orderStore.GetByPartyAndId(
		context.Background(), orderAmendment.PartyID, orderAmendment.OrderID)
	if err != nil {
		e.log.Error("Invalid order reference",
			logging.String("id", order.Id),
			logging.String("party", order.PartyID),
			logging.Error(err))

		return nil, types.ErrInvalidOrderReference
	}
	wasActive := order.Status == types.Order_Active
	if e.log.Check(logging.DebugLevel) {
		e.log.Debug("Existing order found", logging.Order(*order))
	}

	mkt, ok := e.markets[order.MarketID]
	if !ok {
		return nil, types.ErrInvalidMarketID
	}

	// we're passing a pointer here, so we need the wasActive var to be certain we're checking the original
	// order status. It's possible order.Status will reflect the new status value if we don't
	conf, err := mkt.AmendOrder(orderAmendment, order)
	if err != nil {
		return nil, err
	}
	// order was active, not anymore -> decrement gauge
	if wasActive && conf.Order.Status != types.Order_Active {
		metrics.OrderGaugeAdd(-1, order.MarketID)
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
	e.log.Debug("updating engine on new time update")

	// remove expired orders
	e.removeExpiredOrders(t)

	// notify markets of the time expiration
	for _, mkt := range e.markets {
		mkt := mkt
		mkt.OnChainTimeUpdate(t)
	}
}

// Process any data updates (including state changes)
// e.g. removing expired orders from matching engine.
func (e *Engine) removeExpiredOrders(t time.Time) {
	pre := time.Now()
	e.log.Debug("Removing expiring orders from matching engine")

	expiringOrders := []types.Order{}
	tnano := t.UnixNano()
	for _, mkt := range e.markets {
		expiringOrders = append(
			expiringOrders, mkt.RemoveExpiredOrders(tnano)...)
	}

	e.log.Debug("Removed expired orders from matching engine",
		logging.Int("orders-removed", len(expiringOrders)))

	for _, order := range expiringOrders {
		order := order
		err := e.orderStore.Put(order)
		if err != nil {
			e.log.Error("error updating store for remove expiring order",
				logging.Order(order),
				logging.Error(err))
		}
		// order expired, decrement gauge
		metrics.OrderGaugeAdd(-1, order.MarketID)
	}

	e.log.Debug("Updated expired orders in stores",
		logging.Int("orders-removed", len(expiringOrders)))
	metrics.EngineTimeCounterAdd(pre, "all", "execution", "removeExpiredOrders")
}

// Generate creates any data (including storing state changes) in the underlying stores.
// TODO(): maybe call this in onChainTimeUpdate, when the chain time is updated
func (e *Engine) Generate() error {
	err := e.accountBuf.Flush()
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("Failed to commit accounts"))
	}

	for _, mkt := range e.markets {
		err := e.orderStore.Commit()
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("Failed to commit orders for market %s", mkt.GetID()))
		}
		err = e.tradeStore.Commit()
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("Failed to commit trades for market %s", mkt.GetID()))
		}
	}

	return nil
}
