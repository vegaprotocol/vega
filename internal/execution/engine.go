package execution

import (
	"fmt"
	"github.com/pkg/errors"

	types "vega/proto"

	"vega/internal/logging"
	"vega/internal/matching"
	"vega/internal/storage"
	"vega/internal/vegatime"
)

type Engine interface {
	// SubmitOrder takes a new order request and submits it to the execution engine, storing output etc.
	SubmitOrder(order *types.Order) (*types.OrderConfirmation, types.OrderError)

	// CancelOrder takes order details and attempts to cancel if it exists in matching engine, stores etc.
	CancelOrder(order *types.Order) (*types.OrderCancellation, types.OrderError)

	// AmendOrder take order amendment details and attempts to amend the order
	// if it exists and is in a state to be edited.
	AmendOrder(order *types.Amendment) (*types.OrderConfirmation, types.OrderError)

	// Process any data updates (including state changes)
	// e.g. removing expired orders from matching engine.
	Generate() error

	// Generate creates any data (including storing state changes) in the underlying stores.
	Process() error
}

type engine struct {
	*Config
	markets     []string
	matching    matching.Engine
	orderStore  storage.OrderStore
	tradeStore  storage.TradeStore
	candleStore storage.CandleStore
	marketStore storage.MarketStore
	time        vegatime.Service
}

// NewExecutionEngine takes stores and engines and returns
// a new execution engine to process new orders, etc.
func NewExecutionEngine(
	executionConfig *Config,
	matchingEngine matching.Engine,
	time vegatime.Service,
	orderStore storage.OrderStore,
	tradeStore storage.TradeStore,
	candleStore storage.CandleStore,
	marketStore storage.MarketStore,
) Engine {
	e := &engine{
		Config:      executionConfig,
		markets:     []string{"BTC/DEC19"},
		matching:    matchingEngine,
		orderStore:  orderStore,
		tradeStore:  tradeStore,
		marketStore: marketStore,
		candleStore: candleStore,
		time:        time,
	}

	// existing markets are to be loaded via the marketStore as market proto types and can be added at runtime via TM
	for _, marketId := range e.markets {
		mkt := types.Market{
			Name: marketId,
		}
		err := e.marketStore.Post(&mkt)
		if err != nil {
			e.log.Panic("Failed to add default market to market store",
				logging.String("market-id", marketId),
				logging.Error(err))
		}
		err = e.matching.AddOrderBook(marketId)
		if err != nil {
			e.log.Panic("Failed to add default order book(s) to matching engine",
				logging.String("market-id", marketId),
				logging.Error(err))
		}
	}

	return e
}

// SubmitOrder takes a new order request and submits it to the execution engine, storing output etc.
func (e *engine) SubmitOrder(order *types.Order) (*types.OrderConfirmation, types.OrderError) {
	e.log.Debug("Submit order", logging.Order(*order))

	// 1) submit order to matching engine
	confirmation, submitError := e.matching.SubmitOrder(order)
	if confirmation == nil || submitError != types.OrderError_NONE {
		e.log.Error("Failure after submit order from matching engine",
			logging.Order(*order),
			logging.String("error", submitError.String()))

		return nil, submitError
	}

	// 2) Call out to risk engine calculation every N blocks
	// Removed for now

	// 3) save to stores
	// insert aggressive remaining order
	err := e.orderStore.Post(*order)
	if err != nil {
		// Note: writing to store should not prevent flow to other engines
		e.log.Error("Failure storing new order in execution engine (submit)", logging.Error(err))
		// todo: txn or other strategy (https://gitlab.com/vega-protocol/trading-core/issues/160)
	}
	if confirmation.PassiveOrdersAffected != nil {
		// insert all passive orders siting on the book
		for _, order := range confirmation.PassiveOrdersAffected {
			// Note: writing to store should not prevent flow to other engines
			err := e.orderStore.Put(*order)
			if err != nil {
				e.log.Error("Failure storing order update in execution engine (submit)",
					logging.Order(*order),
					logging.Error(err))
				// todo: txn or other strategy (https://gitlab.com/vega-protocol/trading-core/issues/160)
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
				// todo: txn or other strategy (https://gitlab.com/vega-protocol/trading-core/issues/160)
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

	// 4) create or update risk record for this order party etc

	return confirmation, types.OrderError_NONE
}

// AmendOrder take order amendment details and attempts to amend the order
// if it exists and is in a state to be edited.
func (e *engine) AmendOrder(order *types.Amendment) (*types.OrderConfirmation, types.OrderError) {
	e.log.Debug("Amend order")

	existingOrder, err := e.orderStore.GetByPartyAndId(order.Party, order.Id)
	if err != nil {
		e.log.Error("Invalid order reference",
			logging.String("id", order.Id),
			logging.String("party", order.Party),
			logging.Error(err))

		return &types.OrderConfirmation{}, types.OrderError_INVALID_ORDER_REFERENCE
	}

	e.log.Debug("Existing order found", logging.Order(*existingOrder))

	timestamp, _, err := e.time.GetTimeNow()
	if err != nil {
		e.log.Error("Failed to obtain current vega time", logging.Error(err))
		return &types.OrderConfirmation{}, types.OrderError_ORDER_AMEND_FAILURE
		// todo: new order error code required (gitlab.com/vega-protocol/trading-core/issues/178)
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
	newOrder.Timestamp = timestamp.Uint64()
	newOrder.Status = existingOrder.Status
	newOrder.ExpirationDatetime = existingOrder.ExpirationDatetime
	newOrder.ExpirationTimestamp = existingOrder.ExpirationTimestamp
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

	if newOrder.Type == types.Order_GTT && order.ExpirationTimestamp != 0 && order.ExpirationDatetime != "" {
		newOrder.ExpirationTimestamp = order.ExpirationTimestamp
		newOrder.ExpirationDatetime = order.ExpirationDatetime
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
	return &types.OrderConfirmation{}, types.OrderError_EDIT_NOT_ALLOWED
}

// CancelOrder takes order details and attempts to cancel if it exists in matching engine, stores etc.
func (e *engine) CancelOrder(order *types.Order) (*types.OrderCancellation, types.OrderError) {
	e.log.Debug("Cancel order")

	// Cancel order in matching engine
	cancellation, cancelError := e.matching.CancelOrder(order)
	if cancellation == nil || cancelError != types.OrderError_NONE {
		e.log.Error("Failure after cancel order from matching engine",
			logging.Order(*order),
			logging.String("error", cancelError.String()))

		return nil, cancelError
	}

	// Update the order in our stores (will be marked as cancelled)
	err := e.orderStore.Put(*order)
	if err != nil {
		e.log.Error("Failure storing order update in execution engine (cancel)",
			logging.Order(*order),
			logging.Error(err))
		// todo: txn or other strategy (https://gitlab.com/vega-protocol/trading-core/issues/160)
	}

	return cancellation, types.OrderError_NONE
}

func (e *engine) orderCancelReplace(existingOrder, newOrder *types.Order) (*types.OrderConfirmation, types.OrderError) {
	e.log.Debug("Cancel/replace order")

	cancellation, cancelError := e.CancelOrder(existingOrder)
	if cancellation == nil || cancelError != types.OrderError_NONE {
		e.log.Error("Failure after cancel order from matching engine (cancel/replace)",
			logging.OrderWithTag(*existingOrder, "existing-order"),
			logging.OrderWithTag(*newOrder, "new-order"),
			logging.String("error", cancelError.String()))

		return &types.OrderConfirmation{}, cancelError
	}

	return e.SubmitOrder(newOrder)
}

func (e *engine) orderAmendInPlace(newOrder *types.Order) (*types.OrderConfirmation, types.OrderError) {
	amendError := e.matching.AmendOrder(newOrder)
	if amendError != types.OrderError_NONE {
		e.log.Error("Failure after amend order from matching engine (amend-in-place)",
			logging.OrderWithTag(*newOrder, "new-order"),
			logging.String("error", amendError.String()))

		return &types.OrderConfirmation{}, amendError
	}
	err := e.orderStore.Put(*newOrder)
	if err != nil {
		e.log.Error("Failure storing order update in execution engine (amend-in-place)",
			logging.Order(*newOrder),
			logging.Error(err))
		// todo: txn or other strategy (https://gitlab.com/vega-protocol/trading-core/issues/160)
	}
	return &types.OrderConfirmation{}, types.OrderError_NONE
}

func (e *engine) StartCandleBuffer() error {

	// Load current vega-time from the blockchain (via time service)
	stamp, _, err := e.time.GetTimeNow()
	if err != nil {
		return errors.Wrap(err, "Failed to obtain current time from vega-time service")
	}

	// We need a buffer for all current markets on VEGA
	for _, marketId := range e.markets {
		e.log.Debug("Starting candle buffer for market", logging.String("market-id", marketId))

		err := e.candleStore.StartNewBuffer(marketId, stamp.Uint64())
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("Failed to start new candle buffer for market %s", marketId))
		}
	}

	return nil
}

// Process any data updates (including state changes)
// e.g. removing expired orders from matching engine.
func (e *engine) Process() error {
	e.log.Debug("Removing expiring orders from matching engine")

	epochTimeNano, _, err := e.time.GetTimeNow()
	if err != nil {
		return err
	}

	expiringOrders := e.matching.RemoveExpiringOrders(epochTimeNano.Uint64())

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

	return nil
}

// Generate creates any data (including storing state changes) in the underlying stores.
func (e *engine) Generate() error {

	for _, marketId := range e.markets {
		// We need a buffer for all current markets on VEGA
		err := e.candleStore.GenerateCandlesFromBuffer(marketId)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("Failed to generate candles from buffer for market %s", marketId))
		}
		err = e.orderStore.Commit()
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("Failed to commit orders for market %s", marketId))
		}
		err = e.tradeStore.Commit()
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("Failed to commit trades for market %s", marketId))
		}
	}

	return nil
}
