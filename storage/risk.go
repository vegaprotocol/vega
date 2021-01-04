package storage

import (
	"fmt"
	"sync"

	"code.vegaprotocol.io/vega/logging"
	types "code.vegaprotocol.io/vega/proto/gen/golang"

	"github.com/pkg/errors"
)

var (
	ErrNoMarginLevelsForParty  = errors.New("no margin levels for party")
	ErrNoMarginLevelsForMarket = errors.New("party have no margin levels for the market")
	ErrNoRiskFactorsForMarket  = errors.New("no risk factors for market")
)

// Risk is used for memory/RAM based risk storage.
type Risk struct {
	Config
	log *logging.Logger
	// party to market to margin levels
	margins   map[string]map[string]types.MarginLevels
	marginsMu sync.RWMutex

	riskFactors map[string]types.RiskFactor
	riskFMu     sync.RWMutex

	subscribers  map[uint64]chan []types.MarginLevels
	subscriberID uint64
	mu           sync.Mutex
}

// NewRisks returns a concrete implementation of RiskStore.
func NewRisks(log *logging.Logger, c Config) *Risk {
	log = log.Named(namedLogger)
	log.SetLevel(c.Level.Get())

	return &Risk{
		Config:      c,
		log:         log,
		margins:     map[string]map[string]types.MarginLevels{},
		riskFactors: map[string]types.RiskFactor{},
		subscribers: map[uint64]chan []types.MarginLevels{},
	}
}

// ReloadConf update the internal configuration of the risk
func (r *Risk) ReloadConf(config Config) {
	// nothing to do for now
}

func (r *Risk) GetMarketRiskFactors(marketID string) (types.RiskFactor, error) {
	r.riskFMu.RLock()
	defer r.riskFMu.RUnlock()
	rf, ok := r.riskFactors[marketID]
	if !ok {
		return types.RiskFactor{}, ErrNoRiskFactorsForMarket
	}
	return rf, nil
}

func (r *Risk) GetMarginLevelsByID(partyID, marketID string) ([]types.MarginLevels, error) {
	if len(marketID) > 0 {
		ml, err := r.getMarginLevelsByPartyAndMarket(partyID, marketID)
		if err != nil {
			return nil, err
		}
		return []types.MarginLevels{ml}, nil
	}
	return r.getMarginLevelsByParty(partyID)
}

// GetMarginLevels returns the margin levels for a given party
func (r *Risk) getMarginLevelsByPartyAndMarket(partyID, marketID string) (types.MarginLevels, error) {
	r.marginsMu.RLock()
	p, ok := r.margins[partyID]
	if !ok {
		r.marginsMu.RUnlock()
		return types.MarginLevels{}, ErrNoMarginLevelsForParty
	}
	m, ok := p[marketID]
	if !ok {
		r.marginsMu.RUnlock()
		return types.MarginLevels{}, ErrNoMarginLevelsForMarket
	}
	r.marginsMu.RUnlock()
	return m, nil
}

// GetMarginLevels returns the margin levels for a given party
func (r *Risk) getMarginLevelsByParty(partyID string) ([]types.MarginLevels, error) {
	r.marginsMu.RLock()
	_, ok := r.margins[partyID]
	if !ok {
		r.marginsMu.RUnlock()
		return nil, ErrNoMarginLevelsForParty
	}
	out := make([]types.MarginLevels, 0, len(r.margins[partyID]))
	for _, v := range r.margins[partyID] {
		out = append(out, v)
	}
	r.marginsMu.RUnlock()
	return out, nil
}

// Commit typically saves any operations that are queued to underlying storage,
// if supported by underlying storage implementation.
func (r *Risk) Commit() error {
	// No work required with a mem-store implementation.
	return nil
}

// Close can be called to clean up and close any storage
// connections held by the underlying storage mechanism.
func (r *Risk) Close() error {
	// No work required with a mem-store implementation.
	return nil
}

// SaveRiskFactorBatch writes a slice of account changes to the underlying store.
func (r *Risk) SaveRiskFactorBatch(batch []types.RiskFactor) {
	if len(batch) == 0 {
		return
	}

	r.riskFMu.Lock()
	for _, v := range batch {
		v := v
		r.riskFactors[v.Market] = v
	}
	r.riskFMu.Unlock()
}

// SaveMarginLevelsBatch writes a slice of account changes to the underlying store.
func (r *Risk) SaveMarginLevelsBatch(batch []types.MarginLevels) {
	if len(batch) == 0 {
		return
	}

	r.marginsMu.Lock()
	for _, v := range batch {
		if _, ok := r.margins[v.PartyID]; !ok {
			r.margins[v.PartyID] = map[string]types.MarginLevels{}
		}
		r.margins[v.PartyID][v.MarketID] = v
	}
	r.marginsMu.Unlock()
	r.notify(batch)
}

// notify is a helper func used to send any updates to any subscribers for mutations of the
// account store.
func (r *Risk) notify(batch []types.MarginLevels) {
	if len(batch) == 0 {
		return
	}

	r.mu.Lock()
	if len(r.subscribers) == 0 {
		r.log.Debug("No subscribers connected in accounts store")
		r.mu.Unlock()
		return
	}

	var ok bool
	for id, sub := range r.subscribers {
		select {
		case sub <- batch:
			ok = true
		default:
			ok = false
		}
		if ok {
			r.log.Debug("Risk channel updated for subscriber successfully",
				logging.Uint64("id", id))
		} else {
			r.log.Debug("Risk channel could not be updated for subscriber",
				logging.Uint64("id", id))
		}
	}
	r.mu.Unlock()
}

// Subscribe to account store updates, any changes will be pushed out on this channel.
func (r *Risk) Subscribe(c chan []types.MarginLevels) uint64 {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.subscriberID++
	r.subscribers[r.subscriberID] = c

	r.log.Debug("Account subscriber added in account store",
		logging.Uint64("subscriber-id", r.subscriberID))

	return r.subscriberID
}

// Unsubscribe from account store updates.
func (r *Risk) Unsubscribe(id uint64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if len(r.subscribers) == 0 {
		r.log.Debug("Un-subscribe called in risk store, no subscribers connected",
			logging.Uint64("subscriber-id", id))
		return nil
	}

	if _, exists := r.subscribers[id]; exists {
		delete(r.subscribers, id)

		r.log.Debug("Un-subscribe called in risk store, subscriber removed",
			logging.Uint64("subscriber-id", id))

		return nil
	}

	r.log.Warn("Un-subscribe called in risk store, subscriber does not exist",
		logging.Uint64("subscriber-id", id))

	return fmt.Errorf("Risk store subscriber does not exist with id: %d", id)
}
