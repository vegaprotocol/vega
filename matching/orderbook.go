package matching

import (
	"fmt"
	"log"

	"vega/proto"

	"github.com/google/btree"
)

type OrderBook struct {
	name            string
	buy             *OrderBookSide
	sell            *OrderBookSide
	lastTradedPrice uint64
	config          Config
	latestTimestamp uint64
}

// Create an order book with a given name
func NewBook(name string, config Config) *OrderBook {
	book := &OrderBook{
		name:   name,
		config: config,
	}

	book.buy = &OrderBookSide{
		side:   msg.Side_Buy,
		levels: btree.New(priceLevelsBTreeDegree),
	}

	book.sell = &OrderBookSide{
		side:   msg.Side_Sell,
		levels: btree.New(priceLevelsBTreeDegree),
	}

	return book
}

func (b *OrderBook) GetOrderConfirmationChannel() []chan msg.OrderConfirmation {
	return b.config.OrderConfirmationChans
}

// Add an order and attempt to uncross the book, returns a TradeSet protobufs message object
func (b *OrderBook) AddOrder(orderMessage *msg.Order) (*msg.OrderConfirmation, msg.OrderError) {
	if err := b.validateOrder(orderMessage); err != msg.OrderError_NONE {
		return nil, err
	}
	if orderMessage.Timestamp > b.latestTimestamp {
		b.latestTimestamp = orderMessage.Timestamp
	}

	orderMessage.Id = DigestOrderMessage(orderMessage)[:10]
	oppositeSide := b.getOppositeSide(orderMessage.Side)

	log.Println("Entry state:")
	b.printState()

	// uncross with opposite
	trades, lastTradedPrice := oppositeSide.cross(orderMessage)
	if lastTradedPrice != 0 {
		b.lastTradedPrice = lastTradedPrice
	}

	if len(*trades) != 0 {
		log.Println()
		log.Println("After cross state:")
		b.printState()
	}

	// if persist add to tradebook to the right side
	if (orderMessage.Type == msg.Order_GTC || orderMessage.Type == msg.Order_GTT) && orderMessage.Remaining > 0 {
		b.getSide(orderMessage.Side).addOrder(orderMessage)
		log.Println("After addOrder state:")
		b.printState()
	}

	orderConfirmation := MakeResponse(orderMessage, trades)
	return orderConfirmation, msg.OrderError_NONE
}

func (b *OrderBook) printState() {
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
		const lineLength = 64
		if len(priceLevel.orders) > 0 {

			log.Printf("priceLevel: %d", priceLevel.price)

			for _, o := range priceLevel.orders {
				var side string
				if o.Side == msg.Side_Buy {
					side = "BUY"
				} else {
					side = "SELL"
				}

				line := fmt.Sprintf("      %s %s @%d size=%d R=%d Type=%d T=%d %s",
					o.Party, side, o.Price, o.Size, o.Remaining, o.Type, o.Timestamp, o.Id)

				log.Println(line)
			}

			log.Println()
		}
		return true
	}
}

func printSlice(s []Trade) {
	fmt.Printf("len=%d cap=%d\n", len(s), cap(s))
}

func (b OrderBook) getSide(orderSide msg.Side) *OrderBookSide {
	if orderSide == msg.Side_Buy {
		return b.buy
	} else { // side == Sell
		return b.sell
	}
}

func (b *OrderBook) getOppositeSide(orderSide msg.Side) *OrderBookSide {
	if orderSide == msg.Side_Buy {
		return b.sell
	} else { // side == Sell
		return b.buy
	}
}

func (b *OrderBook) GetName() string {
	return b.name
}

func (b *OrderBook) GetMarketData() *msg.MarketData {
	return &msg.MarketData{
		BestBid:         b.buy.bestPrice(),
		BestOffer:       b.sell.bestPrice(),
		LastTradedPrice: b.lastTradedPrice,
	}
}

func (b *OrderBook) GetMarketDepth() *msg.MarketDepth {
	return &msg.MarketDepth{
		BuyOrderCount:   b.buy.getOrderCount(),
		SellOrderCount:  b.sell.getOrderCount(),
		BuyOrderVolume:  b.buy.getTotalVolume(),
		SellOrderVolume: b.sell.getTotalVolume(),
		BuyPriceLevels:  uint64(b.buy.getNumberOfPriceLevels()),
		SellPriceLevels: uint64(b.sell.getNumberOfPriceLevels()),
	}
}

func (b *OrderBook) RemoveOrder(o *msg.Order) {
	b.getSide(o.Side).RemoveOrder(o)
}
