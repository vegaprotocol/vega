package positions

import (
	"context"
	"sync"

	"code.vegaprotocol.io/vega/config"
	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/plugins"
	types "code.vegaprotocol.io/vega/proto"
	"google.golang.org/grpc"

	"github.com/pkg/errors"
)

var (
	ErrMarketNotFound = errors.New("could not find market")
	ErrPartyNotFound  = errors.New("party not found")
)

const (
	PluginName = "positions-api"
)

//go:generate go run github.com/golang/mock/mockgen -destination mocks/positions_subscriber_mock.go -package mocks code.vegaprotocol.io/vega/plugins/positions Subscriber
type Subscriber interface {
	Recv() <-chan []events.SettlePosition
	Done() <-chan struct{}
}

// PlugBuffer - just a local redefinition of the plugins.Buffer interface used for testing
//go:generate go run github.com/golang/mock/mockgen -destination mocks/positions_plugins_buffer_mock.go -package mocks code.vegaprotocol.io/vega/plugins/positions PlugBuffer
type PlugBuffer plugins.Buffers

type Pos struct {
	ctx           context.Context
	conf          Config
	mu            sync.RWMutex // sadly, we still need this because we'll be updating this map and reading from it
	sub           Subscriber
	data          map[string]map[string]types.Position
	log           *logging.Logger
	srv           *grpc.Server
	store         *Store
	subscriptions map[int]subscription
	keys          []int
	smu           sync.Mutex
}

type subscription struct {
	ch   chan<- struct{}
	full chan struct{} // this channel indicates the retry could is reached
}

// New - keep this one here, mainly for testing
func New(ctx context.Context, sub Subscriber, store *Store) *Pos {
	return &Pos{
		ctx:           ctx,
		conf:          DefaultConfig(),
		sub:           sub,
		data:          map[string]map[string]types.Position{},
		store:         store,
		subscriptions: map[int]subscription{},
		keys:          []int{},
	}
}

// New - part of the plugin interface, need thit to make it work
func (p *Pos) New(log *logging.Logger, ctx context.Context, buf plugins.Buffers, srv *grpc.Server, rawCfg interface{}) (plugins.Plugin, error) {
	log = log.Named(PluginName)
	log.Info(
		"initializing new plugin",
		logging.String("plugin-name", PluginName),
	)

	cfg := p.conf
	if err := config.LoadPluginConfig(rawCfg, PluginName, &cfg); err != nil {
		return nil, err
	}
	log.SetLevel(cfg.Level.Get())
	// create store for empty positions
	store := NewPositionsStore(ctx)
	return &Pos{
		ctx:           ctx,
		conf:          cfg,
		sub:           buf.PositionsSub(cfg.SubscriptionBuffer),
		data:          map[string]map[string]types.Position{},
		log:           log,
		srv:           srv,
		store:         store,
		subscriptions: map[int]subscription{},
		keys:          []int{},
	}, nil
}

// Start - just an exposed func that hides the fact that we're using routines
// part of the Plugin interface needed, error returned is _always_ nil
func (p *Pos) Start() error {
	go p.consume(p.ctx)
	return nil
}

func (p *Pos) Subscribe(ch chan<- struct{}) (int, <-chan struct{}) {
	p.smu.Lock()
	k := p.getKey()
	sub := subscription{
		ch:   ch,
		full: make(chan struct{}, 1),
	}
	p.subscriptions[k] = sub
	p.smu.Unlock()
	return k, sub.full
}

func (p *Pos) Unsubscribe(k int) {
	p.smu.Lock()
	// only do something if the key actually exists
	if s, ok := p.subscriptions[k]; ok {
		// make the subscription key available again
		p.keys = append(p.keys, k)
		// remove the current entry
		delete(p.subscriptions, k)
		close(s.full)
	}
	p.smu.Unlock()
}

func (p *Pos) getKey() int {
	if len(p.keys) != 0 {
		k := p.keys[0]
		p.keys = p.keys[1:]
		return k
	}
	return len(p.subscriptions) + 1 // don't allow 0 as a subscription ID
}

// GetPositionsByMarketAndParty get the position of a single trader in a given market
// these funcs need to be moved to the server
func (p *Pos) GetPositionsByMarketAndParty(market, party string) (*types.Position, error) {
	p.mu.RLock()
	mp, ok := p.data[market]
	if !ok {
		p.mu.RUnlock()
		return nil, ErrMarketNotFound
	}
	pos, ok := mp[party]
	if !ok {
		pos = types.Position{
			PartyID:  party,
			MarketID: market,
		}
		// return nil, ErrPartyNotFound
	}
	p.mu.RUnlock()
	return &pos, nil
}

// GetPositionsByParty get all positions for a given trader
func (p *Pos) GetPositionsByParty(party string) ([]*types.Position, error) {
	p.mu.RLock()
	// at most, trader is active in all markets
	positions := make([]*types.Position, 0, len(p.data))
	for _, traders := range p.data {
		if pos, ok := traders[party]; ok {
			positions = append(positions, &pos)
		}
	}
	p.mu.RUnlock()
	if len(positions) == 0 {
		return nil, nil
		// return nil, ErrPartyNotFound
	}
	return positions, nil
}

// GetPositionsByMarket get all trader positions in a given market
func (p *Pos) GetPositionsByMarket(market string) ([]*types.Position, error) {
	p.mu.RLock()
	mp, ok := p.data[market]
	if !ok {
		p.mu.RUnlock()
		return nil, ErrMarketNotFound
	}
	s := make([]*types.Position, 0, len(mp))
	for _, tp := range mp {
		s = append(s, &tp)
	}
	p.mu.RUnlock()
	return s, nil
}

func (p *Pos) consume(ctx context.Context) {
	for {
		select {
		case <-p.sub.Done():
			// if we no longer receive data, we can stop here
			return
		case <-ctx.Done():
			return
		case data, ok := <-p.sub.Recv():
			if !ok {
				// the channel was closed
				return
			}
			if len(data) == 0 {
				continue
			}
			p.mu.RLock()
			// get a copy, we only need a read lock here
			cpy := p.data
			p.mu.RUnlock()
			// the overwrite is done inside the updateData call, with a full lock
			// the reasoning being that this keeps the map accessible for API calls while
			// we're updating positions
			p.updateData(cpy, data)
		}
	}
}

func (p *Pos) updateData(data map[string]map[string]types.Position, raw []events.SettlePosition) {
	for _, sp := range raw {
		mID, tID := sp.MarketID(), sp.Party()
		if _, ok := data[mID]; !ok {
			data[mID] = map[string]types.Position{}
		}
		var (
			calc types.Position
			ok   bool
		)
		// check if we can get calc pos from data map
		if calc, ok = data[mID][tID]; !ok {
			// if not, fall back to previously closed out positions
			// use Pop, gets position and removes it if found
			if pos, err := p.store.Pop(mID, tID); err != nil {
				// if we couldn't find a closed position, create the new one from the event
				calc = evtToProto(sp)
			} else {
				// else, re-open the closed position
				calc = *pos
			}
		}
		updatePosition(&calc, sp)
		if calc.OpenVolume == 0 {
			delete(data[mID], tID)
			p.store.Add(calc)
		} else {
			data[mID][tID] = calc
		}
	}
	// keep lock time to a minimum, we're working on a copy here, and reassign the data field
	// only after everything has been updated (instead of maintaining a lock throughout)
	p.mu.Lock()
	p.data = data
	p.mu.Unlock()
	// now that the data has been updated, let any subscriptions know there's fresh data
	// lock the map in case a subscription is removed meanwhile
	p.smu.Lock()
	for _, sub := range p.subscriptions {
		// only write if this won't block
		if len(sub.ch) != cap(sub.ch) {
			sub.ch <- struct{}{}
		} else {
			select {
			case sub.full <- struct{}{}:
			default:
				// full channel is already set, just skip over this subscription, it'll be closed
			}
		}
	}
	p.smu.Unlock()
}

func updatePosition(p *types.Position, e events.SettlePosition) {
	var (
		// delta uint64
		pnl, delta int64
	)
	tradePnl := make([]int64, 0, len(e.Trades()))
	for _, t := range e.Trades() {
		size, sAbs := t.Size(), absUint64(t.Size())
		// approach each trade using the open volume as a starting-point
		current := p.OpenVolume
		if current != 0 {
			cAbs := absUint64(current)
			// trade direction is actually closing volume
			if (current > 0 && size < 0) || (current < 0 && size > 0) {
				if sAbs > cAbs {
					delta = current
					current = 0
				} else {
					delta = -size
					current += size
				}
			}
			// only increment realised P&L if the size goes the opposite way compared to the the
			// current position
			if (size > 0 && p.OpenVolume <= 0) || (size < 0 && p.OpenVolume >= 0) {
				pnl = delta * int64(t.Price()-p.AverageEntryPrice)
				p.RealisedPNL += pnl
				tradePnl = append(tradePnl, pnl)
				// @TODO store trade record with this realised P&L value
			}
		}
		if net := delta + size; net != 0 {
			if size != p.OpenVolume {
				cAbs := absUint64(p.OpenVolume)
				p.AverageEntryPrice = (p.AverageEntryPrice*cAbs + t.Price()*sAbs) / (sAbs + cAbs)
			} else {
				p.AverageEntryPrice = 0
			}
		}
		p.OpenVolume += size
	}
	// p.PendingVolume = p.OpenVolume + e.Buy() - e.Sell()
	// MTM price * open volume == total value of current pos the entry price/cost of said position
	p.UnrealisedPNL = (int64(e.Price()) - int64(p.AverageEntryPrice)) * p.OpenVolume
	// Technically not needed, but safer to copy the open volume from event regardless
	p.OpenVolume = e.Size()
	if p.OpenVolume != 0 && p.AverageEntryPrice == 0 {
		p.AverageEntryPrice = e.Price()
	}
}

func evtToProto(e events.SettlePosition) types.Position {
	p := types.Position{
		MarketID: e.MarketID(),
		PartyID:  e.Party(),
	}
	// NOTE: We don't call this here because the call is made in updateEvt for all positions
	// we don't want to add the same data twice!
	// updatePosition(&p, e)
	return p
}

func absUint64(v int64) uint64 {
	if v < 0 {
		v *= -1
	}
	return uint64(v)
}
