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

	types "code.vegaprotocol.io/vega/proto"

	"code.vegaprotocol.io/vega/internal/engines"
	"code.vegaprotocol.io/vega/internal/logging"
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
	FetchMostRecentCandle(marketID string, interval types.Interval, descending bool) (*types.Candle, error)
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
	*Config
	markets     map[string]*engines.Market
	orderStore  OrderStore
	tradeStore  TradeStore
	candleStore CandleStore
	marketStore MarketStore
	partyStore  PartyStore
	time        TimeService
}

// NewEngine takes stores and engines and returns
// a new execution engine to process new orders, etc.
func NewEngine(
	executionConfig *Config,
	time TimeService,
	orderStore OrderStore,
	tradeStore TradeStore,
	candleStore CandleStore,
	marketStore MarketStore,
	partyStore PartyStore,
) *Engine {
	e := &Engine{
		Config:      executionConfig,
		markets:     map[string]*engines.Market{},
		candleStore: candleStore,
		orderStore:  orderStore,
		tradeStore:  tradeStore,
		marketStore: marketStore,
		partyStore:  partyStore,
		time:        time,
	}

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

		e.log.Info("New market loaded from configuation",
			logging.String("market-config", path),
			logging.String("market-id", mkt.Id))

		e.SubmitMarket(&mkt)
	}

	e.time.NotifyOnTick(e.onChainTimeUpdate)

	return e
}

func (e *Engine) SubmitMarket(mkt *types.Market) error {
	if _, ok := e.markets[mkt.Id]; ok {
		return ErrMarketAlreadyExist
	}

	now, _ := e.time.GetTimeNow()
	var err error
	e.markets[mkt.Id], err = engines.NewMarket(
		e.Config.Engines, mkt, e.candleStore, e.orderStore, e.partyStore, e.tradeStore, now)
	if err != nil {
		e.log.Panic("Failed to instanciate market market",
			logging.String("market-id", mkt.Id),
			logging.Error(err),
		)
	}

	err = e.marketStore.Post(mkt)
	if err != nil {
		e.log.Panic("Failed to add default market to market store",
			logging.String("market-id", mkt.Id),
			logging.Error(err),
		)
	}

	return nil
}

func (e *Engine) SubmitOrder(order *types.Order) (*types.OrderConfirmation, error) {
	e.log.Debug("Submit order", logging.Order(*order))
	mkt, ok := e.markets[order.Market]
	if !ok {
		return nil, types.ErrInvalidMarketID
	}

	return mkt.SubmitOrder(order)
}

// AmendOrder take order amendment details and attempts to amend the order
// if it exists and is in a state to be edited.
func (e *Engine) AmendOrder(orderAmendment *types.OrderAmendment) (*types.OrderConfirmation, error) {
	e.log.Debug("Amend order")
	// try to get the order first
	order, err := e.orderStore.GetByPartyAndId(
		context.Background(), orderAmendment.Party, orderAmendment.Id)
	if err != nil {
		e.log.Error("Invalid order reference",
			logging.String("id", order.Id),
			logging.String("party", order.Party),
			logging.Error(err))

		return nil, types.ErrInvalidOrderReference
	}
	e.log.Debug("Existing order found", logging.Order(*order))

	mkt, ok := e.markets[order.Market]
	if !ok {
		return nil, types.ErrInvalidMarketID
	}

	return mkt.AmendOrder(orderAmendment, order)
}

// CancelOrder takes order details and attempts to cancel if it exists in matching engine, stores etc.
func (e *Engine) CancelOrder(order *types.Order) (*types.OrderCancellationConfirmation, error) {
	e.log.Debug("Cancel order")
	mkt, ok := e.markets[order.Market]
	if !ok {
		return nil, types.ErrInvalidMarketID
	}

	// Cancel order in matching engine
	return mkt.CancelOrder(order)
}

/*
func (e *Engine) startCandleBuffer() error {

	// Load current vega-time from the blockchain (via time service)
	t, err := e.time.GetTimeNow()
	if err != nil {
		return errors.Wrap(err, "Failed to obtain current time from vega-time service")
	}

	// We need a buffer for all current markets on VEGA
	for _, mkt := range e.markets {
		e.log.Debug(
			"Starting candle buffer for market",
			logging.String("market-id", mkt.GetID()),
		)

		err := e.candleStore.StartNewBuffer(mkt.GetID(), t)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("Failed to start new candle buffer for market %s", mkt.GetID()))
		}
	}

	return nil
}
*/

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
func (e *Engine) removeExpiredOrders(t time.Time) error {
	e.log.Debug("Removing expiring orders from matching engine")

	expiringOrders := []types.Order{}
	for _, mkt := range e.markets {
		expiringOrders = append(
			expiringOrders, mkt.RemoveExpiredOrders(t.UnixNano())...)
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
	}

	e.log.Debug("Updated expired orders in stores",
		logging.Int("orders-removed", len(expiringOrders)))

	// We need to call start new candle buffer for every block with the current implementation.
	// This ensures that empty candles are created for a timestamp and can fill up. We will
	// hopefully revisit candles in the future and improve the design.
	// err := e.startCandleBuffer()
	// if err != nil {
	// return err
	// }

	return nil
}

// Generate creates any data (including storing state changes) in the underlying stores.
// TODO(): maybe call this in onChainTimeUpdate, when the chain time is updated
func (e *Engine) Generate() error {

	for _, mkt := range e.markets {
		// We need a buffer for all current markets on VEGA
		/*
			err := e.candleStore.GenerateCandlesFromBuffer(mkt.GetID())
			if err != nil {
				return errors.Wrap(err, fmt.Sprintf("Failed to generate candles from buffer for market %s", mkt.GetID()))
			}
		*/
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
