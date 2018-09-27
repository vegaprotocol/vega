package core

import (
	"fmt"
	"time"

	"vega/datastore"
	"vega/log"
	"vega/matching"
	"vega/msg"
	"vega/risk"
)

const (
	marketName               = "BTC/DEC18"
	riskCalculationFrequency = 5
)

type Config struct {
	GenesisTime              time.Time
	RiskCalculationFrequency uint64
	AppVersion               string
	AppVersionHash           string
}

type Vega struct {
	Config         *Config
	State          *State
	Statistics     *msg.Statistics
	markets        map[string]*matching.OrderBook
	OrderStore     datastore.OrderStore
	TradeStore     datastore.TradeStore
	PartyStore     datastore.PartyStore
	matchingEngine matching.MatchingEngine
	riskEngine     risk.RiskEngine
}

func New(config *Config, store datastore.StoreProvider) *Vega {

	// Initialise matching engine
	matchingEngine := matching.NewMatchingEngine()

	// Initialise risk engine
	riskEngine := risk.New()

	// todo: version from commit hash, app version incrementing
	statistics := &msg.Statistics{}
	statistics.Status = msg.AppStatus_APP_DISCONNECTED
	statistics.AppVersionHash = config.AppVersionHash
	statistics.AppVersion = config.AppVersion

	return &Vega{
		Config:         config,
		Statistics:     statistics,
		markets:        make(map[string]*matching.OrderBook),
		OrderStore:     store.OrderStore(),
		TradeStore:     store.TradeStore(),
		PartyStore:     store.PartyStore(),
		matchingEngine: matchingEngine,
		riskEngine:     riskEngine,
		State:          NewState(),
	}
}

func GetConfig() *Config {
	return &Config{RiskCalculationFrequency: riskCalculationFrequency}
}

func (v *Vega) SetGenesisTime(genesisTime time.Time) {
	v.Config.GenesisTime = genesisTime
}

func (v *Vega) GetGenesisTime() time.Time {
	return v.Config.GenesisTime
}

func (v *Vega) GetChainHeight() uint64 {
	return uint64(v.State.Height)
}

func (v *Vega) GetRiskFactors(marketName string) (float64, float64, error) {
	return v.riskEngine.GetRiskFactors(marketName)
}

func (v *Vega) InitialiseMarkets() {
	v.matchingEngine.CreateMarket(marketName)
	v.riskEngine.AddNewMarket(&msg.Market{Name: marketName})
	v.Statistics.TotalMarkets = 1
}

func (v *Vega) SubmitOrder(order *msg.Order) (*msg.OrderConfirmation, msg.OrderError) {

	order.Id = fmt.Sprintf("V%d-%d", v.State.Height, v.State.Size)
	order.Timestamp = uint64(v.State.Height)

	log.Infof("SubmitOrder: %+v", order)

	// -----------------------------------------------//
	//----------------- MATCHING ENGINE --------------//

	// 1) submit order to matching engine
	confirmation, errorMsg := v.matchingEngine.SubmitOrder(order)
	if confirmation == nil || errorMsg != msg.OrderError_NONE {
		return nil, errorMsg
	}

	// ------------------------------------------------//
	// 2) --------------- RISK ENGINE -----------------//

	// Call out to risk engine calculation every N blocks
	if order.Timestamp%v.Config.RiskCalculationFrequency == 0 {
		v.riskEngine.RecalculateRisk()
	}

	// -----------------------------------------------//
	//-------------------- STORES --------------------//
	// 3) save to stores

	// insert aggressive remaining order
	err := v.OrderStore.Post(*datastore.NewOrderFromProtoMessage(order))
	if err != nil {
		// Note: writing to store should not prevent flow to other engines
		log.Errorf("OrderStore.Post error: %v", err)
	}
	if confirmation.PassiveOrdersAffected != nil {
		// insert all passive orders siting on the book
		for _, order := range confirmation.PassiveOrdersAffected {
			// Note: writing to store should not prevent flow to other engines
			err := v.OrderStore.Put(*datastore.NewOrderFromProtoMessage(order))
			if err != nil {
				log.Errorf("OrderStore.Put error: %v", err)
			}
		}
	}
	
	v.Statistics.LastOrder = order

	if confirmation.Trades != nil {
		// insert all trades resulted from the executed order
		for idx, trade := range confirmation.Trades {
			trade.Id = fmt.Sprintf("%s-%d", order.Id, idx)

			t := datastore.NewTradeFromProtoMessage(trade, order.Id, confirmation.PassiveOrdersAffected[idx].Id)
			if err := v.TradeStore.Post(*t); err != nil {
				// Note: writing to store should not prevent flow to other engines
				log.Errorf("TradeStore.Post error: %+v", err)
			}

			v.Statistics.LastTrade = trade
		}
	}

	// TODO: ONE METHOD TO create or update risk record for this order party etc

	// ------------------------------------------------//

	return confirmation, msg.OrderError_NONE
}

func (v *Vega) CancelOrder(order *msg.Order) (*msg.OrderCancellation, msg.OrderError) {

	log.Infof("CancelOrder: %+v", order)

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
		// Note: writing to store should not prevent flow to other engines
		log.Errorf("OrderStore.Put error: %v", err)
	}

	// ------------------------------------------------//
	return cancellation, msg.OrderError_NONE
}

func (v *Vega) AmendOrder(amendment *msg.Amendment) (*msg.OrderConfirmation, msg.OrderError) {
	log.Infof("Amendment received: %+v\n", amendment)

	// stores get me order with this reference
	existingOrder, err := v.OrderStore.GetByPartyAndId(amendment.Party, amendment.Id)
	if err != nil {
		fmt.Printf("Error: %+v\n", msg.OrderError_INVALID_ORDER_REFERENCE)
		return &msg.OrderConfirmation{}, msg.OrderError_INVALID_ORDER_REFERENCE
	}

	log.Infof("existingOrder fetched: %+v\n", existingOrder)

	newOrder := msg.OrderPool.Get().(*msg.Order)
	newOrder.Id = existingOrder.Id
	newOrder.Market = existingOrder.Market
	newOrder.Party = existingOrder.Party
	newOrder.Side = existingOrder.Side
	newOrder.Price = existingOrder.Price
	newOrder.Size = existingOrder.Size
	newOrder.Remaining = existingOrder.Remaining
	newOrder.Type = existingOrder.Type
	newOrder.Timestamp = uint64(v.State.Height)
	newOrder.Status = existingOrder.Status
	newOrder.ExpirationDatetime = existingOrder.ExpirationDatetime
	newOrder.ExpirationTimestamp = existingOrder.ExpirationTimestamp
	newOrder.Reference = existingOrder.Reference

	var (
		priceShift, sizeIncrease, sizeDecrease, expiryChange = false, false, false, false
	)

	if amendment.Price != 0 && existingOrder.Price != amendment.Price {
		newOrder.Price = amendment.Price
		priceShift = true
	}

	if amendment.Size != 0 {
		newOrder.Size = amendment.Size
		newOrder.Remaining = amendment.Size
		if amendment.Size > existingOrder.Size {
			sizeIncrease = true
		}
		if amendment.Size < existingOrder.Size {
			sizeDecrease = true
		}
	}

	if newOrder.Type == msg.Order_GTT && amendment.ExpirationTimestamp != 0 && amendment.ExpirationDatetime != "" {
		newOrder.ExpirationTimestamp = amendment.ExpirationTimestamp
		newOrder.ExpirationDatetime = amendment.ExpirationDatetime
		expiryChange = true
	}

	// if increase in size or change in price
	// ---> DO atomic cancel and submit
	if priceShift || sizeIncrease {
		return v.OrderCancelReplace(existingOrder.ToProtoMessage(), newOrder)
	}
	// if decrease in size or change in expiration date
	// ---> DO amend in place in matching engine
	if expiryChange || sizeDecrease {
		return v.OrderAmendInPlace(newOrder)
	}

	log.Infof("Edit not allowed")
	return &msg.OrderConfirmation{}, msg.OrderError_EDIT_NOT_ALLOWED
}

func (v *Vega) OrderCancelReplace(existingOrder, newOrder *msg.Order) (*msg.OrderConfirmation, msg.OrderError) {
	log.Infof("OrderCancelReplace")
	cancellationMessage, err := v.CancelOrder(existingOrder)
	log.Infof("cancellationMessage: %+v\n", cancellationMessage)
	if err != msg.OrderError_NONE {
		fmt.Printf("err : %+v\n", err)
		return &msg.OrderConfirmation{}, err
	}

	return v.SubmitOrder(newOrder)
}

func (v *Vega) OrderAmendInPlace(newOrder *msg.Order) (*msg.OrderConfirmation, msg.OrderError) {

	err := v.matchingEngine.AmendOrder(newOrder)
	if err != msg.OrderError_NONE {
		fmt.Printf("err %+v\n", err)
		return &msg.OrderConfirmation{}, err
	}

	v.OrderStore.Put(*datastore.NewOrderFromProtoMessage(newOrder))

	return &msg.OrderConfirmation{}, msg.OrderError_NONE
}


func (v *Vega) RemoveExpiringOrdersAtTimestamp(timestamp uint64) {
	expiringOrders := v.matchingEngine.RemoveExpiringOrders(timestamp)

	for _, order := range expiringOrders {
		// remove orders from the store
		v.OrderStore.Put(*datastore.NewOrderFromProtoMessage(order))
	}
}

func (v *Vega) NotifySubscribers() {
	v.OrderStore.Notify()
	v.TradeStore.Notify()
}