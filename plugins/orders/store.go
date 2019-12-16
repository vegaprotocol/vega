package orders

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"code.vegaprotocol.io/vega/logging"
	types "code.vegaprotocol.io/vega/proto"
)

var (
	ErrNoOrderForID                 = errors.New("not matching order for id")
	ErrPartyNotFoundInStore         = errors.New("party not found in store")
	ErrPartyOrMarketNotFoundInStore = errors.New("party or market not found in store")
	ErrTimeoutReached               = errors.New("timeout reached")
)

type partyMarket struct {
	partyID, marketID string
}

type orderStore struct {
	cfg Config
	log *logging.Logger
	mu  sync.RWMutex
	// partyid -> marketid -> orderid -> order
	store map[string]map[string]map[string]types.Order
	// orderid -> partyMarketRef
	idrefs map[string]partyMarket

	// subscribtion
	submu        sync.Mutex
	subscribers  map[uint64]chan<- []types.Order
	subscriberID uint64

	// depth
	depth map[string]*Depth
}

func newStore(log *logging.Logger, cfg Config) *orderStore {
	return &orderStore{
		cfg:         cfg,
		log:         log,
		store:       map[string]map[string]map[string]types.Order{},
		idrefs:      map[string]partyMarket{},
		subscribers: map[uint64]chan<- []types.Order{},
		depth:       map[string]*Depth{},
	}
}

func (s *orderStore) SaveBatch(batch []types.Order) {
	s.mu.Lock()
	for _, v := range batch {
		// update market depth
		// TODO: really need to move that somewhere else
		if _, ok := s.depth[v.MarketID]; !ok {
			s.depth[v.MarketID] = NewMarketDepth(v.MarketID)
		}
		s.depth[v.MarketID].Update(v)

		// then update actual store
		party, ok := s.store[v.PartyID]
		if !ok {
			party = map[string]map[string]types.Order{}
			s.store[v.PartyID] = party
		}
		mkt, ok := party[v.MarketID]
		if !ok {
			mkt = map[string]types.Order{}
			party[v.MarketID] = mkt
		}

		if v.Status != types.Order_Active {
			delete(mkt, v.Id)
			delete(s.idrefs, v.Id)
		} else {
			if _, ok := s.idrefs[v.Id]; !ok {
				s.idrefs[v.Id] = partyMarket{partyID: v.PartyID, marketID: v.MarketID}
			}
			mkt[v.Id] = v
		}
	}
	s.mu.Unlock()
	if err := s.notify(batch); err != nil {
		s.log.Error("unable to send batch to subscribers",
			logging.Error(err))
	}
}

func (s *orderStore) GetByID(id string) (*types.Order, error) {
	s.mu.RLock()
	pm, ok := s.idrefs[id]
	if !ok {
		return nil, ErrNoOrderForID
	}
	o := s.store[pm.partyID][pm.marketID][id]
	s.mu.RUnlock()
	return &o, nil
}

func (s *orderStore) GetByPartyID(partyID string) ([]*types.Order, error) {
	s.mu.RLock()
	party, ok := s.store[partyID]
	if !ok {
		s.mu.RUnlock()
		return nil, ErrPartyNotFoundInStore
	}

	var ln int
	for _, v := range party {
		ln += len(v)
	}

	orders := make([]*types.Order, 0, ln)
	for _, mkts := range party {
		for _, ord := range mkts {
			ord := ord
			orders = append(orders, &ord)
		}
	}
	s.mu.RUnlock()
	return orders, nil
}

func (s *orderStore) GetByPartyAndMarketID(partyID, marketID string) ([]*types.Order, error) {
	s.mu.RLock()
	mkt, ok := s.store[partyID][marketID]
	if !ok {
		s.mu.RUnlock()
		return nil, ErrPartyOrMarketNotFoundInStore
	}

	orders := make([]*types.Order, 0, len(mkt))
	for _, ord := range mkt {
		ord := ord
		orders = append(orders, &ord)
	}

	s.mu.RUnlock()
	return orders, nil
}

func (s *orderStore) Subscribe(orders chan<- []types.Order) uint64 {
	s.submu.Lock()
	defer s.submu.Unlock()

	s.subscriberID++
	s.subscribers[s.subscriberID] = orders

	s.log.Debug("Orders subscriber added in order store",
		logging.Uint64("subscriber-id", s.subscriberID))

	return s.subscriberID
}

func (s *orderStore) Unsubscribe(id uint64) error {
	s.submu.Lock()
	defer s.submu.Unlock()

	if len(s.subscribers) == 0 {
		s.log.Debug("Un-subscribe called in order store, no subscribers connected",
			logging.Uint64("subscriber-id", id))
		return nil
	}

	if _, exists := s.subscribers[id]; exists {
		delete(s.subscribers, id)
		s.log.Debug("Un-subscribe called in order store, subscriber removed",
			logging.Uint64("subscriber-id", id))
		return nil
	}

	return fmt.Errorf("subscriber to orders does not exist with id: %d", id)
}

func (s *orderStore) notify(items []types.Order) error {
	if len(items) == 0 {
		return nil
	}

	s.submu.Lock()
	for id, sub := range s.subscribers {
		select {
		case sub <- items:
			s.log.Debug("Orders channel updated for subscriber successfully",
				logging.Uint64("id", id))
		default:
			s.log.Debug("Orders channel could not be updated for subscriber",
				logging.Uint64("id", id))
		}
	}
	s.submu.Unlock()
	return nil
}

// GetMarketDepth calculates and returns order book/depth of market for a given market.
func (s *orderStore) GetMarketDepth(ctx context.Context, market string) (*types.MarketDepth, error) {

	// validate
	depth, ok := s.depth[market]
	if !ok || depth == nil {
		// When a market is new with no orders there will not be any market depth/order book
		// so we do not need to try and calculate the depth cumulative volumes etc
		return &types.MarketDepth{
			MarketID: market,
			Buy:      []*types.PriceLevel{},
			Sell:     []*types.PriceLevel{},
		}, nil
	}

	// load from store
	buy := depth.BuySide()
	sell := depth.SellSide()

	buyPtr := make([]*types.PriceLevel, 0, len(buy))
	sellPtr := make([]*types.PriceLevel, 0, len(sell))

	ctx, cancel := context.WithTimeout(ctx, s.cfg.Timeout.Duration)
	defer cancel()
	deadline, _ := ctx.Deadline()
	// 2 routines, each can push one error on here, so buffer to avoid deadlock
	errCh := make(chan error, 2)
	wg := sync.WaitGroup{}
	wg.Add(2)

	// recalculate accumulated volume, concurrently rather than sequentially
	// make the most of the time we have
	// --- buy side ---
	go func() {
		defer wg.Done()
		var cumulativeVolume uint64
		for i, b := range buy {
			select {
			case <-ctx.Done():
				if deadline.Before(time.Now()) {
					errCh <- ErrTimeoutReached
				}
				return
			default:
				// keep running total
				cumulativeVolume += b.Volume
				buy[i].CumulativeVolume = cumulativeVolume
				buyPtr = append(buyPtr, &buy[i].PriceLevel)
			}
		}
	}()
	// --- sell side ---
	go func() {
		defer wg.Done()
		var cumulativeVolume uint64
		for i, s := range sell {
			select {
			case <-ctx.Done():
				if deadline.Before(time.Now()) {
					errCh <- ErrTimeoutReached
				}
				return
			default:
				// keep running total
				cumulativeVolume += s.Volume
				sell[i].CumulativeVolume = cumulativeVolume
				sellPtr = append(sellPtr, &sell[i].PriceLevel)
			}
		}
	}()
	wg.Wait()
	close(errCh)
	// the second error is the same, they're both ctx.Err()
	for err := range errCh {
		return nil, err
	}

	// return new re-calculated market depth for each side of order book
	return &types.MarketDepth{
		MarketID: market,
		Buy:      buyPtr,
		Sell:     sellPtr,
	}, nil
}
