package buffer

import (
	"sync"

	"code.vegaprotocol.io/vega/events"
	types "code.vegaprotocol.io/vega/proto"
)

type settleConf func(s *Settlement)

type Settlement struct {
	mu    *sync.Mutex
	chBuf int
	buf   map[string]map[string]events.SettlePosition
	calc  map[string]map[string]types.Position // the "persistent" storage
	chans map[int]chan []types.Position
	// chans map[int]chan map[string]map[string]types.Position
	free []int
}

// ChannelBuffer - set default channel buffers to b (default 1)
func ChannelBuffer(b int) settleConf {
	return func(s *Settlement) {
		s.chBuf = b
	}
}

// New - create new settlement buffer
func NewSettlement(opts ...settleConf) *Settlement {
	s := &Settlement{
		mu:    &sync.Mutex{},
		chBuf: 1,
		buf:   map[string]map[string]events.SettlePosition{},
		calc:  map[string]map[string]types.Position{},
		chans: map[int]chan []types.Position{},
		free:  []int{},
	}
	for _, o := range opts {
		o(s)
	}
	return s
}

// Add - add position data to the buffer
func (s *Settlement) Add(p events.SettlePosition) {
	s.mu.Lock()
	mID := p.MarketID()
	if _, ok := s.buf[mID]; !ok {
		s.buf[mID] = map[string]events.SettlePosition{}
	}
	s.buf[mID][p.Party()] = p
	s.mu.Unlock()
}

// Flush - Clear buffer, passing all channels the data
func (s *Settlement) Flush() {
	s.mu.Lock()
	s.updateEvt()
	// we've processed the buffer, clear it
	s.buf = map[string]map[string]events.SettlePosition{}
	// no channels to push to, no need to create slice with data
	if len(s.chans) == 0 {
		s.mu.Unlock()
		return
	}
	sCap := 0
	for _, mc := range s.calc {
		// total traders in each market
		sCap += len(mc)
	}
	positions := make([]types.Position, 0, sCap)
	for _, traders := range s.calc {
		for _, t := range traders {
			positions = append(positions, t)
		}
	}
	// we've got the slice, now pass it on to all "listeners"
	for _, ch := range s.chans {
		ch <- positions
	}
	s.mu.Unlock()
}

// Subscribe - get a channel to get the data from this buffer on flush
func (s *Settlement) Subscirbe() (<-chan []types.Position, int) {
	s.mu.Lock()
	k := s.getKey()
	ch := make(chan []types.Position, s.chBuf)
	s.chans[k] = ch
	s.mu.Unlock()
	return ch, k
}

// Unsubscribe - close channel and remove from active duty
func (s *Settlement) Unsubscribe(k int) {
	s.mu.Lock()
	if ch, ok := s.chans[k]; ok {
		close(ch)
		// mark this key as available
		s.free = append(s.free, k)
	}
	delete(s.chans, k)
	s.mu.Unlock()
}

func (s *Settlement) getKey() int {
	// no need to lock mutex, the caller should have the lock
	if len(s.free) != 0 {
		k := s.free[0]
		// remove first element
		s.free = s.free[1:]
		return k
	}
	// no available keys, the next available one == length of the chans map
	return len(s.chans)
}

func (s *Settlement) updateEvt() {
	for mID, traderEvts := range s.buf {
		// check if this market is brand new, if so: the traders just get created directly
		// while it's not a huge change, this does save pointless lookups in an empty map
		if _, ok := s.calc[mID]; !ok {
			s.calc[mID] = make(map[string]types.Position, len(traderEvts))
			for tID, evt := range traderEvts {
				calc := evtToProto(evt)
				updatePosition(&calc, evt)
				s.calc[mID][tID] = calc
			}
			continue
		}
		// a known/existing market, add/update trader data
		for tID, evt := range traderEvts {
			calc, ok := s.calc[mID][tID]
			// if this trader is new, create the entry
			if !ok {
				// create entry for new trader
				calc = evtToProto(evt)
			}
			// update the price, volumes, unrealisedPnL, etc...
			// this works the same for new traders or existing ones
			updatePosition(&calc, evt)
			// now reassign/add to map
			s.calc[mID][tID] = calc
		}
	}
}

func updatePosition(p *types.Position, e events.SettlePosition) {
	totPrice, totVolume := p.AverageEntryPrice, p.OpenVolume
	totPrice *= absUint64(totVolume)
	for _, t := range e.Trades() {
		price, size := t.Price(), t.Size()
		if size == 0 {
			continue
		}
		app := true
		for i, pt := range p.FifoQueue {
			if pt.Price == price {
				pt.Volume += size
				app = false
				if pt.Volume == 0 {
					p.FifoQueue = p.FifoQueue[:i+copy(p.FifoQueue[i:], p.FifoQueue[i+1:])]
				}
				break
			}
		}
		totPrice += price * absUint64(size)
		totVolume += size
		if app {
			p.FifoQueue = append(p.FifoQueue, &types.PositionTrade{
				Volume: size,
				Price:  price,
			})
		}
	}
	p.AverageEntryPrice = totPrice / absUint64(totVolume)
	// MTM price * open volume == total value of current pos - the entry price/cost of said position
	p.UnrealisedPNL = int64(e.Price())*p.OpenVolume - p.OpenVolume*int64(p.AverageEntryPrice)
}

func evtToProto(e events.SettlePosition) types.Position {
	trades := e.Trades()
	p := types.Position{
		MarketID:   e.MarketID(),
		PartyID:    e.Party(),
		OpenVolume: e.Size(),
		FifoQueue:  make([]*types.PositionTrade, 0, len(trades)),
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
