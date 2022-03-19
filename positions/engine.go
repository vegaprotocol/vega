package positions

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"sync"

	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"
)

// Errors.
var (
	// ErrPositionNotFound signal that a position was not found for a given party.
	ErrPositionNotFound = errors.New("position not found")
)

// Broker (no longer need to mock this, use the broker/mocks wrapper).
type Broker interface {
	Send(event events.Event)
	SendBatch(events []events.Event)
}

// Engine represents the positions engine.
type Engine struct {
	marketID string
	log      *logging.Logger
	Config

	cfgMu sync.Mutex
	// partyID -> MarketPosition
	positions map[string]*MarketPosition

	// this is basically tracking all position to
	// not perform a copy when positions a retrieved by other engines
	// the pointer is hidden behind the interface, and do not expose
	// any function to mutate them, so we can consider it safe to return
	// this slice.
	positionsCpy []events.MarketPosition

	broker Broker
}

// New instantiates a new positions engine.
func New(log *logging.Logger, config Config, marketID string, broker Broker) *Engine {
	// setup logger
	log = log.Named(namedLogger)
	log.SetLevel(config.Level.Get())

	return &Engine{
		marketID:     marketID,
		Config:       config,
		log:          log,
		positions:    map[string]*MarketPosition{},
		positionsCpy: []events.MarketPosition{},
		broker:       broker,
	}
}

func (e *Engine) Hash() []byte {
	// Fields * FieldSize = (8 * 3)
	// Prices = 32 * 2
	output := make([]byte, len(e.positionsCpy)*((8*3)+(32*2)))
	var i int
	for _, p := range e.positionsCpy {
		values := []uint64{
			uint64(p.Size()),
			uint64(p.Buy()),
			uint64(p.Sell()),
		}

		for _, v := range values {
			binary.BigEndian.PutUint64(output[i:], v)
			i += 8
		}

		// Add bytes for VWBuy and VWSell here
		b := p.VWBuy().Bytes()
		copy(output[i:], b[:])
		i += 32
		s := p.VWBuy().Bytes()
		copy(output[i:], s[:])
		i += 32
	}

	return crypto.Hash(output)
}

// ReloadConf update the internal configuration of the positions engine.
func (e *Engine) ReloadConf(cfg Config) {
	e.log.Info("reloading configuration")
	if e.log.GetLevel() != cfg.Level.Get() {
		e.log.Info("updating log level",
			logging.String("old", e.log.GetLevel().String()),
			logging.String("new", cfg.Level.String()),
		)
		e.log.SetLevel(cfg.Level.Get())
	}

	e.cfgMu.Lock()
	e.Config = cfg
	e.cfgMu.Unlock()
}

// RegisterOrder updates the potential positions for a submitted order, as though
// the order were already accepted.
// It returns the updated position.
// The margins+risk engines need the updated position to determine whether the
// order should be accepted.
func (e *Engine) RegisterOrder(ctx context.Context, order *types.Order) *MarketPosition {
	pos, found := e.positions[order.Party]
	if !found {
		pos = NewMarketPosition(order.Party)
		e.positions[order.Party] = pos
		// append the pointer to the slice as well
		e.positionsCpy = append(e.positionsCpy, pos)
	}
	pos.RegisterOrder(order)
	e.broker.Send(events.NewPositionStateEvent(ctx, pos, order.MarketID))
	return pos
}

// UnregisterOrder undoes the actions of RegisterOrder. It is used when an order
// has been rejected by the Risk Engine, or when an order is amended or canceled.
func (e *Engine) UnregisterOrder(ctx context.Context, order *types.Order) *MarketPosition {
	pos, found := e.positions[order.Party]
	if !found {
		e.log.Panic("could not find position in engine when unregistering order",
			logging.Order(*order))
	}
	pos.UnregisterOrder(e.log, order)
	e.broker.Send(events.NewPositionStateEvent(ctx, pos, order.MarketID))
	return pos
}

// AmendOrder unregisters the original order and then registers the newly amended order
// this method is a quicker way of handling separate unregister+register pairs.
func (e *Engine) AmendOrder(ctx context.Context, originalOrder, newOrder *types.Order) *MarketPosition {
	pos, found := e.positions[originalOrder.Party]
	if !found {
		e.log.Panic("could not find position in engine when amending order",
			logging.Order(*originalOrder),
			logging.Order(*newOrder))
	}
	pos.AmendOrder(e.log, originalOrder, newOrder)
	e.broker.Send(events.NewPositionStateEvent(ctx, pos, originalOrder.MarketID))
	return pos
}

// UpdateNetwork - functionally the same as the Update func, except for ignoring the network
// party in the trade (whether it be buyer or seller). This could be incorporated into the Update
// function, but we know when we're adding network trades, and having this check every time is
// wasteful, and would only serve to add complexity to the Update func, and slow it down.
func (e *Engine) UpdateNetwork(trade *types.Trade) []events.MarketPosition {
	// there's only 1 position
	var (
		ok  bool
		pos *MarketPosition
	)
	size := int64(trade.Size)
	if trade.Buyer != "network" {
		pos, ok = e.positions[trade.Buyer]
		if !ok {
			e.log.Panic("could not find buyer position",
				logging.Trade(*trade))
		}

		if pos.buy < int64(trade.Size) {
			e.log.Panic("network trade with a potential buy position < to the trade size",
				logging.PartyID(trade.Buyer),
				logging.Int64("potential-buy", pos.buy),
				logging.Trade(*trade))
		}

		// potential buy pos is smaller now
		pos.buy -= int64(trade.Size)
	} else {
		pos, ok = e.positions[trade.Seller]
		if !ok {
			e.log.Panic("could not find seller position",
				logging.Trade(*trade))
		}

		if pos.sell < int64(trade.Size) {
			e.log.Panic("network trade with a potential sell position < to the trade size",
				logging.PartyID(trade.Seller),
				logging.Int64("potential-sell", pos.sell),
				logging.Trade(*trade))
		}

		// potential sell pos is smaller now
		pos.sell -= int64(trade.Size)
		// size is negative in case of a sale
		size = -size
	}
	pos.size += size

	cpy := pos.Clone()
	return []events.MarketPosition{*cpy}
}

// Update pushes the previous positions on the channel + the updated open volumes of buyer/seller.
func (e *Engine) Update(trade *types.Trade) []events.MarketPosition {
	if trade.BuyOrder == "jeremy-debug" || trade.Seller == "jeremy-debug" {
		fmt.Printf("POS UPDATE: %v\n", trade.String())
	}

	buyer, ok := e.positions[trade.Buyer]
	if !ok {
		e.log.Panic("could not find buyer position",
			logging.Trade(*trade))
	}

	seller, ok := e.positions[trade.Seller]
	if !ok {
		e.log.Panic("could not find seller position",
			logging.Trade(*trade))
	}

	// now we check if the trade is possible based on the potential positions
	// this should always be true, no trade can happen without the equivalent
	// potential position
	if buyer.buy < int64(trade.Size) {
		e.log.Panic("trade with a potential buy position < to the trade size",
			logging.PartyID(trade.Buyer),
			logging.Int64("potential-buy", buyer.buy),
			logging.Trade(*trade))
	}
	if seller.sell < int64(trade.Size) {
		e.log.Panic("trade with a potential sell position < to the trade size",
			logging.PartyID(trade.Seller),
			logging.Int64("potential-sell", buyer.sell),
			logging.Trade(*trade))
	}

	// Update long/short actual position for buyer and seller.
	// The buyer's position increases and the seller's position decreases.
	buyer.size += int64(trade.Size)
	seller.size -= int64(trade.Size)

	// Update potential positions. Potential positions decrease for both buyer and seller.
	buyer.buy -= int64(trade.Size)
	seller.sell -= int64(trade.Size)

	ret := []events.MarketPosition{
		*buyer.Clone(),
		*seller.Clone(),
	}

	if e.log.GetLevel() == logging.DebugLevel {
		e.log.Debug("Positions Updated for trade",
			logging.Trade(*trade),
			logging.String("buyer-position", fmt.Sprintf("%+v", buyer)),
			logging.String("seller-position", fmt.Sprintf("%+v", seller)))
	}
	return ret
}

// RemoveDistressed Removes positions for distressed parties, and returns the most up to date positions we have.
func (e *Engine) RemoveDistressed(parties []events.MarketPosition) []events.MarketPosition {
	ret := make([]events.MarketPosition, 0, len(parties))
	for _, party := range parties {
		e.log.Warn("removing party from positions engine",
			logging.String("party-id", party.Party()))

		party := party.Party()
		if current, ok := e.positions[party]; ok {
			ret = append(ret, current)
		}
		// remove from the map
		delete(e.positions, party)
		// remove from the slice
		for i := range e.positionsCpy {
			if e.positionsCpy[i].Party() == party {
				e.log.Warn("removing party from positions engine (cpy slice)",
					logging.String("party-id", party))
				e.positionsCpy = append(e.positionsCpy[:i], e.positionsCpy[i+1:]...)

				break
			}
		}
	}
	return ret
}

// UpdateMarkPrice update the mark price on all positions and return a slice
// of the updated positions.
func (e *Engine) UpdateMarkPrice(markPrice *num.Uint) []events.MarketPosition {
	for _, pos := range e.positions {
		pos.price.Set(markPrice)
	}
	return e.positionsCpy
}

func (e *Engine) GetOpenInterest() uint64 {
	openInterest := uint64(0)
	for _, pos := range e.positions {
		if pos.size > 0 {
			openInterest += uint64(pos.size)
		}
	}
	return openInterest
}

func (e *Engine) GetOpenInterestGivenTrades(trades []*types.Trade) uint64 {
	oi := e.GetOpenInterest()
	d := int64(0)
	for _, t := range trades {
		bSize, sSize := int64(0), int64(0)
		if p, ok := e.positions[t.Buyer]; ok {
			bSize = p.size
		}
		if p, ok := e.positions[t.Seller]; ok {
			sSize = p.size
		}
		// Change in open interest due to trades equals change in longs
		d += max(0, bSize+int64(t.Size)) - max(0, bSize) + max(0, sSize-int64(t.Size)) - max(0, sSize)
	}
	if d > 0 {
		oi += uint64(d)
	}
	if d < 0 {
		oi -= uint64(-d)
	}

	return oi
}

func max(a int64, b int64) int64 {
	if a >= b {
		return a
	}
	return b
}

// Positions is just the logic to update buyer, will eventually return the MarketPosition we need to push.
func (e *Engine) Positions() []events.MarketPosition {
	return e.positionsCpy
}

// GetPositionByPartyID - return current position for a given party, it's used in margin checks during auctions
// we're not specifying an interface of the return type, and we return a pointer to a copy for the nil.
func (e *Engine) GetPositionByPartyID(partyID string) (*MarketPosition, bool) {
	pos, ok := e.positions[partyID]
	if !ok {
		return nil, false
	}
	cpy := pos.Clone()
	// return a copy
	return cpy, true
}

// Parties returns a list of all the parties in the position engine.
func (e *Engine) Parties() []string {
	parties := make([]string, 0, len(e.positions))
	for _, v := range e.positions {
		parties = append(parties, v.Party())
	}
	return parties
}

// I64MaxAbs - get max value based on absolute values of int64 vals
// keep this function, perhaps we can reuse it in a numutil package
// once we have to deal with decimals etc...
func I64MaxAbs(vals ...int64) int64 {
	var r, m int64
	for _, v := range vals {
		av := v
		if av < 0 {
			av *= -1
		}
		if av > m {
			r = v
			m = av // current max abs is av
		}
	}
	return r
}
