package matching
//
//import (
//	"testing"
//
//	"vega/proto"
//
//	"github.com/stretchr/testify/assert"
//)
//
//func TestUncross_AggressiveOrderCrossingTwoPriceLevels(t *testing.T) {
//	/*
//	ASSUMPTIONS:
//	- no pro-rata at any level
//	- two different levels
//	- aggressive buy order meets single best order from sell side and next single order on next level
//	- no remaining on neither of the sides
//	*/
//
//	const testPrice = 100
//	book := initOrderBook()
//	orderBookSideSell := newSide(msg.Side_Sell)
//
//	priceLevel := NewPriceLevel(orderBookSideSell, testPrice)
//	orderBookSideSell.levels.ReplaceOrInsert(priceLevel)
//
//	orderEntry := &OrderEntry{
//		order: &msg.Order{
//			Market:    "testOrderBook",
//			Party:     "A",
//			Side:      msg.Side_Sell,
//			Price:     priceLevel.price,
//			Size:      100,
//			Remaining: 100,
//			Type:      msg.Order_GTC,
//			Timestamp: 0,
//			Id:        "id-number-one",
//		},
//		Side: msg.Side_Sell,
//		persist: true,
//		dispatchChannels:	book.config.OrderChans,
//	}
//	priceLevel.addOrder(orderEntry)
//
//	priceLevel = NewPriceLevel(orderBookSideSell, testPrice+1)
//	orderBookSideSell.levels.ReplaceOrInsert(priceLevel)
//
//	orderEntry = &OrderEntry{
//		order: &msg.Order{
//			Market:    "testOrderBook",
//			Party:     "B",
//			Side:      msg.Side_Sell,
//			Price:     priceLevel.price,
//			Size:      100,
//			Remaining: 100,
//			Type:      msg.Order_GTC,
//			Timestamp: 0,
//			Id:        "id-number-one",
//		},
//		Side: msg.Side_Sell,
//		persist: true,
//		dispatchChannels:	book.config.OrderChans,
//	}
//	priceLevel.addOrder(orderEntry)
//
//	aggressiveOrder := &OrderEntry{
//		order: &msg.Order{
//			Market:    "testOrderBook",
//			Party:     "C",
//			Side:      msg.Side_Buy,
//			Price:     testPrice+1,
//			Size:      200,
//			Remaining: 200,
//			Type:      msg.Order_GTC,
//			Timestamp: 0,
//			Id:        "id-number-one",
//		},
//		Side: msg.Side_Buy,
//		persist: true,
//		dispatchChannels:	book.config.OrderChans,
//	}
//
//	expectedTrades := []msg.Trade{
//		{
//			Market:    "testOrderBook",
//			Price:     testPrice,
//			Size:      100,
//			Buyer:     "C",
//			Seller:    "A",
//			Aggressor: msg.Side_Buy,
//		},
//		{
//			Market:    "testOrderBook",
//			Price:     testPrice+1,
//			Size:      100,
//			Buyer:     "C",
//			Seller:    "B",
//			Aggressor: msg.Side_Buy,
//		},
//	}
//
//	// cross with opposite
//
//	trades, lastTradedPrice := orderBookSideSell.cross(aggressiveOrder)
//	book.lastTradedPrice = lastTradedPrice
//
//	for _, t := range *trades {
//		for _, c := range book.config.TradeChans {
//			c <- *t.toMessage()
//		}
//	}
//
//	// if persist add to tradebook to the right side
//	orderBookSidebuy := newSide(msg.Side_Buy)
//	if aggressiveOrder.persist && aggressiveOrder.order.Remaining > 0 {
//		orderBookSidebuy.addOrder(aggressiveOrder)
//	}
//
//
//	assert.Equal(t, 2, len(*trades))
//
//	// filled orders should be cleared from the price level
//	assert.Equal(t, 0, len(orderBookSideSell.getPriceLevel(testPrice).orders))
//	assert.Equal(t, 0, len(orderBookSideSell.getPriceLevel(testPrice+1).orders))
//	assert.Equal(t, uint64(0), aggressiveOrder.order.Remaining)
//
//	for i, trade := range *trades {
//		expectTrade(t, trade.msg, &expectedTrades[i])
//	}
//}
//
////func TestAddOrder_AggressiveOrderCrossesTwoPriceLevels(t *testing.T) {
////	/*
////	ASSUMPTIONS:
////	- no pro-rata at any level
////	- two different levels
////	- aggressive buy order meets single best order from sell side and next single order on next level
////	- no remaining on neither of the sides
////	*/
////
////	const testPrice = 100
////	book := initOrderBook()
////	orderBookSideSell := makeSide(msg.Side_Sell, book)
////
////	priceLevel := NewPriceLevel(orderBookSideSell, testPrice)
////	orderBookSideSell.levels.ReplaceOrInsert(priceLevel)
////
////	orderEntry := &OrderEntry{
////		order: &msg.Order{
////			Market:    "testOrderBook",
////			Party:     "A",
////			Side:      msg.Side_Sell,
////			Price:     priceLevel.price,
////			Size:      100,
////			Remaining: 100,
////			Type:      msg.Order_GTC,
////			Timestamp: 0,
////			Id:        "id-number-one",
////		},
////		book: book,
////		side: orderBookSideSell,
////	}
////	priceLevel.addOrder(orderEntry)
////
////	priceLevel = NewPriceLevel(orderBookSideSell, testPrice+1)
////	orderBookSideSell.levels.ReplaceOrInsert(priceLevel)
////
////	orderEntry = &OrderEntry{
////		order: &msg.Order{
////			Market:    "testOrderBook",
////			Party:     "B",
////			Side:      msg.Side_Sell,
////			Price:     priceLevel.price,
////			Size:      100,
////			Remaining: 100,
////			Type:      msg.Order_GTC,
////			Timestamp: 0,
////			Id:        "id-number-one",
////		},
////		book: book,
////		side: orderBookSideSell,
////	}
////	priceLevel.addOrder(orderEntry)
////
////	aggressiveOrder := &OrderEntry{
////		order: &msg.Order{
////			Market:    "testOrderBook",
////			Party:     "C",
////			Side:      msg.Side_Buy,
////			Price:     testPrice+1,
////			Size:      200,
////			Remaining: 200,
////			Type:      msg.Order_GTC,
////			Timestamp: 0,
////			Id:        "id-number-one",
////		},
////		book: book,
////		side: orderBookSideSell,
////	}
////
////	expectedTrades := []msg.Trade{
////		{
////			Market:    "testOrderBook",
////			Price:     testPrice,
////			Size:      100,
////			Buyer:     "C",
////			Seller:    "A",
////			Aggressor: msg.Side_Buy,
////		},
////		{
////			Market:    "testOrderBook",
////			Price:     testPrice+1,
////			Size:      100,
////			Buyer:     "C",
////			Seller:    "B",
////			Aggressor: msg.Side_Buy,
////		},
////	}
////
////	// call addOrder from the side buy of the book
////	orderBookSideBuy := makeSide(msg.Side_Buy, book)
////	orderBookSideBuy.other = orderBookSideSell
////
////	trades := orderBookSideBuy.addOrder(aggressiveOrder)
////
////	assert.Equal(t, 2, len(*trades))
////
////	// filled orders should be cleared from the price level
////	assert.Equal(t, 0, orderBookSideSell.getPriceLevel(testPrice).orders.Len())
////	assert.Equal(t, 0, orderBookSideSell.getPriceLevel(testPrice+1).orders.Len())
////	assert.Equal(t, uint64(0), aggressiveOrder.order.Remaining)
////
////	for i, trade := range *trades {
////		expectTrade(t, trade.msg, &expectedTrades[i])
////	}
////}
//
//func TestAddOrder_AggressiveOrderCrossesTwoPriceLevelsRunAnotherAggressiveOrderOnEmptyBook(t *testing.T) {
//	/*
//	ASSUMPTIONS:
//	- no pro-rata at any level
//	- two different levels
//	- aggressive buy order meets single best order from sell side and next single order on next level
//	- no remaining on neither of the sides
//	*/
//
//	const testPrice = 100
//	book := initOrderBook()
//	orderBookSideSell := newSide(msg.Side_Sell)
//
//	priceLevel := NewPriceLevel(orderBookSideSell, testPrice)
//	orderBookSideSell.levels.ReplaceOrInsert(priceLevel)
//
//	orderEntry := &OrderEntry{
//		order: &msg.Order{
//			Market:    "testOrderBook",
//			Party:     "A",
//			Side:      msg.Side_Sell,
//			Price:     priceLevel.price,
//			Size:      100,
//			Remaining: 100,
//			Type:      msg.Order_GTC,
//			Timestamp: 0,
//			Id:        "id-number-one",
//		},
//		Side: msg.Side_Sell,
//		persist: true,
//		dispatchChannels:	book.config.OrderChans,
//	}
//	priceLevel.addOrder(orderEntry)
//
//	priceLevel = NewPriceLevel(orderBookSideSell, testPrice+1)
//	orderBookSideSell.levels.ReplaceOrInsert(priceLevel)
//
//	orderEntry = &OrderEntry{
//		order: &msg.Order{
//			Market:    "testOrderBook",
//			Party:     "B",
//			Side:      msg.Side_Sell,
//			Price:     priceLevel.price,
//			Size:      100,
//			Remaining: 100,
//			Type:      msg.Order_GTC,
//			Timestamp: 0,
//			Id:        "id-number-one",
//		},
//		Side: msg.Side_Sell,
//		persist: true,
//		dispatchChannels:	book.config.OrderChans,
//	}
//	priceLevel.addOrder(orderEntry)
//
//	aggressiveOrder := &OrderEntry{
//		order: &msg.Order{
//			Market:    "testOrderBook",
//			Party:     "C",
//			Side:      msg.Side_Buy,
//			Price:     testPrice+1,
//			Size:      200,
//			Remaining: 200,
//			Type:      msg.Order_GTC,
//			Timestamp: 0,
//			Id:        "id-number-one",
//		},
//		Side: msg.Side_Buy,
//		persist: true,
//		dispatchChannels:	book.config.OrderChans,
//	}
//
//	expectedTrades := []msg.Trade{
//		{
//			Market:    "testOrderBook",
//			Price:     testPrice,
//			Size:      100,
//			Buyer:     "C",
//			Seller:    "A",
//			Aggressor: msg.Side_Buy,
//		},
//		{
//			Market:    "testOrderBook",
//			Price:     testPrice+1,
//			Size:      100,
//			Buyer:     "C",
//			Seller:    "B",
//			Aggressor: msg.Side_Buy,
//		},
//	}
//
//	// call addOrder from the side buy of the book
//	trades, lastTradedPrice := orderBookSideSell.cross(aggressiveOrder)
//	assert.Equal(t, uint64(testPrice+1), lastTradedPrice)
//	assert.Equal(t, 2, len(*trades))
//
//	// filled orders should be unallocated, therefore price levels should not even exist
//	orderBookSideBuy := newSide(msg.Side_Buy)
//	assert.Equal(t, 0, len(orderBookSideBuy.getPriceLevel(testPrice).orders))
//	assert.Equal(t, 0, len(orderBookSideBuy.getPriceLevel(testPrice+1).orders))
//	assert.Equal(t, uint64(0), aggressiveOrder.order.Remaining)
//
//	for i, trade := range *trades {
//		expectTrade(t, trade.msg, &expectedTrades[i])
//	}
//
//	aggressiveOrder = &OrderEntry{
//		order: &msg.Order{
//			Market:    "testOrderBook",
//			Party:     "C",
//			Side:      msg.Side_Buy,
//			Price:     testPrice+1,
//			Size:      200,
//			Remaining: 200,
//			Type:      msg.Order_GTC,
//			Timestamp: 0,
//			Id:        "id-number-one",
//		},
//		Side: msg.Side_Buy,
//		persist: true,
//		dispatchChannels:	book.config.OrderChans,
//	}
//
//	trades, lastTradedPrice2 := orderBookSideSell.cross(aggressiveOrder)
//	assert.Equal(t, uint64(0), lastTradedPrice2)
//	assert.Equal(t, 0, len(*trades))
//	assert.Equal(t, uint64(200), aggressiveOrder.order.Remaining)
//}
//
//func TestAddOrder_AggressiveOrderCrossesTwoPriceLevelsWithRemainingOnAggressive(t *testing.T) {
//	/*
//	ASSUMPTIONS:
//	- no pro-rata at any level
//	- two different levels
//	- aggressive buy order meets single best order from sell side and next single order on next level
//	- no remaining on neither of the sides
//	*/
//
//	const testPrice = 100
//	book := initOrderBook()
//	orderBookSideSell := newSide(msg.Side_Sell)
//
//	priceLevel := NewPriceLevel(orderBookSideSell, testPrice)
//	orderBookSideSell.levels.ReplaceOrInsert(priceLevel)
//
//	orderEntry := newOrderEntry(
//		&msg.Order{
//			Market:    "testOrderBook",
//			Party:     "A",
//			Side:      msg.Side_Sell,
//			Price:     priceLevel.price,
//			Size:      100,
//			Remaining: 100,
//			Type:      msg.Order_GTC,
//			Timestamp: 0,
//			Id:        "id-number-one",
//		},
//		book.config.OrderChans,
//	)
//
//	priceLevel.addOrder(orderEntry)
//
//	priceLevel = NewPriceLevel(orderBookSideSell, testPrice+1)
//	orderBookSideSell.levels.ReplaceOrInsert(priceLevel)
//
//	orderEntry = newOrderEntry(
//		&msg.Order{
//			Market:    "testOrderBook",
//			Party:     "B",
//			Side:      msg.Side_Sell,
//			Price:     priceLevel.price,
//			Size:      100,
//			Remaining: 100,
//			Type:      msg.Order_GTC,
//			Timestamp: 0,
//			Id:        "id-number-one",
//		},
//		book.config.OrderChans,
//	)
//
//	priceLevel.addOrder(orderEntry)
//
//	aggressiveOrder := newOrderEntry(
//		&msg.Order{
//			Market:    "testOrderBook",
//			Party:     "C",
//			Side:      msg.Side_Buy,
//			Price:     testPrice+1,
//			Size:      250,
//			Remaining: 250,
//			Type:      msg.Order_GTC,
//			Timestamp: 0,
//			Id:        "id-number-one",
//		},
//		book.config.OrderChans,
//	)
//
//	expectedTrades := []msg.Trade{
//		{
//			Market:    "testOrderBook",
//			Price:     testPrice,
//			Size:      100,
//			Buyer:     "C",
//			Seller:    "A",
//			Aggressor: msg.Side_Buy,
//		},
//		{
//			Market:    "testOrderBook",
//			Price:     testPrice+1,
//			Size:      100,
//			Buyer:     "C",
//			Seller:    "B",
//			Aggressor: msg.Side_Buy,
//		},
//	}
//
//	// call addOrder from the side buy of the book
//
//	assert.Equal(t, uint64(250), aggressiveOrder.order.Remaining)
//	assert.Equal(t, true, aggressiveOrder.persist)
//
//	trades, lastTradedPrice := orderBookSideSell.cross(aggressiveOrder)
//	book.lastTradedPrice = lastTradedPrice
//
//	assert.Equal(t, uint64(50), aggressiveOrder.order.Remaining)
//	assert.Equal(t, 2, len(*trades))
//
//	// filled orders should be cleared from the price level
//	orderBookSideBuy := newSide(msg.Side_Buy)
//	assert.Equal(t, 0, len(orderBookSideBuy.getPriceLevel(testPrice).orders))
//	assert.Equal(t, 0, len(orderBookSideBuy.getPriceLevel(testPrice+1).orders))
//
//	for i, trade := range *trades {
//		expectTrade(t, trade.msg, &expectedTrades[i])
//	}
//
//	// expect order to be placed in the orders lookup
//	//_, exists := orderBookSideBuy.book.orders[aggressiveOrder.order.Id]
//	//assert.Equal(t, true, exists)
//
//	// expect order in the lookup to be exact
//	//expectOrder(t, aggressiveOrder.order, orderBookSideBuy.book.orders[aggressiveOrder.order.Id].order)
//}
//
//func TestAddOrder_AggressiveNonPersistentOrderCrossesTwoPriceLevelsWithRemainingOnAggressive(t *testing.T) {
//	/*
//	ASSUMPTIONS:
//	- no pro-rata at any level
//	- two different levels
//	- aggressive buy order meets single best order from sell side and next single order on next level
//	- no remaining on neither of the sides
//	*/
//
//	const testPrice = 100
//	book := initOrderBook()
//	orderBookSideSell := newSide(msg.Side_Sell)
//
//	priceLevel := NewPriceLevel(orderBookSideSell, testPrice)
//	orderBookSideSell.levels.ReplaceOrInsert(priceLevel)
//
//	orderEntry := &OrderEntry{
//		order: &msg.Order{
//			Market:    "testOrderBook",
//			Party:     "A",
//			Side:      msg.Side_Sell,
//			Price:     priceLevel.price,
//			Size:      100,
//			Remaining: 100,
//			Type:      msg.Order_GTC,
//			Timestamp: 0,
//			Id:        "id-number-one",
//		},
//		Side: msg.Side_Sell,
//		persist: true,
//		dispatchChannels:	book.config.OrderChans,
//	}
//	priceLevel.addOrder(orderEntry)
//
//	priceLevel = NewPriceLevel(orderBookSideSell, testPrice+1)
//	orderBookSideSell.levels.ReplaceOrInsert(priceLevel)
//
//	orderEntry = newOrderEntry(
//		&msg.Order{
//			Market:    "testOrderBook",
//			Party:     "B",
//			Side:      msg.Side_Sell,
//			Price:     priceLevel.price,
//			Size:      100,
//			Remaining: 100,
//			Type:      msg.Order_GTC,
//			Timestamp: 0,
//			Id:        "id-number-one",
//		},
//		book.config.OrderChans,
//	)
//
//	priceLevel.addOrder(orderEntry)
//
//	aggressiveOrder := newOrderEntry(
//		&msg.Order{
//			Market:    "testOrderBook",
//			Party:     "C",
//			Side:      msg.Side_Buy,
//			Price:     testPrice+1,
//			Size:      250,
//			Remaining: 250,
//			Type:      msg.Order_FOK,
//			Timestamp: 0,
//			Id:        "id-number-one",
//		},
//		book.config.OrderChans,
//	)
//
//	expectedTrades := []msg.Trade{
//		{
//			Market:    "testOrderBook",
//			Price:     testPrice,
//			Size:      100,
//			Buyer:     "C",
//			Seller:    "A",
//			Aggressor: msg.Side_Buy,
//		},
//		{
//			Market:    "testOrderBook",
//			Price:     testPrice+1,
//			Size:      100,
//			Buyer:     "C",
//			Seller:    "B",
//			Aggressor: msg.Side_Buy,
//		},
//	}
//
//	// call addOrder from the side buy of the book
//
//	trades, lastTradedPrice := orderBookSideSell.cross(aggressiveOrder)
//	book.lastTradedPrice = lastTradedPrice
//
//	assert.Equal(t, 2, len(*trades))
//
//	// filled orders should be cleared from the price level
//	orderBookSideBuy := newSide(msg.Side_Buy)
//	assert.Equal(t, 0, len(orderBookSideBuy.getPriceLevel(testPrice).orders))
//	assert.Equal(t, 0, len(orderBookSideBuy.getPriceLevel(testPrice+1).orders))
//
//	// remaining order should match
//	assert.Equal(t, uint64(50), aggressiveOrder.order.Remaining)
//
//	for i, trade := range *trades {
//		expectTrade(t, trade.msg, &expectedTrades[i])
//	}
//}
//
//
///* Remarks:
//1. persist is set up on an orderEntryFromMessage level...
//2. crossedWith in side.go function is not called
//*/