package engines

import (
	"context"
	"fmt"
	"sync"
	"time"

	"code.vegaprotocol.io/vega/internal/buffer"
	"code.vegaprotocol.io/vega/internal/engines/collateral"
	"code.vegaprotocol.io/vega/internal/engines/matching"
	"code.vegaprotocol.io/vega/internal/engines/position"
	"code.vegaprotocol.io/vega/internal/engines/risk"
	"code.vegaprotocol.io/vega/internal/engines/settlement"
	"code.vegaprotocol.io/vega/internal/logging"
	"code.vegaprotocol.io/vega/internal/storage"
	types "code.vegaprotocol.io/vega/proto"

	"github.com/pkg/errors"
)

//go:generate go run github.com/golang/mock/mockgen -destination mocks/order_store_mock.go -package mocks code.vegaprotocol.io/vega/internal/execution OrderStore
type OrderStore interface {
	GetByPartyAndId(ctx context.Context, party string, id string) (*types.Order, error)
	Post(order types.Order) error
	Put(order types.Order) error
	Commit() error
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/trade_store_mock.go -package mocks code.vegaprotocol.io/vega/internal/execution TradeStore
type TradeStore interface {
	Commit() error
	Post(trade *types.Trade) error
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/candle_store_mock.go -package mocks code.vegaprotocol.io/vega/internal/execution CandleStore
type CandleStore interface {
	GenerateCandlesFromBuffer(market string, buf map[string]types.Candle) error
	FetchMostRecentCandle(marketID string, interval types.Interval, descending bool) (*types.Candle, error)
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/party_store_mock.go -package mocks code.vegaprotocol.io/vega/internal/execution PartyStore
type PartyStore interface {
	GetByID(id string) (*types.Party, error)
	Post(party *types.Party) error
}

type Market struct {
	Config
	log *logging.Logger

	marketcfg   *types.Market
	closingAt   time.Time
	currentTime time.Time
	mu          sync.Mutex

	markPrice uint64

	// engines
	matching           *matching.OrderBook
	tradableInstrument *TradableInstrument
	risk               *risk.Engine
	position           *position.Engine
	settlement         *settlement.Engine
	collateral         *collateral.Engine

	// stores
	candles  CandleStore
	orders   OrderStore
	parties  PartyStore
	trades   TradeStore
	accounts *storage.Account

	// buffers
	candlesBuf *buffer.Candle
}

// NewMarket create a new market using the marketcfg specification
// and the configuration
func NewMarket(
	log *logging.Logger,

	cfg Config,
	marketcfg *types.Market,
	candles CandleStore,
	orders OrderStore,
	parties PartyStore,
	trades TradeStore,
	accounts *storage.Account,
	now time.Time,
) (*Market, error) {
	// setup logger
	log = log.Named(namedLogger)
	log.SetLevel(cfg.Level.Get())

	tradableInstrument, err := NewTradableInstrument(log, marketcfg.TradableInstrument)
	if err != nil {
		return nil, errors.Wrap(err, "unable to intanciate a new market")
	}

	closingAt, err := tradableInstrument.Instrument.GetMarketClosingTime()
	if err != nil {
		return nil, errors.Wrap(err, "unable to get market closing time")
	}

	collateralEngine, err := collateral.New(log, cfg.Collateral, marketcfg.Id, accounts)
	if err != nil {
		return nil, errors.Wrap(err, "unable to set up collateral engine")
	}

	riskengine := risk.New(log, cfg.Risk, tradableInstrument.RiskModel, getInitialFactors())
	positionengine := position.New(log, cfg.Position)
	settleEngine := settlement.New(log, cfg.Settlement)

	// create buffers
	candlesBuf := buffer.NewCandle(marketcfg.Id, candles, now)

	mkt := &Market{
		log:                log,
		Config:             cfg,
		marketcfg:          marketcfg,
		closingAt:          closingAt,
		currentTime:        time.Time{},
		matching:           matching.NewOrderBook(log, cfg.Matching, marketcfg.Id, false),
		tradableInstrument: tradableInstrument,
		risk:               riskengine,
		position:           positionengine,
		settlement:         settleEngine,
		collateral:         collateralEngine,
		candles:            candles,
		orders:             orders,
		parties:            parties,
		trades:             trades,
		accounts:           accounts,
		candlesBuf:         candlesBuf,
	}

	return mkt, nil
}

func (m *Market) ReloadConf(cfg Config) {
	m.log.Info("reloading configuration")
	if m.log.GetLevel() != cfg.Level.Get() {
		m.log.Info("updating log level",
			logging.String("old", m.log.GetLevel().String()),
			logging.String("new", cfg.Level.String()),
		)
		m.log.SetLevel(cfg.Level.Get())
	}

	m.Config = cfg
	m.matching.ReloadConf(cfg.Matching)
	m.risk.ReloadConf(cfg.Risk)
	m.position.ReloadConf(cfg.Position)
	m.settlement.ReloadConf(cfg.Settlement)
	m.collateral.ReloadConf(cfg.Collateral)
}

// GetID returns the id of the given market
func (m *Market) GetID() string {
	return m.marketcfg.Id
}

// OnChainTimeUpdate notify the market of a new chain time update
func (m *Market) OnChainTimeUpdate(t time.Time) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.currentTime = t
	// TODO(): handle market start time

	m.log.Debug("Calculating risk factors (if required)",
		logging.String("market-id", m.marketcfg.Id))

	m.risk.CalculateFactors(t)
	m.risk.UpdatePositions(m.markPrice, m.position.Positions())

	m.log.Debug("Calculated risk factors and updated positions (maybe)",
		logging.String("market-id", m.marketcfg.Id))

	// generated / store the buffered candles
	previousCandlesBuf, err := m.candlesBuf.Start(t)
	if err != nil {
		m.log.Error("unable to get candles buf", logging.Error(err))
	}

	// get the buffered candles from the buffer
	err = m.candles.GenerateCandlesFromBuffer(m.GetID(), previousCandlesBuf)
	if err != nil {
		m.log.Error("Failed to generate candles from buffer for market", logging.String("market-id", m.GetID()))
	}

	if t.After(m.closingAt) {
		// call settlement and stuff
		positions, err := m.settlement.Settle(t)
		if err != nil {
			m.log.Error(
				"Failed to get settle positions on market close",
				logging.Error(err),
			)
			return
		}
		transfers, err := m.collateral.Collect(positions)
		if err != nil {
			m.log.Error(
				"Failed to get ledger movements after settling closed market",
				logging.String("market-id", m.GetID()),
				logging.Error(err),
			)
			return
		}
		// use transfers, unused var thingy
		m.log.Debug(
			"Got transfers on market close (%#v)",
			logging.String("transfers-dump", fmt.Sprintf("%#v", transfers)), // @TODO process these transfers, they contain the ledger movements...
			logging.String("market-id", m.GetID()),
		)
	}
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

	// Verify and add new parties
	party, _ := m.parties.GetByID(order.Party)
	if party == nil {
		p := &types.Party{Name: order.Party}
		err := m.parties.Post(p)
		if err != nil {
			return nil, err
		}
	}

	confirmation, err := m.matching.SubmitOrder(order)
	if confirmation == nil || err != nil {
		m.log.Error("Failure after submit order from matching engine",
			logging.Order(*order),
			logging.Error(err))

		return nil, err
	}

	// Insert aggressive remaining order
	err = m.orders.Post(*order)
	if err != nil {
		m.log.Error("Failure storing new order in execution engine (submit)", logging.Error(err))
	}
	if confirmation.PassiveOrdersAffected != nil {
		// Insert all passive orders siting on the book
		for _, order := range confirmation.PassiveOrdersAffected {
			// Note: writing to store should not prevent flow to other engines
			err := m.orders.Put(*order)
			if err != nil {
				m.log.Error("Failure storing order update in execution engine (submit)",
					logging.Order(*order),
					logging.Error(err))
			}
		}
	}

	if confirmation.Trades != nil {
		// insert all trades resulted from the executed order
		for idx, trade := range confirmation.Trades {
			trade.Id = fmt.Sprintf("%s-%010d", order.Id, idx)
			if order.Side == types.Side_Buy {
				trade.BuyOrder = order.Id
				trade.SellOrder = confirmation.PassiveOrdersAffected[idx].Id
			} else {
				trade.SellOrder = order.Id
				trade.BuyOrder = confirmation.PassiveOrdersAffected[idx].Id
			}

			if err := m.trades.Post(trade); err != nil {
				m.log.Error("Failure storing new trade in execution engine (submit)",
					logging.Trade(*trade),
					logging.Error(err))
			}

			// Save to trade buffer for generating candles etc
			err := m.candlesBuf.AddTrade(*trade)
			if err != nil {
				m.log.Error("Failure adding trade to candle buffer in execution engine (submit)",
					logging.Trade(*trade),
					logging.Error(err))
			}

			// Ensure mark price is always up to date for each market in execution engine
			m.markPrice = trade.Price

			// Update party positions for trade affected
			m.position.Update(trade)

			// Update positions for the market for the trade
			m.risk.UpdatePositions(trade.Price, m.position.Positions())
		}
	}

	return confirmation, nil
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

	cancellation, err := m.matching.CancelOrder(order)
	if cancellation == nil || err != nil {
		m.log.Error("Failure after cancel order from matching engine",
			logging.Order(*order),
			logging.Error(err))
		return nil, err
	}

	// Update the order in our stores (will be marked as cancelled)
	err = m.orders.Put(*order)
	if err != nil {
		m.log.Error("Failure storing order update in execution engine (cancel)",
			logging.Order(*order),
			logging.Error(err))
	}

	return cancellation, nil
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
func (m *Market) AmendOrder(
	orderAmendment *types.OrderAmendment,
	existingOrder *types.Order,
) (*types.OrderConfirmation, error) {
	// Validate Market
	if existingOrder.Market != m.marketcfg.Id {
		m.log.Error("Market ID mismatch",
			logging.Order(*existingOrder),
			logging.String("market", m.marketcfg.Id))
		return &types.OrderConfirmation{}, types.ErrInvalidMarketID
	}

	m.mu.Lock()
	currentTime := m.currentTime
	m.mu.Unlock()

	newOrder := &types.Order{
		Id:        existingOrder.Id,
		Market:    existingOrder.Market,
		Party:     existingOrder.Party,
		Side:      existingOrder.Side,
		Price:     existingOrder.Price,
		Size:      existingOrder.Size,
		Remaining: existingOrder.Remaining,
		Type:      existingOrder.Type,
		CreatedAt: currentTime.UnixNano(),
		Status:    existingOrder.Status,
		ExpiresAt: existingOrder.ExpiresAt,
		Reference: existingOrder.Reference,
	}
	var (
		priceShift, sizeIncrease, sizeDecrease, expiryChange = false, false, false, false
	)

	if orderAmendment.Price != 0 && existingOrder.Price != orderAmendment.Price {
		newOrder.Price = orderAmendment.Price
		priceShift = true
	}

	if orderAmendment.Size != 0 {
		newOrder.Size = orderAmendment.Size
		newOrder.Remaining = orderAmendment.Size
		if orderAmendment.Size > existingOrder.Size {
			sizeIncrease = true
		}
		if orderAmendment.Size < existingOrder.Size {
			sizeDecrease = true
		}
	}

	if newOrder.Type == types.Order_GTT && orderAmendment.ExpiresAt != 0 {
		newOrder.ExpiresAt = orderAmendment.ExpiresAt
		expiryChange = true
	}

	// if increase in size or change in price
	// ---> DO atomic cancel and submit
	if priceShift || sizeIncrease {
		return m.orderCancelReplace(existingOrder, newOrder)
	}
	// if decrease in size or change in expiration date
	// ---> DO amend in place in matching engine
	if expiryChange || sizeDecrease {
		return m.orderAmendInPlace(newOrder)
	}

	m.log.Error("Order amendment not allowed", logging.Order(*existingOrder))
	return &types.OrderConfirmation{}, types.ErrEditNotAllowed

}

func (m *Market) orderCancelReplace(existingOrder, newOrder *types.Order) (*types.OrderConfirmation, error) {
	m.log.Debug("Cancel/replace order")

	cancellation, err := m.CancelOrder(existingOrder)
	if cancellation == nil || err != nil {
		m.log.Error("Failure after cancel order from matching engine (cancel/replace)",
			logging.OrderWithTag(*existingOrder, "existing-order"),
			logging.OrderWithTag(*newOrder, "new-order"),
			logging.Error(err))

		return &types.OrderConfirmation{}, err
	}

	return m.SubmitOrder(newOrder)
}

func (m *Market) orderAmendInPlace(newOrder *types.Order) (*types.OrderConfirmation, error) {
	err := m.matching.AmendOrder(newOrder)
	if err != nil {
		m.log.Error("Failure after amend order from matching engine (amend-in-place)",
			logging.OrderWithTag(*newOrder, "new-order"),
			logging.Error(err))
		return &types.OrderConfirmation{}, err
	}
	err = m.orders.Put(*newOrder)
	if err != nil {
		m.log.Error("Failure storing order update in execution engine (amend-in-place)",
			logging.Order(*newOrder),
			logging.Error(err))
		// todo: txn or other strategy (https://gitlab.com/vega-prxotocol/trading-core/issues/160)
	}
	return &types.OrderConfirmation{}, nil
}

// RemoveExpiredOrders remove all expired orders from the order book
func (m *Market) RemoveExpiredOrders(timestamp int64) []types.Order {
	return m.matching.RemoveExpiredOrders(timestamp)
}

func getInitialFactors() *types.RiskResult {
	return &types.RiskResult{
		RiskFactors: map[string]*types.RiskFactor{
			"Ethereum/Ether": {Long: 0.15, Short: 0.25},
		},
		PredictedNextRiskFactors: map[string]*types.RiskFactor{
			"Ethereum/Ether": {Long: 0.15, Short: 0.25},
		},
	}
}
