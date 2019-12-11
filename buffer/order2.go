package buffer

import (
	"context"

	types "code.vegaprotocol.io/vega/proto"
)

type OrderCh struct {
	base
	buf  []types.Order
	add  chan types.Order
	sub  chan orderSubReq
	subs map[int]chan []types.Order
}

type orderSubReq struct {
	ch    chan OrderSub
	chBuf int
}

type OrderSub struct {
	subscriber
	ch chan []types.Order
}

func NewOrderCh(ctx context.Context) *OrderCh {
	t := &OrderCh{
		base: newBase(),
		buf:  []types.Order{},
		add:  make(chan types.Order),
		sub:  make(chan orderSubReq),
		subs: map[int]chan []types.Order{},
	}
	go t.loop(ctx)
	return t
}

func (t *OrderCh) Add(order types.Order) {
	t.add <- order
}

func (t *OrderCh) Subscribe(buf int) OrderSub {
	ts := orderSubReq{
		ch:    make(chan OrderSub),
		chBuf: buf,
	}
	t.sub <- ts
	sub := <-ts.ch
	close(ts.ch)
	return sub
}

func (t *OrderCh) Unsubscribe(sub OrderSub) {
	sub.cfunc()
	t.unsub <- sub.key
}

// this is the meat of the buffer, any and all calls to the buffer are processed here
func (t *OrderCh) loop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			// close all channels, return
			t.done()
			close(t.sub)
			for _, ch := range t.subs {
				close(ch)
			}
			close(t.add)
			return
		case ts := <-t.sub:
			sCtx, cfunc := context.WithCancel(ctx)
			sub := OrderSub{
				subscriber: subscriber{
					ctx:   sCtx,
					cfunc: cfunc,
					key:   t.getKey(),
				},
				ch: make(chan []types.Order, ts.chBuf),
			}
			t.subs[sub.key] = sub.ch
			t.subscribe(sub.key)
			// return key
			ts.ch <- sub
		case u := <-t.unsub:
			if ch, ok := t.subs[u]; ok {
				close(ch)
				t.keys = append(t.keys, u)
			}
			// remove channel from subs
			delete(t.subs, u)
			t.unsubscribe(u)
		case order := <-t.add:
			t.buf = append(t.buf, order)
		case <-t.flush:
			cpy := t.buf
			// use cap from last slice to ensure the largest possible capacity is preserved
			t.buf = make([]types.Order, 0, cap(cpy))
			for _, ch := range t.subs {
				ch <- cpy
			}
		}
	}
}

func (t *OrderSub) Recv() <-chan []types.Order {
	return t.ch
}
