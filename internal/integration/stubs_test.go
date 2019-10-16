package core_test

import (
	"context"
	"sync"

	"code.vegaprotocol.io/vega/proto"
)

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

func (d *accStub) Get(id string) *proto.Account {
	var ret *proto.Account
	d.mu.Lock()
	if acc, ok := d.data[id]; ok {
		ret = &acc
	}
	d.mu.Unlock()
	return ret
}

type orderStub struct {
	data map[string]proto.Order
	mu   *sync.Mutex
	// not sure if we want to run integration tests against hypothetical failures...
	err error
}

func NewOrderStub() *orderStub {
	return &orderStub{
		data: map[string]proto.Order{},
		mu:   &sync.Mutex{},
	}
}

func (o *orderStub) Post(order proto.Order) error {
	o.mu.Lock()
	o.data[order.Id] = order
	err := o.err
	o.mu.Unlock()
	return err
}

func (o *orderStub) Put(order proto.Order) error {
	return o.Post(order)
}

func (o *orderStub) Commit() error {
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

func (t *transferStub) SaveBatch(b []*proto.TransferResponse) error {
	t.mu.Lock()
	t.data = append(t.data, b...)
	err := t.err
	t.mu.Unlock()
	return err
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
	data map[string]*proto.Trade
	err  error
	mu   *sync.Mutex
}

func NewTradeStub() *tradeStub {
	return &tradeStub{
		data: map[string]*proto.Trade{},
		mu:   &sync.Mutex{},
	}
}

func (t *tradeStub) Commit() error {
	t.mu.Lock()
	err := t.err
	t.data = map[string]*proto.Trade{}
	t.mu.Unlock()
	return err
}

func (t *tradeStub) Post(v *proto.Trade) error {
	t.mu.Lock()
	err := t.err
	t.data[v.Id] = v
	t.mu.Unlock()
	return err
}

func (t *tradeStub) Get(id string) *proto.Trade {
	t.mu.Lock()
	v := t.data[id]
	t.mu.Unlock()
	return v
}
