package positions

import (
	"errors"
	"fmt"
	"sync"

	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/metrics"
	types "code.vegaprotocol.io/vega/proto"
)

// MarketPosition represents the position of a party inside a market
type MarketPosition struct {
	// Actual volume
	size int64
	// Potential volume (orders not yet accepted/rejected)
	buy, sell int64

	partyID string
	price   uint64
}

// Errors
var (
	// ErrPositionNotFound signal that a position was not found for a given party.
	ErrPositionNotFound = errors.New("position not found")
)

// String returns a string representation of a market
func (m MarketPosition) String() string {
	return fmt.Sprintf("size:%v, buy:%v, sell:%v, price:%v, partyID:%v",
		m.size, m.buy, m.sell, m.price, m.partyID)
}

// Buy will returns the potential buys for a given position
func (m MarketPosition) Buy() int64 {
	return m.buy
}

// Sell returns the potential sells for the position
func (m MarketPosition) Sell() int64 {
	return m.sell
}

// Size returns the current size of the position
func (m MarketPosition) Size() int64 {
	return m.size
}

// Party returns the party to which this positions is associated
func (m MarketPosition) Party() string {
	return m.partyID
}

// Price returns the current price for this position
func (m MarketPosition) Price() uint64 {
	return m.price
}

// Engine represents the positions engine
type Engine struct {
	log *logging.Logger
	Config

	cfgMu sync.Mutex
	// partyID -> MarketPosition
	positions map[string]*MarketPosition

	// this is basically tracking all position to
	// not perform a copy when positions a retrieved by other engines
	// the pointer is hidden behing the interface, and do not expose
	// any function to mutate them, so we can consider it safe to return
	// this slice
	positionsCpy []events.MarketPosition
}

// New instantiates a new positions engine
func New(log *logging.Logger, config Config) *Engine {
	// setup logger
	log = log.Named(namedLogger)
	log.SetLevel(config.Level.Get())

	return &Engine{
		Config:       config,
		log:          log,
		positions:    map[string]*MarketPosition{},
		positionsCpy: []events.MarketPosition{},
	}
}

// ReloadConf update the internal configuration of the positions engine
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
func (e *Engine) RegisterOrder(order *types.Order) (*MarketPosition, error) {
	timer := metrics.NewTimeCounter("-", "positions", "RegisterOrder")
	pos, found := e.positions[order.PartyID]
	if !found {
		pos = &MarketPosition{partyID: order.PartyID}
		e.positions[order.PartyID] = pos
		// append the pointer to the slice as well
		e.positionsCpy = append(e.positionsCpy, pos)
	}
	if order.Side == types.Side_Buy {
		pos.buy += int64(order.Remaining)
	} else {
		pos.sell += int64(order.Remaining)
	}
	timer.EngineTimeCounterAdd()
	return pos, nil
}

// UnregisterOrder undoes the actions of RegisterOrder. It is used when an order
// has been rejected by the Risk Engine, or when an order is amended or canceled.
func (e *Engine) UnregisterOrder(order *types.Order) (pos *MarketPosition, err error) {
	timer := metrics.NewTimeCounter("-", "positions", "UnregisterOrder")
	pos, found := e.positions[order.PartyID]
	if !found {
		err = ErrPositionNotFound
	} else {
		if order.Side == types.Side_Buy {
			pos.buy -= int64(order.Remaining)
		} else {
			pos.sell -= int64(order.Remaining)
		}
	}
	timer.EngineTimeCounterAdd()
	return
}

// UpdateNetwork - functionally the same as the Update func, except for ignoring the network
// party in the trade (whether it be buyer or seller). This could be incorporated into the Update
// function, but we know when we're adding network trades, and having this check every time is
// wasteful, and would only serve to add complexity to the Update func, and slow it down
func (e *Engine) UpdateNetwork(trade *types.Trade) []events.MarketPosition {
	// there's only 1 position
	var pos *MarketPosition
	size := int64(trade.Size)
	// there will only be 1 event
	ret := make([]events.MarketPosition, 0, 1)
	if trade.Buyer != "network" {
		if b, ok := e.positions[trade.Buyer]; !ok {
			pos = &MarketPosition{
				partyID: trade.Buyer,
			}
			e.positions[trade.Buyer] = pos
			e.positionsCpy = append(e.positionsCpy, pos)
		} else {
			pos = b
		}
		// potential buy pos is smaller now
		pos.buy -= int64(trade.Size)
	} else {
		if s, ok := e.positions[trade.Seller]; !ok {
			pos = &MarketPosition{
				partyID: trade.Seller,
			}
			e.positions[trade.Seller] = pos
			e.positionsCpy = append(e.positionsCpy, pos)
		} else {
			pos = s
		}
		// potential sell pos is smaller now
		pos.sell -= int64(trade.Size)
		// size is negative in case of a sale
		size = -size
	}
	pos.size += size
	// updated pos should be appended to event slice
	ret = append(ret, *pos)
	return ret
}

// Update pushes the previous positions on the channel + the updated open volumes of buyer/seller
func (e *Engine) Update(trade *types.Trade) []events.MarketPosition {
	// todo(cdm): overflow should be managed at the trade/order creation point. We shouldn't accept an order onto
	// your book that would overflow your position. Order validation requires position store/state lookup.

	buyer, ok := e.positions[trade.Buyer]
	if !ok {
		buyer = &MarketPosition{
			partyID: trade.Buyer,
		}
		e.positions[trade.Buyer] = buyer
		e.positionsCpy = append(e.positionsCpy, buyer)
	}

	seller, ok := e.positions[trade.Seller]
	if !ok {
		seller = &MarketPosition{
			partyID: trade.Seller,
		}
		e.positions[trade.Seller] = seller
		e.positionsCpy = append(e.positionsCpy, seller)
	}
	// Update long/short actual position for buyer and seller.
	// The buyer's position increases and the seller's position decreases.
	buyer.size += int64(trade.Size)
	seller.size -= int64(trade.Size)

	// Update potential positions. Potential positions decrease for both buyer and seller.
	buyer.buy -= int64(trade.Size)
	seller.sell -= int64(trade.Size)

	ret := []events.MarketPosition{
		*buyer,
		*seller,
	}

	if e.log.GetLevel() == logging.DebugLevel {
		e.log.Debug("Positions Updated for trade",
			logging.Trade(*trade),
			logging.String("buyer-position", fmt.Sprintf("%+v", buyer)),
			logging.String("seller-position", fmt.Sprintf("%+v", seller)))
	}

	return ret
}

// RemoveDistressed Removes positions for distressed traders, and returns the most up to date positions we have
func (e *Engine) RemoveDistressed(traders []events.MarketPosition) []events.MarketPosition {
	ret := make([]events.MarketPosition, 0, len(traders))
	for _, trader := range traders {
		party := trader.Party()
		if current, ok := e.positions[party]; ok {
			ret = append(ret, current)
		}
		// remove from the map
		delete(e.positions, party)
		// remove from the slice
		for i, v := range e.positionsCpy {
			if v.Party() == trader.Party() {
				e.positionsCpy = append(e.positionsCpy[:i], e.positionsCpy[i+1:]...)
				break
			}
		}
	}
	return ret
}

// UpdateMarkPrice update the mark price on all positions and return a slice
// of the updated positions
func (e *Engine) UpdateMarkPrice(markPrice uint64) []events.MarketPosition {
	for _, pos := range e.positions {
		pos.price = markPrice
	}
	return e.positionsCpy
}

// Positions is just the logic to update buyer, will eventually return the MarketPosition we need to push
func (e *Engine) Positions() []events.MarketPosition {
	return e.positionsCpy
}

// Parties returns a list of all the parties in the position engine
func (e *Engine) Parties() []string {
	parties := make([]string, 0, len(e.positions))
	for _, v := range e.positions {
		parties = append(parties, v.Party())
	}
	return parties
}
