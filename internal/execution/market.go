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
	"code.vegaprotocol.io/vega/internal/metrics"
	"code.vegaprotocol.io/vega/internal/positions"
	"code.vegaprotocol.io/vega/internal/risk"
	"code.vegaprotocol.io/vega/internal/settlement"
	types "code.vegaprotocol.io/vega/proto"

	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	ErrMarketClosed      = errors.New("market closed")
	ErrTraderDoNotExists = errors.New("trader does not exist")
)

type Market struct {
	log *logging.Logger

	riskConfig       risk.Config
	positionConfig   positions.Config
	settlementConfig settlement.Config
	matchingConfig   matching.Config

	mkt         *types.Market
	closingAt   time.Time
	currentTime time.Time
	mu          sync.Mutex

	markPrice uint64

	// own engines
	matching           *matching.OrderBook
	tradableInstrument *markets.TradableInstrument
	risk               *risk.Engine
	position           *positions.Engine
	settlement         *settlement.Engine

	// deps engines
	collateral  *collateral.Engine
	partyEngine *Party

	// stores
	candles CandleStore
	orders  OrderStore
	parties PartyStore
	trades  TradeStore

	// buffers
	candlesBuf *buffer.Candle

	// metrics
	blockTime *prometheus.CounterVec

	closed bool
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
	riskConfig risk.Config,
	positionConfig positions.Config,
	settlementConfig settlement.Config,
	matchingConfig matching.Config,
	collateralEngine *collateral.Engine,
	partyEngine *Party,
	mkt *types.Market,
	candles CandleStore,
	orders OrderStore,
	parties PartyStore,
	trades TradeStore,
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
		partyEngine:        partyEngine,
		candles:            candles,
		orders:             orders,
		parties:            parties,
		trades:             trades,
		candlesBuf:         candlesBuf,
	}
	SetMarketID(mkt, seq)

	return market, nil
}

// ReloadConf will trigger a reload of all the config settings in the market and all underlying engines
// this is required when hot-reloading any config changes, eg. logger level.
func (m *Market) ReloadConf(
	matchingConfig matching.Config,
	riskConfig risk.Config,
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
func (m *Market) OnChainTimeUpdate(t time.Time) (closed bool) {
	start := time.Now()

	m.mu.Lock()
	defer m.mu.Unlock()

	closed = t.After(m.closingAt)
	m.closed = closed

	m.currentTime = t

	// TODO(): handle market start time

	m.log.Debug("Calculating risk factors (if required)",
		logging.String("market-id", m.mkt.Id))

	m.risk.CalculateFactors(t)

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

	if closed {
		// call settlement and stuff
		positions, err := m.settlement.Settle(t)
		if err != nil {
			m.log.Error(
				"Failed to get settle positions on market close",
				logging.Error(err),
			)
		} else {
			transfers, err := m.collateral.Transfer(m.GetID(), positions)
			if err != nil {
				m.log.Error(
					"Failed to get ledger movements after settling closed market",
					logging.String("market-id", m.GetID()),
					logging.Error(err),
				)
			} else {
				if m.log.GetLevel() == logging.DebugLevel {
					// use transfers, unused var thingy
					for _, v := range transfers {
						m.log.Debug(
							"Got transfers on market close",
							logging.String("transfer", fmt.Sprintf("%v", *v)),
							logging.String("market-id", m.GetID()),
						)
					}
				}

				asset, _ := m.mkt.GetAsset()
				parties := m.partyEngine.GetForMarket(m.GetID())
				clearMarketTransfers, err := m.collateral.ClearMarket(m.GetID(), asset, parties)
				if err != nil {
					m.log.Error("Clear market error",
						logging.String("market-id", m.GetID()),
						logging.Error(err))
				} else {
					if m.log.GetLevel() == logging.DebugLevel {
						// use transfers, unused var thingy
						for _, v := range clearMarketTransfers {
							m.log.Debug(
								"Market cleared with success",
								logging.String("transfer", fmt.Sprintf("%v", *v)),
								logging.String("market-id", m.GetID()),
							)
						}
					}
				}

			}
		}
	}

	metrics.EngineTimeCounterAdd(start, m.mkt.Id, "execution", "OnChainTimeUpdate")
	return
}

// SubmitOrder submits the given order
func (m *Market) SubmitOrder(order *types.Order) (*types.OrderConfirmation, error) {
	if m.closed {
		return nil, ErrMarketClosed
	}

	orderValidity := "invalid"
	startSubmit := time.Now() // do not reset this var
	defer func() {
		metrics.EngineTimeCounterAdd(startSubmit, m.mkt.Id, "execution", "Submit")
		metrics.OrderCounterInc(m.mkt.Id, orderValidity)
	}()

	// Validate market
	if order.MarketID != m.mkt.Id {
		m.log.Error("Market ID mismatch",
			logging.Order(*order),
			logging.String("market", m.mkt.Id))

		return nil, types.ErrInvalidMarketID
	}

	// Verify and add new parties
	start := time.Now()
	party, _ := m.parties.GetByID(order.PartyID)
	if party == nil {
		// trader should be created before even trying to post order
		return nil, ErrTraderDoNotExists
	}
	metrics.EngineTimeCounterAdd(start, m.mkt.Id, "partystore", "GetByID/Post")

	confirmation, err := m.matching.SubmitOrder(order)
	if confirmation == nil || err != nil {
		m.log.Error("Failure after submitting order to matching engine",
			logging.Order(*order),
			logging.Error(err))

		return nil, err
	}
	start = time.Now()

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
	metrics.EngineTimeCounterAdd(start, m.mkt.Id, "orderstore", "Post/Put")

	m.position.RegisterOrder(order)

	if confirmation.Trades != nil {
		// orders can contain several trades, each trade involves 2 traders
		// so there's a max number of N*2 events on the channel where N == number of trades
		tradersCh := make(chan events.MarketPosition, 2*len(confirmation.Trades))
		// now let's set the settlement engine up to listen for trader position changes (closed positions to be settled differently)
		m.settlement.ListenClosed(tradersCh)
		// insert all trades resulted from the executed order
		for idx, trade := range confirmation.Trades {
			// fmt.Printf("------------------------------- TRADE: %v\n", trade)
			trade.Id = fmt.Sprintf("%s-%010d", order.Id, idx)
			if order.Side == types.Side_Buy {
				trade.BuyOrder = order.Id
				trade.SellOrder = confirmation.PassiveOrdersAffected[idx].Id
			} else {
				trade.SellOrder = order.Id
				trade.BuyOrder = confirmation.PassiveOrdersAffected[idx].Id
			}

			start := time.Now()
			if err := m.trades.Post(trade); err != nil {
				m.log.Error("Failure storing new trade in submit order",
					logging.Trade(*trade),
					logging.Error(err))
			}
			metrics.EngineTimeCounterAdd(start, m.mkt.Id, "tradestore", "Post")
			start = time.Now()

			// Save to trade buffer for generating candles etc
			err := m.candlesBuf.AddTrade(*trade)
			if err != nil {
				m.log.Error("Failure adding trade to candle buffer after submit order",
					logging.Trade(*trade),
					logging.Error(err))
			}
			metrics.EngineTimeCounterAdd(start, m.mkt.Id, "candlestore", "AddTrade")
			start = time.Now()

			// Calculate and set current mark price
			m.setMarkPrice(trade)

			// Update positions (this communicates with settlement via channel)
			m.position.Update(trade, tradersCh)
			metrics.EngineTimeCounterAdd(start, m.mkt.Id, "positions", "Update")
		}
		close(tradersCh)
		start = time.Now()
		// now let's get the transfers for MTM settlement
		positions := m.position.Positions()
		events := make([]events.MarketPosition, 0, len(positions))
		for _, p := range positions {
			events = append(events, p)
		}
		settle := m.settlement.SettleOrder(m.markPrice, events)
		metrics.EngineTimeCounterAdd(start, m.mkt.Id, "positions", "Positions+SettleOrder")
		// this belongs outside of trade loop, only call once per order
		margins := m.collateralAndRisk(settle)
		if len(margins) > 0 {
			transfers, closed, err := m.collateral.MarginUpdate(m.GetID(), margins)
			m.log.Debug(
				"Updated margin balances",
				logging.Int("transfer-count", len(transfers)),
				logging.Int("closed-count", len(closed)),
				logging.Error(err),
			)
			// @TODO -> close out any traders that don't have enough margins left
			// if no errors were returned
		}
	}

	orderValidity = "valid"
	return confirmation, nil
}

func (m *Market) setMarkPrice(trade *types.Trade) {
	// The current mark price calculation is simply the last trade
	// in the future this will use varying logic based on market config
	// the responsibility for calculation could be elsewhere for testability
	m.markPrice = trade.Price
}

// this function handles moving money after settle MTM + risk margin updates
// but does not move the money between trader accounts (ie not to/from margin accounts after risk)
func (m *Market) collateralAndRisk(settle []events.Transfer) []events.Risk {
	ctx, cancel := context.WithCancel(context.Background())
	start := time.Now()
	defer cancel()
	transferCh, errCh := m.collateral.TransferCh(m.GetID(), settle)
	go func() {
		err := <-errCh
		if err != nil {
			m.log.Error(
				"Some error in collateral when processing settle MTM transfers",
				logging.Error(err),
			)
			cancel()
		}
		metrics.EngineTimeCounterAdd(start, m.mkt.Id, "collateral", "TransferCh")
	}()
	// let risk engine do its thing here - it returns a slice of money that needs
	// to be moved to and from margin accounts
	riskUpdates := m.risk.UpdateMargins(ctx, transferCh, m.markPrice)
	// m.log.Info("Risk done")
	if len(riskUpdates) == 0 {
		// m.log.Warn("probably no risk margin changes due to error")
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
	if m.closed {
		return nil, ErrMarketClosed
	}

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

	_, err = m.position.UnregisterOrder(order)
	if err != nil {
		m.log.Error("Failure unregistering order in positions engine (cancel)",
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
	if m.closed {
		return nil, ErrMarketClosed
	}

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

	// Unregister existing order to remove order volume from potential position.
	_, err := m.position.UnregisterOrder(existingOrder)
	if err != nil {
		m.log.Error("Failure unregistering existing order in positions engine (amend)",
			logging.Order(*existingOrder),
			logging.Error(err))
	}

	// Register amended order to add order volume to potential position.
	m.position.RegisterOrder(newOrder)

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
func (m *Market) RemoveExpiredOrders(timestamp int64) ([]types.Order, error) {
	if m.closed {
		return nil, ErrMarketClosed
	}

	return m.matching.RemoveExpiredOrders(timestamp), nil
}

func getInitialFactors() *types.RiskResult {
	return &types.RiskResult{
		RiskFactors: map[string]*types.RiskFactor{
			"ETH": {Long: 0.15, Short: 0.25},
		},
		PredictedNextRiskFactors: map[string]*types.RiskFactor{
			"ETH": {Long: 0.15, Short: 0.25},
		},
	}
}
