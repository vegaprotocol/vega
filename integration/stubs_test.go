package core_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"code.vegaprotocol.io/vega/broker"
	"code.vegaprotocol.io/vega/events"
	types "code.vegaprotocol.io/vega/proto"
)

type brokerStub struct {
	mu   sync.Mutex
	data map[events.Type][]events.Event
	subT map[events.Type][]broker.Subscriber
}

func NewBrokerStub() *brokerStub {
	return &brokerStub{
		data: map[events.Type][]events.Event{},
		subT: map[events.Type][]broker.Subscriber{},
	}
}

func (b *brokerStub) Subscribe(sub broker.Subscriber) {
	b.mu.Lock()
	types := sub.Types()
	for _, t := range types {
		if _, ok := b.subT[t]; !ok {
			b.subT[t] = []broker.Subscriber{}
		}
		b.subT[t] = append(b.subT[t], sub)
	}
	b.mu.Unlock()
}

func (b *brokerStub) SendBatch(evts []events.Event) {
	if len(evts) == 0 {
		return
	}
	t := evts[0].Type()
	b.mu.Lock()
	if subs, ok := b.subT[t]; ok {
		for _, sub := range subs {
			if sub.Ack() {
				sub.Push(evts...)
				continue
			}
			select {
			case <-sub.Closed():
				continue
			case <-sub.Skip():
				continue
			case sub.C() <- evts:
				continue
			default:
				continue
			}
		}
	}
	if _, ok := b.data[t]; !ok {
		b.data[t] = []events.Event{}
	}
	b.data[t] = append(b.data[t], evts...)
	b.mu.Unlock()
}

func (b *brokerStub) Send(e events.Event) {
	b.mu.Lock()
	t := e.Type()
	if subs, ok := b.subT[t]; ok {
		for _, sub := range subs {
			if sub.Ack() {
				sub.Push(e)
			} else {
				select {
				case <-sub.Closed():
					continue
				case <-sub.Skip():
					continue
				case sub.C() <- []events.Event{e}:
					continue
				default:
					continue
				}
			}
		}
	}
	if _, ok := b.data[t]; !ok {
		b.data[t] = []events.Event{}
	}
	b.data[t] = append(b.data[t], e)
	b.mu.Unlock()
}

func (b *brokerStub) GetBatch(t events.Type) []events.Event {
	b.mu.Lock()
	r := b.data[t]
	b.mu.Unlock()
	return r
}

// utility func:
func (b *brokerStub) GetTransferResponses() []events.TransferResponse {
	batch := b.GetBatch(events.TransferResponses)
	if len(batch) == 0 {
		return nil
	}
	b.mu.Lock()
	ret := make([]events.TransferResponse, 0, len(batch))
	for _, e := range batch {
		switch et := e.(type) {
		case *events.TransferResponse:
			ret = append(ret, *et)
		case events.TransferResponse:
			ret = append(ret, et)
		}
	}
	b.mu.Unlock()
	return ret
}

func (b *brokerStub) clearOrderEvents() {
	t := events.OrderEvent
	b.mu.Lock()
	r := b.data[t]
	// reallocate new slice
	b.data[t] = make([]events.Event, 0, cap(r))
	b.mu.Unlock()
}

func (b *brokerStub) getOrdersByPartyAndMarket(party, market string) []types.Order {
	orders := b.GetOrderEvents()
	ret := []types.Order{}
	for _, oe := range orders {
		if o := oe.Order(); o.MarketId == market && o.PartyId == party {
			ret = append(ret, *o)
		}
	}
	return ret
}

func (b *brokerStub) GetOrderEvents() []events.Order {
	batch := b.GetBatch(events.OrderEvent)
	if len(batch) == 0 {
		return nil
	}
	ret := make([]events.Order, 0, len(batch))
	for _, e := range batch {
		switch et := e.(type) {
		case *events.Order:
			ret = append(ret, *et)
		case events.Order:
			ret = append(ret, et)
		}
	}
	return ret
}

func (b *brokerStub) GetLPEvents() []events.LiquidityProvision {
	batch := b.GetBatch(events.LiquidityProvisionEvent)
	if len(batch) == 0 {
		return nil
	}
	ret := make([]events.LiquidityProvision, 0, len(batch))
	for _, e := range batch {
		switch et := e.(type) {
		case *events.LiquidityProvision:
			ret = append(ret, *et)
		case events.LiquidityProvision:
			ret = append(ret, et)
		}
	}
	return ret
}

func (b *brokerStub) GetTradeEvents() []events.Trade {
	batch := b.GetBatch(events.TradeEvent)
	if len(batch) == 0 {
		return nil
	}
	ret := make([]events.Trade, 0, len(batch))
	for _, e := range batch {
		switch et := e.(type) {
		case *events.Trade:
			ret = append(ret, *et)
		case events.Trade:
			ret = append(ret, et)
		}
	}
	return ret
}

func (b *brokerStub) GetAccounts() []events.Acc {
	batch := b.GetBatch(events.AccountEvent)
	if len(batch) == 0 {
		return nil
	}
	ret := make(map[string]events.Acc, len(batch))
	for _, e := range batch {
		switch et := e.(type) {
		case *events.Acc:
			ret[et.Account().Id] = *et
		case events.Acc:
			ret[et.Account().Id] = et
		}
	}
	s := make([]events.Acc, 0, len(ret))
	for _, e := range ret {
		s = append(s, e)
	}
	return s
}

func (b *brokerStub) getMarginByPartyAndMarket(partyID, marketID string) (types.MarginLevels, error) {
	batch := b.GetBatch(events.MarginLevelsEvent)
	mapped := map[string]map[string]types.MarginLevels{}
	for _, e := range batch {
		switch et := e.(type) {
		case *events.MarginLevels:
			ml := et.MarginLevels()
			if _, ok := mapped[ml.PartyId]; !ok {
				mapped[ml.PartyId] = map[string]types.MarginLevels{}
			}
			mapped[ml.PartyId][ml.MarketId] = ml
		case events.MarginLevels:
			ml := et.MarginLevels()
			if _, ok := mapped[ml.PartyId]; !ok {
				mapped[ml.PartyId] = map[string]types.MarginLevels{}
			}
			mapped[ml.PartyId][ml.MarketId] = ml
		}
	}
	mkts, ok := mapped[partyID]
	if !ok {
		return types.MarginLevels{}, fmt.Errorf("no margin levels for party (%v)", partyID)
	}
	ml, ok := mkts[marketID]
	if !ok {
		return types.MarginLevels{}, fmt.Errorf("party (%v) have no margin levels for market (%v)", partyID, marketID)
	}
	return ml, nil
}

func (b *brokerStub) getMarketInsurancePoolAccount(market string) (types.Account, error) {
	batch := b.GetAccounts()
	for _, e := range batch {
		v := e.Account()
		if v.Owner == "*" && v.MarketId == market && v.Type == types.AccountType_ACCOUNT_TYPE_INSURANCE {
			return v, nil
		}
	}
	return types.Account{}, errors.New("account does not exist")
}

func (b *brokerStub) getTraderMarginAccount(trader, market string) (types.Account, error) {
	batch := b.GetAccounts()
	for _, e := range batch {
		v := e.Account()
		if v.Owner == trader && v.Type == types.AccountType_ACCOUNT_TYPE_MARGIN && v.MarketId == market {
			return v, nil
		}
	}
	return types.Account{}, errors.New("account does not exist")
}

func (b *brokerStub) getMarketSettlementAccount(market string) (types.Account, error) {
	batch := b.GetAccounts()
	for _, e := range batch {
		v := e.Account()
		if v.Owner == "*" && v.MarketId == market && v.Type == types.AccountType_ACCOUNT_TYPE_SETTLEMENT {
			return v, nil
		}
	}
	return types.Account{}, errors.New("account does not exist")
}

// returns the latest event WRT the trader's general account
func (b *brokerStub) getTraderGeneralAccount(trader, asset string) (ga types.Account, err error) {
	batch := b.GetAccounts()
	err = errors.New("account does not exist")
	for _, e := range batch {
		v := e.Account()
		if v.Owner == trader && v.Type == types.AccountType_ACCOUNT_TYPE_GENERAL && v.Asset == asset {
			ga = v
			err = nil
		}
	}

	return
}

func (b *brokerStub) clearOrderByReference(party, ref string) error {
	b.mu.Lock()
	data := b.data[events.OrderEvent]
	cleared := make([]events.Event, 0, cap(data))
	for _, evt := range data {
		var o *types.Order
		switch e := evt.(type) {
		case *events.Order:
			o = e.Order()
		case events.Order:
			o = e.Order()
		default:
			return errors.New("non-order event ended up in order event group")
		}
		if o.Reference != ref || o.PartyId != party {
			cleared = append(cleared, evt)
		}
	}
	b.data[events.OrderEvent] = cleared
	b.mu.Unlock()
	return nil
}

func (b *brokerStub) getFirstByReference(party, ref string) (types.Order, error) {
	data := b.GetOrderEvents()
	for _, o := range data {
		v := o.Order()
		if v.Reference == ref && v.PartyId == party {
			return *v, nil
		}
	}
	return types.Order{}, fmt.Errorf("no order for party %v and referrence %v", party, ref)
}

func (b *brokerStub) getByReference(party, ref string) (types.Order, error) {
	data := b.GetOrderEvents()

	var last types.Order // we need the most recent event, the order object is not updated (copy v pointer, issue 2353)
	var matched bool = false
	for _, o := range data {
		v := o.Order()
		if v.Reference == ref && v.PartyId == party {
			last = *v
			matched = true
		}
	}
	if matched == true {
		return last, nil
	}
	return types.Order{}, fmt.Errorf("no order for party %v and referrence %v", party, ref)
}

func (b *brokerStub) getTrades() []types.Trade {
	data := b.GetTradeEvents()
	trades := make([]types.Trade, 0, len(data))
	for _, t := range data {
		trades = append(trades, t.Trade())
	}
	return trades
}

func (b *brokerStub) ResetType(t events.Type) {
	b.mu.Lock()
	b.data[t] = []events.Event{}
	b.mu.Unlock()
}

func (b *brokerStub) Reset() {
	b.mu.Lock()
	b.data = map[events.Type][]events.Event{}
	b.mu.Unlock()
}

type timeStub struct {
	now    time.Time
	notify func(context.Context, time.Time)
}

func (t *timeStub) GetTimeNow() (time.Time, error) {
	return t.now, nil
}

func (t *timeStub) SetTime(newNow time.Time) {
	t.now = newNow
	t.notify(context.Background(), t.now)
}

func (t *timeStub) NotifyOnTick(f func(context.Context, time.Time)) {
	t.notify = f
}

type ProposalStub struct {
	data []types.Proposal
}

func NewProposalStub() *ProposalStub {
	return &ProposalStub{
		data: []types.Proposal{},
	}
}

func (p *ProposalStub) Add(v types.Proposal) {
	p.data = append(p.data, v)
}

func (p *ProposalStub) Flush() {}

type VoteStub struct {
	data []types.Vote
}

func NewVoteStub() *VoteStub {
	return &VoteStub{
		data: []types.Vote{},
	}
}

func (v *VoteStub) Add(vote types.Vote) {
	v.data = append(v.data, vote)
}

func (v *VoteStub) Flush() {}
