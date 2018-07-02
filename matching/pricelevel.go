package matching

import (
	"math"
	"sync"

	"vega/proto"


	"github.com/google/btree"
)

type PriceLevel struct {
	price             uint64
	orders            []*OrderEntry
	lookupTable       map[string]int
	volumeAtTimestamp map[uint64]uint64
	reqNumber         uint64

	mutex 			  sync.Mutex
}

func NewPriceLevel(price uint64) *PriceLevel {
	return &PriceLevel{
		price:             price,
		orders:            make([]*OrderEntry, 0),
		volumeAtTimestamp: make(map[uint64]uint64),
		lookupTable:       make(map[string]int),
	}
}

type OrderEntry struct {
	order *msg.Order
	valid bool
}

func newOrderEntry(order *msg.Order) *OrderEntry{
	return &OrderEntry{order, true}
}

func (l PriceLevel) Less(other btree.Item) bool {
	return l.price < other.(*PriceLevel).price
}

func (l *PriceLevel) addOrder(o *msg.Order) {
	// adjust volume by timestamp map for correct pro-rata calculation
	l.increaseVolumeByTimestamp(o)

	// add orders to slice of orders on this price level
	l.orders = append(l.orders, newOrderEntry(o))

	// add index to lookup table for faster removal
	l.lookupTable[o.Id] = len(l.orders) - 1
}

func (l *PriceLevel) collectGarbage() {
	//log.Println("collecting garbage")
	l.mutex.Lock()
	newOrders := make([]*OrderEntry, 0)
	for i, _ := range l.orders {
		if l.orders[i].valid {
			newOrders = append(newOrders, newOrderEntry(l.orders[i].order))
		}
	}
	l.orders = newOrders
	l.mutex.Unlock()
}

//func (l *PriceLevel) removeOrder(o *msg.Order, index int) error {
//	// adjust volume by timestamp map for correct pro-rata calculation
//	l.decreaseVolumeByTimestamp(o)
//
//	l.orders[index].valid = false
//
//	log.Println("l.ReqNumber: ", l.ReqNumber)
//
//	//if math.Mod(float64(l.ReqNumber),100) == 0 {
//	//	log.Println("collecting garbage")
//	//	l.collectGarbage()
//
//
//		// memcopy orders sliced at the index
//		//copy(l.orders[index:], l.orders[index+1:])
//		//l.orders = l.orders[:len(l.orders)-1]
//		//
//		//// delete index from lookupTable
//		//delete(l.lookupTable, o.Id)
//		//
//		//// reindex lookup table
//		//for k, val := range l.lookupTable {
//		//	if val > index {
//		//		l.lookupTable[k] = val - 1
//		//	}
//		//}
//	//}
//
//	return nil
//}

func (l *PriceLevel) increaseVolumeByTimestamp(o *msg.Order) {
	if vbt, exists := l.volumeAtTimestamp[o.Timestamp]; exists {
		l.volumeAtTimestamp[o.Timestamp] = vbt + o.Remaining
	} else {
		l.volumeAtTimestamp[o.Timestamp] = o.Remaining
	}
}

func (l *PriceLevel) decreaseVolumeByTimestamp(o *msg.Order) {
	if vbt, exists := l.volumeAtTimestamp[o.Timestamp]; exists {
		if vbt <= o.Remaining {
			delete(l.volumeAtTimestamp, o.Timestamp)
		} else {
			l.volumeAtTimestamp[o.Timestamp] = vbt - o.Remaining
		}
	}
}

func (l *PriceLevel) adjustVolumeByTimestamp(currentTimestamp uint64, trade *Trade) {
	if vbt, exists := l.volumeAtTimestamp[currentTimestamp]; exists {
		l.volumeAtTimestamp[currentTimestamp] = vbt - trade.size
	}
}

func (l *PriceLevel) uncross(agg *msg.Order, trades *[]Trade, impactedOrders *[]msg.Order) bool {
	//log.Printf("                UNCOROSSING ATTEMPT at price = %d", l.price)
	//log.Println("-> aggressive order: ", agg)
	//log.Println()

	volumeToShare := agg.Remaining

	// start from earliest timestamp
	currentTimestamp := l.earliestTimestamp()
	totalVolumeAtTimestamp := l.volumeAtTimestamp[currentTimestamp]

	//var ordersScheduledForDeletion []msg.Order

	// l.orders is always sorted by timestamps, that is why when iterating we always start from the beginning
	for i := 0; i < len(l.orders); i++ {
		//log.Println("Passive order: ", l.orders[i])
		if !l.orders[i].valid {
			continue
		}

		// See if we are at a new top timestamp
		if currentTimestamp != l.orders[i].order.Timestamp {
			// if consumed all orders on the current timestamp, delete exhausted timestamp and proceed to the next one
			delete(l.volumeAtTimestamp, currentTimestamp)
			// assign new timestamp
			currentTimestamp = l.orders[i].order.Timestamp
			// assign new volume at timestamp
			totalVolumeAtTimestamp = l.volumeAtTimestamp[currentTimestamp]
			volumeToShare = agg.Remaining
		}

		// Get size and make newTrade
		size := l.getVolumeAllocation(agg, l.orders[i].order, volumeToShare, totalVolumeAtTimestamp)
		if size <= 0 {
			panic("Trade.size > order.remaining")
		}

		// New Trade
		trade := newTrade(agg, l.orders[i].order, size)

		// Update Remaining for both aggressive and passive
		agg.Remaining -= trade.size
		l.orders[i].order.Remaining -= trade.size

		// Schedule order for deletion
		if l.orders[i].order.Remaining == 0 {
			//ordersScheduledForDeletion = append(ordersScheduledForDeletion, *l.orders[i].order)
			l.decreaseVolumeByTimestamp(l.orders[i].order)
			l.orders[i].valid = false
			//log.Println("setting valid to false")
		}

		// Update Volumes for the price level
		l.adjustVolumeByTimestamp(currentTimestamp, trade)

		// Update trades
		*trades = append(*trades, *trade)
		*impactedOrders = append(*impactedOrders, *l.orders[i].order)

		// Exit when done
		if agg.Remaining == 0 {
			break
		}
	}

	// Clean passive orders with zero remaining
	//l.clearOrders(ordersScheduledForDeletion)

	//log.Println("                    UNCOROSSING FINISHED                   ")
	//log.Println()

	return agg.Remaining == 0
}

func (l *PriceLevel) clearOrders(ordersScheduledForDeletion []msg.Order) {
	for _, ordersScheduledForDeletion := range ordersScheduledForDeletion {
		l.removeOrderFromPriceLevel(&ordersScheduledForDeletion)
	}
}

func (l *PriceLevel) removeOrderFromPriceLevel(orderForDeletion *msg.Order) error {
	//index := l.lookupTable[orderForDeletion.Id]
	for i, _ := range l.orders {
		if l.orders[i].order.Id == orderForDeletion.Id {
			l.decreaseVolumeByTimestamp(l.orders[i].order)
			l.orders[i].valid = false
			//log.Println("removed")
		}
	}
	return nil

	//l.decreaseVolumeByTimestamp(l.orders[i].order)
	//
	//l.orders[i].valid = false
	//return l.removeOrder(orderForDeletion, index)
}

func (l *PriceLevel) earliestTimestamp() uint64 {
	if len(l.orders) != 0 {
		return l.orders[0].order.Timestamp
	}
	return 0
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

// Returns the min of 2 uint64s
func min(x, y uint64) uint64 {
	if y < x {
		return y
	}
	return x
}
