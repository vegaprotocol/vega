package core_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/proto"
)

type marginsStub struct {
	data map[string]map[string]proto.MarginLevels
	mu   sync.Mutex
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
		if v.Owner == trader && v.Type == proto.AccountType_MARGIN && v.MarketID == market {
			return v, nil
		}
	}
	return proto.Account{}, errors.New("account does not exist")
}

func (s *accStub) getMarketSettlementAccount(market string) (proto.Account, error) {
	for _, v := range s.data {
		if v.Owner == "*" && v.MarketID == market && v.Type == proto.AccountType_SETTLEMENT {
			return v, nil
		}
	}
	return proto.Account{}, errors.New("account does not exist")
}

func (s *accStub) getMarketInsurancePoolAccount(market string) (proto.Account, error) {
	for _, v := range s.data {
		if v.Owner == "*" && v.MarketID == market && v.Type == proto.AccountType_INSURANCE {
			return v, nil
		}
	}
	return proto.Account{}, errors.New("account does not exist")
}

func (s *accStub) getTraderGeneralAccount(trader, asset string) (proto.Account, error) {
	for _, v := range s.data {
		if v.Owner == trader && v.Type == proto.AccountType_GENERAL && v.Asset == asset {
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
	err  error // still not conviced about this one
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
