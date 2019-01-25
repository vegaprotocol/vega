package execution

import (
	"vega/datastore"
	"vega/internal/logging"
	"vega/log"
	"vega/matching"
	"vega/msg"
	"vega/vegatime"
	"fmt"
)

type Config struct {
	log logging.Logger
}

func NewConfig() *Config {
	level := logging.DebugLevel
	logger := logging.NewLogger()
	logger.InitConsoleLogger(level)
	logger.AddExitHandler()
	return &Config{
		log: logger,
	}
}

type Engine interface {
	SubmitOrder(order *msg.Order) (*msg.OrderConfirmation, msg.OrderError)
	CancelOrder(order *msg.Order) (*msg.OrderCancellation, msg.OrderError)
	AmendOrder(order *msg.Amendment) (*msg.OrderConfirmation, msg.OrderError)
}

type engine struct {
	*Config
	matching   matching.MatchingEngine
	orderStore datastore.OrderStore
	tradeStore datastore.TradeStore
	time       vegatime.Service
}

func NewEngine(matching matching.MatchingEngine, time vegatime.Service,
	orderStore datastore.OrderStore, tradeStore datastore.TradeStore) Engine {
	config := NewConfig()
	return &engine{
		config,
		matching,
		orderStore,
		tradeStore,
		time,
	}
}

func (e *engine) SubmitOrder(order *msg.Order) (*msg.OrderConfirmation, msg.OrderError) {
	e.log.Debug("ExecutionEngine: Submit/create order")

	// 1) submit order to matching engine
	confirmation, errorMsg := e.matching.SubmitOrder(order)
	if confirmation == nil || errorMsg != msg.OrderError_NONE {
		return nil, errorMsg
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
			if order.Side == msg.Side_Buy {
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
	
	return confirmation, msg.OrderError_NONE
}

func (e *engine) AmendOrder(order *msg.Amendment) (*msg.OrderConfirmation, msg.OrderError) {
	e.log.Debug("ExecutionEngine: Amend order")

	// stores get me order with this reference
	existingOrder, err := e.orderStore.GetByPartyAndId(order.Party, order.Id)
	if err != nil {
		e.log.Errorf("Error: %+v\n", msg.OrderError_INVALID_ORDER_REFERENCE)
		return &msg.OrderConfirmation{}, msg.OrderError_INVALID_ORDER_REFERENCE
	}

	log.Debugf("Existing order found: %+v\n", existingOrder)

	timestamp, _, err := e.time.GetTimeNow()
	if err != nil {
		e.log.Errorf("error getting current vega time: %s", err)
	}

	newOrder := msg.OrderPool.Get().(*msg.Order)
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

	if newOrder.Type == msg.Order_GTT && order.ExpirationTimestamp != 0 && order.ExpirationDatetime != "" {
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

	log.Infof("Edit not allowed")
	return &msg.OrderConfirmation{}, msg.OrderError_EDIT_NOT_ALLOWED
}

func (e *engine) CancelOrder(order *msg.Order) (*msg.OrderCancellation, msg.OrderError) {
	e.log.Info("ExecutionEngine: Cancel order")

	// -----------------------------------------------//
	//----------------- MATCHING ENGINE --------------//
	// 1) cancel order in matching engine
	cancellation, errorMsg := e.matching.CancelOrder(order)
	if cancellation == nil || errorMsg != msg.OrderError_NONE {
		return nil, errorMsg
	}

	// -----------------------------------------------//
	//-------------------- STORES --------------------//
	// 2) if OK update stores

	// insert aggressive remaining order
	err := e.orderStore.Put(*order)
	if err != nil {
		// Note: writing to store should not prevent flow to other engines
		log.Errorf("OrderStore.Put error: %v", err)
	}

	// ------------------------------------------------//
	return cancellation, msg.OrderError_NONE
}

func (e engine) orderCancelReplace(existingOrder, newOrder *msg.Order) (*msg.OrderConfirmation, msg.OrderError) {
	cancellationMessage, errMsg := e.CancelOrder(existingOrder)
	log.Debugf("ExecutionEngine: cancellationMessage: %+v", cancellationMessage)
	if errMsg != msg.OrderError_NONE {
		log.Errorf("Failed to cancel and replace order: %s -> %s (%s)", existingOrder, newOrder, errMsg)
		return &msg.OrderConfirmation{}, errMsg
	}
	return e.SubmitOrder(newOrder)
}

func (e *engine) orderAmendInPlace(newOrder *msg.Order) (*msg.OrderConfirmation, msg.OrderError) {
	errMsg := e.matching.AmendOrder(newOrder)
	if errMsg != msg.OrderError_NONE {
		log.Errorf("Failed to amend in place order: %s (%s)", newOrder, errMsg)
		return &msg.OrderConfirmation{}, errMsg
	}
	err := e.orderStore.Put(*newOrder)
	if err != nil {
		log.Errorf("Failed to update order store for amend in place: %s - %s", newOrder, err)
	}
	return &msg.OrderConfirmation{}, msg.OrderError_NONE
}
