package mocks

import (
	"sync"

	"code.vegaprotocol.io/vega/core/events"
	"github.com/golang/mock/gomock"
)

// MockBroker - drop in mock that allows us to check the events themselves in unit tests (and as such ensure the state changes are correct)
// We're only overriding the Send and SendBatch functions. The way in which this is done shouldn't be a problem, even when using DoAndReturn, but you never know...
type MockBroker struct {
	// embed the broker mock here... this is how we can end up with a drop-in replacement
	*MockBrokerI

	// settlement has a TestConcurrent test, which causes data race on this wrapped mock
	mu *sync.Mutex
	// all events in a map per type
	// the last of each event type
	// and last events for each event type by ID (e.g. latest order event given the order ID)
	allEvts    map[events.Type][]events.Event
	lastEvts   map[events.Type]events.Event
	lastEvtsID map[events.Type]map[string]events.Event
}

func NewMockBroker(ctrl *gomock.Controller) *MockBroker {
	mbi := NewMockBrokerI(ctrl)
	return &MockBroker{
		MockBrokerI: mbi,
		mu:          &sync.Mutex{},
		allEvts:     map[events.Type][]events.Event{},
		lastEvts:    map[events.Type]events.Event{},
		lastEvtsID:  map[events.Type]map[string]events.Event{},
	}
}

// Send - first call Send on the underlying mock, then add the argument to the various maps.
func (b *MockBroker) Send(event events.Event) {
	// first call the regular mock
	b.MockBrokerI.Send(event)
	b.mu.Lock()
	t := event.Type()
	s, ok := b.allEvts[t]
	if !ok {
		s = []events.Event{}
	}
	s = append(s, event)
	b.allEvts[t] = s
	b.lastEvts[t] = event
	if ok, id := isIDEvt(event); ok {
		m, ok := b.lastEvtsID[t]
		if !ok {
			m = map[string]events.Event{}
		}
		m[id] = event
		b.lastEvtsID[t] = m
	}
	b.mu.Unlock()
}

// GetAllByType returns all events of a given type the mock has received.
func (b *MockBroker) GetAllByType(t events.Type) []events.Event {
	b.mu.Lock()
	allEvts := b.allEvts
	b.mu.Unlock()
	if s, ok := allEvts[t]; ok {
		return s
	}
	return nil
}

// GetLastByType returns the most recent event for a given type. If SendBatch was called, this is the last event of the batch.
func (b *MockBroker) GetLastByType(t events.Type) events.Event {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.lastEvts[t]
}

// GetLastByTypeAndID returns the last event of a given type, for a specific identified (party, market, order, etc...)
// list of implemented events - and ID's used:
//  * Order (by order ID)
//  * Account (by account ID)
//  * Asset (by asset ID)
//  * Auction (by market ID)
//  * Deposit (party ID)
//  * Proposal (proposal ID)
//  * LP (by party ID)
//  * MarginLevels (party ID)
//  * MarketData (market ID)
//  * PosRes (market ID)
//  * RiskFactor (market ID)
//  * SettleDistressed (party ID)
//  * Vote (currently PartyID, might want to use proposalID, too?)
//  * Withdrawal (PartyID)
func (b *MockBroker) GetLastByTypeAndID(t events.Type, id string) events.Event {
	b.mu.Lock()
	m, ok := b.lastEvtsID[t]
	b.mu.Unlock()
	if !ok {
		return nil
	}
	return m[id]
}

// @TODO loss socialization. Given that this is something that would impact several parties, there's most likely
// no real point to filtering by ID.
// Not implemented yet, but worth considering:
//  * Trade
//  * TransferResponse
// Implemented events:
//  * Order (by order ID)
//  * Account (by account ID)
//  * Asset (by asset ID)
//  * Auction (by market ID)
//  * Deposit (party ID)
//  * Proposal (proposal ID)
//  * LP (by party ID)
//  * MarginLevels (party ID)
//  * MarketData (market ID)
//  * PosRes (market ID)
//  * RiskFactor (market ID)
//  * SettleDistressed (party ID)
//  * Vote (currently PartyID, might want to use proposalID, too?)
//  * Withdrawal (PartyID)
func isIDEvt(e events.Event) (bool, string) {
	switch et := e.(type) {
	case *events.Order:
		return true, et.Order().Id
	case events.Order:
		return true, et.Order().Id
	case *events.Acc:
		return true, et.Account().Id
	case events.Acc:
		return true, et.Account().Id
	case *events.Asset:
		return true, et.Asset().Id
	case events.Asset:
		return true, et.Asset().Id
	case *events.Auction:
		return true, et.MarketID()
	case events.Auction:
		return true, et.MarketID()
	case *events.Deposit:
		return true, et.Deposit().PartyId
	case events.Deposit:
		return true, et.Deposit().PartyId
	case *events.Proposal:
		return true, et.ProposalID()
	case events.Proposal:
		return true, et.ProposalID()
	case *events.LiquidityProvision:
		return true, et.PartyID()
	case events.LiquidityProvision:
		return true, et.PartyID()
	case *events.MarginLevels:
		return true, et.PartyID()
	case events.MarginLevels:
		return true, et.PartyID()
	case *events.MarketData:
		return true, et.MarketID()
	case events.MarketData:
		return true, et.MarketID()
	case *events.PosRes:
		return true, et.MarketID()
	case events.PosRes:
		return true, et.MarketID()
	case *events.RiskFactor:
		return true, et.MarketID()
	case events.RiskFactor:
		return true, et.MarketID()
	case *events.SettleDistressed:
		return true, et.PartyID()
	case events.SettleDistressed:
		return true, et.PartyID()
	case *events.Vote:
		return true, et.PartyID()
	case events.Vote:
		return true, et.PartyID()
	case *events.Withdrawal:
		return true, et.PartyID()
	case events.Withdrawal:
		return true, et.PartyID()
	}
	return false, ""
}
