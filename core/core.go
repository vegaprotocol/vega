package core

import (
	"time"

	"vega/datastore"
	"vega/matching"
	"vega/proto"
)

type Vega struct {
	config         *Config
	markets        map[string]*matching.OrderBook
	OrdersStore    datastore.OrderStore
	TradesStore    datastore.TradeStore
	matchingEngine matching.MatchingEngine

	abciHeight int64
}

const marketName = "BTC/DEC18"

const genesisTimeStr = "2018-07-09T12:00:00Z"

type Config struct{}

func New(config *Config, store *datastore.MemoryStoreProvider) *Vega {

	// Initialise matching engine
	matchingEngine := matching.NewMatchingEngine()

	return &Vega{
		config:         config,
		markets:        make(map[string]*matching.OrderBook),
		OrdersStore:    store.OrderStore(),
		TradesStore:    store.TradeStore(),
		matchingEngine: matchingEngine,
		abciHeight:     0,
	}
}

func GetConfig() *Config {
	return &Config{}
}

func (v *Vega) SetAbciHeight(n int64) {
	v.abciHeight = n
}

func (v *Vega) GetAbciHeight() int64 {
	return v.abciHeight
}

func (v *Vega) GetTime() time.Time {
	genesisTime, _ := time.Parse(time.RFC3339, genesisTimeStr)
	return genesisTime.Add(time.Duration(v.abciHeight) * time.Second)
}

func (v *Vega) InitialiseMarkets() {
	v.matchingEngine.CreateMarket(marketName)
}

func (v *Vega) SubmitOrder(order *msg.Order) (*msg.OrderConfirmation, msg.OrderError) {

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
	v.OrdersStore.Post(*datastore.NewOrderFromProtoMessage(order))

	// insert all passive orders siting on the book
	for _, order := range confirmation.PassiveOrdersAffected {
		//UpdateOrders TBD
		v.OrdersStore.Put(*datastore.NewOrderFromProtoMessage(order))
	}

	// insert all trades resulted from the executed order
	for _, trade := range confirmation.Trades {
		//CreateTrade TBD
		v.TradesStore.Post(*datastore.NewTradeFromProtoMessage(trade, order.Id))
	}

	// ------------------------------------------------//
	//------------------- RISK ENGINE -----------------//

	// SOME STUFF

	// ------------------------------------------------//

	return confirmation, msg.OrderError_NONE
}
