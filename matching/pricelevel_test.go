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
///*  SUMMARY OF TESTS:
//	- TestPriceLevelAddsOrdersWithDifferentTimestampCorrectly
//	- TestPriceLevelAddsOrdersWithSameTimestampCorrectly
//	- TestPriceLevelCrossNoProRataNoRemaining
//	- TestPriceLevelCrossNoProRataWithRemainingOnAggressiveOrder
//	- TestPriceLevelCrossNoProRataWithRemainingOnPassiveOrder
//	- TestPriceLevelCrossWithProRataWithNoRemainingOnAggressive
//*/
//
////-------------------------------- ORDERS ALLOCATION ON A SINGLE PRICE LEVEL LIST ------------------------------------//
//
///*
//	Given test price level
//	I add 2 orders waiting at the test price level with different timestamps
//	I expect 2 price level on the price list
//	I remove orders from the price list
//	I expect 0 price level on the price list
//*/
//
//
//func TestPriceLevelAddAndRemoveOrders(t *testing.T) {
//
//	const testPrice = 100
//	book := initOrderBook()
//	orderBookSide := newSide(msg.Side_Sell)
//	priceLevel := NewPriceLevel(orderBookSide, testPrice)
//	orderBookSide.levels.ReplaceOrInsert(priceLevel)
//
//	ordersSitingAtPriceLevel := []OrderEntry{
//		{
//			order:	&msg.Order{
//				Market:    "testOrderBook",
//				Party:     "A",
//				Side:      msg.Side_Sell,
//				Price:     testPrice,
//				Size:      100,
//				Remaining: 100,
//				Type:      msg.Order_GTC,
//				Timestamp: 0,
//				Id:        "id-number-one",
//			},
//			Side: msg.Side_Sell,
//			persist: true,
//			dispatchChannels:	book.config.OrderChans,
//		},
//		{
//			order: &msg.Order{
//				Market:    "testOrderBook",
//				Party:     "B",
//				Side:      msg.Side_Sell,
//				Price:     testPrice,
//				Size:      100,
//				Remaining: 100,
//				Type:      msg.Order_GTC,
//				Timestamp: 0,
//				Id:        "id-number-one",
//			},
//			Side: msg.Side_Sell,
//			persist: true,
//			dispatchChannels:	book.config.OrderChans,
//		},
//		{
//			order: &msg.Order{
//				Market:    "testOrderBook",
//				Party:     "C",
//				Side:      msg.Side_Sell,
//				Price:     testPrice,
//				Size:      200,
//				Remaining: 200,
//				Type:      msg.Order_GTC,
//				Timestamp: 0,
//				Id:        "id-number-one",
//			},
//			Side: msg.Side_Sell,
//			persist: true,
//			dispatchChannels:	book.config.OrderChans,
//		},
//	}
//
//	testPriceLevel := orderBookSide.getPriceLevel(testPrice)
//	testPriceLevel.addOrder(&ordersSitingAtPriceLevel[0])
//	testPriceLevel.addOrder(&ordersSitingAtPriceLevel[1])
//
//	assert.Equal(t, 2, len(testPriceLevel.orders))
//
//	testPriceLevel.removeOrder(&ordersSitingAtPriceLevel[0])
//	testPriceLevel.removeOrder(&ordersSitingAtPriceLevel[1])
//
//	assert.Equal(t, 0, len(testPriceLevel.orders))
//}
//
//
///*
//	Given test price level
//	I have 2 orders waiting at the test price level with different timestamps
//	I fetch orders from order side book at the test price level
//	I expect exactly  2 orders at the test price level with total volume to be correct and protocol messages of those orders to be exact
//	I expect volume by timestamp to contain 2 separate volumes
//	When I remove orders
//	I expect 0 price level on the price list
//*/
//
//func TestPriceLevelAddsOrdersWithDifferentTimestampCorrectly(t *testing.T) {
//	/*
//	ASSUMPTIONS:
//	- no pro-rata
//	- two different timestamps
//	- aggressive buy order meets single best order from sell side
//	- no remaining on neither of the sides
//	*/
//
//	const testPrice = 100
//	orderBookSide := newSide(msg.Side_Sell)
//	priceLevel := NewPriceLevel(orderBookSide, testPrice)
//	orderBookSide.levels.ReplaceOrInsert(priceLevel)
//
//	ordersSitingAtPriceLevel := []OrderEntry{
//		{
//			order: &msg.Order{
//				Market:    "testOrderBook",
//				Party:     "B",
//				Side:      msg.Side_Sell,
//				Price:     testPrice,
//				Size:      100,
//				Remaining: 100,
//				Type:      msg.Order_GTC,
//				Timestamp: 0,
//				Id:        "id-number-one",
//			},
//		},
//		{
//			order: &msg.Order{
//				Market:    "testOrderBook",
//				Party:     "B",
//				Side:      msg.Side_Sell,
//				Price:     testPrice,
//				Size:      200,
//				Remaining: 200,
//				Type:      msg.Order_GTC,
//				Timestamp: 1,
//				Id:        "id-number-one",
//			},
//		},
//	}
//
//	testPriceLevel := orderBookSide.getPriceLevel(testPrice)
//	testPriceLevel.addOrder(&ordersSitingAtPriceLevel[0])
//	testPriceLevel.addOrder(&ordersSitingAtPriceLevel[1])
//
//	//// addingOrders to price level will increase totalVolume accordingly, test totalVolume is correct
//	//assert.Equal(t, uint64(300), orderBookSide.totalVolume)
//
//	// fetch price level from the orderBookSide and make sure data is consistent
//	fetchedTestPriceLevel := orderBookSide.getPriceLevel(testPrice)
//
//	// make sure correct number of order entry was added for this price level
//	assert.Equal(t, len(ordersSitingAtPriceLevel), len(fetchedTestPriceLevel.orders))
//
//	//index := 0
//	//element := fetchedTestPriceLevel.orders.Front()
//	for i, orderEntry := range fetchedTestPriceLevel.orders {
//		//orderEntry := element.Value.(*OrderEntry)
//
//		expectOrder(t, orderEntry.order, ordersSitingAtPriceLevel[i].order)
//
//		//element = element.Next()
//		//index++
//	}
//
//	// check volume by timestamp
//	assert.Equal(t, uint64(100), fetchedTestPriceLevel.volumeByTimestamp[0])
//	assert.Equal(t, uint64(200), fetchedTestPriceLevel.volumeByTimestamp[1])
//
//	// remove orders
//	fetchedTestPriceLevel.removeOrder(&ordersSitingAtPriceLevel[0])
//	fetchedTestPriceLevel.removeOrder(&ordersSitingAtPriceLevel[1])
//
//	// make sure correct number of order entry is zero
//	assert.Equal(t, 0, len(fetchedTestPriceLevel.orders))
//}
//
///*
//	Given test price level
//	I have 2 orders waiting at the test price level with the same timestamps
//	I expect volume by timestamp to contain one aggregated volume for both orders
//	When I fetch from order book side a price level for the test price
//	I repeat the test and exactly the same behaviour should be expected
//	When I remove orders
//	I expect 0 orders on the price level
//*/
//
//func TestPriceLevelAddsOrdersWithSameTimestampCorrectly(t *testing.T) {
//	/*
//	ASSUMPTIONS:
//	- pro-rata
//	- two different timestamps
//	- aggressive buy order meets best order from sell side
//	- no remaining
//	*/
//
//	const testPrice = 100
//	orderBookSide := newSide(msg.Side_Sell)
//	priceLevel := NewPriceLevel(orderBookSide, testPrice)
//	orderBookSide.levels.ReplaceOrInsert(priceLevel)
//
//	ordersSitingAtPriceLevel := []OrderEntry{
//		{
//			order: &msg.Order{
//				Market:    "testOrderBook",
//				Party:     "B",
//				Side:      msg.Side_Sell,
//				Price:     testPrice,
//				Size:      100,
//				Remaining: 100,
//				Type:      msg.Order_GTC,
//				Timestamp: 0,
//				Id:        "id-number-one",
//			},
//		},
//		{
//			order: &msg.Order{
//				Market:    "testOrderBook",
//				Party:     "B",
//				Side:      msg.Side_Sell,
//				Price:     testPrice,
//				Size:      200,
//				Remaining: 200,
//				Type:      msg.Order_GTC,
//				Timestamp: 0,
//				Id:        "id-number-one",
//			},
//		},
//	}
//
//	testPriceLevel := orderBookSide.getPriceLevel(testPrice)
//	testPriceLevel.addOrder(&ordersSitingAtPriceLevel[0])
//	testPriceLevel.addOrder(&ordersSitingAtPriceLevel[1])
//
//	assert.Equal(t, len(ordersSitingAtPriceLevel), len(testPriceLevel.orders))
//	assert.Equal(t, uint64(300), testPriceLevel.volumeByTimestamp[0])
//	// volume at timestamp 1 should not be allocated
//	if _, exist := testPriceLevel.volumeByTimestamp[1]; exist {
//		t.Fail()
//	}
//
//	// when refetch there should be exactly the same behaviour
//	fetchedTestPriceLevel := orderBookSide.getPriceLevel(testPrice)
//	assert.Equal(t, len(ordersSitingAtPriceLevel), len(fetchedTestPriceLevel.orders))
//	assert.Equal(t, uint64(300), fetchedTestPriceLevel.volumeByTimestamp[0])
//	if _, exist := fetchedTestPriceLevel.volumeByTimestamp[1]; exist {
//		t.Fail()
//	}
//
//	// remove orders
//	fetchedTestPriceLevel.removeOrder(&ordersSitingAtPriceLevel[0])
//	fetchedTestPriceLevel.removeOrder(&ordersSitingAtPriceLevel[1])
//
//	// make sure correct number of order entry is zero
//	assert.Equal(t, 0, len(fetchedTestPriceLevel.orders))
//}
//
////-------------------------------- ORDERS CROSSING ON A SINGLE PRICE LEVEL --------------------------------------------//
//
///*
//	Given test price level
//	I have 2 orders waiting at the test price level with different timestamps
//	I want to cross those 2 sell orders with aggressive buy order, no pro-rata, 0 remaining
//	I want 0 orders to be remaining for this price level
//	I want correct trades to be created
//*/
//
//func TestPriceLevelCrossNoProRataNoRemaining(t *testing.T) {
//	/*
//	ASSUMPTIONS:
//	- no pro-rata
//	- two different timestamps
//	- aggressive buy order meets best order from sell side
//	- no remaining
//	*/
//
//	const testPrice = 100
//	orderBookSide := newSide(msg.Side_Sell)
//	priceLevel := NewPriceLevel(orderBookSide, testPrice)
//	orderBookSide.levels.ReplaceOrInsert(priceLevel)
//
//	ordersSitingAtPriceLevel := []OrderEntry{
//		{
//			order: &msg.Order{
//				Market:    "testOrderBook",
//				Party:     "A",
//				Side:      msg.Side_Sell,
//				Price:     testPrice,
//				Size:      100,
//				Remaining: 100,
//				Type:      msg.Order_GTC,
//				Timestamp: 0,
//				Id:        "id-number-one",
//			},
//		},
//		{
//			order: &msg.Order{
//				Market:    "testOrderBook",
//				Party:     "B",
//				Side:      msg.Side_Sell,
//				Price:     testPrice,
//				Size:      200,
//				Remaining: 200,
//				Type:      msg.Order_GTC,
//				Timestamp: 1,
//				Id:        "id-number-one",
//			},
//		},
//	}
//
//	aggressiveOrder := OrderEntry{
//		order: &msg.Order{
//			Market:    "testOrderBook",
//			Party:     "C",
//			Side:      msg.Side_Buy,
//			Price:     testPrice,
//			Size:      300,
//			Remaining: 300,
//			Type:      msg.Order_GTC,
//			Timestamp: 1,
//			Id:        "id-number-one",
//		},
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
//			Price:     testPrice,
//			Size:      200,
//			Buyer:     "C",
//			Seller:    "B",
//			Aggressor: msg.Side_Buy,
//		},
//	}
//
//	testPriceLevel := orderBookSide.getPriceLevel(testPrice)
//	testPriceLevel.addOrder(&ordersSitingAtPriceLevel[0])
//	testPriceLevel.addOrder(&ordersSitingAtPriceLevel[1])
//
//	var trades []Trade
//	filled := testPriceLevel.uncross(&aggressiveOrder, &trades)
//	assert.Equal(t, true, filled)
//	assert.Equal(t, uint64(0), aggressiveOrder.order.Remaining)
//
//	for i, trade := range trades {
//		expectTrade(t, trade.msg, &expectedTrades[i])
//	}
//}
//
///*
//	Given test price level
//	I have 2 orders waiting at the test price level with different timestamps
//	I want to cross those 2 sell orders with aggressive buy order, no pro-rata with remaining on aggressive order
//	I want aggressive orders to be remaining for this price level
//	I want correct trades to be created
//*/
//
//func TestPriceLevelCrossNoProRataWithRemainingOnAggressiveOrder(t *testing.T) {
//	/*
//	ASSUMPTIONS
//	- no pro-rata
//	- two different timestamps
//	- aggressive buy order meets best order from sell side
//	- remaining on aggressive order
//	*/
//	const testPrice = 100
//	orderBookSide := newSide(msg.Side_Sell)
//	priceLevel := NewPriceLevel(orderBookSide, testPrice)
//	orderBookSide.levels.ReplaceOrInsert(priceLevel)
//
//	ordersSitingAtPriceLevel := []OrderEntry{
//		{
//			order: &msg.Order{
//				Market:    "testOrderBook",
//				Party:     "A",
//				Side:      msg.Side_Sell,
//				Price:     testPrice,
//				Size:      100,
//				Remaining: 100,
//				Type:      msg.Order_GTC,
//				Timestamp: 0,
//				Id:        "id-number-one",
//			},
//		},
//		{
//			order: &msg.Order{
//				Market:    "testOrderBook",
//				Party:     "B",
//				Side:      msg.Side_Sell,
//				Price:     testPrice,
//				Size:      200,
//				Remaining: 200,
//				Type:      msg.Order_GTC,
//				Timestamp: 1,
//				Id:        "id-number-one",
//			},
//		},
//	}
//
//	aggressiveOrder := OrderEntry{
//		order: &msg.Order{
//			Market:    "testOrderBook",
//			Party:     "C",
//			Side:      msg.Side_Buy,
//			Price:     testPrice,
//			Size:      400,
//			Remaining: 400,
//			Type:      msg.Order_GTC,
//			Timestamp: 1,
//			Id:        "id-number-one",
//		},
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
//			Price:     testPrice,
//			Size:      200,
//			Buyer:     "C",
//			Seller:    "B",
//			Aggressor: msg.Side_Buy,
//		},
//	}
//
//	testPriceLevel := orderBookSide.getPriceLevel(testPrice)
//	testPriceLevel.addOrder(&ordersSitingAtPriceLevel[0])
//	testPriceLevel.addOrder(&ordersSitingAtPriceLevel[1])
//
//	var trades []Trade
//	filled := testPriceLevel.uncross(&aggressiveOrder, &trades)
//	assert.Equal(t, false, filled)
//	assert.Equal(t, uint64(100), aggressiveOrder.order.Remaining)
//
//	for i, trade := range trades {
//		expectTrade(t, trade.msg, &expectedTrades[i])
//	}
//
//}
//
///*
//	Given test price level
//	I have 2 orders waiting at the test price level with different timestamps
//	I want to cross those 2 sell orders with aggressive buy order, no pro-rata with remaining on passive order
//	I want passive orders to have remaining for this price level
//	I want following trades to be created
//*/
//
//func TestPriceLevelCrossNoProRataWithRemainingOnPassiveOrder(t *testing.T) {
//	/*
//	ASSUMPTIONS:
//	- no pro-rata
//	- two different timestamps
//	- aggressive buy order meets best order from sell side
//	- remaining on passive order
//	*/
//	const testPrice = 100
//	orderBookSide := newSide(msg.Side_Sell)
//	priceLevel := NewPriceLevel(orderBookSide, testPrice)
//	orderBookSide.levels.ReplaceOrInsert(priceLevel)
//
//	ordersSitingAtPriceLevel := []OrderEntry{
//		{
//			order: &msg.Order{
//				Market:    "testOrderBook",
//				Party:     "A",
//				Side:      msg.Side_Sell,
//				Price:     testPrice,
//				Size:      100,
//				Remaining: 100,
//				Type:      msg.Order_GTC,
//				Timestamp: 0,
//				Id:        "id-number-one",
//			},
//		},
//		{
//			order: &msg.Order{
//				Market:    "testOrderBook",
//				Party:     "B",
//				Side:      msg.Side_Sell,
//				Price:     testPrice,
//				Size:      200,
//				Remaining: 200,
//				Type:      msg.Order_GTC,
//				Timestamp: 1,
//				Id:        "id-number-one",
//			},
//		},
//	}
//
//	aggressiveOrder := OrderEntry{
//		order: &msg.Order{
//			Market:    "testOrderBook",
//			Party:     "C",
//			Side:      msg.Side_Buy,
//			Price:     testPrice,
//			Size:      250,
//			Remaining: 250,
//			Type:      msg.Order_GTC,
//			Timestamp: 1,
//			Id:        "id-number-one",
//		},
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
//			Price:     testPrice,
//			Size:      150,
//			Buyer:     "C",
//			Seller:    "B",
//			Aggressor: msg.Side_Buy,
//		},
//	}
//
//	testPriceLevel := orderBookSide.getPriceLevel(testPrice)
//	testPriceLevel.addOrder(&ordersSitingAtPriceLevel[0])
//	testPriceLevel.addOrder(&ordersSitingAtPriceLevel[1])
//
//	var trades []Trade
//	filled := testPriceLevel.uncross(&aggressiveOrder, &trades)
//	assert.Equal(t, true, filled)
//	assert.Equal(t, uint64(50), ordersSitingAtPriceLevel[1].order.Remaining)
//
//	for i, trade := range trades {
//		expectTrade(t, trade.msg, &expectedTrades[i])
//	}
//}
//
///*
//	Given test price level
//	I have 2 orders waiting at the test price level with same timestamps
//	I want to cross those 2 sell orders with aggressive buy order with pro-rata applied and 0 remaining on aggressive order
//	I want aggressive orders to have 0 remaining for this price level
//	I want correct trades to be created
//*/
//
//func TestPriceLevelCrossWithProRataWithNoRemainingOnAggressive(t *testing.T) {
//	/*
//	ASSUMPTIONS:
//	- PRO-RATED!
//	- Same timestamps
//	- aggressive buy order meets best order from sell side
//	- NO remaining
//	*/
//	const testPrice = 100
//	orderBookSide := newSide(msg.Side_Sell)
//	priceLevel := NewPriceLevel(orderBookSide, testPrice)
//	orderBookSide.levels.ReplaceOrInsert(priceLevel)
//
//	ordersSitingAtPriceLevel := []OrderEntry{
//		{
//			order: &msg.Order{
//				Market:    "testOrderBook",
//				Party:     "A",
//				Side:      msg.Side_Sell,
//				Price:     testPrice,
//				Size:      100,
//				Remaining: 100,
//				Type:      msg.Order_GTC,
//				Timestamp: 0,
//				Id:        "id-number-one",
//			},
//		},
//		{
//			order: &msg.Order{
//				Market:    "testOrderBook",
//				Party:     "B",
//				Side:      msg.Side_Sell,
//				Price:     testPrice,
//				Size:      200,
//				Remaining: 200,
//				Type:      msg.Order_GTC,
//				Timestamp: 0,
//				Id:        "id-number-one",
//			},
//		},
//	}
//
//	aggressiveOrder := OrderEntry{
//		order: &msg.Order{
//			Market:    "testOrderBook",
//			Party:     "C",
//			Side:      msg.Side_Buy,
//			Price:     testPrice,
//			Size:      200,
//			Remaining: 200,
//			Type:      msg.Order_GTC,
//			Timestamp: 1,
//			Id:        "id-number-one",
//		},
//	}
//
//	expectedTrades := []msg.Trade{
//		{
//			Market:    "testOrderBook",
//			Price:     testPrice,
//			Size:      67,
//			Buyer:     "C",
//			Seller:    "A",
//			Aggressor: msg.Side_Buy,
//		},
//		{
//			Market:    "testOrderBook",
//			Price:     testPrice,
//			Size:      133,
//			Buyer:     "C",
//			Seller:    "B",
//			Aggressor: msg.Side_Buy,
//		},
//	}
//
//	testPriceLevel := orderBookSide.getPriceLevel(testPrice)
//	testPriceLevel.addOrder(&ordersSitingAtPriceLevel[0])
//	testPriceLevel.addOrder(&ordersSitingAtPriceLevel[1])
//
//	var trades []Trade
//	filled := testPriceLevel.uncross(&aggressiveOrder, &trades)
//	assert.Equal(t, true, filled)
//	assert.Equal(t, uint64(33), ordersSitingAtPriceLevel[0].order.Remaining)
//	assert.Equal(t, uint64(67), ordersSitingAtPriceLevel[1].order.Remaining)
//	assert.Equal(t, uint64(0), aggressiveOrder.order.Remaining)
//
//	for i, trade := range trades {
//		expectTrade(t, trade.msg, &expectedTrades[i])
//	}
//}
//
//// TODO: DEFENSIVE practices
///* Create incorrect orders with:
//	- remaining bigger than size
//	- negative price
//	- zero price
//	- negative timestamp
//*/
//
//// REMARKS:
//// 1. newTrade should not be a part of OrderBook
//// 2. price levels should not be attached to onder entry
//// 3. passiveOE.priceLevel.addOrder(passiveOE) ??????
