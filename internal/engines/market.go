package engines

import (
	"code.vegaprotocol.io/vega/internal/engines/matching"
	"code.vegaprotocol.io/vega/internal/logging"
	types "code.vegaprotocol.io/vega/proto"
	"github.com/pkg/errors"
)

type Market struct {
	*Config
	marketcfg *types.Market

	matching   *matching.OrderBook
	instrument *Instrument
}

// NewMarket create a new market using the marketcfg specification
// and the configuration
func NewMarket(cfg *Config, marketcfg *types.Market) (*Market, error) {
	instrument, err := NewInstrument(marketcfg.TradableInstrument.Instrument)
	if err != nil {
		return nil, errors.Wrap(err, "unable to intanciate a new market")
	}

	mkt := &Market{
		Config:     cfg,
		marketcfg:  marketcfg,
		matching:   matching.NewOrderBook(cfg.Matching, marketcfg.Id, false),
		instrument: instrument,
	}

	return mkt, nil
}

// GetID returns the id of the given market
func (m *Market) GetID() string {
	return m.marketcfg.Id
}

// CancelOrder cancel the given order
func (m *Market) CancelOrder(order *types.Order) (*types.OrderCancellationConfirmation, error) {
	// Validate Market
	if order.Market != m.marketcfg.Id {
		m.log.Error("Market ID mismatch",
			logging.Order(*order),
			logging.String("market", m.marketcfg.Id))

		return nil, types.ErrInvalidMarketID
	}

	return m.matching.CancelOrder(order)
}

// SubmitOrder submits the given order
func (m *Market) SubmitOrder(order *types.Order) (*types.OrderConfirmation, error) {
	// Validate Market
	if order.Market != m.marketcfg.Id {
		m.log.Error("Market ID mismatch",
			logging.Order(*order),
			logging.String("market", m.marketcfg.Id))

		return nil, types.ErrInvalidMarketID
	}

	return m.matching.SubmitOrder(order)
}

// DeleteOrder delete the given order from the order book
func (m *Market) DeleteOrder(order *types.Order) error {
	// Validate Market
	if order.Market != m.marketcfg.Id {
		m.log.Error("Market ID mismatch",
			logging.Order(*order),
			logging.String("market", m.marketcfg.Id))

		return types.ErrInvalidMarketID
	}
	return m.matching.DeleteOrder(order)
}

// AmendOrder amend an existing order from the order book
func (m *Market) AmendOrder(order *types.Order) error {
	// Validate Market
	if order.Market != m.marketcfg.Id {
		m.log.Error("Market ID mismatch",
			logging.Order(*order),
			logging.String("market", m.marketcfg.Id))

		return types.ErrInvalidMarketID
	}

	return m.matching.AmendOrder(order)
}

// RemoveExpiredOrders remove all expired orders from the order book
func (m *Market) RemoveExpiredOrders(timestamp int64) []types.Order {
	return m.matching.RemoveExpiredOrders(timestamp)
}
