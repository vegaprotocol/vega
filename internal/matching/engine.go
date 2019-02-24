package matching

import (
	"fmt"
	"github.com/pkg/errors"
	types "vega/proto"
)

type Engine interface {
	AddOrderBook(marketId string) error
	CancelOrder(order *types.Order) (*types.OrderCancellation, types.OrderError)
	SubmitOrder(order *types.Order) (*types.OrderConfirmation, types.OrderError)
	DeleteOrder(order *types.Order)
	RemoveExpiringOrders(timestamp uint64) []types.Order
	AmendOrder(order *types.Order) types.OrderError
}

type matchingEngine struct {
	*Config
	markets map[string]*OrderBook
}

func NewMatchingEngine(config *Config) Engine {
	return &matchingEngine{
		Config:  config,
		markets: make(map[string]*OrderBook),
	}
}

func (me *matchingEngine) AddOrderBook(marketId string) error {
	if _, exists := me.markets[marketId]; !exists {
		book := NewBook(marketId, me.Config)
		me.markets[marketId] = book
		return nil
	} else {
		return errors.New(fmt.Sprintf("Order book for market %s already exists in matching engine", marketId))
	}
}

func (me *matchingEngine) SubmitOrder(order *types.Order) (*types.OrderConfirmation, types.OrderError) {
	market, exists := me.markets[order.Market]
	if !exists {
		return nil, types.OrderError_INVALID_MARKET_ID
	}

	confirmationMessage, err := market.AddOrder(order)
	if err != types.OrderError_NONE {
		return nil, err
	}

	return confirmationMessage, types.OrderError_NONE
}

func (me *matchingEngine) DeleteOrder(order *types.Order) {
	if market, exists := me.markets[order.Market]; exists {
		market.RemoveOrder(order)
	}
}

func (me *matchingEngine) CancelOrder(order *types.Order) (*types.OrderCancellation, types.OrderError) {
	market, exists := me.markets[order.Market]
	if !exists {
		return nil, types.OrderError_INVALID_MARKET_ID
	}
	cancellationResult, err := market.CancelOrder(order)
	if err != types.OrderError_NONE {
		return nil, err
	}
	return cancellationResult, types.OrderError_NONE
}

func (me *matchingEngine) RemoveExpiringOrders(timestamp uint64) []types.Order {
	var expiringOrders []types.Order
	for _, market := range me.markets {
		expiringOrders = append(expiringOrders, market.RemoveExpiredOrders(timestamp)...)
	}
	return expiringOrders
}

func (me *matchingEngine) AmendOrder(order *types.Order) types.OrderError {
	if market, exists := me.markets[order.Market]; exists {
		return market.AmendOrder(order)
	}
	return types.OrderError_INVALID_MARKET_ID
}
