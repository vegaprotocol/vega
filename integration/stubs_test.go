package core_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"code.vegaprotocol.io/vega/broker"
	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/proto"
)

type brokerStub struct {
	mu   sync.Mutex
	err  error
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

func (b *brokerStub) Send(e events.Event) {
	b.mu.Lock()
	t := e.Type()
	if subs, ok := b.subT[t]; ok {
		go func() {
			for _, sub := range subs {
				if sub.Ack() {
					sub.Push(e)
				} else {
					select {
					case <-sub.Closed():
						continue
					case <-sub.Skip():
						continue
					case sub.C() <- e:
						continue
					}
				}
			}
		}()
	}
	if _, ok := b.data[t]; !ok {
		b.data[e.Type()] = []events.Event{}
	}
	b.data[e.Type()] = append(b.data[e.Type()], e)
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

func (b *brokerStub) getMarginByPartyAndMarket(partyID, marketID string) (proto.MarginLevels, error) {
	batch := b.GetBatch(events.MarginLevelsEvent)
	mapped := map[string]map[string]proto.MarginLevels{}
	for _, e := range batch {
		switch et := e.(type) {
		case *events.MarginLevels:
			ml := et.MarginLevels()
			if _, ok := mapped[ml.PartyID]; !ok {
				mapped[ml.PartyID] = map[string]proto.MarginLevels{}
			}
			mapped[ml.PartyID][ml.MarketID] = ml
		case events.MarginLevels:
			ml := et.MarginLevels()
			if _, ok := mapped[ml.PartyID]; !ok {
				mapped[ml.PartyID] = map[string]proto.MarginLevels{}
			}
			mapped[ml.PartyID][ml.MarketID] = ml
		}
	}
	mkts, ok := mapped[partyID]
	if !ok {
		return proto.MarginLevels{}, fmt.Errorf("no margin levels for party (%v)", partyID)
	}
	ml, ok := mkts[marketID]
	if !ok {
		return proto.MarginLevels{}, fmt.Errorf("party (%v) have no margin levels for market (%v)", partyID, marketID)
	}
	return ml, nil
}

func (b *brokerStub) getMarketInsurancePoolAccount(market string) (proto.Account, error) {
	batch := b.GetAccounts()
	for _, e := range batch {
		v := e.Account()
		if v.Owner == "*" && v.MarketID == market && v.Type == proto.AccountType_ACCOUNT_TYPE_INSURANCE {
			return v, nil
		}
	}
	return proto.Account{}, errors.New("account does not exist")
}

func (b *brokerStub) getTraderMarginAccount(trader, market string) (proto.Account, error) {
	batch := b.GetAccounts()
	for _, e := range batch {
		v := e.Account()
		if v.Owner == trader && v.Type == proto.AccountType_ACCOUNT_TYPE_MARGIN && v.MarketID == market {
			return v, nil
		}
	}
	return proto.Account{}, errors.New("account does not exist")
}

func (b *brokerStub) getMarketSettlementAccount(market string) (proto.Account, error) {
	batch := b.GetAccounts()
	for _, e := range batch {
		v := e.Account()
		if v.Owner == "*" && v.MarketID == market && v.Type == proto.AccountType_ACCOUNT_TYPE_SETTLEMENT {
			return v, nil
		}
	}
	return proto.Account{}, errors.New("account does not exist")
}

func (b *brokerStub) getTraderGeneralAccount(trader, asset string) (proto.Account, error) {
	batch := b.GetAccounts()
	for _, e := range batch {
		v := e.Account()
		if v.Owner == trader && v.Type == proto.AccountType_ACCOUNT_TYPE_GENERAL && v.Asset == asset {
			return v, nil
		}
	}

	return proto.Account{}, errors.New("account does not exist")
}

func (b *brokerStub) getByReference(party, ref string) (proto.Order, error) {
	data := b.GetOrderEvents()
	for _, o := range data {
		v := o.Order()
		if v.Reference == ref && v.PartyID == party {
			return *v, nil
		}
	}
	return proto.Order{}, fmt.Errorf("no order for party %v and referrence %v", party, ref)
}

func (b *brokerStub) getTrades() []proto.Trade {
	data := b.GetTradeEvents()
	trades := make([]proto.Trade, 0, len(data))
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

type marginsStub struct {
	data map[string]map[string]proto.MarginLevels
	mu   sync.Mutex
	err  error // for future use
}

func NewMarginsStub() *marginsStub {
	return &marginsStub{
		data: map[string]map[string]proto.MarginLevels{},
	}
}

func (m *marginsStub) Add(ml proto.MarginLevels) {
	m.mu.Lock()
	if _, ok := m.data[ml.PartyID]; !ok {
		m.data[ml.PartyID] = map[string]proto.MarginLevels{}
	}
	m.data[ml.PartyID][ml.MarketID] = ml
	m.mu.Unlock()
}

func (m *marginsStub) getMarginByPartyAndMarket(partyID, marketID string) (proto.MarginLevels, error) {
	mkts, ok := m.data[partyID]
	if !ok {
		return proto.MarginLevels{}, fmt.Errorf("no margin levels for party (%v)", partyID)
	}
	ml, ok := mkts[marketID]
	if !ok {
		return proto.MarginLevels{}, fmt.Errorf("party (%v) have no margin levels for market (%v)", partyID, marketID)
	}
	return ml, nil
}

func (m *marginsStub) Flush() {}

type accStub struct {
	data map[string]proto.Account
	mu   *sync.Mutex
}

func NewAccountStub() *accStub {
	return &accStub{
		data: map[string]proto.Account{},
		mu:   &sync.Mutex{},
	}
}

func (d *accStub) Add(acc proto.Account) {
	d.mu.Lock()
	d.data[acc.Id] = acc
	d.mu.Unlock()
}

func (s *accStub) getTraderMarginAccount(trader, market string) (proto.Account, error) {
	for _, v := range s.data {
		if v.Owner == trader && v.Type == proto.AccountType_ACCOUNT_TYPE_MARGIN && v.MarketID == market {
			return v, nil
		}
	}
	return proto.Account{}, errors.New("account does not exist")
}

func (s *accStub) getMarketSettlementAccount(market string) (proto.Account, error) {
	for _, v := range s.data {
		if v.Owner == "*" && v.MarketID == market && v.Type == proto.AccountType_ACCOUNT_TYPE_SETTLEMENT {
			return v, nil
		}
	}
	return proto.Account{}, errors.New("account does not exist")
}

func (s *accStub) getMarketInsurancePoolAccount(market string) (proto.Account, error) {
	for _, v := range s.data {
		if v.Owner == "*" && v.MarketID == market && v.Type == proto.AccountType_ACCOUNT_TYPE_INSURANCE {
			return v, nil
		}
	}
	return proto.Account{}, errors.New("account does not exist")
}

func (s *accStub) getTraderGeneralAccount(trader, asset string) (proto.Account, error) {
	for _, v := range s.data {
		if v.Owner == trader && v.Type == proto.AccountType_ACCOUNT_TYPE_GENERAL && v.Asset == asset {
			return v, nil
		}
	}

	return proto.Account{}, errors.New("account does not exist")
}

func (d *accStub) Get(id string) *proto.Account {
	var ret *proto.Account
	d.mu.Lock()
	if acc, ok := d.data[id]; ok {
		ret = &acc
	}
	d.mu.Unlock()
	return ret
}

func (a *accStub) Flush() error {
	return nil
}

type orderStub struct {
	data map[string]proto.Order
	mu   *sync.Mutex
	err  error
}

func (o *orderStub) getByReference(party, ref string) (proto.Order, error) {
	for _, v := range o.data {
		if v.Reference == ref && v.PartyID == party {
			return v, nil
		}
	}
	return proto.Order{}, fmt.Errorf("no order for party %v and referrence %v", party, ref)
}

func NewOrderStub() *orderStub {
	return &orderStub{
		data: map[string]proto.Order{},
		mu:   &sync.Mutex{},
	}
}

func (o *orderStub) Add(order proto.Order) {
	o.mu.Lock()
	o.data[order.Id] = order
	o.mu.Unlock()
}

func (o *orderStub) Flush() error {
	o.mu.Lock()
	err := o.err
	o.mu.Unlock()
	return err
}

// GetByPartyAndID is only used in the execution engine, we're not integrating with that component
// this stub is used on the market integration level
func (o *orderStub) GetByPartyAndID(_ context.Context, party, id string) (*proto.Order, error) {
	var ret *proto.Order
	o.mu.Lock()
	order, ok := o.data[id]
	err := o.err
	o.mu.Unlock()
	if ok && order.PartyID == party {
		ret = &order // should be a pointer to local copy from map already
	}
	return ret, err
}

func (o *orderStub) GetByReference(ref string) (*proto.Order, error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	for _, v := range o.data {
		if v.Reference == ref {
			cpy := v
			return &cpy, nil
		}
	}
	return nil, fmt.Errorf("reference %s not found", ref)
}

func (o *orderStub) Get(id string) *proto.Order {
	var ret *proto.Order
	o.mu.Lock()
	if order, ok := o.data[id]; ok {
		ret = &order
	}
	o.mu.Unlock()
	return ret
}

type transferStub struct {
	data []*proto.TransferResponse
	mu   *sync.Mutex
}

func NewTransferStub() *transferStub {
	return &transferStub{
		data: []*proto.TransferResponse{},
		mu:   &sync.Mutex{},
	}
}

func (t *transferStub) Flush() error {
	t.data = []*proto.TransferResponse{}
	return nil
}

func (t *transferStub) Add(b []*proto.TransferResponse) {
	t.mu.Lock()
	t.data = append(t.data, b...)
	t.mu.Unlock()
}

func (t *transferStub) GetBatch() []*proto.TransferResponse {
	t.mu.Lock()
	b := t.data
	t.mu.Unlock()
	return b
}

func (t *transferStub) Reset() {
	t.mu.Lock()
	t.data = []*proto.TransferResponse{}
	t.mu.Unlock()
}

type tradeStub struct {
	data map[string]proto.Trade
	err  error
	mu   *sync.Mutex
}

func NewTradeStub() *tradeStub {
	return &tradeStub{
		data: map[string]proto.Trade{},
		mu:   &sync.Mutex{},
	}
}

func (t *tradeStub) Flush() error {
	t.mu.Lock()
	err := t.err
	t.data = map[string]proto.Trade{}
	t.mu.Unlock()
	return err
}

func (t *tradeStub) Add(v proto.Trade) {
	t.mu.Lock()
	t.data[v.Id] = v
	t.mu.Unlock()
}

func (t *tradeStub) Get(id string) proto.Trade {
	t.mu.Lock()
	v := t.data[id]
	t.mu.Unlock()
	return v
}

type timeStub struct {
	now    time.Time
	notify func(time.Time)
}

func (t *timeStub) GetTimeNow() (time.Time, error) {
	return t.now, nil
}

func (t *timeStub) SetTime(newNow time.Time) {
	t.now = newNow
	t.notify(t.now)
}

func (t *timeStub) NotifyOnTick(f func(time.Time)) {
	t.notify = f
}

type SettleStub struct {
	data []events.SettlePosition
}

func NewSettlementStub() *SettleStub {
	return &SettleStub{
		data: []events.SettlePosition{},
	}
}

func (p *SettleStub) Add(e []events.SettlePosition) {
	p.data = append(p.data, e...)
}

func (p *SettleStub) Flush() {}

type ProposalStub struct {
	data []proto.Proposal
}

func NewProposalStub() *ProposalStub {
	return &ProposalStub{
		data: []proto.Proposal{},
	}
}

func (p *ProposalStub) Add(v proto.Proposal) {
	p.data = append(p.data, v)
}

func (p *ProposalStub) Flush() {}

type VoteStub struct {
	data []proto.Vote
}

func NewVoteStub() *VoteStub {
	return &VoteStub{
		data: []proto.Vote{},
	}
}

func (v *VoteStub) Add(vote proto.Vote) {
	v.data = append(v.data, vote)
}

func (v *VoteStub) Flush() {}
