package datastore

import "github.com/golang/go/src/pkg/fmt"

type BuySideRemainingOrders struct {
	orders []*remainingOrderInfo
}
type SellSideRemainingOrders struct {
	orders []*remainingOrderInfo
}

type remainingOrderInfo struct {
	price     uint64
	remaining uint64
}

func (ro *BuySideRemainingOrders) insert(newOrder *Order) {
	fmt.Printf("inserting into BuySideRemainingOrders %+v\n", newOrder)
	at := ro.getIndex(newOrder)

	r := &remainingOrderInfo{price: newOrder.Price, remaining: newOrder.Remaining}
	// if not found, append at the end
	if at == -1 {
		ro.orders = append(ro.orders, r)
		fmt.Printf("BuySideRemainingOrders append %\n", len(ro.orders))
		return
	}
	ro.orders = append(ro.orders[:at], append([]*remainingOrderInfo{r}, ro.orders[at:]...)...)
	fmt.Printf("BuySideRemainingOrders %d\n", len(ro.orders))
}

func (ro BuySideRemainingOrders) getIndex(order *Order) int {
	at := -1
	for i, o := range ro.orders {
		if o.price > order.Price {
			continue
		}
		if o.price == order.Price {
			at = i
			break
		}
	}
	return at
}

func (ro *BuySideRemainingOrders) update(updatedOrder *Order) {
	at := ro.getIndex(updatedOrder)

	// if not found, append at the end
	if at == -1 {
		return
	}
	update := &remainingOrderInfo{price: updatedOrder.Price, remaining: updatedOrder.Remaining}
	ro.orders[at] = update
}

func (ro *BuySideRemainingOrders) remove(rmOrder *Order) {
	toDelete := ro.getIndex(rmOrder)
	if toDelete != -1 {
		copy(ro.orders[toDelete:], ro.orders[toDelete+1:])
		ro.orders = ro.orders[:len(ro.orders)-1]
	}
	if toDelete == -1 {
		// TODO: implement ORDER_NOT_FOUND_ERROR and add to protobufs
		return
	}
}

func (ro *SellSideRemainingOrders) insert(newOrder *Order) {
	fmt.Printf("inserting into SellSideRemainingOrders %+v\n", newOrder)
	at := ro.getIndex(newOrder)

	r := &remainingOrderInfo{price: newOrder.Price, remaining: newOrder.Remaining}
	// if not found, append at the end
	if at == -1 {
		ro.orders = append(ro.orders, r)
		fmt.Printf("SellSideRemainingOrders append %d\n", len(ro.orders))
		return
	}
	ro.orders = append(ro.orders[:at], append([]*remainingOrderInfo{r}, ro.orders[at:]...)...)
	fmt.Printf("SellSideRemainingOrders %d\n", len(ro.orders))
}

func (ro SellSideRemainingOrders) getIndex(order *Order) int {
	at := -1
	for i, o := range ro.orders {
		if o.price < order.Price {
			continue
		}
		if o.price == order.Price {
			at = i
			break
		}
	}
	return at
}

func (ro *SellSideRemainingOrders) update(updatedOrder *Order) {
	at := ro.getIndex(updatedOrder)

	// if not found, append at the end
	if at == -1 {
		return
	}
	update := &remainingOrderInfo{price: updatedOrder.Price, remaining: updatedOrder.Remaining}
	ro.orders[at] = update
}

func (ro *SellSideRemainingOrders) remove(rmOrder *Order) {
	toDelete := ro.getIndex(rmOrder)
	if toDelete != -1 {
		copy(ro.orders[toDelete:], ro.orders[toDelete+1:])
		ro.orders = ro.orders[:len(ro.orders)-1]
	}
	if toDelete == -1 {
		// TODO: implement ORDER_NOT_FOUND_ERROR and add to protobufs
		return
	}
}
