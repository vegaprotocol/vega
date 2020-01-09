package execution

import (
	"context"
	"crypto/sha256"
	"encoding/base32"
	"encoding/binary"
	"fmt"
	"math"
	"sync"
	"time"

	"code.vegaprotocol.io/vega/collateral"
	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/markets"
	"code.vegaprotocol.io/vega/matching"
	"code.vegaprotocol.io/vega/metrics"
	"code.vegaprotocol.io/vega/positions"
	types "code.vegaprotocol.io/vega/proto"
	"code.vegaprotocol.io/vega/risk"
	"code.vegaprotocol.io/vega/settlement"

	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
)

var (
	// ErrMarketClosed signals that an action have been tried to be applied on a closed market
	ErrMarketClosed = errors.New("market closed")
	// ErrTraderDoNotExists signals that the trader used does not exists
	ErrTraderDoNotExists = errors.New("trader does not exist")
	// ErrMarginCheckFailed signals that a margin check for a position failed
	ErrMarginCheckFailed = errors.New("margin check failed")
	// ErrMarginCheckInsufficient signals that a margin had not enough funds
	ErrMarginCheckInsufficient = errors.New("insufficient margin")
	// ErrInvalidInitialMarkPrice signals that the initial mark price for a market is invalid
	ErrInvalidInitialMarkPrice = errors.New("invalid initial mark price (mkprice <= 0)")
	// ErrMissingGeneralAccountForParty ...
	ErrMissingGeneralAccountForParty = errors.New("missing general account for party")

	networkPartyID = "network"
)

// Market represents an instance of a market in vega and is in charge of calling
// the engines in order to process all transctiona
type Market struct {
	log   *logging.Logger
	idgen *IDgenerator

	matchingConfig matching.Config

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

	// buffers
	orderBuf        OrderBuf
	partyBuf        PartyBuf
	tradeBuf        TradeBuf
	transferBuf     TransferBuf
	candleBuf       CandleBuf
	marginLevelsBuf MarginLevelsBuf
	settleBuf       SettlementBuf

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
	candleBuf CandleBuf,
	orderBuf OrderBuf,
	partyBuf PartyBuf,
	tradeBuf TradeBuf,
	transferBuf TransferBuf,
	marginLevelsBuf MarginLevelsBuf,
	settlementBuf SettlementBuf,
	now time.Time,
	idgen *IDgenerator,
) (*Market, error) {

	tradableInstrument, err := markets.NewTradableInstrument(log, mkt.TradableInstrument)
	if err != nil {
		return nil, errors.Wrap(err, "unable to instantiate a new market")
	}

	if tradableInstrument.Instrument.InitialMarkPrice == 0 {
		return nil, ErrInvalidInitialMarkPrice
	}

	closingAt, err := tradableInstrument.Instrument.GetMarketClosingTime()
	if err != nil {
		return nil, errors.Wrap(err, "unable to get market closing time")
	}

	book := matching.NewOrderBook(log, matchingConfig, mkt.Id,
		tradableInstrument.Instrument.InitialMarkPrice, false)

	asset := tradableInstrument.Instrument.Product.GetAsset()
	riskEngine := risk.NewEngine(log, riskConfig, tradableInstrument.MarginCalculator,
		tradableInstrument.RiskModel, getInitialFactors(log, mkt, asset), book)
	positionEngine := positions.New(log, positionConfig)
	settleEngine := settlement.New(log, settlementConfig, tradableInstrument.Instrument.Product, mkt.Id, settlementBuf)

	// start first candle
	candleBuf.Start(mkt.Id, now)

	market := &Market{
		log:                log,
		idgen:              idgen,
		mkt:                mkt,
		closingAt:          closingAt,
		currentTime:        now,
		markPrice:          tradableInstrument.Instrument.InitialMarkPrice,
		matching:           book,
		tradableInstrument: tradableInstrument,
		risk:               riskEngine,
		position:           positionEngine,
		settlement:         settleEngine,
		collateral:         collateralEngine,
		partyEngine:        partyEngine,
		orderBuf:           orderBuf,
		partyBuf:           partyBuf,
		tradeBuf:           tradeBuf,
		candleBuf:          candleBuf,
		transferBuf:        transferBuf,
		marginLevelsBuf:    marginLevelsBuf,
		settleBuf:          settlementBuf,
	}

	return market, nil
}

func (m *Market) GetMarketData() types.MarketData {
	bestBidPrice, bestBidVolume := m.matching.BestBidPriceAndVolume()
	bestOfferPrice, bestOfferVolume := m.matching.BestOfferPriceAndVolume()
	return types.MarketData{
		Market:          m.GetID(),
		BestBidPrice:    bestBidPrice,
		BestBidVolume:   bestBidVolume,
		BestOfferPrice:  bestOfferPrice,
		BestOfferVolume: bestOfferVolume,
		MidPrice:        (bestBidPrice + bestOfferPrice) / 2,
		MarkPrice:       m.markPrice,
		Timestamp:       m.currentTime.UnixNano(),
	}
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
	timer := metrics.NewTimeCounter(m.mkt.Id, "market", "OnChainTimeUpdate")

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

	err := m.candleBuf.Flush(m.mkt.Id, t)
	if err != nil {
		m.log.Error("Failed to flush candles from buffer for market",
			logging.String("market-id", m.mkt.Id),
			logging.Error(err),
		)
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
			transfers, err := m.collateral.FinalSettlement(m.GetID(), positions)
			if err != nil {
				m.log.Error(
					"Failed to get ledger movements after settling closed market",
					logging.String("market-id", m.GetID()),
					logging.Error(err),
				)
			} else {
				m.transferBuf.Add(transfers)
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
				parties := m.partyEngine.GetByMarket(m.GetID())
				clearMarketTransfers, err := m.collateral.ClearMarket(m.GetID(), asset, parties)
				if err != nil {
					m.log.Error("Clear market error",
						logging.String("market-id", m.GetID()),
						logging.Error(err))
				} else {
					m.transferBuf.Add(clearMarketTransfers)
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

	timer.EngineTimeCounterAdd()
	return
}

// SubmitOrder submits the given order
func (m *Market) SubmitOrder(order *types.Order) (*types.OrderConfirmation, error) {
	timer := metrics.NewTimeCounter(m.mkt.Id, "market", "SubmitOrder")
	orderValidity := "invalid"
	defer func() {
		timer.EngineTimeCounterAdd()
		metrics.OrderCounterInc(m.mkt.Id, orderValidity)
	}()

	// set those at the begining as even rejected order get through the buffers
	m.idgen.SetID(order)
	order.CreatedAt = m.currentTime.UnixNano()

	if m.closed {
		// adding order to the buffer first
		order.Status = types.Order_Rejected
		order.Reason = types.OrderError_MARKET_CLOSED
		m.orderBuf.Add(*order)
		return nil, ErrMarketClosed
	}

	// Validate market
	if order.MarketID != m.mkt.Id {
		// adding order to the buffer first
		order.Status = types.Order_Rejected
		order.Reason = types.OrderError_INVALID_MARKET_ID
		m.orderBuf.Add(*order)

		m.log.Error("Market ID mismatch",
			logging.Order(*order),
			logging.String("market", m.mkt.Id))

		return nil, types.ErrInvalidMarketID
	}

	// Verify and add new parties
	// party, _ := m.parties.GetByID(order.PartyID)
	party, _ := m.partyEngine.GetByMarketAndID(m.GetID(), order.PartyID)
	if party == nil {
		// adding order to the buffer first
		order.Status = types.Order_Rejected
		order.Reason = types.OrderError_INVALID_PARTY_ID
		m.orderBuf.Add(*order)

		// trader should be created before even trying to post order
		return nil, ErrTraderDoNotExists
	}

	// ensure party have a general account, and margin account is / can be created
	asset, _ := m.mkt.GetAsset()
	_, err := m.collateral.CreatePartyMarginAccount(order.PartyID, order.MarketID, asset)
	if err != nil {
		m.log.Error("Margin account verification failed",
			logging.String("party-id", order.PartyID),
			logging.String("market-id", m.GetID()),
			logging.String("asset", asset),
		)
		// adding order to the buffer first
		order.Status = types.Order_Rejected
		order.Reason = types.OrderError_MISSING_GENERAL_ACCOUNT
		m.orderBuf.Add(*order)
		return nil, ErrMissingGeneralAccountForParty
	}

	// if this is a market order, let's set the price to it now.
	if order.Type == types.Order_MARKET {
		order.Price = m.matching.MarketOrderPrice(order.Side)
	}

	// Register order as potential positions
	pos, err := m.position.RegisterOrder(order)
	if err != nil {
		// adding order to the buffer first
		order.Status = types.Order_Rejected
		order.Reason = types.OrderError_INTERNAL_ERROR
		m.orderBuf.Add(*order)

		m.log.Error("Unable to register potential trader position",
			logging.String("market-id", m.GetID()),
			logging.Error(err))
		return nil, ErrMarginCheckFailed
	}

	// Perform check and allocate margin
	if err = m.checkMarginForOrder(pos, order); err != nil {
		_, err1 := m.position.UnregisterOrder(order)
		if err1 != nil {
			m.log.Error("Unable to unregister potential trader positions",
				logging.String("market-id", m.GetID()),
				logging.Error(err1))
		}

		// adding order to the buffer first
		order.Status = types.Order_Rejected
		order.Reason = types.OrderError_MARGIN_CHECK_FAILED
		m.orderBuf.Add(*order)

		m.log.Error("Unable to check/add margin for trader",
			logging.String("market-id", m.GetID()),
			logging.Error(err))
		return nil, ErrMarginCheckFailed
	}

	// Send the aggressive order into matching engine
	confirmation, err := m.matching.SubmitOrder(order)
	if confirmation == nil || err != nil {
		_, err := m.position.UnregisterOrder(order)
		if err != nil {
			m.log.Error("Unable to unregister potential trader positions",
				logging.String("market-id", m.GetID()),
				logging.Error(err))
		}
		order.Status = types.Order_Rejected
		if oerr, ok := types.IsOrderError(err); ok {
			order.Reason = oerr
		} else {
			// should not happend but still...
			order.Reason = types.OrderError_INTERNAL_ERROR
		}
		m.orderBuf.Add(*order)
		m.log.Error("Failure after submitting order to matching engine",
			logging.Order(*order),
			logging.Error(err))

		return nil, err
	}

	// if order was FOK or IOC some or all of it may have not be consumed, so we need to
	// remove them from the potential orders,
	// then we should be able to process the rest of the order properly.
	if (order.TimeInForce == types.Order_FOK || order.TimeInForce == types.Order_IOC) &&
		confirmation.Order.Remaining != 0 {
		// create a temporary order with the size beeing the remaining
		tmpOrder := *order
		tmpOrder.Size = order.Remaining
		_, err := m.position.UnregisterOrder(&tmpOrder)
		if err != nil {
			m.log.Error("Unable to unregister potential trader positions",
				logging.String("market-id", m.GetID()),
				logging.Error(err))
		}
	}

	// Insert aggressive remaining order
	m.orderBuf.Add(*order)

	if confirmation.PassiveOrdersAffected != nil {
		// Insert or update passive orders siting on the book
		for _, order := range confirmation.PassiveOrdersAffected {
			m.orderBuf.Add(*order)
		}
	}

	if len(confirmation.Trades) > 0 {

		// Calculate and set current mark price
		m.setMarkPrice(confirmation.Trades[len(confirmation.Trades)-1])

		// Insert all trades resulted from the executed order
		for idx, trade := range confirmation.Trades {
			trade.Id = fmt.Sprintf("%s-%010d", order.Id, idx)
			if order.Side == types.Side_Buy {
				trade.BuyOrder = order.Id
				trade.SellOrder = confirmation.PassiveOrdersAffected[idx].Id
			} else {
				trade.SellOrder = order.Id
				trade.BuyOrder = confirmation.PassiveOrdersAffected[idx].Id
			}

			m.tradeBuf.Add(*trade)

			// Save to trade buffer for generating candles etc
			err := m.candleBuf.AddTrade(*trade)
			if err != nil {
				m.log.Error("Failure adding trade to candle buffer after submit order",
					logging.Trade(*trade),
					logging.Error(err))
			}

			// Update positions (this communicates with settlement via channel)
			m.position.Update(trade)
			// add trade to settlement engine for correct MTM settlement of individual trades
			m.settlement.AddTrade(trade)
		}

		// now let's get the transfers for MTM settlement
		events := m.position.UpdateMarkPrice(m.markPrice)
		settle := m.settlement.SettleMTM(m.markPrice, events)

		// Only process collateral and risk once per order, not for every trade
		margins := m.collateralAndRisk(settle)
		if len(margins) > 0 {
			transfers, closed, err := m.collateral.MarginUpdate(m.GetID(), margins)
			if m.log.GetLevel() == logging.DebugLevel {
				m.log.Debug(
					"Updated margin balances",
					logging.Int("transfer-count", len(transfers)),
					logging.Int("closed-count", len(closed)),
					logging.Error(err),
				)
				for _, tr := range transfers {
					for _, v := range tr.GetTransfers() {
						m.log.Debug(
							"Ensured margin on order with success",
							logging.String("transfer", fmt.Sprintf("%v", *v)),
							logging.String("market-id", m.GetID()),
						)
					}
				}
			}
			if err == nil && len(transfers) > 0 {
				m.transferBuf.Add(transfers)
			}
			err = m.resolveClosedOutTraders(closed, order)
			if err != nil {
				m.log.Error("unable to close out traders",
					logging.String("market-id", m.GetID()),
					logging.Error(err))
			}
		}
	}

	orderValidity = "valid" // used in deferred func.
	return confirmation, nil
}

// resolveClosedOutTraders - the traders with the given market position who haven't got sufficient collateral
// need to be closed out -> the network buys/sells the open volume, and trades with the rest of the network
// this flow is similar to the SubmitOrder bit where trades are made, with fewer checks (e.g. no MTM settlement, no risk checks)
// pass in the order which caused traders to be distressed
func (m *Market) resolveClosedOutTraders(distressedMarginEvts []events.Margin, o *types.Order) error {
	if len(distressedMarginEvts) == 0 {
		return nil
	}
	timer := metrics.NewTimeCounter(m.mkt.Id, "market", "resolveClosedOutTraders")
	defer timer.EngineTimeCounterAdd()

	distressedPos := make([]events.MarketPosition, 0, len(distressedMarginEvts))
	for _, v := range distressedMarginEvts {
		distressedPos = append(distressedPos, v)
	}
	// cancel pending orders for traders
	rmorders, err := m.matching.RemoveDistressedOrders(distressedPos)
	if err != nil {
		m.log.Error(
			"Failed to remove distressed traders from the orderbook",
			logging.Error(err),
		)
		return err
	}
	if len(rmorders) == 0 {
		return nil
	}
	// push rm orders into buf ?
	for _, o := range rmorders {
		m.orderBuf.Add(*o)
	}

	mktID := m.GetID()
	// remove the orders from the positions engine
	for _, v := range rmorders {
		_, err = m.position.UnregisterOrder(v)
		if err != nil {
			m.log.Error("unable to unregister order for a distressed party",
				logging.String("party-id", v.PartyID),
				logging.String("market-id", mktID),
				logging.String("order-id", v.Id),
			)
		}
	}

	// now that we closed orders, let's run the risk engine again
	// so it'll separate the positions still in distress from the
	// which have acceptable margins
	okPos, closed := m.risk.ExpectMargins(distressedMarginEvts, m.markPrice)

	if m.log.GetLevel() == logging.DebugLevel {
		for _, v := range okPos {
			m.log.Debug("previously distressed party have now an acceptable margin",
				logging.String("market-id", mktID),
				logging.String("party-id", v.Party()),
			)
		}
	}

	// get the actual position, so we can work out what the total position of the market is going to be
	var networkPos int64
	for _, pos := range closed {
		networkPos += pos.Size()
	}
	if networkPos == 0 {
		// remove accounts, positions and return
		// from settlement engine first
		m.settlement.RemoveDistressed(closed)
		// then from positions
		closed = m.position.RemoveDistressed(closed)
		asset, _ := m.mkt.GetAsset()
		// finally remove from collateral (moving funds where needed)
		var movements *types.TransferResponse
		movements, err = m.collateral.RemoveDistressed(closed, m.GetID(), asset)
		if err != nil {
			m.log.Error(
				"Failed to remove distressed accounts cleanly",
				logging.Error(err),
			)
			return err
		}
		// currently just logging ledger movements, will be added to a stream storage engine in time
		if m.log.GetLevel() == logging.DebugLevel {
			m.log.Debug(
				"Ledger movements after removing distressed traders",
				logging.String("ledger-dump", fmt.Sprintf("%#v", movements.Transfers)),
			)
		}
		return nil
	}
	// network order
	// @TODO this order is more of a placeholder than an actual final version
	// of the network order we'll be using
	size := uint64(math.Abs(float64(networkPos)))
	no := types.Order{
		MarketID:    m.GetID(),
		Remaining:   size,
		Status:      types.Order_Active,
		PartyID:     networkPartyID,  // network is not a party as such
		Side:        types.Side_Sell, // assume sell, price is zero in that case anyway
		CreatedAt:   m.currentTime.UnixNano(),
		Reference:   fmt.Sprintf("LS-%s", o.Id), // liquidity sourcing, reference the order which caused the problem
		TimeInForce: types.Order_FOK,            // this is an all-or-nothing order, so TIF == FOK
		Type:        types.Order_NETWORK,
	}
	no.Size = no.Remaining
	m.idgen.SetID(&no)
	// we need to buy, specify side + max price
	if networkPos < 0 {
		no.Side = types.Side_Buy
	}
	// Send the aggressive order into matching engine
	confirmation, err := m.matching.SubmitOrder(&no)
	if err != nil {
		m.log.Error("Failure after submitting order to matching engine",
			logging.Order(no),
			logging.Error(err))

		return err
	}
	// @NOTE: At this point, the network order was updated by the orderbook
	// the price field now contains the average trade price at which the order was fulfilled
	m.orderBuf.Add(no)
	// if err := m.orders.Post(no); err != nil {
	// 	m.log.Error("Failure storing new order in submit order", logging.Error(err))
	// }

	if confirmation.PassiveOrdersAffected != nil {
		// Insert or update passive orders siting on the book
		for _, order := range confirmation.PassiveOrdersAffected {
			m.orderBuf.Add(*order)
		}
	}

	if confirmation.Trades != nil {
		// Insert all trades resulted from the executed order
		for idx, trade := range confirmation.Trades {
			trade.Id = fmt.Sprintf("%s-%010d", no.Id, idx)
			if no.Side == types.Side_Buy {
				trade.BuyOrder = no.Id
				trade.SellOrder = confirmation.PassiveOrdersAffected[idx].Id
			} else {
				trade.SellOrder = no.Id
				trade.BuyOrder = confirmation.PassiveOrdersAffected[idx].Id
			}

			m.tradeBuf.Add(*trade)

			// Save to trade buffer for generating candles etc
			err = m.candleBuf.AddTrade(*trade)
			if err != nil {
				m.log.Error("Failure adding trade to candle buffer after submit order",
					logging.Trade(*trade),
					logging.Error(err))
			}
			// we skip setting the mark price when the network is trading

			// Update positions - this is a special trade involving the network as party
			// so rather than checking this every time we call Update, call special UpdateNetwork
			m.position.UpdateNetwork(trade)
		}
	}

	if err = m.zeroOutNetwork(size, closed, &no, o); err != nil {
		m.log.Error(
			"Failed to create closing order with distressed traders",
			logging.Error(err),
		)
		return err
	}
	// remove accounts, positions, any funds left on the distressed accounts will be moved to the
	// insurance pool, which needs to happen before we settle the non-distressed traders
	closed = m.position.RemoveDistressed(closed)
	asset, _ := m.mkt.GetAsset()
	movements, err := m.collateral.RemoveDistressed(closed, m.GetID(), asset)
	if err != nil {
		m.log.Error(
			"Failed to remove distressed accounts cleanly",
			logging.Error(err),
		)
		return err
	}
	// currently just logging ledger movements, will be added to a stream storage engine in time
	// only actually perform the Sprintf call if we're running on debug level
	if m.log.GetLevel() == logging.DebugLevel {
		m.log.Debug(
			"Ledger movements after removing distressed traders",
			logging.String("ledger-dump", fmt.Sprintf("%#v", movements.Transfers)),
		)
	}
	// get the updated positions
	evt := m.position.Positions()
	// settle MTM, the positions have changed
	settle := m.settlement.SettleMTM(m.markPrice, evt)
	// we're not interested in the events here, they're used for margin updates
	// we know the margin requirements will be met, and come the next block
	// margins will automatically be checked anyway

	_, responses, err := m.collateral.MarkToMarket(m.GetID(), settle)
	if m.log.GetLevel() == logging.DebugLevel {
		m.log.Debug(
			"ledger movements after MTM on traders who closed out distressed",
			logging.Int("response-count", len(responses)),
			logging.String("raw", fmt.Sprintf("%#v", responses)),
		)
	}
	// send transfer to buffer
	m.transferBuf.Add(responses)
	return err
}

func (m *Market) zeroOutNetwork(size uint64, traders []events.MarketPosition, settleOrder, initial *types.Order) error {
	timer := metrics.NewTimeCounter(m.mkt.Id, "market", "zeroOutNetwork")
	defer timer.EngineTimeCounterAdd()

	tmpOrderBook := matching.NewOrderBook(m.log, m.matchingConfig, m.GetID(), m.markPrice, false)
	side := types.Side_Sell
	if settleOrder.Side == side {
		side = types.Side_Buy
	}
	order := types.Order{
		MarketID:    m.GetID(),
		Remaining:   size,
		Status:      types.Order_Active,
		PartyID:     networkPartyID,
		Side:        side, // assume sell, price is zero in that case anyway
		Price:       settleOrder.Price,
		CreatedAt:   m.currentTime.UnixNano(),
		Reference:   "close-out distressed",
		TimeInForce: types.Order_FOK, // this is an all-or-nothing order, so TIF == FOK
		Type:        types.Order_NETWORK,
	}
	order.Size = order.Remaining
	if _, err := tmpOrderBook.SubmitOrder(&order); err != nil {
		return err
	}
	m.orderBuf.Add(order)
	// traders need to take the opposing side
	side = settleOrder.Side
	// @TODO get trader positions, submit orders for each
	for i, trader := range traders {
		to := types.Order{
			MarketID:    m.GetID(),
			Remaining:   uint64(math.Abs(float64(trader.Size()))),
			Status:      types.Order_Active,
			PartyID:     trader.Party(),
			Side:        side,              // assume sell, price is zero in that case anyway
			Price:       settleOrder.Price, // average price
			CreatedAt:   m.currentTime.UnixNano(),
			Reference:   fmt.Sprintf("distressed-%d-%s", i, initial.Id),
			TimeInForce: types.Order_FOK, // this is an all-or-nothing order, so TIF == FOK
			Type:        types.Order_LIMIT,
		}
		to.Size = to.Remaining
		m.idgen.SetID(&to)
		// store the trader order, too
		m.orderBuf.Add(to)
		res, err := tmpOrderBook.SubmitOrder(&to)
		if err != nil {
			return err
		}
		// now store the resulting trades:
		for _, trade := range res.Trades {
			m.tradeBuf.Add(*trade)
		}
	}
	return nil
}

func (m *Market) checkMarginForOrder(pos *positions.MarketPosition, order *types.Order) error {
	timer := metrics.NewTimeCounter(m.mkt.Id, "market", "checkMarginForOrder")
	defer timer.EngineTimeCounterAdd()

	asset, err := m.mkt.GetAsset()
	if err != nil {
		return errors.Wrap(err, "unable to get risk updates")
	}

	e, err := m.collateral.GetPartyMargin(pos, asset, m.GetID())
	if err != nil {
		return err
	}

	riskUpdate, err := m.collateralAndRiskForOrder(e, m.markPrice)
	if err != nil {
		m.log.Error("unable to top up margin on new order",
			logging.String("party-id", order.PartyID),
			logging.String("market-id", order.MarketID),
			logging.Error(err),
		)
		return ErrMarginCheckInsufficient
	} else if riskUpdate == nil {
		if m.log.GetLevel() == logging.DebugLevel {
			m.log.Debug("No risk updates",
				logging.String("market-id", m.GetID()))
		}
	} else {
		// this should always be a increase to the InitialMargin
		// if it does fail, we need to return an error straight away
		transferResps, closePositions, err := m.collateral.MarginUpdate(m.GetID(), []events.Risk{riskUpdate})
		if err != nil {
			return errors.Wrap(err, "unable to get risk updates")
		}
		m.transferBuf.Add(transferResps)

		if len(closePositions) > 0 {

			// if closeout list is != 0 then we return an error as well, it means the trader did not have enough
			// monies to reach the InitialMargin

			m.log.Error("party did not have enough collateral to reach the InitialMargin",
				logging.Order(*order),
				logging.String("market-id", m.GetID()))

			return ErrMarginCheckInsufficient
		}

		if m.log.GetLevel() == logging.DebugLevel {
			m.log.Debug("Transfers applied for ")
			for _, tr := range transferResps {
				for _, v := range tr.GetTransfers() {
					m.log.Debug(
						"Ensured margin on order with success",
						logging.String("transfer", fmt.Sprintf("%v", *v)),
						logging.String("market-id", m.GetID()),
					)
				}
			}
		}
	}

	return nil
}

// this function handles moving money after settle MTM + risk margin updates
// but does not move the money between trader accounts (ie not to/from margin accounts after risk)
func (m *Market) collateralAndRiskForOrder(e events.Margin, price uint64) (events.Risk, error) {
	timer := metrics.NewTimeCounter(m.mkt.Id, "market", "collateralAndRiskForOrder")
	defer timer.EngineTimeCounterAdd()

	// let risk engine do its thing here - it returns a slice of money that needs
	// to be moved to and from margin accounts
	riskUpdate, err := m.risk.UpdateMarginOnNewOrder(e, price)
	if err != nil {
		return nil, err
	}
	if riskUpdate == nil {
		m.log.Debug("No risk updates after call to Update Margins in collateralAndRisk()")
		return nil, nil
	}

	// push margins into the buffer
	margins := riskUpdate.MarginLevels()
	margins.Timestamp = m.currentTime.UnixNano()
	margins.MarketID = m.GetID()
	m.marginLevelsBuf.Add(*margins)

	if m.log.GetLevel() == logging.DebugLevel {
		m.log.Debug("Got margins transfer on new order")
		transfer := riskUpdate.Transfer()
		m.log.Debug(
			"New margin transfer on order submit",
			logging.String("transfer", fmt.Sprintf("%v", *transfer)),
			logging.String("market-id", m.GetID()),
		)
	}

	return riskUpdate, nil
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
	timer := metrics.NewTimeCounter(m.mkt.Id, "market", "collateralAndRisk")
	evts, response, err := m.collateral.MarkToMarket(m.GetID(), settle)
	if err != nil {
		m.log.Error(
			"Failed to process mark to market settlement (collateral)",
			logging.Error(err),
		)
		return nil
	}
	if m.log.GetLevel() == logging.DebugLevel {
		// @TODO stream the ledger movements here
		m.log.Debug(
			"transfer responses after MTM settlement",
			logging.Int("transfer-count", len(response)),
			logging.String("raw-dump", fmt.Sprintf("%#v", response)),
		)
	}
	// sending response to buffer
	m.transferBuf.Add(response)

	// let risk engine do its thing here - it returns a slice of money that needs
	// to be moved to and from margin accounts
	riskUpdates := m.risk.UpdateMarginsOnSettlement(context.Background(), evts, m.markPrice)
	if len(riskUpdates) == 0 {
		if m.log.GetLevel() == logging.DebugLevel {
			m.log.Debug("No risk updates after call to Update Margins in collateralAndRisk()")
		}
		return nil
	}

	// push margins into the buffer
	t := m.currentTime.UnixNano()
	mktid := m.GetID()
	for _, riskUpdate := range riskUpdates {
		margins := riskUpdate.MarginLevels()
		margins.Timestamp = t
		margins.MarketID = mktid
		m.marginLevelsBuf.Add(*margins)
	}

	timer.EngineTimeCounterAdd()
	return riskUpdates
}

// CancelOrder cancels the given order
func (m *Market) CancelOrder(order *types.Order) (*types.OrderCancellationConfirmation, error) {
	timer := metrics.NewTimeCounter(m.mkt.Id, "market", "CancelOrder")
	defer timer.EngineTimeCounterAdd()

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
	m.orderBuf.Add(*order)
	_, err = m.position.UnregisterOrder(order)
	if err != nil {
		m.log.Error("Failure unregistering order in positions engine (cancel)",
			logging.Order(*order),
			logging.Error(err))
	}

	return cancellation, nil
}

// CancelOrderByID locates order by its Id and cancels it
func (m *Market) CancelOrderByID(orderID string) (*types.OrderCancellationConfirmation, error) {
	order, err := m.matching.GetOrderByID(orderID)
	if err != nil {
		return nil, err
	}
	return m.CancelOrder(order)
}

// DeleteOrder delete the given order from the order book
func (m *Market) DeleteOrder(order *types.Order) (err error) {
	timer := metrics.NewTimeCounter(m.mkt.Id, "market", "DeleteOrder")

	// Validate Market
	if order.MarketID != m.mkt.Id {
		m.log.Error("Market ID mismatch",
			logging.Order(*order),
			logging.String("market", m.mkt.Id))

		err = types.ErrInvalidMarketID
	} else {
		err = m.matching.DeleteOrder(order)
	}
	timer.EngineTimeCounterAdd()
	return
}

// AmendOrder amend an existing order from the order book
func (m *Market) AmendOrder(orderAmendment *types.OrderAmendment) (*types.OrderConfirmation, error) {
	timer := metrics.NewTimeCounter(m.mkt.Id, "market", "AmendOrder")
	defer timer.EngineTimeCounterAdd()

	// Verify that the market is not closed
	if m.closed {
		return nil, ErrMarketClosed
	}

	// Try and locate the existing order specified on the
	// order book in the matching engine for this market
	existingOrder, err := m.matching.GetOrderByID(orderAmendment.OrderID)
	if err != nil {
		m.log.Error("Invalid order reference",
			logging.String("id", orderAmendment.GetOrderID()),
			logging.String("party", orderAmendment.GetPartyID()),
			logging.String("market", orderAmendment.GetMarketID()),
			logging.Error(err))

		return nil, types.ErrInvalidOrderReference
	}

	// Validate Market
	if existingOrder.MarketID != m.mkt.Id {
		m.log.Error("Market ID mismatch",
			logging.String("market-id", m.mkt.Id),
			logging.Order(*existingOrder))

		return &types.OrderConfirmation{}, types.ErrInvalidMarketID
	}

	m.mu.Lock()
	currentTime := m.currentTime
	m.mu.Unlock()

	newOrder := &types.Order{
		Id:          existingOrder.Id,
		MarketID:    existingOrder.MarketID,
		PartyID:     existingOrder.PartyID,
		Side:        existingOrder.Side,
		Price:       existingOrder.Price,
		Size:        existingOrder.Size,
		Remaining:   existingOrder.Remaining,
		TimeInForce: existingOrder.TimeInForce,
		CreatedAt:   currentTime.UnixNano(),
		Status:      existingOrder.Status,
		ExpiresAt:   existingOrder.ExpiresAt,
		Reference:   existingOrder.Reference,
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

	if newOrder.TimeInForce == types.Order_GTT && orderAmendment.ExpiresAt != 0 {
		newOrder.ExpiresAt = orderAmendment.ExpiresAt
		expiryChange = true
	}

	// if increase in size or change in price
	// ---> DO atomic cancel and submit
	if priceShift || sizeIncrease {
		ret, err := m.orderCancelReplace(existingOrder, newOrder)
		return ret, err
	}

	// if decrease in size or change in expiration date
	// ---> DO amend in place in matching engine
	if expiryChange || sizeDecrease {
		return m.orderAmendInPlace(newOrder)
	}

	m.log.Error("Order amendment not allowed", logging.Order(*existingOrder))
	return &types.OrderConfirmation{}, types.ErrEditNotAllowed

}

func (m *Market) orderCancelReplace(existingOrder, newOrder *types.Order) (conf *types.OrderConfirmation, err error) {
	timer := metrics.NewTimeCounter(m.mkt.Id, "market", "orderCancelReplace")

	m.log.Debug("Cancel/replace order")

	cancellation, err := m.CancelOrder(existingOrder)
	if err != nil || cancellation == nil {
		if err == nil {
			err = fmt.Errorf("order cancellation failed (no error given)")
		}
		m.log.Error("Failed to cancel order from matching engine during CancelReplace",
			logging.OrderWithTag(*existingOrder, "existing-order"),
			logging.OrderWithTag(*newOrder, "new-order"),
			logging.Error(err))
	} else {
		conf, err = m.SubmitOrder(newOrder)
	}

	timer.EngineTimeCounterAdd()
	return
}

func (m *Market) orderAmendInPlace(newOrder *types.Order) (*types.OrderConfirmation, error) {
	timer := metrics.NewTimeCounter(m.mkt.Id, "market", "orderAmendInPlace")
	defer timer.EngineTimeCounterAdd()

	_, err := m.position.RegisterOrder(newOrder)
	if err != nil {
		return &types.OrderConfirmation{}, err
	}

	err = m.matching.AmendOrder(newOrder)
	if err != nil {
		m.log.Error("Failure after amend order from matching engine (amend-in-place)",
			logging.OrderWithTag(*newOrder, "new-order"),
			logging.Error(err))
		return &types.OrderConfirmation{}, err
	}
	m.orderBuf.Add(*newOrder)
	return &types.OrderConfirmation{}, nil
}

// RemoveExpiredOrders remove all expired orders from the order book
func (m *Market) RemoveExpiredOrders(timestamp int64) (orderList []types.Order, err error) {
	timer := metrics.NewTimeCounter(m.mkt.Id, "market", "RemoveExpiredOrders")

	if m.closed {
		err = ErrMarketClosed
	} else {
		orderList = m.matching.RemoveExpiredOrders(timestamp)
		// need to remove the expired orders from the potentials positions
		for _, order := range orderList {
			order := order
			_, err = m.position.UnregisterOrder(&order)
			if err != nil {
				m.log.Error("Failure unregistering order in positions engine (cancel)",
					logging.Order(order),
					logging.Error(err))
			}
		}

	}

	timer.EngineTimeCounterAdd()
	return
}

// create an actual risk model, and calculate the risk factors
// if something goes wrong, return the hard-coded values of old
func getInitialFactors(log *logging.Logger, mkt *types.Market, asset string) *types.RiskResult {
	rm, err := risk.NewModel(log, mkt.TradableInstrument.RiskModel, asset)
	// @TODO log this error
	if err != nil {
		return nil
	}
	if ok, fact := rm.CalculateRiskFactors(nil); ok {
		return fact
	}
	// default to hard-coded risk factors
	return &types.RiskResult{
		RiskFactors: map[string]*types.RiskFactor{
			asset: {Long: 0.15, Short: 0.25},
		},
		PredictedNextRiskFactors: map[string]*types.RiskFactor{
			asset: {Long: 0.15, Short: 0.25},
		},
	}
}
