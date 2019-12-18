package buffer

import (
	"context"

	types "code.vegaprotocol.io/vega/proto"
)

type MarketCh struct {
	base
	buf  []types.Market
	add  chan types.Market
	sub  chan marketSubReq
	subs map[int]chan []types.Market
}

type marketSubReq struct {
	ch    chan MarketSub
	chBuf int
}

type MarketSub struct {
	subscriber
	ch chan []types.Market
}

func NewMarketCh(ctx context.Context) *MarketCh {
	mc := MarketCh{
		base: newBase(),
		buf:  []types.Market{},
		add:  make(chan types.Market),
		sub:  make(chan marketSubReq),
		subs: map[int]chan []types.Market{},
	}
	go mc.loop(ctx)
	return &mc
}

func (m *MarketCh) loop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			close(m.sub)
			m.done()
			for _, ch := range m.subs {
				close(ch)
			}
			close(m.add)
			return
		case mkt := <-m.add:
			m.buf = append(m.buf, mkt)
		case u := <-m.unsub:
			if ch, ok := m.subs[u]; ok {
				close(ch)
			}
			delete(m.subs, u)
			m.unsubscribe(u)
		case req := <-m.sub:
			sCtx, cfunc := context.WithCancel(ctx)
			sub := MarketSub{
				subscriber: subscriber{
					ctx:   sCtx,
					cfunc: cfunc,
					key:   m.getKey(),
				},
				ch: make(chan []types.Market, req.chBuf),
			}
			m.subs[sub.key] = sub.ch
			m.subscribe(sub.key)
			req.ch <- sub
		case <-m.flush:
			cpy := m.buf
			m.buf = make([]types.Market, 0, cap(cpy))
			for _, ch := range m.subs {
				ch <- cpy
			}
		}
	}
}

func (m *MarketCh) Add(mkt types.Market) {
	m.add <- mkt
}

func (m *MarketCh) Subscribe(chBuf int) MarketSub {
	req := marketSubReq{
		ch:    make(chan MarketSub),
		chBuf: chBuf,
	}
	m.sub <- req
	sub := <-req.ch
	close(req.ch)
	return sub
}

func (m *MarketCh) Unsubscribe(s MarketSub) {
	s.cfunc()
	m.unsub <- s.key
}

func (m *MarketSub) Recv() <-chan []types.Market {
	return m.ch
}
