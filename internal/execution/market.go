package execution

import (
	"context"
	"crypto/sha256"
	"encoding/base32"
	"encoding/binary"
	"fmt"
	"sync"
	"time"

	"code.vegaprotocol.io/vega/internal/buffer"
	"code.vegaprotocol.io/vega/internal/collateral"
	"code.vegaprotocol.io/vega/internal/events"
	"code.vegaprotocol.io/vega/internal/logging"
	"code.vegaprotocol.io/vega/internal/markets"
	"code.vegaprotocol.io/vega/internal/matching"
	"code.vegaprotocol.io/vega/internal/positions"
	"code.vegaprotocol.io/vega/internal/risk"
	"code.vegaprotocol.io/vega/internal/settlement"
	"code.vegaprotocol.io/vega/internal/storage"
	storcfg "code.vegaprotocol.io/vega/internal/storage/config"
	types "code.vegaprotocol.io/vega/proto"

	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
)

type Market struct {
	log *logging.Logger

	riskConfig       storcfg.RiskConfig
	collateralConfig collateral.Config
	positionConfig   positions.Config
	settlementConfig settlement.Config
	matchingConfig   matching.Config

	mkt         *types.Market
	closingAt   time.Time
	currentTime time.Time
	mu          sync.Mutex

	markPrice uint64

	// engines
	matching           *matching.OrderBook
	tradableInstrument *markets.TradableInstrument
	risk               *risk.Engine
	position           *positions.Engine
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

// SetMarketID assigns a deterministic pseudo-random ID to a Market
func SetMarketID(marketcfg *types.Market, seq uint64) error {
	marketcfg.Id = ""
	marketbytes, err := proto.Marshal(marketcfg)
	if err != nil {
		return err
	}
	if len(marketbytes) == 0 {
		return errors.New("failed to marshal market")
	}

	seqbytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(seqbytes, seq)

	h := sha256.New()
	h.Write(marketbytes)
	h.Write(seqbytes)

	d := h.Sum(nil)
	d = d[:20]
	marketcfg.Id = base32.StdEncoding.EncodeToString(d)
	return nil
}

// NewMarket creates a new market using the market framework configuration and creates underlying engines.
func NewMarket(
	log *logging.Logger,
	riskConfig storcfg.RiskConfig,
	collateralConfig collateral.Config,
	positionConfig positions.Config,
	settlementConfig settlement.Config,
	matchingConfig matching.Config,
	mkt *types.Market,
	candles CandleStore,
	orders OrderStore,
	parties PartyStore,
	trades TradeStore,
	accounts *storage.Account,
	now time.Time,
	seq uint64,
) (*Market, error) {

	tradableInstrument, err := markets.NewTradableInstrument(log, mkt.TradableInstrument)
	if err != nil {
		return nil, errors.Wrap(err, "unable to instantiate a new market")
	}

	closingAt, err := tradableInstrument.Instrument.GetMarketClosingTime()
	if err != nil {
		return nil, errors.Wrap(err, "unable to get market closing time")
	}

	collateralEngine, err := collateral.New(log, collateralConfig, mkt.Id, accounts)
	if err != nil {
		return nil, errors.Wrap(err, "unable to set up collateral engine")
	}

	candlesBuf := buffer.NewCandle(mkt.Id, candles, now)

	riskEngine := risk.NewEngine(log, riskConfig, tradableInstrument.RiskModel, getInitialFactors())
	positionEngine := positions.New(log, positionConfig)
	settleEngine := settlement.New(log, settlementConfig, tradableInstrument.Instrument.Product, mkt.Id)

	market := &Market{
		log:                log,
		mkt:                mkt,
		closingAt:          closingAt,
		currentTime:        time.Time{},
		matching:           matching.NewOrderBook(log, matchingConfig, mkt.Id, false),
		tradableInstrument: tradableInstrument,
		risk:               riskEngine,
		position:           positionEngine,
		settlement:         settleEngine,
		collateral:         collateralEngine,
		candles:            candles,
		orders:             orders,
		parties:            parties,
		trades:             trades,
		accounts:           accounts,
		candlesBuf:         candlesBuf,
	}
	SetMarketID(mkt, seq)

	return market, nil
}

// ReloadConf will trigger a reload of all the config settings in the market and all underlying engines
// this is required when hot-reloading any config changes, eg. logger level.
func (m *Market) ReloadConf(
	matchingConfig matching.Config,
	riskConfig storcfg.RiskConfig,
	collateralConfig collateral.Config,
	positionConfig positions.Config,
	settlementConfig settlement.Config,
) {
	m.log.Info("reloading configuration")
	m.matching.ReloadConf(matchingConfig)
	m.risk.ReloadConf(riskConfig)
	m.position.ReloadConf(positionConfig)
	m.settlement.ReloadConf(settlementConfig)
	m.collateral.ReloadConf(collateralConfig)
}

// GetID returns the id of the given market
func (m *Market) GetID() string {
	return m.mkt.Id
}

// OnChainTimeUpdate notifies the market of a new time event/update.
// todo: make this a more generic function name e.g. OnTimeUpdateEvent
func (m *Market) OnChainTimeUpdate(t time.Time) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.currentTime = t
	// TODO(): handle market start time

	m.log.Debug("Calculating risk factors (if required)",
		logging.String("market-id", m.mkt.Id))

	m.risk.CalculateFactors(t)
	m.risk.UpdatePositions(m.markPrice, m.position.Positions())

	m.log.Debug("Calculated risk factors and updated positions (maybe)",
		logging.String("market-id", m.mkt.Id))

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
		transfers, err := m.collateral.Transfer(positions)
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
	// Validate market
	if order.MarketID != m.mkt.Id {
		m.log.Error("Market ID mismatch",
			logging.Order(*order),
			logging.String("market", m.mkt.Id))

		return nil, types.ErrInvalidMarketID
	}

	// Verify and add new parties
	party, _ := m.parties.GetByID(order.PartyID)
	if party == nil {
		if err := m.parties.Post(&types.Party{Id: order.PartyID}); err != nil {
			return nil, err
		}
		// create accounts if needed
		if err := m.collateral.AddTraderToMarket(order.PartyID); err != nil {
			return nil, err
		}
	}

	confirmation, err := m.matching.SubmitOrder(order)
	if confirmation == nil || err != nil {
		m.log.Error("Failure after submitting order to matching engine",
			logging.Order(*order),
			logging.Error(err))

		return nil, err
	}

	// Insert aggressive remaining order
	err = m.orders.Post(*order)
	if err != nil {
		m.log.Error("Failure storing new order in submit order", logging.Error(err))
	}

	if confirmation.PassiveOrdersAffected != nil {
		// Insert all passive orders siting on the book
		for _, order := range confirmation.PassiveOrdersAffected {
			err := m.orders.Put(*order)
			if err != nil {
				m.log.Fatal("Failure storing order update in submit order",
					logging.Order(*order),
					logging.Error(err))
			}
		}
	}

	if confirmation.Trades != nil {
		// a trade contains 2 traders, so at most, each trade can introduce 2 new positions to the market
		// but usually this won't happen, and even if it does, a buffer of 1 should be enough
		// do this once, we'll use this as a reference for the channel buffer size
		positionCount := len(m.position.Positions()) + 2*len(confirmation.Trades)
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
				m.log.Error("Failure storing new trade in submit order",
					logging.Trade(*trade),
					logging.Error(err))
			}

			// Save to trade buffer for generating candles etc
			err := m.candlesBuf.AddTrade(*trade)
			if err != nil {
				m.log.Error("Failure adding trade to candle buffer after submit order",
					logging.Trade(*trade),
					logging.Error(err))
			}

			// Calculate and set current mark price
			m.setMarkPrice(trade)

			// Update positions and settlement
			settle := m.positionAndSettle(trade, positionCount)
			m.log.Debug("Total length of settle MTM after submit order", logging.Int("settle", len(settle)))

			// Move money after settlement, check risk engine and get margin balances to update (if any)
			margins := m.collateralAndRisk(settle)
			m.log.Debug("Total margin accounts to be updated after submit order", logging.Int("risk-update-len", len(margins)))
		}
	}

	return confirmation, nil
}

func (m *Market) setMarkPrice(trade *types.Trade) {
	// The current mark price calculation is simply the last trade
	// in the future this will use varying logic based on market config
	// the responsibility for calculation could be elsewhere for testability
	m.markPrice = trade.Price
}

func (m *Market) positionAndSettle(trade *types.Trade, posCount int) []events.Transfer {
	// create channel for positions to populate and settlement to consume
	positionCh := make(chan events.MarketPosition, posCount)
	// starting settlement first, the reading routine does more work, so it'll be slower
	// although, it can be moved down if you really want
	settleCh := m.settlement.SettleMTM(*trade, m.markPrice, positionCh) // no pointer, trade is RO
	// Update party positions for trade affected, pushes new positions on channel
	m.position.Update(trade, positionCh)
	// when Update returns, the channel has to be closed, so we can read from the settleCh for collateral...
	close(positionCh)
	// this channel is unbuffered, and therefore can be used in bare return
	return <-settleCh
}

// this function handles moving money after settle MTM + risk margin updates
// but does not move the money between trader accounts (ie not to/from margin accounts after risk)
func (m *Market) collateralAndRisk(settle []events.Transfer) []events.Risk {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	transferCh, errCh := m.collateral.TransferCh(settle)
	go func() {
		err := <-errCh
		if err != nil {
			m.log.Error(
				"Some error in collateral when processing settle MTM transfers",
				logging.Error(err),
			)
			cancel()
		}
	}()
	// let risk engine do its thing here - it returns a slice of money that needs
	// to be moved to and from margin accounts
	riskUpdates := m.risk.UpdateMargins(ctx, transferCh, m.markPrice)
	m.log.Info("Risk done")
	if len(riskUpdates) == 0 {
		m.log.Warn("probably no risk margin changes due to error")
		return nil
	}
	m.log.Debug(
		"Got more stuff to do in collateral",
		logging.String("dump stuff", fmt.Sprintf("%#v", riskUpdates)),
	)
	return riskUpdates
}

// CancelOrder cancels the given order
func (m *Market) CancelOrder(order *types.Order) (*types.OrderCancellationConfirmation, error) {
	// Validate Market
	if order.MarketID != m.mkt.Id {
		m.log.Error("Market ID mismatch",
			logging.Order(*order),
			logging.String("market", m.mkt.Id))

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
	if order.MarketID != m.mkt.Id {
		m.log.Error("Market ID mismatch",
			logging.Order(*order),
			logging.String("market", m.mkt.Id))

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
	if existingOrder.MarketID != m.mkt.Id {
		m.log.Error("Market ID mismatch",
			logging.Order(*existingOrder),
			logging.String("market", m.mkt.Id))
		return &types.OrderConfirmation{}, types.ErrInvalidMarketID
	}

	m.mu.Lock()
	currentTime := m.currentTime
	m.mu.Unlock()

	newOrder := &types.Order{
		Id:        existingOrder.Id,
		MarketID:  existingOrder.MarketID,
		PartyID:   existingOrder.PartyID,
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
		// todo: txn or othe   r strategy (https://gitlab.com/vega-prxotocol/trading-core/issues/160)
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
