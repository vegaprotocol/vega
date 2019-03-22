package engines

import (
	"code.vegaprotocol.io/vega/internal/engines/matching"
	"code.vegaprotocol.io/vega/internal/logging"
	types "code.vegaprotocol.io/vega/proto"
)

type Market struct {
	*Config
	marketcfg *types.Market
	matching  *matching.OrderBook
}

func NewMarket(cfg *Config, marketcfg *types.Market) *Market {
	mkt := &Market{
		Config:    cfg,
		marketcfg: marketcfg,
		matching:  matching.NewOrderBook(cfg.Matching, marketcfg.Id, false),
	}

	return mkt
}

func (m *Market) GetID() string {
	return m.marketcfg.Id
}

func (m *Market) CancelOrder(order *types.Order) (*types.OrderCancellation, error) {
	// Validate Market
	if order.Market != m.marketcfg.Id {
		m.log.Error("Market ID mismatch",
			logging.Order(*order),
			logging.String("market", m.marketcfg.Id))

		return nil, types.ErrInvalidMarketID
	}

	return m.matching.CancelOrder(order)
}

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

func (m *Market) RemoveExpiredOrders(timestamp uint64) []types.Order {
	return m.matching.RemoveExpiredOrders(timestamp)
}
