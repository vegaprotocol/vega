package core

import (
	"fmt"
	"time"

	"vega/datastore"
	"vega/log"
	"vega/matching"
	"vega/msg"
)

const marketName = "BTC/DEC18"

const genesisTimeStr = "2018-07-05T13:36:01Z"

type Config struct{}

type Vega struct {
	config         *Config
	markets        map[string]*matching.OrderBook
	OrdersStore    datastore.OrderStore
	TradesStore    datastore.TradeStore
	matchingEngine matching.MatchingEngine
	State          *State
}

func New(config *Config, store *datastore.MemoryStoreProvider) *Vega {

	// Initialise matching engine
	matchingEngine := matching.NewMatchingEngine()

	return &Vega{
		config:         config,
		markets:        make(map[string]*matching.OrderBook),
		OrdersStore:    store.OrderStore(),
		TradesStore:    store.TradeStore(),
		matchingEngine: matchingEngine,
		State:          newState(),
	}
}

func GetConfig() *Config {
	return &Config{}
}

func (v *Vega) GetAbciHeight() int64 {
	return v.State.Height
}

func (v *Vega) GetTime() time.Time {
	genesisTime, _ := time.Parse(time.RFC3339, genesisTimeStr)
	return genesisTime.Add(time.Duration(v.State.Height) * time.Second)
}

func (v *Vega) InitialiseMarkets() {
	v.matchingEngine.CreateMarket(marketName)
}

func (v *Vega) SubmitOrder(order *msg.Order) (*msg.OrderConfirmation, msg.OrderError) {

	order.Id = fmt.Sprintf("V%d-%d", v.State.Height, v.State.Size)
	order.Timestamp = uint64(v.State.Height)

	//----------------- MATCHING ENGINE --------------//
	// send order to matching engine
	confirmation, err := v.matchingEngine.SubmitOrder(order)
	if confirmation == nil || err != msg.OrderError_NONE {
		// some error handling
		return nil, err
	}

	// -----------------------------------------------//
	//-------------------- STORES --------------------//
	// if OK send to stores

	// insert aggressive remaining order
	v.OrdersStore.Post(*datastore.NewOrderFromProtoMessage(order))

	if confirmation.PassiveOrdersAffected != nil {
		// insert all passive orders siting on the book
		for _, order := range confirmation.PassiveOrdersAffected {
			v.OrdersStore.Put(*datastore.NewOrderFromProtoMessage(order))
		}
	}

	if confirmation.Trades != nil {
		// insert all trades resulted from the executed order
		for idx, trade := range confirmation.Trades {
			trade.Id = fmt.Sprintf("%s-%d", order.Id, idx)

			t := datastore.NewTradeFromProtoMessage(trade, order.Id, confirmation.PassiveOrdersAffected[idx].Id)
			if err := v.TradesStore.Post(*t); err != nil {
				log.Infof("TradesStore.Post error: %+v\n", err)
			}
		}
	}
	// ------------------------------------------------//
	//------------------- RISK ENGINE -----------------//

	// SOME STUFF

	// ------------------------------------------------//

	return confirmation, msg.OrderError_NONE
}
