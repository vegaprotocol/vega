package positions

import (
	"fmt"
	"sync"

	"code.vegaprotocol.io/vega/internal/events"
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
	out := make(map[string]uint64, len(m.margins))
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
	// mark to market for all open positions
	e.updatePositions(trade)
	// update long/short position for buyer and seller
	buyer.size += int64(trade.Size)
	seller.size -= int64(trade.Size)
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
