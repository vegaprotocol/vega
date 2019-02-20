package execution

import (
	"fmt"

	types "vega/proto"

	"vega/internal/matching"
	"vega/internal/storage"
	"vega/internal/vegatime"
	"vega/internal/logging"
)

type Engine interface {
	SubmitOrder(order *types.Order) (*types.OrderConfirmation, types.OrderError)
	CancelOrder(order *types.Order) (*types.OrderCancellation, types.OrderError)
	AmendOrder(order *types.Amendment) (*types.OrderConfirmation, types.OrderError)
}

type engine struct {
	*Config
	matching   matching.MatchingEngine
	orderStore storage.OrderStore
	tradeStore storage.TradeStore
	time       vegatime.Service
}

func NewExecutionEngine(config *Config, matching matching.MatchingEngine, time vegatime.Service,
	orderStore storage.OrderStore, tradeStore storage.TradeStore) Engine {
	return &engine{
		config,
		matching,
		orderStore,
		tradeStore,
		time,
	}
}

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
				e.log.Error("Failure storing order update in execution engine (submit)", logging.Error(err))
				// todo: txn or other strategy (https://gitlab.com/vega-protocol/trading-core/issues/160) 
			}
		}
	}

	// Quick way to store a list of parties for stats output (pre party store)
	//if !containsString(v.Statistics.Parties, order.Party) {
	//	v.Statistics.Parties = append(v.Statistics.Parties, order.Party)
	//}
	//v.Statistics.LastOrder = order

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
				e.log.Error("Failure storing new trade in execution engine (submit)", logging.Error(err))
				// todo: txn or other strategy (https://gitlab.com/vega-protocol/trading-core/issues/160)
			}

			// Save to trade buffer for generating candles etc
			//v.AddTradeToCandleBuffer(trade)
			//v.Statistics.LastTrade = trade
			// todo add trades to candle buffer?
		}
	}

	// 4) create or update risk record for this order party etc

	return confirmation, types.OrderError_NONE
}

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
		// todo: the above requires a new order error code to be added
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
	newOrder.Timestamp = uint64(timestamp)
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
		e.log.Error("Failure storing order update in execution engine (cancel)", logging.Error(err))
		// todo: txn or other strategy (https://gitlab.com/vega-protocol/trading-core/issues/160)
	}

	// ------------------------------------------------//
	return cancellation, types.OrderError_NONE
}

func (e engine) orderCancelReplace(existingOrder, newOrder *types.Order) (*types.OrderConfirmation, types.OrderError) {
	e.log.Debug("Cancel/replace order")

	cancellation, cancelError := e.CancelOrder(existingOrder)
	if cancellation == nil || cancelError != types.OrderError_NONE {
		e.log.Error("Failure after cancel order from matching engine (cancel/replace)",
			logging.OrderWithTag(*existingOrder,"existing-order"),
			logging.OrderWithTag(*newOrder,"new-order"),
			logging.String("error", cancelError.String()))

		return &types.OrderConfirmation{}, cancelError
	}

	return e.SubmitOrder(newOrder)
}

func (e *engine) orderAmendInPlace(newOrder *types.Order) (*types.OrderConfirmation, types.OrderError) {
	amendError := e.matching.AmendOrder(newOrder)
	if amendError != types.OrderError_NONE {
		e.log.Error("Failure after amend order from matching engine (amend-in-place)",
			logging.OrderWithTag(*newOrder,"new-order"),
			logging.String("error", amendError.String()))

		return &types.OrderConfirmation{}, amendError
	}
	err := e.orderStore.Put(*newOrder)
	if err != nil {
		e.log.Error("Failure storing order update in execution engine (amend-in-place)", logging.Error(err))
		// todo: txn or other strategy (https://gitlab.com/vega-protocol/trading-core/issues/160)
	}
	return &types.OrderConfirmation{}, types.OrderError_NONE
}
