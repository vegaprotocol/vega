package matching

import (
	"errors"
	"log"
	"math"

	"vega/proto"

	"github.com/google/btree"
)

type PriceLevel struct {
	price             uint64
	volume            uint64
	volumeByTimestamp map[uint64]uint64
	orders            []msg.Order
}

func NewPriceLevel(price uint64) *PriceLevel {
	return &PriceLevel{
		price:             price,
		orders:            make([]msg.Order, 0),
		volumeByTimestamp: make(map[uint64]uint64),
	}
}

func (l *PriceLevel) firstOrder() *msg.Order {
	return &l.orders[0]
}

func (l *PriceLevel) topTimestamp() uint64 {
	if len(l.orders) != 0 {
		return l.orders[0].Timestamp
	}
	return 0
}

func (l *PriceLevel) addOrder(o *msg.Order) {
	if vbt, exists := l.volumeByTimestamp[o.Timestamp]; exists {
		l.volumeByTimestamp[o.Timestamp] = vbt + o.Remaining
	} else {
		l.volumeByTimestamp[o.Timestamp] = o.Remaining
	}
	l.volume += o.Remaining

	l.orders = append(l.orders, *o)
	log.Println("adding order to the order book: ", o)
	log.Println("state of slice ", l.orders)
}

func (l *PriceLevel) removeOrder(o *msg.Order) {
	log.Println("removeOrder called on ", o)
	if vbt, exists := l.volumeByTimestamp[o.Timestamp]; exists {
		if vbt <= o.Remaining {
			delete(l.volumeByTimestamp, o.Timestamp)
		} else {
			l.volumeByTimestamp[o.Timestamp] = vbt - o.Remaining
		}
	}
	l.volume -= o.Remaining
	i, err := l.getIndexForDelition(o.Id)
	if err != nil {
		return
	}
	//log.Println("index for delition ", i)
	//log.Println("orders before deletion: ", l.orders)
	l.orders = append(l.orders[:i], l.orders[i+1:]...)
	//log.Println("orders after deletion: ", l.orders)
}

func (l *PriceLevel) getIndexForDelition(orderId string) (int, error) {
	for index, orderForDeletion := range l.orders {
		if orderForDeletion.Id == orderId {
			return index, nil
		}
	}
	return 0, errors.New("NOT_FOUND")
}

func (l PriceLevel) Less(other btree.Item) bool {
	otherPrice := other.(*PriceLevel).price
	//return l.price > other.(*PriceLevel).price
	log.Printf("comparing %d with %d \n", l.price, otherPrice)
	return l.price < otherPrice
}

func (l *PriceLevel) uncross(agg *msg.Order, trades *[]Trade) bool {
	log.Printf("                UNCOROSSING ATTEMPT at price = %d", l.price)
	log.Println("-> aggressive order: ", agg)
	log.Println()

	volumeToShare := agg.Remaining
	currentTimestamp := l.topTimestamp()
	initialVolumeAtTimestamp := l.volumeByTimestamp[currentTimestamp]

	var ordersScheduledForDeletion []msg.Order

	// l.orders is always sorted by timestamps, that is why when iterating we always start from the beginning
	for i := 0; i < len(l.orders); i++ {
		log.Println("Passive order: ", l.orders[i])

		// See if we are at a new top time
		if currentTimestamp != l.orders[i].Timestamp {
			delete(l.volumeByTimestamp, currentTimestamp)
			currentTimestamp = l.orders[i].Timestamp
			initialVolumeAtTimestamp = l.volumeByTimestamp[currentTimestamp]
			volumeToShare = agg.Remaining
		}

		// Get size and make newTrade
		size := l.getVolumeAllocation(agg, &l.orders[i], volumeToShare, initialVolumeAtTimestamp)
		if size <= 0 {
			panic("Trade.size > order.remaining")
		}

		// New Trade
		trade := newTrade(agg, &l.orders[i], size)
		log.Printf("Matched: %v\n", trade)

		// Update Remaining for both aggressive and passive
		agg.Remaining -= trade.size
		l.orders[i].Remaining -= trade.size

		// Schedule order for deletion
		if l.orders[i].Remaining == 0 {
			ordersScheduledForDeletion = append(ordersScheduledForDeletion, l.orders[i])
		}

		// Update Volumes for the price level
		l.volume -= trade.size
		if vbt, exists := l.volumeByTimestamp[currentTimestamp]; exists {
			l.volumeByTimestamp[currentTimestamp] = vbt - trade.size
		}

		// Update trades
		*trades = append(*trades, *trade)


		// Exit when done
		if agg.Remaining == 0 {
			break
		}
	}

	// Clean passive orders with zero remaining
	for _, ordersScheduledForDeletion := range ordersScheduledForDeletion {
		l.removeOrder(&ordersScheduledForDeletion)
	}
	log.Println("                    UNCOROSSING FINISHED                ")
	log.Println()
	return agg.Remaining == 0
}

// Get size for a specific trade assuming aggressive order volume is allocated pro-rata among all passive trades
// with the same timestamp by their share of the total volume with the same price and timestamp. (NB: "normal"
// trading would thus *always* increment the logical timestamp between trades.)
func (l *PriceLevel) getVolumeAllocation(
	agg, pass *msg.Order,
	volumeToShare, initialVolumeAtTimestamp uint64) uint64 {

	weight := float64(pass.Remaining) / float64(initialVolumeAtTimestamp)
	size := weight * float64(min(volumeToShare, initialVolumeAtTimestamp))
	if size-math.Trunc(size) > 0 {
		size++ // Otherwise we can end up allocating 1 short because of integer division rounding
	}
	return min(min(uint64(size), agg.Remaining), pass.Remaining)
}
