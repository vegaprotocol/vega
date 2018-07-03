package matching

import (
	"fmt"
	"log"
	//"sync"
	"vega/proto"

	"github.com/golang/protobuf/proto"
	"github.com/google/btree"
	"golang.org/x/crypto/sha3"
	"github.com/golang/go/src/pkg/strconv"
	"sync"
)

var Pool = &sync.Pool{}

type OrderBook struct {
	name            string
	buy             *OrderBookSide
	sell            *OrderBookSide
	lastTradedPrice uint64
	config          Config
	latestTimestamp uint64

	ReqNumber int64
	//mutex     sync.Mutex
	//quit chan bool
}

// Create an order book with a given name
func NewBook(name string, config Config) *OrderBook {
	book := &OrderBook{
		name:   name,
		config: config,
		//quit: make(chan bool),
	}

	book.buy = &OrderBookSide{
		side:   msg.Side_Buy,
		levels: btree.New(priceLevelsBTreeDegree),
	}

	book.sell = &OrderBookSide{
		side:   msg.Side_Sell,
		levels: btree.New(priceLevelsBTreeDegree),
	}
	//go book.scheduleCleanup()
	return book
}

//func (b *OrderBook) Stop() {
//	b.quit <- true
//}

// Add an order and attempt to uncross the book, returns a TradeSet protobufs message object
func (b *OrderBook) AddOrder(order *msg.Order) (*msg.OrderConfirmation, msg.OrderError) {
	if err := b.validateOrder(order); err != msg.OrderError_NONE {
		return nil, err
	}

	//b.mutex.Lock()
	b.ReqNumber++
	//if b.ReqNumber % 100 == 0 {
	//	b.buy.levels.Descend(collectGarbage)
	//	b.sell.levels.Descend(collectGarbage)
	//}

	//order.Id = calculateHash(order)[:10]
	//order.Id = fmt.Sprintf("%d", time.Now().UnixNano())
	order.Id = strconv.FormatInt(b.ReqNumber, 10)

	if order.Timestamp > b.latestTimestamp {
		b.latestTimestamp = order.Timestamp
	}

	//b.PrintState("Entry state:")

	// uncross with opposite
	trades, impactedOrders, lastTradedPrice := b.getOppositeSide(order.Side).uncross(order)
	if lastTradedPrice != 0 {
		b.lastTradedPrice = lastTradedPrice
	}

	// if state of the book changed show state
	if len(trades) != 0 {
		//b.PrintState("After uncross state:")
	}

	// if order is persistent type add to order book to the correct side
	if (order.Type == msg.Order_GTC || order.Type == msg.Order_GTT) && order.Remaining > 0 {
		b.getSide(order.Side).addOrder(order)

		//b.PrintState("After addOrder state:")
	}

	orderConfirmation := makeResponse(order, trades, impactedOrders)
	//b.mutex.Unlock()
	return orderConfirmation, msg.OrderError_NONE
}

func (b *OrderBook) RemoveOrder(order *msg.Order) error {
	//b.mutex.Lock()
	b.ReqNumber++
	err := b.getSide(order.Side).RemoveOrder(order)
	//b.mutex.Unlock()
	return err
}

func (b OrderBook) getSide(orderSide msg.Side) *OrderBookSide {
	if orderSide == msg.Side_Buy {
		return b.buy
	} else {
		return b.sell
	}
}

func (b *OrderBook) getOppositeSide(orderSide msg.Side) *OrderBookSide {
	if orderSide == msg.Side_Buy {
		return b.sell
	} else {
		return b.buy
	}
}

// Calculate the hash (ID) of the order details (as serialised by protobufs)
func calculateHash(order *msg.Order) string {
	bytes, _ := proto.Marshal(order)
	hash := make([]byte, 64)
	sha3.ShakeSum256(hash, bytes)
	return fmt.Sprintf("%x", hash)
}


func makeResponse(order *msg.Order, trades []Trade, impactedOrders []msg.Order) *msg.OrderConfirmation {
	tradeSet := make([]*msg.Trade, 0)
	for _, t := range trades {
		tradeSet = append(tradeSet, t.toMessage())
	}
	passiveOrdersAffected := make([]*msg.Order, 0)
	for i := range impactedOrders {
		passiveOrdersAffected = append(passiveOrdersAffected, &impactedOrders[i])
	}
	return &msg.OrderConfirmation{
		Order: order,
		PassiveOrdersAffected: passiveOrdersAffected,
		Trades:                tradeSet,
	}
}

//func (b *OrderBook) scheduleCleanup() {
//	//var operatingAt int64
//	for {
//		select {
//		case <-b.quit:
//			return
//		default:
//			time.Sleep(1000 * time.Millisecond)
//			//if b.ReqNumber != 0 && operatingAt != b.ReqNumber && b.ReqNumber%1000 == 0 {
//			b.mutex.Lock()
//			b.buy.levels.Descend(collectGarbage)
//			b.sell.levels.Descend(collectGarbage)
//			//log.Println("FINISHED")
//			//operatingAt = b.ReqNumber
//			b.mutex.Unlock()
//			//}
//		}
//	}
//}

//func collectGarbage(i btree.Item) bool {
//		priceLevel := i.(*PriceLevel)
//		priceLevel.collectGarbage()
//		return true
//}


func (b *OrderBook) PrintState(msg string) {
	log.Println()
	log.Println(msg)
	log.Println("------------------------------------------------------------")
	log.Println("                        BUY SIDE                            ")
	b.buy.levels.Descend(printOrders())
	log.Println("------------------------------------------------------------")
	log.Println("                        SELL SIDE                           ")
	b.sell.levels.Ascend(printOrders())
	log.Println("------------------------------------------------------------")

}

func printOrders() func(i btree.Item) bool {
	return func(i btree.Item) bool {
		priceLevel := i.(*PriceLevel)
		if len(priceLevel.orders) > 0 {

			log.Printf("priceLevel: %d", priceLevel.price)

			for _, o := range priceLevel.orders {
				var side string
				if o.order.Side == msg.Side_Buy {
					side = "BUY"
				} else {
					side = "SELL"
				}

				line := fmt.Sprintf("      %s %s @%d size=%d R=%d Type=%d T=%d %s",
					o.order.Party, side, o.order.Price, o.order.Size, o.order.Remaining, o.order.Type, o.order.Timestamp, o.order.Id)

				log.Println(line)
			}

			log.Println()
		}
		return true
	}
}
