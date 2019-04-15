package proto

import (
	"sync"
)

var (
	TradePool = &sync.Pool{
		New: func() interface{} {
			return &Trade{}
		},
	}
	OrderPool = &sync.Pool{
		New: func() interface{} {
			return &Order{}
		},
	}
	OrderConfirmationPool = &sync.Pool{
		New: func() interface{} {
			return &OrderConfirmation{}
		},
	}
)

func (o *OrderConfirmation) Release() {
	for _, trade := range o.Trades {
		TradePool.Put(trade)
	}
	o.Order.Release()
	for _, order := range o.PassiveOrdersAffected {
		order.Release()
	}
	OrderConfirmationPool.Put(o)
}

func (o *Order) Release() {
	if o.Remaining == 0 {
		OrderPool.Put(o)
	}
}
