// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package stubs

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/broker"
	"code.vegaprotocol.io/vega/libs/ptr"
	proto "code.vegaprotocol.io/vega/protos/vega"
	vegapb "code.vegaprotocol.io/vega/protos/vega"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"
)

var AccountDoesNotExistErr = errors.New("account does not exist")

type AssetParty struct {
	Asset, Party string
}

type BrokerStub struct {
	mu   sync.Mutex
	data map[events.Type][]events.Event
	subT map[events.Type][]broker.Subscriber

	immdata      map[events.Type][]events.Event
	immdataSlice []events.Event
}

func NewBrokerStub() *BrokerStub {
	return &BrokerStub{
		data:    map[events.Type][]events.Event{},
		immdata: map[events.Type][]events.Event{},
		subT:    map[events.Type][]broker.Subscriber{},
	}
}

func (b *BrokerStub) Subscribe(sub broker.Subscriber) int {
	b.mu.Lock()
	ty := sub.Types()
	for _, t := range ty {
		if _, ok := b.subT[t]; !ok {
			b.subT[t] = []broker.Subscriber{}
		}
		b.subT[t] = append(b.subT[t], sub)
	}
	b.mu.Unlock()
	return 0
}

func (b *BrokerStub) SetStreaming(v bool) bool { return false }

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
	b.immdataSlice = append(b.immdataSlice, e)
	b.mu.Unlock()
}

func (b *BrokerStub) Stage(e events.Event) {}

func (b *BrokerStub) GetAllEventsSinceCleared() []events.Event {
	b.mu.Lock()
	evs := []events.Event{}
	for _, d := range b.data {
		evs = append(evs, d...)
	}
	b.mu.Unlock()
	return evs
}

func (b *BrokerStub) GetAllEvents() []events.Event {
	b.mu.Lock()
	ret := make([]events.Event, len(b.immdataSlice))
	copy(ret, b.immdataSlice)
	b.mu.Unlock()
	return ret
}

func (b *BrokerStub) GetBatch(t events.Type) []events.Event {
	b.mu.Lock()
	r := b.data[t]
	b.mu.Unlock()
	return r
}

func (b *BrokerStub) GetRejectedOrderAmendments() []events.TxErr {
	return b.filterTxErr(func(errProto eventspb.TxErrorEvent) bool {
		return errProto.GetOrderAmendment() != nil
	})
}

func (b *BrokerStub) GetImmBatch(t events.Type) []events.Event {
	b.mu.Lock()
	r := b.immdata[t]
	b.mu.Unlock()
	return r
}

// GetLedgerMovements returns ledger movements, `mutable` argument specifies if these should be all the scenario events or events that can be cleared by the user.
func (b *BrokerStub) GetLedgerMovements(mutable bool) []events.LedgerMovements {
	batch := b.GetBatch(events.LedgerMovementsEvent)
	if !mutable {
		batch = b.GetImmBatch((events.LedgerMovementsEvent))
	}
	if len(batch) == 0 {
		return nil
	}
	b.mu.Lock()
	ret := make([]events.LedgerMovements, 0, len(batch))
	for _, e := range batch {
		switch et := e.(type) {
		case *events.LedgerMovements:
			ret = append(ret, *et)
		}
	}
	b.mu.Unlock()
	return ret
}

func (b *BrokerStub) GetDistressedOrders() []events.DistressedOrders {
	batch := b.GetImmBatch(events.DistressedOrdersClosedEvent)
	ret := make([]events.DistressedOrders, 0, len(batch))
	for _, e := range batch {
		switch et := e.(type) {
		case *events.DistressedOrders:
			ret = append(ret, *et)
		case events.DistressedOrders:
			ret = append(ret, et)
		}
	}
	return ret
}

func (b *BrokerStub) GetSettleDistressed() []events.SettleDistressed {
	batch := b.GetImmBatch(events.SettleDistressedEvent)
	ret := make([]events.SettleDistressed, 0, len(batch))
	for _, e := range batch {
		switch et := e.(type) {
		case *events.SettleDistressed:
			ret = append(ret, *et)
		case events.SettleDistressed:
			ret = append(ret, et)
		}
	}
	return ret
}

func (b *BrokerStub) GetLossSocializationEvents() []events.LossSocialization {
	evts := b.GetImmBatch(events.LossSocializationEvent)
	typed := make([]events.LossSocialization, 0, len(evts))
	for _, e := range evts {
		if le, ok := e.(events.LossSocialization); ok {
			typed = append(typed, le)
		}
	}
	return typed
}

func (b *BrokerStub) GetLossSoc() []*events.LossSoc {
	evts := b.GetImmBatch(events.LossSocializationEvent)
	typed := make([]*events.LossSoc, 0, len(evts))
	for _, e := range evts {
		switch le := e.(type) {
		case events.LossSoc:
			cpy := le
			typed = append(typed, &cpy)
		case *events.LossSoc:
			typed = append(typed, le)
		}
	}
	return typed
}

func (b *BrokerStub) ClearAllEvents() {
	b.mu.Lock()
	b.data = map[events.Type][]events.Event{}
	b.mu.Unlock()
}

func (b *BrokerStub) ClearTransferResponseEvents() {
	b.mu.Lock()
	cs := make([]events.Event, 0, len(b.data[events.LedgerMovementsEvent]))
	b.data[events.LedgerMovementsEvent] = cs
	b.mu.Unlock()
}

func (b *BrokerStub) ClearTradeEvents() {
	b.mu.Lock()
	te := make([]events.Event, 0, len(b.data[events.TradeEvent]))
	b.data[events.TradeEvent] = te
	b.mu.Unlock()
}

// GetTransfers returns ledger entries, mutable argument specifies if these should be all the scenario events or events that can be cleared by the user.
func (b *BrokerStub) GetTransfers(mutable bool) []*vegapb.LedgerEntry {
	transferEvents := b.GetLedgerMovements(mutable)
	transfers := []*vegapb.LedgerEntry{}
	for _, e := range transferEvents {
		for _, response := range e.LedgerMovements() {
			transfers = append(transfers, response.GetEntries()...)
		}
	}
	return transfers
}

func (b *BrokerStub) GetBookDepth(market string) (sell map[string]uint64, buy map[string]uint64) {
	batch := b.GetImmBatch(events.OrderEvent)
	exp := b.GetImmBatch(events.ExpiredOrdersEvent)
	if len(batch) == 0 {
		return nil, nil
	}
	expForMarket := map[string]struct{}{}
	for _, e := range exp {
		switch et := e.(type) {
		case *events.ExpiredOrders:
			if !et.IsMarket(market) {
				continue
			}
			for _, oid := range et.OrderIDs() {
				expForMarket[oid] = struct{}{}
			}
		case events.ExpiredOrders:
			if !et.IsMarket(market) {
				continue
			}
			for _, oid := range et.OrderIDs() {
				expForMarket[oid] = struct{}{}
			}
		}
	}

	// first get all active orders
	activeOrders := map[string]*vegapb.Order{}
	for _, e := range batch {
		var ord *vegapb.Order
		switch et := e.(type) {
		case *events.Order:
			ord = et.Order()
		case events.Order:
			ord = et.Order()
		default:
			continue
		}

		if ord.MarketId != market {
			continue
		}

		if ord.Status == vegapb.Order_STATUS_ACTIVE {
			activeOrders[ord.Id] = ord
		} else {
			delete(activeOrders, ord.Id)
		}
	}

	// now we have all active orders, let's build both sides
	sell, buy = map[string]uint64{}, map[string]uint64{}
	for id, v := range activeOrders {
		if _, ok := expForMarket[id]; ok {
			continue
		}
		if v.Side == vegapb.Side_SIDE_BUY {
			buy[v.Price] = buy[v.Price] + v.Remaining
			continue
		}
		sell[v.Price] = sell[v.Price] + v.Remaining
	}

	return sell, buy
}

func (b *BrokerStub) GetActiveOrderDepth(marketID string) (sell []*vegapb.Order, buy []*vegapb.Order) {
	batch := b.GetImmBatch(events.OrderEvent)
	if len(batch) == 0 {
		return nil, nil
	}
	active := make(map[string]*vegapb.Order, len(batch))
	for _, e := range batch {
		var ord *vegapb.Order
		switch et := e.(type) {
		case *events.Order:
			ord = et.Order()
		case events.Order:
			ord = et.Order()
		default:
			continue
		}
		if ord.MarketId != marketID {
			continue
		}
		if ord.Status == vegapb.Order_STATUS_ACTIVE {
			active[ord.Id] = ord
		} else {
			delete(active, ord.Id)
		}
	}
	c := len(active) / 2
	if len(active)%2 == 1 {
		c++
	}
	sell, buy = make([]*vegapb.Order, 0, c), make([]*vegapb.Order, 0, c)
	for _, ord := range active {
		if ord.Side == vegapb.Side_SIDE_BUY {
			buy = append(buy, ord)
			continue
		}
		sell = append(sell, ord)
	}

	return sell, buy
}

func (b *BrokerStub) GetMarket(marketID string) *vegapb.Market {
	batch := b.GetBatch(events.MarketUpdatedEvent)
	if len(batch) == 0 {
		return nil
	}

	for i := len(batch) - 1; i >= 0; i-- {
		switch mkt := batch[i].(type) {
		case *events.MarketUpdated:
			if mkt.MarketID() != marketID {
				continue
			}
			m := mkt.Proto()
			return &m
		case events.MarketUpdated:
			if mkt.MarketID() != marketID {
				continue
			}
			m := mkt.Proto()
			return &m
		}
	}

	return nil
}

func (b *BrokerStub) GetLastMarketUpdateState(marketID string) *vegapb.Market {
	batch := b.GetBatch(events.MarketUpdatedEvent)
	if len(batch) == 0 {
		return nil
	}
	var r *vegapb.Market
	for _, evt := range batch {
		switch me := evt.(type) {
		case *events.MarketUpdated:
			if me.MarketID() == marketID {
				t := me.Proto()
				r = &t
			}
		case events.MarketUpdated:
			if me.MarketID() == marketID {
				t := me.Proto()
				r = &t
			}
		}
	}
	return r
}

func (b *BrokerStub) GetMarkPriceSettings(marketID string) *vegapb.CompositePriceConfiguration {
	batch := b.GetBatch(events.MarketUpdatedEvent)
	if len(batch) == 0 {
		return nil
	}
	var r *vegapb.Market
	for _, evt := range batch {
		switch me := evt.(type) {
		case *events.MarketUpdated:
			if me.MarketID() == marketID {
				t := me.Proto()
				r = &t
			}
		case events.MarketUpdated:
			if me.MarketID() == marketID {
				t := me.Proto()
				r = &t
			}
		}
	}
	return r.MarkPriceConfiguration
}

func (b *BrokerStub) GetOrdersByPartyAndMarket(party, market string) []vegapb.Order {
	orders := b.GetOrderEvents()
	ret := []vegapb.Order{}
	for _, oe := range orders {
		if o := oe.Order(); o.MarketId == market && o.PartyId == party {
			ret = append(ret, *o)
		}
	}
	return ret
}

func (b *BrokerStub) GetStopOrderEvents() []events.StopOrder {
	batch := b.GetBatch(events.StopOrderEvent)
	if len(batch) == 0 {
		return nil
	}
	ret := make([]events.StopOrder, 0, len(batch))
	for _, e := range batch {
		switch et := e.(type) {
		case *events.StopOrder:
			// o := vtypes.NewStopOrderFromProto(et.StopOrder())
			ret = append(ret, *et)
		case events.StopOrder:
			// o := vtypes.NewStopOrderFromProto(et.StopOrder())
			ret = append(ret, et)
		}
	}

	return ret
}

func (b *BrokerStub) GetOrderEvents() []events.Order {
	batch := b.GetBatch(events.OrderEvent)
	if len(batch) == 0 {
		return nil
	}
	last := map[string]*types.Order{}
	ret := make([]events.Order, 0, len(batch))
	for _, e := range batch {
		var o *types.Order
		switch et := e.(type) {
		case *events.Order:
			o, _ = types.OrderFromProto(et.Order())
			ret = append(ret, *et)
		case events.Order:
			o, _ = types.OrderFromProto(et.Order())
			ret = append(ret, et)
		}
		last[o.ID] = o
	}
	expired := b.GetBatch(events.ExpiredOrdersEvent)
	for _, e := range expired {
		var ids []string
		switch et := e.(type) {
		case *events.ExpiredOrders:
			ids = et.OrderIDs()
		case events.ExpiredOrders:
			ids = et.OrderIDs()
		}
		for _, id := range ids {
			if o, ok := last[id]; ok {
				o.Status = vegapb.Order_STATUS_EXPIRED
				fe := events.NewOrderEvent(context.Background(), o)
				ret = append(ret, *fe)
			}
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

func (b *BrokerStub) GetAccountEvents() []events.Acc {
	// Use GetImmBatch so that clearing events doesn't affact this method
	batch := b.GetImmBatch(events.AccountEvent)
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

func (b *BrokerStub) GetDeposits() []vegapb.Deposit {
	// Use GetImmBatch so that clearing events doesn't affact this method
	batch := b.GetImmBatch(events.DepositEvent)
	if len(batch) == 0 {
		return nil
	}
	ret := make([]vegapb.Deposit, 0, len(batch))
	for _, e := range batch {
		switch et := e.(type) {
		case *events.Deposit:
			ret = append(ret, et.Deposit())
		}
	}
	return ret
}

func (b *BrokerStub) GetWithdrawals() []vegapb.Withdrawal {
	// Use GetImmBatch so that clearing events doesn't affact this method
	batch := b.GetImmBatch(events.WithdrawalEvent)
	if len(batch) == 0 {
		return nil
	}
	ret := make([]vegapb.Withdrawal, 0, len(batch))
	for _, e := range batch {
		switch et := e.(type) {
		case *events.Withdrawal:
			ret = append(ret, et.Withdrawal())
		}
	}
	return ret
}

func (b *BrokerStub) GetDelegationBalanceEvents(epochSeq string) []events.DelegationBalance {
	batch := b.GetBatch(events.DelegationBalanceEvent)
	if len(batch) == 0 {
		return nil
	}

	s := []events.DelegationBalance{}

	for _, e := range batch {
		switch et := e.(type) {
		case events.DelegationBalance:
			if et.EpochSeq == epochSeq {
				s = append(s, et)
			}
		case *events.DelegationBalance:
			if (*et).EpochSeq == epochSeq {
				s = append(s, *et)
			}
		}
	}
	return s
}

func (b *BrokerStub) GetCurrentEpoch() *events.EpochEvent {
	batch := b.GetBatch(events.EpochUpdate)
	if len(batch) == 0 {
		return nil
	}
	last := batch[len(batch)-1]
	switch et := last.(type) {
	case events.EpochEvent:
		return &et
	case *events.EpochEvent:
		return et
	}
	return nil
}

func (b *BrokerStub) GetDelegationBalance(epochSeq string) []vegapb.Delegation {
	evts := b.GetDelegationBalanceEvents(epochSeq)
	balances := make([]vegapb.Delegation, 0, len(evts))

	for _, e := range evts {
		balances = append(balances, vegapb.Delegation{
			Party:    e.Party,
			NodeId:   e.NodeID,
			EpochSeq: e.EpochSeq,
			Amount:   e.Amount.String(),
		})
	}
	return balances
}

func (b *BrokerStub) GetRewards(epochSeq string) map[AssetParty]events.RewardPayout {
	batch := b.GetBatch(events.RewardPayoutEvent)
	if len(batch) == 0 {
		return nil
	}

	rewards := map[AssetParty]events.RewardPayout{}

	for _, e := range batch {
		switch et := e.(type) {
		case events.RewardPayout:
			if et.EpochSeq == epochSeq {
				rewards[AssetParty{et.Asset, et.Party}] = et
			}
		case *events.RewardPayout:
			if (*et).EpochSeq == epochSeq {
				rewards[AssetParty{et.Asset, et.Party}] = *et
			}
		}
	}
	return rewards
}

func (b *BrokerStub) GetValidatorScores(epochSeq string) map[string]events.ValidatorScore {
	batch := b.GetBatch(events.ValidatorScoreEvent)
	if len(batch) == 0 {
		return nil
	}

	scores := map[string]events.ValidatorScore{}

	for _, e := range batch {
		switch et := e.(type) {
		case events.ValidatorScore:
			if et.EpochSeq == epochSeq && et.ValidatorStatus == "tendermint" {
				scores[et.NodeID] = et
			}
		case *events.ValidatorScore:
			if (*et).EpochSeq == epochSeq && et.ValidatorStatus == "tendermint" {
				scores[et.NodeID] = *et
			}
		}
	}
	return scores
}

func (b *BrokerStub) GetAccounts() []vegapb.Account {
	evts := b.GetAccountEvents()
	accounts := make([]vegapb.Account, 0, len(evts))
	for _, a := range evts {
		accounts = append(accounts, a.Account())
	}
	return accounts
}

func (b *BrokerStub) GetAuctionEvents() []events.Auction {
	batch := b.GetBatch(events.AuctionEvent)

	if len(batch) == 0 {
		return nil
	}
	ret := make([]events.Auction, 0, len(batch))
	for _, e := range batch {
		switch et := e.(type) {
		case *events.Auction:
			ret = append(ret, *et)
		case events.Auction:
			ret = append(ret, et)
		}
	}
	return ret
}

func (b *BrokerStub) GetMarginByPartyAndMarket(partyID, marketID string) (vegapb.MarginLevels, error) {
	batch := b.GetBatch(events.MarginLevelsEvent)
	mapped := map[string]map[string]vegapb.MarginLevels{}
	for _, e := range batch {
		switch et := e.(type) {
		case *events.MarginLevels:
			ml := et.MarginLevels()
			if _, ok := mapped[ml.PartyId]; !ok {
				mapped[ml.PartyId] = map[string]vegapb.MarginLevels{}
			}
			mapped[ml.PartyId][ml.MarketId] = ml
		case events.MarginLevels:
			ml := et.MarginLevels()
			if _, ok := mapped[ml.PartyId]; !ok {
				mapped[ml.PartyId] = map[string]vegapb.MarginLevels{}
			}
			mapped[ml.PartyId][ml.MarketId] = ml
		}
	}
	mkts, ok := mapped[partyID]
	if !ok {
		return vegapb.MarginLevels{}, fmt.Errorf("no margin levels for party (%v)", partyID)
	}
	ml, ok := mkts[marketID]
	if !ok {
		return vegapb.MarginLevels{}, fmt.Errorf("party (%v) have no margin levels for market (%v)", partyID, marketID)
	}
	return ml, nil
}

func (b *BrokerStub) GetMarketInsurancePoolAccount(market string) (vegapb.Account, error) {
	batch := b.GetAccountEvents()
	for _, e := range batch {
		v := e.Account()
		if v.Owner == "*" && v.MarketId == market && v.Type == vegapb.AccountType_ACCOUNT_TYPE_INSURANCE {
			return v, nil
		}
	}
	return vegapb.Account{}, errors.New("account does not exist")
}

func (b *BrokerStub) GetStakingRewardAccount(asset string) (vegapb.Account, error) {
	batch := b.GetAccountEvents()
	for _, e := range batch {
		v := e.Account()
		if v.Owner == "*" && v.Asset == asset && v.Type == vegapb.AccountType_ACCOUNT_TYPE_GLOBAL_REWARD {
			return v, nil
		}
	}
	return vegapb.Account{}, errors.New("account does not exist")
}

func (b *BrokerStub) GetAssetNetworkTreasuryAccount(asset string) (vegapb.Account, error) {
	batch := b.GetAccountEvents()
	for _, e := range batch {
		v := e.Account()
		if v.Owner == "*" && v.Asset == asset && v.Type == vegapb.AccountType_ACCOUNT_TYPE_NETWORK_TREASURY {
			return v, nil
		}
	}
	return vegapb.Account{}, errors.New("account does not exist")
}

func (b *BrokerStub) GetMarketLPLiquidityFeePoolAccount(party, market string) (vegapb.Account, error) {
	batch := b.GetAccountEvents()
	for _, e := range batch {
		v := e.Account()
		if v.Owner == party && v.MarketId == market && v.Type == vegapb.AccountType_ACCOUNT_TYPE_LP_LIQUIDITY_FEES {
			return v, nil
		}
	}
	return vegapb.Account{}, errors.New("account does not exist")
}

func (b *BrokerStub) GetAssetGlobalInsuranceAccount(asset string) (vegapb.Account, error) {
	batch := b.GetAccountEvents()
	for _, e := range batch {
		v := e.Account()
		if v.Owner == "*" && v.Asset == asset && v.Type == vegapb.AccountType_ACCOUNT_TYPE_GLOBAL_INSURANCE {
			return v, nil
		}
	}
	return vegapb.Account{}, errors.New("account does not exist")
}

func (b *BrokerStub) GetMarketLPLiquidityBondAccount(party, market string) (vegapb.Account, error) {
	batch := b.GetAccountEvents()
	for _, e := range batch {
		v := e.Account()
		if v.Owner == party && v.MarketId == market && v.Type == vegapb.AccountType_ACCOUNT_TYPE_BOND {
			return v, nil
		}
	}
	return vegapb.Account{}, errors.New("account does not exist")
}

func (b *BrokerStub) GetMarketLiquidityFeePoolAccount(market string) (vegapb.Account, error) {
	batch := b.GetAccountEvents()
	for _, e := range batch {
		v := e.Account()
		if v.Owner == "*" && v.MarketId == market && v.Type == vegapb.AccountType_ACCOUNT_TYPE_FEES_LIQUIDITY {
			return v, nil
		}
	}
	return vegapb.Account{}, errors.New("account does not exist")
}

func (b *BrokerStub) GetMarketInfrastructureFeePoolAccount(asset string) (vegapb.Account, error) {
	batch := b.GetAccountEvents()
	for _, e := range batch {
		v := e.Account()
		if v.Owner == "*" && v.Asset == asset && v.Type == vegapb.AccountType_ACCOUNT_TYPE_FEES_INFRASTRUCTURE {
			return v, nil
		}
	}
	return vegapb.Account{}, errors.New("account does not exist")
}

func (b *BrokerStub) GetPartyOrderMarginAccount(party, market string) (vegapb.Account, error) {
	batch := b.GetAccountEvents()
	for _, e := range batch {
		v := e.Account()
		if v.Owner == party && v.Type == vegapb.AccountType_ACCOUNT_TYPE_ORDER_MARGIN && v.MarketId == market {
			return v, nil
		}
	}
	return vegapb.Account{}, errors.New("account does not exist")
}

func (b *BrokerStub) GetPartyMarginAccount(party, market string) (vegapb.Account, error) {
	batch := b.GetAccountEvents()
	for _, e := range batch {
		v := e.Account()
		if v.Owner == party && v.Type == vegapb.AccountType_ACCOUNT_TYPE_MARGIN && v.MarketId == market {
			return v, nil
		}
	}
	return vegapb.Account{}, errors.New("account does not exist")
}

func (b *BrokerStub) GetMarketSettlementAccount(market string) (vegapb.Account, error) {
	batch := b.GetAccountEvents()
	for _, e := range batch {
		v := e.Account()
		if v.Owner == "*" && v.MarketId == market && v.Type == vegapb.AccountType_ACCOUNT_TYPE_SETTLEMENT {
			return v, nil
		}
	}
	return vegapb.Account{}, errors.New("account does not exist")
}

// GetPartyGeneralAccount returns the latest event WRT the party's general account.
func (b *BrokerStub) GetPartyGeneralAccount(party, asset string) (ga vegapb.Account, err error) {
	batch := b.GetAccountEvents()
	foundOne := false
	for _, e := range batch {
		v := e.Account()
		if v.Owner == party && v.Type == vegapb.AccountType_ACCOUNT_TYPE_GENERAL && v.Asset == asset {
			ga = v
			foundOne = true
		}
	}
	if !foundOne {
		ga = vegapb.Account{}
		err = errors.New("account does not exist")
	}
	return
}

// GetPartyVestingAccount returns the latest event WRT the party's general account.
func (b *BrokerStub) GetPartyVestingAccount(party, asset string) (ga vegapb.Account, err error) {
	batch := b.GetAccountEvents()
	err = AccountDoesNotExistErr
	for _, e := range batch {
		v := e.Account()
		if v.Owner == party && v.Type == vegapb.AccountType_ACCOUNT_TYPE_VESTING_REWARDS && v.Asset == asset {
			ga = v
			err = nil
		}
	}
	return
}

// GetPartyVestingAccount returns the latest event WRT the party's general account.
func (b *BrokerStub) GetPartyVestedAccount(party, asset string) (ga vegapb.Account, err error) {
	batch := b.GetAccountEvents()
	err = AccountDoesNotExistErr
	for _, e := range batch {
		v := e.Account()
		if v.Owner == party && v.Type == vegapb.AccountType_ACCOUNT_TYPE_VESTED_REWARDS && v.Asset == asset {
			ga = v
			err = nil
		}
	}
	return
}

// GetPartyGeneralAccount returns the latest event WRT the party's general account.
func (b *BrokerStub) GetPartyHoldingAccount(party, asset string) (ga vegapb.Account, err error) {
	batch := b.GetAccountEvents()
	err = errors.New("account does not exist")
	for _, e := range batch {
		v := e.Account()
		if v.Owner == party && v.Type == vegapb.AccountType_ACCOUNT_TYPE_HOLDING && v.Asset == asset {
			ga = v
			err = nil
		}
	}

	return
}

// GetRewardAccountBalance returns the latest event WRT the reward accounts with the given type for the asset.
func (b *BrokerStub) GetRewardAccountBalance(accountType, asset string) (ga vegapb.Account, err error) {
	batch := b.GetAccountEvents()
	at := vegapb.AccountType(proto.AccountType_value[accountType])
	err = errors.New("account does not exist")
	for _, e := range batch {
		v := e.Account()
		if v.Owner == "*" && v.Type == at && v.Asset == asset {
			ga = v
			err = nil
		}
	}

	return
}

func (b *BrokerStub) ReferralSetStats() []*types.ReferralSetStats {
	batch := b.GetBatch(events.ReferralSetStatsUpdatedEvent)

	stats := make([]*types.ReferralSetStats, 0, len(batch))
	for _, event := range batch {
		switch et := event.(type) {
		case *events.ReferralSetStatsUpdated:
			stats = append(stats, et.Unwrap())
		case events.ReferralSetStatsUpdated:
			stats = append(stats, et.Unwrap())
		}
	}
	return stats
}

func (b *BrokerStub) VestingStats() []eventspb.VestingStatsUpdated {
	batch := b.GetBatch(events.VestingStatsUpdatedEvent)

	stats := make([]eventspb.VestingStatsUpdated, 0, len(batch))
	for _, event := range batch {
		switch et := event.(type) {
		case *events.VestingStatsUpdated:
			stats = append(stats, et.Proto())
		case events.VestingStatsUpdated:
			stats = append(stats, et.Proto())
		}
	}
	return stats
}

func (b *BrokerStub) VolumeDiscountStats() []eventspb.VolumeDiscountStatsUpdated {
	batch := b.GetBatch(events.VolumeDiscountStatsUpdatedEvent)

	stats := make([]eventspb.VolumeDiscountStatsUpdated, 0, len(batch))
	for _, event := range batch {
		switch et := event.(type) {
		case *events.VolumeDiscountStatsUpdated:
			stats = append(stats, et.Proto())
		case events.VolumeDiscountStatsUpdated:
			stats = append(stats, et.Proto())
		}
	}
	return stats
}

func (b *BrokerStub) PartyActivityStreaks() []eventspb.PartyActivityStreak {
	batch := b.GetBatch(events.PartyActivityStreakEvent)

	stats := make([]eventspb.PartyActivityStreak, 0, len(batch))
	for _, event := range batch {
		switch et := event.(type) {
		case *events.PartyActivityStreak:
			stats = append(stats, et.Proto())
		case events.PartyActivityStreak:
			stats = append(stats, et.Proto())
		}
	}
	return stats
}

func (b *BrokerStub) GetPartyBondAccount(party, asset string) (ba vegapb.Account, err error) {
	batch := b.GetAccountEvents()
	err = errors.New("account does not exist")
	for _, e := range batch {
		v := e.Account()
		if v.Owner == party && v.Type == vegapb.AccountType_ACCOUNT_TYPE_BOND && v.Asset == asset {
			// may not be the latest ballence, so keep iterating
			ba = v
			err = nil
		}
	}
	return
}

func (b *BrokerStub) GetPartyBondAccountForMarket(party, asset, marketID string) (ba vegapb.Account, err error) {
	batch := b.GetAccountEvents()
	err = errors.New("account does not exist")
	for _, e := range batch {
		v := e.Account()
		if v.Owner == party && v.Type == vegapb.AccountType_ACCOUNT_TYPE_BOND && v.Asset == asset && v.MarketId == marketID {
			// may not be the latest ballence, so keep iterating
			ba = v
			err = nil
		}
	}
	return
}

func (b *BrokerStub) GetPartyVestingAccountForMarket(party, asset, marketID string) (vegapb.Account, error) {
	batch := b.GetAccountEvents()
	for _, e := range batch {
		v := e.Account()
		if v.Owner == party && v.Type == vegapb.AccountType_ACCOUNT_TYPE_VESTING_REWARDS && v.Asset == asset && v.MarketId == marketID {
			return v, nil
		}
	}
	return vegapb.Account{}, errors.New("account does not exist")
}

func (b *BrokerStub) GetPartyVestedAccountForMarket(party, asset, marketID string) (vegapb.Account, error) {
	batch := b.GetAccountEvents()
	for _, e := range batch {
		v := e.Account()
		if v.Owner == party && v.Type == vegapb.AccountType_ACCOUNT_TYPE_VESTED_REWARDS && v.Asset == asset && v.MarketId == marketID {
			return v, nil
		}
	}
	return vegapb.Account{}, errors.New("account does not exist")
}

func (b *BrokerStub) ClearOrderByReference(party, ref string) error {
	b.mu.Lock()
	data := b.data[events.OrderEvent]
	cleared := make([]events.Event, 0, cap(data))
	for _, evt := range data {
		var o *vegapb.Order
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

func (b *BrokerStub) GetFirstByReference(party, ref string) (vegapb.Order, error) {
	data := b.GetOrderEvents()
	for _, o := range data {
		v := o.Order()
		if v.Reference == ref && v.PartyId == party {
			return *v, nil
		}
	}
	return vegapb.Order{}, fmt.Errorf("no order for party %v and reference %v", party, ref)
}

func (b *BrokerStub) GetByReference(party, ref string) (vegapb.Order, error) {
	data := b.GetOrderEvents()

	var last vegapb.Order // we need the most recent event, the order object is not updated (copy v pointer, issue 2353)
	matched := false
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
	return vegapb.Order{}, fmt.Errorf("no order for party %v and reference %v", party, ref)
}

func (b *BrokerStub) GetStopByReference(party, ref string) (eventspb.StopOrderEvent, error) {
	data := b.GetStopOrderEvents()

	var last eventspb.StopOrderEvent // we need the most recent event, the order object is not updated (copy v pointer, issue 2353)
	matched := false
	for _, o := range data {
		v := o.StopOrder()
		if v.Submission.Reference == ref && v.StopOrder.PartyId == party {
			last = *v
			matched = true
		}
	}
	if matched {
		return last, nil
	}
	return eventspb.StopOrderEvent{}, fmt.Errorf("no order for party %v and reference %v", party, ref)
}

func (b *BrokerStub) GetTxErrors() []events.TxErr {
	batch := b.GetBatch(events.TxErrEvent)
	if len(batch) == 0 {
		return nil
	}
	ret := make([]events.TxErr, 0, len(batch))
	for _, e := range batch {
		switch te := e.(type) {
		case *events.TxErr:
			ret = append(ret, *te)
		case events.TxErr:
			ret = append(ret, te)
		}
	}
	return ret
}

func (b *BrokerStub) GetLPSErrors() []events.TxErr {
	errs := b.GetTxErrors()
	if len(errs) == 0 {
		return nil
	}
	ret := make([]events.TxErr, 0, len(errs)/2)
	for _, e := range errs {
		if p := e.Proto(); p.GetLiquidityProvisionSubmission() != nil {
			ret = append(ret, e)
		}
	}
	return ret
}

func (b *BrokerStub) GetTrades() []vegapb.Trade {
	data := b.GetTradeEvents()
	trades := make([]vegapb.Trade, 0, len(data))
	for _, t := range data {
		trades = append(trades, t.Trade())
	}
	return trades
}

// AMM events concentrated here

func (b *BrokerStub) GetAMMPoolEvents() []*events.AMMPool {
	data := b.GetImmBatch(events.AMMPoolEvent)
	ret := make([]*events.AMMPool, 0, len(data))
	for _, e := range data {
		switch et := e.(type) {
		case events.AMMPool:
			ret = append(ret, ptr.From(et))
		case *events.AMMPool:
			ret = append(ret, et)
		}
	}
	return ret
}

func (b *BrokerStub) GetAMMPoolEventsByParty(party string) []*events.AMMPool {
	evts := b.GetAMMPoolEvents()
	ret := make([]*events.AMMPool, 0, 5) // we expect to get more than 1
	for _, e := range evts {
		if e.IsParty(party) {
			ret = append(ret, e)
		}
	}
	return ret
}

func (b *BrokerStub) GetAMMPoolEventsByMarket(id string) []*events.AMMPool {
	evts := b.GetAMMPoolEvents()
	ret := make([]*events.AMMPool, 0, 10)
	for _, e := range evts {
		if e.MarketID() == id {
			ret = append(ret, e)
		}
	}
	return ret
}

func (b *BrokerStub) GetAMMPoolEventsByPartyAndMarket(party, mID string) []*events.AMMPool {
	evts := b.GetAMMPoolEvents()
	ret := make([]*events.AMMPool, 0, 5)
	for _, e := range evts {
		if e.IsParty(party) && e.MarketID() == mID {
			ret = append(ret, e)
		}
	}
	return ret
}

func (b *BrokerStub) GetLastAMMPoolEvents() map[string]map[string]*events.AMMPool {
	ret := map[string]map[string]*events.AMMPool{}
	evts := b.GetAMMPoolEvents()
	for _, e := range evts {
		mID := e.MarketID()
		mmap, ok := ret[mID]
		if !ok {
			mmap = map[string]*events.AMMPool{}
		}
		mmap[e.PartyID()] = e
		ret[mID] = mmap
	}
	return ret
}

func (b *BrokerStub) GetAMMPoolEventMap() map[string]map[string][]*events.AMMPool {
	ret := map[string]map[string][]*events.AMMPool{}
	evts := b.GetAMMPoolEvents()
	for _, e := range evts {
		mID := e.MarketID()
		mmap, ok := ret[mID]
		if !ok {
			mmap = map[string][]*events.AMMPool{}
		}
		pID := e.PartyID()
		ps, ok := mmap[pID]
		if !ok {
			ps = []*events.AMMPool{}
		}
		mmap[pID] = append(ps, e)
		ret[mID] = mmap
	}
	return ret
}

func (b *BrokerStub) ResetType(t events.Type) {
	b.mu.Lock()
	b.data[t] = []events.Event{}
	b.mu.Unlock()
}

func (b *BrokerStub) filterTxErr(predicate func(errProto eventspb.TxErrorEvent) bool) []events.TxErr {
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

func (b *BrokerStub) Reset() {
	b.mu.Lock()
	b.data = map[events.Type][]events.Event{}
	b.mu.Unlock()
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

func (b *BrokerStub) SubscribeBatch(subs ...broker.Subscriber) {}

func (b *BrokerStub) Unsubscribe(k int) {}
