package datastore

type BuySideRemainingOrders []*remainingOrderInfo
type SellSideRemainingOrders []*remainingOrderInfo

type remainingOrderInfo struct {
	price     uint64
	remaining uint64
}

func (orders BuySideRemainingOrders) insert(newOrder *Order) {
	at := orders.getIndex(newOrder)

	r := &remainingOrderInfo{price: newOrder.Price, remaining: newOrder.Remaining}
	// if not found, append at the end
	if at == -1 {
		orders = append(orders, r)
		return
	}
	orders = append(orders[:at], append([]*remainingOrderInfo{r}, orders[at:]...)...)
}

func (orders BuySideRemainingOrders) getIndex(order *Order) int {
	at := -1
	for i, o := range orders {
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

func (orders BuySideRemainingOrders) update(updatedOrder *Order) {
	at := orders.getIndex(updatedOrder)

	// if not found, append at the end
	if at == -1 {
		return
	}
	update := &remainingOrderInfo{price: updatedOrder.Price, remaining: updatedOrder.Remaining}
	orders[at] = update
}

func (orders BuySideRemainingOrders) remove(rmOrder *Order) {
	toDelete := orders.getIndex(rmOrder)
	if toDelete != -1 {
		copy(orders[toDelete:], orders[toDelete+1:])
		orders = orders[:len(orders)-1]
	}
	if toDelete == -1 {
		// TODO: implement ORDER_NOT_FOUND_ERROR and add to protobufs
		return
	}
}

func (orders SellSideRemainingOrders) insert(newOrder *Order) {
	at := orders.getIndex(newOrder)

	r := &remainingOrderInfo{price: newOrder.Price, remaining: newOrder.Remaining}
	// if not found, append at the end
	if at == -1 {
		orders = append(orders, r)
		return
	}
	orders = append(orders[:at], append([]*remainingOrderInfo{r}, orders[at:]...)...)
}

func (orders SellSideRemainingOrders) getIndex(order *Order) int {
	at := -1
	for i, o := range orders {
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

func (orders SellSideRemainingOrders) update(updatedOrder *Order) {
	at := orders.getIndex(updatedOrder)

	// if not found, append at the end
	if at == -1 {
		return
	}
	update := &remainingOrderInfo{price: updatedOrder.Price, remaining: updatedOrder.Remaining}
	orders[at] = update
}

func (orders SellSideRemainingOrders) remove(rmOrder *Order) {
	toDelete := orders.getIndex(rmOrder)
	if toDelete != -1 {
		copy(orders[toDelete:], orders[toDelete+1:])
		orders = orders[:len(orders)-1]
	}
	if toDelete == -1 {
		// TODO: implement ORDER_NOT_FOUND_ERROR and add to protobufs
		return
	}
}
