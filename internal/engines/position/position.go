package position

import (
	"fmt"
	"sync"

	"code.vegaprotocol.io/vega/internal/engines/settlement"
	"code.vegaprotocol.io/vega/internal/logging"
	types "code.vegaprotocol.io/vega/proto"
)

type MarketPosition struct {
	size    int64
	margins map[string]uint64
	partyID string
	price   uint64
}

func (m MarketPosition) String() string {
	return fmt.Sprintf("size: %v, margins: %v, partyID: %v", m.size, m.margins, m.partyID)
}

// Margins returns a copy of the current margins map
func (m *MarketPosition) Margins() map[string]uint64 {
	out := make(map[string]uint64, 0)
	for k, v := range m.margins {
		out[k] = v
	}
	return out
}

// UpdateMargin updates the margin value for a single asset
func (m *MarketPosition) UpdateMargin(assetID string, margin uint64) {
	m.margins[assetID] = margin
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

func (e *Engine) Update(trade *types.Trade, ch chan<- settlement.MarketPosition) {

	// Not using defer e.mu.Unlock(), because defer calls add some overhead
	// and this is called for each transaction, so we want to optimise as much as possible
	// there aren't multiple returns here anyway, so just unlock as and when it's needed
	e.mu.Lock()
	// todo(cdm): overflow should be managed at the trade/order creation point. We shouldn't accept an order onto
	// your book that would overflow your position. Order validation requires position store/state lookup.

	buyer, ok := e.positions[trade.Buyer]
	if !ok {
		buyer = &MarketPosition{
			margins: map[string]uint64{},
			partyID: trade.Buyer,
		}
		e.positions[trade.Buyer] = buyer
	}

	seller, ok := e.positions[trade.Seller]
	if !ok {
		seller = &MarketPosition{
			margins: map[string]uint64{},
			partyID: trade.Seller,
		}
		e.positions[trade.Seller] = seller
	}
	// update positions, potentially settle (depending on positions)
	// these settle positions should *not* be pushed onto the channel
	// they should be returned instead, and passed on to collateral/settlement directly
	if pos := updateBuyerPosition(buyer, trade); pos != nil {
		ch <- pos
	}
	if pos := updateSellerPosition(seller, trade); pos != nil {
		ch <- pos
	}
	// mark to market for all open positions
	e.updateMTMPositions(trade, ch)

	e.log.Debug("Positions Updated for trade",
		logging.Trade(*trade),
		logging.String("buyer-position", fmt.Sprintf("%+v", buyer)),
		logging.String("seller-position", fmt.Sprintf("%+v", seller)))

	// we've set all the values now, unlock after logging
	// because we're working on MarketPosition pointers
	e.mu.Unlock()
}

// iterate over all open positions, for mark to market based on new market price
func (e *Engine) updateMTMPositions(trade *types.Trade, ch chan<- settlement.MarketPosition) {
	for _, pos := range e.positions {
		// no volume (closed out), or position price == market price
		// there's no MTM settlement required, carry on...
		if pos.size == 0 || pos.price == trade.Price {
			// this trader was closed out already, no MTM applies
			continue
		}
		cpy := *pos
		// let settlement handle the old position and mark it to market
		ch <- &cpy
		// we've passed on the old position to the settlement channel already
		// now simply update the price on the position
		pos.price = trade.Price
	}
}

// just the logic to update buyer, will eventually return the SettlePosition we need to push
func updateBuyerPosition(buyer *MarketPosition, trade *types.Trade) *MarketPosition {
	if buyer.size == 0 {
		// position is N long, at current market price, job done
		buyer.size = int64(trade.Size)
		buyer.price = trade.Price
		return nil
	}
	if buyer.size > 0 {
		// we need the old position to be marked to market
		pos := *buyer
		// update the buyer position to the new one
		buyer.price = trade.Price
		// increment the size
		buyer.size += int64(trade.Size)
		return &pos
	}
	// Now, the trader was short, and still might be short after the trade.
	// if trader is still short, we should just let the normal settle position flow take it from here
	buyer.size += int64(trade.Size)
	// buyer is now long, the trade is its own thing, the new position is held at current market price
	// if the trader is still short, we don't update the price, that happens when we do the normal mark-to-market flow for
	// all positions
	if buyer.size > 0 {
		buyer.price = trade.Price
	}
	return nil
}

// same as updateBuyerPosition, only the position volume goes down
func updateSellerPosition(seller *MarketPosition, trade *types.Trade) *MarketPosition {
	// seller had no open positions, so we don't have to check collateral
	if seller.size == 0 {
		seller.size -= int64(trade.Size)
		seller.price = trade.Price
		return nil
	}
	// seller was already short, that position is only going to increase, we can't really close anything here
	if seller.size < 0 {
		// seller is already short, we have to MTM the current position for the trader
		// then update the new one to the market price
		pos := *seller
		// update position
		seller.price = trade.Price
		seller.size -= int64(trade.Size)
		return &pos
	}
	// seller was long, might not be after this...
	seller.size -= int64(trade.Size)
	if seller.size < 0 {
		// seller holds a new short position, at the current market price, update position price and be done with it
		seller.price = trade.Price
	}
	// if seller is still long, we don't want to update the price here, that happens when we're updating all market positions
	return nil
}

func (e *Engine) Positions() []MarketPosition {
	e.mu.RLock()
	out := make([]MarketPosition, 0, len(e.positions))
	for _, value := range e.positions {
		out = append(out, *value)
	}
	e.mu.RUnlock()
	return out
}
