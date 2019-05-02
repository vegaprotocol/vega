package position

import (
	"fmt"
	"sync"

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

func (e *Engine) Update(trade *types.Trade, ch chan<- *types.SettlePosition) {

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
	if s := updateBuyerPosition(buyer, trade); s != nil {
		ch <- s
	}
	if s := updateSellerPosition(seller, trade); s != nil {
		ch <- s
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
func (e *Engine) updateMTMPositions(trade *types.Trade, ch chan<- *types.SettlePosition) {
	for id, pos := range e.positions {
		// no volume (closed out), or position price == market price
		// there's no MTM settlement required, carry on...
		if pos.size == 0 || pos.price == trade.Price {
			// this trader was closed out already, no MTM applies
			continue
		}
		// e.g. position avg -> 90, market price 100:
		// short -> (100 - 90) * -10 => -100 ==> MTM_LOSS
		// long -> (100-90) * 10 => 100 ==> MTM_WIN
		mtmShare := int64(trade.Price-pos.price) * pos.size
		settle := &types.SettlePosition{
			Owner: id,
			Size:  1, // this is an absolute delta based on volume, so size is always 1
			Amount: &types.FinancialAmount{
				Amount: mtmShare, // current delta -> mark price minus current position average
			},
			Type: types.SettleType_MTM_LOSS,
		}
		// we've handled the mark-to-marked share here, so whatever the position, from this point on
		// the traders' positions are volume * market.price
		pos.price = trade.Price
		if mtmShare > 0 {
			// win type
			settle.Type = types.SettleType_MTM_WIN
		}
		// set position
		ch <- settle
	}
}

// just the logic to update buyer, will eventually return the SettlePosition we need to push
func updateBuyerPosition(buyer *MarketPosition, trade *types.Trade) *types.SettlePosition {
	if buyer.size == 0 {
		// position is N long, at current market price, job done
		buyer.size = int64(trade.Size)
		buyer.price = trade.Price
		return nil
	}
	if buyer.size > 0 {
		delta := int64(trade.Price) - int64(buyer.price)
		// mark-to-market for the buyers' current position already
		settle := &types.SettlePosition{
			Owner: buyer.partyID,
			Size:  uint64(buyer.size),
			Amount: &types.FinancialAmount{
				Amount: delta, // current delta -> mark price minus current position average
			},
			Type: types.SettleType_MTM_WIN,
		}
		if delta < 0 {
			// market price went down
			settle.Type = types.SettleType_MTM_LOSS
		}
		// now that we've requested the mark-to-market stuff, we can update the trader position to the current market value already
		buyer.price = trade.Price
		// increment the size
		buyer.size += int64(trade.Size)
		return settle
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
func updateSellerPosition(seller *MarketPosition, trade *types.Trade) *types.SettlePosition {
	// seller had no open positions, so we don't have to check collateral
	if seller.size == 0 {
		seller.size -= int64(trade.Size)
		seller.price = trade.Price
		return nil
	}
	// seller was already short, that position is only going to increase, we can't really close anything here
	if seller.size < 0 {
		// the delta is the inverse of buyer: current position - market price
		// if the market went down, the delta will be positive, and the short positions win
		// if the market went up, delta will be negative, and short positions lost
		delta := int64(seller.price) - int64(trade.Price)
		// mark-to-market for the sellers current position, seller is confirmed short already
		settle := &types.SettlePosition{
			Owner: seller.partyID,
			Size:  uint64(-seller.size), // current volume has to be adjusted to conform to market
			Amount: &types.FinancialAmount{
				Amount: delta, // current delta -> mark price minus current position average
			},
			Type: types.SettleType_MTM_WIN,
		}
		if delta < 0 {
			// market price went down
			settle.Type = types.SettleType_MTM_LOSS
		}
		// seller is already short, calculate price average, and update accordingly
		seller.price = trade.Price
		seller.size -= int64(trade.Size)
		return settle
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
