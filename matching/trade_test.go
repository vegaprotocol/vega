package matching

import (
	"testing"
	"vega/proto"

	"github.com/stretchr/testify/assert"
)

const sseChannelSize = 32

func initOrderBook() *OrderBook {
	orderSseChan := make(chan msg.Order, sseChannelSize)
	tradeSseChan := make(chan msg.Trade, sseChannelSize)

	matchingConfig := DefaultConfig()
	matchingConfig.TradeChans = append(matchingConfig.TradeChans, tradeSseChan)
	matchingConfig.OrderChans = append(matchingConfig.OrderChans, orderSseChan)

	return &OrderBook{name: "testOrderBook", config: matchingConfig}
}

func expectTrade(t *testing.T, expectedTrade, trade *msg.Trade) {
	// run asserts for protocol trade data
	//assert.Equal(t, trade.Market, expectedTrade.Market)
	assert.Equal(t, expectedTrade.Price, trade.Price)
	assert.Equal(t, expectedTrade.Size, trade.Size)
	assert.Equal(t, expectedTrade.Buyer, trade.Buyer)
	assert.Equal(t, expectedTrade.Seller, trade.Seller)
	assert.Equal(t, expectedTrade.Aggressor, trade.Aggressor)
}

func expectOrder(t *testing.T, expectedOrder, order *msg.Order) {
	// run asserts for order
	assert.Equal(t, expectedOrder.Market, order.Market)
	assert.Equal(t, expectedOrder.Party, order.Party)
	assert.Equal(t, expectedOrder.Side, order.Side)
	assert.Equal(t, expectedOrder.Price, order.Price)
	assert.Equal(t, expectedOrder.Size, order.Size)
	assert.Equal(t, expectedOrder.Remaining, order.Remaining)
	assert.Equal(t, expectedOrder.Type, order.Type)
	assert.Equal(t, expectedOrder.Timestamp, order.Timestamp)
}

/*  SUMMARY OF TESTS:
- TestNewTradeNoRemaining
- TestNewTradeWithRemainingOnAggressive
- TestNewTradeWithRemainingOnPassive
*/

/*
	Scenario: two orders of the same size generate trade that is returned and pushed on the channel. Trade messages are expected to match.
	Given following OrderBook
	And given I have passive and aggressive orders
	I create a trade from those two orders
	I expect correct Trade to be returned and pushed onto trade channel
	I expect trade messages to match
	I expect orders to be cleared with 0 remaining
*/

func TestNewTradeNoRemaining(t *testing.T) {
	book := initOrderBook()
	aggressiveOrderMsg := &msg.Order{
		Market:    "testOrderBook",
		Party:     "A",
		Side:      msg.Side_Buy,
		Price:     100,
		Size:      100,
		Remaining: 100,
		Type:      msg.Order_GTC,
		Timestamp: 0,
		Id:        "id-number-one",
	}

	passiveOrderMsg := &msg.Order{
		Market:    "testOrderBook",
		Party:     "B",
		Side:      msg.Side_Sell,
		Price:     100,
		Size:      100,
		Remaining: 100,
		Type:      msg.Order_GTC,
		Timestamp: 0,
		Id:        "id-number-one",
	}

	expectedTrade := &msg.Trade{
		Market:    "testOrderBook",
		Price:     100,
		Size:      100,
		Buyer:     "A",
		Seller:    "B",
		Aggressor: msg.Side_Buy,
	}

	expectedAggressiveClearedOrder := &msg.Order{
		Market:    "testOrderBook",
		Party:     "A",
		Side:      msg.Side_Buy,
		Price:     100,
		Size:      100,
		Remaining: 0,
		Type:      msg.Order_GTC,
		Timestamp: 0,
		Id:        "id-number-one",
	}

	expectedPassiveClearedOrder := &msg.Order{
		Market:    "testOrderBook",
		Party:     "B",
		Side:      msg.Side_Sell,
		Price:     100,
		Size:      100,
		Remaining: 0,
		Type:      msg.Order_GTC,
		Timestamp: 0,
		Id:        "id-number-one",
	}

	aggressiveOE := &OrderEntry{
		order:            aggressiveOrderMsg,
		Side:             msg.Side_Buy,
		persist:          true,
		dispatchChannels: book.config.OrderChans,
	}
	passiveOE := &OrderEntry{
		order:            passiveOrderMsg,
		Side:             msg.Side_Sell,
		persist:          true,
		dispatchChannels: book.config.OrderChans,
	}

	// execute newTrade for those crossing OrderEntries
	trade := newTrade(aggressiveOE, passiveOE, 100)

	// trade is both returned from newTrade function and pushed onto TradeChans - check for both
	expectTrade(t, trade.msg, expectedTrade)

	// We read from channels to obtain two updated orders
	// newTrade method updates passive order first and than aggressive
	// update function pushes new updated order onto the orders channel
	// passive order will appear first in the channel, hence we test with the following order in mind
	for _, orderCh := range book.config.OrderChans {

		clearedOrderMsg := <-orderCh
		expectOrder(t, &clearedOrderMsg, expectedPassiveClearedOrder)

		remainingOrderMsg := <-orderCh
		expectOrder(t, &remainingOrderMsg, expectedAggressiveClearedOrder)
	}
}

/*
	Scenario: two orders of a different size generate trade that is returned and pushed on the channel. Trade messages are expected to match.
	Given following OrderBook
	And given I have passive and aggressive orders
	I create a trade from those two orders
	I expect correct Trade to be returned and pushed onto the trade channel
	I expect trade messages to match
	I expect passive order to be cleared with 0 remaining
	I expect aggressive order to have correct remaining
*/

func TestNewTradeWithRemainingOnAggressive(t *testing.T) {
	book := initOrderBook()
	aggressiveOrderMsg := &msg.Order{
		Market:    "testOrderBook",
		Party:     "A",
		Side:      msg.Side_Buy,
		Price:     100,
		Size:      400,
		Remaining: 400,
		Type:      msg.Order_GTC,
		Timestamp: 0,
		Id:        "id-number-one",
	}

	passiveOrderMsg := &msg.Order{
		Market:    "testOrderBook",
		Party:     "B",
		Side:      msg.Side_Sell,
		Price:     100,
		Size:      100,
		Remaining: 100,
		Type:      msg.Order_GTC,
		Timestamp: 0,
		Id:        "id-number-one",
	}

	expectedTrade := &msg.Trade{
		Market:    "testOrderBook",
		Price:     100,
		Size:      100,
		Buyer:     "A",
		Seller:    "B",
		Aggressor: msg.Side_Buy,
	}

	expectedAggressiveRemainingOrder := &msg.Order{
		Market:    "testOrderBook",
		Party:     "A",
		Side:      msg.Side_Buy,
		Price:     100,
		Size:      400,
		Remaining: 300,
		Type:      msg.Order_GTC,
		Timestamp: 0,
		Id:        "id-number-one",
	}

	expectedPassiveClearedOrder := &msg.Order{
		Market:    "testOrderBook",
		Party:     "B",
		Side:      msg.Side_Sell,
		Price:     100,
		Size:      100,
		Remaining: 0,
		Type:      msg.Order_GTC,
		Timestamp: 0,
		Id:        "id-number-one",
	}

	aggressiveOE := &OrderEntry{
		order: aggressiveOrderMsg,
		Side:             msg.Side_Buy,
		persist:          true,
		dispatchChannels: book.config.OrderChans,
	}
	passiveOE := &OrderEntry{
		order: passiveOrderMsg,
		Side:             msg.Side_Sell,
		persist:          true,
		dispatchChannels: book.config.OrderChans,
	}

	// execute newTrade for those crossing OrderEntries
	trade := newTrade(aggressiveOE, passiveOE, 100)

	// trade is both returned from newTrade function and pushed onto TradeChans - test for both
	expectTrade(t, trade.msg, expectedTrade)

	expectOrder(t, aggressiveOE.order, expectedAggressiveRemainingOrder)
	expectOrder(t, passiveOE.order, expectedPassiveClearedOrder)

	for _, orderCh := range book.config.OrderChans {

		clearedOrderMsg := <-orderCh
		expectOrder(t, &clearedOrderMsg, expectedPassiveClearedOrder)

		remainingOrderMsg := <-orderCh
		expectOrder(t, &remainingOrderMsg, expectedAggressiveRemainingOrder)
	}
}

/*
	Scenario: two orders of a different size generate trade that is returned and pushed on the channel. Trade messages are expected to match.
	Given following OrderBook
	And given I have passive and aggressive orders
	I create a trade from those two orders
	I expect correct Trade to be returned and pushed onto the trade channel
	I expect trade messages to match
	I expect aggressive order to be cleared with 0 remaining
	I expect passive order to have correct remaining
*/

func TestNewTradeWithRemainingOnPassive(t *testing.T) {
	book := initOrderBook()
	aggressiveOrderMsg := &msg.Order{
		Market:    "testOrderBook",
		Party:     "A",
		Side:      msg.Side_Buy,
		Price:     100,
		Size:      100,
		Remaining: 100,
		Type:      msg.Order_GTC,
		Timestamp: 0,
		Id:        "id-number-one",
	}

	passiveOrderMsg := &msg.Order{
		Market:    "testOrderBook",
		Party:     "B",
		Side:      msg.Side_Sell,
		Price:     100,
		Size:      400,
		Remaining: 400,
		Type:      msg.Order_GTC,
		Timestamp: 0,
		Id:        "id-number-one",
	}

	expectedTrade := &msg.Trade{
		Market:    "testOrderBook",
		Price:     100,
		Size:      100,
		Buyer:     "A",
		Seller:    "B",
		Aggressor: msg.Side_Buy,
	}

	expectedAggressiveClearedOrder := &msg.Order{
		Market:    "testOrderBook",
		Party:     "A",
		Side:      msg.Side_Buy,
		Price:     100,
		Size:      100,
		Remaining: 0,
		Type:      msg.Order_GTC,
		Timestamp: 0,
		Id:        "id-number-one",
	}

	expectedPassiveRemainingOrder := &msg.Order{
		Market:    "testOrderBook",
		Party:     "B",
		Side:      msg.Side_Sell,
		Price:     100,
		Size:      400,
		Remaining: 300,
		Type:      msg.Order_GTC,
		Timestamp: 0,
		Id:        "id-number-one",
	}

	aggressiveOE := &OrderEntry{
		order: aggressiveOrderMsg,
		Side:             msg.Side_Buy,
		persist:          true,
		dispatchChannels: book.config.OrderChans,
	}

	passiveOE := &OrderEntry{
		order: passiveOrderMsg,
		Side:             msg.Side_Sell,
		persist:          true,
		dispatchChannels: book.config.OrderChans,
	}

	// execute newTrade for those crossing OrderEntries
	trade := newTrade(aggressiveOE, passiveOE, 100)

	// trade is both returned from newTrade functiona and pushed onto TradeChans - check for both
	expectTrade(t, trade.msg, expectedTrade)

	expectOrder(t, passiveOE.order, expectedPassiveRemainingOrder)
	expectOrder(t, aggressiveOE.order, expectedAggressiveClearedOrder)

	for _, orderCh := range book.config.OrderChans {

		remainingOrderMsg := <-orderCh
		expectOrder(t, &remainingOrderMsg, expectedPassiveRemainingOrder)

		clearedOrderMsg := <-orderCh
		expectOrder(t, &clearedOrderMsg, expectedAggressiveClearedOrder)
	}
}

// Remarks for trades
// 1. send only channels used -- removed
// 2. market is not saved in trade object

// Remarks for orders
// 1. msg is not calculated before OrderEntry push
// 2. msg and order are misnamed - should be name protocolMsg for both trade and order
// 3. should newTrade update orders??

//TODO: TEST multiple order types
