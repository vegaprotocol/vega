package core

import (
	"vega/api"
	"vega/datastore"
	"vega/matching"
	"vega/proto"
	"context"
)

type Vega struct {
	config         *Config
	markets        map[string]*matching.OrderBook
	OrdersService  api.OrderService
	TradesService  api.TradeService
	matchingEngine matching.MatchingEngine
}

const storeChannelSize = 2 << 16
const marketName = "BTC/DEC18"

type Config struct{}

func New(config *Config) *Vega {
	// Storage Service provides read stores for consumer VEGA API
	// Uses in memory storage (maps/slices etc), configurable in future
	storeOrderChan := make(chan msg.Order, storeChannelSize)
	storeTradeChan := make(chan msg.Trade, storeChannelSize)

	storage := &datastore.MemoryStoreProvider{}
	storage.Init([]string{marketName}, storeOrderChan, storeTradeChan)

	// Initialise concrete consumer services
	orderService := api.NewOrderService()
	tradeService := api.NewTradeService()
	orderService.Init(storage.OrderStore())
	tradeService.Init(storage.TradeStore())

	// Initialise matching engine
	matchingEngine := matching.NewMatchingEngine()

	return &Vega{
		config:         config,
		markets:        make(map[string]*matching.OrderBook),
		OrdersService:  &orderService,
		TradesService:  &tradeService,
		matchingEngine: &matchingEngine,
	}
}

func GetConfig() *Config {
	return &Config{}
}

func (v *Vega) InitialiseMarkets() {
	v.matchingEngine.CreateMarket(marketName)
}

func (v *Vega) SubmitOrder(order *msg.Order) (*msg.OrderConfirmation, msg.OrderError){
	vegaCtx := context.Background()

	//----------------- MATCHING ENGINE --------------//
	// send order to matching engine
	confirmation, err := v.matchingEngine.SubmitOrder(order)
	if err == msg.OrderError_NONE {
		// some error handling
		return nil, err
	}

	// -----------------------------------------------//
	//-------------------- STORES --------------------//
	// if OK send to stores

	// insert aggressive remaing order
	v.OrdersService.CreateOrder(vegaCtx, *order)

	// insert all passive orders siting on the book
	//for _, order := range confirmation.PassiveOrdersAffected {
		// UpdateOrders TBD
		//v.OrdersService.UpdateOrders(vegaCtx, *order)
	//}

	// insert all trades resulted from the executed order
	//for _, trade := range confirmation.Trades {
		// CreateTrade TBD
		//v.TradesService.CreateTrade(vegaCtx, *trade)
	//}

	// ------------------------------------------------//
	//------------------- RISK ENGINE -----------------//

	// SOME STUFF

	// ------------------------------------------------//

	return confirmation, msg.OrderError_NONE
}
