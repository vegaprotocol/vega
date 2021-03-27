package stubs

import (
	"errors"
	"fmt"
	"sync"

	"code.vegaprotocol.io/vega/broker"
	"code.vegaprotocol.io/vega/events"
	types "code.vegaprotocol.io/vega/proto"
)

type BrokerStub struct {
	mu   sync.Mutex
	data map[events.Type][]events.Event
	subT map[events.Type][]broker.Subscriber

	immdata map[events.Type][]events.Event
}

func NewBrokerStub() *BrokerStub {
	return &BrokerStub{
		data:    map[events.Type][]events.Event{},
		immdata: map[events.Type][]events.Event{},
		subT:    map[events.Type][]broker.Subscriber{},
	}
}

func (b *BrokerStub) Subscribe(sub broker.Subscriber) {
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

func (b *BrokerStub) SendBatch(evts []events.Event) {
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
	if _, ok := b.immdata[t]; !ok {
		b.immdata[t] = []events.Event{}
	}
	b.data[t] = append(b.data[t], evts...)
	b.immdata[t] = append(b.immdata[t], evts...)
	b.mu.Unlock()
}

func (b *BrokerStub) Send(e events.Event) {
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
	if _, ok := b.immdata[t]; !ok {
		b.immdata[t] = []events.Event{}
	}
	b.data[t] = append(b.data[t], e)
	b.immdata[t] = append(b.immdata[t], e)
	b.mu.Unlock()
}

func (b *BrokerStub) GetBatch(t events.Type) []events.Event {
	b.mu.Lock()
	r := b.data[t]
	b.mu.Unlock()
	return r
}

func (b *BrokerStub) GetRejectedOrderAmendments() []events.TxErr {
	return b.filterTxErr(func(errProto types.TxErrorEvent) bool {
		return errProto.GetOrderAmendment() != nil
	})
}

func (b *BrokerStub) GetImmBatch(t events.Type) []events.Event {
	b.mu.Lock()
	r := b.immdata[t]
	b.mu.Unlock()
	return r
}

func (b *BrokerStub) GetTransferResponses() []events.TransferResponse {
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

func (b *BrokerStub) ClearTransferEvents() {
	t := events.TransferResponses
	b.mu.Lock()
	r := b.data[t]
	b.data[t] = make([]events.Event, 0, cap(r))
	b.mu.Unlock()
}

func (b *BrokerStub) ClearOrderEvents() {
	t := events.OrderEvent
	b.mu.Lock()
	r := b.data[t]
	// reallocate new slice
	b.data[t] = make([]events.Event, 0, cap(r))
	b.mu.Unlock()
}

func (b *BrokerStub) GetBookDepth(market string) (sell map[uint64]uint64, buy map[uint64]uint64) {
	batch := b.GetImmBatch(events.OrderEvent)
	if len(batch) == 0 {
		return nil, nil
	}

	// first get all active orders
	activeOrders := map[string]*types.Order{}
	for _, e := range batch {
		var ord *types.Order
		switch et := e.(type) {
		case *events.Order:
			ord = et.Order()
		case events.Order:
			ord = et.Order()
		}

		if ord.MarketId != market {
			continue
		}

		if ord.Status == types.Order_STATUS_ACTIVE {
			activeOrders[ord.Id] = ord
		} else {
			delete(activeOrders, ord.Id)
		}
	}

	// now we haveall active orders, let's build both sides
	sell, buy = map[uint64]uint64{}, map[uint64]uint64{}
	for _, v := range activeOrders {
		if v.Side == types.Side_SIDE_BUY {
			buy[v.Price] = buy[v.Price] + v.Remaining
			continue
		}
		sell[v.Price] = sell[v.Price] + v.Remaining
	}

	return
}

func (b *BrokerStub) GetOrdersByPartyAndMarket(party, market string) []types.Order {
	orders := b.GetOrderEvents()
	ret := []types.Order{}
	for _, oe := range orders {
		if o := oe.Order(); o.MarketId == market && o.PartyId == party {
			ret = append(ret, *o)
		}
	}
	return ret
}

func (b *BrokerStub) GetOrderEvents() []events.Order {
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

func (b *BrokerStub) GetLPEvents() []events.LiquidityProvision {
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

func (b *BrokerStub) GetTradeEvents() []events.Trade {
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

func (b *BrokerStub) GetAccounts() []events.Acc {
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

func (b *BrokerStub) GetMarginByPartyAndMarket(partyID, marketID string) (types.MarginLevels, error) {
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

func (b *BrokerStub) GetMarketInsurancePoolAccount(market string) (types.Account, error) {
	batch := b.GetAccounts()
	for _, e := range batch {
		v := e.Account()
		if v.Owner == "*" && v.MarketId == market && v.Type == types.AccountType_ACCOUNT_TYPE_INSURANCE {
			return v, nil
		}
	}
	return types.Account{}, errors.New("account does not exist")
}

func (b *BrokerStub) GetTraderMarginAccount(trader, market string) (types.Account, error) {
	batch := b.GetAccounts()
	for _, e := range batch {
		v := e.Account()
		if v.Owner == trader && v.Type == types.AccountType_ACCOUNT_TYPE_MARGIN && v.MarketId == market {
			return v, nil
		}
	}
	return types.Account{}, errors.New("account does not exist")
}

func (b *BrokerStub) GetMarketSettlementAccount(market string) (types.Account, error) {
	batch := b.GetAccounts()
	for _, e := range batch {
		v := e.Account()
		if v.Owner == "*" && v.MarketId == market && v.Type == types.AccountType_ACCOUNT_TYPE_SETTLEMENT {
			return v, nil
		}
	}
	return types.Account{}, errors.New("account does not exist")
}

// GetTraderGeneralAccount returns the latest event WRT the trader's general account
func (b *BrokerStub) GetTraderGeneralAccount(trader, asset string) (ga types.Account, err error) {
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

func (b *BrokerStub) ClearOrderByReference(party, ref string) error {
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

func (b *BrokerStub) GetFirstByReference(party, ref string) (types.Order, error) {
	data := b.GetOrderEvents()
	for _, o := range data {
		v := o.Order()
		if v.Reference == ref && v.PartyId == party {
			return *v, nil
		}
	}
	return types.Order{}, fmt.Errorf("no order for party %v and reference %v", party, ref)
}

func (b *BrokerStub) GetByReference(party, ref string) (types.Order, error) {
	data := b.GetOrderEvents()

	var last types.Order // we need the most recent event, the order object is not updated (copy v pointer, issue 2353)
	var matched = false
	for _, o := range data {
		v := o.Order()
		if v.Reference == ref && v.PartyId == party {
			last = *v
			matched = true
		}
	}
	if matched {
		return last, nil
	}
	return types.Order{}, fmt.Errorf("no order for party %v and reference %v", party, ref)
}

func (b *BrokerStub) GetTrades() []types.Trade {
	data := b.GetTradeEvents()
	trades := make([]types.Trade, 0, len(data))
	for _, t := range data {
		trades = append(trades, t.Trade())
	}
	return trades
}

func (b *BrokerStub) ResetType(t events.Type) {
	b.mu.Lock()
	b.data[t] = []events.Event{}
	b.mu.Unlock()
}

func (b *BrokerStub) filterTxErr(predicate func(errProto types.TxErrorEvent) bool) []events.TxErr {
	batch := b.GetBatch(events.TxErrEvent)
	if len(batch) == 0 {
		return nil
	}

	errs := []events.TxErr{}
	b.mu.Lock()
	for _, e := range batch {
		err := derefTxErr(e)
		errProto := err.Proto()
		if predicate(errProto) {
			errs = append(errs, err)
		}
	}
	b.mu.Unlock()
	return errs
}

func derefTxErr(e events.Event) events.TxErr {
	var dub events.TxErr
	switch et := e.(type) {
	case *events.TxErr:
		dub = *et
	case events.TxErr:
		dub = et
	}
	return dub
}
