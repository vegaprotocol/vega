package core

import (
	"fmt"
	"time"

	"vega/datastore"
	"vega/matching"
	"vega/msg"
)

const marketName = "BTC/DEC18"

const genesisTimeStr = "2018-07-05T13:36:01Z"

type Config struct{}

type Vega struct {
	config         *Config
	markets        map[string]*matching.OrderBook
	OrderStore    datastore.OrderStore
	TradeStore    datastore.TradeStore
	matchingEngine matching.MatchingEngine
	State          *State
}

func New(config *Config, store *datastore.MemoryStoreProvider) *Vega {

	// Initialise matching engine
	matchingEngine := matching.NewMatchingEngine()

	return &Vega{
		config:         config,
		markets:        make(map[string]*matching.OrderBook),
		OrderStore:    store.OrderStore(),
		TradeStore:    store.TradeStore(),
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

	// -----------------------------------------------//
	//----------------- MATCHING ENGINE --------------//
	// 1) submit order to matching engine
	confirmation, errorMsg := v.matchingEngine.SubmitOrder(order)
	if confirmation == nil || errorMsg != msg.OrderError_NONE {
		return nil, errorMsg
	}

	// -----------------------------------------------//
	//-------------------- STORES --------------------//
	// 2) if OK send to stores

	// insert aggressive remaining order
	err := v.OrderStore.Post(*datastore.NewOrderFromProtoMessage(order))
	if err != nil {
		// Note: writing to store should not prevent flow to other engines
		fmt.Printf("OrderStore.Post error: %+v\n", err)
	}

	if confirmation.PassiveOrdersAffected != nil {
		// insert all passive orders siting on the book
		for _, order := range confirmation.PassiveOrdersAffected {
			// Note: writing to store should not prevent flow to other engines
			err := v.OrderStore.Put(*datastore.NewOrderFromProtoMessage(order))
			if err != nil {
				fmt.Printf("OrderStore.Put error: %+v\n", err)
			}
		}
	}

	if confirmation.Trades != nil {
		// insert all trades resulted from the executed order
		for idx, trade := range confirmation.Trades {
			trade.Id = fmt.Sprintf("%s-%d", order.Id, idx)
			t := datastore.NewTradeFromProtoMessage(trade, order.Id, confirmation.PassiveOrdersAffected[idx].Id)
			if err := v.TradeStore.Post(*t); err != nil {
				// Note: writing to store should not prevent flow to other engines
				fmt.Printf("TradeStore.Post error: %+v\n", err)
			}
		}
	}
	
	// ------------------------------------------------//
	//------------------- RISK ENGINE -----------------//

	// 3) PLACEHOLDER

	// ------------------------------------------------//

	return confirmation, msg.OrderError_NONE
}

func (v *Vega) CancelOrder(order *msg.Order) (*msg.OrderCancellation, msg.OrderError) {

	fmt.Printf("%+v", order)
	fmt.Println("")

	// -----------------------------------------------//
	//----------------- MATCHING ENGINE --------------//
	// 1) cancel order in matching engine
	cancellation, errorMsg := v.matchingEngine.CancelOrder(order)
	if cancellation == nil || errorMsg != msg.OrderError_NONE {
		return nil, errorMsg
	}

	// -----------------------------------------------//
	//-------------------- STORES --------------------//
	// 2) if OK update stores

	// insert aggressive remaining order
	err := v.OrderStore.Put(*datastore.NewOrderFromProtoMessage(order))
	if err != nil {
		// Note: writing to store should not prevent flow to other
		fmt.Printf("OrderStore.Put error: %+v\n", err)
	}
	
	return cancellation, msg.OrderError_NONE
}