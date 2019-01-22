package execution

import (
	"vega/msg"
	"vega/internal/logging"
	"vega/log"
	"vega/matching"
	"vega/datastore"
	"fmt"
	"vega/vegatime"
)

type Config struct {
	log logging.Logger
}

type Engine interface {
	SubmitOrder(order *msg.Order) (*msg.OrderConfirmation, msg.OrderError)
	CancelOrder(order *msg.Order) (*msg.OrderCancellation, msg.OrderError)
	AmendOrder(order *msg.Amendment) (*msg.OrderConfirmation, msg.OrderError)
}

type engine struct {
	Config
	matching matching.MatchingEngine
	orderStore datastore.OrderStore
	time vegatime.Service
}

func NewEngine(matching matching.MatchingEngine, time vegatime.Service, orderStore datastore.OrderStore) Engine {
	config := Config{}
	return &engine{
		config,
		matching,
		orderStore,
		time,
	}
}

func (e *engine) SubmitOrder(order *msg.Order) (*msg.OrderConfirmation, msg.OrderError) {
	e.log.Debug("ExecutionEngine: Submit/create order")
	return nil, msg.OrderError_NONE
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
		return v.OrderCancelReplace(existingOrder, newOrder)
	}
	// if decrease in size or change in expiration date
	// ---> DO amend in place in matching engine
	if expiryChange || sizeDecrease {
		return v.OrderAmendInPlace(newOrder)
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
	err := e.orderStore.Put(order)
	if err != nil {
		// Note: writing to store should not prevent flow to other engines
		log.Errorf("OrderStore.Put error: %v", err)
	}

	// ------------------------------------------------//
	return cancellation, msg.OrderError_NONE
}