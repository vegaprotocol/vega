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
	AddTradeToBuffer(trade types.Trade) error
	GenerateCandlesFromBuffer(market string) error
	StartNewBuffer(marketId string, at time.Time) error
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

	protomarkets := []types.Market{}
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
		protomarkets = append(protomarkets, mkt)

		e.log.Info("New market loaded from configuation",
			logging.String("market-config", path),
			logging.String("market-id", mkt.Id))
	}

	// existing markets are to be loaded via the marketStore as market proto types and can be added at runtime via TM
	for _, mkt := range protomarkets {
		mkt := mkt
		err := e.marketStore.Post(&mkt)
		if err != nil {
			e.log.Panic("Failed to add default market to market store",
				logging.String("market-id", mkt.Id),
				logging.Error(err),
			)
		}
		e.markets[mkt.Id], err = engines.NewMarket(executionConfig.Engines, &mkt)
		if err != nil {
			e.log.Panic("Failed to instanciate market market",
				logging.String("market-id", mkt.Id),
				logging.Error(err),
			)
		}
	}

	return e
}

func (e *Engine) SubmitOrder(order *types.Order) (*types.OrderConfirmation, error) {
	e.log.Debug("Submit order", logging.Order(*order))
	mkt, ok := e.markets[order.Market]
	if !ok {
		return nil, types.ErrInvalidMarketID
	}

	// Verify and add new parties
	party, _ := e.partyStore.GetByID(order.Party)
	if party == nil {
		p := &types.Party{Name: order.Party}
		err := e.partyStore.Post(p)
		if err != nil {
			return nil, err
		}
	}

	// Submit order to matching engine
	confirmation, submitError := mkt.SubmitOrder(order)
	if confirmation == nil || submitError != nil {
		e.log.Error("Failure after submit order from matching engine",
			logging.Order(*order),
			logging.Error(submitError))

		return nil, submitError
	}

	// Insert aggressive remaining order
	err := e.orderStore.Post(*order)
	if err != nil {
		e.log.Error("Failure storing new order in execution engine (submit)", logging.Error(err))
	}
	if confirmation.PassiveOrdersAffected != nil {
		// Insert all passive orders siting on the book
		for _, order := range confirmation.PassiveOrdersAffected {
			// Note: writing to store should not prevent flow to other engines
			err := e.orderStore.Put(*order)
			if err != nil {
				e.log.Error("Failure storing order update in execution engine (submit)",
					logging.Order(*order),
					logging.Error(err))
			}
		}
	}

	if confirmation.Trades != nil {
		// insert all trades resulted from the executed order
		for idx, trade := range confirmation.Trades {
			trade.Id = fmt.Sprintf("%s-%010d", order.Id, idx)
			if order.Side == types.Side_Buy {
				trade.BuyOrder = order.Id
				trade.SellOrder = confirmation.PassiveOrdersAffected[idx].Id
			} else {
				trade.SellOrder = order.Id
				trade.BuyOrder = confirmation.PassiveOrdersAffected[idx].Id
			}

			if err := e.tradeStore.Post(trade); err != nil {
				e.log.Error("Failure storing new trade in execution engine (submit)",
					logging.Trade(*trade),
					logging.Error(err))
			}

			// Save to trade buffer for generating candles etc
			err := e.candleStore.AddTradeToBuffer(*trade)
			if err != nil {
				e.log.Error("Failure adding trade to candle buffer in execution engine (submit)",
					logging.Trade(*trade),
					logging.Error(err))
			}
		}
	}

	return confirmation, nil
}

// AmendOrder take order amendment details and attempts to amend the order
// if it exists and is in a state to be edited.
func (e *Engine) AmendOrder(order *types.OrderAmendment) (*types.OrderConfirmation, error) {
	e.log.Debug("Amend order")
	ctx := context.TODO()
	existingOrder, err := e.orderStore.GetByPartyAndId(ctx, order.Party, order.Id)
	if err != nil {
		e.log.Error("Invalid order reference",
			logging.String("id", order.Id),
			logging.String("party", order.Party),
			logging.Error(err))

		return &types.OrderConfirmation{}, types.ErrInvalidOrderReference
	}

	e.log.Debug("Existing order found", logging.Order(*existingOrder))

	timestamp, err := e.time.GetTimeNow()
	if err != nil {
		e.log.Error("Failed to obtain current vega time", logging.Error(err))
		return &types.OrderConfirmation{}, types.ErrVegaTimeFailure
	}

	newOrder := types.OrderPool.Get().(*types.Order)
	newOrder.Id = existingOrder.Id
	newOrder.Market = existingOrder.Market
	newOrder.Party = existingOrder.Party
	newOrder.Side = existingOrder.Side
	newOrder.Price = existingOrder.Price
	newOrder.Size = existingOrder.Size
	newOrder.Remaining = existingOrder.Remaining
	newOrder.Type = existingOrder.Type
	newOrder.CreatedAt = timestamp.UnixNano()
	newOrder.Status = existingOrder.Status
	newOrder.ExpiresAt = existingOrder.ExpiresAt
	newOrder.Reference = existingOrder.Reference

	var (
		priceShift, sizeIncrease, sizeDecrease, expiryChange = false, false, false, false
	)

	if order.Price != 0 && existingOrder.Price != order.Price {
		newOrder.Price = order.Price
		priceShift = true
	}

	if order.Size != 0 {
		newOrder.Size = order.Size
		newOrder.Remaining = order.Size
		if order.Size > existingOrder.Size {
			sizeIncrease = true
		}
		if order.Size < existingOrder.Size {
			sizeDecrease = true
		}
	}

	if newOrder.Type == types.Order_GTT && order.ExpiresAt != 0 {
		newOrder.ExpiresAt = order.ExpiresAt
		expiryChange = true
	}

	// if increase in size or change in price
	// ---> DO atomic cancel and submit
	if priceShift || sizeIncrease {
		return e.orderCancelReplace(existingOrder, newOrder)
	}
	// if decrease in size or change in expiration date
	// ---> DO amend in place in matching engine
	if expiryChange || sizeDecrease {
		return e.orderAmendInPlace(newOrder)
	}

	e.log.Error("Order amendment not allowed", logging.Order(*existingOrder))
	return &types.OrderConfirmation{}, types.ErrEditNotAllowed
}

// CancelOrder takes order details and attempts to cancel if it exists in matching engine, stores etc.
func (e *Engine) CancelOrder(order *types.Order) (*types.OrderCancellationConfirmation, error) {
	e.log.Debug("Cancel order")
	mkt, ok := e.markets[order.Market]
	if !ok {
		return nil, types.ErrInvalidMarketID
	}

	// Cancel order in matching engine
	cancellation, cancelError := mkt.CancelOrder(order)
	if cancellation == nil || cancelError != nil {
		e.log.Panic("Failure after cancel order from matching engine",
			logging.Order(*order),
			logging.Error(cancelError))

		return nil, cancelError
	}

	// Update the order in our stores (will be marked as cancelled)
	err := e.orderStore.Put(*order)
	if err != nil {
		e.log.Error("Failure storing order update in execution engine (cancel)",
			logging.Order(*order),
			logging.Error(err))
	}

	return cancellation, nil
}

func (e *Engine) orderCancelReplace(existingOrder, newOrder *types.Order) (*types.OrderConfirmation, error) {
	e.log.Debug("Cancel/replace order")

	cancellation, cancelError := e.CancelOrder(existingOrder)
	if cancellation == nil || cancelError != nil {
		e.log.Error("Failure after cancel order from matching engine (cancel/replace)",
			logging.OrderWithTag(*existingOrder, "existing-order"),
			logging.OrderWithTag(*newOrder, "new-order"),
			logging.Error(cancelError))

		return &types.OrderConfirmation{}, cancelError
	}

	return e.SubmitOrder(newOrder)
}

func (e *Engine) orderAmendInPlace(newOrder *types.Order) (*types.OrderConfirmation, error) {
	mkt, ok := e.markets[newOrder.Market]
	if !ok {
		return nil, types.ErrInvalidMarketID
	}

	amendError := mkt.AmendOrder(newOrder)
	if amendError != nil {
		e.log.Error("Failure after amend order from matching engine (amend-in-place)",
			logging.OrderWithTag(*newOrder, "new-order"),
			logging.Error(amendError))

		return &types.OrderConfirmation{}, amendError
	}
	err := e.orderStore.Put(*newOrder)
	if err != nil {
		e.log.Error("Failure storing order update in execution engine (amend-in-place)",
			logging.Order(*newOrder),
			logging.Error(err))
		// todo: txn or other strategy (https://gitlab.com/vega-protocol/trading-core/issues/160)
	}
	return &types.OrderConfirmation{}, nil
}

func (e *Engine) StartCandleBuffer() error {

	// Load current vega-time from the blockchain (via time service)
	stamp, err := e.time.GetTimeNow()
	if err != nil {
		return errors.Wrap(err, "Failed to obtain current time from vega-time service")
	}

	// We need a buffer for all current markets on VEGA
	for _, mkt := range e.markets {
		e.log.Debug(
			"Starting candle buffer for market",
			logging.String("market-id", mkt.GetID()),
		)

		err := e.candleStore.StartNewBuffer(mkt.GetID(), stamp)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("Failed to start new candle buffer for market %s", mkt.GetID()))
		}
	}

	return nil
}

// Process any data updates (including state changes)
// e.g. removing expired orders from matching engine.
func (e *Engine) Process() error {
	e.log.Debug("Removing expiring orders from matching engine")

	epochTimeNano, err := e.time.GetTimeNow()
	if err != nil {
		return err
	}

	expiringOrders := []types.Order{}
	for _, mkt := range e.markets {
		expiringOrders = append(
			expiringOrders, mkt.RemoveExpiredOrders(epochTimeNano.UnixNano())...)
	}

	e.log.Debug("Removed expired orders from matching engine",
		logging.Int("orders-removed", len(expiringOrders)))

	for _, order := range expiringOrders {
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
	err = e.StartCandleBuffer()
	if err != nil {
		return err
	}

	return nil
}

// Generate creates any data (including storing state changes) in the underlying stores.
func (e *Engine) Generate() error {

	for _, mkt := range e.markets {
		// We need a buffer for all current markets on VEGA
		err := e.candleStore.GenerateCandlesFromBuffer(mkt.GetID())
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("Failed to generate candles from buffer for market %s", mkt.GetID()))
		}
		err = e.orderStore.Commit()
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
