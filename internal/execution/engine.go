package execution

import (
	"fmt"

	types "vega/proto"

	"vega/internal/matching"
	"vega/internal/storage"
	"vega/internal/vegatime"
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
	e.log.Debug("ExecutionEngine: Submit/create order")

	// 1) submit order to matching engine
	confirmation, errortypes := e.matching.SubmitOrder(order)
	if confirmation == nil || errortypes != types.OrderError_NONE {
		return nil, errortypes
	}

	// 2) Call out to risk engine calculation every N blocks

	// 3) save to stores
	// insert aggressive remaining order
	err := e.orderStore.Post(*order)
	if err != nil {
		// Note: writing to store should not prevent flow to other engines
		e.log.Errorf("ExecutionEngine: order storage error: %s", err)
	}
	if confirmation.PassiveOrdersAffected != nil {
		// insert all passive orders siting on the book
		for _, order := range confirmation.PassiveOrdersAffected {
			// Note: writing to store should not prevent flow to other engines
			err := e.orderStore.Put(*order)
			if err != nil {
				e.log.Errorf("ExecutionEngine: order storage update error: %s", err)
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
				// Note: writing to store should not prevent flow to other engines
				e.log.Errorf("ExecutionEngine: trade storage error: %+v", err)
			}

			// Save to trade buffer for generating candles etc
			//v.AddTradeToCandleBuffer(trade)
			//v.Statistics.LastTrade = trade
		}
	}

	// 4) create or update risk record for this order party etc

	return confirmation, types.OrderError_NONE
}

func (e *engine) AmendOrder(order *types.Amendment) (*types.OrderConfirmation, types.OrderError) {
	e.log.Debug("ExecutionEngine: Amend order")

	// stores get me order with this reference
	existingOrder, err := e.orderStore.GetByPartyAndId(order.Party, order.Id)
	if err != nil {
		e.log.Errorf("Error: %+v\n", types.OrderError_INVALID_ORDER_REFERENCE)
		return &types.OrderConfirmation{}, types.OrderError_INVALID_ORDER_REFERENCE
	}

	e.log.Debugf("Existing order found: %+v\n", existingOrder)

	timestamp, _, err := e.time.GetTimeNow()
	if err != nil {
		e.log.Errorf("error getting current vega time: %s", err)
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
	newOrder.Timestamp = timestamp.UnixNano()
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

	e.log.Infof("Edit not allowed")
	return &types.OrderConfirmation{}, types.OrderError_EDIT_NOT_ALLOWED
}

func (e *engine) CancelOrder(order *types.Order) (*types.OrderCancellation, types.OrderError) {
	e.log.Info("ExecutionEngine: Cancel order")

	// -----------------------------------------------//
	//----------------- MATCHING ENGINE --------------//
	// 1) cancel order in matching engine
	cancellation, errortypes := e.matching.CancelOrder(order)
	if cancellation == nil || errortypes != types.OrderError_NONE {
		return nil, errortypes
	}

	// -----------------------------------------------//
	//-------------------- STORES --------------------//
	// 2) if OK update stores

	// insert aggressive remaining order
	err := e.orderStore.Put(*order)
	if err != nil {
		// Note: writing to store should not prevent flow to other engines
		e.log.Errorf("OrderStore.Put error: %v", err)
	}

	// ------------------------------------------------//
	return cancellation, types.OrderError_NONE
}

func (e engine) orderCancelReplace(existingOrder, newOrder *types.Order) (*types.OrderConfirmation, types.OrderError) {
	cancellationMessage, errtypes := e.CancelOrder(existingOrder)
	e.log.Debugf("ExecutionEngine: cancellationMessage: %+v", cancellationMessage)
	if errtypes != types.OrderError_NONE {
		e.log.Errorf("Failed to cancel and replace order: %s -> %s (%s)", existingOrder, newOrder, errtypes)
		return &types.OrderConfirmation{}, errtypes
	}
	return e.SubmitOrder(newOrder)
}

func (e *engine) orderAmendInPlace(newOrder *types.Order) (*types.OrderConfirmation, types.OrderError) {
	errtypes := e.matching.AmendOrder(newOrder)
	if errtypes != types.OrderError_NONE {
		e.log.Errorf("Failed to amend in place order: %s (%s)", newOrder, errtypes)
		return &types.OrderConfirmation{}, errtypes
	}
	err := e.orderStore.Put(*newOrder)
	if err != nil {
		e.log.Errorf("Failed to update order store for amend in place: %s - %s", newOrder, err)
	}
	return &types.OrderConfirmation{}, types.OrderError_NONE
}
