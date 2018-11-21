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
	LogPriceLevels           bool

}

type Vega struct {
	Config         *Config
	State          *State
	Statistics     *msg.Statistics
	markets        map[string]*matching.OrderBook
	matchingEngine matching.MatchingEngine
	riskEngine     risk.RiskEngine
	OrderStore     datastore.OrderStore
	TradeStore     datastore.TradeStore
	CandleStore	   datastore.CandleStore
	tradesBuffer   map[string][]*msg.Trade

}

func New(config *Config,  orderStore datastore.OrderStore, tradeStore datastore.TradeStore, candleStore datastore.CandleStore) *Vega {

	// Initialise matching engine
	matchingEngine := matching.NewMatchingEngine(config.LogPriceLevels)

	// Initialise risk engine
	riskEngine := risk.New()

	// tradesBuffer for candles
	tradesBuffer := make(map[string][]*msg.Trade, 0)
	
	// todo: version from commit hash, app version incrementing
	statistics := &msg.Statistics{}
	statistics.Status = msg.AppStatus_APP_DISCONNECTED
	statistics.AppVersionHash = config.AppVersionHash
	statistics.AppVersion = config.AppVersion

	return &Vega{
		Config:         config,
		Statistics:     statistics,
		markets:        make(map[string]*matching.OrderBook),
		matchingEngine: matchingEngine,
		riskEngine:     riskEngine,
		State:          NewState(),
		OrderStore:     orderStore,
		TradeStore:     tradeStore,
		CandleStore:    candleStore,
		tradesBuffer:   tradesBuffer,
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

func (v *Vega) GetCurrentTimestamp() uint64 {
	return uint64(v.State.Timestamp)
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
	order.Timestamp = uint64(v.State.Timestamp)

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

	//Call out to risk engine calculation every N blocks
	if order.Timestamp%v.Config.RiskCalculationFrequency == 0 {
		v.riskEngine.RecalculateRisk()
	}

	// -----------------------------------------------//
	//-------------------- STORES --------------------//
	// 3) save to stores

	// insert aggressive remaining order
	err := v.OrderStore.Post(order)
	if err != nil {
		// Note: writing to store should not prevent flow to other engines
		log.Errorf("OrderStore.Post error: %v", err)
	}
	if confirmation.PassiveOrdersAffected != nil {
		// insert all passive orders siting on the book
		for _, order := range confirmation.PassiveOrdersAffected {
			// Note: writing to store should not prevent flow to other engines
			err := v.OrderStore.Put(order)
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
			if order.Side == msg.Side_Buy {
				trade.BuyOrder = order.Id
				trade.SellOrder = confirmation.PassiveOrdersAffected[idx].Id
			} else {
				trade.SellOrder = order.Id
				trade.BuyOrder = confirmation.PassiveOrdersAffected[idx].Id
			}

			if err := v.TradeStore.Post(trade); err != nil {
				// Note: writing to store should not prevent flow to other engines
				log.Errorf("TradeStore.Post error: %+v", err)
			}

			fmt.Printf("Addding trade\n")
			// Save to trade buffer for generating candles etc
			v.tradesBuffer[trade.Market] = append(v.tradesBuffer[trade.Market], trade)

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
	err := v.OrderStore.Put(order)
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

	v.OrderStore.Put(newOrder)

	return &msg.OrderConfirmation{}, msg.OrderError_NONE
}


func (v *Vega) RemoveExpiringOrdersAtTimestamp(timestamp uint64) {
	expiringOrders := v.matchingEngine.RemoveExpiringOrders(timestamp)
	log.Debugf("Removed %v expired orders from matching engine, now update stores", len(expiringOrders))

	for _, order := range expiringOrders {
		v.OrderStore.Put(order)
	}

	log.Debugf("Updated %v expired orders in stores", len(expiringOrders))
}

func (v *Vega) NotifySubscribers() {
	v.OrderStore.Notify()
	v.TradeStore.Notify()
}

// this should act as a separate slow go routine triggered after block is committed
func (v *Vega) GenerateCandles() error {
	fmt.Printf("GenerateCandles called\n")

	// todo: generate/range over all markets!
	market := marketName

	if _, ok := v.tradesBuffer[market]; !ok {
		v.tradesBuffer[market] = nil
		fmt.Printf("Market not found\n")
	}

	fmt.Printf("tradesBuffer %+v\n", v.tradesBuffer)

	// in case there is no trading activity on this market, generate empty candles based on historical values
	if len(v.tradesBuffer[market]) == 0 {
		fmt.Printf("Empty candles... \n")
		if err := v.CandleStore.GenerateEmptyCandles(market, v.GetCurrentTimestamp()); err != nil {
			return err
		}
		return nil
	}

	// generate/update  candles for each trade in the buffer
	fmt.Printf("State of the buffer ... %+v\n", v.tradesBuffer)
	for idx := range v.tradesBuffer[market] {
		if err := v.CandleStore.GenerateCandles(v.tradesBuffer[market][idx]); err != nil {
			return err
		}
	}

	// Notify all subscribers
	v.CandleStore.Notify()

	// Flush the buffer
	v.tradesBuffer[market] = nil

	return nil
}
