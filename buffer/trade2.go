package buffer

import (
	"context"

	types "code.vegaprotocol.io/vega/proto"
)

type TradeCh struct {
	base
	buf  []types.Trade
	add  chan types.Trade
	sub  chan TradeSubReq
	subs map[int]chan []types.Trade
}

type TradeSubReq struct {
	ch    chan TradeSub
	chBuf int
}

type TradeSub struct {
	subscriber
	ch chan []types.Trade
}

func NewTradeCh(ctx context.Context) *TradeCh {
	t := &TradeCh{
		base: newBase(),
		buf:  []types.Trade{},
		add:  make(chan types.Trade),
		sub:  make(chan TradeSubReq),
		subs: map[int]chan []types.Trade{},
	}
	go t.loop(ctx)
	return t
}

func (t *TradeCh) Add(trade types.Trade) {
	t.add <- trade
}

func (t *TradeCh) Subscribe(buf int) TradeSub {
	ts := TradeSubReq{
		ch:    make(chan TradeSub),
		chBuf: buf,
	}
	t.sub <- ts
	sub := <-ts.ch
	close(ts.ch)
	return sub
}

func (t *TradeCh) Unsubscribe(sub TradeSub) {
	sub.cfunc()
	t.unsub <- sub.key
}

// this is the meat of the buffer, any and all calls to the buffer are processed here
func (t *TradeCh) loop(ctx context.Context) {
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
			sub := TradeSub{
				subscriber: subscriber{
					ctx:   sCtx,
					cfunc: cfunc,
					key:   t.getKey(),
				},
				ch: make(chan []types.Trade, ts.chBuf),
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
		case trade := <-t.add:
			t.buf = append(t.buf, trade)
		case <-t.flush:
			cpy := t.buf
			// use cap from last slice to ensure the largest possible capacity is preserved
			t.buf = make([]types.Trade, 0, cap(cpy))
			for _, ch := range t.subs {
				ch <- cpy
			}
		}
	}
}

func (t *TradeSub) Recv() <-chan []types.Trade {
	return t.ch
}
