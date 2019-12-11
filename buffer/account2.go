package buffer

import (
	"context"

	types "code.vegaprotocol.io/vega/proto"
)

type AccountCh struct {
	base
	accs map[string]types.Account
	add  chan types.Account
	sub  chan accSubReq
	subs map[int]chan map[string]types.Account
}

type accSubReq struct {
	ch    chan accSub
	chBuf int
}

type accSub struct {
	subscriber
	ch chan map[string]types.Account
}

func NewAccountCh(ctx context.Context) *AccountCh {
	ac := &AccountCh{
		base: newBase(),
		accs: map[string]types.Account{},
		add:  make(chan types.Account),
		sub:  make(chan accSubReq),
		subs: map[int]chan map[string]types.Account{},
	}
	go ac.loop(ctx)
	return ac
}

func (a *AccountCh) loop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			close(a.sub)
			a.done()
			for _, ch := range a.subs {
				close(ch)
			}
			close(a.add)
			return
		case u := <-a.unsub:
			if ch, ok := a.subs[u]; ok {
				close(ch)
			}
			delete(a.subs, u)
			a.unsubscribe(u)
		case acc := <-a.add:
			key := acc.Id
			acc.Id = ""
			a.accs[key] = acc
		case <-a.flush:
			cpy := a.accs
			a.accs = make(map[string]types.Account, len(cpy))
			for _, ch := range a.subs {
				ch <- cpy
			}
		case req := <-a.sub:
			sCtx, cfunc := context.WithCancel(ctx)
			sub := accSub{
				subscriber: subscriber{
					ctx:   sCtx,
					cfunc: cfunc,
					key:   a.getKey(),
				},
				ch: make(chan map[string]types.Account, req.chBuf),
			}
			a.subs[sub.key] = sub.ch
			a.subscribe(sub.key)
			req.ch <- sub
		}
	}
}

func (a *AccountCh) Add(acc types.Account) {
	a.add <- acc
}

func (a *AccountCh) Subscribe(buf int) accSub {
	req := accSubReq{
		chBuf: buf,
		ch:    make(chan accSub),
	}
	a.sub <- req
	sub := <-req.ch
	close(req.ch)
	return sub
}

func (a *AccountCh) Unsubscribe(sub accSub) {
	// cancel the subscriber context, signaling the subscriber is inactive
	sub.cfunc()
	a.unsub <- sub.key
}

func (a *accSub) Recv() <-chan map[string]types.Account {
	return a.ch
}
