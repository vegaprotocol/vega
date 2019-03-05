package matching

import (
	"fmt"
	types "vega/proto"

	"github.com/pkg/errors"
)

type Engine interface {
	AddOrderBook(marketId string) error
	CancelOrder(order *types.Order) (*types.OrderCancellation, error)
	SubmitOrder(order *types.Order) (*types.OrderConfirmation, error)
	DeleteOrder(order *types.Order)
	RemoveExpiringOrders(timestamp uint64) []types.Order
	AmendOrder(order *types.Order) error
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
		// ProRataMode is not usually enabled on a continuous trading order book,
		// but when we get to discrete trading and auctions it’s possible.
		book := NewBook(me.Config, marketId, false)
		me.markets[marketId] = book
		return nil
	} else {
		return errors.New(fmt.Sprintf("Order book for market %s already exists in matching engine", marketId))
	}
}

func (me *matchingEngine) SubmitOrder(order *types.Order) (*types.OrderConfirmation, error) {
	market, exists := me.markets[order.Market]
	if !exists {
		return nil, types.ErrInvalidMarketID
	}

	confirmationMessage, err := market.AddOrder(order)
	if err != types.OrderError_NONE {
		return nil, err
	}

	return confirmationMessage, nil
}

func (me *matchingEngine) DeleteOrder(order *types.Order) {
	if market, exists := me.markets[order.Market]; exists {
		market.RemoveOrder(order)
	}
}

func (me *matchingEngine) CancelOrder(order *types.Order) (*types.OrderCancellation, error) {
	market, exists := me.markets[order.Market]
	if !exists {
		return nil, types.ErrInvalidMarketID
	}
	cancellationResult, err := market.CancelOrder(order)
	if err != nil {
		return nil, err
	}
	return cancellationResult, nil
}

func (me *matchingEngine) RemoveExpiringOrders(timestamp uint64) []types.Order {
	var expiringOrders []types.Order
	for _, market := range me.markets {
		expiringOrders = append(expiringOrders, market.RemoveExpiredOrders(timestamp)...)
	}
	return expiringOrders
}

func (me *matchingEngine) AmendOrder(order *types.Order) error {
	if market, exists := me.markets[order.Market]; exists {
		return market.AmendOrder(order)
	}
	return types.ErrInvalidMarketID
}
