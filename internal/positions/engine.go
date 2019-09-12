package positions

import (
	"errors"
	"fmt"
	"sync"

	"code.vegaprotocol.io/vega/internal/events"
	"code.vegaprotocol.io/vega/internal/logging"
	types "code.vegaprotocol.io/vega/proto"
)

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
	ErrPositionNotFound = errors.New("position not found")
)

func (m MarketPosition) String() string {
	return fmt.Sprintf("size:%v, buy:%v, sell:%v, price:%v, partyID:%v",
		m.size, m.buy, m.sell, m.price, m.partyID)
}

func (m MarketPosition) Buy() int64 {
	return m.buy
}

func (m MarketPosition) Sell() int64 {
	return m.sell
}

func (m MarketPosition) Size() int64 {
	return m.size
}

func (m MarketPosition) Party() string {
	return m.partyID
}

func (m MarketPosition) Price() uint64 {
	return m.price
}

// UpdatedPosition returns the updated position using the potential
// sells and buys
func (m *MarketPosition) UpdatedPosition(price uint64) *MarketPosition {
	return &MarketPosition{
		buy:     0,
		sell:    0,
		size:    m.size - m.sell + m.buy,
		partyID: m.partyID,
		price:   price,
	}
}

type Engine struct {
	log *logging.Logger
	Config

	cfgMu sync.Mutex
	mu    *sync.RWMutex
	// partyID -> MarketPosition
	positions map[string]*MarketPosition
}

func New(log *logging.Logger, config Config) *Engine {
	// setup logger
	log = log.Named(namedLogger)
	log.SetLevel(config.Level.Get())

	return &Engine{
		Config:    config,
		log:       log,
		mu:        &sync.RWMutex{},
		positions: map[string]*MarketPosition{},
	}
}

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
	e.mu.Lock()
	pos, found := e.positions[order.PartyID]
	if !found {
		pos = &MarketPosition{partyID: order.PartyID}
		e.positions[order.PartyID] = pos
	}
	if order.Side == types.Side_Buy {
		pos.buy += int64(order.Size)
	} else {
		pos.sell += int64(order.Size)
	}
	e.mu.Unlock()
	return pos, nil
}

// UnregisterOrder undoes the actions of RegisterOrder. It is used when an order
// has been rejected by the Risk Engine, or when an order is amended or canceled.
func (e *Engine) UnregisterOrder(order *types.Order) (*MarketPosition, error) {
	e.mu.Lock()
	pos, found := e.positions[order.PartyID]
	e.mu.Unlock()
	if !found {
		return nil, ErrPositionNotFound
	}
	if order.Side == types.Side_Buy {
		pos.buy -= int64(order.Size)
	} else {
		pos.sell -= int64(order.Size)
	}
	return pos, nil
}

// Update pushes the previous positions on the channel + the updated open volumes of buyer/seller
func (e *Engine) Update(trade *types.Trade, ch chan<- events.MarketPosition) {
	// Not using defer e.mu.Unlock(), because defer calls add some overhead
	// and this is called for each transaction, so we want to optimise as much as possible
	// there aren't multiple returns here anyway, so just unlock as and when it's needed
	e.mu.Lock()
	// todo(cdm): overflow should be managed at the trade/order creation point. We shouldn't accept an order onto
	// your book that would overflow your position. Order validation requires position store/state lookup.

	buyer, ok := e.positions[trade.Buyer]
	if !ok {
		buyer = &MarketPosition{
			partyID: trade.Buyer,
		}
		e.positions[trade.Buyer] = buyer
	}

	seller, ok := e.positions[trade.Seller]
	if !ok {
		seller = &MarketPosition{
			partyID: trade.Seller,
		}
		e.positions[trade.Seller] = seller
	}
	// mark to market for all open positions
	e.updatePositions(trade)
	// Update long/short actual position for buyer and seller.
	// The buyer's position increases and the seller's position decreases.
	buyer.size += int64(trade.Size)
	seller.size -= int64(trade.Size)

	// Update potential positions. Potential positions decrease for both buyer and seller.
	buyer.buy -= int64(trade.Size)
	seller.sell -= int64(trade.Size)

	// these positions need to be added, too
	// in case the price of the trade != mark price
	ch <- buyer
	ch <- seller

	if e.log.GetLevel() == logging.DebugLevel {
		e.log.Debug("Positions Updated for trade",
			logging.Trade(*trade),
			logging.String("buyer-position", fmt.Sprintf("%+v", buyer)),
			logging.String("seller-position", fmt.Sprintf("%+v", seller)))
	}

	// we've set all the values now, unlock after logging
	// because we're working on MarketPosition pointers
	e.mu.Unlock()
}

// Removes positions for distressed traders, and returns the most up to date positions we have
func (e *Engine) RemoveDistressed(traders []events.MarketPosition) []events.MarketPosition {
	ret := make([]events.MarketPosition, 0, len(traders))
	e.mu.Lock()
	for _, trader := range traders {
		party := trader.Party()
		if current, ok := e.positions[party]; ok {
			ret = append(ret, current)
		}
		delete(e.positions, party)
	}
	e.mu.Unlock()
	return ret
}

// iterate over all open positions, for mark to market based on new market price
func (e *Engine) updatePositions(trade *types.Trade) {
	for _, pos := range e.positions {
		// just set the price for all positions here (this shouldn't actually be required, but we'll cross that bridge when we get there
		pos.price = trade.Price
	}
}

// just the logic to update buyer, will eventually return the MarketPosition we need to push
func (e *Engine) Positions() []MarketPosition {
	e.mu.RLock()
	out := make([]MarketPosition, 0, len(e.positions))
	for _, value := range e.positions {
		out = append(out, *value)
	}
	e.mu.RUnlock()
	return out
}

func (e *Engine) Parties() []string {
	parties := make([]string, 0, len(e.positions))
	for _, v := range e.positions {
		parties = append(parties, v.Party())
	}
	return parties
}
