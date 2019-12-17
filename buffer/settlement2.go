package buffer

import (
	"context"

	"code.vegaprotocol.io/vega/events"
)

type SettlementCh struct {
	base
	buf  map[string]map[string]events.SettlePosition
	add  chan events.SettlePosition
	sub  chan settleSubReq
	subs map[int]chan []events.SettlePosition
}

type settleSubReq struct {
	ch    chan settleSub
	chBuf int
}

type settleSub struct {
	subscriber
	ch chan []events.SettlePosition
}

func NewSettlementCh(ctx context.Context) *SettlementCh {
	sb := SettlementCh{
		base: newBase(),
		buf:  map[string]map[string]events.SettlePosition{},
		add:  make(chan events.SettlePosition),
		sub:  make(chan settleSubReq),
		subs: map[int]chan []events.SettlePosition{},
	}
	go sb.loop(ctx)
	return &sb
}

func (s *SettlementCh) loop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			close(s.sub)
			s.done()
			for _, ch := range s.subs {
				close(ch)
			}
			close(s.add)
			return
		case e := <-s.add:
			mkt, party := e.MarketID(), e.Party()
			if _, ok := s.buf[mkt]; !ok {
				s.buf[mkt] = map[string]events.SettlePosition{}
			}
			s.buf[mkt][party] = e
		case req := <-s.sub:
			sCtx, cfunc := context.WithCancel(ctx)
			sub := settleSub{
				subscriber: subscriber{
					ctx:   sCtx,
					cfunc: cfunc,
					key:   s.getKey(),
				},
				ch: make(chan []events.SettlePosition, req.chBuf),
			}
			s.subs[sub.key] = sub.ch
			s.subscribe(sub.key)
			req.ch <- sub
		case u := <-s.unsub:
			if ch, ok := s.subs[u]; ok {
				close(ch)
			}
			delete(s.subs, u)
			s.unsubscribe(u)
		case <-s.flush:
			cpy := s.buf
			s.buf = map[string]map[string]events.SettlePosition{}
			// s.buf = make(map[string]map[string]events.SettlePosition, len(cpy))
			slice := make([]events.SettlePosition, 0, len(cpy))
			for _, tmap := range cpy {
				for _, e := range tmap {
					slice = append(slice, e)
				}
			}
			for _, ch := range s.subs {
				ch <- slice
			}
		}
	}
}

func (s *SettlementCh) Add(e events.SettlePosition) {
	s.add <- e
}

func (s *SettlementCh) Subscribe(chBuf int) settleSub {
	req := settleSubReq{
		ch:    make(chan settleSub),
		chBuf: chBuf,
	}
	s.sub <- req
	sub := <-req.ch
	close(req.ch)
	return sub
}

func (s *SettlementCh) Unsubscribe(sub settleSub) {
	sub.cfunc()
	s.unsub <- sub.key
}

func (s *settleSub) Recv() <-chan []events.SettlePosition {
	return s.ch
}
