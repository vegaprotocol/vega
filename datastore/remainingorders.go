package datastore

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
	at := -1
	for i, o := range ro.orders {
		if o.price > newOrder.Price {
			continue
		}
		if o.price <= newOrder.Price {
			at = i
			break
		}
	}

	r := &remainingOrderInfo{price: newOrder.Price, remaining: newOrder.Remaining}
	// if not found, append at the end
	if at == -1 {
		ro.orders = append(ro.orders, r)
		return
	}
	ro.orders = append(ro.orders[:at], append([]*remainingOrderInfo{r}, ro.orders[at:]...)...)
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
	at := -1
	for i, o := range ro.orders {
		if o.price < newOrder.Price {
			continue
		}
		if o.price >= newOrder.Price {
			at = i
			break
		}
	}

	r := &remainingOrderInfo{price: newOrder.Price, remaining: newOrder.Remaining}
	// if not found, append at the end
	if at == -1 {
		ro.orders = append(ro.orders, r)
		return
	}
	ro.orders = append(ro.orders[:at], append([]*remainingOrderInfo{r}, ro.orders[at:]...)...)
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
