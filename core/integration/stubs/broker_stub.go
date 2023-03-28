// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.VEGA file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package stubs

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"code.vegaprotocol.io/vega/libs/broker"

	"code.vegaprotocol.io/vega/core/events"
	vtypes "code.vegaprotocol.io/vega/core/types"
	proto "code.vegaprotocol.io/vega/protos/vega"
	types "code.vegaprotocol.io/vega/protos/vega"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"
)

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

// GetTransfers returns ledger entries, mutable argument specifies if these should be all the scenario events or events that can be cleared by the user.
func (b *BrokerStub) GetTransfers(mutable bool) []*types.LedgerEntry {
	transferEvents := b.GetLedgerMovements(mutable)
	transfers := []*types.LedgerEntry{}
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
	activeOrders := map[string]*types.Order{}
	for _, e := range batch {
		var ord *types.Order
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

		if ord.Status == types.Order_STATUS_ACTIVE {
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
		if v.Side == types.Side_SIDE_BUY {
			buy[v.Price] = buy[v.Price] + v.Remaining
			continue
		}
		sell[v.Price] = sell[v.Price] + v.Remaining
	}

	return sell, buy
}

func (b *BrokerStub) GetActiveOrderDepth(marketID string) (sell []*types.Order, buy []*types.Order) {
	batch := b.GetImmBatch(events.OrderEvent)
	if len(batch) == 0 {
		return nil, nil
	}
	active := make(map[string]*types.Order, len(batch))
	for _, e := range batch {
		var ord *types.Order
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
		if ord.Status == types.Order_STATUS_ACTIVE {
			active[ord.Id] = ord
		} else {
			delete(active, ord.Id)
		}
	}
	c := len(active) / 2
	if len(active)%2 == 1 {
		c++
	}
	sell, buy = make([]*types.Order, 0, c), make([]*types.Order, 0, c)
	for _, ord := range active {
		if ord.Side == types.Side_SIDE_BUY {
			buy = append(buy, ord)
			continue
		}
		sell = append(sell, ord)
	}

	return sell, buy
}

func (b *BrokerStub) GetMarket(marketID string) *types.Market {
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
	last := map[string]*vtypes.Order{}
	ret := make([]events.Order, 0, len(batch))
	for _, e := range batch {
		var o *vtypes.Order
		switch et := e.(type) {
		case *events.Order:
			o, _ = vtypes.OrderFromProto(et.Order())
			ret = append(ret, *et)
		case events.Order:
			o, _ = vtypes.OrderFromProto(et.Order())
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
				o.Status = types.Order_STATUS_EXPIRED
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

func (b *BrokerStub) GetDeposits() []types.Deposit {
	// Use GetImmBatch so that clearing events doesn't affact this method
	batch := b.GetImmBatch(events.DepositEvent)
	if len(batch) == 0 {
		return nil
	}
	ret := make([]types.Deposit, 0, len(batch))
	for _, e := range batch {
		switch et := e.(type) {
		case *events.Deposit:
			ret = append(ret, et.Deposit())
		}
	}
	return ret
}

func (b *BrokerStub) GetWithdrawals() []types.Withdrawal {
	// Use GetImmBatch so that clearing events doesn't affact this method
	batch := b.GetImmBatch(events.WithdrawalEvent)
	if len(batch) == 0 {
		return nil
	}
	ret := make([]types.Withdrawal, 0, len(batch))
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

func (b *BrokerStub) GetDelegationBalance(epochSeq string) []types.Delegation {
	evts := b.GetDelegationBalanceEvents(epochSeq)
	balances := make([]types.Delegation, 0, len(evts))

	for _, e := range evts {
		balances = append(balances, types.Delegation{
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

func (b *BrokerStub) GetAccounts() []types.Account {
	evts := b.GetAccountEvents()
	accounts := make([]types.Account, 0, len(evts))
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
	batch := b.GetAccountEvents()
	for _, e := range batch {
		v := e.Account()
		if v.Owner == "*" && v.MarketId == market && v.Type == types.AccountType_ACCOUNT_TYPE_INSURANCE {
			return v, nil
		}
	}
	return types.Account{}, errors.New("account does not exist")
}

func (b *BrokerStub) GetAssetNetworkTreasuryAccount(asset string) (types.Account, error) {
	batch := b.GetAccountEvents()
	for _, e := range batch {
		v := e.Account()
		if v.Owner == "*" && v.Asset == asset && v.Type == types.AccountType_ACCOUNT_TYPE_GLOBAL_REWARD {
			return v, nil
		}
	}
	return types.Account{}, errors.New("account does not exist")
}

func (b *BrokerStub) GetMarketLiquidityFeePoolAccount(market string) (types.Account, error) {
	batch := b.GetAccountEvents()
	for _, e := range batch {
		v := e.Account()
		if v.Owner == "*" && v.MarketId == market && v.Type == types.AccountType_ACCOUNT_TYPE_FEES_LIQUIDITY {
			return v, nil
		}
	}
	return types.Account{}, errors.New("account does not exist")
}

func (b *BrokerStub) GetMarketInfrastructureFeePoolAccount(asset string) (types.Account, error) {
	batch := b.GetAccountEvents()
	for _, e := range batch {
		v := e.Account()
		if v.Owner == "*" && v.Asset == asset && v.Type == types.AccountType_ACCOUNT_TYPE_FEES_INFRASTRUCTURE {
			return v, nil
		}
	}
	return types.Account{}, errors.New("account does not exist")
}

func (b *BrokerStub) GetPartyMarginAccount(party, market string) (types.Account, error) {
	batch := b.GetAccountEvents()
	for _, e := range batch {
		v := e.Account()
		if v.Owner == party && v.Type == types.AccountType_ACCOUNT_TYPE_MARGIN && v.MarketId == market {
			return v, nil
		}
	}
	return types.Account{}, errors.New("account does not exist")
}

func (b *BrokerStub) GetMarketSettlementAccount(market string) (types.Account, error) {
	batch := b.GetAccountEvents()
	for _, e := range batch {
		v := e.Account()
		if v.Owner == "*" && v.MarketId == market && v.Type == types.AccountType_ACCOUNT_TYPE_SETTLEMENT {
			return v, nil
		}
	}
	return types.Account{}, errors.New("account does not exist")
}

// GetPartyGeneralAccount returns the latest event WRT the party's general account.
func (b *BrokerStub) GetPartyGeneralAccount(party, asset string) (ga types.Account, err error) {
	batch := b.GetAccountEvents()
	err = errors.New("account does not exist")
	for _, e := range batch {
		v := e.Account()
		if v.Owner == party && v.Type == types.AccountType_ACCOUNT_TYPE_GENERAL && v.Asset == asset {
			ga = v
			err = nil
		}
	}

	return
}

// GetRewardAccountBalance returns the latest event WRT the reward accounts with the given type for the asset.
func (b *BrokerStub) GetRewardAccountBalance(accountType, asset string) (ga types.Account, err error) {
	batch := b.GetAccountEvents()
	at := types.AccountType(proto.AccountType_value[accountType])
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

func (b *BrokerStub) GetPartyBondAccount(party, asset string) (ba types.Account, err error) {
	batch := b.GetAccountEvents()
	err = errors.New("account does not exist")
	for _, e := range batch {
		v := e.Account()
		if v.Owner == party && v.Type == types.AccountType_ACCOUNT_TYPE_BOND && v.Asset == asset {
			// may not be the latest ballence, so keep iterating
			ba = v
			err = nil
		}
	}
	return
}

func (b *BrokerStub) GetPartyBondAccountForMarket(party, asset, marketID string) (ba types.Account, err error) {
	batch := b.GetAccountEvents()
	err = errors.New("account does not exist")
	for _, e := range batch {
		v := e.Account()
		if v.Owner == party && v.Type == types.AccountType_ACCOUNT_TYPE_BOND && v.Asset == asset && v.MarketId == marketID {
			// may not be the latest ballence, so keep iterating
			ba = v
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
	return types.Order{}, fmt.Errorf("no order for party %v and reference %v", party, ref)
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
